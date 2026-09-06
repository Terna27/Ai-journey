package services

import (
	"context"
	"errors"
	"strings"

	"github.com/jackc/pgx/v5"

	"music-api/internal/models"
	"music-api/internal/repository"
)

var (
	ErrPlaylistNameRequired = errors.New("playlist name is required")
	ErrPlaylistNameTooLong  = errors.New("playlist name is too long")
	ErrPlaylistNotFound     = errors.New("playlist not found")
	ErrPlaylistTrackExists  = errors.New("music already exists in playlist")
	ErrPlaylistTrackMissing = errors.New("music is not in playlist")
)

type PlaylistRepo interface {
	Create(
		ctx context.Context,
		userID int,
		name string,
		description string,
		isPublic bool,
	) (models.Playlist, error)

	GetByID(
		ctx context.Context,
		playlistID int64,
	) (models.Playlist, error)

	GetByUserID(
		ctx context.Context,
		userID int,
	) ([]models.Playlist, error)

	Update(
		ctx context.Context,
		playlistID int64,
		userID int,
		name string,
		description string,
		isPublic bool,
	) (models.Playlist, error)

	Delete(
		ctx context.Context,
		playlistID int64,
		userID int,
	) error

	AddTrack(
		ctx context.Context,
		playlistID int64,
		userID int,
		musicID int,
	) error

	RemoveTrack(
		ctx context.Context,
		playlistID int64,
		userID int,
		musicID int,
	) error

	GetTracks(
		ctx context.Context,
		playlistID int64,
	) ([]models.Music, error)
}

type PlaylistService struct {
	repo PlaylistRepo
}

func NewPlaylistService(
	repo PlaylistRepo,
) *PlaylistService {
	return &PlaylistService{
		repo: repo,
	}
}

func (s *PlaylistService) CreatePlaylist(
	ctx context.Context,
	userID int,
	name string,
	description string,
	isPublic bool,
) (models.Playlist, error) {
	name = strings.TrimSpace(name)
	description = strings.TrimSpace(description)

	if userID <= 0 {
		return models.Playlist{},
			ErrUserAuthenticationRequired
	}

	if name == "" {
		return models.Playlist{},
			ErrPlaylistNameRequired
	}

	if len(name) > 120 {
		return models.Playlist{},
			ErrPlaylistNameTooLong
	}

	return s.repo.Create(
		ctx,
		userID,
		name,
		description,
		isPublic,
	)
}

func (s *PlaylistService) GetUserPlaylists(
	ctx context.Context,
	userID int,
) ([]models.Playlist, error) {
	if userID <= 0 {
		return nil,
			ErrUserAuthenticationRequired
	}

	return s.repo.GetByUserID(
		ctx,
		userID,
	)
}

func (s *PlaylistService) GetPlaylist(
	ctx context.Context,
	playlistID int64,
	userID int,
) (models.PlaylistDetails, error) {
	if userID <= 0 {
		return models.PlaylistDetails{},
			ErrUserAuthenticationRequired
	}

	playlist, err :=
		s.repo.GetByID(
			ctx,
			playlistID,
		)

	if err != nil {
		if errors.Is(
			err,
			repository.ErrPlaylistNotFound,
		) {
			return models.PlaylistDetails{},
				ErrPlaylistNotFound
		}

		return models.PlaylistDetails{},
			err
	}

	if playlist.UserID != userID &&
		!playlist.IsPublic {
		return models.PlaylistDetails{},
			ErrPlaylistNotFound
	}

	tracks, err :=
		s.repo.GetTracks(
			ctx,
			playlistID,
		)

	if err != nil {
		return models.PlaylistDetails{},
			err
	}

	return models.PlaylistDetails{
		Playlist: playlist,
		Tracks:   tracks,
	}, nil
}

func (s *PlaylistService) UpdatePlaylist(
	ctx context.Context,
	playlistID int64,
	userID int,
	name string,
	description string,
	isPublic bool,
) (models.Playlist, error) {
	name = strings.TrimSpace(name)
	description = strings.TrimSpace(description)

	if userID <= 0 {
		return models.Playlist{},
			ErrUserAuthenticationRequired
	}

	if playlistID <= 0 {
		return models.Playlist{},
			ErrPlaylistNotFound
	}

	if name == "" {
		return models.Playlist{},
			ErrPlaylistNameRequired
	}

	if len(name) > 120 {
		return models.Playlist{},
			ErrPlaylistNameTooLong
	}

	playlist, err :=
		s.repo.Update(
			ctx,
			playlistID,
			userID,
			name,
			description,
			isPublic,
		)

	if errors.Is(
		err,
		repository.ErrPlaylistNotFound,
	) {
		return models.Playlist{},
			ErrPlaylistNotFound
	}

	if err != nil {
		return models.Playlist{},
			err
	}

	return playlist, nil
}

func (s *PlaylistService) DeletePlaylist(
	ctx context.Context,
	playlistID int64,
	userID int,
) error {
	if userID <= 0 {
		return ErrUserAuthenticationRequired
	}

	if playlistID <= 0 {
		return ErrPlaylistNotFound
	}

	err := s.repo.Delete(
		ctx,
		playlistID,
		userID,
	)

	if errors.Is(
		err,
		repository.ErrPlaylistNotFound,
	) {
		return ErrPlaylistNotFound
	}

	return err
}

func (s *PlaylistService) AddTrack(
	ctx context.Context,
	playlistID int64,
	userID int,
	musicID int,
) error {
	if userID <= 0 {
		return ErrUserAuthenticationRequired
	}

	if playlistID <= 0 {
		return ErrPlaylistNotFound
	}

	if musicID <= 0 {
		return ErrMusicNotFound
	}

	err := s.repo.AddTrack(
		ctx,
		playlistID,
		userID,
		musicID,
	)

	switch {
	case err == nil:
		return nil

	case errors.Is(
		err,
		repository.ErrPlaylistNotFound,
	):
		return ErrPlaylistNotFound

	case errors.Is(
		err,
		repository.ErrPlaylistTrackExists,
	):
		return ErrPlaylistTrackExists

	case errors.Is(
		err,
		pgx.ErrNoRows,
	):
		return ErrMusicNotFound

	default:
		return err
	}
}

func (s *PlaylistService) RemoveTrack(
	ctx context.Context,
	playlistID int64,
	userID int,
	musicID int,
) error {
	if userID <= 0 {
		return ErrUserAuthenticationRequired
	}

	if playlistID <= 0 {
		return ErrPlaylistNotFound
	}

	if musicID <= 0 {
		return ErrMusicNotFound
	}

	err := s.repo.RemoveTrack(
		ctx,
		playlistID,
		userID,
		musicID,
	)

	switch {
	case err == nil:
		return nil

	case errors.Is(
		err,
		repository.ErrPlaylistNotFound,
	):
		return ErrPlaylistNotFound

	case errors.Is(
		err,
		repository.ErrPlaylistTrackNotFound,
	):
		return ErrPlaylistTrackMissing

	default:
		return err
	}
}
