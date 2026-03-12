-- name: CreateList :one
INSERT INTO lists (owner_id, title, description, value_type, rank_order, is_public)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING *;

-- name: GetListByID :one
SELECT * FROM lists WHERE id = $1;

-- name: GetUserLists :many
SELECT
  l.*,
  lm.role AS user_role,
  e.id AS own_entry_id,
  e.value_number AS own_entry_value_number,
  e.value_duration_ms AS own_entry_value_duration_ms,
  e.value_text AS own_entry_value_text,
  e.manual_rank AS own_entry_manual_rank
FROM lists l
JOIN list_members lm ON lm.list_id = l.id AND lm.user_id = $1
LEFT JOIN entries e ON e.list_id = l.id AND e.user_id = $1
ORDER BY l.updated_at DESC;

-- name: UpdateList :one
UPDATE lists
SET title = $2, description = $3, is_public = $4, updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: DeleteList :exec
DELETE FROM lists WHERE id = $1;

-- name: GetListByInviteToken :one
SELECT * FROM lists WHERE invite_token = $1;

-- name: GetInvitePreview :one
SELECT
  l.id,
  l.title,
  l.value_type,
  l.is_public,
  (SELECT COUNT(*) FROM list_members WHERE list_id = l.id) AS member_count
FROM lists l
WHERE l.invite_token = $1;
