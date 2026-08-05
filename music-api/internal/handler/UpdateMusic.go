package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"music-api/internal/models"

	"github.com/jackc/pgx/v5"
)

func (h *MusicHandler) UpdateMusic(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")

	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "invalid music post ID", http.StatusBadRequest)
		return
	}

	var req CreateMusicRequest

	defer r.Body.Close()

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid JSON", http.StatusBadRequest)
		return
	}

	if strings.TrimSpace(req.ArtistName) == "" {
		http.Error(w, "artist name cannot be empty", http.StatusBadRequest)
		return
	}

	if strings.TrimSpace(req.SongTitle) == "" {
		http.Error(w, "song title cannot be empty", http.StatusBadRequest)
		return
	}

	if strings.TrimSpace(req.Genre) == "" {
		http.Error(w, "genre cannot be empty", http.StatusBadRequest)
		return
	}

	music := models.Music{
		ArtistName: strings.TrimSpace(req.ArtistName),
		SongTitle:  strings.TrimSpace(req.SongTitle),
		Genre:      strings.TrimSpace(req.Genre),
		ImageURL:   strings.TrimSpace(req.ImageURL),
	}

	updatedMusic, err := h.Repo.Update(r.Context(), id, music)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			http.Error(w, "music post not found", http.StatusNotFound)
			return
		}

		http.Error(w, "failed to update music post", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	if err := json.NewEncoder(w).Encode(updatedMusic); err != nil {
		http.Error(w, "failed to encode response", http.StatusInternalServerError)
		return
	}

}
