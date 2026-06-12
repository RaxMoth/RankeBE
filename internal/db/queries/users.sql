-- name: GetUserByID :one
SELECT * FROM users WHERE id = $1;

-- name: GetUserByEmail :one
SELECT * FROM users WHERE email = $1;

-- name: GetUserByAppleSub :one
SELECT * FROM users WHERE apple_sub = $1;

-- name: CreateUser :one
INSERT INTO users (email, display_name, auth_provider, password_hash)
VALUES ($1, $2, 'email', $3)
RETURNING *;

-- name: CreateAppleUser :one
INSERT INTO users (email, display_name, auth_provider, apple_sub)
VALUES ($1, $2, 'apple', $3)
RETURNING *;

-- name: UpdateDisplayName :one
UPDATE users SET display_name = $2 WHERE id = $1
RETURNING *;

-- DeleteUser cascades through every FK in 001_init.sql:
--   refresh_tokens (user_id), entries (user_id), list_members (user_id),
--   lists (owner_id). A user pressing "Delete account" therefore also
--   wipes any board they own — known limitation, see README.
-- name: DeleteUser :exec
DELETE FROM users WHERE id = $1;

-- name: GetUserProfile :one
SELECT
  u.id,
  u.display_name,
  u.avatar_url,
  u.created_at,
  (SELECT COUNT(*) FROM list_members lm
     JOIN lists l ON l.id = lm.list_id
     WHERE lm.user_id = u.id AND l.is_public = TRUE) AS public_board_count
FROM users u
WHERE u.id = $1;

-- name: GetPublicBoardsForUser :many
SELECT
  l.*,
  lm.role AS user_role,
  e.id AS own_entry_id,
  e.value_number AS own_entry_value_number,
  e.value_duration_ms AS own_entry_value_duration_ms,
  e.value_text AS own_entry_value_text,
  e.manual_rank AS own_entry_manual_rank,
  (SELECT COUNT(*) FROM list_members WHERE list_id = l.id) AS member_count
FROM lists l
JOIN list_members lm ON lm.list_id = l.id AND lm.user_id = $1
LEFT JOIN entries e ON e.list_id = l.id AND e.user_id = $1
WHERE l.is_public = TRUE
ORDER BY l.updated_at DESC;
