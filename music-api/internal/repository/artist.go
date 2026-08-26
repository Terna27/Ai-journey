package repository

import (
	"context"
	"errors"

	"music-api/internal/models"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type ArtistRepository struct {
	DB *pgxpool.Pool
}

func NewArtistRepository(db *pgxpool.Pool) *ArtistRepository {
	return &ArtistRepository{
		DB: db,
	}
}

func (r *ArtistRepository) Create(
	ctx context.Context,
	artist models.Artist,
) (models.Artist, error) {

	query := `
		INSERT INTO artists (
			name,
			email,
			password_hash
		)
		VALUES ($1, $2, $3)
		RETURNING
			id,
			name,
			email,
			password_hash,
			created_at,
			updated_at
	`

	err := r.DB.QueryRow(
		ctx,
		query,
		artist.Name,
		artist.Email,
		artist.PasswordHash,
	).Scan(
		&artist.ID,
		&artist.Name,
		&artist.Email,
		&artist.PasswordHash,
		&artist.CreatedAt,
		&artist.UpdatedAt,
	)

	if err != nil {
		return models.Artist{}, err
	}

	return artist, nil
}

func (r *ArtistRepository) GetByEmail(
	ctx context.Context,
	email string,
) (models.Artist, error) {

	query := `
		SELECT
			id,
			name,
			email,
			password_hash,
			created_at,
			updated_at
		FROM artists
		WHERE email = $1
	`

	var artist models.Artist

	err := r.DB.QueryRow(
		ctx,
		query,
		email,
	).Scan(
		&artist.ID,
		&artist.Name,
		&artist.Email,
		&artist.PasswordHash,
		&artist.CreatedAt,
		&artist.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return models.Artist{}, pgx.ErrNoRows
		}

		return models.Artist{}, err
	}

	return artist, nil
}

func (r *ArtistRepository) GetByID(
	ctx context.Context,
	id int,
) (models.Artist, error) {

	query := `
		SELECT
			id,
			name,
			email,
			password_hash,
			created_at,
			updated_at
		FROM artists
		WHERE id = $1
	`

	var artist models.Artist

	err := r.DB.QueryRow(
		ctx,
		query,
		id,
	).Scan(
		&artist.ID,
		&artist.Name,
		&artist.Email,
		&artist.PasswordHash,
		&artist.CreatedAt,
		&artist.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return models.Artist{}, pgx.ErrNoRows
		}

		return models.Artist{}, err
	}

	return artist, nil
}
