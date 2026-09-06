package handler

import (
	"errors"
	"net/http"
	"strconv"

	"music-api/internal/middleware"
	"music-api/internal/services"
)

// LikeMusic records a like for the authenticated user.
func (h *MusicHandler) LikeMusic(
	w http.ResponseWriter,
	r *http.Request,
) {
	musicID, err := strconv.Atoi(
		r.PathValue("id"),
	)
	if err != nil || musicID < 1 {
		WriteError(
			w,
			http.StatusBadRequest,
			"INVALID_MUSIC_ID",
			"invalid music post ID",
		)
		return
	}

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

	updatedMusic, err :=
		h.Service.LikeMusicForUser(
			r.Context(),
			userID,
			musicID,
		)

	if err != nil {
		switch {
		case errors.Is(
			err,
			services.ErrMusicNotFound,
		):
			WriteError(
				w,
				http.StatusNotFound,
				"MUSIC_NOT_FOUND",
				"music post not found",
			)

		case errors.Is(
			err,
			services.ErrAlreadyLiked,
		):
			WriteError(
				w,
				http.StatusConflict,
				"ALREADY_LIKED",
				"you have already liked this music post",
			)

		case errors.Is(
			err,
			services.ErrUserAuthenticationRequired,
		):
			WriteError(
				w,
				http.StatusUnauthorized,
				"AUTHENTICATION_REQUIRED",
				"user authentication is required",
			)

		default:
			WriteError(
				w,
				http.StatusInternalServerError,
				"INTERNAL_ERROR",
				"failed to like music post",
			)
		}

		return
	}

	WriteJSON(
		w,
		http.StatusOK,
		updatedMusic,
	)
}

// UnlikeMusic removes the authenticated user's like.
func (h *MusicHandler) UnlikeMusic(
	w http.ResponseWriter,
	r *http.Request,
) {
	musicID, err := strconv.Atoi(
		r.PathValue("id"),
	)
	if err != nil || musicID < 1 {
		WriteError(
			w,
			http.StatusBadRequest,
			"INVALID_MUSIC_ID",
			"invalid music post ID",
		)
		return
	}

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

	updatedMusic, err :=
		h.Service.UnlikeMusicForUser(
			r.Context(),
			userID,
			musicID,
		)

	if err != nil {
		switch {
		case errors.Is(
			err,
			services.ErrMusicNotFound,
		):
			WriteError(
				w,
				http.StatusNotFound,
				"MUSIC_NOT_FOUND",
				"music post not found",
			)

		case errors.Is(
			err,
			services.ErrNotLiked,
		):
			WriteError(
				w,
				http.StatusConflict,
				"NOT_LIKED",
				"you have not liked this music post",
			)

		case errors.Is(
			err,
			services.ErrUserAuthenticationRequired,
		):
			WriteError(
				w,
				http.StatusUnauthorized,
				"AUTHENTICATION_REQUIRED",
				"user authentication is required",
			)

		default:
			WriteError(
				w,
				http.StatusInternalServerError,
				"INTERNAL_ERROR",
				"failed to unlike music post",
			)
		}

		return
	}

	WriteJSON(
		w,
		http.StatusOK,
		updatedMusic,
	)
}

// GetLikedMusic returns the authenticated user's liked-song library.
func (h *MusicHandler) GetLikedMusic(
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

	musicList, err :=
		h.Service.GetLikedMusic(
			r.Context(),
			userID,
		)

	if err != nil {
		if errors.Is(
			err,
			services.ErrUserAuthenticationRequired,
		) {
			WriteError(
				w,
				http.StatusUnauthorized,
				"AUTHENTICATION_REQUIRED",
				"user authentication is required",
			)
			return
		}

		WriteError(
			w,
			http.StatusInternalServerError,
			"INTERNAL_ERROR",
			"failed to load liked music",
		)
		return
	}

	WriteJSON(
		w,
		http.StatusOK,
		musicList,
	)
}
