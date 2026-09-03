package handler

import (
	"errors"
	"net/http"
	"strconv"

	"music-api/internal/services"
)

func (h *MusicHandler) GetMusic(w http.ResponseWriter, r *http.Request) {
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

	music, err := h.Service.GetMusic(r.Context(), id)
	if err != nil {
		if errors.Is(err, services.ErrMusicNotFound) {
			WriteError(
				w,
				http.StatusNotFound,
				"MUSIC_NOT_FOUND",
				"music post not found",
			)
			return
		}

		WriteError(
			w,
			http.StatusInternalServerError,
			"INTERNAL_ERROR",
			"failed to retrieve music post",
		)
		return
	}

	WriteJSON(w, http.StatusOK, music)
}
