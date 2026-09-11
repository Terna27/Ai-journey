package services

import (
	"context"
	"fmt"

	"music-api/internal/models"
)

const (
	DefaultDiscoveryTrackLimit   = 8
	DefaultDiscoveryReleaseLimit = 8
	DefaultDiscoveryArtistLimit  = 8
	DefaultDiscoveryGenreLimit   = 12

	// We deliberately return a larger pool than the number of
	// videos visible at once. The frontend can rotate and
	// randomly select from this ranked candidate set.
	DefaultHeroArtistLimit = 20
)

type DiscoveryRepository interface {
	HeroArtists(
		ctx context.Context,
		limit int,
	) ([]models.DiscoveryHeroArtist, error)

	TrendingTracks(
		ctx context.Context,
		limit int,
	) ([]models.Music, error)

	NewTracks(
		ctx context.Context,
		limit int,
	) ([]models.Music, error)

	NewReleases(
		ctx context.Context,
		limit int,
	) ([]models.SearchRelease, error)

	PopularArtists(
		ctx context.Context,
		limit int,
	) ([]models.SearchArtist, error)

	Genres(
		ctx context.Context,
		limit int,
	) ([]models.DiscoveryGenre, error)
}

type DiscoveryService struct {
	repo DiscoveryRepository
}

func NewDiscoveryService(
	repo DiscoveryRepository,
) *DiscoveryService {
	return &DiscoveryService{
		repo: repo,
	}
}

func (s *DiscoveryService) HeroArtists(
	ctx context.Context,
) (models.DiscoveryHeroResult, error) {
	artists, err :=
		s.repo.HeroArtists(
			ctx,
			DefaultHeroArtistLimit,
		)
	if err != nil {
		return models.DiscoveryHeroResult{},
			fmt.Errorf(
				"load hero artists: %w",
				err,
			)
	}

	if artists == nil {
		artists = make(
			[]models.DiscoveryHeroArtist,
			0,
		)
	}

	return models.DiscoveryHeroResult{
		Artists: artists,
	}, nil
}

func (s *DiscoveryService) Discover(
	ctx context.Context,
) (models.DiscoveryResult, error) {
	result := models.DiscoveryResult{
		TrendingTracks: make(
			[]models.Music,
			0,
		),

		NewTracks: make(
			[]models.Music,
			0,
		),

		NewReleases: make(
			[]models.SearchRelease,
			0,
		),

		PopularArtists: make(
			[]models.SearchArtist,
			0,
		),

		Genres: make(
			[]models.DiscoveryGenre,
			0,
		),
	}

	trendingTracks, err :=
		s.repo.TrendingTracks(
			ctx,
			DefaultDiscoveryTrackLimit,
		)
	if err != nil {
		return models.DiscoveryResult{},
			fmt.Errorf(
				"load trending tracks: %w",
				err,
			)
	}

	result.TrendingTracks =
		trendingTracks

	newTracks, err :=
		s.repo.NewTracks(
			ctx,
			DefaultDiscoveryTrackLimit,
		)
	if err != nil {
		return models.DiscoveryResult{},
			fmt.Errorf(
				"load new tracks: %w",
				err,
			)
	}

	result.NewTracks = newTracks

	newReleases, err :=
		s.repo.NewReleases(
			ctx,
			DefaultDiscoveryReleaseLimit,
		)
	if err != nil {
		return models.DiscoveryResult{},
			fmt.Errorf(
				"load new releases: %w",
				err,
			)
	}

	result.NewReleases =
		newReleases

	popularArtists, err :=
		s.repo.PopularArtists(
			ctx,
			DefaultDiscoveryArtistLimit,
		)
	if err != nil {
		return models.DiscoveryResult{},
			fmt.Errorf(
				"load popular artists: %w",
				err,
			)
	}

	result.PopularArtists =
		popularArtists

	genres, err :=
		s.repo.Genres(
			ctx,
			DefaultDiscoveryGenreLimit,
		)
	if err != nil {
		return models.DiscoveryResult{},
			fmt.Errorf(
				"load genres: %w",
				err,
			)
	}

	result.Genres = genres

	return result, nil
}
