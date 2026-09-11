CREATE TABLE track_lyrics (
    music_id BIGINT PRIMARY KEY
        REFERENCES music(id)
        ON DELETE CASCADE,

    plain_lyrics TEXT NOT NULL DEFAULT '',

    synced_lines JSONB NOT NULL DEFAULT '[]'::jsonb,

    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),

    CONSTRAINT track_lyrics_synced_lines_array
        CHECK (
            jsonb_typeof(synced_lines) = 'array'
        )
);

CREATE INDEX idx_track_lyrics_updated_at
    ON track_lyrics(updated_at DESC);