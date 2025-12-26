-- name: CreateUrl :one
INSERT INTO
  urls (id, long_url, is_custom, user_id)
VALUES
  ($1, $2, $3, $4)
RETURNING
  *;

-- name: GetUserUrls :many
SELECT
  id,
  long_url,
  created_at,
  is_custom,
  COUNT(*) OVER () as total_count
FROM
  urls
WHERE
  user_id = $1
ORDER BY
  created_at DESC
LIMIT
  $2
OFFSET
  $3;

-- name: GetLongUrl :one
SELECT
  long_url
FROM
  urls
WHERE
  id = $1
LIMIT
  1;

-- name: DeleteUserURL :execrows
DELETE FROM urls
WHERE
  id = $1
  AND user_id = $2;
