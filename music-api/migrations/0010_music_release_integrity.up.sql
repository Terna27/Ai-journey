BEGIN;

-- =========================================================
-- MUSIC / RELEASE INTEGRITY
--
-- release_id and track_number represent one logical
-- association.
--
-- Valid states:
--
--   Standalone track:
--     release_id   IS NULL
--     track_number IS NULL
--
--   Release track:
--     release_id   IS NOT NULL
--     track_number IS NOT NULL
--
-- Historical data created before the repository-level
-- detach fix may contain:
--
--     release_id   IS NULL
--     track_number IS NOT NULL
--
-- Repair those rows before enforcing the invariant.
-- =========================================================


-- =========================================================
-- 1. REPAIR HISTORICAL INCONSISTENT ROWS
-- =========================================================

UPDATE music
SET track_number = NULL
WHERE release_id IS NULL
  AND track_number IS NOT NULL;


-- =========================================================
-- 2. REQUIRE RELEASE MEMBERSHIP AND TRACK NUMBER TO EXIST
--    TOGETHER
-- =========================================================

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1
        FROM pg_constraint
        WHERE conname =
            'music_release_track_consistent'
          AND conrelid =
            'music'::regclass
    ) THEN
        ALTER TABLE music
        ADD CONSTRAINT music_release_track_consistent
        CHECK (
            (
                release_id IS NULL
                AND track_number IS NULL
            )
            OR
            (
                release_id IS NOT NULL
                AND track_number IS NOT NULL
            )
        );
    END IF;
END
$$;


-- =========================================================
-- 3. FIX DIRECT DATABASE RELEASE DELETION
--
-- The FK created in migration 0009 uses:
--
--     ON DELETE SET NULL
--
-- PostgreSQL would therefore clear release_id while leaving
-- track_number untouched.
--
-- Replace it with a trigger-driven detach followed by the
-- existing SET NULL FK behavior.
--
-- The trigger clears both values before the release row is
-- deleted, keeping direct SQL deletion consistent with the
-- application repository.
-- =========================================================

CREATE OR REPLACE FUNCTION
clear_music_release_assignment_before_release_delete()
RETURNS TRIGGER
LANGUAGE plpgsql
AS $$
BEGIN
    UPDATE music
    SET
        release_id = NULL,
        track_number = NULL
    WHERE release_id = OLD.id;

    RETURN OLD;
END;
$$;


DROP TRIGGER IF EXISTS
trg_clear_music_release_assignment_before_delete
ON releases;


CREATE TRIGGER
trg_clear_music_release_assignment_before_delete
BEFORE DELETE
ON releases
FOR EACH ROW
EXECUTE FUNCTION
clear_music_release_assignment_before_release_delete();


COMMIT;