package services

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"
	"unicode"

	"github.com/jackc/pgx/v5"

	"music-api/internal/models"
	"music-api/internal/repository"
)

const (
	maxPodcastTitleRunes       = 200
	maxPodcastDescriptionRunes = 10_000
	maxPodcastCategoryRunes    = 100

	maxPodcastEpisodeTitleRunes       = 200
	maxPodcastEpisodeDescriptionRunes = 20_000

	defaultPodcastPageLimit = 20
	maxPodcastPageLimit     = 100
)

var (
	ErrPodcastNotFound = errors.New(
		"podcast not found",
	)

	ErrPodcastTitleRequired = errors.New(
		"podcast title is required",
	)

	ErrPodcastTitleTooLong = errors.New(
		"podcast title must not exceed 200 characters",
	)

	ErrPodcastDescriptionTooLong = errors.New(
		"podcast description must not exceed 10000 characters",
	)

	ErrPodcastCategoryTooLong = errors.New(
		"podcast category must not exceed 100 characters",
	)

	ErrInvalidPodcastStatus = errors.New(
		"invalid podcast status",
	)

	ErrPodcastArtworkIncomplete = errors.New(
		"podcast artwork asset is incomplete",
	)

	ErrPodcastEpisodeNotFound = errors.New(
		"podcast episode not found",
	)

	ErrPodcastEpisodeTitleRequired = errors.New(
		"podcast episode title is required",
	)

	ErrPodcastEpisodeTitleTooLong = errors.New(
		"podcast episode title must not exceed 200 characters",
	)

	ErrPodcastEpisodeDescriptionTooLong = errors.New(
		"podcast episode description must not exceed 20000 characters",
	)

	ErrInvalidPodcastEpisodeType = errors.New(
		"invalid podcast episode type",
	)

	ErrInvalidPodcastEpisodeStatus = errors.New(
		"invalid podcast episode status",
	)

	ErrInvalidPodcastSeasonNumber = errors.New(
		"season number must be greater than zero",
	)

	ErrInvalidPodcastEpisodeNumber = errors.New(
		"episode number must be greater than zero",
	)

	ErrInvalidPodcastDuration = errors.New(
		"episode duration must not be negative",
	)

	ErrPodcastEpisodeAudioIncomplete = errors.New(
		"podcast episode audio asset is incomplete",
	)

	ErrPodcastEpisodeArtworkIncomplete = errors.New(
		"podcast episode artwork asset is incomplete",
	)

	ErrPodcastEpisodeAudioRequired = errors.New(
		"podcast episode audio is required before publishing",
	)

	ErrPodcastPublishRequiresArtwork = errors.New(
		"podcast artwork is required before publishing",
	)

	ErrPodcastScheduleRequired = errors.New(
		"scheduled_at is required for a scheduled episode",
	)

	ErrPodcastScheduleMustBeFuture = errors.New(
		"scheduled_at must be in the future",
	)

	ErrPodcastScheduleNotAllowed = errors.New(
		"scheduled_at is only allowed for scheduled episodes",
	)

	ErrPodcastEpisodeNumberExists = errors.New(
		"an episode with this season and episode number already exists",
	)

	ErrPodcastSlugConflict = errors.New(
		"unable to allocate a unique podcast slug",
	)

	ErrPodcastEpisodeSlugConflict = errors.New(
		"unable to allocate a unique episode slug",
	)
)

type PodcastMediaAsset struct {
	URL      string
	PublicID string
}

type CreatePodcastInput struct {
	Title       string
	Description string
	Category    string

	Artwork *PodcastMediaAsset

	IsExplicit bool
}

type UpdatePodcastInput struct {
	Title       string
	Description string
	Category    string

	Artwork *PodcastMediaAsset

	Status models.PodcastStatus

	IsExplicit bool
}

type CreatePodcastEpisodeInput struct {
	Title       string
	Description string

	SeasonNumber  int
	EpisodeNumber *int

	EpisodeType models.PodcastEpisodeType

	Audio *PodcastMediaAsset

	Artwork *PodcastMediaAsset

	DurationMS int64

	Status models.PodcastEpisodeStatus

	IsExplicit bool

	ScheduledAt *time.Time
}

type UpdatePodcastEpisodeInput struct {
	Title       string
	Description string

	SeasonNumber  int
	EpisodeNumber *int

	EpisodeType models.PodcastEpisodeType

	Audio *PodcastMediaAsset

	Artwork *PodcastMediaAsset

	DurationMS int64

	Status models.PodcastEpisodeStatus

	IsExplicit bool

	ScheduledAt *time.Time
}

type PodcastService struct {
	repo *repository.PodcastRepository
}

func NewPodcastService(
	repo *repository.PodcastRepository,
) *PodcastService {
	return &PodcastService{
		repo: repo,
	}
}

func (s *PodcastService) CreatePodcast(
	ctx context.Context,
	ownerUserID int,
	in CreatePodcastInput,
) (models.Podcast, error) {
	if ownerUserID <= 0 {
		return models.Podcast{}, ErrUnauthorized
	}

	title, description, category, err :=
		validatePodcastText(
			in.Title,
			in.Description,
			in.Category,
		)
	if err != nil {
		return models.Podcast{}, err
	}

	podcast := models.Podcast{
		OwnerUserID: ownerUserID,

		Title:       title,
		Description: description,
		Category:    category,

		Status: models.PodcastStatusDraft,

		IsExplicit: in.IsExplicit,
	}

	if in.Artwork != nil {
		if err := validatePodcastMediaAsset(
			*in.Artwork,
			ErrPodcastArtworkIncomplete,
		); err != nil {
			return models.Podcast{}, err
		}

		podcast.ArtworkURL =
			strings.TrimSpace(in.Artwork.URL)

		podcast.ArtworkPublicID =
			strings.TrimSpace(in.Artwork.PublicID)
	}

	baseSlug := slugify(title)

	for attempt := 0; attempt < 20; attempt++ {
		podcast.Slug =
			slugForAttempt(
				baseSlug,
				attempt,
			)

		created, err :=
			s.repo.CreatePodcast(
				ctx,
				podcast,
			)

		if err == nil {
			return created, nil
		}

		if errors.Is(
			err,
			repository.ErrPodcastSlugExists,
		) {
			continue
		}

		return models.Podcast{}, err
	}

	return models.Podcast{},
		ErrPodcastSlugConflict
}

func (s *PodcastService) GetPublicPodcast(
	ctx context.Context,
	id int64,
) (models.PodcastDetails, error) {
	if id <= 0 {
		return models.PodcastDetails{},
			ErrPodcastNotFound
	}

	podcast, err :=
		s.repo.GetPublishedPodcastByID(
			ctx,
			id,
		)
	if err != nil {
		return models.PodcastDetails{},
			mapPodcastNotFound(
				err,
				ErrPodcastNotFound,
			)
	}

	episodes, err :=
		s.repo.GetPublishedEpisodesByPodcastID(
			ctx,
			id,
			maxPodcastPageLimit,
			0,
		)
	if err != nil {
		return models.PodcastDetails{}, err
	}

	return models.PodcastDetails{
		Podcast:  podcast,
		Episodes: episodes,
	}, nil
}

func (s *PodcastService) GetPublicPodcastBySlug(
	ctx context.Context,
	slug string,
) (models.PodcastDetails, error) {
	slug = strings.TrimSpace(slug)

	if slug == "" {
		return models.PodcastDetails{},
			ErrPodcastNotFound
	}

	podcast, err :=
		s.repo.GetPublishedPodcastBySlug(
			ctx,
			slug,
		)
	if err != nil {
		return models.PodcastDetails{},
			mapPodcastNotFound(
				err,
				ErrPodcastNotFound,
			)
	}

	episodes, err :=
		s.repo.GetPublishedEpisodesByPodcastID(
			ctx,
			podcast.ID,
			maxPodcastPageLimit,
			0,
		)
	if err != nil {
		return models.PodcastDetails{}, err
	}

	return models.PodcastDetails{
		Podcast:  podcast,
		Episodes: episodes,
	}, nil
}

func (s *PodcastService) GetOwnedPodcast(
	ctx context.Context,
	id int64,
	ownerUserID int,
) (models.PodcastDetails, error) {
	if ownerUserID <= 0 {
		return models.PodcastDetails{},
			ErrUnauthorized
	}

	if id <= 0 {
		return models.PodcastDetails{},
			ErrPodcastNotFound
	}

	podcast, err :=
		s.repo.GetOwnedPodcastByID(
			ctx,
			id,
			ownerUserID,
		)
	if err != nil {
		return models.PodcastDetails{},
			mapPodcastNotFound(
				err,
				ErrPodcastNotFound,
			)
	}

	episodes, err :=
		s.repo.GetEpisodesByPodcastID(
			ctx,
			id,
		)
	if err != nil {
		return models.PodcastDetails{}, err
	}

	return models.PodcastDetails{
		Podcast:  podcast,
		Episodes: episodes,
	}, nil
}

func (s *PodcastService) GetMyPodcasts(
	ctx context.Context,
	ownerUserID int,
) ([]models.Podcast, error) {
	if ownerUserID <= 0 {
		return nil, ErrUnauthorized
	}

	return s.repo.GetPodcastsByOwnerUserID(
		ctx,
		ownerUserID,
	)
}

func (s *PodcastService) GetPublishedPodcasts(
	ctx context.Context,
	limit int,
	offset int,
) ([]models.Podcast, error) {
	limit, offset =
		normalizePodcastPagination(
			limit,
			offset,
		)

	return s.repo.GetPublishedPodcasts(
		ctx,
		limit,
		offset,
	)
}

func (s *PodcastService) UpdatePodcast(
	ctx context.Context,
	id int64,
	ownerUserID int,
	in UpdatePodcastInput,
) (models.Podcast, error) {
	if ownerUserID <= 0 {
		return models.Podcast{}, ErrUnauthorized
	}

	if id <= 0 {
		return models.Podcast{},
			ErrPodcastNotFound
	}

	existing, err :=
		s.repo.GetOwnedPodcastByID(
			ctx,
			id,
			ownerUserID,
		)
	if err != nil {
		return models.Podcast{},
			mapPodcastNotFound(
				err,
				ErrPodcastNotFound,
			)
	}

	title, description, category, err :=
		validatePodcastText(
			in.Title,
			in.Description,
			in.Category,
		)
	if err != nil {
		return models.Podcast{}, err
	}

	if !validPodcastStatus(in.Status) {
		return models.Podcast{},
			ErrInvalidPodcastStatus
	}

	existing.Title = title
	existing.Description = description
	existing.Category = category
	existing.IsExplicit = in.IsExplicit

	if in.Artwork != nil {
		if err := validatePodcastMediaAsset(
			*in.Artwork,
			ErrPodcastArtworkIncomplete,
		); err != nil {
			return models.Podcast{}, err
		}

		existing.ArtworkURL =
			strings.TrimSpace(in.Artwork.URL)

		existing.ArtworkPublicID =
			strings.TrimSpace(in.Artwork.PublicID)
	}

	switch in.Status {
	case models.PodcastStatusPublished:
		if strings.TrimSpace(
			existing.ArtworkURL,
		) == "" ||
			strings.TrimSpace(
				existing.ArtworkPublicID,
			) == "" {
			return models.Podcast{},
				ErrPodcastPublishRequiresArtwork
		}

		if existing.PublishedAt == nil {
			now := time.Now().UTC()
			existing.PublishedAt = &now
		}

	case models.PodcastStatusDraft:
		// A podcast returned to draft is no longer publicly
		// published. Clear published_at so a later publication
		// receives a new publication timestamp.
		existing.PublishedAt = nil

	case models.PodcastStatusArchived:
		// Preserve the historical publication timestamp.
	}

	existing.Status = in.Status

	// Slugs are stable identifiers. Renaming a podcast must not
	// silently break existing public URLs or shared links.
	updated, err :=
		s.repo.UpdatePodcast(
			ctx,
			existing,
		)

	if err != nil {
		if errors.Is(
			err,
			pgx.ErrNoRows,
		) {
			return models.Podcast{},
				ErrPodcastNotFound
		}

		return models.Podcast{}, err
	}

	return updated, nil
}

func (s *PodcastService) DeletePodcast(
	ctx context.Context,
	id int64,
	ownerUserID int,
) (models.PodcastDetails, error) {
	if ownerUserID <= 0 {
		return models.PodcastDetails{},
			ErrUnauthorized
	}

	details, err :=
		s.GetOwnedPodcast(
			ctx,
			id,
			ownerUserID,
		)
	if err != nil {
		return models.PodcastDetails{}, err
	}

	err = s.repo.DeletePodcast(
		ctx,
		id,
		ownerUserID,
	)
	if err != nil {
		return models.PodcastDetails{},
			mapPodcastNotFound(
				err,
				ErrPodcastNotFound,
			)
	}

	// Return the pre-delete asset metadata so the handler can
	// clean up Cloudinary only after the database delete has
	// succeeded.
	return details, nil
}

func (s *PodcastService) CreateEpisode(
	ctx context.Context,
	podcastID int64,
	ownerUserID int,
	in CreatePodcastEpisodeInput,
) (models.PodcastEpisode, error) {
	if ownerUserID <= 0 {
		return models.PodcastEpisode{},
			ErrUnauthorized
	}

	if podcastID <= 0 {
		return models.PodcastEpisode{},
			ErrPodcastNotFound
	}

	_, err :=
		s.repo.GetOwnedPodcastByID(
			ctx,
			podcastID,
			ownerUserID,
		)
	if err != nil {
		return models.PodcastEpisode{},
			mapPodcastNotFound(
				err,
				ErrPodcastNotFound,
			)
	}

	episode, err :=
		buildPodcastEpisode(
			podcastID,
			in.Title,
			in.Description,
			in.SeasonNumber,
			in.EpisodeNumber,
			in.EpisodeType,
			in.Audio,
			in.Artwork,
			in.DurationMS,
			in.Status,
			in.IsExplicit,
			in.ScheduledAt,
		)
	if err != nil {
		return models.PodcastEpisode{}, err
	}

	baseSlug := slugify(episode.Title)

	for attempt := 0; attempt < 20; attempt++ {
		episode.Slug =
			slugForAttempt(
				baseSlug,
				attempt,
			)

		created, err :=
			s.repo.CreateEpisode(
				ctx,
				episode,
			)

		if err == nil {
			return created, nil
		}

		switch {
		case errors.Is(
			err,
			repository.ErrPodcastEpisodeSlugExists,
		):
			continue

		case errors.Is(
			err,
			repository.ErrPodcastEpisodeNumberExists,
		):
			return models.PodcastEpisode{},
				ErrPodcastEpisodeNumberExists

		default:
			return models.PodcastEpisode{}, err
		}
	}

	return models.PodcastEpisode{},
		ErrPodcastEpisodeSlugConflict
}

func (s *PodcastService) GetPublicEpisode(
	ctx context.Context,
	id int64,
) (models.PodcastEpisode, error) {
	if id <= 0 {
		return models.PodcastEpisode{},
			ErrPodcastEpisodeNotFound
	}

	episode, err :=
		s.repo.GetPublishedEpisodeByID(
			ctx,
			id,
		)
	if err != nil {
		return models.PodcastEpisode{},
			mapPodcastNotFound(
				err,
				ErrPodcastEpisodeNotFound,
			)
	}

	// A published episode must not become publicly reachable
	// through an unpublished/archived podcast.
	podcast, err :=
		s.repo.GetPublishedPodcastByID(
			ctx,
			episode.PodcastID,
		)
	if err != nil {
		return models.PodcastEpisode{},
			mapPodcastNotFound(
				err,
				ErrPodcastEpisodeNotFound,
			)
	}

	_ = podcast

	return episode, nil
}

func (s *PodcastService) GetOwnedEpisode(
	ctx context.Context,
	id int64,
	ownerUserID int,
) (models.PodcastEpisode, error) {
	if ownerUserID <= 0 {
		return models.PodcastEpisode{},
			ErrUnauthorized
	}

	if id <= 0 {
		return models.PodcastEpisode{},
			ErrPodcastEpisodeNotFound
	}

	episode, err :=
		s.repo.GetOwnedEpisodeByID(
			ctx,
			id,
			ownerUserID,
		)
	if err != nil {
		return models.PodcastEpisode{},
			mapPodcastNotFound(
				err,
				ErrPodcastEpisodeNotFound,
			)
	}

	return episode, nil
}

func (s *PodcastService) UpdateEpisode(
	ctx context.Context,
	id int64,
	ownerUserID int,
	in UpdatePodcastEpisodeInput,
) (models.PodcastEpisode, error) {
	if ownerUserID <= 0 {
		return models.PodcastEpisode{},
			ErrUnauthorized
	}

	existing, err :=
		s.GetOwnedEpisode(
			ctx,
			id,
			ownerUserID,
		)
	if err != nil {
		return models.PodcastEpisode{}, err
	}

	// Media updates are patch-like. Resolve omitted media to
	// the existing Cloudinary assets before validation so a
	// metadata-only PATCH can still publish an episode that
	// already has uploaded audio/artwork.
	audio := in.Audio
	if audio == nil &&
		existing.AudioURL != "" &&
		existing.AudioPublicID != "" {
		audio = &PodcastMediaAsset{
			URL:      existing.AudioURL,
			PublicID: existing.AudioPublicID,
		}
	}

	artwork := in.Artwork
	if artwork == nil &&
		existing.ArtworkURL != "" &&
		existing.ArtworkPublicID != "" {
		artwork = &PodcastMediaAsset{
			URL:      existing.ArtworkURL,
			PublicID: existing.ArtworkPublicID,
		}
	}

	updated, err :=
		buildPodcastEpisode(
			existing.PodcastID,
			in.Title,
			in.Description,
			in.SeasonNumber,
			in.EpisodeNumber,
			in.EpisodeType,
			audio,
			artwork,
			in.DurationMS,
			in.Status,
			in.IsExplicit,
			in.ScheduledAt,
		)
	if err != nil {
		return models.PodcastEpisode{}, err
	}

	updated.ID = existing.ID
	updated.CreatedAt = existing.CreatedAt

	if err := applyEpisodeLifecycle(
		&updated,
		existing,
		in.Status,
		in.ScheduledAt,
	); err != nil {
		return models.PodcastEpisode{}, err
	}

	// Episode slugs are also stable identifiers. A title edit
	// must not invalidate an existing episode URL.
	updated.Slug = existing.Slug

	result, err :=
		s.repo.UpdateEpisode(
			ctx,
			updated,
			ownerUserID,
		)

	if err != nil {
		switch {
		case errors.Is(
			err,
			repository.ErrPodcastEpisodeNumberExists,
		):
			return models.PodcastEpisode{},
				ErrPodcastEpisodeNumberExists

		case errors.Is(
			err,
			pgx.ErrNoRows,
		):
			return models.PodcastEpisode{},
				ErrPodcastEpisodeNotFound

		default:
			return models.PodcastEpisode{}, err
		}
	}

	return result, nil
}

func (s *PodcastService) DeleteEpisode(
	ctx context.Context,
	id int64,
	ownerUserID int,
) (models.PodcastEpisode, error) {
	episode, err :=
		s.GetOwnedEpisode(
			ctx,
			id,
			ownerUserID,
		)
	if err != nil {
		return models.PodcastEpisode{}, err
	}

	err = s.repo.DeleteEpisode(
		ctx,
		id,
		ownerUserID,
	)
	if err != nil {
		return models.PodcastEpisode{},
			mapPodcastNotFound(
				err,
				ErrPodcastEpisodeNotFound,
			)
	}

	// Return deleted asset metadata so Cloudinary cleanup can
	// occur only after the database delete succeeds.
	return episode, nil
}

func buildPodcastEpisode(
	podcastID int64,
	titleValue string,
	descriptionValue string,
	seasonNumber int,
	episodeNumber *int,
	episodeType models.PodcastEpisodeType,
	audio *PodcastMediaAsset,
	artwork *PodcastMediaAsset,
	durationMS int64,
	status models.PodcastEpisodeStatus,
	isExplicit bool,
	scheduledAt *time.Time,
) (models.PodcastEpisode, error) {
	title := strings.TrimSpace(titleValue)

	if title == "" {
		return models.PodcastEpisode{},
			ErrPodcastEpisodeTitleRequired
	}

	if len([]rune(title)) >
		maxPodcastEpisodeTitleRunes {
		return models.PodcastEpisode{},
			ErrPodcastEpisodeTitleTooLong
	}

	description :=
		strings.TrimSpace(
			descriptionValue,
		)

	if len([]rune(description)) >
		maxPodcastEpisodeDescriptionRunes {
		return models.PodcastEpisode{},
			ErrPodcastEpisodeDescriptionTooLong
	}

	if seasonNumber <= 0 {
		return models.PodcastEpisode{},
			ErrInvalidPodcastSeasonNumber
	}

	if episodeNumber != nil &&
		*episodeNumber <= 0 {
		return models.PodcastEpisode{},
			ErrInvalidPodcastEpisodeNumber
	}

	if !validPodcastEpisodeType(
		episodeType,
	) {
		return models.PodcastEpisode{},
			ErrInvalidPodcastEpisodeType
	}

	if !validPodcastEpisodeStatus(
		status,
	) {
		return models.PodcastEpisode{},
			ErrInvalidPodcastEpisodeStatus
	}

	if durationMS < 0 {
		return models.PodcastEpisode{},
			ErrInvalidPodcastDuration
	}

	episode := models.PodcastEpisode{
		PodcastID: podcastID,

		Title:       title,
		Description: description,

		SeasonNumber:  seasonNumber,
		EpisodeNumber: episodeNumber,

		EpisodeType: episodeType,

		DurationMS: durationMS,

		Status: status,

		IsExplicit: isExplicit,
	}

	if audio != nil {
		if err := validatePodcastMediaAsset(
			*audio,
			ErrPodcastEpisodeAudioIncomplete,
		); err != nil {
			return models.PodcastEpisode{}, err
		}

		episode.AudioURL =
			strings.TrimSpace(audio.URL)

		episode.AudioPublicID =
			strings.TrimSpace(audio.PublicID)
	}

	if artwork != nil {
		if err := validatePodcastMediaAsset(
			*artwork,
			ErrPodcastEpisodeArtworkIncomplete,
		); err != nil {
			return models.PodcastEpisode{}, err
		}

		episode.ArtworkURL =
			strings.TrimSpace(artwork.URL)

		episode.ArtworkPublicID =
			strings.TrimSpace(artwork.PublicID)
	}

	if err := applyEpisodeLifecycle(
		&episode,
		models.PodcastEpisode{},
		status,
		scheduledAt,
	); err != nil {
		return models.PodcastEpisode{}, err
	}

	return episode, nil
}

func applyEpisodeLifecycle(
	episode *models.PodcastEpisode,
	existing models.PodcastEpisode,
	status models.PodcastEpisodeStatus,
	scheduledAt *time.Time,
) error {
	now := time.Now().UTC()

	switch status {
	case models.PodcastEpisodeStatusDraft:
		episode.ScheduledAt = nil
		episode.PublishedAt = nil

	case models.PodcastEpisodeStatusScheduled:
		if scheduledAt == nil {
			return ErrPodcastScheduleRequired
		}

		scheduled :=
			scheduledAt.UTC()

		if !scheduled.After(now) {
			return ErrPodcastScheduleMustBeFuture
		}

		if !podcastEpisodeHasAudio(*episode) {
			return ErrPodcastEpisodeAudioRequired
		}

		episode.ScheduledAt = &scheduled
		episode.PublishedAt = nil

	case models.PodcastEpisodeStatusPublished:
		if scheduledAt != nil {
			return ErrPodcastScheduleNotAllowed
		}

		if !podcastEpisodeHasAudio(*episode) {
			return ErrPodcastEpisodeAudioRequired
		}

		episode.ScheduledAt = nil

		if existing.PublishedAt != nil {
			episode.PublishedAt =
				existing.PublishedAt
		} else {
			episode.PublishedAt = &now
		}

	case models.PodcastEpisodeStatusLive:
		if scheduledAt != nil {
			return ErrPodcastScheduleNotAllowed
		}

		episode.ScheduledAt = nil

		if existing.PublishedAt != nil {
			episode.PublishedAt =
				existing.PublishedAt
		}

	case models.PodcastEpisodeStatusEnded:
		if scheduledAt != nil {
			return ErrPodcastScheduleNotAllowed
		}

		episode.ScheduledAt = nil

		if existing.PublishedAt != nil {
			episode.PublishedAt =
				existing.PublishedAt
		}

	case models.PodcastEpisodeStatusArchived:
		if scheduledAt != nil {
			return ErrPodcastScheduleNotAllowed
		}

		episode.ScheduledAt = nil
		episode.PublishedAt =
			existing.PublishedAt

	default:
		return ErrInvalidPodcastEpisodeStatus
	}

	return nil
}

func podcastEpisodeHasAudio(
	episode models.PodcastEpisode,
) bool {
	return strings.TrimSpace(
		episode.AudioURL,
	) != "" &&
		strings.TrimSpace(
			episode.AudioPublicID,
		) != ""
}

func validatePodcastText(
	titleValue string,
	descriptionValue string,
	categoryValue string,
) (string, string, string, error) {
	title :=
		strings.TrimSpace(
			titleValue,
		)

	if title == "" {
		return "", "", "",
			ErrPodcastTitleRequired
	}

	if len([]rune(title)) >
		maxPodcastTitleRunes {
		return "", "", "",
			ErrPodcastTitleTooLong
	}

	description :=
		strings.TrimSpace(
			descriptionValue,
		)

	if len([]rune(description)) >
		maxPodcastDescriptionRunes {
		return "", "", "",
			ErrPodcastDescriptionTooLong
	}

	category :=
		strings.TrimSpace(
			categoryValue,
		)

	if len([]rune(category)) >
		maxPodcastCategoryRunes {
		return "", "", "",
			ErrPodcastCategoryTooLong
	}

	return title,
		description,
		category,
		nil
}

func validatePodcastMediaAsset(
	asset PodcastMediaAsset,
	incompleteError error,
) error {
	if strings.TrimSpace(asset.URL) == "" ||
		strings.TrimSpace(asset.PublicID) == "" {
		return incompleteError
	}

	return nil
}

func validPodcastStatus(
	status models.PodcastStatus,
) bool {
	switch status {
	case models.PodcastStatusDraft,
		models.PodcastStatusPublished,
		models.PodcastStatusArchived:
		return true

	default:
		return false
	}
}

func validPodcastEpisodeType(
	episodeType models.PodcastEpisodeType,
) bool {
	switch episodeType {
	case models.PodcastEpisodeTypeFull,
		models.PodcastEpisodeTypeTrailer,
		models.PodcastEpisodeTypeBonus:
		return true

	default:
		return false
	}
}

func validPodcastEpisodeStatus(
	status models.PodcastEpisodeStatus,
) bool {
	switch status {
	case models.PodcastEpisodeStatusDraft,
		models.PodcastEpisodeStatusScheduled,
		models.PodcastEpisodeStatusPublished,
		models.PodcastEpisodeStatusLive,
		models.PodcastEpisodeStatusEnded,
		models.PodcastEpisodeStatusArchived:
		return true

	default:
		return false
	}
}

var slugSeparatorPattern = regexp.MustCompile(`-+`)

func slugify(
	value string,
) string {
	value = strings.TrimSpace(
		strings.ToLower(value),
	)

	var builder strings.Builder
	previousSeparator := false

	for _, char := range value {
		switch {
		case unicode.IsLetter(char) ||
			unicode.IsDigit(char):
			builder.WriteRune(char)
			previousSeparator = false

		default:
			if builder.Len() > 0 &&
				!previousSeparator {
				builder.WriteByte('-')
				previousSeparator = true
			}
		}
	}

	slug :=
		strings.Trim(
			builder.String(),
			"-",
		)

	slug =
		slugSeparatorPattern.ReplaceAllString(
			slug,
			"-",
		)

	if slug == "" {
		return "podcast"
	}

	return slug
}

func slugForAttempt(
	base string,
	attempt int,
) string {
	if attempt <= 0 {
		return base
	}

	return fmt.Sprintf(
		"%s-%d",
		base,
		attempt+1,
	)
}

func normalizePodcastPagination(
	limit int,
	offset int,
) (int, int) {
	if limit <= 0 {
		limit = defaultPodcastPageLimit
	}

	if limit > maxPodcastPageLimit {
		limit = maxPodcastPageLimit
	}

	if offset < 0 {
		offset = 0
	}

	return limit, offset
}

func mapPodcastNotFound(
	err error,
	notFound error,
) error {
	if errors.Is(
		err,
		pgx.ErrNoRows,
	) {
		return notFound
	}

	return err
}
