import type { Music } from './music'
import type {
  ReleaseType,
} from './release'

export type DiscoveryGenre = {
  name: string
  track_count: number
}

export type DiscoveryArtist = {
  id: number
  name: string
}

export type DiscoveryHeroArtist = {
  id: number
  name: string
  bio?: string
  profile_image_url?: string
  hero_video_url: string
  hero_video_poster_url?: string
  engagement_score: number
  track_count: number
}

export type DiscoveryHeroResponse = {
  artists: DiscoveryHeroArtist[]
}

export type DiscoveryRelease = {
  id: number
  artist_id: number
  artist_name: string
  title: string
  release_type: ReleaseType
  cover_image_url: string
  description: string
  release_date: string | null
}

export type DiscoveryResponse = {
  trending_tracks: Music[]
  new_tracks: Music[]
  new_releases: DiscoveryRelease[]
  popular_artists: DiscoveryArtist[]
  genres: DiscoveryGenre[]
}