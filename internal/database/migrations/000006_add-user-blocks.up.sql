BEGIN;

CREATE TABLE IF NOT EXISTS user_blocks (
  id SERIAL PRIMARY KEY,
  user_id TEXT NOT NULL,
  user_email TEXT,
  blocked_by TEXT NOT NULL,
  blocked_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  unblocked_by TEXT,
  unblocked_at TIMESTAMPTZ,
  reason TEXT
);

CREATE UNIQUE INDEX IF NOT EXISTS user_blocks_user_id_idx ON user_blocks (user_id);

CREATE INDEX IF NOT EXISTS user_blocks_blocked_at_desc_idx ON user_blocks (blocked_at DESC);

COMMIT;
