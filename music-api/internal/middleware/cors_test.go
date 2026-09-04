package middleware

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func testCORSHandler() http.Handler {
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	return CORS(
		CORSConfig{
			AllowedOrigins: []string{
				"http://localhost:5173",
			},
			AllowedMethods: []string{
				http.MethodGet,
				http.MethodPost,
				http.MethodPut,
				http.MethodPatch,
				http.MethodDelete,
				http.MethodOptions,
			},
			AllowedHeaders: []string{
				"Authorization",
				"Content-Type",
			},
		},
		next,
	)
}

func TestCORSAllowedOrigin(t *testing.T) {
	handler := testCORSHandler()

	req := httptest.NewRequest(
		http.MethodGet,
		"/api/v1/music",
		nil,
	)

	req.Header.Set(
		"Origin",
		"http://localhost:5173",
	)

	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf(
			"expected status %d, got %d",
			http.StatusOK,
			rec.Code,
		)
	}

	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "http://localhost:5173" {
		t.Fatalf(
			"unexpected Access-Control-Allow-Origin: %q",
			got,
		)
	}
}

func TestCORSPreflight(t *testing.T) {
	handler := testCORSHandler()

	req := httptest.NewRequest(
		http.MethodOptions,
		"/api/v1/artists/register",
		nil,
	)

	req.Header.Set(
		"Origin",
		"http://localhost:5173",
	)

	req.Header.Set(
		"Access-Control-Request-Method",
		http.MethodPost,
	)

	req.Header.Set(
		"Access-Control-Request-Headers",
		"Content-Type",
	)

	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf(
			"expected status %d, got %d",
			http.StatusNoContent,
			rec.Code,
		)
	}

	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "http://localhost:5173" {
		t.Fatalf(
			"unexpected Access-Control-Allow-Origin: %q",
			got,
		)
	}

	if got := rec.Header().Get("Access-Control-Allow-Methods"); !strings.Contains(got, http.MethodPost) {
		t.Fatalf(
			"expected POST in allowed methods, got %q",
			got,
		)
	}

	if got := rec.Header().Get("Access-Control-Allow-Headers"); !strings.Contains(got, "Content-Type") {
		t.Fatalf(
			"expected Content-Type in allowed headers, got %q",
			got,
		)
	}
}

func TestCORSRejectsUnknownOrigin(t *testing.T) {
	handler := testCORSHandler()

	req := httptest.NewRequest(
		http.MethodGet,
		"/api/v1/music",
		nil,
	)

	req.Header.Set(
		"Origin",
		"https://example.com",
	)

	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "" {
		t.Fatalf(
			"expected no Access-Control-Allow-Origin header, got %q",
			got,
		)
	}
}

func TestCORSAllowsRequestsWithoutOrigin(t *testing.T) {
	handler := testCORSHandler()

	req := httptest.NewRequest(
		http.MethodGet,
		"/health",
		nil,
	)

	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf(
			"expected status %d, got %d",
			http.StatusOK,
			rec.Code,
		)
	}
}
