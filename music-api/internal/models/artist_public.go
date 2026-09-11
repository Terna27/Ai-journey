package models

import "time"

// PublicArtist contains only information that is safe to expose
// on a public artist profile.
type PublicArtist struct {
	ID   int    `json:"id"`
	Name string `json:"name"`

	Bio *string `json:"bio,omitempty"`

	ProfileImageURL *string `json:"profile_image_url,omitempty"`

	HeroVideoURL *string `json:"hero_video_url,omitempty"`

	HeroVideoPosterURL *string `json:"hero_video_poster_url,omitempty"`

	CreatedAt time.Time `json:"created_at"`
}

// ArtistProfile is the public representation of an artist page.
type ArtistProfile struct {
	Artist PublicArtist `json:"artist"`
	Tracks []Music      `json:"tracks"`
}
