package service

import (
	"context"
	"errors"
	"strings"

	"music-api/internal/models"
	"music-api/internal/repository"

	"github.com/jackc/pgx/v5"
)

// MusicRepo is the slice of the repository the service depends on. Declaring
// the interface here (on the consumer side) lets the service be unit-tested
// with a fake and keeps this package from importing a concrete database type.
// The concrete *repository.MusicRepository satisfies it structurally.
type MusicRepo interface {
	Create(ctx context.Context, m models.Music) (models.Music, error)
	GetAll(ctx context.Context, search, genre, sortBy string, page, limit int) ([]models.Music, error)
	GetByID(ctx context.Context, id int) (models.Music, error)
	Update(ctx context.Context, id int, m models.Music) (models.Music, error)
	Delete(ctx context.Context, id int) error
	RecordLike(ctx context.Context, musicID int, likerID string) (models.Music, error)
}

// Domain errors. Each layer translates the errors of the layer below into its
// own vocabulary; handlers map these to HTTP status codes with errors.Is.
var (
	ErrArtistNameRequired = errors.New("artist name cannot be empty")
	ErrSongTitleRequired  = errors.New("song title cannot be empty")
	ErrGenreRequired      = errors.New("genre cannot be empty")
	ErrMusicNotFound      = errors.New("music post not found")
	ErrAlreadyLiked       = errors.New("already liked")
)

// CreateMusicInput carries the fields needed to create or fully replace a
// music post. Validation is the service's responsibility, not the caller's.
type CreateMusicInput struct {
	ArtistName string
	SongTitle  string
	Genre      string
	ImageURL   string
	AudioKey   string
}

// UpdateMusicInput carries a partial update. A nil pointer means "leave this
// field unchanged"; a non-nil pointer to an empty string is a validation error
// for the required fields.
type UpdateMusicInput struct {
	ArtistName *string
	SongTitle  *string
	Genre      *string
	ImageURL   *string
	AudioKey   *string
}

// MusicService holds the business rules for music posts.
type MusicService struct {
	repo MusicRepo
}

func NewMusicService(repo MusicRepo) *MusicService {
	return &MusicService{repo: repo}
}

// CreateMusic validates required fields and persists a new music post.
func (s *MusicService) CreateMusic(ctx context.Context, in CreateMusicInput) (models.Music, error) {
	music, err := buildMusic(in)
	if err != nil {
		return models.Music{}, err
	}

	return s.repo.Create(ctx, music)
}

// ListMusic returns music posts matching the given filters. Query-string
// parsing (page/limit bounds, etc.) is HTTP concern and stays in the handler;
// the service receives already-parsed values.
func (s *MusicService) ListMusic(ctx context.Context, search, genre, sortBy string, page, limit int) ([]models.Music, error) {
	return s.repo.GetAll(ctx, search, genre, sortBy, page, limit)
}

// GetMusic returns a single music post, or ErrMusicNotFound.
func (s *MusicService) GetMusic(ctx context.Context, id int) (models.Music, error) {
	music, err := s.repo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return models.Music{}, ErrMusicNotFound
		}
		return models.Music{}, err
	}
	return music, nil
}

// UpdateMusic validates and fully replaces an existing music post.
func (s *MusicService) UpdateMusic(ctx context.Context, id int, in CreateMusicInput) (models.Music, error) {
	music, err := buildMusic(in)
	if err != nil {
		return models.Music{}, err
	}

	updated, err := s.repo.Update(ctx, id, music)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return models.Music{}, ErrMusicNotFound
		}
		return models.Music{}, err
	}
	return updated, nil
}

// PatchMusic applies a partial update: it loads the current post, overlays the
// provided fields, and saves. Required fields may not be set to empty.
func (s *MusicService) PatchMusic(ctx context.Context, id int, in UpdateMusicInput) (models.Music, error) {
	music, err := s.repo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return models.Music{}, ErrMusicNotFound
		}
		return models.Music{}, err
	}

	if in.ArtistName != nil {
		name := strings.TrimSpace(*in.ArtistName)
		if name == "" {
			return models.Music{}, ErrArtistNameRequired
		}
		music.ArtistName = name
	}

	if in.SongTitle != nil {
		title := strings.TrimSpace(*in.SongTitle)
		if title == "" {
			return models.Music{}, ErrSongTitleRequired
		}
		music.SongTitle = title
	}

	if in.Genre != nil {
		genre := strings.TrimSpace(*in.Genre)
		if genre == "" {
			return models.Music{}, ErrGenreRequired
		}
		music.Genre = genre
	}

	if in.ImageURL != nil {
		music.ImageURL = strings.TrimSpace(*in.ImageURL)
	}

	if in.AudioKey != nil {
		music.AudioKey = strings.TrimSpace(*in.AudioKey)
	}

	updated, err := s.repo.Update(ctx, id, music)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return models.Music{}, ErrMusicNotFound
		}
		return models.Music{}, err
	}
	return updated, nil
}

// DeleteMusic removes a music post, or returns ErrMusicNotFound.
func (s *MusicService) DeleteMusic(ctx context.Context, id int) error {
	err := s.repo.Delete(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrMusicNotFound
		}
		return err
	}
	return nil
}

// LikeMusic records a like from likerID. The business rule "a caller can only
// like a song once" is enforced by the repository; this method translates its
// errors into the service vocabulary.
func (s *MusicService) LikeMusic(ctx context.Context, id int, likerID string) (models.Music, error) {
	music, err := s.repo.RecordLike(ctx, id, likerID)
	if err != nil {
		switch {
		case errors.Is(err, pgx.ErrNoRows):
			return models.Music{}, ErrMusicNotFound
		case errors.Is(err, repository.ErrAlreadyLiked):
			return models.Music{}, ErrAlreadyLiked
		default:
			return models.Music{}, err
		}
	}
	return music, nil
}

// buildMusic trims and validates the required fields shared by create and
// full-update, returning a models.Music ready to persist.
func buildMusic(in CreateMusicInput) (models.Music, error) {
	artist := strings.TrimSpace(in.ArtistName)
	if artist == "" {
		return models.Music{}, ErrArtistNameRequired
	}

	title := strings.TrimSpace(in.SongTitle)
	if title == "" {
		return models.Music{}, ErrSongTitleRequired
	}

	genre := strings.TrimSpace(in.Genre)
	if genre == "" {
		return models.Music{}, ErrGenreRequired
	}

	return models.Music{
		ArtistName: artist,
		SongTitle:  title,
		Genre:      genre,
		ImageURL:   strings.TrimSpace(in.ImageURL),
		AudioKey:   strings.TrimSpace(in.AudioKey),
	}, nil
}
