import type { Music } from './music'

/*
 * Playback session tracking (Phase 3.8).
 *
 * A session is created when a logged-in verified listener
 * starts a real music track, receives periodic progress
 * while audio genuinely plays, and is completed when the
 * track ends, the user switches tracks, playback stops,
 * or the app unloads.
 */

export type PlaybackSession = {
  id: string

  user_id: number
  music_id: number

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

export type CreatePlaybackSessionInput = {
  music_id: number
  duration_ms: number
  position_ms: number
}

export type UpdatePlaybackProgressInput = {
  duration_ms: number
  position_ms: number
  listened_ms: number
}

export type ListeningHistoryItem = {
  music: Music

  last_position_ms: number
  duration_ms: number

  qualified_play_count: number

  last_played_at: string
  last_qualified_at: string | null
}
