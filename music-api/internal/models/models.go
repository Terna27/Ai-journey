package models

import "time"

type Music struct {
	ID         int       `json:"id"`
	ArtistName string    `json:"artist_name"`
	SongTitle  string    `json:"song_title"`
	Genre      string    `json:"genre"`
	ImageURL   string    `json:"image_url"`
	Likes      int       `json:"likes"`
	Loves      int       `json:"loves"`
	Rating     float64   `json:"rating"`
	DatePosted time.Time `json:"date_posted"`
}
