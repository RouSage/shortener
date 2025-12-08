BEGIN;

DROP INDEX IF EXISTS idx_user_id_created_at;

CREATE INDEX idx_urls_user_id ON urls (user_id);

COMMIT;
