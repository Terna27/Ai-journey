package services

import (
	"context"
	"errors"
	"strings"
	"unicode/utf8"

	"music-api/internal/models"

	"github.com/jackc/pgx/v5"
)

const (
	MaxPlainLyricsCharacters = 100000
	MaxSyncedLyricLines      = 5000
	MaxLyricLineCharacters   = 1000
)

var (
	ErrLyricsNotFound = errors.New(
		"lyrics not found",
	)

	ErrLyricsRequired = errors.New(
		"lyrics cannot be empty",
	)

	ErrLyricsTooLong = errors.New(
		"plain lyrics are too long",
	)

	ErrTooManyLyricLines = errors.New(
		"too many synchronized lyric lines",
	)

	ErrLyricLineRequired = errors.New(
		"synchronized lyric lines cannot be empty",
	)

	ErrLyricLineTooLong = errors.New(
		"synchronized lyric line is too long",
	)

	ErrInvalidLyricTimestamp = errors.New(
		"lyric timestamps must be zero or greater",
	)

	ErrLyricTimestampsNotIncreasing = errors.New(
		"lyric timestamps must be strictly increasing",
	)

	ErrInvalidLyricRange = errors.New(
		"lyric end time must be later than its start time",
	)

	ErrOverlappingLyricRanges = errors.New(
		"lyric timing ranges must not overlap",
	)
)

type LyricsRepo interface {
	GetByMusicID(
		ctx context.Context,
		musicID int,
	) (models.TrackLyrics, error)

	UpsertOwned(
		ctx context.Context,
		musicID int,
		artistID int,
		plainLyrics string,
		syncedLines []models.LyricLine,
	) (models.TrackLyrics, error)

	DeleteOwned(
		ctx context.Context,
		musicID int,
		artistID int,
	) error
}

type LyricsService struct {
	repo LyricsRepo
}

func NewLyricsService(
	repo LyricsRepo,
) *LyricsService {
	return &LyricsService{
		repo: repo,
	}
}

type UpsertLyricsInput struct {
	PlainLyrics string

	SyncedLines []models.LyricLine
}

func (s *LyricsService) GetLyrics(
	ctx context.Context,
	musicID int,
) (models.TrackLyrics, error) {
	lyrics, err :=
		s.repo.GetByMusicID(
			ctx,
			musicID,
		)

	if err != nil {
		if errors.Is(
			err,
			pgx.ErrNoRows,
		) {
			return models.TrackLyrics{},
				ErrLyricsNotFound
		}

		return models.TrackLyrics{}, err
	}

	return lyrics, nil
}

func (s *LyricsService) UpsertLyrics(
	ctx context.Context,
	musicID int,
	artistID int,
	in UpsertLyricsInput,
) (models.TrackLyrics, error) {
	if artistID <= 0 {
		return models.TrackLyrics{},
			ErrUnauthorized
	}

	plainLyrics :=
		strings.TrimSpace(
			in.PlainLyrics,
		)

	if utf8.RuneCountInString(
		plainLyrics,
	) > MaxPlainLyricsCharacters {
		return models.TrackLyrics{},
			ErrLyricsTooLong
	}

	if len(in.SyncedLines) >
		MaxSyncedLyricLines {
		return models.TrackLyrics{},
			ErrTooManyLyricLines
	}

	syncedLines := make(
		[]models.LyricLine,
		0,
		len(in.SyncedLines),
	)

	var previousTime int64 = -1

	var previousEndMS int64

	for _, line := range in.SyncedLines {
		text :=
			strings.TrimSpace(
				line.Text,
			)

		if text == "" {
			return models.TrackLyrics{},
				ErrLyricLineRequired
		}

		if utf8.RuneCountInString(
			text,
		) > MaxLyricLineCharacters {
			return models.TrackLyrics{},
				ErrLyricLineTooLong
		}

		if line.TimeMS < 0 {
			return models.TrackLyrics{},
				ErrInvalidLyricTimestamp
		}

		// An explicit end time forms a timing range. Zero
		// keeps the historical open-ended behaviour (the
		// line runs until the next line begins).
		if line.EndMS < 0 {
			return models.TrackLyrics{},
				ErrInvalidLyricTimestamp
		}

		if line.EndMS != 0 &&
			line.EndMS <= line.TimeMS {
			return models.TrackLyrics{},
				ErrInvalidLyricRange
		}

		// A range may not extend into the previous line's
		// explicit range.
		if previousEndMS > line.TimeMS {
			return models.TrackLyrics{},
				ErrOverlappingLyricRanges
		}

		if line.TimeMS <= previousTime {
			return models.TrackLyrics{},
				ErrLyricTimestampsNotIncreasing
		}

		previousTime = line.TimeMS
		previousEndMS = line.EndMS

		syncedLines = append(
			syncedLines,
			models.LyricLine{
				TimeMS: line.TimeMS,
				EndMS:  line.EndMS,
				Text:   text,
			},
		)
	}

	if plainLyrics == "" &&
		len(syncedLines) == 0 {
		return models.TrackLyrics{},
			ErrLyricsRequired
	}

	// Artists adding synchronized lyrics should not
	// have to duplicate the entire text manually.
	if plainLyrics == "" &&
		len(syncedLines) > 0 {
		lines := make(
			[]string,
			0,
			len(syncedLines),
		)

		for _, line := range syncedLines {
			lines = append(
				lines,
				line.Text,
			)
		}

		plainLyrics =
			strings.Join(
				lines,
				"\n",
			)
	}

	lyrics, err :=
		s.repo.UpsertOwned(
			ctx,
			musicID,
			artistID,
			plainLyrics,
			syncedLines,
		)

	if err != nil {
		if errors.Is(
			err,
			pgx.ErrNoRows,
		) {
			// Preserve the existing application's
			// ownership-hiding convention.
			return models.TrackLyrics{},
				ErrMusicNotFound
		}

		return models.TrackLyrics{}, err
	}

	return lyrics, nil
}

func (s *LyricsService) DeleteLyrics(
	ctx context.Context,
	musicID int,
	artistID int,
) error {
	if artistID <= 0 {
		return ErrUnauthorized
	}

	err := s.repo.DeleteOwned(
		ctx,
		musicID,
		artistID,
	)

	if err != nil {
		if errors.Is(
			err,
			pgx.ErrNoRows,
		) {
			return ErrLyricsNotFound
		}

		return err
	}

	return nil
}

func IsLyricsValidationError(
	err error,
) bool {
	return errors.Is(
		err,
		ErrLyricsRequired,
	) ||
		errors.Is(
			err,
			ErrLyricsTooLong,
		) ||
		errors.Is(
			err,
			ErrTooManyLyricLines,
		) ||
		errors.Is(
			err,
			ErrLyricLineRequired,
		) ||
		errors.Is(
			err,
			ErrLyricLineTooLong,
		) ||
		errors.Is(
			err,
			ErrInvalidLyricTimestamp,
		) ||
		errors.Is(
			err,
			ErrInvalidLyricRange,
		) ||
		errors.Is(
			err,
			ErrOverlappingLyricRanges,
		) ||
		errors.Is(
			err,
			ErrLyricTimestampsNotIncreasing,
		)
}
