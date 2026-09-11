package services

import (
	"context"
	"errors"
	"testing"

	"music-api/internal/models"
)

// fakeLyricsRepo is an in-memory implementation of
// LyricsRepo used by the lyrics service tests.
type fakeLyricsRepo struct {
	upsertedPlain  string
	upsertedLines  []models.LyricLine
	upsertArtistID int
}

func (f *fakeLyricsRepo) GetByMusicID(
	ctx context.Context,
	musicID int,
) (models.TrackLyrics, error) {
	return models.TrackLyrics{}, nil
}

func (f *fakeLyricsRepo) UpsertOwned(
	ctx context.Context,
	musicID int,
	artistID int,
	plainLyrics string,
	syncedLines []models.LyricLine,
) (models.TrackLyrics, error) {
	f.upsertArtistID = artistID
	f.upsertedPlain = plainLyrics
	f.upsertedLines = syncedLines

	return models.TrackLyrics{
		MusicID:     musicID,
		PlainLyrics: plainLyrics,
		SyncedLines: syncedLines,
	}, nil
}

func (f *fakeLyricsRepo) DeleteOwned(
	ctx context.Context,
	musicID int,
	artistID int,
) error {
	return nil
}

func TestUpsertLyricsAcceptsMergedRange(
	t *testing.T,
) {
	repo := &fakeLyricsRepo{}

	service := NewLyricsService(
		repo,
	)

	lyrics, err := service.UpsertLyrics(
		context.Background(),
		7,
		3,
		UpsertLyricsInput{
			SyncedLines: []models.LyricLine{
				{
					TimeMS: 10000,
					EndMS:  25000,
					Text:   "Hello from the other side",
				},
				{
					TimeMS: 30000,
					Text:   "Next line",
				},
			},
		},
	)

	if err != nil {
		t.Fatalf(
			"expected merged range to be accepted, got %v",
			err,
		)
	}

	if len(lyrics.SyncedLines) != 2 {
		t.Fatalf(
			"expected 2 synced lines, got %d",
			len(lyrics.SyncedLines),
		)
	}

	merged := lyrics.SyncedLines[0]

	if merged.TimeMS != 10000 ||
		merged.EndMS != 25000 {
		t.Fatalf(
			"expected merged range 10000..25000, got %d..%d",
			merged.TimeMS,
			merged.EndMS,
		)
	}

	if merged.Text !=
		"Hello from the other side" {
		t.Fatalf(
			"expected merged text to survive, got %q",
			merged.Text,
		)
	}
}

func TestUpsertLyricsRejectsEndBeforeStart(
	t *testing.T,
) {
	service := NewLyricsService(
		&fakeLyricsRepo{},
	)

	_, err := service.UpsertLyrics(
		context.Background(),
		7,
		3,
		UpsertLyricsInput{
			SyncedLines: []models.LyricLine{
				{
					TimeMS: 10000,
					EndMS:  9000,
					Text:   "invalid",
				},
			},
		},
	)

	if !errors.Is(
		err,
		ErrInvalidLyricRange,
	) {
		t.Fatalf(
			"expected ErrInvalidLyricRange, got %v",
			err,
		)
	}
}

func TestUpsertLyricsRejectsOverlappingRanges(
	t *testing.T,
) {
	service := NewLyricsService(
		&fakeLyricsRepo{},
	)

	_, err := service.UpsertLyrics(
		context.Background(),
		7,
		3,
		UpsertLyricsInput{
			SyncedLines: []models.LyricLine{
				{
					TimeMS: 10000,
					EndMS:  20000,
					Text:   "first",
				},
				{
					TimeMS: 15000,
					Text:   "overlaps",
				},
			},
		},
	)

	if !errors.Is(
		err,
		ErrOverlappingLyricRanges,
	) {
		t.Fatalf(
			"expected ErrOverlappingLyricRanges, got %v",
			err,
		)
	}
}

func TestUpsertLyricsAcceptsContiguousRanges(
	t *testing.T,
) {
	repo := &fakeLyricsRepo{}

	service := NewLyricsService(
		repo,
	)

	_, err := service.UpsertLyrics(
		context.Background(),
		7,
		3,
		UpsertLyricsInput{
			SyncedLines: []models.LyricLine{
				{
					TimeMS: 10000,
					EndMS:  15000,
					Text:   "first",
				},
				{
					TimeMS: 15000,
					EndMS:  20000,
					Text:   "second",
				},
			},
		},
	)

	if err != nil {
		t.Fatalf(
			"expected contiguous ranges to be accepted, got %v",
			err,
		)
	}

	if repo.upsertedLines[0].EndMS != 15000 {
		t.Fatalf(
			"expected explicit end to persist, got %d",
			repo.upsertedLines[0].EndMS,
		)
	}
}

func TestUpsertLyricsRejectsNegativeEnd(
	t *testing.T,
) {
	service := NewLyricsService(
		&fakeLyricsRepo{},
	)

	_, err := service.UpsertLyrics(
		context.Background(),
		7,
		3,
		UpsertLyricsInput{
			SyncedLines: []models.LyricLine{
				{
					TimeMS: 10000,
					EndMS:  -1,
					Text:   "negative",
				},
			},
		},
	)

	if !errors.Is(
		err,
		ErrInvalidLyricTimestamp,
	) {
		t.Fatalf(
			"expected ErrInvalidLyricTimestamp, got %v",
			err,
		)
	}
}
