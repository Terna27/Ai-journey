export type Music = {
  id: number
  artist_id: number | null
  artist_name: string
  song_title: string
  genre: string
  image_url: string
  image_public_id: string
  audio_url: string
  audio_public_id: string
  audio_key: string
  likes: number
  loves: number
  rating: number
  date_posted: string
}

export type UploadMusicInput = {
  artistName: string
  songTitle: string
  genre: string
  image: File
  audio: File
}
