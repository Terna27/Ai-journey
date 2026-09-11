export type LyricLine = {
  time_ms: number

  /*
   * Explicit timing range for merged lyric lines.
   * Absent/zero means the line is open-ended and
   * runs until the next line begins (the behaviour
   * of all lyrics saved before ranges existed).
   */
  end_ms?: number

  text: string
}

export type TrackLyrics = {
  music_id: number
  plain_lyrics: string
  synced_lines: LyricLine[]
  created_at: string
  updated_at: string
}
