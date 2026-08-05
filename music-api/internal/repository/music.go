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

func NewMusicRepository(db *pgxpool.Pool) *MusicRepository {
	return &MusicRepository{
		DB: db,
	}
}
func (r *MusicRepository) Create(ctx context.Context, music models.Music) (models.Music, error) {

	query := `
		INSERT INTO music (
			artist_name,
			song_title,
			genre,
			image_url
		)
		VALUES ($1, $2, $3, $4)
		RETURNING
			id,
			artist_name,
			song_title,
			genre,
			image_url,
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
	).Scan(
		&music.ID,
		&music.ArtistName,
		&music.SongTitle,
		&music.Genre,
		&music.ImageURL,
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
		artist_name,
		song_title,
		genre,
		image_url,
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
			&music.ArtistName,
			&music.SongTitle,
			&music.Genre,
			&music.ImageURL,
			&music.Likes,
			&music.Loves,
			&music.Rating,
			&music.DatePosted,
		)
		if err != nil {
			return nil, err
		}

		musicList = append(musicList, music)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return musicList, nil

}

func (r *MusicRepository) GetByID(ctx context.Context, id int) (models.Music, error) {

	query := `
		SELECT
			id,
			artist_name,
			song_title,
			genre,
			image_url,
			likes,
			loves,
			rating,
			date_posted
		FROM music
		WHERE id = $1
	`

	var music models.Music

	err := r.DB.QueryRow(ctx, query, id).Scan(
		&music.ID,
		&music.ArtistName,
		&music.SongTitle,
		&music.Genre,
		&music.ImageURL,
		&music.Likes,
		&music.Loves,
		&music.Rating,
		&music.DatePosted,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return models.Music{}, pgx.ErrNoRows
		}

		return models.Music{}, err
	}

	return music, nil
}

func (r *MusicRepository) Update(ctx context.Context, id int, music models.Music) (models.Music, error) {

	query := `
		UPDATE music
		SET
			artist_name = $1,
			song_title = $2,
			genre = $3,
			image_url = $4
		WHERE id = $5
		RETURNING
			id,
			artist_name,
			song_title,
			genre,
			image_url,
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
		id,
	).Scan(
		&music.ID,
		&music.ArtistName,
		&music.SongTitle,
		&music.Genre,
		&music.ImageURL,
		&music.Likes,
		&music.Loves,
		&music.Rating,
		&music.DatePosted,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return models.Music{}, pgx.ErrNoRows
		}

		return models.Music{}, err
	}

	return music, nil
}

func (r *MusicRepository) Delete(ctx context.Context, id int) error {

	query := `
		DELETE FROM music
		WHERE id = $1
	`

	result, err := r.DB.Exec(ctx, query, id)
	if err != nil {
		return err
	}

	if result.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}

	return nil
}

func (r *MusicRepository) Like(ctx context.Context, id int) (models.Music, error) {

	query := `
		UPDATE music
		SET likes = likes + 1
		WHERE id = $1
		RETURNING
			id,
			artist_name,
			song_title,
			genre,
			image_url,
			likes,
			loves,
			rating,
			date_posted
	`

	var music models.Music

	err := r.DB.QueryRow(
		ctx,
		query,
		id,
	).Scan(
		&music.ID,
		&music.ArtistName,
		&music.SongTitle,
		&music.Genre,
		&music.ImageURL,
		&music.Likes,
		&music.Loves,
		&music.Rating,
		&music.DatePosted,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return models.Music{}, pgx.ErrNoRows
		}

		return models.Music{}, err
	}

	return music, nil
}
