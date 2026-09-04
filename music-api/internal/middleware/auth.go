package middleware

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"music-api/internal/repository"
	"music-api/internal/services"

	"github.com/jackc/pgx/v5"
)

type authContextKey string

const (
	userIDKey   authContextKey = "user_id"
	artistIDKey authContextKey = "artist_id"
)

type authErrorResponse struct {
	Error authAPIError `json:"error"`
}

type authAPIError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// JWTAuth authenticates both the new user tokens and temporary legacy
// artist tokens.
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

		identity, err := jwtService.ValidateIdentity(parts[1])
		if err != nil {
			writeAuthError(
				w,
				http.StatusUnauthorized,
				"INVALID_TOKEN",
				"invalid or expired authentication token",
			)
			return
		}

		ctx := r.Context()

		if identity.UserID > 0 {
			SetUserID(
				ctx,
				strconv.Itoa(identity.UserID),
			)

			ctx = context.WithValue(
				ctx,
				userIDKey,
				identity.UserID,
			)
		}

		// Temporary support for old artist JWTs.
		if identity.ArtistID > 0 {
			SetUserID(
				ctx,
				strconv.Itoa(identity.ArtistID),
			)

			ctx = context.WithValue(
				ctx,
				artistIDKey,
				identity.ArtistID,
			)
		}

		next.ServeHTTP(
			w,
			r.WithContext(ctx),
		)
	})
}

// RequireArtist converts the authenticated user identity into an artist
// identity for artist-only routes.
//
// Legacy artist tokens already contain artist_id and pass through.
func RequireArtist(
	artistRepo *repository.ArtistRepository,
	next http.Handler,
) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Temporary legacy compatibility.
		if _, ok := ArtistIDFromContext(r.Context()); ok {
			next.ServeHTTP(w, r)
			return
		}

		userID, ok := UserIDFromContext(r.Context())
		if !ok {
			writeAuthError(
				w,
				http.StatusUnauthorized,
				"AUTHENTICATION_REQUIRED",
				"user authentication is required",
			)
			return
		}

		artist, err := artistRepo.GetByUserID(
			r.Context(),
			userID,
		)
		if err != nil {
			if err == pgx.ErrNoRows {
				writeAuthError(
					w,
					http.StatusForbidden,
					"ARTIST_PROFILE_REQUIRED",
					"an artist profile is required for this action",
				)
				return
			}

			writeAuthError(
				w,
				http.StatusInternalServerError,
				"INTERNAL_ERROR",
				"failed to verify artist profile",
			)
			return
		}

		ctx := context.WithValue(
			r.Context(),
			artistIDKey,
			artist.ID,
		)

		next.ServeHTTP(
			w,
			r.WithContext(ctx),
		)
	})
}

func UserIDFromContext(ctx context.Context) (int, bool) {
	userID, ok := ctx.Value(userIDKey).(int)

	if !ok || userID <= 0 {
		return 0, false
	}

	return userID, true
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
