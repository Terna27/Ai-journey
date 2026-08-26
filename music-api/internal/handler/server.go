package handler

import "music-api/internal/services"

type MusicHandler struct {
	Service           *services.MusicService
	ArtistService     *services.ArtistService
	JWTService        *services.JWTService
	CloudinaryService *services.CloudinaryService
}

func NewMusicHandler(
	svc *services.MusicService,
	artistService *services.ArtistService,
	jwtService *services.JWTService,
	cloudinaryService *services.CloudinaryService,
) *MusicHandler {
	return &MusicHandler{
		Service:           svc,
		ArtistService:     artistService,
		JWTService:        jwtService,
		CloudinaryService: cloudinaryService,
	}
}
