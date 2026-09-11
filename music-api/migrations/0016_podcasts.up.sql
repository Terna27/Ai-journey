BEGIN;

-- =========================================================
-- PODCASTS
--
-- Podcasts belong directly to user accounts rather than
-- artist profiles.
--
-- This keeps podcast creation available to any eligible
-- account while still allowing artists to host podcasts.
-- =========================================================

CREATE TABLE podcasts (
    id BIGSERIAL PRIMARY KEY,

    owner_user_id BIGINT NOT NULL
        REFERENCES users(id)
        ON DELETE CASCADE,

    title TEXT NOT NULL,

    slug TEXT NOT NULL,

    description TEXT NOT NULL
        DEFAULT '',

    category TEXT NOT NULL
        DEFAULT '',

    artwork_url TEXT NOT NULL
        DEFAULT '',

    artwork_public_id TEXT NOT NULL
        DEFAULT '',

    status TEXT NOT NULL
        DEFAULT 'DRAFT',

    is_explicit BOOLEAN NOT NULL
        DEFAULT FALSE,

    published_at TIMESTAMPTZ,

    created_at TIMESTAMPTZ NOT NULL
        DEFAULT NOW(),

    updated_at TIMESTAMPTZ NOT NULL
        DEFAULT NOW(),

    CONSTRAINT podcasts_title_not_blank
        CHECK (
            length(trim(title)) > 0
        ),

    CONSTRAINT podcasts_slug_not_blank
        CHECK (
            length(trim(slug)) > 0
        ),

    CONSTRAINT podcasts_status_valid
        CHECK (
            status IN (
                'DRAFT',
                'PUBLISHED',
                'ARCHIVED'
            )
        ),

    CONSTRAINT podcasts_artwork_pair_consistent
        CHECK (
            (
                artwork_url = ''
                AND artwork_public_id = ''
            )
            OR
            (
                artwork_url <> ''
                AND artwork_public_id <> ''
            )
        )
);


-- Slugs are case-insensitively unique.
CREATE UNIQUE INDEX idx_podcasts_slug_unique
ON podcasts (
    lower(slug)
);


-- Fast owner dashboard lookup.
CREATE INDEX idx_podcasts_owner_created
ON podcasts (
    owner_user_id,
    created_at DESC
);


-- Public podcast discovery.
CREATE INDEX idx_podcasts_public
ON podcasts (
    status,
    published_at DESC,
    id DESC
);


-- Category discovery.
CREATE INDEX idx_podcasts_category_public
ON podcasts (
    category,
    status,
    published_at DESC
);


-- Fuzzy title search.
--
-- pg_trgm is already enabled by migration 0011.
CREATE INDEX idx_podcasts_title_trgm
ON podcasts
USING GIN (
    title gin_trgm_ops
);


CREATE INDEX idx_podcasts_description_trgm
ON podcasts
USING GIN (
    description gin_trgm_ops
);


-- =========================================================
-- PODCAST EPISODES
--
-- Episodes deliberately do not use the music table.
--
-- This allows podcast-specific behavior such as seasons,
-- episode numbers, scheduled publishing, live states,
-- podcast history, and future podcast analytics.
-- =========================================================

CREATE TABLE podcast_episodes (
    id BIGSERIAL PRIMARY KEY,

    podcast_id BIGINT NOT NULL
        REFERENCES podcasts(id)
        ON DELETE CASCADE,

    title TEXT NOT NULL,

    slug TEXT NOT NULL,

    description TEXT NOT NULL
        DEFAULT '',

    season_number INT NOT NULL
        DEFAULT 1,

    episode_number INT,

    episode_type TEXT NOT NULL
        DEFAULT 'FULL',

    audio_url TEXT NOT NULL
        DEFAULT '',

    audio_public_id TEXT NOT NULL
        DEFAULT '',

    artwork_url TEXT NOT NULL
        DEFAULT '',

    artwork_public_id TEXT NOT NULL
        DEFAULT '',

    duration_ms BIGINT NOT NULL
        DEFAULT 0,

    status TEXT NOT NULL
        DEFAULT 'DRAFT',

    is_explicit BOOLEAN NOT NULL
        DEFAULT FALSE,

    scheduled_at TIMESTAMPTZ,

    published_at TIMESTAMPTZ,

    created_at TIMESTAMPTZ NOT NULL
        DEFAULT NOW(),

    updated_at TIMESTAMPTZ NOT NULL
        DEFAULT NOW(),

    CONSTRAINT podcast_episodes_title_not_blank
        CHECK (
            length(trim(title)) > 0
        ),

    CONSTRAINT podcast_episodes_slug_not_blank
        CHECK (
            length(trim(slug)) > 0
        ),

    CONSTRAINT podcast_episodes_season_positive
        CHECK (
            season_number > 0
        ),

    CONSTRAINT podcast_episodes_number_positive
        CHECK (
            episode_number IS NULL
            OR episode_number > 0
        ),

    CONSTRAINT podcast_episodes_duration_nonnegative
        CHECK (
            duration_ms >= 0
        ),

    CONSTRAINT podcast_episodes_type_valid
        CHECK (
            episode_type IN (
                'FULL',
                'TRAILER',
                'BONUS'
            )
        ),

    CONSTRAINT podcast_episodes_status_valid
        CHECK (
            status IN (
                'DRAFT',
                'SCHEDULED',
                'PUBLISHED',
                'LIVE',
                'ENDED',
                'ARCHIVED'
            )
        ),

    CONSTRAINT podcast_episodes_audio_pair_consistent
        CHECK (
            (
                audio_url = ''
                AND audio_public_id = ''
            )
            OR
            (
                audio_url <> ''
                AND audio_public_id <> ''
            )
        ),

    CONSTRAINT podcast_episodes_artwork_pair_consistent
        CHECK (
            (
                artwork_url = ''
                AND artwork_public_id = ''
            )
            OR
            (
                artwork_url <> ''
                AND artwork_public_id <> ''
            )
        )
);


-- Episode URLs only need to be unique inside their podcast.
CREATE UNIQUE INDEX idx_podcast_episodes_slug_unique
ON podcast_episodes (
    podcast_id,
    lower(slug)
);


-- Prevent duplicate numbered episodes inside one season.
--
-- Episodes without an episode_number are allowed.
CREATE UNIQUE INDEX idx_podcast_episodes_number_unique
ON podcast_episodes (
    podcast_id,
    season_number,
    episode_number
)
WHERE episode_number IS NOT NULL;


-- Owner/public episode browsing.
CREATE INDEX idx_podcast_episodes_podcast_created
ON podcast_episodes (
    podcast_id,
    created_at DESC
);


CREATE INDEX idx_podcast_episodes_public
ON podcast_episodes (
    podcast_id,
    status,
    published_at DESC,
    id DESC
);


-- Useful for ordered season views.
CREATE INDEX idx_podcast_episodes_season_number
ON podcast_episodes (
    podcast_id,
    season_number,
    episode_number
);


-- Future scheduled publishing worker.
CREATE INDEX idx_podcast_episodes_scheduled
ON podcast_episodes (
    scheduled_at
)
WHERE status = 'SCHEDULED';


-- Future/current live episode lookup.
CREATE INDEX idx_podcast_episodes_live
ON podcast_episodes (
    podcast_id,
    updated_at DESC
)
WHERE status = 'LIVE';


-- Search indexes.
CREATE INDEX idx_podcast_episodes_title_trgm
ON podcast_episodes
USING GIN (
    title gin_trgm_ops
);


CREATE INDEX idx_podcast_episodes_description_trgm
ON podcast_episodes
USING GIN (
    description gin_trgm_ops
);


COMMIT;
