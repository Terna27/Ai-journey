CREATE TABLE IF NOT EXISTS playlists (
    id BIGSERIAL PRIMARY KEY,

    user_id BIGINT NOT NULL
        REFERENCES users(id)
        ON DELETE CASCADE,

    name TEXT NOT NULL,

    description TEXT NOT NULL DEFAULT '',

    is_public BOOLEAN NOT NULL DEFAULT FALSE,

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_playlists_user_id
ON playlists(user_id);

CREATE INDEX IF NOT EXISTS idx_playlists_user_created
ON playlists(user_id, created_at DESC);


CREATE TABLE IF NOT EXISTS playlist_tracks (
    playlist_id BIGINT NOT NULL
        REFERENCES playlists(id)
        ON DELETE CASCADE,

    music_id INT NOT NULL
        REFERENCES music(id)
        ON DELETE CASCADE,

    position INT NOT NULL,

    added_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    PRIMARY KEY (
        playlist_id,
        music_id
    )
);

CREATE INDEX IF NOT EXISTS idx_playlist_tracks_playlist_position
ON playlist_tracks(
    playlist_id,
    position
);

CREATE INDEX IF NOT EXISTS idx_playlist_tracks_music_id
ON playlist_tracks(music_id);


ALTER TABLE playlist_tracks
ADD CONSTRAINT playlist_tracks_position_positive
CHECK (position > 0);