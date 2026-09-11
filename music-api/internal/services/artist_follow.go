package services

import (
	"context"
	"errors"

	"music-api/internal/models"
	"music-api/internal/repository"

	"github.com/jackc/pgx/v5"
)

var (
	ErrInvalidFollowUserID = errors.New(
		"invalid user ID",
	)

	ErrInvalidFollowArtistID = errors.New(
		"invalid artist ID",
	)

	ErrFollowArtistNotFound = errors.New(
		"artist not found",
	)

	ErrCannotFollowOwnArtist = errors.New(
		"cannot follow your own artist profile",
	)
)

// ArtistFollowStatus contains the current user's follow relationship
// with an artist and the artist's authoritative follower count.
type ArtistFollowStatus struct {
	IsFollowing   bool  `json:"is_following"`
	FollowerCount int64 `json:"follower_count"`
}

type ArtistFollowService struct {
	followRepo *repository.ArtistFollowRepository
	artistRepo *repository.ArtistRepository
}

func NewArtistFollowService(
	followRepo *repository.ArtistFollowRepository,
	artistRepo *repository.ArtistRepository,
) *ArtistFollowService {
	return &ArtistFollowService{
		followRepo: followRepo,
		artistRepo: artistRepo,
	}
}

func (s *ArtistFollowService) Follow(
	ctx context.Context,
	userID int,
	artistID int,
) (ArtistFollowStatus, error) {
	_, err := s.validateRelationship(
		ctx,
		userID,
		artistID,
		true,
	)
	if err != nil {
		return ArtistFollowStatus{}, err
	}

	if err := s.followRepo.Follow(
		ctx,
		userID,
		artistID,
	); err != nil {
		return ArtistFollowStatus{}, err
	}

	return s.GetStatus(
		ctx,
		userID,
		artistID,
	)
}

func (s *ArtistFollowService) Unfollow(
	ctx context.Context,
	userID int,
	artistID int,
) (ArtistFollowStatus, error) {
	_, err := s.validateRelationship(
		ctx,
		userID,
		artistID,
		false,
	)
	if err != nil {
		return ArtistFollowStatus{}, err
	}

	if err := s.followRepo.Unfollow(
		ctx,
		userID,
		artistID,
	); err != nil {
		return ArtistFollowStatus{}, err
	}

	return s.GetStatus(
		ctx,
		userID,
		artistID,
	)
}

func (s *ArtistFollowService) GetStatus(
	ctx context.Context,
	userID int,
	artistID int,
) (ArtistFollowStatus, error) {
	_, err := s.validateRelationship(
		ctx,
		userID,
		artistID,
		false,
	)
	if err != nil {
		return ArtistFollowStatus{}, err
	}

	isFollowing,
		followerCount,
		err := s.followRepo.GetStatus(
		ctx,
		userID,
		artistID,
	)
	if err != nil {
		return ArtistFollowStatus{}, err
	}

	return ArtistFollowStatus{
		IsFollowing:   isFollowing,
		FollowerCount: followerCount,
	}, nil
}

func (s *ArtistFollowService) GetFollowerCount(
	ctx context.Context,
	artistID int,
) (int64, error) {
	if artistID <= 0 {
		return 0,
			ErrInvalidFollowArtistID
	}

	_, err := s.artistRepo.GetByID(
		ctx,
		artistID,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return 0,
				ErrFollowArtistNotFound
		}

		return 0, err
	}

	return s.followRepo.FollowerCount(
		ctx,
		artistID,
	)
}

func (s *ArtistFollowService) validateRelationship(
	ctx context.Context,
	userID int,
	artistID int,
	disallowSelf bool,
) (models.Artist, error) {
	if userID <= 0 {
		return models.Artist{},
			ErrInvalidFollowUserID
	}

	if artistID <= 0 {
		return models.Artist{},
			ErrInvalidFollowArtistID
	}

	artist, err := s.artistRepo.GetByID(
		ctx,
		artistID,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return models.Artist{},
				ErrFollowArtistNotFound
		}

		return models.Artist{}, err
	}

	if disallowSelf &&
		artist.UserID == userID {
		return models.Artist{},
			ErrCannotFollowOwnArtist
	}

	return artist, nil
}
