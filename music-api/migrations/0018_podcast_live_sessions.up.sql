BEGIN;

-- =========================================================
-- PODCAST LIVE SESSIONS (Phase 4.2)
--
-- A live session is the RUNTIME record of one live audio
-- broadcast for one podcast episode. The episode remains
-- the editorial/content record; this table tracks the
-- live runtime (scheduling, provider room, recording).
--
-- Live audio itself is NEVER streamed through the Go HTTP
-- server. The provider (LiveKit/WebRTC) transports media;
-- the backend owns authentication, ownership, state, and
-- short-lived participant tokens.
-- =========================================================

CREATE TABLE podcast_live_sessions (
    id UUID PRIMARY KEY
        DEFAULT gen_random_uuid(),

    episode_id BIGINT NOT NULL
        REFERENCES podcast_episodes(id)
        ON DELETE CASCADE,

    host_user_id BIGINT NOT NULL
        REFERENCES users(id)
        ON DELETE CASCADE,

    provider TEXT NOT NULL
        DEFAULT 'LIVEKIT',

    -- Generated server-side, unique and unpredictable so
    -- room names cannot be guessed or collide.
    provider_room_name TEXT NOT NULL,

    -- Filled in when the provider reports the room's SID
    -- (set after the room is first created/started).
    provider_room_sid TEXT,

    status TEXT NOT NULL
        DEFAULT 'SCHEDULED',

    scheduled_start_at TIMESTAMPTZ NOT NULL,

    started_at TIMESTAMPTZ,

    ended_at TIMESTAMPTZ,

    recording_status TEXT NOT NULL
        DEFAULT 'NONE',

    recording_provider_asset_id TEXT,

    recording_url TEXT,

    created_at TIMESTAMPTZ NOT NULL
        DEFAULT NOW(),

    updated_at TIMESTAMPTZ NOT NULL
        DEFAULT NOW(),

    CONSTRAINT podcast_live_sessions_provider_valid
        CHECK (
            provider IN ('LIVEKIT')
        ),

    CONSTRAINT podcast_live_sessions_room_name_not_blank
        CHECK (
            length(trim(provider_room_name)) > 0
        ),

    CONSTRAINT podcast_live_sessions_status_valid
        CHECK (
            status IN (
                'SCHEDULED',
                'LIVE',
                'ENDED',
                'CANCELLED'
            )
        ),

    CONSTRAINT podcast_live_sessions_recording_status_valid
        CHECK (
            recording_status IN (
                'NONE',
                'STARTING',
                'RECORDING',
                'PROCESSING',
                'READY',
                'FAILED'
            )
        ),

    -- LIVE and ENDED sessions must have actually started.
    CONSTRAINT podcast_live_sessions_started_required
        CHECK (
            status NOT IN ('LIVE', 'ENDED')
            OR started_at IS NOT NULL
        ),

    -- Only an ENDED session has an end timestamp.
    CONSTRAINT podcast_live_sessions_ended_required
        CHECK (
            status <> 'ENDED'
            OR ended_at IS NOT NULL
        ),

    -- ended_at can never precede started_at.
    CONSTRAINT podcast_live_sessions_ended_not_before_started
        CHECK (
            started_at IS NULL
            OR ended_at IS NULL
            OR ended_at >= started_at
        ),

    -- A finished broadcast must have a recording result
    -- once processing has concluded.
    CONSTRAINT podcast_live_sessions_recording_ready_has_url
        CHECK (
            recording_status <> 'READY'
            OR recording_url IS NOT NULL
        )
);

-- One room name per provider, forever: room names are
-- generated server-side and must never collide.
CREATE UNIQUE INDEX idx_podcast_live_sessions_room_unique
ON podcast_live_sessions (
    provider,
    provider_room_name
);

-- =========================================================
-- ONE LIVE SESSION PER EPISODE
--
-- An episode can have at most one non-cancelled live
-- session (SCHEDULED, LIVE, or ENDED are all mutually
-- exclusive, and ENDED is terminal). A CANCELLED session
-- keeps its row for audit but no longer occupies the
-- episode's live slot, so an owner can reschedule a
-- broadcast that was cancelled before it started.
--
-- The partial unique index also rejects duplicate
-- scheduling while any non-cancelled session exists.
-- =========================================================

CREATE UNIQUE INDEX idx_podcast_live_sessions_episode_unique
ON podcast_live_sessions (
    episode_id
)
WHERE status <> 'CANCELLED';

-- Host dashboard: my sessions by state and schedule.
CREATE INDEX idx_podcast_live_sessions_host_status
ON podcast_live_sessions (
    host_user_id,
    status,
    scheduled_start_at
);

-- Public discovery of upcoming broadcasts.
CREATE INDEX idx_podcast_live_sessions_upcoming
ON podcast_live_sessions (
    scheduled_start_at,
    id
)
WHERE status = 'SCHEDULED';

-- Public discovery of currently-live broadcasts.
CREATE INDEX idx_podcast_live_sessions_live
ON podcast_live_sessions (
    started_at DESC,
    id
)
WHERE status = 'LIVE';

-- Episode -> live session lookup.
CREATE INDEX idx_podcast_live_sessions_episode
ON podcast_live_sessions (
    episode_id
);

-- Background/worker scanning of recordings that are still
-- being produced or processed.
CREATE INDEX idx_podcast_live_sessions_recording_active
ON podcast_live_sessions (
    recording_status,
    updated_at
)
WHERE recording_status IN (
    'STARTING',
    'RECORDING',
    'PROCESSING'
);

COMMIT;
