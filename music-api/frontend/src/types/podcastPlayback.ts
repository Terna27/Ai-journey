import type {
  Podcast,
  PodcastEpisode,
} from './podcast'

/*
 * Podcast playback tracking (Phase 4.1).
 *
 * Mirrors the music playback session model but with
 * dedicated podcast endpoints and table storage. A
 * session represents one continuous podcast listening
 * period for one authenticated user; history persists
 * across sessions.
 */

export type PodcastPlaybackSession = {
  id: string

  user_id: number
  episode_id: number

  duration_ms: number
  position_ms: number
  listened_ms: number

  qualified: boolean
  qualified_at: string | null

  completed: boolean
  completed_at: string | null

  started_at: string
  last_activity_at: string

  created_at: string
  updated_at: string
}

export type CreatePodcastPlaybackSessionInput = {
  episode_id: number
  duration_ms: number
  position_ms: number
}

export type UpdatePodcastPlaybackProgressInput = {
  duration_ms: number
  position_ms: number
  listened_ms: number
}

/*
 * The history payload embeds the episode and its podcast
 * so cards (and the Resume button) render without any
 * additional requests.
 */
export type PodcastListeningHistoryItem = {
  podcast: Podcast
  episode: PodcastEpisode

  last_position_ms: number
  duration_ms: number

  qualified_play_count: number

  completed: boolean
  completed_at: string | null

  last_played_at: string
  last_qualified_at: string | null
}

export type PodcastContinueListeningItem =
  PodcastListeningHistoryItem
