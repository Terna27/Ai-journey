package models

import "time"

// ReleaseType identifies the kind of music release.
type ReleaseType string

const (
	ReleaseTypeSingle ReleaseType = "SINGLE"
	ReleaseTypeEP     ReleaseType = "EP"
	ReleaseTypeAlbum  ReleaseType = "ALBUM"
)

// Release represents a collection of music published by an artist.
//
// A SINGLE, EP, or ALBUM is represented by the same domain model.
// Individual tracks remain stored in the music table.
type Release struct {
	ID int `json:"id"`

	ArtistID int `json:"artist_id"`

	Title string `json:"title"`

	ReleaseType ReleaseType `json:"release_type"`

	CoverImageURL string `json:"cover_image_url"`

	CoverImagePublicID string `json:"cover_image_public_id"`

	Description string `json:"description"`

	ReleaseDate *time.Time `json:"release_date"`

	IsPublished bool `json:"is_published"`

	CreatedAt time.Time `json:"created_at"`

	UpdatedAt time.Time `json:"updated_at"`
}

// ReleaseDetails is the public representation of a release together
// with its ordered tracks.
type ReleaseDetails struct {
	Release Release `json:"release"`

	Tracks []Music `json:"tracks"`
}
