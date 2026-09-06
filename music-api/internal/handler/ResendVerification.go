package handler

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strings"

	"music-api/internal/services"

	"github.com/jackc/pgx/v5"
)

type ResendVerificationRequest struct {
	Email string `json:"email"`
}

func (h *MusicHandler) ResendVerification(
	w http.ResponseWriter,
	r *http.Request,
) {
	var req ResendVerificationRequest

	defer r.Body.Close()

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteError(
			w,
			http.StatusBadRequest,
			"INVALID_REQUEST",
			"invalid request body",
		)
		return
	}

	email := strings.ToLower(
		strings.TrimSpace(req.Email),
	)

	if email == "" {
		WriteError(
			w,
			http.StatusBadRequest,
			"VALIDATION_ERROR",
			"email is required",
		)
		return
	}

	user, err := h.UserService.GetByEmail(
		r.Context(),
		email,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			writeResendVerificationSuccess(w)
			return
		}

		log.Printf(
			"ResendVerification: failed to load user: %v",
			err,
		)

		WriteError(
			w,
			http.StatusInternalServerError,
			"INTERNAL_ERROR",
			"failed to process verification request",
		)
		return
	}

	if user.EmailVerified {
		WriteJSON(
			w,
			http.StatusOK,
			map[string]string{
				"message": "email address is already verified",
			},
		)
		return
	}

	if err := h.EmailVerificationService.SendVerification(
		r.Context(),
		user,
	); err != nil {
		if errors.Is(
			err,
			services.ErrEmailAlreadyVerified,
		) {
			WriteJSON(
				w,
				http.StatusOK,
				map[string]string{
					"message": "email address is already verified",
				},
			)
			return
		}

		log.Printf(
			"ResendVerification: failed for user %d: %v",
			user.ID,
			err,
		)

		WriteError(
			w,
			http.StatusServiceUnavailable,
			"VERIFICATION_EMAIL_FAILED",
			"verification email could not be sent; please try again",
		)
		return
	}

	writeResendVerificationSuccess(w)
}

func writeResendVerificationSuccess(
	w http.ResponseWriter,
) {
	WriteJSON(
		w,
		http.StatusOK,
		map[string]string{
			"message": "if the account exists and requires verification, a verification email has been sent",
		},
	)
}
