package models

import "time"

// PodcastStatus represents the lifecycle state of a podcast.
type PodcastStatus string

const (
	PodcastStatusDraft     PodcastStatus = "DRAFT"
	PodcastStatusPublished PodcastStatus = "PUBLISHED"
	PodcastStatusArchived  PodcastStatus = "ARCHIVED"
)

// PodcastEpisodeStatus represents the lifecycle state of a
// podcast episode.
//
// SCHEDULED, LIVE, and ENDED are included now so the model
// remains compatible with future scheduled/live podcast work.
type PodcastEpisodeStatus string

const (
	PodcastEpisodeStatusDraft     PodcastEpisodeStatus = "DRAFT"
	PodcastEpisodeStatusScheduled PodcastEpisodeStatus = "SCHEDULED"
	PodcastEpisodeStatusPublished PodcastEpisodeStatus = "PUBLISHED"
	PodcastEpisodeStatusLive      PodcastEpisodeStatus = "LIVE"
	PodcastEpisodeStatusEnded     PodcastEpisodeStatus = "ENDED"
	PodcastEpisodeStatusArchived  PodcastEpisodeStatus = "ARCHIVED"
)

// PodcastEpisodeType describes the editorial role of an episode.
type PodcastEpisodeType string

const (
	PodcastEpisodeTypeFull    PodcastEpisodeType = "FULL"
	PodcastEpisodeTypeTrailer PodcastEpisodeType = "TRAILER"
	PodcastEpisodeTypeBonus   PodcastEpisodeType = "BONUS"
)

// Podcast represents one podcast show/series.
//
// Ownership belongs directly to a user account rather than an
// artist profile.
type Podcast struct {
	ID int64 `json:"id"`

	OwnerUserID int `json:"owner_user_id"`

	Title       string `json:"title"`
	Slug        string `json:"slug"`
	Description string `json:"description"`
	Category    string `json:"category"`

	ArtworkURL      string `json:"artwork_url"`
	ArtworkPublicID string `json:"-"`

	Status PodcastStatus `json:"status"`

	IsExplicit bool `json:"is_explicit"`

	PublishedAt *time.Time `json:"published_at"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// PodcastEpisode represents one episode belonging to a podcast.
type PodcastEpisode struct {
	ID int64 `json:"id"`

	PodcastID int64 `json:"podcast_id"`

	Title       string `json:"title"`
	Slug        string `json:"slug"`
	Description string `json:"description"`

	SeasonNumber  int  `json:"season_number"`
	EpisodeNumber *int `json:"episode_number"`

	EpisodeType PodcastEpisodeType `json:"episode_type"`

	AudioURL      string `json:"audio_url"`
	AudioPublicID string `json:"-"`

	ArtworkURL      string `json:"artwork_url"`
	ArtworkPublicID string `json:"-"`

	DurationMS int64 `json:"duration_ms"`

	Status PodcastEpisodeStatus `json:"status"`

	IsExplicit bool `json:"is_explicit"`

	ScheduledAt *time.Time `json:"scheduled_at"`
	PublishedAt *time.Time `json:"published_at"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// PodcastDetails is the public representation of a podcast
// together with its episodes.
//
// Public API handlers will decide whether the episode collection
// contains only published episodes or owner-visible drafts.
type PodcastDetails struct {
	Podcast Podcast `json:"podcast"`

	Episodes []PodcastEpisode `json:"episodes"`
}
