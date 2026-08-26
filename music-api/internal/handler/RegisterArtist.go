package handler

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"

	"music-api/internal/services"

	"github.com/jackc/pgx/v5"
)

type RegisterArtistRequest struct {
	Name     string `json:"name"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

func (h *MusicHandler) RegisterArtist(w http.ResponseWriter, r *http.Request) {

	var req RegisterArtistRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(
			w,
			"invalid request body",
			http.StatusBadRequest,
		)
		return
	}

	artist, err := h.ArtistService.Register(
		r.Context(),
		req.Name,
		req.Email,
		req.Password,
	)

	if err != nil {

		switch {
		case errors.Is(err, services.ErrArtistNameRequired):
			http.Error(w, err.Error(), http.StatusBadRequest)

		case errors.Is(err, services.ErrArtistEmailRequired):
			http.Error(w, err.Error(), http.StatusBadRequest)

		case errors.Is(err, services.ErrArtistPasswordRequired):
			http.Error(w, err.Error(), http.StatusBadRequest)

		case errors.Is(err, services.ErrArtistPasswordTooShort):
			http.Error(w, err.Error(), http.StatusBadRequest)

		case errors.Is(err, pgx.ErrNoRows):
			http.Error(w, "artist not found", http.StatusNotFound)

		default:
			log.Printf("RegisterArtist failed: %v", err)

			http.Error(
				w,
				"failed to create artist",
				http.StatusInternalServerError,
			)
		}

		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)

	if err := json.NewEncoder(w).Encode(artist); err != nil {
		return
	}
}
