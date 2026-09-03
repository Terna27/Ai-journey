package services

import (
	"context"
	"errors"
	"testing"

	"music-api/internal/models"
	"music-api/internal/repository"

	"github.com/jackc/pgx/v5"
)

// fakeRepo is an in-memory implementation of MusicRepo used by the service
// tests. It also records artist IDs so ownership propagation can be tested.
type fakeRepo struct {
	created         models.Music
	createdArtistID int

	getByIDResult models.Music
	getByIDErr    error

	updateResult   models.Music
	updateErr      error
	updatedWith    models.Music
	updatedID      int
	updateArtistID int

	deleteErr      error
	deletedID      int
	deleteArtistID int

	recordLikeResult  models.Music
	recordLikeErr     error
	recordLikeCalls   int
	recordLikeLikerID string
}

func (f *fakeRepo) Create(
	ctx context.Context,
	artistID int,
	m models.Music,
) (models.Music, error) {
	f.created = m
	f.createdArtistID = artistID

	// Simulate what the real repository does after INSERT ... RETURNING.
	m.ArtistID = intPtr(artistID)

	return m, nil
}

func (f *fakeRepo) GetAll(
	ctx context.Context,
	search string,
	genre string,
	sortBy string,
	page int,
	limit int,
) ([]models.Music, error) {
	return nil, nil
}

func (f *fakeRepo) GetByID(
	ctx context.Context,
	id int,
) (models.Music, error) {
	return f.getByIDResult, f.getByIDErr
}

func (f *fakeRepo) Update(
	ctx context.Context,
	id int,
	artistID int,
	m models.Music,
) (models.Music, error) {
	f.updatedID = id
	f.updateArtistID = artistID
	f.updatedWith = m

	if f.updateErr != nil {
		return models.Music{}, f.updateErr
	}

	if (f.updateResult == models.Music{}) {
		m.ID = id
		m.ArtistID = intPtr(artistID)
		return m, nil
	}

	return f.updateResult, nil
}

func (f *fakeRepo) Delete(
	ctx context.Context,
	id int,
	artistID int,
) error {
	f.deletedID = id
	f.deleteArtistID = artistID

	return f.deleteErr
}

func (f *fakeRepo) RecordLike(
	ctx context.Context,
	musicID int,
	likerID string,
) (models.Music, error) {
	f.recordLikeCalls++
	f.recordLikeLikerID = likerID

	return f.recordLikeResult, f.recordLikeErr
}

func strptr(s string) *string {
	return &s
}

func intPtr(i int) *int {
	return &i
}

func TestCreateMusicValidation(t *testing.T) {
	tests := []struct {
		name    string
		in      CreateMusicInput
		wantErr error
	}{
		{
			"empty artist",
			CreateMusicInput{
				ArtistName: "  ",
				SongTitle:  "s",
				Genre:      "g",
			},
			ErrArtistNameRequired,
		},
		{
			"empty title",
			CreateMusicInput{
				ArtistName: "a",
				SongTitle:  "",
				Genre:      "g",
			},
			ErrSongTitleRequired,
		},
		{
			"empty genre",
			CreateMusicInput{
				ArtistName: "a",
				SongTitle:  "s",
				Genre:      "   ",
			},
			ErrGenreRequired,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := NewMusicService(&fakeRepo{})

			_, err := svc.CreateMusic(
				context.Background(),
				1,
				tt.in,
			)

			if !errors.Is(err, tt.wantErr) {
				t.Fatalf(
					"want %v, got %v",
					tt.wantErr,
					err,
				)
			}
		})
	}
}

func TestCreateMusicRequiresArtistAuthentication(t *testing.T) {
	repo := &fakeRepo{}
	svc := NewMusicService(repo)

	_, err := svc.CreateMusic(
		context.Background(),
		0,
		CreateMusicInput{
			ArtistName: "Artist",
			SongTitle:  "Song",
			Genre:      "Afrobeats",
		},
	)

	if !errors.Is(err, ErrUnauthorized) {
		t.Fatalf(
			"want ErrUnauthorized, got %v",
			err,
		)
	}
}

func TestCreateMusicTrimsWhitespaceAndPassesArtistID(t *testing.T) {
	repo := &fakeRepo{}
	svc := NewMusicService(repo)

	const artistID = 42

	music, err := svc.CreateMusic(
		context.Background(),
		artistID,
		CreateMusicInput{
			ArtistName: "  Burna Boy  ",
			SongTitle:  "  Last Last  ",
			Genre:      " Afrobeats ",
			ImageURL:   "  http://img  ",
		},
	)

	if err != nil {
		t.Fatalf(
			"unexpected error: %v",
			err,
		)
	}

	if repo.createdArtistID != artistID {
		t.Errorf(
			"want artistID=%d, got %d",
			artistID,
			repo.createdArtistID,
		)
	}

	if repo.created.ArtistName != "Burna Boy" {
		t.Errorf(
			"artist not trimmed: %q",
			repo.created.ArtistName,
		)
	}

	if repo.created.SongTitle != "Last Last" {
		t.Errorf(
			"title not trimmed: %q",
			repo.created.SongTitle,
		)
	}

	if repo.created.Genre != "Afrobeats" {
		t.Errorf(
			"genre not trimmed: %q",
			repo.created.Genre,
		)
	}

	if repo.created.ImageURL != "http://img" {
		t.Errorf(
			"image URL not trimmed: %q",
			repo.created.ImageURL,
		)
	}

	if music.ArtistID == nil {
		t.Fatal("expected created music to contain artist_id")
	}

	if *music.ArtistID != artistID {
		t.Errorf(
			"want returned artist_id=%d, got %d",
			artistID,
			*music.ArtistID,
		)
	}
}

func TestGetMusicTranslatesNotFound(t *testing.T) {
	repo := &fakeRepo{
		getByIDErr: pgx.ErrNoRows,
	}

	svc := NewMusicService(repo)

	_, err := svc.GetMusic(
		context.Background(),
		1,
	)

	if !errors.Is(err, ErrMusicNotFound) {
		t.Fatalf(
			"want ErrMusicNotFound, got %v",
			err,
		)
	}
}

func TestUpdateMusicPassesArtistID(t *testing.T) {
	repo := &fakeRepo{}
	svc := NewMusicService(repo)

	const (
		musicID  = 15
		artistID = 42
	)

	_, err := svc.UpdateMusic(
		context.Background(),
		musicID,
		artistID,
		CreateMusicInput{
			ArtistName: "Artist",
			SongTitle:  "Updated Song",
			Genre:      "Pop",
		},
	)

	if err != nil {
		t.Fatalf(
			"unexpected error: %v",
			err,
		)
	}

	if repo.updatedID != musicID {
		t.Errorf(
			"want musicID=%d, got %d",
			musicID,
			repo.updatedID,
		)
	}

	if repo.updateArtistID != artistID {
		t.Errorf(
			"want artistID=%d, got %d",
			artistID,
			repo.updateArtistID,
		)
	}
}

func TestUpdateMusicRequiresArtistAuthentication(t *testing.T) {
	repo := &fakeRepo{}
	svc := NewMusicService(repo)

	_, err := svc.UpdateMusic(
		context.Background(),
		1,
		0,
		CreateMusicInput{
			ArtistName: "Artist",
			SongTitle:  "Song",
			Genre:      "Pop",
		},
	)

	if !errors.Is(err, ErrUnauthorized) {
		t.Fatalf(
			"want ErrUnauthorized, got %v",
			err,
		)
	}
}

func TestPatchMusicAppliesOnlyProvidedFields(t *testing.T) {
	ownerID := 42

	repo := &fakeRepo{
		getByIDResult: models.Music{
			ID:         1,
			ArtistID:   &ownerID,
			ArtistName: "Old Artist",
			SongTitle:  "Old Title",
			Genre:      "Old Genre",
			ImageURL:   "old.jpg",
		},
	}

	svc := NewMusicService(repo)

	_, err := svc.PatchMusic(
		context.Background(),
		1,
		ownerID,
		UpdateMusicInput{
			SongTitle: strptr("New Title"),
		},
	)

	if err != nil {
		t.Fatalf(
			"unexpected error: %v",
			err,
		)
	}

	if repo.updatedWith.SongTitle != "New Title" {
		t.Errorf(
			"title not updated: %q",
			repo.updatedWith.SongTitle,
		)
	}

	if repo.updatedWith.ArtistName != "Old Artist" {
		t.Errorf(
			"artist should be unchanged, got %q",
			repo.updatedWith.ArtistName,
		)
	}

	if repo.updatedWith.Genre != "Old Genre" {
		t.Errorf(
			"genre should be unchanged, got %q",
			repo.updatedWith.Genre,
		)
	}

	if repo.updateArtistID != ownerID {
		t.Errorf(
			"want artistID=%d, got %d",
			ownerID,
			repo.updateArtistID,
		)
	}
}

func TestPatchMusicRejectsEmptyProvidedField(t *testing.T) {
	repo := &fakeRepo{
		getByIDResult: models.Music{
			ID:         1,
			ArtistName: "A",
			SongTitle:  "S",
			Genre:      "G",
		},
	}

	svc := NewMusicService(repo)

	_, err := svc.PatchMusic(
		context.Background(),
		1,
		42,
		UpdateMusicInput{
			ArtistName: strptr("   "),
		},
	)

	if !errors.Is(err, ErrArtistNameRequired) {
		t.Fatalf(
			"want ErrArtistNameRequired, got %v",
			err,
		)
	}
}

func TestPatchMusicNotFound(t *testing.T) {
	repo := &fakeRepo{
		getByIDErr: pgx.ErrNoRows,
	}

	svc := NewMusicService(repo)

	_, err := svc.PatchMusic(
		context.Background(),
		99,
		42,
		UpdateMusicInput{
			Genre: strptr("Pop"),
		},
	)

	if !errors.Is(err, ErrMusicNotFound) {
		t.Fatalf(
			"want ErrMusicNotFound, got %v",
			err,
		)
	}
}

func TestPatchMusicRequiresArtistAuthentication(t *testing.T) {
	repo := &fakeRepo{}
	svc := NewMusicService(repo)

	_, err := svc.PatchMusic(
		context.Background(),
		1,
		0,
		UpdateMusicInput{
			Genre: strptr("Pop"),
		},
	)

	if !errors.Is(err, ErrUnauthorized) {
		t.Fatalf(
			"want ErrUnauthorized, got %v",
			err,
		)
	}
}

func TestDeleteMusicPassesArtistID(t *testing.T) {
	repo := &fakeRepo{}
	svc := NewMusicService(repo)

	const (
		musicID  = 25
		artistID = 42
	)

	err := svc.DeleteMusic(
		context.Background(),
		musicID,
		artistID,
	)

	if err != nil {
		t.Fatalf(
			"unexpected error: %v",
			err,
		)
	}

	if repo.deletedID != musicID {
		t.Errorf(
			"want deleted music ID=%d, got %d",
			musicID,
			repo.deletedID,
		)
	}

	if repo.deleteArtistID != artistID {
		t.Errorf(
			"want artistID=%d, got %d",
			artistID,
			repo.deleteArtistID,
		)
	}
}

func TestDeleteMusicRequiresArtistAuthentication(t *testing.T) {
	repo := &fakeRepo{}
	svc := NewMusicService(repo)

	err := svc.DeleteMusic(
		context.Background(),
		1,
		0,
	)

	if !errors.Is(err, ErrUnauthorized) {
		t.Fatalf(
			"want ErrUnauthorized, got %v",
			err,
		)
	}
}

func TestDeleteMusicTranslatesNotFound(t *testing.T) {
	repo := &fakeRepo{
		deleteErr: pgx.ErrNoRows,
	}

	svc := NewMusicService(repo)

	err := svc.DeleteMusic(
		context.Background(),
		999,
		42,
	)

	if !errors.Is(err, ErrMusicNotFound) {
		t.Fatalf(
			"want ErrMusicNotFound, got %v",
			err,
		)
	}
}

func TestLikeMusicSurfacesAlreadyLiked(t *testing.T) {
	repo := &fakeRepo{
		recordLikeErr: repository.ErrAlreadyLiked,
	}

	svc := NewMusicService(repo)

	_, err := svc.LikeMusic(
		context.Background(),
		1,
		"42",
	)

	if !errors.Is(err, ErrAlreadyLiked) {
		t.Fatalf(
			"want ErrAlreadyLiked, got %v",
			err,
		)
	}

	if repo.recordLikeLikerID != "42" {
		t.Errorf(
			"likerID not passed through, got %q",
			repo.recordLikeLikerID,
		)
	}
}

func TestLikeMusicTranslatesNotFound(t *testing.T) {
	repo := &fakeRepo{
		recordLikeErr: pgx.ErrNoRows,
	}

	svc := NewMusicService(repo)

	_, err := svc.LikeMusic(
		context.Background(),
		1,
		"42",
	)

	if !errors.Is(err, ErrMusicNotFound) {
		t.Fatalf(
			"want ErrMusicNotFound, got %v",
			err,
		)
	}
}

func TestLikeMusicSuccess(t *testing.T) {
	repo := &fakeRepo{
		recordLikeResult: models.Music{
			ID:    1,
			Likes: 1,
		},
	}

	svc := NewMusicService(repo)

	music, err := svc.LikeMusic(
		context.Background(),
		1,
		"42",
	)

	if err != nil {
		t.Fatalf(
			"unexpected error: %v",
			err,
		)
	}

	if music.Likes != 1 {
		t.Errorf(
			"want likes=1, got %d",
			music.Likes,
		)
	}

	if repo.recordLikeCalls != 1 {
		t.Errorf(
			"want RecordLike called once, got %d",
			repo.recordLikeCalls,
		)
	}
}
