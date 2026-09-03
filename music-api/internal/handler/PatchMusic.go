package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"music-api/internal/middleware"
	"music-api/internal/services"
)

func (h *MusicHandler) PatchMusic(w http.ResponseWriter, r *http.Request) {
	artistID, ok := middleware.ArtistIDFromContext(r.Context())
	if !ok {
		WriteError(
			w,
			http.StatusUnauthorized,
			"AUTHENTICATION_REQUIRED",
			"artist authentication is required",
		)
		return
	}

	id, err := strconv.Atoi(r.PathValue("id"))
	if err != nil || id < 1 {
		WriteError(
			w,
			http.StatusBadRequest,
			"INVALID_MUSIC_ID",
			"invalid music post ID",
		)
		return
	}

	var req UpdateMusicRequest

	defer r.Body.Close()

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		var maxBytesErr *http.MaxBytesError

		if errors.As(err, &maxBytesErr) {
			WriteError(
				w,
				http.StatusRequestEntityTooLarge,
				"REQUEST_TOO_LARGE",
				"request body is too large",
			)
			return
		}

		WriteError(
			w,
			http.StatusBadRequest,
			"INVALID_REQUEST",
			"invalid request body",
		)
		return
	}

	updatedMusic, err := h.Service.PatchMusic(
		r.Context(),
		id,
		artistID,
		services.UpdateMusicInput{
			ArtistName: req.ArtistName,
			SongTitle:  req.SongTitle,
			Genre:      req.Genre,
			ImageURL:   req.ImageURL,
			AudioKey:   req.AudioKey,
		},
	)
	if err != nil {
		switch {
		case isValidationError(err):
			WriteError(
				w,
				http.StatusBadRequest,
				"VALIDATION_ERROR",
				err.Error(),
			)

		case errors.Is(err, services.ErrUnauthorized):
			WriteError(
				w,
				http.StatusUnauthorized,
				"AUTHENTICATION_REQUIRED",
				"artist authentication is required",
			)

		case errors.Is(err, services.ErrMusicNotFound):
			WriteError(
				w,
				http.StatusNotFound,
				"MUSIC_NOT_FOUND",
				"music post not found",
			)

		default:
			WriteError(
				w,
				http.StatusInternalServerError,
				"INTERNAL_ERROR",
				"failed to update music post",
			)
		}

		return
	}

	WriteJSON(w, http.StatusOK, updatedMusic)
}
