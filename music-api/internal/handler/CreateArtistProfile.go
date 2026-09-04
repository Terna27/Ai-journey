package handler

import (
	"errors"
	"log"
	"net/http"

	"music-api/internal/middleware"
	"music-api/internal/services"

	"github.com/jackc/pgx/v5"
)

func (h *MusicHandler) CreateArtistProfile(
	w http.ResponseWriter,
	r *http.Request,
) {
	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		WriteError(
			w,
			http.StatusUnauthorized,
			"AUTHENTICATION_REQUIRED",
			"user authentication is required",
		)
		return
	}

	user, err := h.UserService.GetByID(
		r.Context(),
		userID,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			WriteError(
				w,
				http.StatusUnauthorized,
				"USER_NOT_FOUND",
				"authenticated user no longer exists",
			)
			return
		}

		log.Printf(
			"CreateArtistProfile: failed to load user %d: %v",
			userID,
			err,
		)

		WriteError(
			w,
			http.StatusInternalServerError,
			"INTERNAL_ERROR",
			"failed to load user account",
		)
		return
	}

	artist, err := h.ArtistService.CreateForUser(
		r.Context(),
		user,
	)
	if err != nil {
		switch {
		case errors.Is(err, services.ErrArtistProfileExists):
			WriteError(
				w,
				http.StatusConflict,
				"ARTIST_PROFILE_EXISTS",
				"this account already has an artist profile",
			)

		default:
			log.Printf(
				"CreateArtistProfile: failed for user %d: %v",
				userID,
				err,
			)

			WriteError(
				w,
				http.StatusInternalServerError,
				"INTERNAL_ERROR",
				"failed to create artist profile",
			)
		}

		return
	}

	WriteJSON(
		w,
		http.StatusCreated,
		artist,
	)
}
