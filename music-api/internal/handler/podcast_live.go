package handler

import (
	"errors"
	"log"
	"net/http"
	"strings"
	"time"

	"music-api/internal/middleware"
	"music-api/internal/repository"
	"music-api/internal/services"
)

type PodcastLiveHandler struct {
	Service *services.PodcastLiveService

	// JWTService enables OPTIONAL authentication on public
	// endpoints (anonymous live listening with stable
	// identities for signed-in users).
	JWTService *services.JWTService
}

func NewPodcastLiveHandler(
	service *services.PodcastLiveService,
	jwtService *services.JWTService,
) *PodcastLiveHandler {
	return &PodcastLiveHandler{
		Service:    service,
		JWTService: jwtService,
	}
}

// SchedulePodcastLiveRequest is the client request for
// scheduling a live broadcast.
//
// Only the schedule is accepted. Host, provider room, and
// every runtime timestamp are decided server-side.
type SchedulePodcastLiveRequest struct {
	ScheduledStartAt string `json:"scheduled_start_at"`
}

// ScheduleLiveEpisode schedules a live broadcast for one of
// the caller's draft episodes.
//
// POST /api/v1/me/podcast-episodes/{id}/live/schedule
func (h *PodcastLiveHandler) ScheduleLiveEpisode(
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

	var req SchedulePodcastLiveRequest

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

	scheduledStartAt, err :=
		parsePodcastLiveTimestamp(
			req.ScheduledStartAt,
		)
	if err != nil {
		WriteError(
			w,
			http.StatusBadRequest,
			"VALIDATION_ERROR",
			"scheduled_start_at must be an RFC 3339 timestamp",
		)
		return
	}

	details, err :=
		h.Service.ScheduleLiveEpisode(
			r.Context(),
			userID,
			episodeID,
			services.ScheduleLiveEpisodeInput{
				ScheduledStartAt: scheduledStartAt,
			},
		)

	if err != nil {
		writePodcastLiveError(
			w,
			"ScheduleLiveEpisode",
			err,
		)
		return
	}

	WriteJSON(
		w,
		http.StatusCreated,
		details,
	)
}

// GetLiveByEpisode returns the caller's live session for an
// episode.
//
// GET /api/v1/me/podcast-episodes/{id}/live
func (h *PodcastLiveHandler) GetLiveByEpisode(
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

	details, err :=
		h.Service.GetOwnedLiveByEpisode(
			r.Context(),
			userID,
			episodeID,
		)

	if err != nil {
		writePodcastLiveError(
			w,
			"GetLiveByEpisode",
			err,
		)
		return
	}

	WriteJSON(
		w,
		http.StatusOK,
		details,
	)
}

// StartLiveEpisode starts a scheduled broadcast.
//
// POST /api/v1/me/podcast-episodes/{id}/live/start
func (h *PodcastLiveHandler) StartLiveEpisode(
	w http.ResponseWriter,
	r *http.Request,
) {
	userID, episodeID, ok :=
		parsePodcastLiveEpisodeRequest(
			w,
			r,
		)

	if !ok {
		return
	}

	details, err :=
		h.Service.StartLiveEpisode(
			r.Context(),
			userID,
			episodeID,
		)

	if err != nil {
		writePodcastLiveError(
			w,
			"StartLiveEpisode",
			err,
		)
		return
	}

	WriteJSON(
		w,
		http.StatusOK,
		details,
	)
}

// EndLiveEpisode ends a running broadcast.
//
// POST /api/v1/me/podcast-episodes/{id}/live/end
func (h *PodcastLiveHandler) EndLiveEpisode(
	w http.ResponseWriter,
	r *http.Request,
) {
	userID, episodeID, ok :=
		parsePodcastLiveEpisodeRequest(
			w,
			r,
		)

	if !ok {
		return
	}

	details, err :=
		h.Service.EndLiveEpisode(
			r.Context(),
			userID,
			episodeID,
		)

	if err != nil {
		writePodcastLiveError(
			w,
			"EndLiveEpisode",
			err,
		)
		return
	}

	WriteJSON(
		w,
		http.StatusOK,
		details,
	)
}

// CancelLiveEpisode cancels a scheduled broadcast.
//
// POST /api/v1/me/podcast-episodes/{id}/live/cancel
func (h *PodcastLiveHandler) CancelLiveEpisode(
	w http.ResponseWriter,
	r *http.Request,
) {
	userID, episodeID, ok :=
		parsePodcastLiveEpisodeRequest(
			w,
			r,
		)

	if !ok {
		return
	}

	details, err :=
		h.Service.CancelLiveEpisode(
			r.Context(),
			userID,
			episodeID,
		)

	if err != nil {
		writePodcastLiveError(
			w,
			"CancelLiveEpisode",
			err,
		)
		return
	}

	WriteJSON(
		w,
		http.StatusOK,
		details,
	)
}

// PublishLiveRecording publishes the finished recording of an
// ENDED live session as the episode's regular audio. Host
// only, and only with the host's explicit confirmation.
//
// POST /api/v1/me/podcast-live-sessions/{id}/publish-recording
func (h *PodcastLiveHandler) PublishLiveRecording(
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

	details, err :=
		h.Service.PublishLiveRecording(
			r.Context(),
			userID,
			r.PathValue("id"),
		)

	if err != nil {
		writePodcastLiveError(
			w,
			"PublishLiveRecording",
			err,
		)
		return
	}

	WriteJSON(
		w,
		http.StatusOK,
		details,
	)
}

// GetHostToken mints a short-lived host token for the
// session's room. Host only.
//
// POST /api/v1/podcast-live-sessions/{id}/host-token
func (h *PodcastLiveHandler) GetHostToken(
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

	token, err :=
		h.Service.GetHostToken(
			r.Context(),
			userID,
			r.PathValue("id"),
		)

	if err != nil {
		writePodcastLiveError(
			w,
			"GetHostToken",
			err,
		)
		return
	}

	WriteJSON(
		w,
		http.StatusOK,
		token,
	)
}

// GetListenerToken mints a short-lived subscribe-only token
// for a publicly joinable live session.
//
// Anonymous listening is allowed. Signed-in users get a
// stable derived identity; the caller never supplies an
// identity or permissions.
//
// POST /api/v1/podcast-live-sessions/{id}/listener-token
func (h *PodcastLiveHandler) GetListenerToken(
	w http.ResponseWriter,
	r *http.Request,
) {
	userID, authOK := h.optionalUserID(r)

	// An explicitly supplied but invalid token is rejected
	// rather than silently downgraded to anonymous.
	if !authOK {
		WriteError(
			w,
			http.StatusUnauthorized,
			"INVALID_TOKEN",
			"invalid authentication token",
		)
		return
	}

	token, err :=
		h.Service.GetListenerToken(
			r.Context(),
			r.PathValue("id"),
			userID,
		)

	if err != nil {
		writePodcastLiveError(
			w,
			"GetListenerToken",
			err,
		)
		return
	}

	WriteJSON(
		w,
		http.StatusOK,
		token,
	)
}

// -----------------------------------------------------------------
// Public discovery
// -----------------------------------------------------------------

// ListUpcomingLive returns scheduled public broadcasts.
//
// GET /api/v1/podcast-live/upcoming
func (h *PodcastLiveHandler) ListUpcomingLive(
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

	broadcasts, err :=
		h.Service.ListUpcomingLive(
			r.Context(),
			limit,
			offset,
		)

	if err != nil {
		writePodcastLiveError(
			w,
			"ListUpcomingLive",
			err,
		)
		return
	}

	WriteJSON(
		w,
		http.StatusOK,
		broadcasts,
	)
}

// ListCurrentlyLive returns broadcasts that are live right
// now.
//
// GET /api/v1/podcast-live/current
func (h *PodcastLiveHandler) ListCurrentlyLive(
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

	broadcasts, err :=
		h.Service.ListCurrentlyLive(
			r.Context(),
			limit,
			offset,
		)

	if err != nil {
		writePodcastLiveError(
			w,
			"ListCurrentlyLive",
			err,
		)
		return
	}

	WriteJSON(
		w,
		http.StatusOK,
		broadcasts,
	)
}

// GetPublicLive returns the public broadcast view of one live
// session.
//
// GET /api/v1/podcast-live/{id}
func (h *PodcastLiveHandler) GetPublicLive(
	w http.ResponseWriter,
	r *http.Request,
) {
	broadcast, err :=
		h.Service.GetPublicLive(
			r.Context(),
			r.PathValue("id"),
		)

	if err != nil {
		writePodcastLiveError(
			w,
			"GetPublicLive",
			err,
		)
		return
	}

	WriteJSON(
		w,
		http.StatusOK,
		broadcast,
	)
}

// -----------------------------------------------------------------
// Internal helpers
// -----------------------------------------------------------------

// parsePodcastLiveEpisodeRequest extracts the authenticated
// user and episode ID shared by the episode-scoped live
// endpoints. It writes the error response itself on failure.
func parsePodcastLiveEpisodeRequest(
	w http.ResponseWriter,
	r *http.Request,
) (int, int64, bool) {
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
		return 0, 0, false
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
		return 0, 0, false
	}

	return userID, episodeID, true
}

// optionalUserID resolves the caller's user ID when a valid
// Bearer token is present. A missing header means anonymous
// (0, true). An explicitly supplied but invalid or malformed
// token returns (0, false): silently downgrading a failed
// authentication to anonymous would hide real problems from
// the caller.
func (h *PodcastLiveHandler) optionalUserID(
	r *http.Request,
) (int, bool) {
	header :=
		strings.TrimSpace(
			r.Header.Get("Authorization"),
		)

	if header == "" {
		return 0, true
	}

	parts := strings.SplitN(
		header,
		" ",
		2,
	)

	if len(parts) != 2 ||
		!strings.EqualFold(
			parts[0],
			"Bearer",
		) {
		return 0, false
	}

	identity, err :=
		h.JWTService.ValidateIdentity(
			strings.TrimSpace(parts[1]),
		)
	if err != nil {
		return 0, false
	}

	if identity.UserID <= 0 {
		return 0, false
	}

	return identity.UserID, true
}

func parsePodcastLiveTimestamp(
	value string,
) (time.Time, error) {
	value =
		strings.TrimSpace(
			value,
		)

	if value == "" {
		return time.Time{},
			errors.New(
				"scheduled_start_at is required",
			)
	}

	return time.Parse(
		time.RFC3339,
		value,
	)
}

func writePodcastLiveError(
	w http.ResponseWriter,
	operation string,
	err error,
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
		repository.ErrPodcastLiveSessionNotFound,
	):
		WriteError(
			w,
			http.StatusNotFound,
			"PODCAST_LIVE_SESSION_NOT_FOUND",
			"podcast live session not found",
		)

	case errors.Is(
		err,
		services.ErrPodcastLiveAlreadyScheduled,
	):
		WriteError(
			w,
			http.StatusConflict,
			"LIVE_SESSION_ALREADY_EXISTS",
			"episode already has a live session",
		)

	case errors.Is(
		err,
		services.ErrPodcastLiveEpisodeNotSchedulable,
	):
		WriteError(
			w,
			http.StatusConflict,
			"EPISODE_NOT_SCHEDULABLE",
			"episode cannot be scheduled for a live broadcast",
		)

	case errors.Is(
		err,
		services.ErrPodcastLivePodcastNotAllowed,
	):
		WriteError(
			w,
			http.StatusConflict,
			"PODCAST_NOT_LIVE_ELIGIBLE",
			"podcast cannot host a live broadcast",
		)

	case errors.Is(
		err,
		services.ErrPodcastLiveStartTooEarly,
	):
		WriteError(
			w,
			http.StatusConflict,
			"LIVE_START_TOO_EARLY",
			"live broadcast cannot start this early",
		)

	case errors.Is(
		err,
		repository.ErrPodcastLiveSessionNotScheduled,
	):
		WriteError(
			w,
			http.StatusConflict,
			"LIVE_SESSION_NOT_SCHEDULED",
			"podcast live session is not scheduled",
		)

	case errors.Is(
		err,
		repository.ErrPodcastLiveSessionNotLive,
	):
		WriteError(
			w,
			http.StatusConflict,
			"LIVE_SESSION_NOT_LIVE",
			"podcast live session is not live",
		)

	case errors.Is(
		err,
		repository.ErrPodcastLiveSessionNotCancellable,
	):
		WriteError(
			w,
			http.StatusConflict,
			"LIVE_SESSION_NOT_CANCELLABLE",
			"podcast live session cannot be cancelled",
		)

	case errors.Is(
		err,
		repository.ErrPodcastLiveSessionNotEnded,
	):
		WriteError(
			w,
			http.StatusConflict,
			"LIVE_SESSION_NOT_ENDED",
			"podcast live session has not ended",
		)

	case errors.Is(
		err,
		repository.ErrPodcastLiveRecordingNotReady,
	):
		WriteError(
			w,
			http.StatusConflict,
			"LIVE_RECORDING_NOT_READY",
			"no finished recording is available to publish",
		)

	case errors.Is(
		err,
		services.ErrPodcastLiveNotJoinable,
	):
		WriteError(
			w,
			http.StatusConflict,
			"LIVE_SESSION_NOT_JOINABLE",
			"live session is not joinable",
		)

	case errors.Is(
		err,
		services.ErrLiveProviderNotConfigured,
	):
		WriteError(
			w,
			http.StatusServiceUnavailable,
			"LIVE_PROVIDER_NOT_CONFIGURED",
			"live audio streaming is not configured on this server",
		)

	case services.IsPodcastLiveValidationError(
		err,
	):
		WriteError(
			w,
			http.StatusBadRequest,
			"VALIDATION_ERROR",
			err.Error(),
		)

	default:
		log.Printf(
			"podcast live handler: %s: %v",
			operation,
			err,
		)

		WriteError(
			w,
			http.StatusInternalServerError,
			"INTERNAL_ERROR",
			"live podcast request failed",
		)
	}
}
