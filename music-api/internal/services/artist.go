package services

import (
	"context"
	"errors"
	"strings"

	"music-api/internal/models"
	"music-api/internal/repository"

	"github.com/jackc/pgx/v5"
	"golang.org/x/crypto/bcrypt"
)

var (
	ErrArtistEmailRequired    = errors.New("artist email is required")
	ErrArtistPasswordRequired = errors.New("artist password is required")
	ErrArtistPasswordTooShort = errors.New("artist password must be at least 8 characters")
	ErrArtistProfileExists    = errors.New("artist profile already exists")
	ErrArtistBioTooLong       = errors.New("artist bio must not exceed 2000 characters")
)

type ArtistService struct {
	Repository *repository.ArtistRepository
}

func NewArtistService(
	repo *repository.ArtistRepository,
) *ArtistService {
	return &ArtistService{
		Repository: repo,
	}
}

func (s *ArtistService) Register(
	ctx context.Context,
	name string,
	email string,
	password string,
) (models.Artist, error) {
	name = strings.TrimSpace(name)
	email = strings.ToLower(
		strings.TrimSpace(email),
	)
	password = strings.TrimSpace(password)

	if name == "" {
		return models.Artist{}, ErrArtistNameRequired
	}

	if email == "" {
		return models.Artist{}, ErrArtistEmailRequired
	}

	if password == "" {
		return models.Artist{}, ErrArtistPasswordRequired
	}

	if len(password) < 8 {
		return models.Artist{}, ErrArtistPasswordTooShort
	}

	passwordHash, err :=
		bcrypt.GenerateFromPassword(
			[]byte(password),
			bcrypt.DefaultCost,
		)
	if err != nil {
		return models.Artist{}, err
	}

	artist := models.Artist{
		Name:         name,
		Email:        email,
		PasswordHash: string(passwordHash),
	}

	return s.Repository.Create(
		ctx,
		artist,
	)
}

func (s *ArtistService) GetByEmail(
	ctx context.Context,
	email string,
) (models.Artist, error) {
	email = strings.ToLower(
		strings.TrimSpace(email),
	)

	return s.Repository.GetByEmail(
		ctx,
		email,
	)
}

func (s *ArtistService) GetByID(
	ctx context.Context,
	id int,
) (models.Artist, error) {
	return s.Repository.GetByID(
		ctx,
		id,
	)
}

func (s *ArtistService) CreateForUser(
	ctx context.Context,
	user models.User,
) (models.Artist, error) {
	if user.ID <= 0 {
		return models.Artist{}, ErrUnauthorized
	}

	_, err := s.Repository.GetByUserID(
		ctx,
		user.ID,
	)

	if err == nil {
		return models.Artist{},
			ErrArtistProfileExists
	}

	if !errors.Is(err, pgx.ErrNoRows) {
		return models.Artist{}, err
	}

	artist, err :=
		s.Repository.CreateForUser(
			ctx,
			user,
		)

	if err != nil {
		if errors.Is(
			err,
			repository.ErrArtistProfileExists,
		) {
			return models.Artist{},
				ErrArtistProfileExists
		}

		return models.Artist{}, err
	}

	return artist, nil
}

func (s *ArtistService) GetByUserID(
	ctx context.Context,
	userID int,
) (models.Artist, error) {
	return s.Repository.GetByUserID(
		ctx,
		userID,
	)
}

type ArtistMediaAsset struct {
	URL      string
	PublicID string
}

type UpdateArtistProfileInput struct {
	Bio *string

	ProfileImage *ArtistMediaAsset

	HeroVideo *ArtistMediaAsset

	HeroVideoPoster *ArtistMediaAsset
}

func (s *ArtistService) UpdateProfile(
	ctx context.Context,
	artistID int,
	input UpdateArtistProfileInput,
) (models.Artist, error) {
	if artistID <= 0 {
		return models.Artist{}, ErrUnauthorized
	}

	artist, err := s.Repository.GetByID(
		ctx,
		artistID,
	)
	if err != nil {
		return models.Artist{}, err
	}

	if input.Bio != nil {
		bio := strings.TrimSpace(
			*input.Bio,
		)

		if len([]rune(bio)) > 2000 {
			return models.Artist{},
				ErrArtistBioTooLong
		}

		if bio == "" {
			artist.Bio = nil
		} else {
			artist.Bio = &bio
		}
	}

	if input.ProfileImage != nil {
		if err := validateArtistMediaAsset(
			*input.ProfileImage,
		); err != nil {
			return models.Artist{}, err
		}

		artist.ProfileImageURL =
			stringPointer(
				input.ProfileImage.URL,
			)

		artist.ProfileImagePublicID =
			stringPointer(
				input.ProfileImage.PublicID,
			)
	}

	if input.HeroVideo != nil {
		if err := validateArtistMediaAsset(
			*input.HeroVideo,
		); err != nil {
			return models.Artist{}, err
		}

		artist.HeroVideoURL =
			stringPointer(
				input.HeroVideo.URL,
			)

		artist.HeroVideoPublicID =
			stringPointer(
				input.HeroVideo.PublicID,
			)
	}

	if input.HeroVideoPoster != nil {
		if err := validateArtistMediaAsset(
			*input.HeroVideoPoster,
		); err != nil {
			return models.Artist{}, err
		}

		artist.HeroVideoPosterURL =
			stringPointer(
				input.HeroVideoPoster.URL,
			)

		artist.HeroVideoPosterPublicID =
			stringPointer(
				input.HeroVideoPoster.PublicID,
			)
	}

	return s.Repository.UpdateProfile(
		ctx,
		artist,
	)
}

func validateArtistMediaAsset(
	asset ArtistMediaAsset,
) error {
	if strings.TrimSpace(asset.URL) == "" ||
		strings.TrimSpace(asset.PublicID) == "" {
		return errors.New(
			"artist media asset is incomplete",
		)
	}

	return nil
}

func stringPointer(value string) *string {
	value = strings.TrimSpace(value)

	if value == "" {
		return nil
	}

	return &value
}
