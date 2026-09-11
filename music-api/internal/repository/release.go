package repository

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"music-api/internal/models"
)

var (
	ErrReleaseTrackNotOwned = errors.New(
		"track does not belong to artist",
	)

	ErrReleaseTrackAlreadyAssigned = errors.New(
		"track already belongs to another release",
	)
)

type ReleaseRepository struct {
	db *pgxpool.Pool
}

func NewReleaseRepository(
	db *pgxpool.Pool,
) *ReleaseRepository {
	return &ReleaseRepository{
		db: db,
	}
}

func (r *ReleaseRepository) Create(
	ctx context.Context,
	release models.Release,
) (models.Release, error) {
	query := `
		INSERT INTO releases (
			artist_id,
			title,
			release_type,
			cover_image_url,
			cover_image_public_id,
			description,
			release_date,
			is_published
		)
		VALUES (
			$1,
			$2,
			$3,
			$4,
			$5,
			$6,
			$7,
			$8
		)
		RETURNING
			id,
			artist_id,
			title,
			release_type,
			cover_image_url,
			cover_image_public_id,
			description,
			release_date,
			is_published,
			created_at,
			updated_at
	`

	err := r.db.QueryRow(
		ctx,
		query,
		release.ArtistID,
		release.Title,
		release.ReleaseType,
		release.CoverImageURL,
		release.CoverImagePublicID,
		release.Description,
		release.ReleaseDate,
		release.IsPublished,
	).Scan(
		&release.ID,
		&release.ArtistID,
		&release.Title,
		&release.ReleaseType,
		&release.CoverImageURL,
		&release.CoverImagePublicID,
		&release.Description,
		&release.ReleaseDate,
		&release.IsPublished,
		&release.CreatedAt,
		&release.UpdatedAt,
	)

	if err != nil {
		return models.Release{}, err
	}

	return release, nil
}

func (r *ReleaseRepository) GetByID(
	ctx context.Context,
	id int,
) (models.Release, error) {
	var release models.Release

	query := `
		SELECT
			id,
			artist_id,
			title,
			release_type,
			cover_image_url,
			cover_image_public_id,
			description,
			release_date,
			is_published,
			created_at,
			updated_at
		FROM releases
		WHERE id = $1
	`

	err := r.db.QueryRow(
		ctx,
		query,
		id,
	).Scan(
		&release.ID,
		&release.ArtistID,
		&release.Title,
		&release.ReleaseType,
		&release.CoverImageURL,
		&release.CoverImagePublicID,
		&release.Description,
		&release.ReleaseDate,
		&release.IsPublished,
		&release.CreatedAt,
		&release.UpdatedAt,
	)

	if err != nil {
		return models.Release{}, err
	}

	return release, nil
}

func (r *ReleaseRepository) GetByArtistID(
	ctx context.Context,
	artistID int,
) ([]models.Release, error) {
	query := `
		SELECT
			id,
			artist_id,
			title,
			release_type,
			cover_image_url,
			cover_image_public_id,
			description,
			release_date,
			is_published,
			created_at,
			updated_at
		FROM releases
		WHERE artist_id = $1
		ORDER BY
			release_date DESC NULLS LAST,
			created_at DESC
	`

	rows, err := r.db.Query(
		ctx,
		query,
		artistID,
	)
	if err != nil {
		return nil, err
	}

	defer rows.Close()

	releases := make(
		[]models.Release,
		0,
	)

	for rows.Next() {
		var release models.Release

		err := rows.Scan(
			&release.ID,
			&release.ArtistID,
			&release.Title,
			&release.ReleaseType,
			&release.CoverImageURL,
			&release.CoverImagePublicID,
			&release.Description,
			&release.ReleaseDate,
			&release.IsPublished,
			&release.CreatedAt,
			&release.UpdatedAt,
		)

		if err != nil {
			return nil, err
		}

		releases = append(
			releases,
			release,
		)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return releases, nil
}

func (r *ReleaseRepository) GetTracks(
	ctx context.Context,
	releaseID int,
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
		WHERE release_id = $1
		ORDER BY
			track_number ASC NULLS LAST,
			id ASC
	`

	rows, err := r.db.Query(
		ctx,
		query,
		releaseID,
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

func (r *ReleaseRepository) AddTrack(
	ctx context.Context,
	releaseID int,
	artistID int,
	musicID int,
	trackNumber int,
) error {
	commandTag, err := r.db.Exec(
		ctx,
		`
			UPDATE music
			SET
				release_id = $1,
				track_number = $2
			WHERE id = $3
			  AND artist_id = $4
			  AND release_id IS NULL
		`,
		releaseID,
		trackNumber,
		musicID,
		artistID,
	)

	if err != nil {
		return err
	}

	if commandTag.RowsAffected() == 1 {
		return nil
	}

	var existingArtistID *int
	var existingReleaseID *int

	err = r.db.QueryRow(
		ctx,
		`
			SELECT
				artist_id,
				release_id
			FROM music
			WHERE id = $1
		`,
		musicID,
	).Scan(
		&existingArtistID,
		&existingReleaseID,
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

	if existingArtistID == nil ||
		*existingArtistID != artistID {

		return ErrReleaseTrackNotOwned
	}

	if existingReleaseID != nil {
		return ErrReleaseTrackAlreadyAssigned
	}

	return pgx.ErrNoRows
}

func (r *ReleaseRepository) RemoveTrack(
	ctx context.Context,
	releaseID int,
	artistID int,
	musicID int,
) error {
	commandTag, err := r.db.Exec(
		ctx,
		`
			UPDATE music
			SET
				release_id = NULL,
				track_number = NULL
			WHERE id = $1
			  AND artist_id = $2
			  AND release_id = $3
		`,
		musicID,
		artistID,
		releaseID,
	)

	if err != nil {
		return err
	}

	if commandTag.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}

	return nil
}

func (r *ReleaseRepository) Update(
	ctx context.Context,
	release models.Release,
) (models.Release, error) {
	query := `
		UPDATE releases
		SET
			title = $1,
			release_type = $2,
			cover_image_url = $3,
			cover_image_public_id = $4,
			description = $5,
			release_date = $6,
			is_published = $7,
			updated_at = NOW()
		WHERE id = $8
		  AND artist_id = $9
		RETURNING
			id,
			artist_id,
			title,
			release_type,
			cover_image_url,
			cover_image_public_id,
			description,
			release_date,
			is_published,
			created_at,
			updated_at
	`

	err := r.db.QueryRow(
		ctx,
		query,
		release.Title,
		release.ReleaseType,
		release.CoverImageURL,
		release.CoverImagePublicID,
		release.Description,
		release.ReleaseDate,
		release.IsPublished,
		release.ID,
		release.ArtistID,
	).Scan(
		&release.ID,
		&release.ArtistID,
		&release.Title,
		&release.ReleaseType,
		&release.CoverImageURL,
		&release.CoverImagePublicID,
		&release.Description,
		&release.ReleaseDate,
		&release.IsPublished,
		&release.CreatedAt,
		&release.UpdatedAt,
	)

	if err != nil {
		return models.Release{}, err
	}

	return release, nil
}
func (r *ReleaseRepository) Delete(
	ctx context.Context,
	id int,
	artistID int,
) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}

	defer func() {
		_ = tx.Rollback(ctx)
	}()

	// Verify that the release exists and belongs to this artist.
	// FOR UPDATE prevents another transaction from modifying or
	// deleting the release while this operation is in progress.
	var releaseID int

	err = tx.QueryRow(
		ctx,
		`
		SELECT id
		FROM releases
		WHERE id = $1
		  AND artist_id = $2
		FOR UPDATE
		`,
		id,
		artistID,
	).Scan(
		&releaseID,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return pgx.ErrNoRows
		}

		return err
	}

	// Detach every track from the release completely.
	//
	// release_id and track_number form one logical association,
	// so they must both be cleared together.
	_, err = tx.Exec(
		ctx,
		`
		UPDATE music
		SET
			release_id = NULL,
			track_number = NULL
		WHERE release_id = $1
		`,
		id,
	)
	if err != nil {
		return err
	}

	commandTag, err := tx.Exec(
		ctx,
		`
		DELETE FROM releases
		WHERE id = $1
		  AND artist_id = $2
		`,
		id,
		artistID,
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
