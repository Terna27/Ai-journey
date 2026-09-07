package services

import (
	"context"
	"errors"
	"strings"

	"music-api/internal/models"
	"music-api/internal/repository"

	"github.com/jackc/pgx/v5"
)

// MusicRepo defines the music repository operations required by MusicService.
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

	GetByArtistID(
		ctx context.Context,
		artistID int,
	) ([]models.Music, error)

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

	// RecordLike is temporarily retained because existing tests and
	// legacy code still depend on it. New authenticated likes use
	// UserMusicLikeRepo instead.
	RecordLike(
		ctx context.Context,
		musicID int,
		likerID string,
	) (models.Music, error)
}

// UserMusicLikeRepo defines authenticated-user library operations.
//
// Keeping this as an interface also makes the service easy to unit test
// without requiring a real PostgreSQL database.
type UserMusicLikeRepo interface {
	LikeMusic(
		ctx context.Context,
		userID int,
		musicID int,
	) (models.Music, error)

	UnlikeMusic(
		ctx context.Context,
		userID int,
		musicID int,
	) (models.Music, error)

	GetLikedMusic(
		ctx context.Context,
		userID int,
	) ([]models.Music, error)

	IsLiked(
		ctx context.Context,
		userID int,
		musicID int,
	) (bool, error)
}

// Domain errors.
var (
	ErrArtistNameRequired = errors.New(
		"artist name cannot be empty",
	)

	ErrSongTitleRequired = errors.New(
		"song title cannot be empty",
	)

	ErrGenreRequired = errors.New(
		"genre cannot be empty",
	)

	ErrMusicNotFound = errors.New(
		"music post not found",
	)

	ErrAlreadyLiked = errors.New(
		"already liked",
	)

	ErrNotLiked = errors.New(
		"music is not liked by this user",
	)

	ErrUnauthorized = errors.New(
		"artist authentication required",
	)

	ErrUserAuthenticationRequired = errors.New(
		"user authentication required",
	)
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

	userMusicLikeRepo UserMusicLikeRepo
}

// NewMusicService creates the music service.
//
// userMusicLikeRepo is optional temporarily so older unit tests that call
// NewMusicService(repo) continue to compile while we migrate the like system.
// Production wiring passes the authenticated-user like repository.
func NewMusicService(
	repo MusicRepo,
	userMusicLikeRepos ...UserMusicLikeRepo,
) *MusicService {
	service := &MusicService{
		repo: repo,
	}

	if len(userMusicLikeRepos) > 0 {
		service.userMusicLikeRepo =
			userMusicLikeRepos[0]
	}

	return service
}

// CreateMusic validates the music data and creates a track owned by the
// authenticated artist.
func (s *MusicService) CreateMusic(
	ctx context.Context,
	artistID int,
	in CreateMusicInput,
) (models.Music, error) {
	if artistID <= 0 {
		return models.Music{},
			ErrUnauthorized
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
		if errors.Is(
			err,
			pgx.ErrNoRows,
		) {
			return models.Music{},
				ErrMusicNotFound
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
		return models.Music{},
			ErrUnauthorized
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
		if errors.Is(
			err,
			pgx.ErrNoRows,
		) {
			return models.Music{},
				ErrMusicNotFound
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
		return models.Music{},
			ErrUnauthorized
	}

	music, err := s.repo.GetByID(
		ctx,
		id,
	)

	if err != nil {
		if errors.Is(
			err,
			pgx.ErrNoRows,
		) {
			return models.Music{},
				ErrMusicNotFound
		}

		return models.Music{}, err
	}

	if in.ArtistName != nil {
		name := strings.TrimSpace(
			*in.ArtistName,
		)

		if name == "" {
			return models.Music{},
				ErrArtistNameRequired
		}

		music.ArtistName = name
	}

	if in.SongTitle != nil {
		title := strings.TrimSpace(
			*in.SongTitle,
		)

		if title == "" {
			return models.Music{},
				ErrSongTitleRequired
		}

		music.SongTitle = title
	}

	if in.Genre != nil {
		genre := strings.TrimSpace(
			*in.Genre,
		)

		if genre == "" {
			return models.Music{},
				ErrGenreRequired
		}

		music.Genre = genre
	}

	if in.ImageURL != nil {
		music.ImageURL =
			strings.TrimSpace(
				*in.ImageURL,
			)
	}

	if in.AudioKey != nil {
		music.AudioKey =
			strings.TrimSpace(
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
		if errors.Is(
			err,
			pgx.ErrNoRows,
		) {
			return models.Music{},
				ErrMusicNotFound
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
		if errors.Is(
			err,
			pgx.ErrNoRows,
		) {
			return ErrMusicNotFound
		}

		return err
	}

	return nil
}

// LikeMusic is the temporary legacy IP-based like operation.
//
// It remains during this migration because existing tests still cover it.
// The active authenticated API will be switched to LikeMusicForUser.
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
		case errors.Is(
			err,
			pgx.ErrNoRows,
		):
			return models.Music{},
				ErrMusicNotFound

		case errors.Is(
			err,
			repository.ErrAlreadyLiked,
		):
			return models.Music{},
				ErrAlreadyLiked

		default:
			return models.Music{}, err
		}
	}

	return music, nil
}

// LikeMusicForUser records a like belonging to an authenticated user.
func (s *MusicService) LikeMusicForUser(
	ctx context.Context,
	userID int,
	musicID int,
) (models.Music, error) {
	if userID <= 0 {
		return models.Music{},
			ErrUserAuthenticationRequired
	}

	if s.userMusicLikeRepo == nil {
		return models.Music{},
			errors.New(
				"user music like repository is not configured",
			)
	}

	music, err :=
		s.userMusicLikeRepo.LikeMusic(
			ctx,
			userID,
			musicID,
		)

	if err != nil {
		switch {
		case errors.Is(
			err,
			pgx.ErrNoRows,
		):
			return models.Music{},
				ErrMusicNotFound

		case errors.Is(
			err,
			repository.ErrAlreadyLiked,
		):
			return models.Music{},
				ErrAlreadyLiked

		default:
			return models.Music{}, err
		}
	}

	return music, nil
}

// UnlikeMusicForUser removes a like belonging to an authenticated user.
func (s *MusicService) UnlikeMusicForUser(
	ctx context.Context,
	userID int,
	musicID int,
) (models.Music, error) {
	if userID <= 0 {
		return models.Music{},
			ErrUserAuthenticationRequired
	}

	if s.userMusicLikeRepo == nil {
		return models.Music{},
			errors.New(
				"user music like repository is not configured",
			)
	}

	music, err :=
		s.userMusicLikeRepo.UnlikeMusic(
			ctx,
			userID,
			musicID,
		)

	if err != nil {
		switch {
		case errors.Is(
			err,
			pgx.ErrNoRows,
		):
			return models.Music{},
				ErrMusicNotFound

		case errors.Is(
			err,
			repository.ErrNotLiked,
		):
			return models.Music{},
				ErrNotLiked

		default:
			return models.Music{}, err
		}
	}

	return music, nil
}

// GetLikedMusic returns the authenticated user's saved/liked tracks.
func (s *MusicService) GetLikedMusic(
	ctx context.Context,
	userID int,
) ([]models.Music, error) {
	if userID <= 0 {
		return nil,
			ErrUserAuthenticationRequired
	}

	if s.userMusicLikeRepo == nil {
		return nil,
			errors.New(
				"user music like repository is not configured",
			)
	}

	return s.userMusicLikeRepo.GetLikedMusic(
		ctx,
		userID,
	)
}

// IsMusicLiked reports whether an authenticated user likes a track.
func (s *MusicService) IsMusicLiked(
	ctx context.Context,
	userID int,
	musicID int,
) (bool, error) {
	if userID <= 0 {
		return false,
			ErrUserAuthenticationRequired
	}

	if s.userMusicLikeRepo == nil {
		return false,
			errors.New(
				"user music like repository is not configured",
			)
	}

	return s.userMusicLikeRepo.IsLiked(
		ctx,
		userID,
		musicID,
	)
}

// buildMusic trims and validates fields shared by create and full update.
func buildMusic(
	in CreateMusicInput,
) (models.Music, error) {
	artist := strings.TrimSpace(
		in.ArtistName,
	)

	if artist == "" {
		return models.Music{},
			ErrArtistNameRequired
	}

	title := strings.TrimSpace(
		in.SongTitle,
	)

	if title == "" {
		return models.Music{},
			ErrSongTitleRequired
	}

	genre := strings.TrimSpace(
		in.Genre,
	)

	if genre == "" {
		return models.Music{},
			ErrGenreRequired
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

// GetMusicByArtistID returns all tracks owned by an artist.
func (s *MusicService) GetMusicByArtistID(
	ctx context.Context,
	artistID int,
) ([]models.Music, error) {
	if artistID < 1 {
		return []models.Music{}, nil
	}

	return s.repo.GetByArtistID(
		ctx,
		artistID,
	)
}
