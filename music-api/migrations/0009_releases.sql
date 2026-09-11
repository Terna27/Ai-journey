BEGIN;

-- =========================================================
-- RELEASES
--
-- A release belongs to an artist and can represent:
--
--   SINGLE
--   EP
--   ALBUM
--
-- Existing tracks remain valid because music.release_id
-- will be nullable.
-- =========================================================

CREATE TABLE IF NOT EXISTS releases (
    id BIGSERIAL PRIMARY KEY,

    artist_id BIGINT NOT NULL
        REFERENCES artists(id)
        ON DELETE CASCADE,

    title TEXT NOT NULL,

    release_type TEXT NOT NULL,

    cover_image_url TEXT NOT NULL DEFAULT '',

    cover_image_public_id TEXT NOT NULL DEFAULT '',

    description TEXT NOT NULL DEFAULT '',

    release_date DATE,

    is_published BOOLEAN NOT NULL DEFAULT FALSE,

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT releases_type_valid
        CHECK (
            release_type IN (
                'SINGLE',
                'EP',
                'ALBUM'
            )
        ),

    CONSTRAINT releases_title_not_blank
        CHECK (
            length(trim(title)) > 0
        )
);


-- =========================================================
-- RELEASE LOOKUP INDEXES
-- =========================================================

CREATE INDEX IF NOT EXISTS idx_releases_artist_id
ON releases(artist_id);


CREATE INDEX IF NOT EXISTS idx_releases_artist_created
ON releases(
    artist_id,
    created_at DESC
);


CREATE INDEX IF NOT EXISTS idx_releases_public
ON releases(
    is_published,
    release_date DESC
);


-- =========================================================
-- CONNECT MUSIC TRACKS TO RELEASES
--
-- Nullable by design:
--
-- release_id IS NULL
--     = standalone / legacy track
--
-- release_id IS NOT NULL
--     = track belongs to a release
-- =========================================================

ALTER TABLE music
ADD COLUMN IF NOT EXISTS release_id BIGINT;


DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1
        FROM pg_constraint
        WHERE conname =
            'music_release_id_fkey'
          AND conrelid =
            'music'::regclass
    ) THEN
        ALTER TABLE music
        ADD CONSTRAINT music_release_id_fkey
        FOREIGN KEY (release_id)
        REFERENCES releases(id)
        ON DELETE SET NULL;
    END IF;
END
$$;


CREATE INDEX IF NOT EXISTS idx_music_release_id
ON music(release_id);


-- =========================================================
-- TRACK ORDER WITHIN A RELEASE
--
-- music.id stays the actual track identity.
--
-- track_number is optional so existing rows and transitional
-- uploads remain safe.
-- =========================================================

ALTER TABLE music
ADD COLUMN IF NOT EXISTS track_number INT;


DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1
        FROM pg_constraint
        WHERE conname =
            'music_track_number_positive'
          AND conrelid =
            'music'::regclass
    ) THEN
        ALTER TABLE music
        ADD CONSTRAINT music_track_number_positive
        CHECK (
            track_number IS NULL
            OR track_number > 0
        );
    END IF;
END
$$;


-- Prevent two tracks in the same release from occupying the
-- same track number.
--
-- PostgreSQL unique indexes permit multiple NULL values, so
-- standalone or transitional tracks with NULL track_number
-- remain valid.
CREATE UNIQUE INDEX IF NOT EXISTS idx_music_release_track_number_unique
ON music(
    release_id,
    track_number
)
WHERE
    release_id IS NOT NULL
    AND track_number IS NOT NULL;


COMMIT;