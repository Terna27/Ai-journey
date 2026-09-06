package repository

import (
	"context"
	"errors"

	"music-api/internal/models"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrUserEmailExists = errors.New(
	"user email already exists",
)

type UserRepository struct {
	DB *pgxpool.Pool
}

func NewUserRepository(
	db *pgxpool.Pool,
) *UserRepository {
	return &UserRepository{
		DB: db,
	}
}

func (r *UserRepository) Create(
	ctx context.Context,
	user models.User,
) (models.User, error) {
	query := `
		INSERT INTO users (
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
			email_verified,
			email_verified_at,
			created_at,
			updated_at
	`

	err := r.DB.QueryRow(
		ctx,
		query,
		user.Name,
		user.Email,
		user.PasswordHash,
	).Scan(
		&user.ID,
		&user.Name,
		&user.Email,
		&user.PasswordHash,
		&user.EmailVerified,
		&user.EmailVerifiedAt,
		&user.CreatedAt,
		&user.UpdatedAt,
	)

	if err != nil {
		var pgErr *pgconn.PgError

		if errors.As(err, &pgErr) &&
			pgErr.Code == "23505" {
			return models.User{},
				ErrUserEmailExists
		}

		return models.User{}, err
	}

	return user, nil
}

func (r *UserRepository) GetByEmail(
	ctx context.Context,
	email string,
) (models.User, error) {
	query := `
		SELECT
			id,
			name,
			email,
			password_hash,
			email_verified,
			email_verified_at,
			created_at,
			updated_at
		FROM users
		WHERE email = $1
	`

	var user models.User

	err := r.DB.QueryRow(
		ctx,
		query,
		email,
	).Scan(
		&user.ID,
		&user.Name,
		&user.Email,
		&user.PasswordHash,
		&user.EmailVerified,
		&user.EmailVerifiedAt,
		&user.CreatedAt,
		&user.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return models.User{},
				pgx.ErrNoRows
		}

		return models.User{}, err
	}

	return user, nil
}

func (r *UserRepository) GetByID(
	ctx context.Context,
	id int,
) (models.User, error) {
	query := `
		SELECT
			id,
			name,
			email,
			password_hash,
			email_verified,
			email_verified_at,
			created_at,
			updated_at
		FROM users
		WHERE id = $1
	`

	var user models.User

	err := r.DB.QueryRow(
		ctx,
		query,
		id,
	).Scan(
		&user.ID,
		&user.Name,
		&user.Email,
		&user.PasswordHash,
		&user.EmailVerified,
		&user.EmailVerifiedAt,
		&user.CreatedAt,
		&user.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return models.User{},
				pgx.ErrNoRows
		}

		return models.User{}, err
	}

	return user, nil
}

func (r *UserRepository) MarkEmailVerified(
	ctx context.Context,
	userID int,
) error {
	query := `
		UPDATE users
		SET
			email_verified = TRUE,
			email_verified_at = NOW(),
			updated_at = NOW()
		WHERE id = $1
	`

	commandTag, err := r.DB.Exec(
		ctx,
		query,
		userID,
	)
	if err != nil {
		return err
	}

	if commandTag.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}

	return nil
}
