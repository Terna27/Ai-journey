package models

import "time"

type Music struct {
	ID            int       `json:"id"`
	ArtistID      *int      `json:"artist_id"`
	ArtistName    string    `json:"artist_name"`
	SongTitle     string    `json:"song_title"`
	Genre         string    `json:"genre"`
	ImageURL      string    `json:"image_url"`
	ImagePublicID string    `json:"image_public_id"`
	AudioURL      string    `json:"audio_url"`
	AudioPublicID string    `json:"audio_public_id"`
	AudioKey      string    `json:"audio_key"`
	Likes         int       `json:"likes"`
	Loves         int       `json:"loves"`
	Rating        float64   `json:"rating"`
	DatePosted    time.Time `json:"date_posted"`
}

type Artist struct {
	ID           int       `json:"id"`
	UserID       int       `json:"user_id,omitempty"`
	Name         string    `json:"name"`
	Email        string    `json:"email"`
	PasswordHash string    `json:"-"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type User struct {
	ID              int        `json:"id"`
	Name            string     `json:"name"`
	Email           string     `json:"email"`
	PasswordHash    string     `json:"-"`
	EmailVerified   bool       `json:"email_verified"`
	EmailVerifiedAt *time.Time `json:"email_verified_at"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
}

type EmailVerificationToken struct {
	ID        int64      `json:"id"`
	UserID    int        `json:"user_id"`
	TokenHash string     `json:"-"`
	ExpiresAt time.Time  `json:"expires_at"`
	UsedAt    *time.Time `json:"used_at"`
	CreatedAt time.Time  `json:"created_at"`
}

type Playlist struct {
	ID          int64     `json:"id"`
	UserID      int       `json:"user_id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	IsPublic    bool      `json:"is_public"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type PlaylistDetails struct {
	Playlist Playlist `json:"playlist"`
	Tracks   []Music  `json:"tracks"`
}
