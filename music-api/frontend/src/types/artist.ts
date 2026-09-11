import type { Music } from './music'

export type PublicArtist = {
  id: number
  name: string

  bio?: string
  profile_image_url?: string
  hero_video_url?: string
  hero_video_poster_url?: string

  created_at: string
}

export type ArtistProfileResponse = {
  artist: PublicArtist
  tracks: Music[]
}