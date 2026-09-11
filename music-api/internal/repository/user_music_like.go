package repository

import (
	"context"
	"errors"

	"music-api/internal/models"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrNotLiked = errors.New(
	"music is not liked by this user",
)

type UserMusicLikeRepository struct {
	DB *pgxpool.Pool
}

func NewUserMusicLikeRepository(
	db *pgxpool.Pool,
) *UserMusicLikeRepository {
	return &UserMusicLikeRepository{
		DB: db,
	}
}

// LikeMusic records an authenticated user's like.
//
// The like row and cached music.likes counter are updated
// inside one transaction so they cannot become inconsistent.
func (r *UserMusicLikeRepository) LikeMusic(
	ctx context.Context,
	userID int,
	musicID int,
) (models.Music, error) {
	tx, err := r.DB.Begin(ctx)
	if err != nil {
		return models.Music{}, err
	}

	defer func() {
		_ = tx.Rollback(ctx)
	}()

	var exists int

	err = tx.QueryRow(
		ctx,
		`
		SELECT id
		FROM music
		WHERE id = $1
		`,
		musicID,
	).Scan(
		&exists,
	)

	if err != nil {
		if errors.Is(
			err,
			pgx.ErrNoRows,
		) {
			return models.Music{},
				pgx.ErrNoRows
		}

		return models.Music{}, err
	}

	result, err := tx.Exec(
		ctx,
		`
		INSERT INTO user_music_likes (
			user_id,
			music_id
		)
		VALUES ($1, $2)
		ON CONFLICT DO NOTHING
		`,
		userID,
		musicID,
	)

	if err != nil {
		return models.Music{}, err
	}

	if result.RowsAffected() == 0 {
		return models.Music{},
			ErrAlreadyLiked
	}

	music, err :=
		updateAndReturnMusic(
			ctx,
			tx,
			musicID,
			`
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
			`,
		)

	if err != nil {
		return models.Music{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return models.Music{}, err
	}

	return music, nil
}

// UnlikeMusic removes an authenticated user's like.
//
// The relationship deletion and cached counter decrement happen
// inside the same transaction.
func (r *UserMusicLikeRepository) UnlikeMusic(
	ctx context.Context,
	userID int,
	musicID int,
) (models.Music, error) {
	tx, err := r.DB.Begin(ctx)
	if err != nil {
		return models.Music{}, err
	}

	defer func() {
		_ = tx.Rollback(ctx)
	}()

	var exists int

	err = tx.QueryRow(
		ctx,
		`
		SELECT id
		FROM music
		WHERE id = $1
		`,
		musicID,
	).Scan(
		&exists,
	)

	if err != nil {
		if errors.Is(
			err,
			pgx.ErrNoRows,
		) {
			return models.Music{},
				pgx.ErrNoRows
		}

		return models.Music{}, err
	}

	result, err := tx.Exec(
		ctx,
		`
		DELETE FROM user_music_likes
		WHERE user_id = $1
		  AND music_id = $2
		`,
		userID,
		musicID,
	)

	if err != nil {
		return models.Music{}, err
	}

	if result.RowsAffected() == 0 {
		return models.Music{},
			ErrNotLiked
	}

	music, err :=
		updateAndReturnMusic(
			ctx,
			tx,
			musicID,
			`
			UPDATE music
			SET likes = GREATEST(
				likes - 1,
				0
			)
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
			`,
		)

	if err != nil {
		return models.Music{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return models.Music{}, err
	}

	return music, nil
}

// GetLikedMusic returns the authenticated user's liked tracks,
// newest likes first.
func (r *UserMusicLikeRepository) GetLikedMusic(
	ctx context.Context,
	userID int,
) ([]models.Music, error) {
	rows, err := r.DB.Query(
		ctx,
		`
		SELECT
			m.id,
			m.artist_id,
			m.release_id,
			m.track_number,
			m.artist_name,
			m.song_title,
			m.genre,
			COALESCE(m.image_url, ''),
			COALESCE(m.image_public_id, ''),
			COALESCE(m.audio_url, ''),
			COALESCE(m.audio_public_id, ''),
			COALESCE(m.audio_key, ''),
			m.likes,
			m.loves,
			m.rating,
			m.date_posted
		FROM user_music_likes uml
		INNER JOIN music m
			ON m.id = uml.music_id
		WHERE uml.user_id = $1
		ORDER BY uml.created_at DESC
		`,
		userID,
	)

	if err != nil {
		return nil, err
	}

	defer rows.Close()

	musicList :=
		make(
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

		musicList =
			append(
				musicList,
				music,
			)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return musicList, nil
}

// IsLiked reports whether a user currently likes a specific track.
//
// We will use this later when returning track details and when
// rendering the heart state in React.
func (r *UserMusicLikeRepository) IsLiked(
	ctx context.Context,
	userID int,
	musicID int,
) (bool, error) {
	var exists bool

	err := r.DB.QueryRow(
		ctx,
		`
		SELECT EXISTS (
			SELECT 1
			FROM user_music_likes
			WHERE user_id = $1
			  AND music_id = $2
		)
		`,
		userID,
		musicID,
	).Scan(
		&exists,
	)

	if err != nil {
		return false, err
	}

	return exists, nil
}

func updateAndReturnMusic(
	ctx context.Context,
	tx pgx.Tx,
	musicID int,
	query string,
) (models.Music, error) {
	var music models.Music

	err := tx.QueryRow(
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

	return music, nil
}
