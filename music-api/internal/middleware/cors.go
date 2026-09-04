package middleware

import (
	"net/http"
	"strings"
)

type CORSConfig struct {
	AllowedOrigins []string
	AllowedMethods []string
	AllowedHeaders []string
}

func CORS(cfg CORSConfig, next http.Handler) http.Handler {
	allowedOrigins := make(map[string]struct{}, len(cfg.AllowedOrigins))

	for _, origin := range cfg.AllowedOrigins {
		origin = strings.TrimSpace(origin)

		if origin != "" {
			allowedOrigins[origin] = struct{}{}
		}
	}

	methods := strings.Join(cfg.AllowedMethods, ", ")
	headers := strings.Join(cfg.AllowedHeaders, ", ")

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")

		if origin == "" {
			next.ServeHTTP(w, r)
			return
		}

		if _, allowed := allowedOrigins[origin]; !allowed {
			next.ServeHTTP(w, r)
			return
		}

		w.Header().Set(
			"Access-Control-Allow-Origin",
			origin,
		)

		w.Header().Set(
			"Vary",
			"Origin",
		)

		w.Header().Set(
			"Access-Control-Allow-Methods",
			methods,
		)

		w.Header().Set(
			"Access-Control-Allow-Headers",
			headers,
		)

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}
