package service

import (
	"context"
	"errors"
	"testing"

	"music-api/internal/models"
	"music-api/internal/repository"

	"github.com/jackc/pgx/v5"
)

// fakeRepo is an in-memory MusicRepo for unit-testing the service without a
// database. Each field lets a test control what the corresponding method
// returns; captured arguments are recorded for assertions.
type fakeRepo struct {
	created models.Music

	getByIDResult models.Music
	getByIDErr    error

	updateResult models.Music
	updateErr    error
	updatedWith  models.Music

	recordLikeResult  models.Music
	recordLikeErr     error
	recordLikeCalls   int
	recordLikeLikerID string
}

func (f *fakeRepo) Create(ctx context.Context, m models.Music) (models.Music, error) {
	f.created = m
	return m, nil
}

func (f *fakeRepo) GetAll(ctx context.Context, search, genre, sortBy string, page, limit int) ([]models.Music, error) {
	return nil, nil
}

func (f *fakeRepo) GetByID(ctx context.Context, id int) (models.Music, error) {
	return f.getByIDResult, f.getByIDErr
}

func (f *fakeRepo) Update(ctx context.Context, id int, m models.Music) (models.Music, error) {
	f.updatedWith = m
	if f.updateErr != nil {
		return models.Music{}, f.updateErr
	}
	if (f.updateResult == models.Music{}) {
		return m, nil
	}
	return f.updateResult, nil
}

func (f *fakeRepo) Delete(ctx context.Context, id int) error {
	return nil
}

func (f *fakeRepo) RecordLike(ctx context.Context, musicID int, likerID string) (models.Music, error) {
	f.recordLikeCalls++
	f.recordLikeLikerID = likerID
	return f.recordLikeResult, f.recordLikeErr
}

func strptr(s string) *string { return &s }

func TestCreateMusicValidation(t *testing.T) {
	tests := []struct {
		name    string
		in      CreateMusicInput
		wantErr error
	}{
		{"empty artist", CreateMusicInput{ArtistName: "  ", SongTitle: "s", Genre: "g"}, ErrArtistNameRequired},
		{"empty title", CreateMusicInput{ArtistName: "a", SongTitle: "", Genre: "g"}, ErrSongTitleRequired},
		{"empty genre", CreateMusicInput{ArtistName: "a", SongTitle: "s", Genre: "   "}, ErrGenreRequired},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := NewMusicService(&fakeRepo{})
			_, err := svc.CreateMusic(context.Background(), tt.in)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("want %v, got %v", tt.wantErr, err)
			}
		})
	}
}

func TestCreateMusicTrimsWhitespace(t *testing.T) {
	repo := &fakeRepo{}
	svc := NewMusicService(repo)

	_, err := svc.CreateMusic(context.Background(), CreateMusicInput{
		ArtistName: "  Burna Boy  ",
		SongTitle:  "  Last Last  ",
		Genre:      " Afrobeats ",
		ImageURL:   "  http://img  ",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if repo.created.ArtistName != "Burna Boy" {
		t.Errorf("artist not trimmed: %q", repo.created.ArtistName)
	}
	if repo.created.SongTitle != "Last Last" {
		t.Errorf("title not trimmed: %q", repo.created.SongTitle)
	}
	if repo.created.Genre != "Afrobeats" {
		t.Errorf("genre not trimmed: %q", repo.created.Genre)
	}
	if repo.created.ImageURL != "http://img" {
		t.Errorf("image url not trimmed: %q", repo.created.ImageURL)
	}
}

func TestGetMusicTranslatesNotFound(t *testing.T) {
	repo := &fakeRepo{getByIDErr: pgx.ErrNoRows}
	svc := NewMusicService(repo)

	_, err := svc.GetMusic(context.Background(), 1)
	if !errors.Is(err, ErrMusicNotFound) {
		t.Fatalf("want ErrMusicNotFound, got %v", err)
	}
}

func TestPatchMusicAppliesOnlyProvidedFields(t *testing.T) {
	repo := &fakeRepo{
		getByIDResult: models.Music{
			ID:         1,
			ArtistName: "Old Artist",
			SongTitle:  "Old Title",
			Genre:      "Old Genre",
			ImageURL:   "old.jpg",
		},
	}
	svc := NewMusicService(repo)

	_, err := svc.PatchMusic(context.Background(), 1, UpdateMusicInput{
		SongTitle: strptr("New Title"),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Only the title should have changed; the rest carry over.
	if repo.updatedWith.SongTitle != "New Title" {
		t.Errorf("title not updated: %q", repo.updatedWith.SongTitle)
	}
	if repo.updatedWith.ArtistName != "Old Artist" {
		t.Errorf("artist should be unchanged, got %q", repo.updatedWith.ArtistName)
	}
	if repo.updatedWith.Genre != "Old Genre" {
		t.Errorf("genre should be unchanged, got %q", repo.updatedWith.Genre)
	}
}

func TestPatchMusicRejectsEmptyProvidedField(t *testing.T) {
	repo := &fakeRepo{getByIDResult: models.Music{ID: 1, ArtistName: "A", SongTitle: "S", Genre: "G"}}
	svc := NewMusicService(repo)

	_, err := svc.PatchMusic(context.Background(), 1, UpdateMusicInput{
		ArtistName: strptr("   "),
	})
	if !errors.Is(err, ErrArtistNameRequired) {
		t.Fatalf("want ErrArtistNameRequired, got %v", err)
	}
}

func TestPatchMusicNotFound(t *testing.T) {
	repo := &fakeRepo{getByIDErr: pgx.ErrNoRows}
	svc := NewMusicService(repo)

	_, err := svc.PatchMusic(context.Background(), 99, UpdateMusicInput{Genre: strptr("Pop")})
	if !errors.Is(err, ErrMusicNotFound) {
		t.Fatalf("want ErrMusicNotFound, got %v", err)
	}
}

func TestLikeMusicSurfacesAlreadyLiked(t *testing.T) {
	repo := &fakeRepo{recordLikeErr: repository.ErrAlreadyLiked}
	svc := NewMusicService(repo)

	_, err := svc.LikeMusic(context.Background(), 1, "1.2.3.4")
	if !errors.Is(err, ErrAlreadyLiked) {
		t.Fatalf("want ErrAlreadyLiked, got %v", err)
	}
	if repo.recordLikeLikerID != "1.2.3.4" {
		t.Errorf("likerID not passed through, got %q", repo.recordLikeLikerID)
	}
}

func TestLikeMusicTranslatesNotFound(t *testing.T) {
	repo := &fakeRepo{recordLikeErr: pgx.ErrNoRows}
	svc := NewMusicService(repo)

	_, err := svc.LikeMusic(context.Background(), 1, "1.2.3.4")
	if !errors.Is(err, ErrMusicNotFound) {
		t.Fatalf("want ErrMusicNotFound, got %v", err)
	}
}

func TestLikeMusicSuccess(t *testing.T) {
	repo := &fakeRepo{recordLikeResult: models.Music{ID: 1, Likes: 1}}
	svc := NewMusicService(repo)

	music, err := svc.LikeMusic(context.Background(), 1, "1.2.3.4")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if music.Likes != 1 {
		t.Errorf("want likes=1, got %d", music.Likes)
	}
	if repo.recordLikeCalls != 1 {
		t.Errorf("want RecordLike called once, got %d", repo.recordLikeCalls)
	}
}
