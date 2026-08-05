package handler

import "music-api/internal/repository"

type MusicHandler struct {
	Repo *repository.MusicRepository
}

func NewMusicHandler(repo *repository.MusicRepository) *MusicHandler {
	return &MusicHandler{
		Repo: repo,
	}
}
