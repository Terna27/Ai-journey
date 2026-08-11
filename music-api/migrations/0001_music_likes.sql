-- 0001_music_likes.sql
--
-- Records who has liked each music post so the "one like per caller" rule can
-- be enforced. Today liker_id holds the client IP (there is no per-user auth
-- yet); when real user identity lands, store the user id here instead and
-- rename the column. The (music_id, liker_id) primary key is what actually
-- guarantees uniqueness — even a race cannot produce a duplicate like.

CREATE TABLE IF NOT EXISTS music_likes (
    music_id   INT         NOT NULL REFERENCES music(id) ON DELETE CASCADE,
    liker_id   TEXT        NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (music_id, liker_id)
);
