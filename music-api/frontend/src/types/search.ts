import type { Music } from './music'
import type { ReleaseType } from './release'

export type SearchType =
  | 'all'
  | 'track'
  | 'artist'
  | 'release'

export type SearchSort =
  | 'relevance'
  | 'newest'
  | 'popular'

export type SearchArtist = {
  id: number
  name: string
}

export type SearchRelease = {
  id: number
  artist_id: number
  artist_name: string
  title: string
  release_type: ReleaseType
  cover_image_url: string
  description: string
  release_date: string | null
}

export type SearchPagination = {
  page: number
  limit: number
  track_total: number
  artist_total: number
  release_total: number
  track_has_more: boolean
  artist_has_more: boolean
  release_has_more: boolean
}

export type SearchResponse = {
  query: string
  type: SearchType
  sort: SearchSort
  genre?: string
  tracks: Music[]
  artists: SearchArtist[]
  releases: SearchRelease[]
  pagination: SearchPagination
}

export type SearchOptions = {
  query: string
  type?: SearchType
  sort?: SearchSort
  genre?: string
  page?: number
  limit?: number
}