package handler

import (
	"encoding/json"
	"net/http"
	"strings"

	"notes-Api/internal/models"
)

func (h *NoteHandler) CreateNote(w http.ResponseWriter, r *http.Request) {
	var req CreateNoteRequest

	defer r.Body.Close()

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid JSON request body", http.StatusBadRequest)
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

	note := models.Note{
		ArtistName: strings.TrimSpace(req.ArtistName),
		SongTitle:  strings.TrimSpace(req.SongTitle),
		Genre:      strings.TrimSpace(req.Genre),
		ImageURL:   strings.TrimSpace(req.ImageURL),
	}

	createdNote, err := h.Repo.Create(r.Context(), note)
	if err != nil {
		http.Error(w, "failed to create music post", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)

	if err := json.NewEncoder(w).Encode(createdNote); err != nil {
		http.Error(w, "failed to encode response", http.StatusInternalServerError)
		return
	}

}
