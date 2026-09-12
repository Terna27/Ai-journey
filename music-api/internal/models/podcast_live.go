package models

import "time"

// PodcastLiveSessionStatus represents the lifecycle state of
// one live broadcast runtime.
//
// SCHEDULED -> LIVE -> ENDED is the happy path. CANCELLED is
// only reachable from SCHEDULED, before the broadcast starts.
type PodcastLiveSessionStatus string

const (
	PodcastLiveStatusScheduled PodcastLiveSessionStatus = "SCHEDULED"
	PodcastLiveStatusLive      PodcastLiveSessionStatus = "LIVE"
	PodcastLiveStatusEnded     PodcastLiveSessionStatus = "ENDED"
	PodcastLiveStatusCancelled PodcastLiveSessionStatus = "CANCELLED"
)

// PodcastLiveRecordingStatus tracks the lifecycle of a live
// recording owned by the provider (LiveKit egress).
type PodcastLiveRecordingStatus string

const (
	PodcastLiveRecordingNone       PodcastLiveRecordingStatus = "NONE"
	PodcastLiveRecordingStarting   PodcastLiveRecordingStatus = "STARTING"
	PodcastLiveRecordingRecording  PodcastLiveRecordingStatus = "RECORDING"
	PodcastLiveRecordingProcessing PodcastLiveRecordingStatus = "PROCESSING"
	PodcastLiveRecordingReady      PodcastLiveRecordingStatus = "READY"
	PodcastLiveRecordingFailed     PodcastLiveRecordingStatus = "FAILED"
)

// PodcastLiveSession is the runtime record of one live audio
// broadcast for one podcast episode.
//
// The episode remains the editorial/content record; this
// entity owns the live runtime (schedule, provider room,
// recording state). Live audio itself never flows through
// the Go HTTP server.
//
// Provider room identifiers and recording asset IDs are
// internal server-side details: they are hidden from every
// JSON response. Clients join rooms exclusively with the
// short-lived participant tokens minted by the backend.
type PodcastLiveSession struct {
	ID string `json:"id"`

	EpisodeID int64 `json:"episode_id"`

	HostUserID int `json:"host_user_id"`

	Provider         string  `json:"-"`
	ProviderRoomName string  `json:"-"`
	ProviderRoomSID  *string `json:"-"`

	Status           PodcastLiveSessionStatus `json:"status"`
	ScheduledStartAt time.Time                `json:"scheduled_start_at"`

	StartedAt *time.Time `json:"started_at"`
	EndedAt   *time.Time `json:"ended_at"`

	RecordingStatus          PodcastLiveRecordingStatus `json:"recording_status"`
	RecordingProviderAssetID *string                    `json:"-"`
	RecordingURL             *string                    `json:"recording_url"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// PodcastLiveSessionPublic is the public projection of a
// live session: schedule and state only. Provider room
// identifiers, recording asset IDs, and host details stay
// server-side.
type PodcastLiveSessionPublic struct {
	ID string `json:"id"`

	EpisodeID int64 `json:"episode_id"`

	Status           PodcastLiveSessionStatus `json:"status"`
	ScheduledStartAt time.Time                `json:"scheduled_start_at"`

	StartedAt *time.Time `json:"started_at"`
	EndedAt   *time.Time `json:"ended_at"`

	RecordingStatus PodcastLiveRecordingStatus `json:"recording_status"`
}

// Public returns the public projection of the session.
func (s PodcastLiveSession) Public() PodcastLiveSessionPublic {
	return PodcastLiveSessionPublic{
		ID:               s.ID,
		EpisodeID:        s.EpisodeID,
		Status:           s.Status,
		ScheduledStartAt: s.ScheduledStartAt,
		StartedAt:        s.StartedAt,
		EndedAt:          s.EndedAt,
		RecordingStatus:  s.RecordingStatus,
	}
}

// PodcastLiveBroadcast is the publicly visible joined payload
// for one live broadcast: the live session state together with
// its episode and podcast, so discovery UIs need no extra
// requests.
type PodcastLiveBroadcast struct {
	LiveSession PodcastLiveSessionPublic `json:"live_session"`

	Episode PodcastEpisode `json:"episode"`
	Podcast Podcast        `json:"podcast"`
}

// PodcastLiveDetails is the owner-facing view of a live
// session: the full runtime record together with its episode
// and podcast.
type PodcastLiveDetails struct {
	LiveSession PodcastLiveSession `json:"live_session"`

	Episode PodcastEpisode `json:"episode"`
	Podcast Podcast        `json:"podcast"`
}
