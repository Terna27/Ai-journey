package models

import "time"

// LyricLine represents one synchronized lyric line.
//
// TimeMS is measured from the beginning of the
// audio track in milliseconds.
//
// EndMS optionally records an explicit timing range
// for lines merged in the lyrics editor. Zero means
// the line is open-ended and runs until the next line
// begins, which is the only behaviour older lyrics
// data ever had, so existing rows stay valid.
type LyricLine struct {
	TimeMS int64  `json:"time_ms"`
	EndMS  int64  `json:"end_ms,omitempty"`
	Text   string `json:"text"`
}

// TrackLyrics contains the lyrics belonging to one
// music track.
//
// Lyrics are intentionally separate from Music so
// discovery, search, playlists and other ordinary
// track requests do not download large lyric bodies.
type TrackLyrics struct {
	MusicID int `json:"music_id"`

	PlainLyrics string `json:"plain_lyrics"`

	SyncedLines []LyricLine `json:"synced_lines"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
