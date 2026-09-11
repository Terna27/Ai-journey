BEGIN;

DROP TRIGGER IF EXISTS
trg_clear_music_release_assignment_before_delete
ON releases;

DROP FUNCTION IF EXISTS
clear_music_release_assignment_before_release_delete();

ALTER TABLE music
DROP CONSTRAINT IF EXISTS
music_release_track_consistent;

COMMIT;