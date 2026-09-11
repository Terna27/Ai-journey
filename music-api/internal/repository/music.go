package repository

import (
	"context"
	"errors"

	"music-api/internal/models"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type MusicRepository struct {
	DB *pgxpool.Pool
}

var ErrAlreadyLiked = errors.New("music already liked by this caller")

func NewMusicRepository(db *pgxpool.Pool) *MusicRepository {
	return &MusicRepository{
		DB: db,
	}
}

// Create creates a new music record owned by the authenticated artist.
func (r *MusicRepository) Create(
	ctx context.Context,
	artistID int,
	music models.Music,
) (models.Music, error) {
	query := `
		INSERT INTO music (
			artist_id,
			artist_name,
			song_title,
			genre,
			image_url,
			image_public_id,
			audio_url,
			audio_public_id,
			audio_key
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING
			id,
			artist_id,
			release_id,
			track_number,
			artist_name,
			song_title,
			genre,
			COALESCE(image_url, ''),
			COALESCE(image_public_id, ''),
			COALESCE(audio_url, ''),
			COALESCE(audio_public_id, ''),
			COALESCE(audio_key, ''),
			likes,
			loves,
			rating,
			date_posted
	`

	err := r.DB.QueryRow(
		ctx,
		query,
		artistID,
		music.ArtistName,
		music.SongTitle,
		music.Genre,
		music.ImageURL,
		music.ImagePublicID,
		music.AudioURL,
		music.AudioPublicID,
		music.AudioKey,
	).Scan(
		&music.ID,
		&music.ArtistID,
		&music.ReleaseID,
		&music.TrackNumber,
		&music.ArtistName,
		&music.SongTitle,
		&music.Genre,
		&music.ImageURL,
		&music.ImagePublicID,
		&music.AudioURL,
		&music.AudioPublicID,
		&music.AudioKey,
		&music.Likes,
		&music.Loves,
		&music.Rating,
		&music.DatePosted,
	)

	if err != nil {
		return models.Music{}, err
	}

	return music, nil
}

func (r *MusicRepository) GetAll(
	ctx context.Context,
	search string,
	genre string,
	sortBy string,
	page int,
	limit int,
) ([]models.Music, error) {
	orderBy := "date_posted DESC"

	switch sortBy {
	case "rating":
		orderBy = "rating DESC"
	case "likes":
		orderBy = "likes DESC"
	case "loves":
		orderBy = "loves DESC"
	case "oldest":
		orderBy = "date_posted ASC"
	case "", "newest":
		orderBy = "date_posted DESC"
	}

	offset := (page - 1) * limit

	query := `
		SELECT
			id,
			artist_id,
			release_id,
			track_number,
			artist_name,
			song_title,
			genre,
			COALESCE(image_url, ''),
			COALESCE(image_public_id, ''),
			COALESCE(audio_url, ''),
			COALESCE(audio_public_id, ''),
			COALESCE(audio_key, ''),
			likes,
			loves,
			rating,
			date_posted
		FROM music
		WHERE
			(
				$1 = ''
				OR artist_name ILIKE '%' || $1 || '%'
				OR song_title ILIKE '%' || $1 || '%'
			)
			AND (
				$2 = ''
				OR genre ILIKE $2
			)
		ORDER BY ` + orderBy + `
		LIMIT $3
		OFFSET $4
	`

	rows, err := r.DB.Query(
		ctx,
		query,
		search,
		genre,
		limit,
		offset,
	)
	if err != nil {
		return nil, err
	}

	defer rows.Close()

	musicList := []models.Music{}

	for rows.Next() {
		var music models.Music

		err := rows.Scan(
			&music.ID,
			&music.ArtistID,
			&music.ReleaseID,
			&music.TrackNumber,
			&music.ArtistName,
			&music.SongTitle,
			&music.Genre,
			&music.ImageURL,
			&music.ImagePublicID,
			&music.AudioURL,
			&music.AudioPublicID,
			&music.AudioKey,
			&music.Likes,
			&music.Loves,
			&music.Rating,
			&music.DatePosted,
		)
		if err != nil {
			return nil, err
		}

		musicList = append(
			musicList,
			music,
		)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return musicList, nil
}

func (r *MusicRepository) GetByID(
	ctx context.Context,
	id int,
) (models.Music, error) {
	query := `
		SELECT
			id,
			artist_id,
			release_id,
			track_number,
			artist_name,
			song_title,
			genre,
			COALESCE(image_url, ''),
			COALESCE(image_public_id, ''),
			COALESCE(audio_url, ''),
			COALESCE(audio_public_id, ''),
			COALESCE(audio_key, ''),
			likes,
			loves,
			rating,
			date_posted
		FROM music
		WHERE id = $1
	`

	var music models.Music

	err := r.DB.QueryRow(
		ctx,
		query,
		id,
	).Scan(
		&music.ID,
		&music.ArtistID,
		&music.ReleaseID,
		&music.TrackNumber,
		&music.ArtistName,
		&music.SongTitle,
		&music.Genre,
		&music.ImageURL,
		&music.ImagePublicID,
		&music.AudioURL,
		&music.AudioPublicID,
		&music.AudioKey,
		&music.Likes,
		&music.Loves,
		&music.Rating,
		&music.DatePosted,
	)

	if err != nil {
		if errors.Is(
			err,
			pgx.ErrNoRows,
		) {
			return models.Music{}, pgx.ErrNoRows
		}

		return models.Music{}, err
	}

	return music, nil
}

// Update only updates a music record when it belongs to artistID.
func (r *MusicRepository) Update(
	ctx context.Context,
	id int,
	artistID int,
	music models.Music,
) (models.Music, error) {
	query := `
		UPDATE music
		SET
			artist_name = $1,
			song_title = $2,
			genre = $3,
			image_url = $4,
			image_public_id = $5,
			audio_url = $6,
			audio_public_id = $7,
			audio_key = $8
		WHERE id = $9
		  AND artist_id = $10
		RETURNING
			id,
			artist_id,
			release_id,
			track_number,
			artist_name,
			song_title,
			genre,
			COALESCE(image_url, ''),
			COALESCE(image_public_id, ''),
			COALESCE(audio_url, ''),
			COALESCE(audio_public_id, ''),
			COALESCE(audio_key, ''),
			likes,
			loves,
			rating,
			date_posted
	`

	err := r.DB.QueryRow(
		ctx,
		query,
		music.ArtistName,
		music.SongTitle,
		music.Genre,
		music.ImageURL,
		music.ImagePublicID,
		music.AudioURL,
		music.AudioPublicID,
		music.AudioKey,
		id,
		artistID,
	).Scan(
		&music.ID,
		&music.ArtistID,
		&music.ReleaseID,
		&music.TrackNumber,
		&music.ArtistName,
		&music.SongTitle,
		&music.Genre,
		&music.ImageURL,
		&music.ImagePublicID,
		&music.AudioURL,
		&music.AudioPublicID,
		&music.AudioKey,
		&music.Likes,
		&music.Loves,
		&music.Rating,
		&music.DatePosted,
	)

	if err != nil {
		if errors.Is(
			err,
			pgx.ErrNoRows,
		) {
			return models.Music{}, pgx.ErrNoRows
		}

		return models.Music{}, err
	}

	return music, nil
}

// Delete only deletes a music record when it belongs to artistID.
func (r *MusicRepository) Delete(
	ctx context.Context,
	id int,
	artistID int,
) error {
	query := `
		DELETE FROM music
		WHERE id = $1
		  AND artist_id = $2
	`

	result, err := r.DB.Exec(
		ctx,
		query,
		id,
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

// RecordLike records one like per likerID.
//
// A caller does not need to own the music to like it.
func (r *MusicRepository) RecordLike(
	ctx context.Context,
	musicID int,
	likerID string,
) (models.Music, error) {
	tx, err := r.DB.Begin(ctx)
	if err != nil {
		return models.Music{}, err
	}

	defer tx.Rollback(ctx)

	var exists int

	err = tx.QueryRow(
		ctx,
		`SELECT id FROM music WHERE id = $1`,
		musicID,
	).Scan(&exists)

	if err != nil {
		if errors.Is(
			err,
			pgx.ErrNoRows,
		) {
			return models.Music{}, pgx.ErrNoRows
		}

		return models.Music{}, err
	}

	result, err := tx.Exec(
		ctx,
		`
			INSERT INTO music_likes (
				music_id,
				liker_id
			)
			VALUES ($1, $2)
			ON CONFLICT DO NOTHING
		`,
		musicID,
		likerID,
	)

	if err != nil {
		return models.Music{}, err
	}

	if result.RowsAffected() == 0 {
		return models.Music{}, ErrAlreadyLiked
	}

	query := `
		UPDATE music
		SET likes = likes + 1
		WHERE id = $1
		RETURNING
			id,
			artist_id,
			release_id,
			track_number,
			artist_name,
			song_title,
			genre,
			COALESCE(image_url, ''),
			COALESCE(image_public_id, ''),
			COALESCE(audio_url, ''),
			COALESCE(audio_public_id, ''),
			COALESCE(audio_key, ''),
			likes,
			loves,
			rating,
			date_posted
	`

	var music models.Music

	err = tx.QueryRow(
		ctx,
		query,
		musicID,
	).Scan(
		&music.ID,
		&music.ArtistID,
		&music.ReleaseID,
		&music.TrackNumber,
		&music.ArtistName,
		&music.SongTitle,
		&music.Genre,
		&music.ImageURL,
		&music.ImagePublicID,
		&music.AudioURL,
		&music.AudioPublicID,
		&music.AudioKey,
		&music.Likes,
		&music.Loves,
		&music.Rating,
		&music.DatePosted,
	)

	if err != nil {
		return models.Music{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return models.Music{}, err
	}

	return music, nil
}

// GetByArtistID returns all music owned by a specific artist.
// Artist ownership is determined by music.artist_id, never artist_name.
func (r *MusicRepository) GetByArtistID(
	ctx context.Context,
	artistID int,
) ([]models.Music, error) {
	query := `
		SELECT
			id,
			artist_id,
			release_id,
			track_number,
			artist_name,
			song_title,
			genre,
			COALESCE(image_url, ''),
			COALESCE(image_public_id, ''),
			COALESCE(audio_url, ''),
			COALESCE(audio_public_id, ''),
			COALESCE(audio_key, ''),
			likes,
			loves,
			rating,
			date_posted
		FROM music
		WHERE artist_id = $1
		ORDER BY date_posted DESC
	`

	rows, err := r.DB.Query(
		ctx,
		query,
		artistID,
	)
	if err != nil {
		return nil, err
	}

	defer rows.Close()

	tracks := make(
		[]models.Music,
		0,
	)

	for rows.Next() {
		var music models.Music

		err := rows.Scan(
			&music.ID,
			&music.ArtistID,
			&music.ReleaseID,
			&music.TrackNumber,
			&music.ArtistName,
			&music.SongTitle,
			&music.Genre,
			&music.ImageURL,
			&music.ImagePublicID,
			&music.AudioURL,
			&music.AudioPublicID,
			&music.AudioKey,
			&music.Likes,
			&music.Loves,
			&music.Rating,
			&music.DatePosted,
		)

		if err != nil {
			return nil, err
		}

		tracks = append(
			tracks,
			music,
		)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return tracks, nil
}
