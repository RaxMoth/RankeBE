-- name: UpsertEntry :one
INSERT INTO entries (list_id, user_id, value_number, value_duration_ms, value_text, note, updated_at)
VALUES ($1, $2, $3, $4, $5, $6, NOW())
ON CONFLICT (list_id, user_id)
DO UPDATE SET
  value_number      = EXCLUDED.value_number,
  value_duration_ms = EXCLUDED.value_duration_ms,
  value_text        = EXCLUDED.value_text,
  note              = EXCLUDED.note,
  updated_at        = NOW()
RETURNING *;

-- name: DeleteEntry :exec
DELETE FROM entries WHERE id = $1;

-- name: DeleteEntryByListAndUser :exec
DELETE FROM entries WHERE list_id = $1 AND user_id = $2;

-- name: GetRankedEntriesByNumber :many
SELECT
  e.*,
  u.display_name,
  u.avatar_url,
  ROW_NUMBER() OVER (ORDER BY e.value_number ASC NULLS LAST) AS rank
FROM entries e
JOIN users u ON u.id = e.user_id
WHERE e.list_id = $1
ORDER BY rank ASC;

-- name: GetRankedEntriesByNumberDesc :many
SELECT
  e.*,
  u.display_name,
  u.avatar_url,
  ROW_NUMBER() OVER (ORDER BY e.value_number DESC NULLS LAST) AS rank
FROM entries e
JOIN users u ON u.id = e.user_id
WHERE e.list_id = $1
ORDER BY rank ASC;

-- name: GetRankedEntriesByDuration :many
SELECT
  e.*,
  u.display_name,
  u.avatar_url,
  ROW_NUMBER() OVER (ORDER BY e.value_duration_ms ASC NULLS LAST) AS rank
FROM entries e
JOIN users u ON u.id = e.user_id
WHERE e.list_id = $1
ORDER BY rank ASC;

-- name: GetRankedEntriesByDurationDesc :many
SELECT
  e.*,
  u.display_name,
  u.avatar_url,
  ROW_NUMBER() OVER (ORDER BY e.value_duration_ms DESC NULLS LAST) AS rank
FROM entries e
JOIN users u ON u.id = e.user_id
WHERE e.list_id = $1
ORDER BY rank ASC;

-- name: GetRankedEntriesByText :many
SELECT
  e.*,
  u.display_name,
  u.avatar_url,
  ROW_NUMBER() OVER (ORDER BY e.manual_rank ASC NULLS LAST) AS rank
FROM entries e
JOIN users u ON u.id = e.user_id
WHERE e.list_id = $1
ORDER BY rank ASC;

-- name: UpdateEntryManualRank :exec
UPDATE entries SET manual_rank = $2, updated_at = NOW() WHERE id = $1;
