package repository

import (
	"context"

	"music-api/internal/models"

	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	searchTextSimilarityThreshold  = 0.12
	searchGenreSimilarityThreshold = 0.25
)

type SearchRepository struct {
	DB *pgxpool.Pool
}

func NewSearchRepository(
	db *pgxpool.Pool,
) *SearchRepository {
	return &SearchRepository{
		DB: db,
	}
}

// SearchTracks searches playable public music using:
//
//   - exact matches
//   - prefix matches
//   - substring matches
//   - pg_trgm fuzzy similarity
//
// Tracks without a usable audio URL are excluded from public search.
//
// genre is optional. When provided, it also supports fuzzy genre
// filtering.
//
// sort supports:
//
//	relevance
//	newest
//	popular
func (r *SearchRepository) SearchTracks(
	ctx context.Context,
	query string,
	genre string,
	sort string,
	limit int,
	offset int,
) ([]models.Music, error) {
	rows, err := r.DB.Query(
		ctx,
		`
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
		WHERE
			COALESCE(TRIM(audio_url), '') <> ''
			AND (
				song_title ILIKE '%' || $1 || '%'
				OR artist_name ILIKE '%' || $1 || '%'
				OR genre ILIKE '%' || $1 || '%'
				OR similarity(song_title, $1) >= $6
				OR similarity(artist_name, $1) >= $6
				OR similarity(genre, $1) >= $7
			)
			AND (
				$2 = ''
				OR genre ILIKE '%' || $2 || '%'
				OR similarity(genre, $2) >= $7
			)
		ORDER BY
			CASE
				WHEN $3 = 'relevance' THEN
					CASE
						WHEN LOWER(song_title) = LOWER($1) THEN 0
						WHEN LOWER(song_title) LIKE LOWER($1) || '%' THEN 1
						WHEN song_title ILIKE '%' || $1 || '%' THEN 2

						WHEN LOWER(artist_name) = LOWER($1) THEN 3
						WHEN LOWER(artist_name) LIKE LOWER($1) || '%' THEN 4
						WHEN artist_name ILIKE '%' || $1 || '%' THEN 5

						WHEN LOWER(genre) = LOWER($1) THEN 6
						WHEN LOWER(genre) LIKE LOWER($1) || '%' THEN 7
						WHEN genre ILIKE '%' || $1 || '%' THEN 8

						ELSE 9
					END
				ELSE 0
			END ASC,

			CASE
				WHEN $3 = 'relevance'
					THEN GREATEST(
						similarity(song_title, $1),
						similarity(artist_name, $1),
						similarity(genre, $1)
					)
			END DESC NULLS LAST,

			CASE
				WHEN $3 = 'popular'
					THEN likes
			END DESC NULLS LAST,

			CASE
				WHEN $3 = 'popular'
					THEN loves
			END DESC NULLS LAST,

			CASE
				WHEN $3 = 'popular'
					THEN rating
			END DESC NULLS LAST,

			CASE
				WHEN $3 = 'newest'
					THEN date_posted
			END DESC NULLS LAST,

			CASE
				WHEN $3 = 'relevance'
					THEN likes
			END DESC NULLS LAST,

			date_posted DESC,
			id DESC
		LIMIT $4
		OFFSET $5
		`,
		query,
		genre,
		sort,
		limit,
		offset,
		searchTextSimilarityThreshold,
		searchGenreSimilarityThreshold,
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

// CountTracks returns the total number of playable tracks matching
// the same search conditions used by SearchTracks.
func (r *SearchRepository) CountTracks(
	ctx context.Context,
	query string,
	genre string,
) (int, error) {
	var total int

	err := r.DB.QueryRow(
		ctx,
		`
		SELECT COUNT(*)
		FROM music
		WHERE
			COALESCE(TRIM(audio_url), '') <> ''
			AND (
				song_title ILIKE '%' || $1 || '%'
				OR artist_name ILIKE '%' || $1 || '%'
				OR genre ILIKE '%' || $1 || '%'
				OR similarity(song_title, $1) >= $3
				OR similarity(artist_name, $1) >= $3
				OR similarity(genre, $1) >= $4
			)
			AND (
				$2 = ''
				OR genre ILIKE '%' || $2 || '%'
				OR similarity(genre, $2) >= $4
			)
		`,
		query,
		genre,
		searchTextSimilarityThreshold,
		searchGenreSimilarityThreshold,
	).Scan(
		&total,
	)

	if err != nil {
		return 0, err
	}

	return total, nil
}

// SearchArtists searches public artist profiles by display name.
//
// Matching supports:
//
//   - exact matches
//   - prefix matches
//   - substring matches
//   - pg_trgm fuzzy similarity
//
// Popularity is calculated from likes on music owned by the artist.
// No private user/account data is selected.
func (r *SearchRepository) SearchArtists(
	ctx context.Context,
	query string,
	sort string,
	limit int,
	offset int,
) ([]models.SearchArtist, error) {
	rows, err := r.DB.Query(
		ctx,
		`
		SELECT
			a.id,
			a.name
		FROM artists a
		LEFT JOIN music m
			ON m.artist_id = a.id
		WHERE
			a.name ILIKE '%' || $1 || '%'
			OR similarity(a.name, $1) >= $5
		GROUP BY
			a.id,
			a.name
		ORDER BY
			CASE
				WHEN $2 = 'relevance' THEN
					CASE
						WHEN LOWER(a.name) = LOWER($1) THEN 0
						WHEN LOWER(a.name) LIKE LOWER($1) || '%' THEN 1
						WHEN a.name ILIKE '%' || $1 || '%' THEN 2
						ELSE 3
					END
				ELSE 0
			END ASC,

			CASE
				WHEN $2 = 'relevance'
					THEN similarity(a.name, $1)
			END DESC NULLS LAST,

			CASE
				WHEN $2 = 'popular'
					THEN COALESCE(SUM(m.likes), 0)
			END DESC NULLS LAST,

			CASE
				WHEN $2 = 'newest'
					THEN a.id
			END DESC NULLS LAST,

			CASE
				WHEN $2 = 'relevance'
					THEN COALESCE(SUM(m.likes), 0)
			END DESC NULLS LAST,

			a.name ASC,
			a.id DESC
		LIMIT $3
		OFFSET $4
		`,
		query,
		sort,
		limit,
		offset,
		searchTextSimilarityThreshold,
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

		err := rows.Scan(
			&artist.ID,
			&artist.Name,
		)
		if err != nil {
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

// CountArtists returns the total number of public artist profiles
// matching the same search conditions used by SearchArtists.
func (r *SearchRepository) CountArtists(
	ctx context.Context,
	query string,
) (int, error) {
	var total int

	err := r.DB.QueryRow(
		ctx,
		`
		SELECT COUNT(*)
		FROM artists
		WHERE
			name ILIKE '%' || $1 || '%'
			OR similarity(name, $1) >= $2
		`,
		query,
		searchTextSimilarityThreshold,
	).Scan(
		&total,
	)

	if err != nil {
		return 0, err
	}

	return total, nil
}

// SearchReleases searches published releases only.
//
// Draft releases must never appear in public search.
//
// Matching supports fuzzy searches on:
//
//   - release title
//   - artist name
//
// Popularity is calculated from likes on tracks currently attached
// to the release.
func (r *SearchRepository) SearchReleases(
	ctx context.Context,
	query string,
	sort string,
	limit int,
	offset int,
) ([]models.SearchRelease, error) {
	rows, err := r.DB.Query(
		ctx,
		`
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
		INNER JOIN artists a
			ON a.id = r.artist_id
		LEFT JOIN music m
			ON m.release_id = r.id
		WHERE
			r.is_published = TRUE
			AND (
				r.title ILIKE '%' || $1 || '%'
				OR a.name ILIKE '%' || $1 || '%'
				OR similarity(r.title, $1) >= $5
				OR similarity(a.name, $1) >= $5
			)
		GROUP BY
			r.id,
			r.artist_id,
			a.name,
			r.title,
			r.release_type,
			r.cover_image_url,
			r.description,
			r.release_date,
			r.created_at
		ORDER BY
			CASE
				WHEN $2 = 'relevance' THEN
					CASE
						WHEN LOWER(r.title) = LOWER($1) THEN 0
						WHEN LOWER(r.title) LIKE LOWER($1) || '%' THEN 1
						WHEN r.title ILIKE '%' || $1 || '%' THEN 2

						WHEN LOWER(a.name) = LOWER($1) THEN 3
						WHEN LOWER(a.name) LIKE LOWER($1) || '%' THEN 4
						WHEN a.name ILIKE '%' || $1 || '%' THEN 5

						ELSE 6
					END
				ELSE 0
			END ASC,

			CASE
				WHEN $2 = 'relevance'
					THEN GREATEST(
						similarity(r.title, $1),
						similarity(a.name, $1)
					)
			END DESC NULLS LAST,

			CASE
				WHEN $2 = 'popular'
					THEN COALESCE(SUM(m.likes), 0)
			END DESC NULLS LAST,

			CASE
				WHEN $2 = 'newest'
					THEN COALESCE(
						r.release_date,
						r.created_at::date
					)
			END DESC NULLS LAST,

			CASE
				WHEN $2 = 'relevance'
					THEN COALESCE(SUM(m.likes), 0)
			END DESC NULLS LAST,

			r.release_date DESC NULLS LAST,
			r.created_at DESC,
			r.id DESC
		LIMIT $3
		OFFSET $4
		`,
		query,
		sort,
		limit,
		offset,
		searchTextSimilarityThreshold,
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

		err := rows.Scan(
			&release.ID,
			&release.ArtistID,
			&release.ArtistName,
			&release.Title,
			&release.ReleaseType,
			&release.CoverImageURL,
			&release.Description,
			&release.ReleaseDate,
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

// CountReleases returns the number of published releases matching
// the same search conditions used by SearchReleases.
func (r *SearchRepository) CountReleases(
	ctx context.Context,
	query string,
) (int, error) {
	var total int

	err := r.DB.QueryRow(
		ctx,
		`
		SELECT COUNT(*)
		FROM releases r
		INNER JOIN artists a
			ON a.id = r.artist_id
		WHERE
			r.is_published = TRUE
			AND (
				r.title ILIKE '%' || $1 || '%'
				OR a.name ILIKE '%' || $1 || '%'
				OR similarity(r.title, $1) >= $2
				OR similarity(a.name, $1) >= $2
			)
		`,
		query,
		searchTextSimilarityThreshold,
	).Scan(
		&total,
	)

	if err != nil {
		return 0, err
	}

	return total, nil
}
