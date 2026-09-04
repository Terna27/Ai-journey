package handler

import "music-api/internal/services"

type MusicHandler struct {
	Service           *services.MusicService
	ArtistService     *services.ArtistService
	UserService       *services.UserService
	JWTService        *services.JWTService
	CloudinaryService *services.CloudinaryService
}

func NewMusicHandler(
	svc *services.MusicService,
	artistService *services.ArtistService,
	userService *services.UserService,
	jwtService *services.JWTService,
	cloudinaryService *services.CloudinaryService,
) *MusicHandler {
	return &MusicHandler{
		Service:           svc,
		ArtistService:     artistService,
		UserService:       userService,
		JWTService:        jwtService,
		CloudinaryService: cloudinaryService,
	}
}
