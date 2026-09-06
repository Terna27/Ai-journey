-- 0007_user_music_likes.sql
--
-- Introduces authenticated, user-owned likes.
--
-- The old music_likes table used IP addresses in liker_id before
-- real user accounts existed. We intentionally leave that table
-- untouched so historical data is not falsely assigned to users.
--
-- All new authenticated likes will be stored here.

CREATE TABLE IF NOT EXISTS user_music_likes (
    user_id BIGINT NOT NULL
        REFERENCES users(id)
        ON DELETE CASCADE,

    music_id INT NOT NULL
        REFERENCES music(id)
        ON DELETE CASCADE,

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    PRIMARY KEY (user_id, music_id)
);

-- Efficiently fetch all users who liked a given track.
CREATE INDEX IF NOT EXISTS idx_user_music_likes_music_id
ON user_music_likes (music_id);

-- Efficiently load a user's liked-music library newest first.
CREATE INDEX IF NOT EXISTS idx_user_music_likes_user_created
ON user_music_likes (user_id, created_at DESC);