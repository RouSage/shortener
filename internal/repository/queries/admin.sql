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
ORDER BY
  created_at DESC
LIMIT
  sqlc.arg ('limit')
OFFSET
  sqlc.arg ('offset');
