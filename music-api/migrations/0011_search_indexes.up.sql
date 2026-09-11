BEGIN;

-- =========================================================
-- SEARCH OPTIMIZATION
--
-- The public search API performs substring searches using:
--
--     ILIKE '%query%'
--
-- PostgreSQL B-tree indexes cannot efficiently support this
-- pattern. pg_trgm provides trigram indexes that can accelerate
-- substring and similarity searches.
-- =========================================================


-- =========================================================
-- 1. ENABLE TRIGRAM SEARCH
-- =========================================================

CREATE EXTENSION IF NOT EXISTS pg_trgm;


-- =========================================================
-- 2. MUSIC SEARCH INDEXES
--
-- SearchTracks and CountTracks search:
--
--     song_title
--     artist_name
--     genre
-- =========================================================

CREATE INDEX IF NOT EXISTS
idx_music_song_title_trgm
ON music
USING GIN (
    song_title gin_trgm_ops
);


CREATE INDEX IF NOT EXISTS
idx_music_artist_name_trgm
ON music
USING GIN (
    artist_name gin_trgm_ops
);


CREATE INDEX IF NOT EXISTS
idx_music_genre_trgm
ON music
USING GIN (
    genre gin_trgm_ops
);


-- =========================================================
-- 3. ARTIST SEARCH INDEX
--
-- SearchArtists and CountArtists search:
--
--     artists.name ILIKE '%query%'
-- =========================================================

CREATE INDEX IF NOT EXISTS
idx_artists_name_trgm
ON artists
USING GIN (
    name gin_trgm_ops
);


-- =========================================================
-- 4. RELEASE SEARCH INDEX
--
-- SearchReleases searches published releases by title.
--
-- Artist-name searching is handled by idx_artists_name_trgm
-- on the joined artists table.
--
-- Only published releases participate in public search, so
-- use a partial index instead of indexing draft releases.
-- =========================================================

CREATE INDEX IF NOT EXISTS
idx_releases_published_title_trgm
ON releases
USING GIN (
    title gin_trgm_ops
)
WHERE is_published = TRUE;


COMMIT;