package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v5"
)

func (h *NoteHandler) PatchNote(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")

	id, err := strconv.Atoi(idStr)
	if err != nil || id < 1 {
		http.Error(w, "invalid music post ID", http.StatusBadRequest)
		return
	}

	var req UpdateNoteRequest

	defer r.Body.Close()

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid JSON", http.StatusBadRequest)
		return
	}

	note, err := h.Repo.GetByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			http.Error(w, "music post not found", http.StatusNotFound)
			return
		}

		http.Error(w, "failed to get music post", http.StatusInternalServerError)
		return
	}

	if req.ArtistName != nil {
		artistName := strings.TrimSpace(*req.ArtistName)

		if artistName == "" {
			http.Error(w, "artist name cannot be empty", http.StatusBadRequest)
			return
		}

		note.ArtistName = artistName
	}

	if req.SongTitle != nil {
		songTitle := strings.TrimSpace(*req.SongTitle)

		if songTitle == "" {
			http.Error(w, "song title cannot be empty", http.StatusBadRequest)
			return
		}

		note.SongTitle = songTitle
	}

	if req.Genre != nil {
		genre := strings.TrimSpace(*req.Genre)

		if genre == "" {
			http.Error(w, "genre cannot be empty", http.StatusBadRequest)
			return
		}

		note.Genre = genre
	}

	if req.ImageURL != nil {
		note.ImageURL = strings.TrimSpace(*req.ImageURL)
	}

	updatedNote, err := h.Repo.Update(r.Context(), id, note)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			http.Error(w, "music post not found", http.StatusNotFound)
			return
		}

		http.Error(w, "failed to update music post", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	if err := json.NewEncoder(w).Encode(updatedNote); err != nil {
		http.Error(w, "failed to encode response", http.StatusInternalServerError)
		return
	}

}
