BEGIN;

ALTER TABLE urls
DROP CONSTRAINT custom_urls_require_user;

COMMIT;
