package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgconn"

	"music-api/internal/middleware"
	"music-api/internal/models"
	"music-api/internal/services"
)

type ReleaseHandler struct {
	Service *services.ReleaseService
}

func NewReleaseHandler(
	service *services.ReleaseService,
) *ReleaseHandler {
	return &ReleaseHandler{
		Service: service,
	}
}

type CreateReleaseRequest struct {
	Title string `json:"title"`

	ReleaseType models.ReleaseType `json:"release_type"`

	CoverImageURL string `json:"cover_image_url"`

	CoverImagePublicID string `json:"cover_image_public_id"`

	Description string `json:"description"`

	ReleaseDate string `json:"release_date"`

	IsPublished bool `json:"is_published"`
}

type UpdateReleaseRequest struct {
	Title string `json:"title"`

	ReleaseType models.ReleaseType `json:"release_type"`

	CoverImageURL string `json:"cover_image_url"`

	CoverImagePublicID string `json:"cover_image_public_id"`

	Description string `json:"description"`

	ReleaseDate string `json:"release_date"`

	IsPublished bool `json:"is_published"`
}

type AddReleaseTrackRequest struct {
	TrackNumber int `json:"track_number"`
}

// CreateRelease creates a new release belonging to the
// authenticated artist.
func (h *ReleaseHandler) CreateRelease(
	w http.ResponseWriter,
	r *http.Request,
) {
	artistID, ok :=
		middleware.ArtistIDFromContext(
			r.Context(),
		)

	if !ok {
		WriteError(
			w,
			http.StatusUnauthorized,
			"AUTHENTICATION_REQUIRED",
			"artist authentication is required",
		)
		return
	}

	var req CreateReleaseRequest

	defer r.Body.Close()

	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(&req); err != nil {
		WriteError(
			w,
			http.StatusBadRequest,
			"INVALID_REQUEST",
			"invalid request body",
		)
		return
	}

	releaseDate, err :=
		parseReleaseDate(
			req.ReleaseDate,
		)

	if err != nil {
		WriteError(
			w,
			http.StatusBadRequest,
			"INVALID_RELEASE_DATE",
			"release_date must use YYYY-MM-DD format",
		)
		return
	}

	release, err :=
		h.Service.CreateRelease(
			r.Context(),
			artistID,
			services.CreateReleaseInput{
				Title: req.Title,

				ReleaseType: req.ReleaseType,

				CoverImageURL: req.CoverImageURL,

				CoverImagePublicID: req.CoverImagePublicID,

				Description: req.Description,

				ReleaseDate: releaseDate,

				IsPublished: req.IsPublished,
			},
		)

	if err != nil {
		writeReleaseServiceError(
			w,
			err,
			"failed to create release",
		)
		return
	}

	WriteJSON(
		w,
		http.StatusCreated,
		release,
	)
}

// GetRelease returns a published release together with
// its ordered tracks.
//
// Draft releases are deliberately not exposed publicly.
func (h *ReleaseHandler) GetRelease(
	w http.ResponseWriter,
	r *http.Request,
) {
	releaseID, ok :=
		parsePositivePathID(
			r.PathValue("id"),
		)

	if !ok {
		WriteError(
			w,
			http.StatusBadRequest,
			"INVALID_RELEASE_ID",
			"invalid release ID",
		)
		return
	}

	details, err :=
		h.Service.GetRelease(
			r.Context(),
			releaseID,
		)

	if err != nil {
		writeReleaseServiceError(
			w,
			err,
			"failed to load release",
		)
		return
	}

	if !details.Release.IsPublished {
		WriteError(
			w,
			http.StatusNotFound,
			"RELEASE_NOT_FOUND",
			"release not found",
		)
		return
	}

	WriteJSON(
		w,
		http.StatusOK,
		details,
	)
}

// GetArtistReleases returns only published releases belonging
// to the requested artist.
func (h *ReleaseHandler) GetArtistReleases(
	w http.ResponseWriter,
	r *http.Request,
) {
	artistID, ok :=
		parsePositivePathID(
			r.PathValue("id"),
		)

	if !ok {
		WriteError(
			w,
			http.StatusBadRequest,
			"INVALID_ARTIST_ID",
			"invalid artist ID",
		)
		return
	}

	releases, err :=
		h.Service.GetArtistReleases(
			r.Context(),
			artistID,
		)

	if err != nil {
		WriteError(
			w,
			http.StatusInternalServerError,
			"INTERNAL_ERROR",
			"failed to load artist releases",
		)
		return
	}

	published := make(
		[]models.Release,
		0,
		len(releases),
	)

	for _, release := range releases {

		if release.IsPublished {
			published = append(
				published,
				release,
			)
		}
	}

	WriteJSON(
		w,
		http.StatusOK,
		published,
	)
}

// UpdateRelease updates an authenticated artist's own release.
func (h *ReleaseHandler) UpdateRelease(
	w http.ResponseWriter,
	r *http.Request,
) {
	artistID, ok :=
		middleware.ArtistIDFromContext(
			r.Context(),
		)

	if !ok {
		WriteError(
			w,
			http.StatusUnauthorized,
			"AUTHENTICATION_REQUIRED",
			"artist authentication is required",
		)
		return
	}

	releaseID, ok :=
		parsePositivePathID(
			r.PathValue("id"),
		)

	if !ok {
		WriteError(
			w,
			http.StatusBadRequest,
			"INVALID_RELEASE_ID",
			"invalid release ID",
		)
		return
	}

	var req UpdateReleaseRequest

	defer r.Body.Close()

	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(&req); err != nil {
		WriteError(
			w,
			http.StatusBadRequest,
			"INVALID_REQUEST",
			"invalid request body",
		)
		return
	}

	releaseDate, err :=
		parseReleaseDate(
			req.ReleaseDate,
		)

	if err != nil {
		WriteError(
			w,
			http.StatusBadRequest,
			"INVALID_RELEASE_DATE",
			"release_date must use YYYY-MM-DD format",
		)
		return
	}

	release, err :=
		h.Service.UpdateRelease(
			r.Context(),
			releaseID,
			artistID,
			services.UpdateReleaseInput{
				Title: req.Title,

				ReleaseType: req.ReleaseType,

				CoverImageURL: req.CoverImageURL,

				CoverImagePublicID: req.CoverImagePublicID,

				Description: req.Description,

				ReleaseDate: releaseDate,

				IsPublished: req.IsPublished,
			},
		)

	if err != nil {
		writeReleaseServiceError(
			w,
			err,
			"failed to update release",
		)
		return
	}

	WriteJSON(
		w,
		http.StatusOK,
		release,
	)
}

// DeleteRelease deletes the artist-owned release.
//
// Tracks themselves are preserved because music.release_id
// uses ON DELETE SET NULL.
func (h *ReleaseHandler) DeleteRelease(
	w http.ResponseWriter,
	r *http.Request,
) {
	artistID, ok :=
		middleware.ArtistIDFromContext(
			r.Context(),
		)

	if !ok {
		WriteError(
			w,
			http.StatusUnauthorized,
			"AUTHENTICATION_REQUIRED",
			"artist authentication is required",
		)
		return
	}

	releaseID, ok :=
		parsePositivePathID(
			r.PathValue("id"),
		)

	if !ok {
		WriteError(
			w,
			http.StatusBadRequest,
			"INVALID_RELEASE_ID",
			"invalid release ID",
		)
		return
	}

	err := h.Service.DeleteRelease(
		r.Context(),
		releaseID,
		artistID,
	)

	if err != nil {
		writeReleaseServiceError(
			w,
			err,
			"failed to delete release",
		)
		return
	}

	w.WriteHeader(
		http.StatusNoContent,
	)
}

// AddTrack adds an artist-owned track to an artist-owned release.
func (h *ReleaseHandler) AddTrack(
	w http.ResponseWriter,
	r *http.Request,
) {
	artistID, ok :=
		middleware.ArtistIDFromContext(
			r.Context(),
		)

	if !ok {
		WriteError(
			w,
			http.StatusUnauthorized,
			"AUTHENTICATION_REQUIRED",
			"artist authentication is required",
		)
		return
	}

	releaseID, ok :=
		parsePositivePathID(
			r.PathValue("id"),
		)

	if !ok {
		WriteError(
			w,
			http.StatusBadRequest,
			"INVALID_RELEASE_ID",
			"invalid release ID",
		)
		return
	}

	musicID, ok :=
		parsePositivePathID(
			r.PathValue("musicID"),
		)

	if !ok {
		WriteError(
			w,
			http.StatusBadRequest,
			"INVALID_MUSIC_ID",
			"invalid music ID",
		)
		return
	}

	var req AddReleaseTrackRequest

	defer r.Body.Close()

	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(&req); err != nil {
		WriteError(
			w,
			http.StatusBadRequest,
			"INVALID_REQUEST",
			"invalid request body",
		)
		return
	}

	err := h.Service.AddTrack(
		r.Context(),
		releaseID,
		artistID,
		musicID,
		req.TrackNumber,
	)

	if err != nil {
		writeReleaseServiceError(
			w,
			err,
			"failed to add track to release",
		)
		return
	}

	details, err :=
		h.Service.GetRelease(
			r.Context(),
			releaseID,
		)

	if err != nil {
		writeReleaseServiceError(
			w,
			err,
			"track was added but release could not be reloaded",
		)
		return
	}

	WriteJSON(
		w,
		http.StatusOK,
		details,
	)
}

// RemoveTrack removes a track from a release without
// deleting the music record.
func (h *ReleaseHandler) RemoveTrack(
	w http.ResponseWriter,
	r *http.Request,
) {
	artistID, ok :=
		middleware.ArtistIDFromContext(
			r.Context(),
		)

	if !ok {
		WriteError(
			w,
			http.StatusUnauthorized,
			"AUTHENTICATION_REQUIRED",
			"artist authentication is required",
		)
		return
	}

	releaseID, ok :=
		parsePositivePathID(
			r.PathValue("id"),
		)

	if !ok {
		WriteError(
			w,
			http.StatusBadRequest,
			"INVALID_RELEASE_ID",
			"invalid release ID",
		)
		return
	}

	musicID, ok :=
		parsePositivePathID(
			r.PathValue("musicID"),
		)

	if !ok {
		WriteError(
			w,
			http.StatusBadRequest,
			"INVALID_MUSIC_ID",
			"invalid music ID",
		)
		return
	}

	err := h.Service.RemoveTrack(
		r.Context(),
		releaseID,
		artistID,
		musicID,
	)

	if err != nil {
		writeReleaseServiceError(
			w,
			err,
			"failed to remove track from release",
		)
		return
	}

	w.WriteHeader(
		http.StatusNoContent,
	)
}

func parsePositivePathID(
	value string,
) (int, bool) {
	id, err := strconv.Atoi(
		strings.TrimSpace(value),
	)

	if err != nil || id <= 0 {
		return 0, false
	}

	return id, true
}

func parseReleaseDate(
	value string,
) (*time.Time, error) {
	value = strings.TrimSpace(value)

	if value == "" {
		return nil, nil
	}

	parsed, err := time.Parse(
		"2006-01-02",
		value,
	)

	if err != nil {
		return nil, err
	}

	return &parsed, nil
}

func writeReleaseServiceError(
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
			"artist authentication is required",
		)

	case errors.Is(
		err,
		services.ErrReleaseTitleRequired,
	):
		WriteError(
			w,
			http.StatusBadRequest,
			"VALIDATION_ERROR",
			err.Error(),
		)

	case errors.Is(
		err,
		services.ErrInvalidReleaseType,
	):
		WriteError(
			w,
			http.StatusBadRequest,
			"INVALID_RELEASE_TYPE",
			"release type must be SINGLE, EP, or ALBUM",
		)

	case errors.Is(
		err,
		services.ErrInvalidReleaseTrackNumber,
	):
		WriteError(
			w,
			http.StatusBadRequest,
			"INVALID_TRACK_NUMBER",
			err.Error(),
		)

	case errors.Is(
		err,
		services.ErrReleaseNotFound,
	):
		WriteError(
			w,
			http.StatusNotFound,
			"RELEASE_NOT_FOUND",
			"release not found",
		)

	case errors.Is(
		err,
		services.ErrMusicNotFound,
	):
		WriteError(
			w,
			http.StatusNotFound,
			"MUSIC_NOT_FOUND",
			"music track not found",
		)

	case errors.Is(
		err,
		services.ErrReleaseTrackNotOwned,
	):
		// Do not expose ownership details.
		WriteError(
			w,
			http.StatusNotFound,
			"MUSIC_NOT_FOUND",
			"music track not found",
		)

	case errors.Is(
		err,
		services.ErrReleaseTrackAlreadyAssigned,
	):
		WriteError(
			w,
			http.StatusConflict,
			"TRACK_ALREADY_ASSIGNED",
			"track already belongs to another release",
		)

	case isUniqueConstraintError(err):
		WriteError(
			w,
			http.StatusConflict,
			"TRACK_NUMBER_CONFLICT",
			"another track already uses that track number in this release",
		)

	default:
		WriteError(
			w,
			http.StatusInternalServerError,
			"INTERNAL_ERROR",
			internalMessage,
		)
	}
}

func isUniqueConstraintError(
	err error,
) bool {
	var pgErr *pgconn.PgError

	if !errors.As(
		err,
		&pgErr,
	) {
		return false
	}

	return pgErr.Code == "23505"
}

// GetMyReleases returns every release owned by the
// authenticated artist, including unpublished drafts.
func (h *ReleaseHandler) GetMyReleases(
	w http.ResponseWriter,
	r *http.Request,
) {
	artistID, ok :=
		middleware.ArtistIDFromContext(
			r.Context(),
		)

	if !ok {
		WriteError(
			w,
			http.StatusUnauthorized,
			"AUTHENTICATION_REQUIRED",
			"artist authentication is required",
		)
		return
	}

	releases, err :=
		h.Service.GetArtistReleases(
			r.Context(),
			artistID,
		)

	if err != nil {
		WriteError(
			w,
			http.StatusInternalServerError,
			"INTERNAL_ERROR",
			"failed to load releases",
		)
		return
	}

	WriteJSON(
		w,
		http.StatusOK,
		releases,
	)
}

// GetMyRelease returns one release owned by the authenticated
// artist together with its ordered tracks.
//
// Unlike the public GetRelease handler, this endpoint intentionally
// allows the owner to retrieve an unpublished draft.
func (h *ReleaseHandler) GetMyRelease(
	w http.ResponseWriter,
	r *http.Request,
) {
	artistID, ok :=
		middleware.ArtistIDFromContext(
			r.Context(),
		)

	if !ok {
		WriteError(
			w,
			http.StatusUnauthorized,
			"AUTHENTICATION_REQUIRED",
			"artist authentication is required",
		)
		return
	}

	releaseID, ok :=
		parsePositivePathID(
			r.PathValue("id"),
		)

	if !ok {
		WriteError(
			w,
			http.StatusBadRequest,
			"INVALID_RELEASE_ID",
			"invalid release ID",
		)
		return
	}

	details, err :=
		h.Service.GetRelease(
			r.Context(),
			releaseID,
		)

	if err != nil {
		writeReleaseServiceError(
			w,
			err,
			"failed to load release",
		)
		return
	}

	// Do not reveal another artist's release through
	// an authenticated ownership endpoint.
	if details.Release.ArtistID != artistID {
		WriteError(
			w,
			http.StatusNotFound,
			"RELEASE_NOT_FOUND",
			"release not found",
		)
		return
	}

	WriteJSON(
		w,
		http.StatusOK,
		details,
	)
}
