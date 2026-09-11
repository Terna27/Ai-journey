package repository

import (
	"context"

	"music-api/internal/models"

	"github.com/jackc/pgx/v5/pgxpool"
)

type DiscoveryRepository struct {
	DB *pgxpool.Pool
}

func NewDiscoveryRepository(
	db *pgxpool.Pool,
) *DiscoveryRepository {
	return &DiscoveryRepository{
		DB: db,
	}
}

// HeroArtists returns artists eligible for the rotating homepage
// hero.
//
// An artist must have a hero video before they can appear.
//
// The current engagement score uses the signals already available
// in the platform:
//
//	likes * 3
//	+ loves * 4
//
// Once qualified listening history and artist followers exist,
// they can become stronger ranking signals without requiring any
// frontend contract changes.
func (r *DiscoveryRepository) HeroArtists(
	ctx context.Context,
	limit int,
) ([]models.DiscoveryHeroArtist, error) {
	query := `
		SELECT
			a.id,
			a.name,
			COALESCE(a.bio, ''),
			COALESCE(a.profile_image_url, ''),
			a.hero_video_url,
			COALESCE(a.hero_video_poster_url, ''),
			COALESCE(
				SUM(
					COALESCE(m.likes, 0) * 3
					+ COALESCE(m.loves, 0) * 4
				),
				0
			)::bigint AS engagement_score,
			COUNT(m.id)::int AS track_count
		FROM artists a
		LEFT JOIN music m
			ON m.artist_id = a.id
		WHERE
			a.hero_video_url IS NOT NULL
			AND BTRIM(a.hero_video_url) <> ''
		GROUP BY
			a.id,
			a.name,
			a.bio,
			a.profile_image_url,
			a.hero_video_url,
			a.hero_video_poster_url,
			a.updated_at
		ORDER BY
			engagement_score DESC,
			a.updated_at DESC,
			a.id DESC
		LIMIT $1
	`

	rows, err := r.DB.Query(
		ctx,
		query,
		limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	artists := make(
		[]models.DiscoveryHeroArtist,
		0,
	)

	for rows.Next() {
		var artist models.DiscoveryHeroArtist

		if err := rows.Scan(
			&artist.ID,
			&artist.Name,
			&artist.Bio,
			&artist.ProfileImageURL,
			&artist.HeroVideoURL,
			&artist.HeroVideoPosterURL,
			&artist.EngagementScore,
			&artist.TrackCount,
		); err != nil {
			return nil, err
		}

		artists = append(
			artists,
			artist,
		)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return artists, nil
}

// TrendingTracks returns tracks ranked by the engagement signals
// currently available in the music table.
//
// This is an MVP trending score. Once listening history exists,
// recent plays will become the strongest signal.
func (r *DiscoveryRepository) TrendingTracks(
	ctx context.Context,
	limit int,
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
		ORDER BY
			(
				likes * 3
				+ loves * 4
				+ rating * 2
			) DESC,
			date_posted DESC,
			id DESC
		LIMIT $1
	`

	return r.queryMusic(
		ctx,
		query,
		limit,
	)
}

// NewTracks returns the newest music uploaded to the catalog.
func (r *DiscoveryRepository) NewTracks(
	ctx context.Context,
	limit int,
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
		ORDER BY
			date_posted DESC,
			id DESC
		LIMIT $1
	`

	return r.queryMusic(
		ctx,
		query,
		limit,
	)
}

// NewReleases returns only published releases.
//
// Draft releases must never appear in public discovery.
func (r *DiscoveryRepository) NewReleases(
	ctx context.Context,
	limit int,
) ([]models.SearchRelease, error) {
	query := `
		SELECT
			r.id,
			r.artist_id,
			a.name,
			r.title,
			r.release_type,
			COALESCE(r.cover_image_url, ''),
			COALESCE(r.description, ''),
			r.release_date
		FROM releases r
		JOIN artists a
			ON a.id = r.artist_id
		WHERE r.is_published = TRUE
		ORDER BY
			COALESCE(
				r.release_date,
				r.created_at::date
			) DESC,
			r.created_at DESC,
			r.id DESC
		LIMIT $1
	`

	rows, err := r.DB.Query(
		ctx,
		query,
		limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	releases := make(
		[]models.SearchRelease,
		0,
	)

	for rows.Next() {
		var release models.SearchRelease

		if err := rows.Scan(
			&release.ID,
			&release.ArtistID,
			&release.ArtistName,
			&release.Title,
			&release.ReleaseType,
			&release.CoverImageURL,
			&release.Description,
			&release.ReleaseDate,
		); err != nil {
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

// PopularArtists ranks artists using engagement accumulated across
// all music that belongs to each artist.
//
// Legacy music without artist_id is deliberately excluded because
// it cannot be safely associated with an artist profile.
func (r *DiscoveryRepository) PopularArtists(
	ctx context.Context,
	limit int,
) ([]models.SearchArtist, error) {
	query := `
		SELECT
			a.id,
			a.name
		FROM artists a
		JOIN music m
			ON m.artist_id = a.id
		GROUP BY
			a.id,
			a.name
		ORDER BY
			COALESCE(SUM(m.likes), 0) DESC,
			COALESCE(SUM(m.loves), 0) DESC,
			COALESCE(AVG(m.rating), 0) DESC,
			COUNT(m.id) DESC,
			a.id DESC
		LIMIT $1
	`

	rows, err := r.DB.Query(
		ctx,
		query,
		limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	artists := make(
		[]models.SearchArtist,
		0,
	)

	for rows.Next() {
		var artist models.SearchArtist

		if err := rows.Scan(
			&artist.ID,
			&artist.Name,
		); err != nil {
			return nil, err
		}

		artists = append(
			artists,
			artist,
		)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return artists, nil
}

// Genres returns normalized genre names generated from real tracks.
//
// LOWER() prevents "Country" and "country" from appearing as two
// separate discovery categories.
func (r *DiscoveryRepository) Genres(
	ctx context.Context,
	limit int,
) ([]models.DiscoveryGenre, error) {
	query := `
		SELECT
			LOWER(TRIM(genre)) AS genre_name,
			COUNT(*)::int AS track_count
		FROM music
		WHERE TRIM(genre) <> ''
		GROUP BY LOWER(TRIM(genre))
		ORDER BY
			COUNT(*) DESC,
			LOWER(TRIM(genre)) ASC
		LIMIT $1
	`

	rows, err := r.DB.Query(
		ctx,
		query,
		limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	genres := make(
		[]models.DiscoveryGenre,
		0,
	)

	for rows.Next() {
		var genre models.DiscoveryGenre

		if err := rows.Scan(
			&genre.Name,
			&genre.TrackCount,
		); err != nil {
			return nil, err
		}

		genres = append(
			genres,
			genre,
		)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return genres, nil
}

func (r *DiscoveryRepository) queryMusic(
	ctx context.Context,
	query string,
	limit int,
) ([]models.Music, error) {
	rows, err := r.DB.Query(
		ctx,
		query,
		limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	music := make(
		[]models.Music,
		0,
	)

	for rows.Next() {
		var track models.Music

		if err := rows.Scan(
			&track.ID,
			&track.ArtistID,
			&track.ReleaseID,
			&track.TrackNumber,
			&track.ArtistName,
			&track.SongTitle,
			&track.Genre,
			&track.ImageURL,
			&track.ImagePublicID,
			&track.AudioURL,
			&track.AudioPublicID,
			&track.AudioKey,
			&track.Likes,
			&track.Loves,
			&track.Rating,
			&track.DatePosted,
		); err != nil {
			return nil, err
		}

		music = append(
			music,
			track,
		)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return music, nil
}
