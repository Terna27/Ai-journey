package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"music-api/internal/service"
)

func (h *MusicHandler) PatchMusic(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")

	id, err := strconv.Atoi(idStr)
	if err != nil || id < 1 {
		http.Error(w, "invalid music post ID", http.StatusBadRequest)
		return
	}

	var req UpdateMusicRequest

	defer r.Body.Close()

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid JSON", http.StatusBadRequest)
		return
	}

	updatedMusic, err := h.Service.PatchMusic(r.Context(), id, service.UpdateMusicInput{
		ArtistName: req.ArtistName,
		SongTitle:  req.SongTitle,
		Genre:      req.Genre,
		ImageURL:   req.ImageURL,
	})
	if err != nil {
		switch {
		case isValidationError(err):
			http.Error(w, err.Error(), http.StatusBadRequest)
		case errors.Is(err, service.ErrMusicNotFound):
			http.Error(w, "music post not found", http.StatusNotFound)
		default:
			http.Error(w, "failed to update music post", http.StatusInternalServerError)
		}
		return
	}

	w.Header().Set("Content-Type", "application/json")

	if err := json.NewEncoder(w).Encode(updatedMusic); err != nil {
		http.Error(w, "failed to encode response", http.StatusInternalServerError)
		return
	}
}
