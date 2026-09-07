package models

import "time"

// PublicArtist contains only information that is safe to expose
// on a public artist profile.
type PublicArtist struct {
	ID        int       `json:"id"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"created_at"`
}

// ArtistProfile is the public representation of an artist page.
type ArtistProfile struct {
	Artist PublicArtist `json:"artist"`
	Tracks []Music      `json:"tracks"`
}
