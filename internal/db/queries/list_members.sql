-- name: CreateListMember :one
INSERT INTO list_members (list_id, user_id, role)
VALUES ($1, $2, $3)
RETURNING *;

-- name: GetListMember :one
SELECT * FROM list_members WHERE list_id = $1 AND user_id = $2;

-- name: ListMembers :many
SELECT lm.*, u.display_name, u.avatar_url
FROM list_members lm
JOIN users u ON u.id = lm.user_id
WHERE lm.list_id = $1
ORDER BY lm.joined_at ASC;

-- name: UpdateMemberRole :one
UPDATE list_members SET role = $3
WHERE list_id = $1 AND user_id = $2
RETURNING *;

-- name: DeleteListMember :exec
DELETE FROM list_members WHERE list_id = $1 AND user_id = $2;
