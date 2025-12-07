-- name: CreateUrl :one
INSERT INTO
  urls (id, long_url, is_custom, user_id)
VALUES
  ($1, $2, $3, $4)
RETURNING
  *;

-- name: GetLongUrl :one
SELECT
  long_url
FROM
  urls
WHERE
  id = $1
LIMIT
  1;

-- name: DeleteUrl :execrows
DELETE FROM urls
WHERE
  id = $1;
