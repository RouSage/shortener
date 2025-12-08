BEGIN;

DROP INDEX IF EXISTS idx_urls_user_id;

CREATE INDEX IF NOT EXISTS idx_user_id_created_at ON urls (user_id, created_at DESC);

COMMIT;
