package handler

import (
	"errors"
	"net/http"
	"strconv"

	"music-api/internal/service"
)

func (h *MusicHandler) DeleteMusic(w http.ResponseWriter, r *http.Request) {

	idStr := r.PathValue("id")

	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "invalid music post ID", http.StatusBadRequest)
		return
	}

	err = h.Service.DeleteMusic(r.Context(), id)
	if err != nil {
		if errors.Is(err, service.ErrMusicNotFound) {
			http.Error(w, "music post not found", http.StatusNotFound)
			return
		}

		http.Error(w, "failed to delete music post", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
