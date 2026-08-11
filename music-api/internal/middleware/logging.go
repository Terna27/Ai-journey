package middleware

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"log/slog"
	"net/http"
	"strings"
	"time"
)

// ctxKey is an unexported type for context keys defined in this package.
// Using a private type prevents collisions with keys set by other packages.
type ctxKey int

const logFieldsKey ctxKey = iota

// logFields holds per-request values that middleware and handlers enrich as
// the request travels down the stack. It is stored in the context as a
// pointer, so a value set by an inner handler (e.g. the authenticated user id)
// is visible to the outer logging middleware after the response is written.
type logFields struct {
	requestID string
	userID    string
}

// SetUserID records the authenticated user id for the current request so it
// appears in the access log. Call this from auth middleware or a handler once
// the caller's identity is known. It is a no-op if the request did not pass
// through RequestLogger.
func SetUserID(ctx context.Context, userID string) {
	if f, ok := ctx.Value(logFieldsKey).(*logFields); ok {
		f.userID = userID
	}
}

// RequestIDFromContext returns the request id assigned to the current request,
// or "" if the request did not pass through RequestLogger.
func RequestIDFromContext(ctx context.Context) string {
	if f, ok := ctx.Value(logFieldsKey).(*logFields); ok {
		return f.requestID
	}
	return ""
}

// responseRecorder wraps http.ResponseWriter to remember the status code that
// was written, so the logger can report it. Handlers that never call
// WriteHeader implicitly send 200, which we record on the first Write.
type responseRecorder struct {
	http.ResponseWriter
	status  int
	written bool
}

func (rr *responseRecorder) WriteHeader(code int) {
	if !rr.written {
		rr.status = code
		rr.written = true
	}
	rr.ResponseWriter.WriteHeader(code)
}

func (rr *responseRecorder) Write(b []byte) (int, error) {
	if !rr.written {
		rr.status = http.StatusOK
		rr.written = true
	}
	return rr.ResponseWriter.Write(b)
}

// RequestLogger is middleware that assigns each request an id, echoes it back
// in the X-Request-ID header, and logs one structured line per request once
// the handler completes:
//
//	level=INFO msg=request request_id=abc123 method=GET path=/music status=200 duration=14ms ip=... user_id=anonymous
//
// It should be the outermost middleware so it also captures requests rejected
// by inner middleware such as auth.
func RequestLogger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Reuse an upstream request id (e.g. from a load balancer) when it
		// looks sane, otherwise mint a fresh one.
		requestID := sanitizeRequestID(r.Header.Get("X-Request-ID"))
		if requestID == "" {
			requestID = newRequestID()
		}

		fields := &logFields{
			requestID: requestID,
			userID:    "anonymous",
		}
		ctx := context.WithValue(r.Context(), logFieldsKey, fields)
		r = r.WithContext(ctx)

		// Echo the id back so clients and logs can be correlated.
		w.Header().Set("X-Request-ID", requestID)

		recorder := &responseRecorder{ResponseWriter: w, status: http.StatusOK}

		next.ServeHTTP(recorder, r)

		slog.Info(
			"request",
			"request_id", fields.requestID,
			"method", r.Method,
			"path", r.URL.Path,
			"status", recorder.status,
			"duration", time.Since(start),
			"ip", GetClientIP(r),
			"user_id", fields.userID,
		)
	})
}

// newRequestID returns a random 16-character hex id. crypto/rand is used so
// ids are unpredictable; it effectively never fails, but we degrade to a
// constant rather than panic if it does.
func newRequestID() string {
	b := make([]byte, 8)
	if _, err := rand.Read(b); err != nil {
		return "unknown"
	}
	return hex.EncodeToString(b)
}

// sanitizeRequestID accepts an upstream request id only if it is short and made
// of safe characters, to keep untrusted input out of the logs.
func sanitizeRequestID(id string) string {
	id = strings.TrimSpace(id)
	if id == "" || len(id) > 64 {
		return ""
	}
	for _, c := range id {
		switch {
		case c >= 'a' && c <= 'z',
			c >= 'A' && c <= 'Z',
			c >= '0' && c <= '9',
			c == '-', c == '_':
		default:
			return ""
		}
	}
	return id
}
