BEGIN;

-- =========================================================
-- PODCAST PLAYBACK TRACKING (Phase 4.1)
--
-- Dedicated podcast playback tables. The existing music
-- playback tables (playback_sessions / listening_history)
-- are deliberately NOT made polymorphic: keeping separate
-- tables preserves working music analytics and allows
-- podcast-specific resume/completion behavior to evolve
-- independently.
-- =========================================================

CREATE TABLE podcast_playback_sessions (
    id UUID PRIMARY KEY,

    user_id BIGINT NOT NULL
        REFERENCES users(id)
        ON DELETE CASCADE,

    episode_id BIGINT NOT NULL
        REFERENCES podcast_episodes(id)
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

    CONSTRAINT podcast_playback_sessions_duration_non_negative
        CHECK (duration_ms >= 0),

    CONSTRAINT podcast_playback_sessions_position_non_negative
        CHECK (position_ms >= 0),

    CONSTRAINT podcast_playback_sessions_listened_non_negative
        CHECK (listened_ms >= 0),

    CONSTRAINT podcast_playback_sessions_position_within_duration
        CHECK (
            duration_ms = 0
            OR position_ms <= duration_ms
        ),

    CONSTRAINT podcast_playback_sessions_qualified_timestamp
        CHECK (
            qualified = FALSE
            OR qualified_at IS NOT NULL
        ),

    CONSTRAINT podcast_playback_sessions_completed_timestamp
        CHECK (
            completed = FALSE
            OR completed_at IS NOT NULL
        )
);

CREATE TABLE podcast_listening_history (
    user_id BIGINT NOT NULL
        REFERENCES users(id)
        ON DELETE CASCADE,

    episode_id BIGINT NOT NULL
        REFERENCES podcast_episodes(id)
        ON DELETE CASCADE,

    last_position_ms BIGINT NOT NULL DEFAULT 0,
    duration_ms BIGINT NOT NULL DEFAULT 0,
    qualified_play_count BIGINT NOT NULL DEFAULT 0,

    completed BOOLEAN NOT NULL DEFAULT FALSE,

    last_played_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    last_qualified_at TIMESTAMPTZ,
    completed_at TIMESTAMPTZ,

    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),

    PRIMARY KEY (user_id, episode_id),

    CONSTRAINT podcast_listening_history_position_non_negative
        CHECK (last_position_ms >= 0),

    CONSTRAINT podcast_listening_history_duration_non_negative
        CHECK (duration_ms >= 0),

    CONSTRAINT podcast_listening_history_qualified_count_non_negative
        CHECK (qualified_play_count >= 0),

    CONSTRAINT podcast_listening_history_position_within_duration
        CHECK (
            duration_ms = 0
            OR last_position_ms <= duration_ms
        ),

    CONSTRAINT podcast_listening_history_completed_timestamp
        CHECK (
            completed = FALSE
            OR completed_at IS NOT NULL
        )
);

-- User recent sessions.
CREATE INDEX idx_podcast_playback_sessions_user_started
ON podcast_playback_sessions (
    user_id,
    started_at DESC
);

-- Episode qualified plays (future podcast analytics).
CREATE INDEX idx_podcast_playback_sessions_episode_qualified
ON podcast_playback_sessions (
    episode_id,
    qualified_at DESC
)
WHERE qualified = TRUE;

-- Incomplete user sessions.
CREATE INDEX idx_podcast_playback_sessions_user_active
ON podcast_playback_sessions (
    user_id,
    last_activity_at DESC
)
WHERE completed = FALSE;

-- Recent qualified podcast plays.
CREATE INDEX idx_podcast_playback_sessions_qualified_recent
ON podcast_playback_sessions (
    qualified_at DESC,
    episode_id
)
WHERE qualified = TRUE;

-- User recent history.
CREATE INDEX idx_podcast_listening_history_user_recent
ON podcast_listening_history (
    user_id,
    last_played_at DESC
);

-- Continue Listening: meaningful progress, not completed.
CREATE INDEX idx_podcast_listening_history_user_continue
ON podcast_listening_history (
    user_id,
    last_played_at DESC
)
WHERE completed = FALSE
  AND last_position_ms > 0;

-- Qualified history for future analytics.
CREATE INDEX idx_podcast_listening_history_user_qualified
ON podcast_listening_history (
    user_id,
    qualified_play_count DESC,
    last_played_at DESC
);

COMMIT;
