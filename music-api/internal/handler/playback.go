package handler

import (
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"strconv"

	"music-api/internal/middleware"
	"music-api/internal/repository"
	"music-api/internal/services"
)

const maxPlaybackRequestBytes int64 = 64 * 1024

type PlaybackHandler struct {
	Service *services.PlaybackService
}

func NewPlaybackHandler(
	service *services.PlaybackService,
) *PlaybackHandler {
	return &PlaybackHandler{
		Service: service,
	}
}

// CreateSession starts a new authenticated playback session.
//
// POST /api/v1/playback/sessions
func (h *PlaybackHandler) CreateSession(
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

	var input services.CreatePlaybackSessionInput

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
		handlePlaybackError(
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

// GetSession returns an authenticated user's playback
// session.
//
// GET /api/v1/playback/sessions/{session_id}
func (h *PlaybackHandler) GetSession(
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
		handlePlaybackError(
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

// UpdateProgress records the user's latest playback
// position and actual accumulated listening time.
//
// PATCH /api/v1/playback/sessions/{session_id}
func (h *PlaybackHandler) UpdateProgress(
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

	var input services.UpdatePlaybackProgressInput

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
		handlePlaybackError(
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

// CompleteSession marks a playback session complete.
//
// This endpoint is intentionally safe to retry.
//
// POST /api/v1/playback/sessions/{session_id}/complete
func (h *PlaybackHandler) CompleteSession(
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

	var input services.UpdatePlaybackProgressInput

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
		handlePlaybackError(
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

// GetListeningHistory returns the authenticated user's
// recently played music.
//
// GET /api/v1/me/listening-history?limit=20&offset=0
func (h *PlaybackHandler) GetListeningHistory(
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
		parsePlaybackHistoryQuery(r)

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
		h.Service.GetListeningHistory(
			r.Context(),
			userID,
			query,
		)

	if err != nil {
		handlePlaybackError(
			w,
			"GetListeningHistory",
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

func parsePlaybackHistoryQuery(
	r *http.Request,
) (services.PlaybackHistoryQuery, error) {
	var result services.PlaybackHistoryQuery

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

func decodePlaybackJSON(
	w http.ResponseWriter,
	r *http.Request,
	destination any,
) error {
	r.Body =
		http.MaxBytesReader(
			w,
			r.Body,
			maxPlaybackRequestBytes,
		)

	decoder :=
		json.NewDecoder(r.Body)

	decoder.DisallowUnknownFields()

	if err := decoder.Decode(
		destination,
	); err != nil {
		return errors.New(
			"invalid JSON request body",
		)
	}

	var extra any

	err := decoder.Decode(&extra)

	if !errors.Is(err, io.EOF) {
		return errors.New(
			"request body must contain exactly one JSON object",
		)
	}

	return nil
}

func handlePlaybackError(
	w http.ResponseWriter,
	operation string,
	err error,
) {
	switch {
	case services.IsPlaybackValidationError(
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
		repository.ErrPlaybackSessionNotFound,
	):
		WriteError(
			w,
			http.StatusNotFound,
			"PLAYBACK_SESSION_NOT_FOUND",
			"playback session not found",
		)

	case errors.Is(
		err,
		repository.ErrPlaybackSessionCompleted,
	):
		WriteError(
			w,
			http.StatusConflict,
			"PLAYBACK_SESSION_COMPLETED",
			"playback session is already completed",
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
