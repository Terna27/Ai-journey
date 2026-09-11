package handler

import (
	"errors"
	"log"
	"net/http"
	"strconv"

	"music-api/internal/middleware"
	"music-api/internal/repository"
	"music-api/internal/services"
)

type PodcastPlaybackHandler struct {
	Service *services.PodcastPlaybackService
}

func NewPodcastPlaybackHandler(
	service *services.PodcastPlaybackService,
) *PodcastPlaybackHandler {
	return &PodcastPlaybackHandler{
		Service: service,
	}
}

// CreateSession starts a new authenticated podcast playback
// session.
//
// POST /api/v1/podcast-playback/sessions
func (h *PodcastPlaybackHandler) CreateSession(
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

	var input services.CreatePodcastPlaybackSessionInput

	if err := decodePlaybackJSON(
		w,
		r,
		&input,
	); err != nil {
		WriteError(
			w,
			http.StatusBadRequest,
			"INVALID_REQUEST",
			err.Error(),
		)
		return
	}

	session, err :=
		h.Service.CreateSession(
			r.Context(),
			userID,
			input,
		)

	if err != nil {
		handlePodcastPlaybackError(
			w,
			"CreateSession",
			err,
		)
		return
	}

	WriteJSON(
		w,
		http.StatusCreated,
		session,
	)
}

// GetSession returns an authenticated user's podcast
// playback session.
//
// GET /api/v1/podcast-playback/sessions/{session_id}
func (h *PodcastPlaybackHandler) GetSession(
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

	sessionID :=
		r.PathValue("session_id")

	session, err :=
		h.Service.GetSession(
			r.Context(),
			sessionID,
			userID,
		)

	if err != nil {
		handlePodcastPlaybackError(
			w,
			"GetSession",
			err,
		)
		return
	}

	WriteJSON(
		w,
		http.StatusOK,
		session,
	)
}

// UpdateProgress records the user's latest podcast playback
// position and accumulated listening time.
//
// PATCH /api/v1/podcast-playback/sessions/{session_id}
func (h *PodcastPlaybackHandler) UpdateProgress(
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

	sessionID :=
		r.PathValue("session_id")

	var input services.UpdatePodcastPlaybackProgressInput

	if err := decodePlaybackJSON(
		w,
		r,
		&input,
	); err != nil {
		WriteError(
			w,
			http.StatusBadRequest,
			"INVALID_REQUEST",
			err.Error(),
		)
		return
	}

	session, err :=
		h.Service.UpdateProgress(
			r.Context(),
			sessionID,
			userID,
			input,
		)

	if err != nil {
		handlePodcastPlaybackError(
			w,
			"UpdateProgress",
			err,
		)
		return
	}

	WriteJSON(
		w,
		http.StatusOK,
		session,
	)
}

// CompleteSession marks a podcast playback session
// complete. Safe to retry.
//
// POST /api/v1/podcast-playback/sessions/{session_id}/complete
func (h *PodcastPlaybackHandler) CompleteSession(
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

	sessionID :=
		r.PathValue("session_id")

	var input services.UpdatePodcastPlaybackProgressInput

	if err := decodePlaybackJSON(
		w,
		r,
		&input,
	); err != nil {
		WriteError(
			w,
			http.StatusBadRequest,
			"INVALID_REQUEST",
			err.Error(),
		)
		return
	}

	session, err :=
		h.Service.CompleteSession(
			r.Context(),
			sessionID,
			userID,
			input,
		)

	if err != nil {
		handlePodcastPlaybackError(
			w,
			"CompleteSession",
			err,
		)
		return
	}

	WriteJSON(
		w,
		http.StatusOK,
		session,
	)
}

// GetPodcastListeningHistory returns the authenticated
// user's recently played podcast episodes.
//
// GET /api/v1/me/podcast-listening-history?limit=20&offset=0
func (h *PodcastPlaybackHandler) GetPodcastListeningHistory(
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

	query, err :=
		parsePodcastHistoryQuery(r)

	if err != nil {
		WriteError(
			w,
			http.StatusBadRequest,
			"INVALID_PAGINATION",
			err.Error(),
		)
		return
	}

	history, err :=
		h.Service.GetPodcastListeningHistory(
			r.Context(),
			userID,
			query,
		)

	if err != nil {
		handlePodcastPlaybackError(
			w,
			"GetPodcastListeningHistory",
			err,
		)
		return
	}

	WriteJSON(
		w,
		http.StatusOK,
		history,
	)
}

// GetContinueListening returns the authenticated user's
// unfinished podcast episodes with meaningful progress.
//
// GET /api/v1/me/podcast-continue-listening?limit=20&offset=0
func (h *PodcastPlaybackHandler) GetContinueListening(
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

	query, err :=
		parsePodcastHistoryQuery(r)

	if err != nil {
		WriteError(
			w,
			http.StatusBadRequest,
			"INVALID_PAGINATION",
			err.Error(),
		)
		return
	}

	items, err :=
		h.Service.GetContinueListening(
			r.Context(),
			userID,
			query,
		)

	if err != nil {
		handlePodcastPlaybackError(
			w,
			"GetContinueListening",
			err,
		)
		return
	}

	WriteJSON(
		w,
		http.StatusOK,
		items,
	)
}

func parsePodcastHistoryQuery(
	r *http.Request,
) (services.PodcastHistoryQuery, error) {
	var result services.PodcastHistoryQuery

	limitValue :=
		r.URL.Query().Get("limit")

	if limitValue != "" {
		limit, err :=
			strconv.Atoi(limitValue)

		if err != nil ||
			limit < 1 {
			return result,
				errors.New(
					"limit must be a positive integer",
				)
		}

		result.Limit = limit
	}

	offsetValue :=
		r.URL.Query().Get("offset")

	if offsetValue != "" {
		offset, err :=
			strconv.Atoi(offsetValue)

		if err != nil ||
			offset < 0 {
			return result,
				errors.New(
					"offset must be zero or greater",
				)
		}

		result.Offset = offset
	}

	return result, nil
}

func handlePodcastPlaybackError(
	w http.ResponseWriter,
	operation string,
	err error,
) {
	switch {
	case services.IsPodcastPlaybackValidationError(
		err,
	):
		WriteError(
			w,
			http.StatusBadRequest,
			"VALIDATION_ERROR",
			err.Error(),
		)

	case errors.Is(
		err,
		repository.ErrPodcastPlaybackSessionNotFound,
	):
		WriteError(
			w,
			http.StatusNotFound,
			"PODCAST_PLAYBACK_SESSION_NOT_FOUND",
			"podcast playback session not found",
		)

	case errors.Is(
		err,
		repository.ErrPodcastPlaybackSessionCompleted,
	):
		WriteError(
			w,
			http.StatusConflict,
			"PODCAST_PLAYBACK_SESSION_COMPLETED",
			"podcast playback session is already completed",
		)

	default:
		log.Printf(
			"%s failed: %v",
			operation,
			err,
		)

		WriteError(
			w,
			http.StatusInternalServerError,
			"INTERNAL_ERROR",
			"playback request failed",
		)
	}
}
