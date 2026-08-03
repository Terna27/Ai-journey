package middleware

import (
	"net/http"
	"os"
)

func APIKey(next http.Handler) http.Handler {
	expectedKey := os.Getenv("API_KEY")

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		providedKey := r.Header.Get("X-API-Key")

		if expectedKey == "" {
			http.Error(
				w,
				"API key is not configured",
				http.StatusInternalServerError,
			)
			return
		}

		if providedKey == "" {
			http.Error(
				w,
				"API key is required",
				http.StatusUnauthorized,
			)
			return
		}

		if providedKey != expectedKey {
			http.Error(
				w,
				"invalid API key",
				http.StatusUnauthorized,
			)
			return
		}

		next.ServeHTTP(w, r)
	})
}