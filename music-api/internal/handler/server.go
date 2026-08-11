package handler

import "music-api/internal/service"

type MusicHandler struct {
	Service *service.MusicService
}

func NewMusicHandler(svc *service.MusicService) *MusicHandler {
	return &MusicHandler{
		Service: svc,
	}
}
