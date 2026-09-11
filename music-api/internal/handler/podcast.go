package handler

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strconv"
	"strings"

	"music-api/internal/middleware"
	"music-api/internal/models"
	"music-api/internal/repository"
	"music-api/internal/services"
)

type PodcastHandler struct {
	Service *services.PodcastService

	CloudinaryService *services.CloudinaryService
}

func NewPodcastHandler(
	service *services.PodcastService,
	cloudinaryService *services.CloudinaryService,
) *PodcastHandler {
	return &PodcastHandler{
		Service: service,

		CloudinaryService: cloudinaryService,
	}
}

type CreatePodcastRequest struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	Category    string `json:"category"`
	IsExplicit  bool   `json:"is_explicit"`
}

type UpdatePodcastRequest struct {
	Title       *string               `json:"title"`
	Description *string               `json:"description"`
	Category    *string               `json:"category"`
	Status      *models.PodcastStatus `json:"status"`
	IsExplicit  *bool                 `json:"is_explicit"`
}

type CreatePodcastEpisodeRequest struct {
	Title       string `json:"title"`
	Description string `json:"description"`

	SeasonNumber  int  `json:"season_number"`
	EpisodeNumber *int `json:"episode_number"`

	EpisodeType models.PodcastEpisodeType `json:"episode_type"`

	DurationMS int64 `json:"duration_ms"`

	Status models.PodcastEpisodeStatus `json:"status"`

	IsExplicit bool `json:"is_explicit"`
}

type UpdatePodcastEpisodeRequest struct {
	Title       *string `json:"title"`
	Description *string `json:"description"`

	SeasonNumber  *int `json:"season_number"`
	EpisodeNumber *int `json:"episode_number"`

	EpisodeType *models.PodcastEpisodeType `json:"episode_type"`

	DurationMS *int64 `json:"duration_ms"`

	Status *models.PodcastEpisodeStatus `json:"status"`

	IsExplicit *bool `json:"is_explicit"`
}

func (h *PodcastHandler) CreatePodcast(
	w http.ResponseWriter,
	r *http.Request,
) {
	userID, ok :=
		middleware.UserIDFromContext(
			r.Context(),
		)

	if !ok {
		WriteError(
			w,
			http.StatusUnauthorized,
			"AUTHENTICATION_REQUIRED",
			"user authentication is required",
		)
		return
	}

	var req CreatePodcastRequest

	if err := decodePodcastJSON(
		r,
		&req,
	); err != nil {
		WriteError(
			w,
			http.StatusBadRequest,
			"INVALID_REQUEST",
			"invalid request body",
		)
		return
	}

	podcast, err :=
		h.Service.CreatePodcast(
			r.Context(),
			userID,
			services.CreatePodcastInput{
				Title: req.Title,

				Description: req.Description,

				Category: req.Category,

				IsExplicit: req.IsExplicit,
			},
		)

	if err != nil {
		writePodcastServiceError(
			w,
			err,
			"failed to create podcast",
		)
		return
	}

	WriteJSON(
		w,
		http.StatusCreated,
		podcast,
	)
}

func (h *PodcastHandler) GetPublishedPodcasts(
	w http.ResponseWriter,
	r *http.Request,
) {
	limit :=
		parsePodcastQueryInt(
			r.URL.Query().Get("limit"),
			20,
		)

	offset :=
		parsePodcastQueryInt(
			r.URL.Query().Get("offset"),
			0,
		)

	podcasts, err :=
		h.Service.GetPublishedPodcasts(
			r.Context(),
			limit,
			offset,
		)

	if err != nil {
		log.Printf(
			"GetPublishedPodcasts: %v",
			err,
		)

		WriteError(
			w,
			http.StatusInternalServerError,
			"INTERNAL_ERROR",
			"failed to load podcasts",
		)
		return
	}

	WriteJSON(
		w,
		http.StatusOK,
		podcasts,
	)
}

func (h *PodcastHandler) GetPublicPodcast(
	w http.ResponseWriter,
	r *http.Request,
) {
	podcastID, ok :=
		parsePositiveInt64PathID(
			r.PathValue("id"),
		)

	if !ok {
		WriteError(
			w,
			http.StatusBadRequest,
			"INVALID_PODCAST_ID",
			"invalid podcast ID",
		)
		return
	}

	podcast, err :=
		h.Service.GetPublicPodcast(
			r.Context(),
			podcastID,
		)

	if err != nil {
		writePodcastServiceError(
			w,
			err,
			"failed to load podcast",
		)
		return
	}

	WriteJSON(
		w,
		http.StatusOK,
		podcast,
	)
}

func (h *PodcastHandler) GetPublicPodcastBySlug(
	w http.ResponseWriter,
	r *http.Request,
) {
	slug :=
		strings.TrimSpace(
			r.PathValue("slug"),
		)

	if slug == "" {
		WriteError(
			w,
			http.StatusBadRequest,
			"INVALID_PODCAST_SLUG",
			"invalid podcast slug",
		)
		return
	}

	podcast, err :=
		h.Service.GetPublicPodcastBySlug(
			r.Context(),
			slug,
		)

	if err != nil {
		writePodcastServiceError(
			w,
			err,
			"failed to load podcast",
		)
		return
	}

	WriteJSON(
		w,
		http.StatusOK,
		podcast,
	)
}

func (h *PodcastHandler) GetMyPodcasts(
	w http.ResponseWriter,
	r *http.Request,
) {
	userID, ok :=
		middleware.UserIDFromContext(
			r.Context(),
		)

	if !ok {
		WriteError(
			w,
			http.StatusUnauthorized,
			"AUTHENTICATION_REQUIRED",
			"user authentication is required",
		)
		return
	}

	podcasts, err :=
		h.Service.GetMyPodcasts(
			r.Context(),
			userID,
		)

	if err != nil {
		writePodcastServiceError(
			w,
			err,
			"failed to load podcasts",
		)
		return
	}

	WriteJSON(
		w,
		http.StatusOK,
		podcasts,
	)
}

func (h *PodcastHandler) GetMyPodcast(
	w http.ResponseWriter,
	r *http.Request,
) {
	userID, ok :=
		middleware.UserIDFromContext(
			r.Context(),
		)

	if !ok {
		WriteError(
			w,
			http.StatusUnauthorized,
			"AUTHENTICATION_REQUIRED",
			"user authentication is required",
		)
		return
	}

	podcastID, ok :=
		parsePositiveInt64PathID(
			r.PathValue("id"),
		)

	if !ok {
		WriteError(
			w,
			http.StatusBadRequest,
			"INVALID_PODCAST_ID",
			"invalid podcast ID",
		)
		return
	}

	podcast, err :=
		h.Service.GetOwnedPodcast(
			r.Context(),
			podcastID,
			userID,
		)

	if err != nil {
		writePodcastServiceError(
			w,
			err,
			"failed to load podcast",
		)
		return
	}

	WriteJSON(
		w,
		http.StatusOK,
		podcast,
	)
}

func (h *PodcastHandler) UpdatePodcast(
	w http.ResponseWriter,
	r *http.Request,
) {
	userID, ok :=
		middleware.UserIDFromContext(
			r.Context(),
		)

	if !ok {
		WriteError(
			w,
			http.StatusUnauthorized,
			"AUTHENTICATION_REQUIRED",
			"user authentication is required",
		)
		return
	}

	podcastID, ok :=
		parsePositiveInt64PathID(
			r.PathValue("id"),
		)

	if !ok {
		WriteError(
			w,
			http.StatusBadRequest,
			"INVALID_PODCAST_ID",
			"invalid podcast ID",
		)
		return
	}

	var req UpdatePodcastRequest

	if err := decodePodcastJSON(
		r,
		&req,
	); err != nil {
		WriteError(
			w,
			http.StatusBadRequest,
			"INVALID_REQUEST",
			"invalid request body",
		)
		return
	}

	// PATCH semantics require us to preserve fields the client
	// did not provide. Load the current owned resource first.
	current, err :=
		h.Service.GetOwnedPodcast(
			r.Context(),
			podcastID,
			userID,
		)

	if err != nil {
		writePodcastServiceError(
			w,
			err,
			"failed to load podcast",
		)
		return
	}

	title := current.Podcast.Title
	description := current.Podcast.Description
	category := current.Podcast.Category
	status := current.Podcast.Status
	isExplicit := current.Podcast.IsExplicit

	if req.Title != nil {
		title = *req.Title
	}

	if req.Description != nil {
		description = *req.Description
	}

	if req.Category != nil {
		category = *req.Category
	}

	if req.IsExplicit != nil {
		isExplicit = *req.IsExplicit
	}

	if req.Status != nil {
		if !phaseFourPodcastStatusAllowed(
			*req.Status,
		) {
			WriteError(
				w,
				http.StatusBadRequest,
				"INVALID_PODCAST_STATUS",
				"status must be DRAFT, PUBLISHED, or ARCHIVED",
			)
			return
		}

		status = *req.Status
	}

	podcast, err :=
		h.Service.UpdatePodcast(
			r.Context(),
			podcastID,
			userID,
			services.UpdatePodcastInput{
				Title: title,

				Description: description,

				Category: category,

				Status: status,

				IsExplicit: isExplicit,
			},
		)

	if err != nil {
		writePodcastServiceError(
			w,
			err,
			"failed to update podcast",
		)
		return
	}

	WriteJSON(
		w,
		http.StatusOK,
		podcast,
	)
}

func (h *PodcastHandler) DeletePodcast(
	w http.ResponseWriter,
	r *http.Request,
) {
	userID, ok :=
		middleware.UserIDFromContext(
			r.Context(),
		)

	if !ok {
		WriteError(
			w,
			http.StatusUnauthorized,
			"AUTHENTICATION_REQUIRED",
			"user authentication is required",
		)
		return
	}

	podcastID, ok :=
		parsePositiveInt64PathID(
			r.PathValue("id"),
		)

	if !ok {
		WriteError(
			w,
			http.StatusBadRequest,
			"INVALID_PODCAST_ID",
			"invalid podcast ID",
		)
		return
	}

	deleted, err :=
		h.Service.DeletePodcast(
			r.Context(),
			podcastID,
			userID,
		)

	if err != nil {
		writePodcastServiceError(
			w,
			err,
			"failed to delete podcast",
		)
		return
	}

	// The database delete has succeeded. Cloudinary cleanup is
	// therefore best-effort and must not turn a valid deletion
	// into an HTTP failure.
	if h.CloudinaryService != nil {
		cleanupPodcastAsset(
			h,
			r,
			deleted.Podcast.ArtworkPublicID,
			"image",
			"DeletePodcast: failed to remove podcast artwork",
		)

		for _, episode := range deleted.Episodes {
			cleanupPodcastAsset(
				h,
				r,
				episode.AudioPublicID,
				"video",
				"DeletePodcast: failed to remove episode audio",
			)

			cleanupPodcastAsset(
				h,
				r,
				episode.ArtworkPublicID,
				"image",
				"DeletePodcast: failed to remove episode artwork",
			)
		}
	}

	w.WriteHeader(
		http.StatusNoContent,
	)
}

func (h *PodcastHandler) CreateEpisode(
	w http.ResponseWriter,
	r *http.Request,
) {
	userID, ok :=
		middleware.UserIDFromContext(
			r.Context(),
		)

	if !ok {
		WriteError(
			w,
			http.StatusUnauthorized,
			"AUTHENTICATION_REQUIRED",
			"user authentication is required",
		)
		return
	}

	podcastID, ok :=
		parsePositiveInt64PathID(
			r.PathValue("id"),
		)

	if !ok {
		WriteError(
			w,
			http.StatusBadRequest,
			"INVALID_PODCAST_ID",
			"invalid podcast ID",
		)
		return
	}

	var req CreatePodcastEpisodeRequest

	if err := decodePodcastJSON(
		r,
		&req,
	); err != nil {
		WriteError(
			w,
			http.StatusBadRequest,
			"INVALID_REQUEST",
			"invalid request body",
		)
		return
	}

	if !phaseFourEpisodeStatusAllowed(
		req.Status,
	) {
		WriteError(
			w,
			http.StatusBadRequest,
			"INVALID_EPISODE_STATUS",
			"status must be DRAFT, PUBLISHED, or ARCHIVED",
		)
		return
	}

	episode, err :=
		h.Service.CreateEpisode(
			r.Context(),
			podcastID,
			userID,
			services.CreatePodcastEpisodeInput{
				Title: req.Title,

				Description: req.Description,

				SeasonNumber: req.SeasonNumber,

				EpisodeNumber: req.EpisodeNumber,

				EpisodeType: req.EpisodeType,

				DurationMS: req.DurationMS,

				Status: req.Status,

				IsExplicit: req.IsExplicit,
			},
		)

	if err != nil {
		writePodcastServiceError(
			w,
			err,
			"failed to create podcast episode",
		)
		return
	}

	WriteJSON(
		w,
		http.StatusCreated,
		episode,
	)
}

func (h *PodcastHandler) GetPublicEpisode(
	w http.ResponseWriter,
	r *http.Request,
) {
	episodeID, ok :=
		parsePositiveInt64PathID(
			r.PathValue("id"),
		)

	if !ok {
		WriteError(
			w,
			http.StatusBadRequest,
			"INVALID_EPISODE_ID",
			"invalid podcast episode ID",
		)
		return
	}

	episode, err :=
		h.Service.GetPublicEpisode(
			r.Context(),
			episodeID,
		)

	if err != nil {
		writePodcastServiceError(
			w,
			err,
			"failed to load podcast episode",
		)
		return
	}

	WriteJSON(
		w,
		http.StatusOK,
		episode,
	)
}

func (h *PodcastHandler) UpdateEpisode(
	w http.ResponseWriter,
	r *http.Request,
) {
	userID, ok :=
		middleware.UserIDFromContext(
			r.Context(),
		)

	if !ok {
		WriteError(
			w,
			http.StatusUnauthorized,
			"AUTHENTICATION_REQUIRED",
			"user authentication is required",
		)
		return
	}

	episodeID, ok :=
		parsePositiveInt64PathID(
			r.PathValue("id"),
		)

	if !ok {
		WriteError(
			w,
			http.StatusBadRequest,
			"INVALID_EPISODE_ID",
			"invalid podcast episode ID",
		)
		return
	}

	var req UpdatePodcastEpisodeRequest

	if err := decodePodcastJSON(
		r,
		&req,
	); err != nil {
		WriteError(
			w,
			http.StatusBadRequest,
			"INVALID_REQUEST",
			"invalid request body",
		)
		return
	}

	// Load the existing resource so omitted PATCH fields retain
	// their current values.
	current, err :=
		h.Service.GetOwnedEpisode(
			r.Context(),
			episodeID,
			userID,
		)

	if err != nil {
		writePodcastServiceError(
			w,
			err,
			"failed to load podcast episode",
		)
		return
	}

	title := current.Title
	description := current.Description
	seasonNumber := current.SeasonNumber
	episodeNumber := current.EpisodeNumber
	episodeType := current.EpisodeType
	durationMS := current.DurationMS
	status := current.Status
	isExplicit := current.IsExplicit

	if req.Title != nil {
		title = *req.Title
	}

	if req.Description != nil {
		description = *req.Description
	}

	if req.SeasonNumber != nil {
		seasonNumber = *req.SeasonNumber
	}

	// In Phase 4.0, a supplied episode_number replaces the
	// existing number. Omission preserves it.
	if req.EpisodeNumber != nil {
		value := *req.EpisodeNumber
		episodeNumber = &value
	}

	if req.EpisodeType != nil {
		episodeType = *req.EpisodeType
	}

	if req.DurationMS != nil {
		durationMS = *req.DurationMS
	}

	if req.IsExplicit != nil {
		isExplicit = *req.IsExplicit
	}

	if req.Status != nil {
		if !phaseFourEpisodeStatusAllowed(
			*req.Status,
		) {
			WriteError(
				w,
				http.StatusBadRequest,
				"INVALID_EPISODE_STATUS",
				"status must be DRAFT, PUBLISHED, or ARCHIVED",
			)
			return
		}

		status = *req.Status
	}

	episode, err :=
		h.Service.UpdateEpisode(
			r.Context(),
			episodeID,
			userID,
			services.UpdatePodcastEpisodeInput{
				Title: title,

				Description: description,

				SeasonNumber: seasonNumber,

				EpisodeNumber: episodeNumber,

				EpisodeType: episodeType,

				DurationMS: durationMS,

				Status: status,

				IsExplicit: isExplicit,

				ScheduledAt: current.ScheduledAt,
			},
		)

	if err != nil {
		writePodcastServiceError(
			w,
			err,
			"failed to update podcast episode",
		)
		return
	}

	WriteJSON(
		w,
		http.StatusOK,
		episode,
	)
}

func (h *PodcastHandler) DeleteEpisode(
	w http.ResponseWriter,
	r *http.Request,
) {
	userID, ok :=
		middleware.UserIDFromContext(
			r.Context(),
		)

	if !ok {
		WriteError(
			w,
			http.StatusUnauthorized,
			"AUTHENTICATION_REQUIRED",
			"user authentication is required",
		)
		return
	}

	episodeID, ok :=
		parsePositiveInt64PathID(
			r.PathValue("id"),
		)

	if !ok {
		WriteError(
			w,
			http.StatusBadRequest,
			"INVALID_EPISODE_ID",
			"invalid podcast episode ID",
		)
		return
	}

	deleted, err :=
		h.Service.DeleteEpisode(
			r.Context(),
			episodeID,
			userID,
		)

	if err != nil {
		writePodcastServiceError(
			w,
			err,
			"failed to delete podcast episode",
		)
		return
	}

	// PostgreSQL is already updated at this point. Remove the
	// external Cloudinary assets on a best-effort basis.
	if h.CloudinaryService != nil {
		cleanupPodcastAsset(
			h,
			r,
			deleted.AudioPublicID,
			"video",
			"DeleteEpisode: failed to remove episode audio",
		)

		cleanupPodcastAsset(
			h,
			r,
			deleted.ArtworkPublicID,
			"image",
			"DeleteEpisode: failed to remove episode artwork",
		)
	}

	w.WriteHeader(
		http.StatusNoContent,
	)
}

func decodePodcastJSON(
	r *http.Request,
	destination any,
) error {
	defer r.Body.Close()

	decoder :=
		json.NewDecoder(
			r.Body,
		)

	decoder.DisallowUnknownFields()

	if err := decoder.Decode(
		destination,
	); err != nil {
		return err
	}

	// Reject a second JSON value instead of silently
	// accepting trailing request data.
	var extra any

	if err := decoder.Decode(
		&extra,
	); err == nil {
		return errors.New(
			"request body must contain exactly one JSON object",
		)
	}

	return nil
}

func parsePositiveInt64PathID(
	value string,
) (int64, bool) {
	id, err :=
		strconv.ParseInt(
			strings.TrimSpace(value),
			10,
			64,
		)

	if err != nil || id <= 0 {
		return 0, false
	}

	return id, true
}

func parsePodcastQueryInt(
	value string,
	fallback int,
) int {
	value =
		strings.TrimSpace(
			value,
		)

	if value == "" {
		return fallback
	}

	number, err :=
		strconv.Atoi(
			value,
		)

	if err != nil {
		return fallback
	}

	return number
}

func phaseFourPodcastStatusAllowed(
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

func phaseFourEpisodeStatusAllowed(
	status models.PodcastEpisodeStatus,
) bool {
	switch status {
	case models.PodcastEpisodeStatusDraft,
		models.PodcastEpisodeStatusPublished,
		models.PodcastEpisodeStatusArchived:
		return true

	default:
		return false
	}
}

func writePodcastServiceError(
	w http.ResponseWriter,
	err error,
	internalMessage string,
) {
	switch {
	case errors.Is(
		err,
		services.ErrUnauthorized,
	):
		WriteError(
			w,
			http.StatusUnauthorized,
			"AUTHENTICATION_REQUIRED",
			"authentication is required",
		)

	case errors.Is(
		err,
		services.ErrPodcastNotFound,
	):
		WriteError(
			w,
			http.StatusNotFound,
			"PODCAST_NOT_FOUND",
			"podcast not found",
		)

	case errors.Is(
		err,
		services.ErrPodcastEpisodeNotFound,
	):
		WriteError(
			w,
			http.StatusNotFound,
			"EPISODE_NOT_FOUND",
			"podcast episode not found",
		)

	case errors.Is(
		err,
		services.ErrPodcastEpisodeNumberExists,
	):
		WriteError(
			w,
			http.StatusConflict,
			"EPISODE_NUMBER_EXISTS",
			err.Error(),
		)

	case errors.Is(
		err,
		services.ErrPodcastSlugConflict,
	),
		errors.Is(
			err,
			services.ErrPodcastEpisodeSlugConflict,
		),
		errors.Is(
			err,
			repository.ErrPodcastSlugExists,
		),
		errors.Is(
			err,
			repository.ErrPodcastEpisodeSlugExists,
		):
		WriteError(
			w,
			http.StatusConflict,
			"SLUG_CONFLICT",
			"could not allocate a unique slug",
		)

	case isPodcastValidationError(err):
		WriteError(
			w,
			http.StatusBadRequest,
			"VALIDATION_ERROR",
			err.Error(),
		)

	default:
		log.Printf(
			"podcast handler: %s: %v",
			internalMessage,
			err,
		)

		WriteError(
			w,
			http.StatusInternalServerError,
			"INTERNAL_ERROR",
			internalMessage,
		)
	}
}

func isPodcastValidationError(
	err error,
) bool {
	validationErrors := []error{
		services.ErrPodcastTitleRequired,
		services.ErrPodcastTitleTooLong,
		services.ErrPodcastDescriptionTooLong,
		services.ErrPodcastCategoryTooLong,
		services.ErrInvalidPodcastStatus,
		services.ErrPodcastArtworkIncomplete,
		services.ErrPodcastPublishRequiresArtwork,

		services.ErrPodcastEpisodeTitleRequired,
		services.ErrPodcastEpisodeTitleTooLong,
		services.ErrPodcastEpisodeDescriptionTooLong,
		services.ErrInvalidPodcastEpisodeType,
		services.ErrInvalidPodcastEpisodeStatus,
		services.ErrInvalidPodcastSeasonNumber,
		services.ErrInvalidPodcastEpisodeNumber,
		services.ErrInvalidPodcastDuration,
		services.ErrPodcastEpisodeAudioIncomplete,
		services.ErrPodcastEpisodeArtworkIncomplete,
		services.ErrPodcastEpisodeAudioRequired,
		services.ErrPodcastScheduleRequired,
		services.ErrPodcastScheduleMustBeFuture,
		services.ErrPodcastScheduleNotAllowed,
	}

	for _, candidate := range validationErrors {
		if errors.Is(
			err,
			candidate,
		) {
			return true
		}
	}

	return false
}
