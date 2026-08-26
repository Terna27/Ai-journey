
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

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(
			w,
			"invalid request body",
			http.StatusBadRequest,
		)
		return
	}

	req.Email = strings.ToLower(strings.TrimSpace(req.Email))

	if req.Email == "" || req.Password == "" {
		http.Error(
			w,
			"email and password are required",
			http.StatusBadRequest,
		)
		return
	}

	artist, err := h.ArtistService.GetByEmail(
		r.Context(),
		req.Email,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			http.Error(
				w,
				"invalid email or password",
				http.StatusUnauthorized,
			)
			return
		}

		http.Error(
			w,
			"failed to login",
			http.StatusInternalServerError,
		)
		return
	}

	if err := bcrypt.CompareHashAndPassword(
		[]byte(artist.PasswordHash),
		[]byte(req.Password),
	); err != nil {
		http.Error(
			w,
			"invalid email or password",
			http.StatusUnauthorized,
		)
		return
	}

	token, err := h.JWTService.GenerateToken(
		artist.ID,
		artist.Email,
	)

	if err != nil {
		http.Error(
			w,
			"failed to generate token",
			http.StatusInternalServerError,
		)
		return
	}

	response := LoginArtistResponse{
		Token: token,
	}

	response.Artist.ID = artist.ID
	response.Artist.Name = artist.Name
	response.Artist.Email = artist.Email

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	if err := json.NewEncoder(w).Encode(response); err != nil {
		return
	}
}

