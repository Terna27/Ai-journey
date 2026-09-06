package handler

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"

	"music-api/internal/services"
)

type VerifyEmailRequest struct {
	Token string `json:"token"`
}

func (h *MusicHandler) VerifyEmail(
	w http.ResponseWriter,
	r *http.Request,
) {
	var req VerifyEmailRequest

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

	err := h.EmailVerificationService.Verify(
		r.Context(),
		req.Token,
	)
	if err != nil {
		switch {
		case errors.Is(
			err,
			services.ErrVerificationTokenInvalid,
		):
			WriteError(
				w,
				http.StatusBadRequest,
				"INVALID_VERIFICATION_TOKEN",
				"verification link is invalid or expired",
			)

		case errors.Is(
			err,
			services.ErrEmailAlreadyVerified,
		):
			WriteJSON(
				w,
				http.StatusOK,
				map[string]string{
					"message": "email address is already verified",
				},
			)

		default:
			log.Printf(
				"VerifyEmail failed: %v",
				err,
			)

			WriteError(
				w,
				http.StatusInternalServerError,
				"INTERNAL_ERROR",
				"failed to verify email address",
			)
		}

		return
	}

	WriteJSON(
		w,
		http.StatusOK,
		map[string]string{
			"message": "email address verified successfully",
		},
	)
}
