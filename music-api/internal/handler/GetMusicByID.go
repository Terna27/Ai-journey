package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/jackc/pgx/v5"
)

func (h *MusicHandler) GetMusic(w http.ResponseWriter, r *http.Request) {

	idStr := r.PathValue("id")

	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "invalid music post ID", http.StatusBadRequest)
		return
	}

	music, err := h.Repo.GetByID(r.Context(), id)
	if err != nil {

		if errors.Is(err, pgx.ErrNoRows) {
			http.Error(w, "music post not found", http.StatusNotFound)
			return
		}

		http.Error(w, "failed to retrieve music post", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	if err := json.NewEncoder(w).Encode(music); err != nil {
		http.Error(w, "failed to encode response", http.StatusInternalServerError)
	}
}
