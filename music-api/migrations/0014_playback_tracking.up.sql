BEGIN;

CREATE TABLE playback_sessions (
    id UUID PRIMARY KEY,

    user_id BIGINT NOT NULL
        REFERENCES users(id)
        ON DELETE CASCADE,

    music_id BIGINT NOT NULL
        REFERENCES music(id)
        ON DELETE CASCADE,

    duration_ms BIGINT NOT NULL DEFAULT 0,

    position_ms BIGINT NOT NULL DEFAULT 0,

    listened_ms BIGINT NOT NULL DEFAULT 0,

    qualified BOOLEAN NOT NULL DEFAULT FALSE,

    qualified_at TIMESTAMPTZ,

    completed BOOLEAN NOT NULL DEFAULT FALSE,

    completed_at TIMESTAMPTZ,

    started_at TIMESTAMPTZ NOT NULL DEFAULT now(),

    last_activity_at TIMESTAMPTZ NOT NULL DEFAULT now(),

    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),

    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),

    CONSTRAINT playback_sessions_duration_non_negative
        CHECK (duration_ms >= 0),

    CONSTRAINT playback_sessions_position_non_negative
        CHECK (position_ms >= 0),

    CONSTRAINT playback_sessions_listened_non_negative
        CHECK (listened_ms >= 0),

    CONSTRAINT playback_sessions_position_within_duration
        CHECK (
            duration_ms = 0
            OR position_ms <= duration_ms
        ),

    CONSTRAINT playback_sessions_qualified_timestamp
        CHECK (
            qualified = FALSE
            OR qualified_at IS NOT NULL
        ),

    CONSTRAINT playback_sessions_completed_timestamp
        CHECK (
            completed = FALSE
            OR completed_at IS NOT NULL
        )
);

CREATE TABLE listening_history (
    user_id BIGINT NOT NULL
        REFERENCES users(id)
        ON DELETE CASCADE,

    music_id BIGINT NOT NULL
        REFERENCES music(id)
        ON DELETE CASCADE,

    last_position_ms BIGINT NOT NULL DEFAULT 0,

    duration_ms BIGINT NOT NULL DEFAULT 0,

    qualified_play_count BIGINT NOT NULL DEFAULT 0,

    last_played_at TIMESTAMPTZ NOT NULL DEFAULT now(),

    last_qualified_at TIMESTAMPTZ,

    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),

    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),

    PRIMARY KEY (user_id, music_id),

    CONSTRAINT listening_history_position_non_negative
        CHECK (last_position_ms >= 0),

    CONSTRAINT listening_history_duration_non_negative
        CHECK (duration_ms >= 0),

    CONSTRAINT listening_history_qualified_count_non_negative
        CHECK (qualified_play_count >= 0),

    CONSTRAINT listening_history_position_within_duration
        CHECK (
            duration_ms = 0
            OR last_position_ms <= duration_ms
        )
);

CREATE INDEX idx_playback_sessions_user_started
ON playback_sessions (
    user_id,
    started_at DESC
);

CREATE INDEX idx_playback_sessions_music_qualified
ON playback_sessions (
    music_id,
    qualified_at DESC
)
WHERE qualified = TRUE;

CREATE INDEX idx_playback_sessions_user_active
ON playback_sessions (
    user_id,
    last_activity_at DESC
)
WHERE completed = FALSE;

CREATE INDEX idx_playback_sessions_qualified_recent
ON playback_sessions (
    qualified_at DESC,
    music_id
)
WHERE qualified = TRUE;

CREATE INDEX idx_listening_history_user_recent
ON listening_history (
    user_id,
    last_played_at DESC
);

CREATE INDEX idx_listening_history_user_qualified
ON listening_history (
    user_id,
    qualified_play_count DESC,
    last_played_at DESC
);

COMMIT;
