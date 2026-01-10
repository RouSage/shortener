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

-- name: DeleteURL :execrows
DELETE FROM urls
WHERE
  id = sqlc.arg ('id');

-- name: DeleteAllUserURLs :many
DELETE FROM urls
WHERE
  user_id = sqlc.arg ('user_id')::text
RETURNING
  id;

-- name: BlockUser :one
INSERT INTO
  user_blocks (user_id, user_email, blocked_by, reason)
VALUES
  ($1, $2, $3, $4)
ON CONFLICT (user_id) DO UPDATE
SET
  user_email = EXCLUDED.user_email,
  blocked_by = EXCLUDED.blocked_by,
  reason = EXCLUDED.reason,
  unblocked_by = NULL,
  unblocked_at = NULL
RETURNING
  *;

-- name: UnblockUser :one
UPDATE user_blocks
SET
  unblocked_by = sqlc.arg ('unblocked_by')::text,
  unblocked_at = NOW()
WHERE
  user_id = sqlc.arg ('user_id')
RETURNING
  *;

-- name: GetUserBlocks :many
SELECT
  id,
  user_id,
  user_email,
  blocked_by,
  blocked_at,
  unblocked_by,
  unblocked_at,
  COUNT(*) OVER () as total_count
FROM
  user_blocks
ORDER BY
  blocked_at DESC
LIMIT
  sqlc.arg ('limit')
OFFSET
  sqlc.arg ('offset');
