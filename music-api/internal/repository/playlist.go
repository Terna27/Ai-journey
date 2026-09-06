package repository

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"music-api/internal/models"
)

var (
	ErrPlaylistNotFound      = errors.New("playlist not found")
	ErrPlaylistTrackExists   = errors.New("music already exists in playlist")
	ErrPlaylistTrackNotFound = errors.New("music is not in playlist")
)

type PlaylistRepository struct {
	DB *pgxpool.Pool
}

func NewPlaylistRepository(
	db *pgxpool.Pool,
) *PlaylistRepository {
	return &PlaylistRepository{
		DB: db,
	}
}

func (r *PlaylistRepository) Create(
	ctx context.Context,
	userID int,
	name string,
	description string,
	isPublic bool,
) (models.Playlist, error) {
	var playlist models.Playlist

	err := r.DB.QueryRow(
		ctx,
		`
		INSERT INTO playlists (
			user_id,
			name,
			description,
			is_public
		)
		VALUES ($1, $2, $3, $4)
		RETURNING
			id,
			user_id,
			name,
			description,
			is_public,
			created_at,
			updated_at
		`,
		userID,
		name,
		description,
		isPublic,
	).Scan(
		&playlist.ID,
		&playlist.UserID,
		&playlist.Name,
		&playlist.Description,
		&playlist.IsPublic,
		&playlist.CreatedAt,
		&playlist.UpdatedAt,
	)

	if err != nil {
		return models.Playlist{}, err
	}

	return playlist, nil
}

func (r *PlaylistRepository) GetByID(
	ctx context.Context,
	playlistID int64,
) (models.Playlist, error) {
	var playlist models.Playlist

	err := r.DB.QueryRow(
		ctx,
		`
		SELECT
			id,
			user_id,
			name,
			description,
			is_public,
			created_at,
			updated_at
		FROM playlists
		WHERE id = $1
		`,
		playlistID,
	).Scan(
		&playlist.ID,
		&playlist.UserID,
		&playlist.Name,
		&playlist.Description,
		&playlist.IsPublic,
		&playlist.CreatedAt,
		&playlist.UpdatedAt,
	)

	if errors.Is(err, pgx.ErrNoRows) {
		return models.Playlist{}, ErrPlaylistNotFound
	}

	if err != nil {
		return models.Playlist{}, err
	}

	return playlist, nil
}

func (r *PlaylistRepository) GetByUserID(
	ctx context.Context,
	userID int,
) ([]models.Playlist, error) {
	rows, err := r.DB.Query(
		ctx,
		`
		SELECT
			id,
			user_id,
			name,
			description,
			is_public,
			created_at,
			updated_at
		FROM playlists
		WHERE user_id = $1
		ORDER BY created_at DESC
		`,
		userID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	playlists := make(
		[]models.Playlist,
		0,
	)

	for rows.Next() {
		var playlist models.Playlist

		err := rows.Scan(
			&playlist.ID,
			&playlist.UserID,
			&playlist.Name,
			&playlist.Description,
			&playlist.IsPublic,
			&playlist.CreatedAt,
			&playlist.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}

		playlists = append(
			playlists,
			playlist,
		)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return playlists, nil
}

func (r *PlaylistRepository) Update(
	ctx context.Context,
	playlistID int64,
	userID int,
	name string,
	description string,
	isPublic bool,
) (models.Playlist, error) {
	var playlist models.Playlist

	err := r.DB.QueryRow(
		ctx,
		`
		UPDATE playlists
		SET
			name = $1,
			description = $2,
			is_public = $3,
			updated_at = NOW()
		WHERE id = $4
		  AND user_id = $5
		RETURNING
			id,
			user_id,
			name,
			description,
			is_public,
			created_at,
			updated_at
		`,
		name,
		description,
		isPublic,
		playlistID,
		userID,
	).Scan(
		&playlist.ID,
		&playlist.UserID,
		&playlist.Name,
		&playlist.Description,
		&playlist.IsPublic,
		&playlist.CreatedAt,
		&playlist.UpdatedAt,
	)

	if errors.Is(err, pgx.ErrNoRows) {
		return models.Playlist{}, ErrPlaylistNotFound
	}

	if err != nil {
		return models.Playlist{}, err
	}

	return playlist, nil
}

func (r *PlaylistRepository) Delete(
	ctx context.Context,
	playlistID int64,
	userID int,
) error {
	commandTag, err := r.DB.Exec(
		ctx,
		`
		DELETE FROM playlists
		WHERE id = $1
		  AND user_id = $2
		`,
		playlistID,
		userID,
	)
	if err != nil {
		return err
	}

	if commandTag.RowsAffected() == 0 {
		return ErrPlaylistNotFound
	}

	return nil
}

func (r *PlaylistRepository) AddTrack(
	ctx context.Context,
	playlistID int64,
	userID int,
	musicID int,
) error {
	tx, err := r.DB.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	var ownerID int

	err = tx.QueryRow(
		ctx,
		`
		SELECT user_id
		FROM playlists
		WHERE id = $1
		FOR UPDATE
		`,
		playlistID,
	).Scan(
		&ownerID,
	)

	if errors.Is(err, pgx.ErrNoRows) {
		return ErrPlaylistNotFound
	}

	if err != nil {
		return err
	}

	if ownerID != userID {
		return ErrPlaylistNotFound
	}

	var musicExists bool

	err = tx.QueryRow(
		ctx,
		`
		SELECT EXISTS (
			SELECT 1
			FROM music
			WHERE id = $1
		)
		`,
		musicID,
	).Scan(
		&musicExists,
	)
	if err != nil {
		return err
	}

	if !musicExists {
		return pgx.ErrNoRows
	}

	var nextPosition int

	err = tx.QueryRow(
		ctx,
		`
		SELECT
			COALESCE(
				MAX(position),
				0
			) + 1
		FROM playlist_tracks
		WHERE playlist_id = $1
		`,
		playlistID,
	).Scan(
		&nextPosition,
	)
	if err != nil {
		return err
	}

	commandTag, err := tx.Exec(
		ctx,
		`
		INSERT INTO playlist_tracks (
			playlist_id,
			music_id,
			position
		)
		VALUES ($1, $2, $3)
		ON CONFLICT DO NOTHING
		`,
		playlistID,
		musicID,
		nextPosition,
	)
	if err != nil {
		return err
	}

	if commandTag.RowsAffected() == 0 {
		return ErrPlaylistTrackExists
	}

	err = tx.Commit(ctx)
	if err != nil {
		return err
	}

	return nil
}

func (r *PlaylistRepository) RemoveTrack(
	ctx context.Context,
	playlistID int64,
	userID int,
	musicID int,
) error {
	tx, err := r.DB.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	var ownerID int

	err = tx.QueryRow(
		ctx,
		`
		SELECT user_id
		FROM playlists
		WHERE id = $1
		FOR UPDATE
		`,
		playlistID,
	).Scan(
		&ownerID,
	)

	if errors.Is(err, pgx.ErrNoRows) {
		return ErrPlaylistNotFound
	}

	if err != nil {
		return err
	}

	if ownerID != userID {
		return ErrPlaylistNotFound
	}

	commandTag, err := tx.Exec(
		ctx,
		`
		DELETE FROM playlist_tracks
		WHERE playlist_id = $1
		  AND music_id = $2
		`,
		playlistID,
		musicID,
	)
	if err != nil {
		return err
	}

	if commandTag.RowsAffected() == 0 {
		return ErrPlaylistTrackNotFound
	}

	_, err = tx.Exec(
		ctx,
		`
		WITH ordered AS (
			SELECT
				playlist_id,
				music_id,
				ROW_NUMBER() OVER (
					ORDER BY position, added_at
				)::INT AS new_position
			FROM playlist_tracks
			WHERE playlist_id = $1
		)
		UPDATE playlist_tracks pt
		SET position = ordered.new_position
		FROM ordered
		WHERE pt.playlist_id = ordered.playlist_id
		  AND pt.music_id = ordered.music_id
		`,
		playlistID,
	)
	if err != nil {
		return err
	}

	err = tx.Commit(ctx)
	if err != nil {
		return err
	}

	return nil
}

func (r *PlaylistRepository) GetTracks(
	ctx context.Context,
	playlistID int64,
) ([]models.Music, error) {
	rows, err := r.DB.Query(
		ctx,
		`
		SELECT
			m.id,
			m.artist_id,
			m.artist_name,
			m.song_title,
			m.genre,
			m.image_url,
			m.image_public_id,
			m.audio_url,
			m.audio_public_id,
			m.audio_key,
			m.likes,
			m.loves,
			m.rating,
			m.date_posted
		FROM playlist_tracks pt
		INNER JOIN music m
			ON m.id = pt.music_id
		WHERE pt.playlist_id = $1
		ORDER BY
			pt.position ASC,
			pt.added_at ASC
		`,
		playlistID,
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
		var track models.Music

		err := rows.Scan(
			&track.ID,
			&track.ArtistID,
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
		)
		if err != nil {
			return nil, err
		}

		tracks = append(
			tracks,
			track,
		)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return tracks, nil
}
