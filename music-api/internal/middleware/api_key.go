package middleware

import (
	"net"
	"net/http"
	"os"
	"strings"
)

func GetClientIP(r *http.Request) string {
	forwardedFor := r.Header.Get("X-Forwarded-For")

	if forwardedFor != "" {
		ips := strings.Split(forwardedFor, ",")

		return strings.TrimSpace(ips[0])
	}

	realIP := r.Header.Get("X-Real-IP")

	if realIP != "" {
		return strings.TrimSpace(realIP)
	}

	ip, _, err := net.SplitHostPort(r.RemoteAddr)

	if err != nil {
		return r.RemoteAddr
	}

	if ip == "::1" || ip == "127.0.0.1" {
		return ip
	}

	return ip
}

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

		// The request is authenticated. With a single shared key there is no
		// distinct user yet, so we tag the caller generically. When real user
		// auth lands, set the actual user id here instead.
		SetUserID(r.Context(), "api-key")

		next.ServeHTTP(w, r)
	})
}
