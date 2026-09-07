import type { Music } from './music'

export type Playlist = {
  id: number
  user_id: number
  name: string
  description: string
  is_public: boolean
  created_at: string
  updated_at: string
}

export type PlaylistDetails = {
  playlist: Playlist
  tracks: Music[]
}

export type CreatePlaylistInput = {
  name: string
  description: string
  is_public: boolean
}

export type UpdatePlaylistInput = {
  name: string
  description: string
  is_public: boolean
}
