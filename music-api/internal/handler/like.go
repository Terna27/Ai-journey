package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"music-api/internal/middleware"
	"music-api/internal/services"
)

func (h *MusicHandler) LikeMusic(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")

	id, err := strconv.Atoi(idStr)
	if err != nil || id < 1 {
		http.Error(w, "invalid music post ID", http.StatusBadRequest)
		return
	}

	// Identify the caller so the "one like per caller" rule has something to
	// key on. There is no per-user auth yet, so we use the client IP; when
	// real users exist this becomes the authenticated user id.
	likerID := middleware.GetClientIP(r)

	updatedMusic, err := h.Service.LikeMusic(r.Context(), id, likerID)
	if err != nil {
		switch {
		case errors.Is(err, services.ErrMusicNotFound):
			http.Error(w, "music post not found", http.StatusNotFound)
		case errors.Is(err, services.ErrAlreadyLiked):
			http.Error(w, "you have already liked this music post", http.StatusConflict)
		default:
			http.Error(w, "failed to like music post", http.StatusInternalServerError)
		}
		return
	}

	w.Header().Set("Content-Type", "application/json")

	if err := json.NewEncoder(w).Encode(updatedMusic); err != nil {
		http.Error(
			w,
			"failed to encode response",
			http.StatusInternalServerError,
		)
		return
	}
}
