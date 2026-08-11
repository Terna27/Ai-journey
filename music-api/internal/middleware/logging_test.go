package middleware

import (
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// captureLogs points slog at a buffer for the duration of a test and restores
// the previous default logger afterward.
func captureLogs(t *testing.T) *strings.Builder {
	t.Helper()

	buf := &strings.Builder{}
	prev := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(buf, nil)))
	t.Cleanup(func() { slog.SetDefault(prev) })

	return buf
}

func TestRequestLoggerSetsRequestIDHeader(t *testing.T) {
	captureLogs(t)

	handler := RequestLogger(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTeapot)
	}))

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/music", nil))

	if got := rec.Header().Get("X-Request-ID"); got == "" {
		t.Fatal("expected X-Request-ID header to be set, got empty")
	}
}

func TestRequestLoggerReusesSaneUpstreamID(t *testing.T) {
	captureLogs(t)

	handler := RequestLogger(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))

	req := httptest.NewRequest(http.MethodGet, "/music", nil)
	req.Header.Set("X-Request-ID", "abc123")

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if got := rec.Header().Get("X-Request-ID"); got != "abc123" {
		t.Fatalf("expected upstream id to be reused, got %q", got)
	}
}

func TestRequestLoggerRejectsUnsafeUpstreamID(t *testing.T) {
	captureLogs(t)

	handler := RequestLogger(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))

	req := httptest.NewRequest(http.MethodGet, "/music", nil)
	req.Header.Set("X-Request-ID", "not valid id!")

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if got := rec.Header().Get("X-Request-ID"); got == "not valid id!" {
		t.Fatal("unsafe upstream id should have been replaced")
	}
}

func TestRequestLoggerLogsStatusAndFields(t *testing.T) {
	buf := captureLogs(t)

	handler := RequestLogger(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		SetUserID(r.Context(), "api-key")
		w.WriteHeader(http.StatusNotFound)
	}))

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodDelete, "/music/9", nil))

	out := buf.String()
	for _, want := range []string{
		"status=404",
		"method=DELETE",
		"path=/music/9",
		"user_id=api-key",
		"request_id=",
		"duration=",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("log line missing %q\ngot: %s", want, out)
		}
	}
}

func TestRequestLoggerDefaultsToAnonymous(t *testing.T) {
	buf := captureLogs(t)

	handler := RequestLogger(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/music", nil))

	if !strings.Contains(buf.String(), "user_id=anonymous") {
		t.Errorf("expected user_id=anonymous for unauthenticated request\ngot: %s", buf.String())
	}
}
