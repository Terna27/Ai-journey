import type {
  Podcast,
  PodcastEpisode,
} from './podcast'

export type PodcastLiveSessionStatus =
  | 'SCHEDULED'
  | 'LIVE'
  | 'ENDED'
  | 'CANCELLED'

export type PodcastLiveRecordingStatus =
  | 'NONE'
  | 'STARTING'
  | 'RECORDING'
  | 'PROCESSING'
  | 'READY'
  | 'FAILED'

// Owner-facing live session. Provider room identifiers are
// intentionally absent: the backend never exposes them.
export type PodcastLiveSession = {
  id: string
  episode_id: number
  status: PodcastLiveSessionStatus
  scheduled_start_at: string
  started_at: string | null
  ended_at: string | null
  recording_status: PodcastLiveRecordingStatus
  recording_url: string | null
  created_at: string
  updated_at: string
}

// Public (listener-facing) live session view.
export type PodcastLiveSessionPublic = {
  id: string
  episode_id: number
  status: PodcastLiveSessionStatus
  scheduled_start_at: string
  started_at: string | null
  ended_at: string | null
  recording_status: PodcastLiveRecordingStatus
}

export type PodcastLiveDetails = {
  live_session: PodcastLiveSession
  episode: PodcastEpisode
  podcast: Podcast
}

export type PodcastLiveBroadcast = {
  live_session: PodcastLiveSessionPublic
  episode: PodcastEpisode
  podcast: Podcast
}

// Short-lived participant credential minted by the backend.
// Contains only the signed token and the public connection
// URL — never provider secrets.
export type LiveAccessToken = {
  token: string
  connect_url: string
  expires_at: string
}
