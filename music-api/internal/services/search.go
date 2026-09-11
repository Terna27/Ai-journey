package services

import (
	"context"
	"errors"
	"strings"

	"music-api/internal/models"
	"music-api/internal/repository"
)

var (
	ErrSearchQueryRequired = errors.New(
		"search query is required",
	)

	ErrSearchQueryTooShort = errors.New(
		"search query must contain at least 2 characters",
	)

	ErrSearchQueryTooLong = errors.New(
		"search query must not exceed 100 characters",
	)

	ErrInvalidSearchType = errors.New(
		"search type must be all, track, artist, or release",
	)

	ErrInvalidSearchSort = errors.New(
		"search sort must be relevance, newest, or popular",
	)

	ErrInvalidSearchPage = errors.New(
		"search page must be a positive integer",
	)

	ErrInvalidSearchLimit = errors.New(
		"search limit must be between 1 and 50",
	)

	ErrSearchGenreTooLong = errors.New(
		"search genre must not exceed 50 characters",
	)
)

const (
	DefaultSearchType = "all"

	SearchTypeAll = "all"

	SearchTypeTrack = "track"

	SearchTypeArtist = "artist"

	SearchTypeRelease = "release"

	DefaultSearchSort = "relevance"

	SearchSortRelevance = "relevance"

	SearchSortNewest = "newest"

	SearchSortPopular = "popular"

	DefaultSearchPage = 1

	DefaultSearchLimit = 10

	MaxSearchLimit = 50
)

type SearchOptions struct {
	Query string

	Type string

	Sort string

	Genre string

	Page int

	Limit int
}

type SearchService struct {
	Repository *repository.SearchRepository
}

func NewSearchService(
	repository *repository.SearchRepository,
) *SearchService {
	return &SearchService{
		Repository: repository,
	}
}

func (s *SearchService) Search(
	ctx context.Context,
	options SearchOptions,
) (models.SearchResult, error) {
	options.Query =
		strings.TrimSpace(
			options.Query,
		)

	options.Type =
		strings.ToLower(
			strings.TrimSpace(
				options.Type,
			),
		)

	options.Sort =
		strings.ToLower(
			strings.TrimSpace(
				options.Sort,
			),
		)

	options.Genre =
		strings.TrimSpace(
			options.Genre,
		)

	if options.Query == "" {
		return models.SearchResult{},
			ErrSearchQueryRequired
	}

	queryLength :=
		len(
			[]rune(
				options.Query,
			),
		)

	if queryLength < 2 {
		return models.SearchResult{},
			ErrSearchQueryTooShort
	}

	if queryLength > 100 {
		return models.SearchResult{},
			ErrSearchQueryTooLong
	}

	if options.Type == "" {
		options.Type =
			DefaultSearchType
	}

	switch options.Type {
	case SearchTypeAll,
		SearchTypeTrack,
		SearchTypeArtist,
		SearchTypeRelease:

	default:
		return models.SearchResult{},
			ErrInvalidSearchType
	}

	if options.Sort == "" {
		options.Sort =
			DefaultSearchSort
	}

	switch options.Sort {
	case SearchSortRelevance,
		SearchSortNewest,
		SearchSortPopular:

	default:
		return models.SearchResult{},
			ErrInvalidSearchSort
	}

	if len(
		[]rune(
			options.Genre,
		),
	) > 50 {
		return models.SearchResult{},
			ErrSearchGenreTooLong
	}

	if options.Page <= 0 {
		return models.SearchResult{},
			ErrInvalidSearchPage
	}

	if options.Limit <= 0 ||
		options.Limit > MaxSearchLimit {

		return models.SearchResult{},
			ErrInvalidSearchLimit
	}

	offset :=
		(options.Page - 1) *
			options.Limit

	tracks :=
		make(
			[]models.Music,
			0,
		)

	artists :=
		make(
			[]models.SearchArtist,
			0,
		)

	releases :=
		make(
			[]models.SearchRelease,
			0,
		)

	var trackTotal int

	var artistTotal int

	var releaseTotal int

	// =========================
	// TRACK SEARCH
	// =========================

	if options.Type == SearchTypeAll ||
		options.Type == SearchTypeTrack {

		var err error

		trackTotal, err =
			s.Repository.CountTracks(
				ctx,
				options.Query,
				options.Genre,
			)
		if err != nil {
			return models.SearchResult{}, err
		}

		tracks, err =
			s.Repository.SearchTracks(
				ctx,
				options.Query,
				options.Genre,
				options.Sort,
				options.Limit,
				offset,
			)
		if err != nil {
			return models.SearchResult{}, err
		}
	}

	// =========================
	// ARTIST SEARCH
	// =========================

	if options.Type == SearchTypeAll ||
		options.Type == SearchTypeArtist {

		var err error

		artistTotal, err =
			s.Repository.CountArtists(
				ctx,
				options.Query,
			)
		if err != nil {
			return models.SearchResult{}, err
		}

		artists, err =
			s.Repository.SearchArtists(
				ctx,
				options.Query,
				options.Sort,
				options.Limit,
				offset,
			)
		if err != nil {
			return models.SearchResult{}, err
		}
	}

	// =========================
	// RELEASE SEARCH
	// =========================

	if options.Type == SearchTypeAll ||
		options.Type == SearchTypeRelease {

		var err error

		releaseTotal, err =
			s.Repository.CountReleases(
				ctx,
				options.Query,
			)
		if err != nil {
			return models.SearchResult{}, err
		}

		releases, err =
			s.Repository.SearchReleases(
				ctx,
				options.Query,
				options.Sort,
				options.Limit,
				offset,
			)
		if err != nil {
			return models.SearchResult{}, err
		}
	}

	return models.SearchResult{
		Query: options.Query,

		Type: options.Type,

		Sort: options.Sort,

		Genre: options.Genre,

		Tracks: tracks,

		Artists: artists,

		Releases: releases,

		Pagination: models.SearchPagination{
			Page: options.Page,

			Limit: options.Limit,

			TrackTotal: trackTotal,

			ArtistTotal: artistTotal,

			ReleaseTotal: releaseTotal,

			TrackHasMore: offset+
				len(tracks) <
				trackTotal,

			ArtistHasMore: offset+
				len(artists) <
				artistTotal,

			ReleaseHasMore: offset+
				len(releases) <
				releaseTotal,
		},
	}, nil
}
