package services

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"

	"music-api/internal/models"
	"music-api/internal/repository"
)

var (
	ErrReleaseNotFound = errors.New(
		"release not found",
	)

	ErrReleaseTitleRequired = errors.New(
		"release title is required",
	)

	ErrInvalidReleaseType = errors.New(
		"invalid release type",
	)

	ErrInvalidReleaseTrackNumber = errors.New(
		"track number must be greater than zero",
	)

	ErrReleaseTrackNotOwned = errors.New(
		"track does not belong to authenticated artist",
	)

	ErrReleaseTrackAlreadyAssigned = errors.New(
		"track already belongs to another release",
	)
)

// ReleaseRepository defines the persistence operations required by
// ReleaseService.
type ReleaseRepository interface {
	Create(
		ctx context.Context,
		release models.Release,
	) (models.Release, error)

	GetByID(
		ctx context.Context,
		id int,
	) (models.Release, error)

	GetByArtistID(
		ctx context.Context,
		artistID int,
	) ([]models.Release, error)

	GetTracks(
		ctx context.Context,
		releaseID int,
	) ([]models.Music, error)

	AddTrack(
		ctx context.Context,
		releaseID int,
		artistID int,
		musicID int,
		trackNumber int,
	) error

	RemoveTrack(
		ctx context.Context,
		releaseID int,
		artistID int,
		musicID int,
	) error

	Update(
		ctx context.Context,
		release models.Release,
	) (models.Release, error)

	Delete(
		ctx context.Context,
		id int,
		artistID int,
	) error
}

// ReleaseService contains release business rules.
type ReleaseService struct {
	repo ReleaseRepository
}

// CreateReleaseInput contains fields accepted when creating a release.
type CreateReleaseInput struct {
	Title string

	ReleaseType models.ReleaseType

	CoverImageURL string

	CoverImagePublicID string

	Description string

	ReleaseDate *time.Time

	IsPublished bool
}

// UpdateReleaseInput contains the complete editable release state.
type UpdateReleaseInput struct {
	Title string

	ReleaseType models.ReleaseType

	CoverImageURL string

	CoverImagePublicID string

	Description string

	ReleaseDate *time.Time

	IsPublished bool
}

// NewReleaseService creates the release service.
func NewReleaseService(
	repo ReleaseRepository,
) *ReleaseService {
	return &ReleaseService{
		repo: repo,
	}
}

// CreateRelease creates a release owned by the authenticated artist.
func (s *ReleaseService) CreateRelease(
	ctx context.Context,
	artistID int,
	in CreateReleaseInput,
) (models.Release, error) {
	if artistID <= 0 {
		return models.Release{},
			ErrUnauthorized
	}

	title := strings.TrimSpace(
		in.Title,
	)

	if title == "" {
		return models.Release{},
			ErrReleaseTitleRequired
	}

	if !validReleaseType(
		in.ReleaseType,
	) {
		return models.Release{},
			ErrInvalidReleaseType
	}

	release := models.Release{
		ArtistID: artistID,

		Title: title,

		ReleaseType: in.ReleaseType,

		CoverImageURL: strings.TrimSpace(
			in.CoverImageURL,
		),

		CoverImagePublicID: strings.TrimSpace(
			in.CoverImagePublicID,
		),

		Description: strings.TrimSpace(
			in.Description,
		),

		ReleaseDate: in.ReleaseDate,

		IsPublished: in.IsPublished,
	}

	return s.repo.Create(
		ctx,
		release,
	)
}

// GetRelease returns a release and its ordered tracks.
func (s *ReleaseService) GetRelease(
	ctx context.Context,
	id int,
) (models.ReleaseDetails, error) {
	if id <= 0 {
		return models.ReleaseDetails{},
			ErrReleaseNotFound
	}

	release, err :=
		s.repo.GetByID(
			ctx,
			id,
		)

	if err != nil {
		if errors.Is(
			err,
			pgx.ErrNoRows,
		) {
			return models.ReleaseDetails{},
				ErrReleaseNotFound
		}

		return models.ReleaseDetails{},
			err
	}

	tracks, err :=
		s.repo.GetTracks(
			ctx,
			id,
		)

	if err != nil {
		return models.ReleaseDetails{},
			err
	}

	return models.ReleaseDetails{
		Release: release,
		Tracks:  tracks,
	}, nil
}

// GetArtistReleases returns releases belonging to an artist.
func (s *ReleaseService) GetArtistReleases(
	ctx context.Context,
	artistID int,
) ([]models.Release, error) {
	if artistID <= 0 {
		return []models.Release{},
			nil
	}

	return s.repo.GetByArtistID(
		ctx,
		artistID,
	)
}

// UpdateRelease updates an artist-owned release.
func (s *ReleaseService) UpdateRelease(
	ctx context.Context,
	id int,
	artistID int,
	in UpdateReleaseInput,
) (models.Release, error) {
	if artistID <= 0 {
		return models.Release{},
			ErrUnauthorized
	}

	if id <= 0 {
		return models.Release{},
			ErrReleaseNotFound
	}

	title := strings.TrimSpace(
		in.Title,
	)

	if title == "" {
		return models.Release{},
			ErrReleaseTitleRequired
	}

	if !validReleaseType(
		in.ReleaseType,
	) {
		return models.Release{},
			ErrInvalidReleaseType
	}

	release := models.Release{
		ID: id,

		ArtistID: artistID,

		Title: title,

		ReleaseType: in.ReleaseType,

		CoverImageURL: strings.TrimSpace(
			in.CoverImageURL,
		),

		CoverImagePublicID: strings.TrimSpace(
			in.CoverImagePublicID,
		),

		Description: strings.TrimSpace(
			in.Description,
		),

		ReleaseDate: in.ReleaseDate,

		IsPublished: in.IsPublished,
	}

	updated, err :=
		s.repo.Update(
			ctx,
			release,
		)

	if err != nil {
		if errors.Is(
			err,
			pgx.ErrNoRows,
		) {
			return models.Release{},
				ErrReleaseNotFound
		}

		return models.Release{},
			err
	}

	return updated, nil
}

// DeleteRelease deletes a release only when it belongs to the
// authenticated artist.
func (s *ReleaseService) DeleteRelease(
	ctx context.Context,
	id int,
	artistID int,
) error {
	if artistID <= 0 {
		return ErrUnauthorized
	}

	if id <= 0 {
		return ErrReleaseNotFound
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
			return ErrReleaseNotFound
		}

		return err
	}

	return nil
}

// AddTrack assigns an artist-owned track to an artist-owned release.
func (s *ReleaseService) AddTrack(
	ctx context.Context,
	releaseID int,
	artistID int,
	musicID int,
	trackNumber int,
) error {
	if artistID <= 0 {
		return ErrUnauthorized
	}

	if releaseID <= 0 {
		return ErrReleaseNotFound
	}

	if musicID <= 0 {
		return ErrMusicNotFound
	}

	if trackNumber <= 0 {
		return ErrInvalidReleaseTrackNumber
	}

	release, err :=
		s.repo.GetByID(
			ctx,
			releaseID,
		)

	if err != nil {
		if errors.Is(
			err,
			pgx.ErrNoRows,
		) {
			return ErrReleaseNotFound
		}

		return err
	}

	// Do not reveal another artist's release through a protected
	// ownership operation.
	if release.ArtistID != artistID {
		return ErrReleaseNotFound
	}

	err = s.repo.AddTrack(
		ctx,
		releaseID,
		artistID,
		musicID,
		trackNumber,
	)

	if err != nil {
		switch {
		case errors.Is(
			err,
			pgx.ErrNoRows,
		):
			return ErrMusicNotFound

		case errors.Is(
			err,
			repository.ErrReleaseTrackNotOwned,
		):
			return ErrReleaseTrackNotOwned

		case errors.Is(
			err,
			repository.ErrReleaseTrackAlreadyAssigned,
		):
			return ErrReleaseTrackAlreadyAssigned

		default:
			return err
		}
	}

	return nil
}

// RemoveTrack removes a track from an artist-owned release without
// deleting the underlying music record.
func (s *ReleaseService) RemoveTrack(
	ctx context.Context,
	releaseID int,
	artistID int,
	musicID int,
) error {
	if artistID <= 0 {
		return ErrUnauthorized
	}

	if releaseID <= 0 {
		return ErrReleaseNotFound
	}

	if musicID <= 0 {
		return ErrMusicNotFound
	}

	release, err :=
		s.repo.GetByID(
			ctx,
			releaseID,
		)

	if err != nil {
		if errors.Is(
			err,
			pgx.ErrNoRows,
		) {
			return ErrReleaseNotFound
		}

		return err
	}

	if release.ArtistID != artistID {
		return ErrReleaseNotFound
	}

	err = s.repo.RemoveTrack(
		ctx,
		releaseID,
		artistID,
		musicID,
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

func validReleaseType(
	releaseType models.ReleaseType,
) bool {
	switch releaseType {
	case models.ReleaseTypeSingle,
		models.ReleaseTypeEP,
		models.ReleaseTypeAlbum:
		return true

	default:
		return false
	}
}
