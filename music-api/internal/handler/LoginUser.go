package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/jackc/pgx/v5"
	"golang.org/x/crypto/bcrypt"
)

type LoginUserRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type LoginUserResponse struct {
	Token string `json:"token"`

	User struct {
		ID    int    `json:"id"`
		Name  string `json:"name"`
		Email string `json:"email"`
	} `json:"user"`
}

func (h *MusicHandler) LoginUser(
	w http.ResponseWriter,
	r *http.Request,
) {
	var req LoginUserRequest

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

	req.Email = strings.ToLower(
		strings.TrimSpace(req.Email),
	)

	if req.Email == "" || req.Password == "" {
		WriteError(
			w,
			http.StatusBadRequest,
			"VALIDATION_ERROR",
			"email and password are required",
		)
		return
	}

	user, err := h.UserService.GetByEmail(
		r.Context(),
		req.Email,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			WriteError(
				w,
				http.StatusUnauthorized,
				"INVALID_CREDENTIALS",
				"invalid email or password",
			)
			return
		}

		WriteError(
			w,
			http.StatusInternalServerError,
			"INTERNAL_ERROR",
			"failed to login",
		)
		return
	}

	if err := bcrypt.CompareHashAndPassword(
		[]byte(user.PasswordHash),
		[]byte(req.Password),
	); err != nil {
		WriteError(
			w,
			http.StatusUnauthorized,
			"INVALID_CREDENTIALS",
			"invalid email or password",
		)
		return
	}

	token, err := h.JWTService.GenerateUserToken(
		user.ID,
		user.Email,
	)
	if err != nil {
		WriteError(
			w,
			http.StatusInternalServerError,
			"INTERNAL_ERROR",
			"failed to login",
		)
		return
	}

	response := LoginUserResponse{
		Token: token,
	}

	response.User.ID = user.ID
	response.User.Name = user.Name
	response.User.Email = user.Email

	WriteJSON(
		w,
		http.StatusOK,
		response,
	)
}
