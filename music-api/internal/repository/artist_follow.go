package repository

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

// ArtistFollowRepository owns persistence for the relationship
// between users and artists they follow.
//
// Business rules such as preventing an artist owner from following
// their own artist profile belong in the service layer.
type ArtistFollowRepository struct {
	DB *pgxpool.Pool
}

func NewArtistFollowRepository(
	db *pgxpool.Pool,
) *ArtistFollowRepository {
	return &ArtistFollowRepository{
		DB: db,
	}
}

// Follow creates a user -> artist follow relationship.
//
// The composite primary key on (user_id, artist_id) guarantees
// uniqueness. ON CONFLICT DO NOTHING makes the operation idempotent,
// including when duplicate requests arrive concurrently.
func (r *ArtistFollowRepository) Follow(
	ctx context.Context,
	userID int,
	artistID int,
) error {
	query := `
		INSERT INTO artist_follows (
			user_id,
			artist_id
		)
		VALUES ($1, $2)
		ON CONFLICT (user_id, artist_id)
		DO NOTHING
	`

	_, err := r.DB.Exec(
		ctx,
		query,
		userID,
		artistID,
	)

	return err
}

// Unfollow removes a user -> artist follow relationship.
//
// DELETE is naturally idempotent: removing a relationship that does
// not exist is still considered successful.
func (r *ArtistFollowRepository) Unfollow(
	ctx context.Context,
	userID int,
	artistID int,
) error {
	query := `
		DELETE FROM artist_follows
		WHERE user_id = $1
		  AND artist_id = $2
	`

	_, err := r.DB.Exec(
		ctx,
		query,
		userID,
		artistID,
	)

	return err
}

// IsFollowing reports whether userID currently follows artistID.
func (r *ArtistFollowRepository) IsFollowing(
	ctx context.Context,
	userID int,
	artistID int,
) (bool, error) {
	query := `
		SELECT EXISTS (
			SELECT 1
			FROM artist_follows
			WHERE user_id = $1
			  AND artist_id = $2
		)
	`

	var isFollowing bool

	err := r.DB.QueryRow(
		ctx,
		query,
		userID,
		artistID,
	).Scan(
		&isFollowing,
	)
	if err != nil {
		return false, err
	}

	return isFollowing, nil
}

// FollowerCount returns the authoritative follower count for an artist.
//
// The count is derived from artist_follows rather than maintained as a
// denormalized counter so it cannot drift from the underlying rows.
func (r *ArtistFollowRepository) FollowerCount(
	ctx context.Context,
	artistID int,
) (int64, error) {
	query := `
		SELECT COUNT(*)::bigint
		FROM artist_follows
		WHERE artist_id = $1
	`

	var count int64

	err := r.DB.QueryRow(
		ctx,
		query,
		artistID,
	).Scan(
		&count,
	)
	if err != nil {
		return 0, err
	}

	return count, nil
}

// GetStatus returns both personalized follow state and the artist's
// public follower count in one database round trip.
func (r *ArtistFollowRepository) GetStatus(
	ctx context.Context,
	userID int,
	artistID int,
) (
	bool,
	int64,
	error,
) {
	query := `
		SELECT
			EXISTS (
				SELECT 1
				FROM artist_follows
				WHERE user_id = $1
				  AND artist_id = $2
			) AS is_following,
			(
				SELECT COUNT(*)::bigint
				FROM artist_follows
				WHERE artist_id = $2
			) AS follower_count
	`

	var (
		isFollowing   bool
		followerCount int64
	)

	err := r.DB.QueryRow(
		ctx,
		query,
		userID,
		artistID,
	).Scan(
		&isFollowing,
		&followerCount,
	)
	if err != nil {
		return false, 0, err
	}

	return isFollowing, followerCount, nil
}
