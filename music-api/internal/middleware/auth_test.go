package middleware

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"music-api/internal/services"
)

func TestJWTAuthMissingToken(t *testing.T) {
	jwtService := services.NewJWTService(
		"test-secret",
	)

	next := http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			t.Fatal("protected handler should not be called")
		},
	)

	protected := JWTAuth(
		jwtService,
		next,
	)

	req := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/music",
		nil,
	)

	rec := httptest.NewRecorder()

	protected.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf(
			"expected status 401, got %d",
			rec.Code,
		)
	}

	if !strings.Contains(
		rec.Body.String(),
		"AUTHENTICATION_REQUIRED",
	) {
		t.Fatalf(
			"expected AUTHENTICATION_REQUIRED, got %s",
			rec.Body.String(),
		)
	}
}

func TestJWTAuthInvalidToken(t *testing.T) {
	jwtService := services.NewJWTService(
		"test-secret",
	)

	next := http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			t.Fatal("protected handler should not be called")
		},
	)

	protected := JWTAuth(
		jwtService,
		next,
	)

	req := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/music",
		nil,
	)

	req.Header.Set(
		"Authorization",
		"Bearer invalid-token",
	)

	rec := httptest.NewRecorder()

	protected.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf(
			"expected status 401, got %d",
			rec.Code,
		)
	}

	if !strings.Contains(
		rec.Body.String(),
		"INVALID_TOKEN",
	) {
		t.Fatalf(
			"expected INVALID_TOKEN, got %s",
			rec.Body.String(),
		)
	}
}

func TestJWTAuthValidToken(t *testing.T) {
	jwtService := services.NewJWTService(
		"test-secret",
	)

	token, err := jwtService.GenerateToken(
		42,
		"artist@example.com",
	)
	if err != nil {
		t.Fatalf(
			"failed to generate token: %v",
			err,
		)
	}

	next := http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			artistID, ok := ArtistIDFromContext(
				r.Context(),
			)

			if !ok {
				t.Fatal(
					"expected artist ID in context",
				)
			}

			if artistID != 42 {
				t.Fatalf(
					"expected artist ID 42, got %d",
					artistID,
				)
			}

			w.WriteHeader(http.StatusOK)
		},
	)

	protected := JWTAuth(
		jwtService,
		next,
	)

	req := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/music",
		nil,
	)

	req.Header.Set(
		"Authorization",
		"Bearer "+token,
	)

	rec := httptest.NewRecorder()

	protected.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf(
			"expected status 200, got %d",
			rec.Code,
		)
	}
}

func TestJWTAuthMalformedAuthorizationHeader(t *testing.T) {
	jwtService := services.NewJWTService(
		"test-secret",
	)

	next := http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			t.Fatal("protected handler should not be called")
		},
	)

	protected := JWTAuth(
		jwtService,
		next,
	)

	req := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/music",
		nil,
	)

	req.Header.Set(
		"Authorization",
		"invalid-header",
	)

	rec := httptest.NewRecorder()

	protected.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf(
			"expected status 401, got %d",
			rec.Code,
		)
	}

	if !strings.Contains(
		rec.Body.String(),
		"INVALID_AUTHORIZATION_HEADER",
	) {
		t.Fatalf(
			"expected INVALID_AUTHORIZATION_HEADER, got %s",
			rec.Body.String(),
		)
	}
}
