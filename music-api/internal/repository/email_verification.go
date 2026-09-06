package repository

import (
	"context"
	"errors"
	"time"

	"music-api/internal/models"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type EmailVerificationRepository struct {
	DB *pgxpool.Pool
}

func NewEmailVerificationRepository(
	db *pgxpool.Pool,
) *EmailVerificationRepository {
	return &EmailVerificationRepository{
		DB: db,
	}
}

func (r *EmailVerificationRepository) Create(
	ctx context.Context,
	userID int,
	tokenHash string,
	expiresAt time.Time,
) (models.EmailVerificationToken, error) {
	query := `
		INSERT INTO email_verification_tokens (
			user_id,
			token_hash,
			expires_at
		)
		VALUES ($1, $2, $3)
		RETURNING
			id,
			user_id,
			token_hash,
			expires_at,
			used_at,
			created_at
	`

	var token models.EmailVerificationToken

	err := r.DB.QueryRow(
		ctx,
		query,
		userID,
		tokenHash,
		expiresAt,
	).Scan(
		&token.ID,
		&token.UserID,
		&token.TokenHash,
		&token.ExpiresAt,
		&token.UsedAt,
		&token.CreatedAt,
	)

	if err != nil {
		return models.EmailVerificationToken{}, err
	}

	return token, nil
}

func (r *EmailVerificationRepository) GetValidByHash(
	ctx context.Context,
	tokenHash string,
) (models.EmailVerificationToken, error) {
	query := `
		SELECT
			id,
			user_id,
			token_hash,
			expires_at,
			used_at,
			created_at
		FROM email_verification_tokens
		WHERE token_hash = $1
		  AND used_at IS NULL
		  AND expires_at > NOW()
	`

	var token models.EmailVerificationToken

	err := r.DB.QueryRow(
		ctx,
		query,
		tokenHash,
	).Scan(
		&token.ID,
		&token.UserID,
		&token.TokenHash,
		&token.ExpiresAt,
		&token.UsedAt,
		&token.CreatedAt,
	)

	if err != nil {
		return models.EmailVerificationToken{}, err
	}

	return token, nil
}

func (r *EmailVerificationRepository) MarkUsed(
	ctx context.Context,
	id int64,
) error {
	query := `
		UPDATE email_verification_tokens
		SET used_at = NOW()
		WHERE id = $1
		  AND used_at IS NULL
	`

	commandTag, err := r.DB.Exec(
		ctx,
		query,
		id,
	)
	if err != nil {
		return err
	}

	if commandTag.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}

	return nil
}

func (r *EmailVerificationRepository) InvalidateForUser(
	ctx context.Context,
	userID int,
) error {
	query := `
		UPDATE email_verification_tokens
		SET used_at = NOW()
		WHERE user_id = $1
		  AND used_at IS NULL
	`

	_, err := r.DB.Exec(
		ctx,
		query,
		userID,
	)

	return err
}

// Verify atomically consumes a valid verification token and marks
// the associated user's email address as verified.
//
// Both database changes happen inside one transaction. If either
// operation fails, neither change is committed.
func (r *EmailVerificationRepository) Verify(
	ctx context.Context,
	tokenHash string,
) error {
	tx, err := r.DB.Begin(ctx)
	if err != nil {
		return err
	}

	defer func() {
		_ = tx.Rollback(ctx)
	}()

	var userID int

	err = tx.QueryRow(
		ctx,
		`
			UPDATE email_verification_tokens
			SET used_at = NOW()
			WHERE token_hash = $1
			  AND used_at IS NULL
			  AND expires_at > NOW()
			RETURNING user_id
		`,
		tokenHash,
	).Scan(
		&userID,
	)

	if err != nil {
		if errors.Is(
			err,
			pgx.ErrNoRows,
		) {
			return pgx.ErrNoRows
		}

		return err
	}

	commandTag, err := tx.Exec(
		ctx,
		`
			UPDATE users
			SET
				email_verified = TRUE,
				email_verified_at = COALESCE(
					email_verified_at,
					NOW()
				),
				updated_at = NOW()
			WHERE id = $1
		`,
		userID,
	)

	if err != nil {
		return err
	}

	if commandTag.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}

	if err := tx.Commit(ctx); err != nil {
		return err
	}

	return nil
}
