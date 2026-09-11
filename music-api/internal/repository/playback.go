package repository

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"music-api/internal/models"
)

var (
	ErrPlaybackSessionNotFound = errors.New(
		"playback session not found",
	)

	ErrPlaybackSessionCompleted = errors.New(
		"playback session is already completed",
	)
)

type PlaybackRepository struct {
	DB *pgxpool.Pool
}

func NewPlaybackRepository(
	db *pgxpool.Pool,
) *PlaybackRepository {
	return &PlaybackRepository{
		DB: db,
	}
}

// CreateSession creates a new playback session.
//
// The session ID is supplied by the service layer.
// We use UUIDs so playback-session identifiers are
// difficult to enumerate and safe to expose through
// authenticated API routes.
func (r *PlaybackRepository) CreateSession(
	ctx context.Context,
	sessionID string,
	userID int,
	musicID int,
	durationMS int64,
	positionMS int64,
) (models.PlaybackSession, error) {
	query := `
		INSERT INTO playback_sessions (
			id,
			user_id,
			music_id,
			duration_ms,
			position_ms
		)
		SELECT
			$1::uuid,
			$2::bigint,
			m.id,
			$4::bigint,
			LEAST(
				$5::bigint,
				CASE
					WHEN $4::bigint > 0
					THEN $4::bigint
					ELSE $5::bigint
				END
			)
		FROM music AS m
		WHERE m.id = $3::bigint
		  AND NULLIF(
			BTRIM(m.audio_url),
			''
		  ) IS NOT NULL
		RETURNING
			id::text,
			user_id,
			music_id,
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

	session, err := scanPlaybackSession(
		r.DB.QueryRow(
			ctx,
			query,
			sessionID,
			userID,
			musicID,
			durationMS,
			positionMS,
		),
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return models.PlaybackSession{}, pgx.ErrNoRows
	}

	if err != nil {
		return models.PlaybackSession{}, err
	}

	return session, nil
}

// GetSession returns one playback session only when it
// belongs to userID.
//
// Returning the same not-found error for a missing session
// and another user's session avoids leaking ownership.
func (r *PlaybackRepository) GetSession(
	ctx context.Context,
	sessionID string,
	userID int,
) (models.PlaybackSession, error) {
	query := `
		SELECT
			id::text,
			user_id,
			music_id,
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
		FROM playback_sessions
		WHERE id = $1::uuid
		  AND user_id = $2
	`

	session, err := scanPlaybackSession(
		r.DB.QueryRow(
			ctx,
			query,
			sessionID,
			userID,
		),
	)

	if errors.Is(err, pgx.ErrNoRows) {
		return models.PlaybackSession{},
			ErrPlaybackSessionNotFound
	}

	if err != nil {
		return models.PlaybackSession{}, err
	}

	return session, nil
}

// UpdateProgress atomically:
//
//  1. locks the playback session (SELECT ... FOR UPDATE),
//  2. validates ownership,
//  3. clamps reported listened time against server
//     wall-clock elapsed time (anti-manipulation),
//  4. updates playback progress using only accepted
//     server-side values,
//  5. evaluates qualification from the ACCEPTED value,
//     never raw client input,
//  6. transitions the session to qualified at most once,
//  7. upserts listening history,
//  8. increments qualified_play_count only on the first
//     qualification transition.
//
// Because qualification is derived from the same clamped
// value that is persisted, and the session row is locked
// for the duration of the transaction, concurrent requests
// can never qualify the same session twice.
func (r *PlaybackRepository) UpdateProgress(
	ctx context.Context,
	sessionID string,
	userID int,
	positionMS int64,
	durationMS int64,
	listenedMS int64,
) (models.PlaybackSession, error) {
	tx, err := r.DB.Begin(ctx)
	if err != nil {
		return models.PlaybackSession{}, err
	}
	defer tx.Rollback(ctx)

	current, err := getPlaybackSessionForUpdate(
		ctx,
		tx,
		sessionID,
		userID,
	)
	if err != nil {
		return models.PlaybackSession{}, err
	}

	if current.Completed {
		return models.PlaybackSession{},
			ErrPlaybackSessionCompleted
	}

	next := resolveNextProgress(
		current,
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

	query := `
		UPDATE playback_sessions
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
			last_activity_at = now(),
			updated_at = now()
		WHERE id = $5::uuid
		  AND user_id = $6
		RETURNING
			id::text,
			user_id,
			music_id,
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

	updated, err := scanPlaybackSession(
		tx.QueryRow(
			ctx,
			query,
			next.durationMS,
			next.positionMS,
			next.listenedMS,
			shouldQualify,
			sessionID,
			userID,
		),
	)
	if err != nil {
		return models.PlaybackSession{}, err
	}

	err = upsertListeningHistory(
		ctx,
		tx,
		updated,
		justQualified,
	)
	if err != nil {
		return models.PlaybackSession{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return models.PlaybackSession{}, err
	}

	return updated, nil
}

// CompleteSession finalizes a playback session.
//
// Qualification is still evaluated from the accepted
// clamped listened time so a short track that reaches its
// threshold at the very end does not lose its legitimate
// qualified play.
//
// Completion is idempotent: a second call returns the
// already completed session without touching listening
// history or play counts again.
func (r *PlaybackRepository) CompleteSession(
	ctx context.Context,
	sessionID string,
	userID int,
	positionMS int64,
	durationMS int64,
	listenedMS int64,
) (models.PlaybackSession, error) {
	tx, err := r.DB.Begin(ctx)
	if err != nil {
		return models.PlaybackSession{}, err
	}
	defer tx.Rollback(ctx)

	current, err := getPlaybackSessionForUpdate(
		ctx,
		tx,
		sessionID,
		userID,
	)
	if err != nil {
		return models.PlaybackSession{}, err
	}

	// Completion is idempotent. Returning the already
	// completed session makes duplicate browser completion
	// events harmless.
	if current.Completed {
		return current, nil
	}

	next := resolveNextProgress(
		current,
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

	query := `
		UPDATE playback_sessions
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
			completed = TRUE,
			completed_at = now(),
			last_activity_at = now(),
			updated_at = now()
		WHERE id = $5::uuid
		  AND user_id = $6
		RETURNING
			id::text,
			user_id,
			music_id,
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

	updated, err := scanPlaybackSession(
		tx.QueryRow(
			ctx,
			query,
			next.durationMS,
			next.positionMS,
			next.listenedMS,
			shouldQualify,
			sessionID,
			userID,
		),
	)
	if err != nil {
		return models.PlaybackSession{}, err
	}

	err = upsertListeningHistory(
		ctx,
		tx,
		updated,
		justQualified,
	)
	if err != nil {
		return models.PlaybackSession{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return models.PlaybackSession{}, err
	}

	return updated, nil
}

// GetListeningHistory returns the user's most recently
// played tracks together with their latest playback state.
//
// limit/offset are supplied by the service layer after
// validation.
func (r *PlaybackRepository) GetListeningHistory(
	ctx context.Context,
	userID int,
	limit int,
	offset int,
) ([]models.ListeningHistoryItem, error) {
	query := `
		SELECT
			m.id,
			m.artist_id,
			m.release_id,
			m.track_number,
			m.artist_name,
			m.song_title,
			m.genre,
			m.image_url,
			m.image_public_id,
			m.audio_url,
			m.audio_public_id,
			m.audio_key,
			m.likes,
			m.loves,
			m.rating,
			m.date_posted,

			h.last_position_ms,
			h.duration_ms,
			h.qualified_play_count,
			h.last_played_at,
			h.last_qualified_at

		FROM listening_history AS h

		INNER JOIN music AS m
			ON m.id = h.music_id

		WHERE h.user_id = $1

		ORDER BY
			h.last_played_at DESC,
			m.id DESC

		LIMIT $2
		OFFSET $3
	`

	rows, err := r.DB.Query(
		ctx,
		query,
		userID,
		limit,
		offset,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make(
		[]models.ListeningHistoryItem,
		0,
	)

	for rows.Next() {
		var item models.ListeningHistoryItem

		err := rows.Scan(
			&item.Music.ID,
			&item.Music.ArtistID,
			&item.Music.ReleaseID,
			&item.Music.TrackNumber,
			&item.Music.ArtistName,
			&item.Music.SongTitle,
			&item.Music.Genre,
			&item.Music.ImageURL,
			&item.Music.ImagePublicID,
			&item.Music.AudioURL,
			&item.Music.AudioPublicID,
			&item.Music.AudioKey,
			&item.Music.Likes,
			&item.Music.Loves,
			&item.Music.Rating,
			&item.Music.DatePosted,

			&item.LastPositionMS,
			&item.DurationMS,
			&item.QualifiedPlayCount,
			&item.LastPlayedAt,
			&item.LastQualifiedAt,
		)
		if err != nil {
			return nil, err
		}

		items = append(
			items,
			item,
		)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return items, nil
}

// -----------------------------------------------------------------
// Transaction helpers
// -----------------------------------------------------------------

// nextProgress is the fully accepted server-side progress
// state that may be persisted for a playback session.
type nextProgress struct {
	durationMS int64
	positionMS int64
	listenedMS int64
}

// progressBaseline is the locked, already-persisted state
// a progress call is validated against. Both music and
// podcast playback sessions map onto it so the accepted-
// value rules stay identical.
type progressBaseline struct {
	startedAt      time.Time
	lastActivityAt time.Time

	listenedMS int64
	durationMS int64
}

// resolveNextProgress turns a client-reported music
// progress payload into the accepted values for the locked
// session row. See resolveAcceptedProgress for the rules.
func resolveNextProgress(
	current models.PlaybackSession,
	positionMS int64,
	durationMS int64,
	listenedMS int64,
) nextProgress {
	return resolveAcceptedProgress(
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
}

// resolveAcceptedProgress turns a client-reported progress
// payload into the values the server is willing to accept,
// based on the locked current session row.
//
// Anti-manipulation rules applied here:
//
//  1. listened_ms never moves backwards (stale or
//     out-of-order progress calls are harmless).
//  2. reported listened_ms may never exceed the
//     wall-clock time elapsed since the session started,
//     and each increment may never exceed the wall-clock
//     time elapsed since the previous accepted progress
//     call — in both cases plus a small documented
//     tolerance. A client claiming 999999999 ms of
//     listening two seconds into a session is clamped
//     to ~7 seconds and therefore cannot qualify.
//  3. position_ms is clamped to the track duration.
//  4. a zero duration never erases a previously known
//     duration (frontend metadata can be briefly
//     unavailable).
//
// The wall-clock cap uses the DB-stamped started_at, so
// clock skew of the client is irrelevant. Since this runs
// inside the transaction while the row is locked, the
// accepted value used for qualification is exactly the
// value persisted — there is no window where qualification
// can be computed from a value the server rejected.
func resolveAcceptedProgress(
	baseline progressBaseline,
	positionMS int64,
	durationMS int64,
	listenedMS int64,
) nextProgress {
	// 1. Listened time is monotonic within a session.
	nextListenedMS := maxInt64(
		baseline.listenedMS,
		listenedMS,
	)

	// 2. Cap listened time against server wall-clock
	//    time. The DB clock stamped started_at and
	//    last_activity_at, so this is immune to client
	//    clock manipulation.
	//
	//    Two caps are applied:
	//
	//    a) The total may never exceed the wall-clock
	//       time elapsed since the session started
	//       (plus tolerance).
	//
	//    b) The INCREMENT may never exceed the wall-clock
	//       time elapsed since the previous accepted
	//       progress call (plus tolerance). Without this
	//       second cap, a client could keep a session
	//       open, wait, and then claim the entire wait as
	//       listening time in one request.
	sessionCapMS := time.
		Since(baseline.startedAt).
		Milliseconds() +
		models.PlaybackListenedToleranceMS

	incrementElapsedMS := time.
		Since(baseline.lastActivityAt).
		Milliseconds()

	// The frontend normally reports playback progress every
	// eight seconds. Never allow one request to claim an
	// arbitrarily large idle gap merely because the session
	// remained open.
	//
	// We still add PlaybackListenedToleranceMS so ordinary
	// browser scheduling delays, network latency, timer
	// throttling, and small reporting jitter do not discard
	// legitimate listening.
	const playbackProgressReportIntervalMS int64 = 8000

	incrementCapMS :=
		incrementElapsedMS +
			models.PlaybackListenedToleranceMS

	maxIncrementCapMS :=
		playbackProgressReportIntervalMS +
			models.PlaybackListenedToleranceMS

	if incrementCapMS > maxIncrementCapMS {
		incrementCapMS = maxIncrementCapMS
	}

	if incrementCapMS < 0 {
		incrementCapMS = 0
	}

	if nextListenedMS > sessionCapMS {
		nextListenedMS = sessionCapMS
	}

	incrementMaxMS :=
		baseline.listenedMS + incrementCapMS

	if nextListenedMS > incrementMaxMS {
		nextListenedMS = incrementMaxMS
	}

	if nextListenedMS < 0 {
		nextListenedMS = 0
	}

	// 3. Duration: never replace a known duration with an
	//    unknown one.
	nextDurationMS := durationMS

	if nextDurationMS == 0 &&
		baseline.durationMS > 0 {
		nextDurationMS =
			baseline.durationMS
	}

	// 4. Position must stay within the track.
	nextPositionMS := positionMS

	if nextDurationMS > 0 &&
		nextPositionMS > nextDurationMS {
		nextPositionMS =
			nextDurationMS
	}

	if nextPositionMS < 0 {
		nextPositionMS = 0
	}

	return nextProgress{
		durationMS: nextDurationMS,
		positionMS: nextPositionMS,
		listenedMS: nextListenedMS,
	}
}

func getPlaybackSessionForUpdate(
	ctx context.Context,
	tx pgx.Tx,
	sessionID string,
	userID int,
) (models.PlaybackSession, error) {
	query := `
		SELECT
			id::text,
			user_id,
			music_id,
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
		FROM playback_sessions
		WHERE id = $1::uuid
		  AND user_id = $2
		FOR UPDATE
	`

	session, err := scanPlaybackSession(
		tx.QueryRow(
			ctx,
			query,
			sessionID,
			userID,
		),
	)

	if errors.Is(err, pgx.ErrNoRows) {
		return models.PlaybackSession{},
			ErrPlaybackSessionNotFound
	}

	if err != nil {
		return models.PlaybackSession{}, err
	}

	return session, nil
}

func upsertListeningHistory(
	ctx context.Context,
	tx pgx.Tx,
	session models.PlaybackSession,
	justQualified bool,
) error {
	qualifiedIncrement := int64(0)

	if justQualified {
		qualifiedIncrement = 1
	}

	query := `
		INSERT INTO listening_history (
			user_id,
			music_id,
			last_position_ms,
			duration_ms,
			qualified_play_count,
			last_played_at,
			last_qualified_at
		)
		VALUES (
			$1,
			$2,
			$3,
			$4,
			$5,
			now(),
			CASE
				WHEN $5::bigint > 0
				THEN now()
				ELSE NULL
			END
		)

		ON CONFLICT (
			user_id,
			music_id
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
							ELSE listening_history.duration_ms
						END
					) > 0
					THEN LEAST(
						EXCLUDED.last_position_ms,
						CASE
							WHEN EXCLUDED.duration_ms > 0
							THEN EXCLUDED.duration_ms
							ELSE listening_history.duration_ms
						END
					)
					ELSE EXCLUDED.last_position_ms
				END,

			duration_ms =
				CASE
					WHEN EXCLUDED.duration_ms > 0
					THEN EXCLUDED.duration_ms
					ELSE listening_history.duration_ms
				END,

			qualified_play_count =
				listening_history.qualified_play_count
				+ EXCLUDED.qualified_play_count,

			last_played_at = now(),

			last_qualified_at =
				CASE
					WHEN EXCLUDED.qualified_play_count > 0
					THEN now()
					ELSE listening_history.last_qualified_at
				END,

			updated_at = now()
	`

	_, err := tx.Exec(
		ctx,
		query,
		session.UserID,
		session.MusicID,
		session.PositionMS,
		session.DurationMS,
		qualifiedIncrement,
	)

	return err
}

// -----------------------------------------------------------------
// Scanning helpers
// -----------------------------------------------------------------

type playbackSessionRow interface {
	Scan(dest ...any) error
}

func scanPlaybackSession(
	row playbackSessionRow,
) (models.PlaybackSession, error) {
	var session models.PlaybackSession

	err := row.Scan(
		&session.ID,
		&session.UserID,
		&session.MusicID,
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
		return models.PlaybackSession{}, err
	}

	return session, nil
}

func maxInt64(
	a int64,
	b int64,
) int64 {
	if a > b {
		return a
	}

	return b
}
