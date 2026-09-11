BEGIN;

ALTER TABLE artists
DROP CONSTRAINT IF EXISTS artists_hero_video_poster_pair_check;

ALTER TABLE artists
DROP CONSTRAINT IF EXISTS artists_hero_video_pair_check;

ALTER TABLE artists
DROP CONSTRAINT IF EXISTS artists_profile_image_pair_check;

ALTER TABLE artists
DROP CONSTRAINT IF EXISTS artists_bio_length_check;


ALTER TABLE artists
DROP COLUMN IF EXISTS hero_video_poster_public_id,
DROP COLUMN IF EXISTS hero_video_poster_url,
DROP COLUMN IF EXISTS hero_video_public_id,
DROP COLUMN IF EXISTS hero_video_url,
DROP COLUMN IF EXISTS profile_image_public_id,
DROP COLUMN IF EXISTS profile_image_url,
DROP COLUMN IF EXISTS bio;

COMMIT;
