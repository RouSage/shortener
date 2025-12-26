-- name: GetURLs :many
SELECT
  id,
  long_url,
  created_at,
  is_custom,
  user_id,
  COUNT(*) OVER () as total_count
FROM
  urls
WHERE
  (
    sqlc.narg ('is_custom')::boolean IS NULL
    OR is_custom = sqlc.narg ('is_custom')::boolean
  )
  AND (
    sqlc.narg ('user_id')::text IS NULL
    OR user_id = sqlc.narg ('user_id')::text
  )
ORDER BY
  created_at DESC
LIMIT
  sqlc.arg ('limit')
OFFSET
  sqlc.arg ('offset');
