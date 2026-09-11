package models

import "time"

// SearchArtist is the safe public artist representation returned by search.
//
// Email, password hash, user ID, and other private account information
// must never be exposed through public search.
type SearchArtist struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

// SearchRelease is the lightweight public release representation
// returned by the unified search endpoint.
type SearchRelease struct {
	ID            int         `json:"id"`
	ArtistID      int         `json:"artist_id"`
	ArtistName    string      `json:"artist_name"`
	Title         string      `json:"title"`
	ReleaseType   ReleaseType `json:"release_type"`
	CoverImageURL string      `json:"cover_image_url"`
	Description   string      `json:"description"`
	ReleaseDate   *time.Time  `json:"release_date"`
}

// SearchPagination describes pagination independently for each
// result category.
//
// The search endpoint uses one shared page and limit, but tracks,
// artists, and releases have independent result sets and totals.
type SearchPagination struct {
	Page int `json:"page"`

	Limit int `json:"limit"`

	TrackTotal int `json:"track_total"`

	ArtistTotal int `json:"artist_total"`

	ReleaseTotal int `json:"release_total"`

	TrackHasMore bool `json:"track_has_more"`

	ArtistHasMore bool `json:"artist_has_more"`

	ReleaseHasMore bool `json:"release_has_more"`
}

// SearchResult is the response returned by GET /api/v1/search.
type SearchResult struct {
	Query string `json:"query"`

	Type string `json:"type"`

	Sort string `json:"sort"`

	Genre string `json:"genre,omitempty"`

	Tracks []Music `json:"tracks"`

	Artists []SearchArtist `json:"artists"`

	Releases []SearchRelease `json:"releases"`

	Pagination SearchPagination `json:"pagination"`
}
