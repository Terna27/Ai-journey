package handler

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"

	"music-api/internal/services"
)

type RegisterUserRequest struct {
	Name     string `json:"name"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

func (h *MusicHandler) RegisterUser(
	w http.ResponseWriter,
	r *http.Request,
) {
	var req RegisterUserRequest

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

	user, err := h.UserService.Register(
		r.Context(),
		req.Name,
		req.Email,
		req.Password,
	)
	if err != nil {
		switch {
		case errors.Is(err, services.ErrUserNameRequired),
			errors.Is(err, services.ErrUserEmailRequired),
			errors.Is(err, services.ErrUserPasswordRequired),
			errors.Is(err, services.ErrUserPasswordTooShort):

			WriteError(
				w,
				http.StatusBadRequest,
				"VALIDATION_ERROR",
				err.Error(),
			)

		default:
			log.Printf("RegisterUser failed: %v", err)

			WriteError(
				w,
				http.StatusInternalServerError,
				"INTERNAL_ERROR",
				"failed to create user",
			)
		}

		return
	}

	WriteJSON(
		w,
		http.StatusCreated,
		user,
	)
}