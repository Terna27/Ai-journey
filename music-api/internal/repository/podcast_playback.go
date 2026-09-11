package repository

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"music-api/internal/models"
)

var (
	ErrPodcastPlaybackSessionNotFound = errors.New(
		"podcast playback session not found",
	)

	ErrPodcastPlaybackSessionCompleted = errors.New(
		"podcast playback session is already completed",
	)
)

type PodcastPlaybackRepository struct {
	db *pgxpool.Pool
}

func NewPodcastPlaybackRepository(
	db *pgxpool.Pool,
) *PodcastPlaybackRepository {
	return &PodcastPlaybackRepository{
		db: db,
	}
}

// CreateSession creates a new podcast playback session.
//
// The session ID is supplied by the service layer (UUIDs
// keep session identifiers unenumerable).
//
// The INSERT ... SELECT only produces a row when the
// episode is publicly playable: the episode and its
// podcast are both PUBLISHED and the episode has audio.
// Unusable drafts can therefore never receive a listener
// playback session.
func (r *PodcastPlaybackRepository) CreateSession(
	ctx context.Context,
	sessionID string,
	userID int,
	episodeID int64,
	durationMS int64,
	positionMS int64,
) (models.PodcastPlaybackSession, error) {
	query := `
		INSERT INTO podcast_playback_sessions (
			id,
			user_id,
			episode_id,
			duration_ms,
			position_ms
		)
		SELECT
			$1::uuid,
			$2::bigint,
			e.id,
			$4::bigint,
			LEAST(
				$5::bigint,
				CASE
					WHEN $4::bigint > 0
					THEN $4::bigint
					ELSE $5::bigint
				END
			)
		FROM podcast_episodes AS e
		INNER JOIN podcasts AS p
			ON p.id = e.podcast_id
		WHERE e.id = $3::bigint
		  AND e.status = 'PUBLISHED'
		  AND p.status = 'PUBLISHED'
		  AND NULLIF(
			BTRIM(e.audio_url),
			''
		  ) IS NOT NULL
		RETURNING
			id::text,
			user_id,
			episode_id,
			duration_ms,
			position_ms,
			listened_ms,
			qualified,
			qualified_at,
			completed,
			completed_at,
			started_at,
			last_activity_at,
			created_at,
			updated_at
	`

	session, err := scanPodcastPlaybackSession(
		r.db.QueryRow(
			ctx,
			query,
			sessionID,
			userID,
			episodeID,
			durationMS,
			positionMS,
		),
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return models.PodcastPlaybackSession{},
			pgx.ErrNoRows
	}

	if err != nil {
		return models.PodcastPlaybackSession{}, err
	}

	return session, nil
}

// GetSession returns one podcast playback session only
// when it belongs to userID.
//
// Another user's session is indistinguishable from a
// missing one, so session ownership is never leaked.
func (r *PodcastPlaybackRepository) GetSession(
	ctx context.Context,
	sessionID string,
	userID int,
) (models.PodcastPlaybackSession, error) {
	query := `
		SELECT
			id::text,
			user_id,
			episode_id,
			duration_ms,
			position_ms,
			listened_ms,
			qualified,
			qualified_at,
			completed,
			completed_at,
			started_at,
			last_activity_at,
			created_at,
			updated_at
		FROM podcast_playback_sessions
		WHERE id = $1::uuid
		  AND user_id = $2
	`

	session, err := scanPodcastPlaybackSession(
		r.db.QueryRow(
			ctx,
			query,
			sessionID,
			userID,
		),
	)

	if errors.Is(err, pgx.ErrNoRows) {
		return models.PodcastPlaybackSession{},
			ErrPodcastPlaybackSessionNotFound
	}

	if err != nil {
		return models.PodcastPlaybackSession{}, err
	}

	return session, nil
}

// UpdateProgress atomically:
//
//  1. locks the session (SELECT ... FOR UPDATE),
//  2. validates ownership,
//  3. clamps reported listened time against server
//     wall-clock elapsed time (anti-manipulation),
//  4. updates progress using only accepted server-side
//     values,
//  5. evaluates qualification from the ACCEPTED value,
//  6. transitions to qualified at most once,
//  7. auto-completes when the accepted position is close
//     to the end of a known-duration episode,
//  8. upserts podcast listening history (reactivating a
//     completed episode that is genuinely being replayed).
//
// The clamping and qualification math is the same shared
// resolveAcceptedProgress helper the music playback
// repository uses, so both media kinds share one
// anti-manipulation model.
func (r *PodcastPlaybackRepository) UpdateProgress(
	ctx context.Context,
	sessionID string,
	userID int,
	positionMS int64,
	durationMS int64,
	listenedMS int64,
) (models.PodcastPlaybackSession, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return models.PodcastPlaybackSession{}, err
	}
	defer tx.Rollback(ctx)

	current, err := getPodcastSessionForUpdate(
		ctx,
		tx,
		sessionID,
		userID,
	)
	if err != nil {
		return models.PodcastPlaybackSession{}, err
	}

	if current.Completed {
		return models.PodcastPlaybackSession{},
			ErrPodcastPlaybackSessionCompleted
	}

	next := resolveAcceptedProgress(
		progressBaseline{
			startedAt:      current.StartedAt,
			lastActivityAt: current.LastActivityAt,
			listenedMS:     current.ListenedMS,
			durationMS:     current.DurationMS,
		},
		positionMS,
		durationMS,
		listenedMS,
	)

	shouldQualify := models.IsQualifiedPlay(
		next.listenedMS,
		next.durationMS,
	)

	justQualified :=
		!current.Qualified &&
			shouldQualify

	// Server-consistent completion: an accepted position
	// close enough to the end completes the episode even
	// without a client `ended` event, so nearly-finished
	// episodes leave Continue Listening.
	shouldComplete :=
		models.IsPodcastPlaybackNearlyComplete(
			next.positionMS,
			next.durationMS,
		)

	updated, err := updatePodcastSessionRow(
		ctx,
		tx,
		sessionID,
		userID,
		next,
		shouldQualify,
		shouldComplete,
	)
	if err != nil {
		return models.PodcastPlaybackSession{}, err
	}

	// Progress with neither an accepted position nor any
	// listened time is not meaningful: it must not create a
	// zero-progress history row, and it must not reactivate
	// a completed episode as a "replay".
	if next.positionMS <= 0 &&
		next.listenedMS <= 0 {
		if err := tx.Commit(ctx); err != nil {
			return models.PodcastPlaybackSession{}, err
		}

		return updated, nil
	}

	err = upsertPodcastListeningHistory(
		ctx,
		tx,
		updated,
		justQualified,
		updated.Completed,
	)
	if err != nil {
		return models.PodcastPlaybackSession{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return models.PodcastPlaybackSession{}, err
	}

	return updated, nil
}

// CompleteSession finalizes a podcast playback session.
//
// Qualification is still evaluated from the accepted
// clamped listened time, and completion is idempotent: a
// second call returns the already completed session
// without touching history or play counts again.
func (r *PodcastPlaybackRepository) CompleteSession(
	ctx context.Context,
	sessionID string,
	userID int,
	positionMS int64,
	durationMS int64,
	listenedMS int64,
) (models.PodcastPlaybackSession, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return models.PodcastPlaybackSession{}, err
	}
	defer tx.Rollback(ctx)

	current, err := getPodcastSessionForUpdate(
		ctx,
		tx,
		sessionID,
		userID,
	)
	if err != nil {
		return models.PodcastPlaybackSession{}, err
	}

	// Completion is idempotent. Duplicate browser
	// completion events are harmless.
	if current.Completed {
		return current, nil
	}

	next := resolveAcceptedProgress(
		progressBaseline{
			startedAt:      current.StartedAt,
			lastActivityAt: current.LastActivityAt,
			listenedMS:     current.ListenedMS,
			durationMS:     current.DurationMS,
		},
		positionMS,
		durationMS,
		listenedMS,
	)

	shouldQualify := models.IsQualifiedPlay(
		next.listenedMS,
		next.durationMS,
	)

	justQualified :=
		!current.Qualified &&
			shouldQualify

	// The SESSION row is always finalized here (stopping,
	// switching, unloading and ending all end the session
	// lifecycle). The HISTORY row, however, only records
	// the episode as completed when the accepted position
	// is genuinely close to the end: a session finalized
	// because the user stopped halfway must NOT kick the
	// episode out of Continue Listening.
	//
	// A natural `ended` event reports a position at (or
	// within ~15 seconds of) the duration, so genuinely
	// finished episodes complete. Seeking near the end can
	// complete an episode but never manufactures listened
	// time, so qualification stays protected.
	shouldComplete :=
		models.IsPodcastPlaybackNearlyComplete(
			next.positionMS,
			next.durationMS,
		)

	updated, err := updatePodcastSessionRow(
		ctx,
		tx,
		sessionID,
		userID,
		next,
		shouldQualify,
		true,
	)
	if err != nil {
		return models.PodcastPlaybackSession{}, err
	}

	// A session finalized without any accepted progress
	// (created but abandoned before genuine playback) must
	// not leave a zero-progress history row behind.
	if next.positionMS <= 0 &&
		next.listenedMS <= 0 {
		if err := tx.Commit(ctx); err != nil {
			return models.PodcastPlaybackSession{}, err
		}

		return updated, nil
	}

	err = upsertPodcastListeningHistory(
		ctx,
		tx,
		updated,
		justQualified,
		shouldComplete,
	)
	if err != nil {
		return models.PodcastPlaybackSession{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return models.PodcastPlaybackSession{}, err
	}

	return updated, nil
}

// GetPodcastListeningHistory returns the user's most
// recently played episodes together with their podcasts.
//
// Only publicly playable episodes are listed: archived or
// unpublished content the listener can no longer access is
// excluded. limit/offset are validated by the service.
func (r *PodcastPlaybackRepository) GetPodcastListeningHistory(
	ctx context.Context,
	userID int,
	limit int,
	offset int,
) ([]models.PodcastListeningHistoryItem, error) {
	query := podcastHistoryQuery(`
		  AND h.user_id = $1
	`) + `
		ORDER BY
			h.last_played_at DESC,
			h.episode_id DESC

		LIMIT $2
		OFFSET $3
	`

	return r.queryPodcastHistoryItems(
		ctx,
		query,
		userID,
		limit,
		offset,
	)
}

// GetContinueListening returns the user's unfinished
// episodes with meaningful progress.
//
// Completed episodes and zero-progress episodes are
// excluded by the partial index backed predicate, and the
// join drops episodes that are no longer playable.
func (r *PodcastPlaybackRepository) GetContinueListening(
	ctx context.Context,
	userID int,
	limit int,
	offset int,
) ([]models.PodcastContinueListeningItem, error) {
	query := podcastHistoryQuery(`
		  AND h.user_id = $1
		  AND h.completed = FALSE
		  AND h.last_position_ms > 0
	`) + `
		ORDER BY
			h.last_played_at DESC,
			h.episode_id DESC

		LIMIT $2
		OFFSET $3
	`

	items, err := r.queryPodcastHistoryItems(
		ctx,
		query,
		userID,
		limit,
		offset,
	)

	return items, err
}

// -----------------------------------------------------------------
// Internal helpers
// -----------------------------------------------------------------

// podcastPlaybackColumns is the shared column list of a
// podcast playback session row.
const podcastPlaybackColumns = `
	id::text,
	user_id,
	episode_id,
	duration_ms,
	position_ms,
	listened_ms,
	qualified,
	qualified_at,
	completed,
	completed_at,
	started_at,
	last_activity_at,
	created_at,
	updated_at
`

// updatePodcastSessionRow persists accepted progress and
// the qualification/completion transitions for a locked
// session. Completion and qualification are latched: once
// TRUE they stay TRUE, and their timestamps are only
// stamped on the transition.
func updatePodcastSessionRow(
	ctx context.Context,
	tx pgx.Tx,
	sessionID string,
	userID int,
	next nextProgress,
	shouldQualify bool,
	shouldComplete bool,
) (models.PodcastPlaybackSession, error) {
	query := `
		UPDATE podcast_playback_sessions
		SET
			duration_ms = $1,
			position_ms = $2,
			listened_ms = $3,
			qualified =
				qualified OR $4,
			qualified_at =
				CASE
					WHEN qualified = FALSE
					 AND $4 = TRUE
					THEN now()
					ELSE qualified_at
				END,
			completed =
				completed OR $5,
			completed_at =
				CASE
					WHEN completed = FALSE
					 AND $5 = TRUE
					THEN now()
					ELSE completed_at
				END,
			last_activity_at = now(),
			updated_at = now()
		WHERE id = $6::uuid
		  AND user_id = $7
		RETURNING
	` + podcastPlaybackColumns

	return scanPodcastPlaybackSession(
		tx.QueryRow(
			ctx,
			query,
			next.durationMS,
			next.positionMS,
			next.listenedMS,
			shouldQualify,
			shouldComplete,
			sessionID,
			userID,
		),
	)
}

func getPodcastSessionForUpdate(
	ctx context.Context,
	tx pgx.Tx,
	sessionID string,
	userID int,
) (models.PodcastPlaybackSession, error) {
	query := `
		SELECT
	` + podcastPlaybackColumns + `
		FROM podcast_playback_sessions
		WHERE id = $1::uuid
		  AND user_id = $2
		FOR UPDATE
	`

	session, err := scanPodcastPlaybackSession(
		tx.QueryRow(
			ctx,
			query,
			sessionID,
			userID,
		),
	)

	if errors.Is(err, pgx.ErrNoRows) {
		return models.PodcastPlaybackSession{},
			ErrPodcastPlaybackSessionNotFound
	}

	if err != nil {
		return models.PodcastPlaybackSession{}, err
	}

	return session, nil
}

// upsertPodcastListeningHistory maintains the one history
// row per (user_id, episode_id).
//
//   - qualified_play_count increments only on the first
//     qualification of the session (justQualified), and
//     last_qualified_at is only refreshed then.
//   - completed is stored from the server-decided session
//     state.
//   - A non-completing progress upsert from a NEW session
//     reactivates a previously completed episode
//     (completed = FALSE, completed_at cleared): genuine
//     replay progress starts a new listening lifecycle.
//     Merely creating a session never writes history, so a
//     session that is created but never played cannot
//     reset a completed episode.
func upsertPodcastListeningHistory(
	ctx context.Context,
	tx pgx.Tx,
	session models.PodcastPlaybackSession,
	justQualified bool,
	completed bool,
) error {
	qualifiedIncrement := int64(0)

	if justQualified {
		qualifiedIncrement = 1
	}

	query := `
		INSERT INTO podcast_listening_history (
			user_id,
			episode_id,
			last_position_ms,
			duration_ms,
			qualified_play_count,
			completed,
			last_played_at,
			last_qualified_at,
			completed_at
		)
		VALUES (
			$1,
			$2,
			$3,
			$4,
			$5,
			$6,
			now(),
			CASE
				WHEN $5::bigint > 0
				THEN now()
				ELSE NULL
			END,
			CASE
				WHEN $6 = TRUE
				THEN now()
				ELSE NULL
			END
		)

		ON CONFLICT (
			user_id,
			episode_id
		)
		DO UPDATE SET
			-- The stored duration is the new one when known,
			-- otherwise the previously known one is kept.
			-- last_position_ms is clamped against that
			-- effective duration so the CHECK constraint
			-- (position <= duration) always holds.
			last_position_ms =
				CASE
					WHEN (
						CASE
							WHEN EXCLUDED.duration_ms > 0
							THEN EXCLUDED.duration_ms
							ELSE podcast_listening_history.duration_ms
						END
					) > 0
					THEN LEAST(
						EXCLUDED.last_position_ms,
						CASE
							WHEN EXCLUDED.duration_ms > 0
							THEN EXCLUDED.duration_ms
							ELSE podcast_listening_history.duration_ms
						END
					)
					ELSE EXCLUDED.last_position_ms
				END,

			duration_ms =
				CASE
					WHEN EXCLUDED.duration_ms > 0
					THEN EXCLUDED.duration_ms
					ELSE podcast_listening_history.duration_ms
				END,

			qualified_play_count =
				podcast_listening_history.qualified_play_count
				+ EXCLUDED.qualified_play_count,

			completed = EXCLUDED.completed,

			completed_at =
				CASE
					WHEN EXCLUDED.completed = TRUE
					THEN now()
					ELSE NULL
				END,

			last_played_at = now(),

			last_qualified_at =
				CASE
					WHEN EXCLUDED.qualified_play_count > 0
					THEN now()
					ELSE podcast_listening_history.last_qualified_at
				END,

			updated_at = now()
	`

	_, err := tx.Exec(
		ctx,
		query,
		session.UserID,
		session.EpisodeID,
		session.PositionMS,
		session.DurationMS,
		qualifiedIncrement,
		completed,
	)

	return err
}

// podcastHistoryQuery builds the shared SELECT for history
// payloads. extraPredicate is appended to the WHERE clause
// and must start with AND.
func podcastHistoryQuery(
	extraPredicate string,
) string {
	return `
		SELECT
			p.id,
			p.owner_user_id,
			p.title,
			p.slug,
			p.description,
			p.category,
			p.artwork_url,
			p.artwork_public_id,
			p.status,
			p.is_explicit,
			p.published_at,
			p.created_at,
			p.updated_at,

			e.id,
			e.podcast_id,
			e.title,
			e.slug,
			e.description,
			e.season_number,
			e.episode_number,
			e.episode_type,
			e.audio_url,
			e.audio_public_id,
			e.artwork_url,
			e.artwork_public_id,
			e.duration_ms,
			e.status,
			e.is_explicit,
			e.scheduled_at,
			e.published_at,
			e.created_at,
			e.updated_at,

			h.last_position_ms,
			h.duration_ms,
			h.qualified_play_count,
			h.completed,
			h.last_played_at,
			h.last_qualified_at,
			h.completed_at

		FROM podcast_listening_history AS h

		INNER JOIN podcast_episodes AS e
			ON e.id = h.episode_id

		INNER JOIN podcasts AS p
			ON p.id = e.podcast_id

		-- Only publicly playable episodes are returned: the
		-- listener can no longer resume archived or
		-- unpublished content, and episodes without audio
		-- cannot be played at all.
		WHERE e.status = 'PUBLISHED'
		  AND p.status = 'PUBLISHED'
		  AND NULLIF(
			BTRIM(e.audio_url),
			''
		  ) IS NOT NULL

		` + extraPredicate
}

func (r *PodcastPlaybackRepository) queryPodcastHistoryItems(
	ctx context.Context,
	query string,
	args ...any,
) ([]models.PodcastListeningHistoryItem, error) {
	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make(
		[]models.PodcastListeningHistoryItem,
		0,
	)

	for rows.Next() {
		var item models.PodcastListeningHistoryItem

		err := rows.Scan(
			&item.Podcast.ID,
			&item.Podcast.OwnerUserID,
			&item.Podcast.Title,
			&item.Podcast.Slug,
			&item.Podcast.Description,
			&item.Podcast.Category,
			&item.Podcast.ArtworkURL,
			&item.Podcast.ArtworkPublicID,
			&item.Podcast.Status,
			&item.Podcast.IsExplicit,
			&item.Podcast.PublishedAt,
			&item.Podcast.CreatedAt,
			&item.Podcast.UpdatedAt,

			&item.Episode.ID,
			&item.Episode.PodcastID,
			&item.Episode.Title,
			&item.Episode.Slug,
			&item.Episode.Description,
			&item.Episode.SeasonNumber,
			&item.Episode.EpisodeNumber,
			&item.Episode.EpisodeType,
			&item.Episode.AudioURL,
			&item.Episode.AudioPublicID,
			&item.Episode.ArtworkURL,
			&item.Episode.ArtworkPublicID,
			&item.Episode.DurationMS,
			&item.Episode.Status,
			&item.Episode.IsExplicit,
			&item.Episode.ScheduledAt,
			&item.Episode.PublishedAt,
			&item.Episode.CreatedAt,
			&item.Episode.UpdatedAt,

			&item.LastPositionMS,
			&item.DurationMS,
			&item.QualifiedPlayCount,
			&item.Completed,
			&item.LastPlayedAt,
			&item.LastQualifiedAt,
			&item.CompletedAt,
		)
		if err != nil {
			return nil, err
		}

		items = append(items, item)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return items, nil
}

// -----------------------------------------------------------------
// Scanning helpers
// -----------------------------------------------------------------

type podcastPlaybackSessionRow interface {
	Scan(dest ...any) error
}

func scanPodcastPlaybackSession(
	row podcastPlaybackSessionRow,
) (models.PodcastPlaybackSession, error) {
	var session models.PodcastPlaybackSession

	err := row.Scan(
		&session.ID,
		&session.UserID,
		&session.EpisodeID,
		&session.DurationMS,
		&session.PositionMS,
		&session.ListenedMS,
		&session.Qualified,
		&session.QualifiedAt,
		&session.Completed,
		&session.CompletedAt,
		&session.StartedAt,
		&session.LastActivityAt,
		&session.CreatedAt,
		&session.UpdatedAt,
	)
	if err != nil {
		return models.PodcastPlaybackSession{}, err
	}

	return session, nil
}
