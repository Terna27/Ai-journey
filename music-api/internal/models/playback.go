package models

import (
	"math"
	"time"
)

// QualifiedPlayMaxMS is the absolute ceiling for the
// qualified-play listening threshold: 30 seconds.
const QualifiedPlayMaxMS int64 = 30_000

// PlaybackListenedToleranceMS is the grace period added
// to the server-side wall-clock cap on listened_ms.
//
// listened_ms reported by a client may never exceed the
// wall-clock time elapsed since the session started, plus
// this tolerance. The tolerance exists because:
//
//   - the progress call is in flight while playback
//     continues (network latency),
//   - the application server clock and the database
//     clock (which stamps started_at) are different
//     hosts and may skew slightly.
//
// Tradeoff: a manipulated client can inflate listened_ms
// by at most this tolerance per progress call, but the
// wall-clock cap still forces ~real time to pass before
// a play can qualify, so farming qualified plays still
// costs roughly real listening time.
const PlaybackListenedToleranceMS int64 = 5_000

// QualifiedPlayThresholdMS returns how much genuine
// listening time qualifies a play for the given track
// duration.
//
// The rule is: 30 seconds OR 50% of the track duration,
// whichever comes first. Unknown duration (0) requires
// the full 30 seconds.
func QualifiedPlayThresholdMS(
	durationMS int64,
) int64 {
	if durationMS <= 0 {
		return QualifiedPlayMaxMS
	}

	halfDurationMS := int64(
		math.Ceil(
			float64(durationMS) / 2,
		),
	)

	if halfDurationMS < QualifiedPlayMaxMS {
		return halfDurationMS
	}

	return QualifiedPlayMaxMS
}

// IsQualifiedPlay reports whether the accepted (already
// server-side clamped) listening time meets the
// qualification threshold for the given duration.
//
// Callers must only pass listenedMS values that the
// server has accepted, never raw client input.
func IsQualifiedPlay(
	listenedMS int64,
	durationMS int64,
) bool {
	if listenedMS < 0 {
		return false
	}

	return listenedMS >=
		QualifiedPlayThresholdMS(
			durationMS,
		)
}

// PlaybackSession represents one continuous listening
// session for one authenticated user and one music track.
//
// PositionMS and ListenedMS intentionally represent
// different concepts:
//
//   - PositionMS: where the playhead currently is.
//   - ListenedMS: how much genuine playback time has
//     accumulated.
//
// Keeping them separate prevents seeking forward from
// being interpreted as actual listening.
type PlaybackSession struct {
	ID string `json:"id"`

	UserID  int `json:"user_id"`
	MusicID int `json:"music_id"`

	DurationMS int64 `json:"duration_ms"`
	PositionMS int64 `json:"position_ms"`
	ListenedMS int64 `json:"listened_ms"`

	Qualified   bool       `json:"qualified"`
	QualifiedAt *time.Time `json:"qualified_at"`

	Completed   bool       `json:"completed"`
	CompletedAt *time.Time `json:"completed_at"`

	StartedAt      time.Time `json:"started_at"`
	LastActivityAt time.Time `json:"last_activity_at"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// ListeningHistory represents the latest known listening
// state for one user and one music track.
//
// There is one row per (user_id, music_id).
type ListeningHistory struct {
	UserID  int `json:"user_id"`
	MusicID int `json:"music_id"`

	LastPositionMS int64 `json:"last_position_ms"`
	DurationMS     int64 `json:"duration_ms"`

	QualifiedPlayCount int64 `json:"qualified_play_count"`

	LastPlayedAt    time.Time  `json:"last_played_at"`
	LastQualifiedAt *time.Time `json:"last_qualified_at"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// ListeningHistoryItem is the public/user-facing history
// representation.
//
// It contains the track together with the user's latest
// playback information so the frontend does not need to
// make another request for every recently played song.
type ListeningHistoryItem struct {
	Music Music `json:"music"`

	LastPositionMS int64 `json:"last_position_ms"`
	DurationMS     int64 `json:"duration_ms"`

	QualifiedPlayCount int64 `json:"qualified_play_count"`

	LastPlayedAt    time.Time  `json:"last_played_at"`
	LastQualifiedAt *time.Time `json:"last_qualified_at"`
}
