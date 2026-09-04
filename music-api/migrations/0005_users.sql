BEGIN;

-- =========================================================
-- USERS
--
-- Every person using the platform will have one user account.
-- Artist-specific information remains in the artists table.
--
-- During this migration we copy existing artist credentials
-- into users without removing anything from artists yet.
-- =========================================================

CREATE TABLE IF NOT EXISTS users (
    id BIGSERIAL PRIMARY KEY,

    name TEXT NOT NULL,

    email TEXT NOT NULL UNIQUE,

    password_hash TEXT NOT NULL,

    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),

    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);


-- =========================================================
-- MIGRATE EXISTING ARTIST ACCOUNTS
--
-- Preserve the current IDs where possible.
--
-- Existing artist:
--     artists.id = 4
--
-- becomes:
--     users.id   = 4
--     artists.id = 4
--
-- This gives us a simple initial mapping while preserving
-- all existing music.artist_id ownership.
-- =========================================================

INSERT INTO users (
    id,
    name,
    email,
    password_hash,
    created_at,
    updated_at
)
SELECT
    id,
    name,
    email,
    password_hash,
    created_at,
    updated_at
FROM artists
ON CONFLICT DO NOTHING;


-- =========================================================
-- CONNECT ARTISTS TO USERS
-- =========================================================

ALTER TABLE artists
ADD COLUMN IF NOT EXISTS user_id BIGINT;


-- Existing artists are connected to the corresponding user.
--
-- Match by email rather than assuming IDs always match.
-- This makes the migration safer if it is ever applied to
-- a database where a user record already exists.
UPDATE artists AS a
SET user_id = u.id
FROM users AS u
WHERE a.user_id IS NULL
  AND LOWER(a.email) = LOWER(u.email);


-- =========================================================
-- SAFETY CHECK
--
-- Do not complete this migration if an existing artist failed
-- to receive a user account.
-- =========================================================

DO $$
BEGIN
    IF EXISTS (
        SELECT 1
        FROM artists
        WHERE user_id IS NULL
    ) THEN
        RAISE EXCEPTION
            'users migration failed: one or more artists do not have a user_id';
    END IF;
END
$$;


-- Every artist must now belong to exactly one user.
ALTER TABLE artists
ALTER COLUMN user_id SET NOT NULL;


-- One user cannot accidentally have multiple artist records
-- at this stage of the platform.
CREATE UNIQUE INDEX IF NOT EXISTS idx_artists_user_id_unique
ON artists(user_id);


-- =========================================================
-- FOREIGN KEY
-- =========================================================

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1
        FROM pg_constraint
        WHERE conname = 'artists_user_id_fkey'
          AND conrelid = 'artists'::regclass
    ) THEN
        ALTER TABLE artists
        ADD CONSTRAINT artists_user_id_fkey
        FOREIGN KEY (user_id)
        REFERENCES users(id)
        ON DELETE CASCADE;
    END IF;
END
$$;


-- =========================================================
-- USERS LOOKUP INDEX
-- =========================================================

CREATE INDEX IF NOT EXISTS idx_users_email
ON users(email);


-- =========================================================
-- RESET USERS ID SEQUENCE
--
-- Because existing artist IDs were inserted explicitly,
-- move the users sequence past the highest current user ID.
-- Otherwise the next registration could try to reuse ID 1.
-- =========================================================

SELECT setval(
    pg_get_serial_sequence('users', 'id'),
    COALESCE(
        (SELECT MAX(id) FROM users),
        1
    ),
    true
);


COMMIT;
