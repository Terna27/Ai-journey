package handler

import (
	"encoding/json"
	"errors"
	"net/http"

	"music-api/internal/service"
)

func (h *MusicHandler) CreateMusic(w http.ResponseWriter, r *http.Request) {
	var req CreateMusicRequest

	defer r.Body.Close()

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid JSON request body", http.StatusBadRequest)
		return
	}

	createdMusic, err := h.Service.CreateMusic(r.Context(), service.CreateMusicInput{
		ArtistName: req.ArtistName,
		SongTitle:  req.SongTitle,
		Genre:      req.Genre,
		ImageURL:   req.ImageURL,
	})
	if err != nil {
		if isValidationError(err) {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		http.Error(w, "failed to create music post", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)

	if err := json.NewEncoder(w).Encode(createdMusic); err != nil {
		http.Error(w, "failed to encode response", http.StatusInternalServerError)
		return
	}
}

// isValidationError reports whether err is one of the service's field-level
// validation errors, which map to 400 Bad Request.
func isValidationError(err error) bool {
	return errors.Is(err, service.ErrArtistNameRequired) ||
		errors.Is(err, service.ErrSongTitleRequired) ||
		errors.Is(err, service.ErrGenreRequired)
}
