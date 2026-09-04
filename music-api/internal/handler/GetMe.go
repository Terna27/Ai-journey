package handler

import (
	"errors"
	"log"
	"net/http"

	"music-api/internal/middleware"
	"music-api/internal/models"

	"github.com/jackc/pgx/v5"
)

type MeResponse struct {
	User   models.User    `json:"user"`
	Artist *models.Artist `json:"artist"`
}

func (h *MusicHandler) GetMe(
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
			"GetMe: failed to load user %d: %v",
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
	artist, err := h.ArtistService.GetByUserID(
		r.Context(),
		userID,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			WriteJSON(
				w,
				http.StatusOK,
				MeResponse{
					User:   user,
					Artist: nil,
				},
			)
			return
		}

		log.Printf(
			"GetMe: failed to load artist profile for user %d: %v",
			userID,
			err,
		)

		WriteError(
			w,
			http.StatusInternalServerError,
			"INTERNAL_ERROR",
			"failed to load artist profile",
		)
		return
	}

	WriteJSON(
		w,
		http.StatusOK,
		MeResponse{
			User:   user,
			Artist: &artist,
		},
	)
}
