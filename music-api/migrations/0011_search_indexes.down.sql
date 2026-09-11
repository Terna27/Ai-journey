BEGIN;

DROP INDEX IF EXISTS
idx_releases_published_title_trgm;

DROP INDEX IF EXISTS
idx_artists_name_trgm;

DROP INDEX IF EXISTS
idx_music_genre_trgm;

DROP INDEX IF EXISTS
idx_music_artist_name_trgm;

DROP INDEX IF EXISTS
idx_music_song_title_trgm;

-- Do not DROP EXTENSION pg_trgm here.
--
-- Other application features or future migrations may depend
-- on the extension after this migration has been applied.

COMMIT;