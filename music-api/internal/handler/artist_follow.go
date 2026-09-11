package handler

import (
	"errors"
	"log"
	"net/http"
	"strconv"

	"music-api/internal/middleware"
	"music-api/internal/services"
)

type ArtistFollowHandler struct {
	Service *services.ArtistFollowService
}

func NewArtistFollowHandler(
	service *services.ArtistFollowService,
) *ArtistFollowHandler {
	return &ArtistFollowHandler{
		Service: service,
	}
}

func (h *ArtistFollowHandler) Follow(
	w http.ResponseWriter,
	r *http.Request,
) {
	userID, ok :=
		middleware.UserIDFromContext(
			r.Context(),
		)

	if !ok {
		WriteError(
			w,
			http.StatusUnauthorized,
			"AUTHENTICATION_REQUIRED",
			"user authentication is required",
		)
		return
	}

	artistID, err := parseFollowArtistID(r)
	if err != nil {
		WriteError(
			w,
			http.StatusBadRequest,
			"INVALID_ARTIST_ID",
			"invalid artist ID",
		)
		return
	}

	status, err := h.Service.Follow(
		r.Context(),
		userID,
		artistID,
	)
	if err != nil {
		handleArtistFollowError(
			w,
			"follow artist",
			err,
		)
		return
	}

	WriteJSON(
		w,
		http.StatusOK,
		status,
	)
}

func (h *ArtistFollowHandler) Unfollow(
	w http.ResponseWriter,
	r *http.Request,
) {
	userID, ok :=
		middleware.UserIDFromContext(
			r.Context(),
		)

	if !ok {
		WriteError(
			w,
			http.StatusUnauthorized,
			"AUTHENTICATION_REQUIRED",
			"user authentication is required",
		)
		return
	}

	artistID, err := parseFollowArtistID(r)
	if err != nil {
		WriteError(
			w,
			http.StatusBadRequest,
			"INVALID_ARTIST_ID",
			"invalid artist ID",
		)
		return
	}

	status, err := h.Service.Unfollow(
		r.Context(),
		userID,
		artistID,
	)
	if err != nil {
		handleArtistFollowError(
			w,
			"unfollow artist",
			err,
		)
		return
	}

	WriteJSON(
		w,
		http.StatusOK,
		status,
	)
}

func (h *ArtistFollowHandler) GetFollowerCount(
	w http.ResponseWriter,
	r *http.Request,
) {
	artistID, err := parseFollowArtistID(r)
	if err != nil {
		WriteError(
			w,
			http.StatusBadRequest,
			"INVALID_ARTIST_ID",
			"invalid artist ID",
		)
		return
	}

	count, err := h.Service.GetFollowerCount(
		r.Context(),
		artistID,
	)
	if err != nil {
		handleArtistFollowError(
			w,
			"get artist follower count",
			err,
		)
		return
	}

	WriteJSON(
		w,
		http.StatusOK,
		map[string]int64{
			"follower_count": count,
		},
	)
}

func (h *ArtistFollowHandler) GetStatus(
	w http.ResponseWriter,
	r *http.Request,
) {
	userID, ok :=
		middleware.UserIDFromContext(
			r.Context(),
		)

	if !ok {
		WriteError(
			w,
			http.StatusUnauthorized,
			"AUTHENTICATION_REQUIRED",
			"user authentication is required",
		)
		return
	}

	artistID, err := parseFollowArtistID(r)
	if err != nil {
		WriteError(
			w,
			http.StatusBadRequest,
			"INVALID_ARTIST_ID",
			"invalid artist ID",
		)
		return
	}

	status, err := h.Service.GetStatus(
		r.Context(),
		userID,
		artistID,
	)
	if err != nil {
		handleArtistFollowError(
			w,
			"get artist follow status",
			err,
		)
		return
	}

	WriteJSON(
		w,
		http.StatusOK,
		status,
	)
}

func parseFollowArtistID(
	r *http.Request,
) (int, error) {
	artistID, err := strconv.Atoi(
		r.PathValue("id"),
	)
	if err != nil || artistID <= 0 {
		return 0,
			services.ErrInvalidFollowArtistID
	}

	return artistID, nil
}

func handleArtistFollowError(
	w http.ResponseWriter,
	operation string,
	err error,
) {
	switch {
	case errors.Is(
		err,
		services.ErrInvalidFollowUserID,
	):
		WriteError(
			w,
			http.StatusUnauthorized,
			"AUTHENTICATION_REQUIRED",
			"user authentication is required",
		)

	case errors.Is(
		err,
		services.ErrInvalidFollowArtistID,
	):
		WriteError(
			w,
			http.StatusBadRequest,
			"INVALID_ARTIST_ID",
			"invalid artist ID",
		)

	case errors.Is(
		err,
		services.ErrFollowArtistNotFound,
	):
		WriteError(
			w,
			http.StatusNotFound,
			"ARTIST_NOT_FOUND",
			"artist not found",
		)

	case errors.Is(
		err,
		services.ErrCannotFollowOwnArtist,
	):
		WriteError(
			w,
			http.StatusBadRequest,
			"CANNOT_FOLLOW_OWN_ARTIST",
			"you cannot follow your own artist profile",
		)

	default:
		log.Printf(
			"%s failed: %v",
			operation,
			err,
		)

		WriteError(
			w,
			http.StatusInternalServerError,
			"INTERNAL_ERROR",
			"artist follow operation failed",
		)
	}
}
