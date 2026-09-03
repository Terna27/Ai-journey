package handler

import (
	"errors"
	"net/http"
	"strconv"

	"music-api/internal/middleware"
	"music-api/internal/services"
)

func (h *MusicHandler) DeleteMusic(w http.ResponseWriter, r *http.Request) {
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

	err = h.Service.DeleteMusic(
		r.Context(),
		id,
		artistID,
	)
	if err != nil {
		switch {
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
				"failed to delete music post",
			)
		}

		return
	}

	w.WriteHeader(http.StatusNoContent)
}
