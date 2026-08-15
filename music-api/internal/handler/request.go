package handler

type CreateMusicRequest struct {
	ArtistName string `json:"artist_name"`
	SongTitle  string `json:"song_title"`
	Genre      string `json:"genre"`
	ImageURL   string `json:"image_url"`
	AudioKey   string `json:"audio_key"`
}

type UpdateMusicRequest struct {
	ArtistName *string `json:"artist_name"`
	SongTitle  *string `json:"song_title"`
	Genre      *string `json:"genre"`
	ImageURL   *string `json:"image_url"`
	AudioKey   *string `json:"audio_key"`
}
