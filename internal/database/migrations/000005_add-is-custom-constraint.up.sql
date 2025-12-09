BEGIN;

ALTER TABLE urls
ADD CONSTRAINT custom_urls_require_user CHECK (
  is_custom = false
  OR user_id IS NOT NULL
);

COMMIT;
