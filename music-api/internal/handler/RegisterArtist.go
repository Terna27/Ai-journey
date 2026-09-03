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

	defer r.Body.Close()

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		var maxBytesErr *http.MaxBytesError

		if errors.As(err, &maxBytesErr) {
			WriteError(
				w,
				http.StatusRequestEntityTooLarge,
				"REQUEST_TOO_LARGE",
				"request body is too large",
			)
			return
		}

		WriteError(
			w,
			http.StatusBadRequest,
			"INVALID_REQUEST",
			"invalid request body",
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
		case errors.Is(err, services.ErrArtistNameRequired),
			errors.Is(err, services.ErrArtistEmailRequired),
			errors.Is(err, services.ErrArtistPasswordRequired),
			errors.Is(err, services.ErrArtistPasswordTooShort):

			WriteError(
				w,
				http.StatusBadRequest,
				"VALIDATION_ERROR",
				err.Error(),
			)

		case errors.Is(err, pgx.ErrNoRows):
			WriteError(
				w,
				http.StatusNotFound,
				"ARTIST_NOT_FOUND",
				"artist not found",
			)

		default:
			log.Printf("RegisterArtist failed: %v", err)

			WriteError(
				w,
				http.StatusInternalServerError,
				"INTERNAL_ERROR",
				"failed to create artist",
			)
		}

		return
	}

	WriteJSON(w, http.StatusCreated, artist)
}
