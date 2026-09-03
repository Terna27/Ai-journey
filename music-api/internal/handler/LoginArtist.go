package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/jackc/pgx/v5"
	"golang.org/x/crypto/bcrypt"
)

type LoginArtistRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type LoginArtistResponse struct {
	Token  string `json:"token"`
	Artist struct {
		ID    int    `json:"id"`
		Name  string `json:"name"`
		Email string `json:"email"`
	} `json:"artist"`
}

func (h *MusicHandler) LoginArtist(w http.ResponseWriter, r *http.Request) {
	var req LoginArtistRequest

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

	req.Email = strings.ToLower(strings.TrimSpace(req.Email))

	if req.Email == "" || req.Password == "" {
		WriteError(
			w,
			http.StatusBadRequest,
			"VALIDATION_ERROR",
			"email and password are required",
		)
		return
	}

	artist, err := h.ArtistService.GetByEmail(
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
		[]byte(artist.PasswordHash),
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

	token, err := h.JWTService.GenerateToken(
		artist.ID,
		artist.Email,
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

	response := LoginArtistResponse{
		Token: token,
	}

	response.Artist.ID = artist.ID
	response.Artist.Name = artist.Name
	response.Artist.Email = artist.Email

	WriteJSON(w, http.StatusOK, response)
}
