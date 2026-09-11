package repository

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"music-api/internal/models"
)

var (
	ErrPodcastSlugExists = errors.New(
		"podcast slug already exists",
	)

	ErrPodcastEpisodeSlugExists = errors.New(
		"podcast episode slug already exists",
	)

	ErrPodcastEpisodeNumberExists = errors.New(
		"podcast episode number already exists",
	)
)

type PodcastRepository struct {
	db *pgxpool.Pool
}

func NewPodcastRepository(
	db *pgxpool.Pool,
) *PodcastRepository {
	return &PodcastRepository{
		db: db,
	}
}

type podcastRowScanner interface {
	Scan(dest ...any) error
}

func scanPodcast(
	row podcastRowScanner,
) (models.Podcast, error) {
	var podcast models.Podcast

	err := row.Scan(
		&podcast.ID,
		&podcast.OwnerUserID,
		&podcast.Title,
		&podcast.Slug,
		&podcast.Description,
		&podcast.Category,
		&podcast.ArtworkURL,
		&podcast.ArtworkPublicID,
		&podcast.Status,
		&podcast.IsExplicit,
		&podcast.PublishedAt,
		&podcast.CreatedAt,
		&podcast.UpdatedAt,
	)
	if err != nil {
		return models.Podcast{}, err
	}

	return podcast, nil
}

func scanPodcastEpisode(
	row podcastRowScanner,
) (models.PodcastEpisode, error) {
	var episode models.PodcastEpisode

	err := row.Scan(
		&episode.ID,
		&episode.PodcastID,
		&episode.Title,
		&episode.Slug,
		&episode.Description,
		&episode.SeasonNumber,
		&episode.EpisodeNumber,
		&episode.EpisodeType,
		&episode.AudioURL,
		&episode.AudioPublicID,
		&episode.ArtworkURL,
		&episode.ArtworkPublicID,
		&episode.DurationMS,
		&episode.Status,
		&episode.IsExplicit,
		&episode.ScheduledAt,
		&episode.PublishedAt,
		&episode.CreatedAt,
		&episode.UpdatedAt,
	)
	if err != nil {
		return models.PodcastEpisode{}, err
	}

	return episode, nil
}

func podcastColumns() string {
	return `
		id,
		owner_user_id,
		title,
		slug,
		description,
		category,
		artwork_url,
		artwork_public_id,
		status,
		is_explicit,
		published_at,
		created_at,
		updated_at
	`
}

func podcastEpisodeColumns() string {
	return `
		id,
		podcast_id,
		title,
		slug,
		description,
		season_number,
		episode_number,
		episode_type,
		audio_url,
		audio_public_id,
		artwork_url,
		artwork_public_id,
		duration_ms,
		status,
		is_explicit,
		scheduled_at,
		published_at,
		created_at,
		updated_at
	`
}

func (r *PodcastRepository) CreatePodcast(
	ctx context.Context,
	podcast models.Podcast,
) (models.Podcast, error) {
	query := `
		INSERT INTO podcasts (
			owner_user_id,
			title,
			slug,
			description,
			category,
			artwork_url,
			artwork_public_id,
			status,
			is_explicit,
			published_at
		)
		VALUES (
			$1,
			$2,
			$3,
			$4,
			$5,
			$6,
			$7,
			$8,
			$9,
			$10
		)
		RETURNING
	` + podcastColumns()

	created, err := scanPodcast(
		r.db.QueryRow(
			ctx,
			query,
			podcast.OwnerUserID,
			podcast.Title,
			podcast.Slug,
			podcast.Description,
			podcast.Category,
			podcast.ArtworkURL,
			podcast.ArtworkPublicID,
			podcast.Status,
			podcast.IsExplicit,
			podcast.PublishedAt,
		),
	)
	if err != nil {
		return models.Podcast{},
			mapPodcastDatabaseError(err)
	}

	return created, nil
}

func (r *PodcastRepository) GetPodcastByID(
	ctx context.Context,
	id int64,
) (models.Podcast, error) {
	query := `
		SELECT
	` + podcastColumns() + `
		FROM podcasts
		WHERE id = $1
	`

	return scanPodcast(
		r.db.QueryRow(
			ctx,
			query,
			id,
		),
	)
}

func (r *PodcastRepository) GetOwnedPodcastByID(
	ctx context.Context,
	id int64,
	ownerUserID int,
) (models.Podcast, error) {
	query := `
		SELECT
	` + podcastColumns() + `
		FROM podcasts
		WHERE id = $1
		  AND owner_user_id = $2
	`

	return scanPodcast(
		r.db.QueryRow(
			ctx,
			query,
			id,
			ownerUserID,
		),
	)
}

func (r *PodcastRepository) GetPublishedPodcastByID(
	ctx context.Context,
	id int64,
) (models.Podcast, error) {
	query := `
		SELECT
	` + podcastColumns() + `
		FROM podcasts
		WHERE id = $1
		  AND status = 'PUBLISHED'
	`

	return scanPodcast(
		r.db.QueryRow(
			ctx,
			query,
			id,
		),
	)
}

func (r *PodcastRepository) GetPublishedPodcastBySlug(
	ctx context.Context,
	slug string,
) (models.Podcast, error) {
	query := `
		SELECT
	` + podcastColumns() + `
		FROM podcasts
		WHERE lower(slug) = lower($1)
		  AND status = 'PUBLISHED'
	`

	return scanPodcast(
		r.db.QueryRow(
			ctx,
			query,
			slug,
		),
	)
}

func (r *PodcastRepository) GetPodcastsByOwnerUserID(
	ctx context.Context,
	ownerUserID int,
) ([]models.Podcast, error) {
	query := `
		SELECT
	` + podcastColumns() + `
		FROM podcasts
		WHERE owner_user_id = $1
		ORDER BY created_at DESC, id DESC
	`

	rows, err := r.db.Query(
		ctx,
		query,
		ownerUserID,
	)
	if err != nil {
		return nil, err
	}

	defer rows.Close()

	podcasts := make(
		[]models.Podcast,
		0,
	)

	for rows.Next() {
		podcast, err := scanPodcast(rows)
		if err != nil {
			return nil, err
		}

		podcasts = append(
			podcasts,
			podcast,
		)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return podcasts, nil
}

func (r *PodcastRepository) GetPublishedPodcasts(
	ctx context.Context,
	limit int,
	offset int,
) ([]models.Podcast, error) {
	query := `
		SELECT
	` + podcastColumns() + `
		FROM podcasts
		WHERE status = 'PUBLISHED'
		ORDER BY
			published_at DESC NULLS LAST,
			id DESC
		LIMIT $1
		OFFSET $2
	`

	rows, err := r.db.Query(
		ctx,
		query,
		limit,
		offset,
	)
	if err != nil {
		return nil, err
	}

	defer rows.Close()

	podcasts := make(
		[]models.Podcast,
		0,
	)

	for rows.Next() {
		podcast, err := scanPodcast(rows)
		if err != nil {
			return nil, err
		}

		podcasts = append(
			podcasts,
			podcast,
		)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return podcasts, nil
}

func (r *PodcastRepository) UpdatePodcast(
	ctx context.Context,
	podcast models.Podcast,
) (models.Podcast, error) {
	query := `
		UPDATE podcasts
		SET
			title = $1,
			slug = $2,
			description = $3,
			category = $4,
			artwork_url = $5,
			artwork_public_id = $6,
			status = $7,
			is_explicit = $8,
			published_at = $9,
			updated_at = NOW()
		WHERE id = $10
		  AND owner_user_id = $11
		RETURNING
	` + podcastColumns()

	updated, err := scanPodcast(
		r.db.QueryRow(
			ctx,
			query,
			podcast.Title,
			podcast.Slug,
			podcast.Description,
			podcast.Category,
			podcast.ArtworkURL,
			podcast.ArtworkPublicID,
			podcast.Status,
			podcast.IsExplicit,
			podcast.PublishedAt,
			podcast.ID,
			podcast.OwnerUserID,
		),
	)
	if err != nil {
		return models.Podcast{},
			mapPodcastDatabaseError(err)
	}

	return updated, nil
}

func (r *PodcastRepository) DeletePodcast(
	ctx context.Context,
	id int64,
	ownerUserID int,
) error {
	commandTag, err := r.db.Exec(
		ctx,
		`
			DELETE FROM podcasts
			WHERE id = $1
			  AND owner_user_id = $2
		`,
		id,
		ownerUserID,
	)
	if err != nil {
		return err
	}

	if commandTag.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}

	return nil
}

func (r *PodcastRepository) CreateEpisode(
	ctx context.Context,
	episode models.PodcastEpisode,
) (models.PodcastEpisode, error) {
	query := `
		INSERT INTO podcast_episodes (
			podcast_id,
			title,
			slug,
			description,
			season_number,
			episode_number,
			episode_type,
			audio_url,
			audio_public_id,
			artwork_url,
			artwork_public_id,
			duration_ms,
			status,
			is_explicit,
			scheduled_at,
			published_at
		)
		VALUES (
			$1,
			$2,
			$3,
			$4,
			$5,
			$6,
			$7,
			$8,
			$9,
			$10,
			$11,
			$12,
			$13,
			$14,
			$15,
			$16
		)
		RETURNING
	` + podcastEpisodeColumns()

	created, err := scanPodcastEpisode(
		r.db.QueryRow(
			ctx,
			query,
			episode.PodcastID,
			episode.Title,
			episode.Slug,
			episode.Description,
			episode.SeasonNumber,
			episode.EpisodeNumber,
			episode.EpisodeType,
			episode.AudioURL,
			episode.AudioPublicID,
			episode.ArtworkURL,
			episode.ArtworkPublicID,
			episode.DurationMS,
			episode.Status,
			episode.IsExplicit,
			episode.ScheduledAt,
			episode.PublishedAt,
		),
	)
	if err != nil {
		return models.PodcastEpisode{},
			mapPodcastDatabaseError(err)
	}

	return created, nil
}

func (r *PodcastRepository) GetEpisodeByID(
	ctx context.Context,
	id int64,
) (models.PodcastEpisode, error) {
	query := `
		SELECT
	` + podcastEpisodeColumns() + `
		FROM podcast_episodes
		WHERE id = $1
	`

	return scanPodcastEpisode(
		r.db.QueryRow(
			ctx,
			query,
			id,
		),
	)
}

func (r *PodcastRepository) GetOwnedEpisodeByID(
	ctx context.Context,
	id int64,
	ownerUserID int,
) (models.PodcastEpisode, error) {
	query := `
		SELECT
			pe.id,
			pe.podcast_id,
			pe.title,
			pe.slug,
			pe.description,
			pe.season_number,
			pe.episode_number,
			pe.episode_type,
			pe.audio_url,
			pe.audio_public_id,
			pe.artwork_url,
			pe.artwork_public_id,
			pe.duration_ms,
			pe.status,
			pe.is_explicit,
			pe.scheduled_at,
			pe.published_at,
			pe.created_at,
			pe.updated_at
		FROM podcast_episodes pe
		INNER JOIN podcasts p
			ON p.id = pe.podcast_id
		WHERE pe.id = $1
		  AND p.owner_user_id = $2
	`

	return scanPodcastEpisode(
		r.db.QueryRow(
			ctx,
			query,
			id,
			ownerUserID,
		),
	)
}

func (r *PodcastRepository) GetPublishedEpisodeByID(
	ctx context.Context,
	id int64,
) (models.PodcastEpisode, error) {
	query := `
		SELECT
	` + podcastEpisodeColumns() + `
		FROM podcast_episodes
		WHERE id = $1
		  AND status = 'PUBLISHED'
	`

	return scanPodcastEpisode(
		r.db.QueryRow(
			ctx,
			query,
			id,
		),
	)
}

func (r *PodcastRepository) GetPublishedEpisodeBySlug(
	ctx context.Context,
	podcastID int64,
	slug string,
) (models.PodcastEpisode, error) {
	query := `
		SELECT
	` + podcastEpisodeColumns() + `
		FROM podcast_episodes
		WHERE podcast_id = $1
		  AND lower(slug) = lower($2)
		  AND status = 'PUBLISHED'
	`

	return scanPodcastEpisode(
		r.db.QueryRow(
			ctx,
			query,
			podcastID,
			slug,
		),
	)
}

func (r *PodcastRepository) GetEpisodesByPodcastID(
	ctx context.Context,
	podcastID int64,
) ([]models.PodcastEpisode, error) {
	query := `
		SELECT
	` + podcastEpisodeColumns() + `
		FROM podcast_episodes
		WHERE podcast_id = $1
		ORDER BY
			season_number DESC,
			episode_number DESC NULLS LAST,
			created_at DESC,
			id DESC
	`

	rows, err := r.db.Query(
		ctx,
		query,
		podcastID,
	)
	if err != nil {
		return nil, err
	}

	defer rows.Close()

	episodes := make(
		[]models.PodcastEpisode,
		0,
	)

	for rows.Next() {
		episode, err := scanPodcastEpisode(rows)
		if err != nil {
			return nil, err
		}

		episodes = append(
			episodes,
			episode,
		)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return episodes, nil
}

func (r *PodcastRepository) GetPublishedEpisodesByPodcastID(
	ctx context.Context,
	podcastID int64,
	limit int,
	offset int,
) ([]models.PodcastEpisode, error) {
	query := `
		SELECT
	` + podcastEpisodeColumns() + `
		FROM podcast_episodes
		WHERE podcast_id = $1
		  AND status = 'PUBLISHED'
		ORDER BY
			published_at DESC NULLS LAST,
			id DESC
		LIMIT $2
		OFFSET $3
	`

	rows, err := r.db.Query(
		ctx,
		query,
		podcastID,
		limit,
		offset,
	)
	if err != nil {
		return nil, err
	}

	defer rows.Close()

	episodes := make(
		[]models.PodcastEpisode,
		0,
	)

	for rows.Next() {
		episode, err := scanPodcastEpisode(rows)
		if err != nil {
			return nil, err
		}

		episodes = append(
			episodes,
			episode,
		)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return episodes, nil
}

func (r *PodcastRepository) UpdateEpisode(
	ctx context.Context,
	episode models.PodcastEpisode,
	ownerUserID int,
) (models.PodcastEpisode, error) {
	query := `
		UPDATE podcast_episodes pe
		SET
			title = $1,
			slug = $2,
			description = $3,
			season_number = $4,
			episode_number = $5,
			episode_type = $6,
			audio_url = $7,
			audio_public_id = $8,
			artwork_url = $9,
			artwork_public_id = $10,
			duration_ms = $11,
			status = $12,
			is_explicit = $13,
			scheduled_at = $14,
			published_at = $15,
			updated_at = NOW()
		FROM podcasts p
		WHERE pe.id = $16
		  AND p.id = pe.podcast_id
		  AND p.owner_user_id = $17
		RETURNING
			pe.id,
			pe.podcast_id,
			pe.title,
			pe.slug,
			pe.description,
			pe.season_number,
			pe.episode_number,
			pe.episode_type,
			pe.audio_url,
			pe.audio_public_id,
			pe.artwork_url,
			pe.artwork_public_id,
			pe.duration_ms,
			pe.status,
			pe.is_explicit,
			pe.scheduled_at,
			pe.published_at,
			pe.created_at,
			pe.updated_at
	`

	updated, err := scanPodcastEpisode(
		r.db.QueryRow(
			ctx,
			query,
			episode.Title,
			episode.Slug,
			episode.Description,
			episode.SeasonNumber,
			episode.EpisodeNumber,
			episode.EpisodeType,
			episode.AudioURL,
			episode.AudioPublicID,
			episode.ArtworkURL,
			episode.ArtworkPublicID,
			episode.DurationMS,
			episode.Status,
			episode.IsExplicit,
			episode.ScheduledAt,
			episode.PublishedAt,
			episode.ID,
			ownerUserID,
		),
	)
	if err != nil {
		return models.PodcastEpisode{},
			mapPodcastDatabaseError(err)
	}

	return updated, nil
}

func (r *PodcastRepository) DeleteEpisode(
	ctx context.Context,
	id int64,
	ownerUserID int,
) error {
	commandTag, err := r.db.Exec(
		ctx,
		`
			DELETE FROM podcast_episodes pe
			USING podcasts p
			WHERE pe.id = $1
			  AND p.id = pe.podcast_id
			  AND p.owner_user_id = $2
		`,
		id,
		ownerUserID,
	)
	if err != nil {
		return err
	}

	if commandTag.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}

	return nil
}

func mapPodcastDatabaseError(
	err error,
) error {
	var pgErr *pgconn.PgError

	if !errors.As(err, &pgErr) {
		return err
	}

	if pgErr.Code != "23505" {
		return err
	}

	switch pgErr.ConstraintName {
	case "idx_podcasts_slug_unique":
		return ErrPodcastSlugExists

	case "idx_podcast_episodes_slug_unique":
		return ErrPodcastEpisodeSlugExists

	case "idx_podcast_episodes_number_unique":
		return ErrPodcastEpisodeNumberExists

	default:
		return err
	}
}
