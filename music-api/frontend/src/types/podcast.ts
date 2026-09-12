export type PodcastStatus =
  | 'DRAFT'
  | 'PUBLISHED'
  | 'ARCHIVED'

// Live statuses (SCHEDULED, LIVE, ENDED) are set only through
// the live session endpoints — never through episode PATCH.
export type PodcastEpisodeStatus =
  | 'DRAFT'
  | 'PUBLISHED'
  | 'ARCHIVED'
  | 'SCHEDULED'
  | 'LIVE'
  | 'ENDED'

export type PodcastEpisodeType =
  | 'FULL'
  | 'TRAILER'
  | 'BONUS'

export type Podcast = {
  id: number
  owner_user_id: number
  title: string
  slug: string
  description: string
  category: string
  artwork_url: string
  status: PodcastStatus
  is_explicit: boolean
  published_at: string | null
  created_at: string
  updated_at: string
}

export type PodcastEpisode = {
  id: number
  podcast_id: number
  title: string
  slug: string
  description: string
  season_number: number
  episode_number: number | null
  episode_type: PodcastEpisodeType
  audio_url: string
  artwork_url: string
  duration_ms: number
  status: PodcastEpisodeStatus
  is_explicit: boolean
  scheduled_at: string | null
  published_at: string | null
  created_at: string
  updated_at: string
}

export type PodcastDetails = {
  podcast: Podcast
  episodes: PodcastEpisode[]
}

export type CreatePodcastInput = {
  title: string
  description?: string
  category?: string
  is_explicit?: boolean
}

export type UpdatePodcastInput = {
  title?: string
  description?: string
  category?: string
  status?: PodcastStatus
  is_explicit?: boolean
}

export type CreatePodcastEpisodeInput = {
  title: string
  description?: string
  season_number?: number
  episode_number?: number
  episode_type?: PodcastEpisodeType
  duration_ms?: number
  status?: PodcastEpisodeStatus
  is_explicit?: boolean
}

export type UpdatePodcastEpisodeInput = {
  title?: string
  description?: string
  season_number?: number
  episode_number?: number
  episode_type?: PodcastEpisodeType
  duration_ms?: number
  status?: PodcastEpisodeStatus
  is_explicit?: boolean
}

export type UpdatePodcastArtworkInput = {
  artwork: File
}

export type UpdatePodcastEpisodeMediaInput = {
  audio?: File
  artwork?: File
}
