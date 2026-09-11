package models

// DiscoveryGenre represents a genre available for browsing.
//
// TrackCount allows the frontend to show how much content exists
// inside each genre without making another request.
type DiscoveryGenre struct {
	Name       string `json:"name"`
	TrackCount int    `json:"track_count"`
}

// DiscoveryHeroArtist is the safe public artist representation
// used by the rotating homepage hero.
//
// Only public presentation data is exposed here. Cloudinary public
// IDs, account IDs, email addresses, and other private data must
// never be returned by this endpoint.
type DiscoveryHeroArtist struct {
	ID                 int    `json:"id"`
	Name               string `json:"name"`
	Bio                string `json:"bio,omitempty"`
	ProfileImageURL    string `json:"profile_image_url,omitempty"`
	HeroVideoURL       string `json:"hero_video_url"`
	HeroVideoPosterURL string `json:"hero_video_poster_url,omitempty"`
	EngagementScore    int64  `json:"engagement_score"`
	TrackCount         int    `json:"track_count"`
}

// DiscoveryHeroResult is returned by
// GET /api/v1/discovery/hero-artists.
//
// The backend returns a ranked candidate pool. The frontend may
// rotate and randomly select from this pool rather than always
// showing the highest-ranked artist.
type DiscoveryHeroResult struct {
	Artists []DiscoveryHeroArtist `json:"artists"`
}

// DiscoveryResult is returned by GET /api/v1/discover.
//
// These sections are intentionally generated from database data.
// Nothing here is hardcoded by the frontend.
type DiscoveryResult struct {
	TrendingTracks []Music `json:"trending_tracks"`

	NewTracks []Music `json:"new_tracks"`

	NewReleases []SearchRelease `json:"new_releases"`

	PopularArtists []SearchArtist `json:"popular_artists"`

	Genres []DiscoveryGenre `json:"genres"`
}
