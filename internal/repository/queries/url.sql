-- name: CreateUrl :one
INSERT INTO urls (id, long_url)
VALUES ($1, $2)
RETURNING *;

-- name: GetLongUrl :one
SELECT long_url
  FROM urls
 WHERE id = $1
 LIMIT 1;
