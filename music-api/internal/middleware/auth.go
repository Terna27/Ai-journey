package middleware

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"music-api/internal/services"
)

type authContextKey string

const artistIDKey authContextKey = "artist_id"

type authErrorResponse struct {
	Error authAPIError `json:"error"`
}

type authAPIError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func JWTAuth(
	jwtService *services.JWTService,
	next http.Handler,
) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := strings.TrimSpace(
			r.Header.Get("Authorization"),
		)

		if authHeader == "" {
			writeAuthError(
				w,
				http.StatusUnauthorized,
				"AUTHENTICATION_REQUIRED",
				"authorization token is required",
			)
			return
		}

		parts := strings.Fields(authHeader)

		if len(parts) != 2 ||
			!strings.EqualFold(parts[0], "Bearer") ||
			parts[1] == "" {

			writeAuthError(
				w,
				http.StatusUnauthorized,
				"INVALID_AUTHORIZATION_HEADER",
				"authorization header must use Bearer token",
			)
			return
		}

		artistID, err := jwtService.ValidateToken(parts[1])
		if err != nil {
			writeAuthError(
				w,
				http.StatusUnauthorized,
				"INVALID_TOKEN",
				"invalid or expired authentication token",
			)
			return
		}

		// Make the authenticated artist visible in request logs.
		SetUserID(
			r.Context(),
			strconv.Itoa(artistID),
		)

		ctx := context.WithValue(
			r.Context(),
			artistIDKey,
			artistID,
		)

		next.ServeHTTP(
			w,
			r.WithContext(ctx),
		)
	})
}

func ArtistIDFromContext(ctx context.Context) (int, bool) {
	artistID, ok := ctx.Value(artistIDKey).(int)

	if !ok || artistID <= 0 {
		return 0, false
	}

	return artistID, true
}

func writeAuthError(
	w http.ResponseWriter,
	status int,
	code string,
	message string,
) {
	w.Header().Set(
		"Content-Type",
		"application/json",
	)

	w.WriteHeader(status)

	_ = json.NewEncoder(w).Encode(
		authErrorResponse{
			Error: authAPIError{
				Code:    code,
				Message: message,
			},
		},
	)
}
