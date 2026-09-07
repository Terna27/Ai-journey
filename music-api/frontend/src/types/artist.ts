import type { Music } from './music'

export type PublicArtist = {
  id: number
  name: string
  created_at: string
}

export type ArtistProfileResponse = {
  artist: PublicArtist
  tracks: Music[]
}