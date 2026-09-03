package services

import (
	"context"
	"errors"
	"strings"

	"music-api/internal/models"
	"music-api/internal/repository"

	"github.com/jackc/pgx/v5"
)

// MusicRepo defines the repository operations required by MusicService.
//
// Ownership-sensitive operations receive artistID so that the repository can
// enforce ownership directly in the database.
type MusicRepo interface {
	Create(
		ctx context.Context,
		artistID int,
		m models.Music,
	) (models.Music, error)

	GetAll(
		ctx context.Context,
		search string,
		genre string,
		sortBy string,
		page int,
		limit int,
	) ([]models.Music, error)

	GetByID(
		ctx context.Context,
		id int,
	) (models.Music, error)

	Update(
		ctx context.Context,
		id int,
		artistID int,
		m models.Music,
	) (models.Music, error)

	Delete(
		ctx context.Context,
		id int,
		artistID int,
	) error

	RecordLike(
		ctx context.Context,
		musicID int,
		likerID string,
	) (models.Music, error)
}

// Domain errors.
var (
	ErrArtistNameRequired = errors.New("artist name cannot be empty")
	ErrSongTitleRequired  = errors.New("song title cannot be empty")
	ErrGenreRequired      = errors.New("genre cannot be empty")
	ErrMusicNotFound      = errors.New("music post not found")
	ErrAlreadyLiked       = errors.New("already liked")
	ErrUnauthorized       = errors.New("artist authentication required")
)

// CreateMusicInput contains the fields needed to create or fully replace
// a music post.
type CreateMusicInput struct {
	ArtistName    string
	SongTitle     string
	Genre         string
	ImageURL      string
	ImagePublicID string
	AudioURL      string
	AudioPublicID string
	AudioKey      string
}

// UpdateMusicInput represents a partial update.
//
// nil means that the field should remain unchanged.
type UpdateMusicInput struct {
	ArtistName *string
	SongTitle  *string
	Genre      *string
	ImageURL   *string
	AudioKey   *string
}

// MusicService contains the business logic for music.
type MusicService struct {
	repo MusicRepo
}

func NewMusicService(repo MusicRepo) *MusicService {
	return &MusicService{
		repo: repo,
	}
}

// CreateMusic validates the music data and creates a track owned by the
// authenticated artist.
func (s *MusicService) CreateMusic(
	ctx context.Context,
	artistID int,
	in CreateMusicInput,
) (models.Music, error) {
	if artistID <= 0 {
		return models.Music{}, ErrUnauthorized
	}

	music, err := buildMusic(in)
	if err != nil {
		return models.Music{}, err
	}

	return s.repo.Create(
		ctx,
		artistID,
		music,
	)
}

// ListMusic returns music matching the supplied filters.
func (s *MusicService) ListMusic(
	ctx context.Context,
	search string,
	genre string,
	sortBy string,
	page int,
	limit int,
) ([]models.Music, error) {
	return s.repo.GetAll(
		ctx,
		search,
		genre,
		sortBy,
		page,
		limit,
	)
}

// GetMusic returns one music post.
func (s *MusicService) GetMusic(
	ctx context.Context,
	id int,
) (models.Music, error) {
	music, err := s.repo.GetByID(
		ctx,
		id,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return models.Music{}, ErrMusicNotFound
		}

		return models.Music{}, err
	}

	return music, nil
}

// UpdateMusic fully updates a track.
//
// The repository receives artistID and must only update the track when the
// authenticated artist owns it.
func (s *MusicService) UpdateMusic(
	ctx context.Context,
	id int,
	artistID int,
	in CreateMusicInput,
) (models.Music, error) {
	if artistID <= 0 {
		return models.Music{}, ErrUnauthorized
	}

	music, err := buildMusic(in)
	if err != nil {
		return models.Music{}, err
	}

	updated, err := s.repo.Update(
		ctx,
		id,
		artistID,
		music,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return models.Music{}, ErrMusicNotFound
		}

		return models.Music{}, err
	}

	return updated, nil
}

// PatchMusic partially updates a track.
//
// The final database update is ownership protected using artistID.
func (s *MusicService) PatchMusic(
	ctx context.Context,
	id int,
	artistID int,
	in UpdateMusicInput,
) (models.Music, error) {
	if artistID <= 0 {
		return models.Music{}, ErrUnauthorized
	}

	music, err := s.repo.GetByID(
		ctx,
		id,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return models.Music{}, ErrMusicNotFound
		}

		return models.Music{}, err
	}

	if in.ArtistName != nil {
		name := strings.TrimSpace(
			*in.ArtistName,
		)

		if name == "" {
			return models.Music{}, ErrArtistNameRequired
		}

		music.ArtistName = name
	}

	if in.SongTitle != nil {
		title := strings.TrimSpace(
			*in.SongTitle,
		)

		if title == "" {
			return models.Music{}, ErrSongTitleRequired
		}

		music.SongTitle = title
	}

	if in.Genre != nil {
		genre := strings.TrimSpace(
			*in.Genre,
		)

		if genre == "" {
			return models.Music{}, ErrGenreRequired
		}

		music.Genre = genre
	}

	if in.ImageURL != nil {
		music.ImageURL = strings.TrimSpace(
			*in.ImageURL,
		)
	}

	if in.AudioKey != nil {
		music.AudioKey = strings.TrimSpace(
			*in.AudioKey,
		)
	}

	updated, err := s.repo.Update(
		ctx,
		id,
		artistID,
		music,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return models.Music{}, ErrMusicNotFound
		}

		return models.Music{}, err
	}

	return updated, nil
}

// DeleteMusic deletes a track only when it belongs to the authenticated
// artist.
func (s *MusicService) DeleteMusic(
	ctx context.Context,
	id int,
	artistID int,
) error {
	if artistID <= 0 {
		return ErrUnauthorized
	}

	err := s.repo.Delete(
		ctx,
		id,
		artistID,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrMusicNotFound
		}

		return err
	}

	return nil
}

// LikeMusic records a like.
//
// Likes are not restricted by music ownership.
func (s *MusicService) LikeMusic(
	ctx context.Context,
	id int,
	likerID string,
) (models.Music, error) {
	music, err := s.repo.RecordLike(
		ctx,
		id,
		likerID,
	)

	if err != nil {
		switch {
		case errors.Is(err, pgx.ErrNoRows):
			return models.Music{}, ErrMusicNotFound

		case errors.Is(
			err,
			repository.ErrAlreadyLiked,
		):
			return models.Music{}, ErrAlreadyLiked

		default:
			return models.Music{}, err
		}
	}

	return music, nil
}

// buildMusic trims and validates fields shared by create and full update.
func buildMusic(
	in CreateMusicInput,
) (models.Music, error) {
	artist := strings.TrimSpace(
		in.ArtistName,
	)

	if artist == "" {
		return models.Music{}, ErrArtistNameRequired
	}

	title := strings.TrimSpace(
		in.SongTitle,
	)

	if title == "" {
		return models.Music{}, ErrSongTitleRequired
	}

	genre := strings.TrimSpace(
		in.Genre,
	)

	if genre == "" {
		return models.Music{}, ErrGenreRequired
	}

	return models.Music{
		ArtistName: artist,
		SongTitle:  title,
		Genre:      genre,

		ImageURL: strings.TrimSpace(
			in.ImageURL,
		),

		ImagePublicID: strings.TrimSpace(
			in.ImagePublicID,
		),

		AudioURL: strings.TrimSpace(
			in.AudioURL,
		),

		AudioPublicID: strings.TrimSpace(
			in.AudioPublicID,
		),

		AudioKey: strings.TrimSpace(
			in.AudioKey,
		),
	}, nil
}
