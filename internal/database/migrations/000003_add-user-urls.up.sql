BEGIN;

ALTER TABLE urls
ADD COLUMN user_id TEXT;

CREATE INDEX idx_urls_user_id ON urls (user_id);

COMMIT;
