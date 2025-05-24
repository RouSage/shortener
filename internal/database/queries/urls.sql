-- name: GetLongUrl :one
SELECT long_url
  FROM urls
 WHERE id = $1
 LIMIT 1;
