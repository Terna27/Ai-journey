package models

import "time"

// PodcastCompletionRemainingMS is the maximum remaining
// time for an episode to count as completed even if the
// media element never fires its `ended` event.
//
// Combined with the 95% position threshold below, this is
// the single server-consistent completion rule: an episode
// is complete when playback reaches the natural end, or
// the remaining time is at most this many milliseconds, or
// the accepted position reaches 95% of the known duration.
//
// Completion is deliberately independent of qualification:
// seeking straight to the end can complete an episode but
// never manufactures genuine listened time, so a qualified
// play still requires real listening.
const PodcastCompletionRemainingMS int64 = 15_000

// podcastCompletionPercentDenominator keeps the 95%
// completion check in integer arithmetic.
const (
	podcastCompletionPercentNumerator   int64 = 95
	podcastCompletionPercentDenominator int64 = 100
)

// IsPodcastPlaybackNearlyComplete reports whether the
// accepted playback position is close enough to the end of
// a known-duration episode to treat it as completed.
//
// Unknown duration (0) can never be nearly complete; the
// client's `ended` event remains the only completion
// signal in that case.
func IsPodcastPlaybackNearlyComplete(
	positionMS int64,
	durationMS int64,
) bool {
	if durationMS <= 0 || positionMS <= 0 {
		return false
	}

	if positionMS > durationMS {
		return true
	}

	remainingMS := durationMS - positionMS

	if remainingMS <= PodcastCompletionRemainingMS {
		return true
	}

	return positionMS*
		podcastCompletionPercentDenominator >=
		durationMS*
			podcastCompletionPercentNumerator
}

// PodcastPlaybackSession represents one continuous
// listening session for one authenticated user and one
// podcast episode.
//
// PositionMS and ListenedMS intentionally represent
// different concepts, exactly like music playback:
//
//   - PositionMS: where the playhead currently is.
//   - ListenedMS: how much genuine playback time has
//     accumulated.
//
// A completed episode is replayed by creating a NEW
// session; session ids are never reused across listening
// lifecycles.
type PodcastPlaybackSession struct {
	ID string `json:"id"`

	UserID    int   `json:"user_id"`
	EpisodeID int64 `json:"episode_id"`

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

// PodcastListeningHistory represents the latest known
// listening state for one user and one podcast episode.
//
// There is one row per (user_id, episode_id). When a
// completed episode is genuinely replayed, the row
// reactivates: completed becomes FALSE and completed_at is
// cleared by the new lifecycle's first accepted progress.
type PodcastListeningHistory struct {
	UserID    int   `json:"user_id"`
	EpisodeID int64 `json:"episode_id"`

	LastPositionMS int64 `json:"last_position_ms"`
	DurationMS     int64 `json:"duration_ms"`

	QualifiedPlayCount int64 `json:"qualified_play_count"`

	Completed   bool       `json:"completed"`
	CompletedAt *time.Time `json:"completed_at"`

	LastPlayedAt    time.Time  `json:"last_played_at"`
	LastQualifiedAt *time.Time `json:"last_qualified_at"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// PodcastListeningHistoryItem is the public/user-facing
// podcast history representation.
//
// It embeds the episode together with its podcast so the
// frontend can render a card (and resume playback) without
// any additional requests.
type PodcastListeningHistoryItem struct {
	Podcast Podcast        `json:"podcast"`
	Episode PodcastEpisode `json:"episode"`

	LastPositionMS int64 `json:"last_position_ms"`
	DurationMS     int64 `json:"duration_ms"`

	QualifiedPlayCount int64 `json:"qualified_play_count"`

	Completed   bool       `json:"completed"`
	CompletedAt *time.Time `json:"completed_at"`

	LastPlayedAt    time.Time  `json:"last_played_at"`
	LastQualifiedAt *time.Time `json:"last_qualified_at"`
}

// PodcastContinueListeningItem has the same payload shape
// as a history item; Continue Listening is simply the
// subset of history rows that are incomplete and have
// meaningful progress.
type PodcastContinueListeningItem = PodcastListeningHistoryItem
