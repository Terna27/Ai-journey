BEGIN;

-- =========================================================
-- ARTIST PROFILE MEDIA
--
-- Canonical presentation media for an artist profile.
--
-- Binary assets are stored in Cloudinary.
-- PostgreSQL stores:
--   1. the delivery URL used by clients
--   2. the Cloudinary public ID used for asset management
--
-- All fields are nullable so existing artist profiles remain
-- valid after this migration.
-- =========================================================

ALTER TABLE artists
ADD COLUMN IF NOT EXISTS bio TEXT,

ADD COLUMN IF NOT EXISTS profile_image_url TEXT,
ADD COLUMN IF NOT EXISTS profile_image_public_id TEXT,

ADD COLUMN IF NOT EXISTS hero_video_url TEXT,
ADD COLUMN IF NOT EXISTS hero_video_public_id TEXT,

ADD COLUMN IF NOT EXISTS hero_video_poster_url TEXT,
ADD COLUMN IF NOT EXISTS hero_video_poster_public_id TEXT;


-- =========================================================
-- BIO LENGTH
--
-- Keep artist biographies reasonably sized at the database
-- boundary as a final layer of protection.
-- Application-level validation will also enforce this.
-- =========================================================

ALTER TABLE artists
ADD CONSTRAINT artists_bio_length_check
CHECK (
    bio IS NULL
    OR char_length(bio) <= 2000
);


-- =========================================================
-- CLOUDINARY ASSET PAIR INTEGRITY
--
-- A delivery URL and its Cloudinary management ID must either
-- both exist or both be NULL.
-- =========================================================

ALTER TABLE artists
ADD CONSTRAINT artists_profile_image_pair_check
CHECK (
    (profile_image_url IS NULL AND profile_image_public_id IS NULL)
    OR
    (profile_image_url IS NOT NULL AND profile_image_public_id IS NOT NULL)
);


ALTER TABLE artists
ADD CONSTRAINT artists_hero_video_pair_check
CHECK (
    (hero_video_url IS NULL AND hero_video_public_id IS NULL)
    OR
    (hero_video_url IS NOT NULL AND hero_video_public_id IS NOT NULL)
);


ALTER TABLE artists
ADD CONSTRAINT artists_hero_video_poster_pair_check
CHECK (
    (
        hero_video_poster_url IS NULL
        AND hero_video_poster_public_id IS NULL
    )
    OR
    (
        hero_video_poster_url IS NOT NULL
        AND hero_video_poster_public_id IS NOT NULL
    )
);

COMMIT;
