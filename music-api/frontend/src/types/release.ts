import type { Music } from './music'

export type ReleaseType =
  | 'SINGLE'
  | 'EP'
  | 'ALBUM'

export type Release = {
  id: number
  artist_id: number
  title: string
  release_type: ReleaseType
  cover_image_url: string
  cover_image_public_id: string
  description: string
  release_date: string | null
  is_published: boolean
  created_at: string
  updated_at: string
}

export type ReleaseDetails = {
  release: Release
  tracks: Music[]
}

export type CreateReleaseInput = {
  title: string
  release_type: ReleaseType
  cover_image_url: string
  cover_image_public_id: string
  description: string
  release_date: string
  is_published: boolean
}

export type UpdateReleaseInput =
  CreateReleaseInput

export type AddReleaseTrackInput = {
  track_number: number
}