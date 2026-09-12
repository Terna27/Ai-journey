package repository

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"music-api/internal/models"
)

var (
	ErrPodcastLiveSessionNotFound = errors.New(
		"podcast live session not found",
	)

	// ErrPodcastLiveEpisodeTaken: the episode already has a
	// non-cancelled live session (unique index).
	ErrPodcastLiveEpisodeTaken = errors.New(
		"episode already has a live session",
	)

	// ErrPodcastLiveRoomNameConflict: generated room name
	// collided with an existing room (defensive; names are
	// random UUIDs).
	ErrPodcastLiveRoomNameConflict = errors.New(
		"live room name already exists",
	)

	ErrPodcastLiveSessionNotScheduled = errors.New(
		"podcast live session is not scheduled",
	)

	ErrPodcastLiveSessionNotLive = errors.New(
		"podcast live session is not live",
	)

	ErrPodcastLiveSessionNotCancellable = errors.New(
		"podcast live session cannot be cancelled",
	)

	// ErrPodcastLiveSessionNotEnded: the recording publish
	// flow only accepts sessions that finished (ENDED).
	ErrPodcastLiveSessionNotEnded = errors.New(
		"podcast live session has not ended",
	)

	// ErrPodcastLiveRecordingNotReady: no finished recording
	// exists to publish as episode audio.
	ErrPodcastLiveRecordingNotReady = errors.New(
		"live recording is not ready",
	)

	// ErrPodcastLiveEpisodeInconsistent: the episode row was
	// not in the state the session transition expected. The
	// transaction is rolled back so both stay consistent.
	ErrPodcastLiveEpisodeInconsistent = errors.New(
		"podcast episode state is inconsistent with its live session",
	)
)

type PodcastLiveRepository struct {
	db *pgxpool.Pool
}

func NewPodcastLiveRepository(
	db *pgxpool.Pool,
) *PodcastLiveRepository {
	return &PodcastLiveRepository{
		db: db,
	}
}

// podcastLiveSessionColumns is the shared column list of a
// podcast live session row.
const podcastLiveSessionColumns = `
	id::text,
	episode_id,
	host_user_id,
	provider,
	provider_room_name,
	provider_room_sid,
	status,
	scheduled_start_at,
	started_at,
	ended_at,
	recording_status,
	recording_provider_asset_id,
	recording_url,
	created_at,
	updated_at
`

// podcastLiveSessionColumnsAliased is the full live-session
// projection qualified with the "l" table alias. It is used
// in joined queries to prevent ambiguous column references.
const podcastLiveSessionColumnsAliased = `
	l.id::text,
	l.episode_id,
	l.host_user_id,
	l.provider,
	l.provider_room_name,
	l.provider_room_sid,
	l.status,
	l.scheduled_start_at,
	l.started_at,
	l.ended_at,
	l.recording_status,
	l.recording_provider_asset_id,
	l.recording_url,
	l.created_at,
	l.updated_at
`

// podcastLivePublicColumns is the public projection of a live
// session row, as used by discovery payloads.
const podcastLivePublicColumns = `
	l.id::text,
	l.episode_id,
	l.status,
	l.scheduled_start_at,
	l.started_at,
	l.ended_at,
	l.recording_status
`

// ScheduleLiveSession atomically:
//
//  1. transitions the episode DRAFT -> SCHEDULED and stamps
//     its scheduled_at,
//  2. inserts the live session (SCHEDULED).
//
// The service layer supplies the UUID, the provider room
// name, and the schedule; the host is the caller by
// construction.
//
// The unique partial index on episode_id (status <> CANCELLED)
// rejects a second live session for an episode that already
// has one, and the conditional episode UPDATE rejects any
// episode that is no longer a draft. Either failure rolls the
// whole transaction back.
func (r *PodcastLiveRepository) ScheduleLiveSession(
	ctx context.Context,
	session models.PodcastLiveSession,
) (models.PodcastLiveSession, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return models.PodcastLiveSession{}, err
	}
	defer tx.Rollback(ctx)

	tag, err := tx.Exec(
		ctx,
		`
			UPDATE podcast_episodes
			SET
				status = 'SCHEDULED',
				scheduled_at = $2,
				updated_at = NOW()
			WHERE id = $1
			  AND status = 'DRAFT'
		`,
		session.EpisodeID,
		session.ScheduledStartAt,
	)
	if err != nil {
		return models.PodcastLiveSession{}, err
	}

	if tag.RowsAffected() == 0 {
		return models.PodcastLiveSession{},
			ErrPodcastLiveEpisodeInconsistent
	}

	created, err := scanPodcastLiveSession(
		tx.QueryRow(
			ctx,
			`
				INSERT INTO podcast_live_sessions (
					id,
					episode_id,
					host_user_id,
					provider,
					provider_room_name,
					status,
					scheduled_start_at
				)
				VALUES (
					$1::uuid,
					$2,
					$3,
					$4,
					$5,
					$6,
					$7
				)
				RETURNING
			`+podcastLiveSessionColumns,
			session.ID,
			session.EpisodeID,
			session.HostUserID,
			session.Provider,
			session.ProviderRoomName,
			session.Status,
			session.ScheduledStartAt,
		),
	)
	if err != nil {
		return models.PodcastLiveSession{},
			mapPodcastLiveDatabaseError(err)
	}

	if err := tx.Commit(ctx); err != nil {
		return models.PodcastLiveSession{}, err
	}

	return created, nil
}

// GetLiveSessionByID returns one live session regardless of
// ownership. Callers must apply their own authorization.
func (r *PodcastLiveRepository) GetLiveSessionByID(
	ctx context.Context,
	sessionID string,
) (models.PodcastLiveSession, error) {
	query := `
		SELECT
	` + podcastLiveSessionColumns + `
		FROM podcast_live_sessions
		WHERE id = $1::uuid
	`

	session, err := scanPodcastLiveSession(
		r.db.QueryRow(
			ctx,
			query,
			sessionID,
		),
	)
	if err != nil {
		return models.PodcastLiveSession{},
			mapPodcastLiveNotFound(err)
	}

	return session, nil
}

// GetLiveSessionByEpisodeID returns the episode's most recent
// live session (cancelled sessions keep their rows, so the
// latest row is the meaningful one).
func (r *PodcastLiveRepository) GetLiveSessionByEpisodeID(
	ctx context.Context,
	episodeID int64,
) (models.PodcastLiveSession, error) {
	query := `
		SELECT
	` + podcastLiveSessionColumns + `
		FROM podcast_live_sessions
		WHERE episode_id = $1
		ORDER BY created_at DESC, id
		LIMIT 1
	`

	session, err := scanPodcastLiveSession(
		r.db.QueryRow(
			ctx,
			query,
			episodeID,
		),
	)
	if err != nil {
		return models.PodcastLiveSession{},
			mapPodcastLiveNotFound(err)
	}

	return session, nil
}

// GetOwnedLiveSession returns a live session only when it
// belongs to hostUserID. Another host's session is
// indistinguishable from a missing one, so ownership is never
// leaked.
func (r *PodcastLiveRepository) GetOwnedLiveSession(
	ctx context.Context,
	sessionID string,
	hostUserID int,
) (models.PodcastLiveSession, error) {
	query := `
		SELECT
	` + podcastLiveSessionColumns + `
		FROM podcast_live_sessions
		WHERE id = $1::uuid
		  AND host_user_id = $2
	`

	session, err := scanPodcastLiveSession(
		r.db.QueryRow(
			ctx,
			query,
			sessionID,
			hostUserID,
		),
	)
	if err != nil {
		return models.PodcastLiveSession{},
			mapPodcastLiveNotFound(err)
	}

	return session, nil
}

// GetOwnedLiveDetails returns the owner-facing view of a live
// session: the full runtime record joined with its episode and
// podcast.
func (r *PodcastLiveRepository) GetOwnedLiveDetails(
	ctx context.Context,
	sessionID string,
	hostUserID int,
) (models.PodcastLiveDetails, error) {
	query := podcastLiveOwnerQuery(`
		  AND l.id = $1::uuid
		  AND l.host_user_id = $2
	`)

	details, err := scanPodcastLiveDetails(
		r.db.QueryRow(
			ctx,
			query,
			sessionID,
			hostUserID,
		),
	)
	if err != nil {
		return models.PodcastLiveDetails{},
			mapPodcastLiveNotFound(err)
	}

	return details, nil
}

// ListUpcomingLiveSessions returns scheduled broadcasts of
// published podcasts, soonest first.
//
// Only publicly visible events are returned: the parent
// podcast must be PUBLISHED.
func (r *PodcastLiveRepository) ListUpcomingLiveSessions(
	ctx context.Context,
	limit int,
	offset int,
) ([]models.PodcastLiveBroadcast, error) {
	query := podcastLiveBroadcastQuery(`
		  AND l.status = 'SCHEDULED'
		  AND e.status = 'SCHEDULED'
	`) + `
		ORDER BY
			l.scheduled_start_at ASC,
			l.id
		LIMIT $1
		OFFSET $2
	`

	return r.queryPodcastLiveBroadcasts(
		ctx,
		query,
		limit,
		offset,
	)
}

// ListCurrentlyLiveSessions returns broadcasts that are live
// right now, most recently started first.
func (r *PodcastLiveRepository) ListCurrentlyLiveSessions(
	ctx context.Context,
	limit int,
	offset int,
) ([]models.PodcastLiveBroadcast, error) {
	query := podcastLiveBroadcastQuery(`
		  AND l.status = 'LIVE'
		  AND e.status = 'LIVE'
	`) + `
		ORDER BY
			l.started_at DESC,
			l.id
		LIMIT $1
		OFFSET $2
	`

	return r.queryPodcastLiveBroadcasts(
		ctx,
		query,
		limit,
		offset,
	)
}

// GetPublicLiveSessionByID returns the public broadcast view
// of one live session.
//
// Cancelled sessions and sessions of unpublished podcasts are
// indistinguishable from missing ones.
func (r *PodcastLiveRepository) GetPublicLiveSessionByID(
	ctx context.Context,
	sessionID string,
) (models.PodcastLiveBroadcast, error) {
	query := podcastLiveBroadcastQuery(`
		  AND l.id = $1::uuid
		  AND l.status IN ('SCHEDULED', 'LIVE', 'ENDED')
	`)

	broadcast, err := scanPodcastLiveBroadcast(
		r.db.QueryRow(
			ctx,
			query,
			sessionID,
		),
	)
	if err != nil {
		return models.PodcastLiveBroadcast{},
			mapPodcastLiveNotFound(err)
	}

	return broadcast, nil
}

// StartLiveSession atomically transitions the live session
// SCHEDULED -> LIVE and the episode SCHEDULED -> LIVE.
//
// started_at is stamped server-side. Any wrong-state input
// rolls the whole transaction back, so the session and episode
// can never disagree.
func (r *PodcastLiveRepository) StartLiveSession(
	ctx context.Context,
	sessionID string,
	hostUserID int,
) (models.PodcastLiveSession, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return models.PodcastLiveSession{}, err
	}
	defer tx.Rollback(ctx)

	current, err := getPodcastLiveSessionForUpdate(
		ctx,
		tx,
		sessionID,
		hostUserID,
	)
	if err != nil {
		return models.PodcastLiveSession{}, err
	}

	if current.Status != models.PodcastLiveStatusScheduled {
		return models.PodcastLiveSession{},
			ErrPodcastLiveSessionNotScheduled
	}

	updated, err := scanPodcastLiveSession(
		tx.QueryRow(
			ctx,
			`
				UPDATE podcast_live_sessions
				SET
					status = 'LIVE',
					started_at = NOW(),
					updated_at = NOW()
				WHERE id = $1::uuid
				  AND host_user_id = $2
				  AND status = 'SCHEDULED'
				RETURNING
			`+podcastLiveSessionColumns,
			sessionID,
			hostUserID,
		),
	)
	if err != nil {
		return models.PodcastLiveSession{}, err
	}

	tag, err := tx.Exec(
		ctx,
		`
			UPDATE podcast_episodes
			SET
				status = 'LIVE',
				updated_at = NOW()
			WHERE id = $1
			  AND status = 'SCHEDULED'
		`,
		current.EpisodeID,
	)
	if err != nil {
		return models.PodcastLiveSession{}, err
	}

	if tag.RowsAffected() == 0 {
		return models.PodcastLiveSession{},
			ErrPodcastLiveEpisodeInconsistent
	}

	if err := tx.Commit(ctx); err != nil {
		return models.PodcastLiveSession{}, err
	}

	return updated, nil
}

// EndLiveSession atomically transitions the live session
// LIVE -> ENDED and the episode LIVE -> ENDED.
//
// ended_at is stamped server-side and can never precede
// started_at (CHECK constraint).
func (r *PodcastLiveRepository) EndLiveSession(
	ctx context.Context,
	sessionID string,
	hostUserID int,
) (models.PodcastLiveSession, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return models.PodcastLiveSession{}, err
	}
	defer tx.Rollback(ctx)

	current, err := getPodcastLiveSessionForUpdate(
		ctx,
		tx,
		sessionID,
		hostUserID,
	)
	if err != nil {
		return models.PodcastLiveSession{}, err
	}

	if current.Status != models.PodcastLiveStatusLive {
		return models.PodcastLiveSession{},
			ErrPodcastLiveSessionNotLive
	}

	updated, err := scanPodcastLiveSession(
		tx.QueryRow(
			ctx,
			`
				UPDATE podcast_live_sessions
				SET
					status = 'ENDED',
					ended_at = NOW(),
					updated_at = NOW()
				WHERE id = $1::uuid
				  AND host_user_id = $2
				  AND status = 'LIVE'
				RETURNING
			`+podcastLiveSessionColumns,
			sessionID,
			hostUserID,
		),
	)
	if err != nil {
		return models.PodcastLiveSession{}, err
	}

	tag, err := tx.Exec(
		ctx,
		`
			UPDATE podcast_episodes
			SET
				status = 'ENDED',
				updated_at = NOW()
			WHERE id = $1
			  AND status = 'LIVE'
		`,
		current.EpisodeID,
	)
	if err != nil {
		return models.PodcastLiveSession{}, err
	}

	if tag.RowsAffected() == 0 {
		return models.PodcastLiveSession{},
			ErrPodcastLiveEpisodeInconsistent
	}

	if err := tx.Commit(ctx); err != nil {
		return models.PodcastLiveSession{}, err
	}

	return updated, nil
}

// CancelLiveSession atomically cancels a SCHEDULED live
// session and returns its episode to DRAFT, clearing the
// episode's scheduled_at.
//
// A LIVE or ENDED session can never be cancelled.
func (r *PodcastLiveRepository) CancelLiveSession(
	ctx context.Context,
	sessionID string,
	hostUserID int,
) (models.PodcastLiveSession, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return models.PodcastLiveSession{}, err
	}
	defer tx.Rollback(ctx)

	current, err := getPodcastLiveSessionForUpdate(
		ctx,
		tx,
		sessionID,
		hostUserID,
	)
	if err != nil {
		return models.PodcastLiveSession{}, err
	}

	if current.Status != models.PodcastLiveStatusScheduled {
		return models.PodcastLiveSession{},
			ErrPodcastLiveSessionNotCancellable
	}

	updated, err := scanPodcastLiveSession(
		tx.QueryRow(
			ctx,
			`
				UPDATE podcast_live_sessions
				SET
					status = 'CANCELLED',
					updated_at = NOW()
				WHERE id = $1::uuid
				  AND host_user_id = $2
				  AND status = 'SCHEDULED'
				RETURNING
			`+podcastLiveSessionColumns,
			sessionID,
			hostUserID,
		),
	)
	if err != nil {
		return models.PodcastLiveSession{}, err
	}

	tag, err := tx.Exec(
		ctx,
		`
			UPDATE podcast_episodes
			SET
				status = 'DRAFT',
				scheduled_at = NULL,
				updated_at = NOW()
			WHERE id = $1
			  AND status = 'SCHEDULED'
		`,
		current.EpisodeID,
	)
	if err != nil {
		return models.PodcastLiveSession{}, err
	}

	if tag.RowsAffected() == 0 {
		return models.PodcastLiveSession{},
			ErrPodcastLiveEpisodeInconsistent
	}

	if err := tx.Commit(ctx); err != nil {
		return models.PodcastLiveSession{}, err
	}

	return updated, nil
}

// SetProviderRoomSID records the provider-assigned room SID
// once the room exists.
func (r *PodcastLiveRepository) SetProviderRoomSID(
	ctx context.Context,
	sessionID string,
	hostUserID int,
	roomSID string,
) (models.PodcastLiveSession, error) {
	session, err := scanPodcastLiveSession(
		r.db.QueryRow(
			ctx,
			`
				UPDATE podcast_live_sessions
				SET
					provider_room_sid = $3,
					updated_at = NOW()
				WHERE id = $1::uuid
				  AND host_user_id = $2
				RETURNING
			`+podcastLiveSessionColumns,
			sessionID,
			hostUserID,
			roomSID,
		),
	)
	if err != nil {
		return models.PodcastLiveSession{},
			mapPodcastLiveNotFound(err)
	}

	return session, nil
}

// UpdateLiveRecordingStatus advances the recording state of a
// live session. A nil asset ID keeps the previously stored
// one.
func (r *PodcastLiveRepository) UpdateLiveRecordingStatus(
	ctx context.Context,
	sessionID string,
	hostUserID int,
	status models.PodcastLiveRecordingStatus,
	providerAssetID *string,
) (models.PodcastLiveSession, error) {
	session, err := scanPodcastLiveSession(
		r.db.QueryRow(
			ctx,
			`
				UPDATE podcast_live_sessions
				SET
					recording_status = $3,
					recording_provider_asset_id =
						COALESCE(
							$4,
							recording_provider_asset_id
						),
					updated_at = NOW()
				WHERE id = $1::uuid
				  AND host_user_id = $2
				RETURNING
			`+podcastLiveSessionColumns,
			sessionID,
			hostUserID,
			status,
			providerAssetID,
		),
	)
	if err != nil {
		return models.PodcastLiveSession{},
			mapPodcastLiveNotFound(err)
	}

	return session, nil
}

// SetLiveRecordingResult records a finished recording (status
// READY plus its playable URL) for an owned live session.
func (r *PodcastLiveRepository) SetLiveRecordingResult(
	ctx context.Context,
	sessionID string,
	hostUserID int,
	recordingURL string,
) (models.PodcastLiveSession, error) {
	session, err := scanPodcastLiveSession(
		r.db.QueryRow(
			ctx,
			`
				UPDATE podcast_live_sessions
				SET
					recording_status = 'READY',
					recording_url = $3,
					updated_at = NOW()
				WHERE id = $1::uuid
				  AND host_user_id = $2
				RETURNING
			`+podcastLiveSessionColumns,
			sessionID,
			hostUserID,
			recordingURL,
		),
	)
	if err != nil {
		return models.PodcastLiveSession{},
			mapPodcastLiveNotFound(err)
	}

	return session, nil
}

// PublishLiveRecording atomically turns the finished
// recording of an ENDED live session into the episode's
// published audio:
//
//  1. the episode's audio_url is set to the recording URL,
//  2. the episode transitions ENDED -> PUBLISHED with
//     published_at stamped server-side,
//  3. the episode's scheduled_at is cleared.
//
// The episode keeps its regular playback pipeline (Phase 4.1):
// once published it is just a normal on-demand episode.
func (r *PodcastLiveRepository) PublishLiveRecording(
	ctx context.Context,
	sessionID string,
	hostUserID int,
) (models.PodcastLiveSession, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return models.PodcastLiveSession{}, err
	}
	defer tx.Rollback(ctx)

	current, err := getPodcastLiveSessionForUpdate(
		ctx,
		tx,
		sessionID,
		hostUserID,
	)
	if err != nil {
		return models.PodcastLiveSession{}, err
	}

	if current.Status != models.PodcastLiveStatusEnded {
		return models.PodcastLiveSession{},
			ErrPodcastLiveSessionNotEnded
	}

	if current.RecordingStatus != models.PodcastLiveRecordingReady ||
		current.RecordingURL == nil ||
		*current.RecordingURL == "" {
		return models.PodcastLiveSession{},
			ErrPodcastLiveRecordingNotReady
	}

	tag, err := tx.Exec(
		ctx,
		`
			UPDATE podcast_episodes
			SET
				audio_url = $2,
				status = 'PUBLISHED',
				published_at = NOW(),
				scheduled_at = NULL,
				updated_at = NOW()
			WHERE id = $1
			  AND status = 'ENDED'
		`,
		current.EpisodeID,
		*current.RecordingURL,
	)
	if err != nil {
		return models.PodcastLiveSession{}, err
	}

	if tag.RowsAffected() == 0 {
		return models.PodcastLiveSession{},
			ErrPodcastLiveEpisodeInconsistent
	}

	if err := tx.Commit(ctx); err != nil {
		return models.PodcastLiveSession{}, err
	}

	return current, nil
}

// -----------------------------------------------------------------
// Internal helpers
// -----------------------------------------------------------------

func getPodcastLiveSessionForUpdate(
	ctx context.Context,
	tx pgx.Tx,
	sessionID string,
	hostUserID int,
) (models.PodcastLiveSession, error) {
	query := `
		SELECT
	` + podcastLiveSessionColumns + `
		FROM podcast_live_sessions
		WHERE id = $1::uuid
		  AND host_user_id = $2
		FOR UPDATE
	`

	session, err := scanPodcastLiveSession(
		tx.QueryRow(
			ctx,
			query,
			sessionID,
			hostUserID,
		),
	)
	if err != nil {
		return models.PodcastLiveSession{},
			mapPodcastLiveNotFound(err)
	}

	return session, nil
}

// podcastLiveBroadcastQuery builds the shared SELECT for
// PUBLIC live broadcast payloads: the live session joined with
// its episode and podcast. extraPredicate is appended to the
// WHERE clause and must start with AND.
//
// Only published podcasts are ever publicly visible, and the
// episode must still exist with playable state.
func podcastLiveBroadcastQuery(
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

		` + podcastLivePublicColumns + `

		FROM podcast_live_sessions AS l

		INNER JOIN podcast_episodes AS e
			ON e.id = l.episode_id

		INNER JOIN podcasts AS p
			ON p.id = e.podcast_id

		WHERE p.status = 'PUBLISHED'
		  AND e.status IN (
			'SCHEDULED',
			'LIVE',
			'ENDED'
		  )

		` + extraPredicate
}

// podcastLiveOwnerQuery is the owner-facing variant: the full
// live session record joined with episode and podcast, with no
// publication filter (owners see their drafts).
func podcastLiveOwnerQuery(
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

		` + podcastLiveSessionColumnsAliased + `

		FROM podcast_live_sessions AS l

		INNER JOIN podcast_episodes AS e
			ON e.id = l.episode_id

		INNER JOIN podcasts AS p
			ON p.id = e.podcast_id

		WHERE TRUE
		` + extraPredicate
}

func (r *PodcastLiveRepository) queryPodcastLiveBroadcasts(
	ctx context.Context,
	query string,
	args ...any,
) ([]models.PodcastLiveBroadcast, error) {
	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	broadcasts := make(
		[]models.PodcastLiveBroadcast,
		0,
	)

	for rows.Next() {
		broadcast, err := scanPodcastLiveBroadcast(rows)
		if err != nil {
			return nil, err
		}

		broadcasts = append(
			broadcasts,
			broadcast,
		)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return broadcasts, nil
}

// -----------------------------------------------------------------
// Scanning helpers
// -----------------------------------------------------------------

type podcastLiveSessionRow interface {
	Scan(dest ...any) error
}

func scanPodcastLiveSession(
	row podcastLiveSessionRow,
) (models.PodcastLiveSession, error) {
	var session models.PodcastLiveSession

	err := row.Scan(
		&session.ID,
		&session.EpisodeID,
		&session.HostUserID,
		&session.Provider,
		&session.ProviderRoomName,
		&session.ProviderRoomSID,
		&session.Status,
		&session.ScheduledStartAt,
		&session.StartedAt,
		&session.EndedAt,
		&session.RecordingStatus,
		&session.RecordingProviderAssetID,
		&session.RecordingURL,
		&session.CreatedAt,
		&session.UpdatedAt,
	)
	if err != nil {
		return models.PodcastLiveSession{}, err
	}

	return session, nil
}

func scanPodcastLiveBroadcast(
	row podcastLiveSessionRow,
) (models.PodcastLiveBroadcast, error) {
	var broadcast models.PodcastLiveBroadcast

	err := row.Scan(
		&broadcast.Podcast.ID,
		&broadcast.Podcast.OwnerUserID,
		&broadcast.Podcast.Title,
		&broadcast.Podcast.Slug,
		&broadcast.Podcast.Description,
		&broadcast.Podcast.Category,
		&broadcast.Podcast.ArtworkURL,
		&broadcast.Podcast.ArtworkPublicID,
		&broadcast.Podcast.Status,
		&broadcast.Podcast.IsExplicit,
		&broadcast.Podcast.PublishedAt,
		&broadcast.Podcast.CreatedAt,
		&broadcast.Podcast.UpdatedAt,

		&broadcast.Episode.ID,
		&broadcast.Episode.PodcastID,
		&broadcast.Episode.Title,
		&broadcast.Episode.Slug,
		&broadcast.Episode.Description,
		&broadcast.Episode.SeasonNumber,
		&broadcast.Episode.EpisodeNumber,
		&broadcast.Episode.EpisodeType,
		&broadcast.Episode.AudioURL,
		&broadcast.Episode.AudioPublicID,
		&broadcast.Episode.ArtworkURL,
		&broadcast.Episode.ArtworkPublicID,
		&broadcast.Episode.DurationMS,
		&broadcast.Episode.Status,
		&broadcast.Episode.IsExplicit,
		&broadcast.Episode.ScheduledAt,
		&broadcast.Episode.PublishedAt,
		&broadcast.Episode.CreatedAt,
		&broadcast.Episode.UpdatedAt,

		&broadcast.LiveSession.ID,
		&broadcast.LiveSession.EpisodeID,
		&broadcast.LiveSession.Status,
		&broadcast.LiveSession.ScheduledStartAt,
		&broadcast.LiveSession.StartedAt,
		&broadcast.LiveSession.EndedAt,
		&broadcast.LiveSession.RecordingStatus,
	)
	if err != nil {
		return models.PodcastLiveBroadcast{}, err
	}

	return broadcast, nil
}

func scanPodcastLiveDetails(
	row podcastLiveSessionRow,
) (models.PodcastLiveDetails, error) {
	var details models.PodcastLiveDetails

	err := row.Scan(
		&details.Podcast.ID,
		&details.Podcast.OwnerUserID,
		&details.Podcast.Title,
		&details.Podcast.Slug,
		&details.Podcast.Description,
		&details.Podcast.Category,
		&details.Podcast.ArtworkURL,
		&details.Podcast.ArtworkPublicID,
		&details.Podcast.Status,
		&details.Podcast.IsExplicit,
		&details.Podcast.PublishedAt,
		&details.Podcast.CreatedAt,
		&details.Podcast.UpdatedAt,

		&details.Episode.ID,
		&details.Episode.PodcastID,
		&details.Episode.Title,
		&details.Episode.Slug,
		&details.Episode.Description,
		&details.Episode.SeasonNumber,
		&details.Episode.EpisodeNumber,
		&details.Episode.EpisodeType,
		&details.Episode.AudioURL,
		&details.Episode.AudioPublicID,
		&details.Episode.ArtworkURL,
		&details.Episode.ArtworkPublicID,
		&details.Episode.DurationMS,
		&details.Episode.Status,
		&details.Episode.IsExplicit,
		&details.Episode.ScheduledAt,
		&details.Episode.PublishedAt,
		&details.Episode.CreatedAt,
		&details.Episode.UpdatedAt,

		&details.LiveSession.ID,
		&details.LiveSession.EpisodeID,
		&details.LiveSession.HostUserID,
		&details.LiveSession.Provider,
		&details.LiveSession.ProviderRoomName,
		&details.LiveSession.ProviderRoomSID,
		&details.LiveSession.Status,
		&details.LiveSession.ScheduledStartAt,
		&details.LiveSession.StartedAt,
		&details.LiveSession.EndedAt,
		&details.LiveSession.RecordingStatus,
		&details.LiveSession.RecordingProviderAssetID,
		&details.LiveSession.RecordingURL,
		&details.LiveSession.CreatedAt,
		&details.LiveSession.UpdatedAt,
	)
	if err != nil {
		return models.PodcastLiveDetails{}, err
	}

	return details, nil
}

func mapPodcastLiveNotFound(
	err error,
) error {
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrPodcastLiveSessionNotFound
	}

	return err
}

func mapPodcastLiveDatabaseError(
	err error,
) error {
	var pgErr *pgconn.PgError

	if !errors.As(err, &pgErr) {
		return err
	}

	if pgErr.Code != "23505" {
		return err
	}

	switch pgErr.ConstraintName {
	case "idx_podcast_live_sessions_episode_unique":
		return ErrPodcastLiveEpisodeTaken

	case "idx_podcast_live_sessions_room_unique":
		return ErrPodcastLiveRoomNameConflict

	default:
		return err
	}
}
