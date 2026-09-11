CREATE TABLE artist_follows (
    user_id BIGINT NOT NULL
        REFERENCES users(id)
        ON DELETE CASCADE,

    artist_id BIGINT NOT NULL
        REFERENCES artists(id)
        ON DELETE CASCADE,

    created_at TIMESTAMPTZ NOT NULL
        DEFAULT NOW(),

    PRIMARY KEY (user_id, artist_id)
);

CREATE INDEX idx_artist_follows_artist_created_at
    ON artist_follows (
        artist_id,
        created_at DESC
    );
