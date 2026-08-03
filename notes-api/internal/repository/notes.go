package repository

import (
	"context"
	"errors"

	"notes-Api/internal/models"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type NoteRepository struct {
	DB *pgxpool.Pool
}

func NewNoteRepository(db *pgxpool.Pool) *NoteRepository {
	return &NoteRepository{
		DB: db,
	}
}
func (r *NoteRepository) Create(ctx context.Context, note models.Note) (models.Note, error) {

	query := `
		INSERT INTO notes (
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
		note.ArtistName,
		note.SongTitle,
		note.Genre,
		note.ImageURL,
	).Scan(
		&note.ID,
		&note.ArtistName,
		&note.SongTitle,
		&note.Genre,
		&note.ImageURL,
		&note.Likes,
		&note.Loves,
		&note.Rating,
		&note.DatePosted,
	)

	if err != nil {
		return models.Note{}, err
	}

	return note, nil
}
func (r *NoteRepository) GetAll(
	ctx context.Context,
	search string,
	genre string,
	sortBy string,
	page int,
	limit int,
) ([]models.Note, error) {
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
	FROM notes
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

	notes := []models.Note{}

	for rows.Next() {
		var note models.Note

		err := rows.Scan(
			&note.ID,
			&note.ArtistName,
			&note.SongTitle,
			&note.Genre,
			&note.ImageURL,
			&note.Likes,
			&note.Loves,
			&note.Rating,
			&note.DatePosted,
		)
		if err != nil {
			return nil, err
		}

		notes = append(notes, note)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return notes, nil

}

func (r *NoteRepository) GetByID(ctx context.Context, id int) (models.Note, error) {

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
		FROM notes
		WHERE id = $1
	`

	var note models.Note

	err := r.DB.QueryRow(ctx, query, id).Scan(
		&note.ID,
		&note.ArtistName,
		&note.SongTitle,
		&note.Genre,
		&note.ImageURL,
		&note.Likes,
		&note.Loves,
		&note.Rating,
		&note.DatePosted,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return models.Note{}, pgx.ErrNoRows
		}

		return models.Note{}, err
	}

	return note, nil
}

func (r *NoteRepository) Update(ctx context.Context, id int, note models.Note) (models.Note, error) {

	query := `
		UPDATE notes
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
		note.ArtistName,
		note.SongTitle,
		note.Genre,
		note.ImageURL,
		id,
	).Scan(
		&note.ID,
		&note.ArtistName,
		&note.SongTitle,
		&note.Genre,
		&note.ImageURL,
		&note.Likes,
		&note.Loves,
		&note.Rating,
		&note.DatePosted,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return models.Note{}, pgx.ErrNoRows
		}

		return models.Note{}, err
	}

	return note, nil
}

func (r *NoteRepository) Delete(ctx context.Context, id int) error {

	query := `
		DELETE FROM notes
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

func (r *NoteRepository) Like(ctx context.Context, id int) (models.Note, error) {

	query := `
		UPDATE notes
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

	var note models.Note

	err := r.DB.QueryRow(
		ctx,
		query,
		id,
	).Scan(
		&note.ID,
		&note.ArtistName,
		&note.SongTitle,
		&note.Genre,
		&note.ImageURL,
		&note.Likes,
		&note.Loves,
		&note.Rating,
		&note.DatePosted,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return models.Note{}, pgx.ErrNoRows
		}

		return models.Note{}, err
	}

	return note, nil
}
