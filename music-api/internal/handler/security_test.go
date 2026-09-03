package handler

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"music-api/internal/middleware"
	"music-api/internal/models"
	"music-api/internal/services"

	"github.com/jackc/pgx/v5"
)

// TestProtectedPatchRequiresAuthentication verifies that the HTTP security
// layer rejects PATCH requests that do not contain a JWT.
//
// The handler itself should never be reached because JWTAuth rejects the
// request first.
func TestProtectedPatchRequiresAuthentication(t *testing.T) {
	jwtService := services.NewJWTService("test-secret")

	mux := http.NewServeMux()

	mux.Handle(
		"PATCH /api/v1/music/{id}",
		middleware.JWTAuth(
			jwtService,
			http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				t.Fatal("protected PATCH handler should not be reached without authentication")
			}),
		),
	)

	req := httptest.NewRequest(
		http.MethodPatch,
		"/api/v1/music/25",
		strings.NewReader(`{"song_title":"Unauthorized change"}`),
	)

	req.Header.Set("Content-Type", "application/json")

	rec := httptest.NewRecorder()

	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf(
			"expected status %d, got %d",
			http.StatusUnauthorized,
			rec.Code,
		)
	}

	expected := `"code":"AUTHENTICATION_REQUIRED"`

	if !strings.Contains(rec.Body.String(), expected) {
		t.Fatalf(
			"expected response to contain %s, got %s",
			expected,
			rec.Body.String(),
		)
	}
}

// TestProtectedPatchRejectsInvalidToken verifies that a malformed or invalid
// JWT cannot reach a protected music handler.
func TestProtectedPatchRejectsInvalidToken(t *testing.T) {
	jwtService := services.NewJWTService("test-secret")

	mux := http.NewServeMux()

	mux.Handle(
		"PATCH /api/v1/music/{id}",
		middleware.JWTAuth(
			jwtService,
			http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				t.Fatal("protected PATCH handler should not be reached with an invalid token")
			}),
		),
	)

	req := httptest.NewRequest(
		http.MethodPatch,
		"/api/v1/music/25",
		strings.NewReader(`{"song_title":"Unauthorized change"}`),
	)

	req.Header.Set(
		"Authorization",
		"Bearer definitely-not-a-valid-token",
	)

	req.Header.Set("Content-Type", "application/json")

	rec := httptest.NewRecorder()

	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf(
			"expected status %d, got %d",
			http.StatusUnauthorized,
			rec.Code,
		)
	}

	expected := `"code":"INVALID_TOKEN"`

	if !strings.Contains(rec.Body.String(), expected) {
		t.Fatalf(
			"expected response to contain %s, got %s",
			expected,
			rec.Body.String(),
		)
	}
}

// TestProtectedDeleteRequiresAuthentication verifies that DELETE cannot reach
// the protected handler when the request has no JWT.
func TestProtectedDeleteRequiresAuthentication(t *testing.T) {
	jwtService := services.NewJWTService("test-secret")

	mux := http.NewServeMux()

	mux.Handle(
		"DELETE /api/v1/music/{id}",
		middleware.JWTAuth(
			jwtService,
			http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				t.Fatal("protected DELETE handler should not be reached without authentication")
			}),
		),
	)

	req := httptest.NewRequest(
		http.MethodDelete,
		"/api/v1/music/25",
		nil,
	)

	rec := httptest.NewRecorder()

	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf(
			"expected status %d, got %d",
			http.StatusUnauthorized,
			rec.Code,
		)
	}

	expected := `"code":"AUTHENTICATION_REQUIRED"`

	if !strings.Contains(rec.Body.String(), expected) {
		t.Fatalf(
			"expected response to contain %s, got %s",
			expected,
			rec.Body.String(),
		)
	}
}

// TestProtectedRouteAcceptsValidToken verifies the complete JWT middleware
// handoff: a valid JWT reaches the protected handler and the authenticated
// artist ID is available through the request context.
func TestProtectedRouteAcceptsValidToken(t *testing.T) {
	jwtService := services.NewJWTService("test-secret")

	token, err := jwtService.GenerateToken(
		42,
		"artist@example.com",
	)
	if err != nil {
		t.Fatalf("failed to generate test token: %v", err)
	}

	mux := http.NewServeMux()

	mux.Handle(
		"DELETE /api/v1/music/{id}",
		middleware.JWTAuth(
			jwtService,
			http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				artistID, ok := middleware.ArtistIDFromContext(r.Context())
				if !ok {
					t.Fatal("expected authenticated artist ID in request context")
				}

				if artistID != 42 {
					t.Fatalf(
						"expected artist ID 42, got %d",
						artistID,
					)
				}

				w.WriteHeader(http.StatusNoContent)
			}),
		),
	)

	req := httptest.NewRequest(
		http.MethodDelete,
		"/api/v1/music/25",
		nil,
	)

	req.Header.Set(
		"Authorization",
		"Bearer "+token,
	)

	rec := httptest.NewRecorder()

	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf(
			"expected status %d, got %d; body=%s",
			http.StatusNoContent,
			rec.Code,
			rec.Body.String(),
		)
	}
}

// ownershipTestRepo is an in-memory repository used to verify ownership
// behavior without connecting to PostgreSQL.
type ownershipTestRepo struct {
	music   models.Music
	ownerID int
	deleted bool
}

func newOwnershipTestRepo(ownerID int) *ownershipTestRepo {
	artistID := ownerID

	return &ownershipTestRepo{
		ownerID: ownerID,
		music: models.Music{
			ID:         25,
			ArtistID:   &artistID,
			ArtistName: "Security Test Artist A",
			SongTitle:  "Ownership Security Test",
			Genre:      "Test",
			ImageURL:   "https://example.com/test-image.jpg",
			AudioKey:   "security-test-audio",
		},
	}
}

func (r *ownershipTestRepo) Create(
	ctx context.Context,
	artistID int,
	music models.Music,
) (models.Music, error) {
	music.ArtistID = &artistID
	r.music = music
	r.ownerID = artistID

	return music, nil
}

func (r *ownershipTestRepo) GetAll(
	ctx context.Context,
	search string,
	genre string,
	sortBy string,
	page int,
	limit int,
) ([]models.Music, error) {
	if r.deleted {
		return []models.Music{}, nil
	}

	return []models.Music{r.music}, nil
}

func (r *ownershipTestRepo) GetByID(
	ctx context.Context,
	id int,
) (models.Music, error) {
	if r.deleted || id != r.music.ID {
		return models.Music{}, pgx.ErrNoRows
	}

	return r.music, nil
}

func (r *ownershipTestRepo) Update(
	ctx context.Context,
	id int,
	artistID int,
	music models.Music,
) (models.Music, error) {
	if r.deleted ||
		id != r.music.ID ||
		artistID != r.ownerID {
		return models.Music{}, pgx.ErrNoRows
	}

	music.ID = r.music.ID

	ownerID := r.ownerID
	music.ArtistID = &ownerID

	r.music = music

	return r.music, nil
}

func (r *ownershipTestRepo) Delete(
	ctx context.Context,
	id int,
	artistID int,
) error {
	if r.deleted ||
		id != r.music.ID ||
		artistID != r.ownerID {
		return pgx.ErrNoRows
	}

	r.deleted = true

	return nil
}

func (r *ownershipTestRepo) RecordLike(
	ctx context.Context,
	musicID int,
	likerID string,
) (models.Music, error) {
	if r.deleted || musicID != r.music.ID {
		return models.Music{}, pgx.ErrNoRows
	}

	r.music.Likes++

	return r.music, nil
}

// newOwnershipTestServer builds the same security chain used by the
// application:
//
// JWT -> Handler -> Service -> ownership-aware repository.
func newOwnershipTestServer(
	t *testing.T,
	ownerID int,
) (*http.ServeMux, *ownershipTestRepo, *services.JWTService) {
	t.Helper()

	repo := newOwnershipTestRepo(ownerID)

	musicService := services.NewMusicService(repo)
	jwtService := services.NewJWTService("ownership-test-secret")

	musicHandler := &MusicHandler{
		Service:    musicService,
		JWTService: jwtService,
	}

	mux := http.NewServeMux()

	mux.Handle(
		"PATCH /api/v1/music/{id}",
		middleware.JWTAuth(
			jwtService,
			http.HandlerFunc(musicHandler.PatchMusic),
		),
	)

	mux.Handle(
		"DELETE /api/v1/music/{id}",
		middleware.JWTAuth(
			jwtService,
			http.HandlerFunc(musicHandler.DeleteMusic),
		),
	)

	return mux, repo, jwtService
}

func generateOwnershipTestToken(
	t *testing.T,
	jwtService *services.JWTService,
	artistID int,
) string {
	t.Helper()

	token, err := jwtService.GenerateToken(
		artistID,
		"security-test@example.com",
	)
	if err != nil {
		t.Fatalf(
			"failed to generate JWT for artist %d: %v",
			artistID,
			err,
		)
	}

	return token
}

// TestOwnerCanPatchMusic verifies the full authenticated ownership flow for
// an artist updating their own track.
func TestOwnerCanPatchMusic(t *testing.T) {
	mux, repo, jwtService := newOwnershipTestServer(t, 2)

	token := generateOwnershipTestToken(
		t,
		jwtService,
		2,
	)

	req := httptest.NewRequest(
		http.MethodPatch,
		"/api/v1/music/25",
		strings.NewReader(
			`{"song_title":"Ownership Security Test Updated"}`,
		),
	)

	req.Header.Set(
		"Authorization",
		"Bearer "+token,
	)

	req.Header.Set(
		"Content-Type",
		"application/json",
	)

	rec := httptest.NewRecorder()

	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf(
			"expected status %d, got %d; body=%s",
			http.StatusOK,
			rec.Code,
			rec.Body.String(),
		)
	}

	if repo.music.SongTitle != "Ownership Security Test Updated" {
		t.Fatalf(
			"expected owner update to change title, got %q",
			repo.music.SongTitle,
		)
	}

	if repo.music.ArtistID == nil || *repo.music.ArtistID != 2 {
		t.Fatalf(
			"expected music to remain owned by artist 2",
		)
	}
}

// TestNonOwnerCannotPatchMusic verifies that another authenticated artist
// cannot modify a track they do not own.
func TestNonOwnerCannotPatchMusic(t *testing.T) {
	mux, repo, jwtService := newOwnershipTestServer(t, 2)

	token := generateOwnershipTestToken(
		t,
		jwtService,
		3,
	)

	originalTitle := repo.music.SongTitle

	req := httptest.NewRequest(
		http.MethodPatch,
		"/api/v1/music/25",
		strings.NewReader(
			`{"song_title":"Hacked By Artist B"}`,
		),
	)

	req.Header.Set(
		"Authorization",
		"Bearer "+token,
	)

	req.Header.Set(
		"Content-Type",
		"application/json",
	)

	rec := httptest.NewRecorder()

	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf(
			"expected status %d, got %d; body=%s",
			http.StatusNotFound,
			rec.Code,
			rec.Body.String(),
		)
	}

	if !strings.Contains(
		rec.Body.String(),
		`"code":"MUSIC_NOT_FOUND"`,
	) {
		t.Fatalf(
			"expected MUSIC_NOT_FOUND response, got %s",
			rec.Body.String(),
		)
	}

	if repo.music.SongTitle != originalTitle {
		t.Fatalf(
			"non-owner changed track title from %q to %q",
			originalTitle,
			repo.music.SongTitle,
		)
	}
}

// TestNonOwnerCannotDeleteMusic verifies that an authenticated artist cannot
// delete another artist's track.
func TestNonOwnerCannotDeleteMusic(t *testing.T) {
	mux, repo, jwtService := newOwnershipTestServer(t, 2)

	token := generateOwnershipTestToken(
		t,
		jwtService,
		3,
	)

	req := httptest.NewRequest(
		http.MethodDelete,
		"/api/v1/music/25",
		nil,
	)

	req.Header.Set(
		"Authorization",
		"Bearer "+token,
	)

	rec := httptest.NewRecorder()

	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf(
			"expected status %d, got %d; body=%s",
			http.StatusNotFound,
			rec.Code,
			rec.Body.String(),
		)
	}

	if !strings.Contains(
		rec.Body.String(),
		`"code":"MUSIC_NOT_FOUND"`,
	) {
		t.Fatalf(
			"expected MUSIC_NOT_FOUND response, got %s",
			rec.Body.String(),
		)
	}

	if repo.deleted {
		t.Fatal(
			"non-owner must not be able to delete another artist's track",
		)
	}
}

// TestOwnerCanDeleteMusic verifies that the authenticated owner can delete
// their own track.
func TestOwnerCanDeleteMusic(t *testing.T) {
	mux, repo, jwtService := newOwnershipTestServer(t, 2)

	token := generateOwnershipTestToken(
		t,
		jwtService,
		2,
	)

	req := httptest.NewRequest(
		http.MethodDelete,
		"/api/v1/music/25",
		nil,
	)

	req.Header.Set(
		"Authorization",
		"Bearer "+token,
	)

	rec := httptest.NewRecorder()

	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf(
			"expected status %d, got %d; body=%s",
			http.StatusNoContent,
			rec.Code,
			rec.Body.String(),
		)
	}

	if !repo.deleted {
		t.Fatal(
			"expected owner's track to be deleted",
		)
	}
}
