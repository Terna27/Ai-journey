package handler

type CreateNoteRequest struct {
	ArtistName string `json:"artist_name"`
	SongTitle  string `json:"song_title"`
	Genre      string `json:"genre"`
	ImageURL   string `json:"image_url"`
}



type UpdateNoteRequest struct {
	ArtistName *string `json:"artist_name"`
	SongTitle  *string `json:"song_title"`
	Genre      *string `json:"genre"`
	ImageURL   *string `json:"image_url"`
}
