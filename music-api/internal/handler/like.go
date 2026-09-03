package handler

import (
	"errors"
	"net/http"
	"strconv"

	"music-api/internal/middleware"
	"music-api/internal/services"
)

func (h *MusicHandler) LikeMusic(w http.ResponseWriter, r *http.Request) {
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

	likerID := middleware.GetClientIP(r)

	updatedMusic, err := h.Service.LikeMusic(
		r.Context(),
		id,
		likerID,
	)
	if err != nil {
		switch {
		case errors.Is(err, services.ErrMusicNotFound):
			WriteError(
				w,
				http.StatusNotFound,
				"MUSIC_NOT_FOUND",
				"music post not found",
			)

		case errors.Is(err, services.ErrAlreadyLiked):
			WriteError(
				w,
				http.StatusConflict,
				"ALREADY_LIKED",
				"you have already liked this music post",
			)

		default:
			WriteError(
				w,
				http.StatusInternalServerError,
				"INTERNAL_ERROR",
				"failed to like music post",
			)
		}

		return
	}

	WriteJSON(w, http.StatusOK, updatedMusic)
}
