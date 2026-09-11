package repository

import (
	"context"
	"errors"

	"music-api/internal/models"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrArtistProfileExists = errors.New("artist profile already exists")

type ArtistRepository struct {
	DB *pgxpool.Pool
}

func NewArtistRepository(db *pgxpool.Pool) *ArtistRepository {
	return &ArtistRepository{
		DB: db,
	}
}

// artistRowScanner is implemented by pgx.Row and allows all artist
// queries to use one consistent scan definition.
type artistRowScanner interface {
	Scan(dest ...any) error
}

func scanArtist(row artistRowScanner) (models.Artist, error) {
	var artist models.Artist

	err := row.Scan(
		&artist.ID,
		&artist.UserID,
		&artist.Name,
		&artist.Email,
		&artist.PasswordHash,
		&artist.Bio,
		&artist.ProfileImageURL,
		&artist.ProfileImagePublicID,
		&artist.HeroVideoURL,
		&artist.HeroVideoPublicID,
		&artist.HeroVideoPosterURL,
		&artist.HeroVideoPosterPublicID,
		&artist.CreatedAt,
		&artist.UpdatedAt,
	)
	if err != nil {
		return models.Artist{}, err
	}

	return artist, nil
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
			user_id,
			name,
			email,
			password_hash,
			bio,
			profile_image_url,
			profile_image_public_id,
			hero_video_url,
			hero_video_public_id,
			hero_video_poster_url,
			hero_video_poster_public_id,
			created_at,
			updated_at
		FROM artists
		WHERE email = $1
	`

	artist, err := scanArtist(
		r.DB.QueryRow(
			ctx,
			query,
			email,
		),
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
			user_id,
			name,
			email,
			password_hash,
			bio,
			profile_image_url,
			profile_image_public_id,
			hero_video_url,
			hero_video_public_id,
			hero_video_poster_url,
			hero_video_poster_public_id,
			created_at,
			updated_at
		FROM artists
		WHERE id = $1
	`

	artist, err := scanArtist(
		r.DB.QueryRow(
			ctx,
			query,
			id,
		),
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return models.Artist{}, pgx.ErrNoRows
		}

		return models.Artist{}, err
	}

	return artist, nil
}

func (r *ArtistRepository) GetByUserID(
	ctx context.Context,
	userID int,
) (models.Artist, error) {
	query := `
		SELECT
			id,
			user_id,
			name,
			email,
			password_hash,
			bio,
			profile_image_url,
			profile_image_public_id,
			hero_video_url,
			hero_video_public_id,
			hero_video_poster_url,
			hero_video_poster_public_id,
			created_at,
			updated_at
		FROM artists
		WHERE user_id = $1
	`

	artist, err := scanArtist(
		r.DB.QueryRow(
			ctx,
			query,
			userID,
		),
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return models.Artist{}, pgx.ErrNoRows
		}

		return models.Artist{}, err
	}

	return artist, nil
}

func (r *ArtistRepository) CreateForUser(
	ctx context.Context,
	user models.User,
) (models.Artist, error) {
	query := `
		INSERT INTO artists (
			user_id,
			name,
			email,
			password_hash
		)
		VALUES ($1, $2, $3, $4)
		RETURNING
			id,
			user_id,
			name,
			email,
			password_hash,
			bio,
			profile_image_url,
			profile_image_public_id,
			hero_video_url,
			hero_video_public_id,
			hero_video_poster_url,
			hero_video_poster_public_id,
			created_at,
			updated_at
	`

	artist, err := scanArtist(
		r.DB.QueryRow(
			ctx,
			query,
			user.ID,
			user.Name,
			user.Email,
			user.PasswordHash,
		),
	)
	if err != nil {
		var pgErr *pgconn.PgError

		if errors.As(err, &pgErr) &&
			pgErr.Code == "23505" {
			return models.Artist{}, ErrArtistProfileExists
		}

		return models.Artist{}, err
	}

	return artist, nil
}

func (r *ArtistRepository) UpdateProfile(
	ctx context.Context,
	artist models.Artist,
) (models.Artist, error) {
	query := `
		UPDATE artists
		SET
			bio = $2,
			profile_image_url = $3,
			profile_image_public_id = $4,
			hero_video_url = $5,
			hero_video_public_id = $6,
			hero_video_poster_url = $7,
			hero_video_poster_public_id = $8,
			updated_at = now()
		WHERE id = $1
		RETURNING
			id,
			user_id,
			name,
			email,
			password_hash,
			bio,
			profile_image_url,
			profile_image_public_id,
			hero_video_url,
			hero_video_public_id,
			hero_video_poster_url,
			hero_video_poster_public_id,
			created_at,
			updated_at
	`

	updatedArtist, err := scanArtist(
		r.DB.QueryRow(
			ctx,
			query,
			artist.ID,
			artist.Bio,
			artist.ProfileImageURL,
			artist.ProfileImagePublicID,
			artist.HeroVideoURL,
			artist.HeroVideoPublicID,
			artist.HeroVideoPosterURL,
			artist.HeroVideoPosterPublicID,
		),
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return models.Artist{}, pgx.ErrNoRows
		}

		return models.Artist{}, err
	}

	return updatedArtist, nil
}
