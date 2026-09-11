package repository

import (
	"context"
	"encoding/json"

	"music-api/internal/models"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type LyricsRepository struct {
	DB *pgxpool.Pool
}

func NewLyricsRepository(
	db *pgxpool.Pool,
) *LyricsRepository {
	return &LyricsRepository{
		DB: db,
	}
}

// GetByMusicID returns publicly readable lyrics for
// one track.
func (r *LyricsRepository) GetByMusicID(
	ctx context.Context,
	musicID int,
) (models.TrackLyrics, error) {
	query := `
		SELECT
			music_id,
			plain_lyrics,
			synced_lines,
			created_at,
			updated_at
		FROM track_lyrics
		WHERE music_id = $1
	`

	return scanTrackLyrics(
		r.DB.QueryRow(
			ctx,
			query,
			musicID,
		),
	)
}

// UpsertOwned creates or replaces the lyrics only when
// the supplied music track belongs to artistID.
//
// INSERT ... SELECT makes ownership enforcement part of
// the database operation itself.
func (r *LyricsRepository) UpsertOwned(
	ctx context.Context,
	musicID int,
	artistID int,
	plainLyrics string,
	syncedLines []models.LyricLine,
) (models.TrackLyrics, error) {
	syncedJSON, err := json.Marshal(
		syncedLines,
	)
	if err != nil {
		return models.TrackLyrics{}, err
	}

	query := `
		INSERT INTO track_lyrics (
			music_id,
			plain_lyrics,
			synced_lines
		)
		SELECT
			m.id,
			$3,
			$4::jsonb
		FROM music m
		WHERE m.id = $1
		  AND m.artist_id = $2

		ON CONFLICT (music_id)
		DO UPDATE SET
			plain_lyrics = EXCLUDED.plain_lyrics,
			synced_lines = EXCLUDED.synced_lines,
			updated_at = now()

		RETURNING
			music_id,
			plain_lyrics,
			synced_lines,
			created_at,
			updated_at
	`

	return scanTrackLyrics(
		r.DB.QueryRow(
			ctx,
			query,
			musicID,
			artistID,
			plainLyrics,
			syncedJSON,
		),
	)
}

// DeleteOwned removes lyrics only when the underlying
// track belongs to artistID.
//
// A non-owner receives the same no-row result as a
// nonexistent resource so ownership information is not
// leaked.
func (r *LyricsRepository) DeleteOwned(
	ctx context.Context,
	musicID int,
	artistID int,
) error {
	query := `
		DELETE FROM track_lyrics AS lyrics
		USING music AS m
		WHERE lyrics.music_id = $1
		  AND m.id = lyrics.music_id
		  AND m.artist_id = $2
	`

	result, err := r.DB.Exec(
		ctx,
		query,
		musicID,
		artistID,
	)
	if err != nil {
		return err
	}

	if result.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}

	return nil
}

type trackLyricsRow interface {
	Scan(dest ...any) error
}

func scanTrackLyrics(
	row trackLyricsRow,
) (models.TrackLyrics, error) {
	var lyrics models.TrackLyrics

	var syncedJSON []byte

	err := row.Scan(
		&lyrics.MusicID,
		&lyrics.PlainLyrics,
		&syncedJSON,
		&lyrics.CreatedAt,
		&lyrics.UpdatedAt,
	)
	if err != nil {
		return models.TrackLyrics{}, err
	}

	if len(syncedJSON) == 0 {
		lyrics.SyncedLines =
			[]models.LyricLine{}

		return lyrics, nil
	}

	if err := json.Unmarshal(
		syncedJSON,
		&lyrics.SyncedLines,
	); err != nil {
		return models.TrackLyrics{}, err
	}

	if lyrics.SyncedLines == nil {
		lyrics.SyncedLines =
			[]models.LyricLine{}
	}

	return lyrics, nil
}

// Keep errors import available for future repository
// extensions without changing the public API.
