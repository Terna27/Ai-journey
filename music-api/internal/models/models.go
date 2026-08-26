package models

import "time"

type Music struct {
	ID            int       `json:"id"`
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
	Name         string    `json:"name"`
	Email        string    `json:"email"`
	PasswordHash string    `json:"-"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}