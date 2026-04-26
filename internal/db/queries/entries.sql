-- name: UpsertEntry :one
INSERT INTO entries (
  list_id, user_id,
  value_number, value_duration_ms, value_text,
  note, status, previous_rank, updated_at
)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, NOW())
ON CONFLICT (list_id, user_id)
DO UPDATE SET
  value_number      = EXCLUDED.value_number,
  value_duration_ms = EXCLUDED.value_duration_ms,
  value_text        = EXCLUDED.value_text,
  note              = EXCLUDED.note,
  status            = EXCLUDED.status,
  previous_rank     = EXCLUDED.previous_rank,
  updated_at        = NOW()
RETURNING *;

-- name: GetEntryByListAndUser :one
SELECT * FROM entries WHERE list_id = $1 AND user_id = $2;

-- name: DeleteEntry :exec
DELETE FROM entries WHERE id = $1;

-- name: DeleteEntryByListAndUser :exec
DELETE FROM entries WHERE list_id = $1 AND user_id = $2;

-- ── ranked feeds (only "approved" entries are ranked) ────────────────

-- name: GetRankedEntriesByNumber :many
SELECT
  e.*,
  u.display_name,
  u.avatar_url,
  ROW_NUMBER() OVER (ORDER BY e.value_number ASC NULLS LAST) AS rank
FROM entries e
JOIN users u ON u.id = e.user_id
WHERE e.list_id = $1 AND e.status = 'approved'
ORDER BY rank ASC;

-- name: GetRankedEntriesByNumberDesc :many
SELECT
  e.*,
  u.display_name,
  u.avatar_url,
  ROW_NUMBER() OVER (ORDER BY e.value_number DESC NULLS LAST) AS rank
FROM entries e
JOIN users u ON u.id = e.user_id
WHERE e.list_id = $1 AND e.status = 'approved'
ORDER BY rank ASC;

-- name: GetRankedEntriesByDuration :many
SELECT
  e.*,
  u.display_name,
  u.avatar_url,
  ROW_NUMBER() OVER (ORDER BY e.value_duration_ms ASC NULLS LAST) AS rank
FROM entries e
JOIN users u ON u.id = e.user_id
WHERE e.list_id = $1 AND e.status = 'approved'
ORDER BY rank ASC;

-- name: GetRankedEntriesByDurationDesc :many
SELECT
  e.*,
  u.display_name,
  u.avatar_url,
  ROW_NUMBER() OVER (ORDER BY e.value_duration_ms DESC NULLS LAST) AS rank
FROM entries e
JOIN users u ON u.id = e.user_id
WHERE e.list_id = $1 AND e.status = 'approved'
ORDER BY rank ASC;

-- name: GetRankedEntriesByText :many
SELECT
  e.*,
  u.display_name,
  u.avatar_url,
  ROW_NUMBER() OVER (ORDER BY e.manual_rank ASC NULLS LAST) AS rank
FROM entries e
JOIN users u ON u.id = e.user_id
WHERE e.list_id = $1 AND e.status = 'approved'
ORDER BY rank ASC;

-- ── current rank lookups (used to populate previous_rank on upsert) ──

-- name: GetCurrentRankByNumber :one
SELECT rank::INT AS rank FROM (
  SELECT user_id, ROW_NUMBER() OVER (ORDER BY value_number ASC NULLS LAST) AS rank
  FROM entries WHERE list_id = $1 AND status = 'approved'
) r WHERE user_id = $2;

-- name: GetCurrentRankByNumberDesc :one
SELECT rank::INT AS rank FROM (
  SELECT user_id, ROW_NUMBER() OVER (ORDER BY value_number DESC NULLS LAST) AS rank
  FROM entries WHERE list_id = $1 AND status = 'approved'
) r WHERE user_id = $2;

-- name: GetCurrentRankByDuration :one
SELECT rank::INT AS rank FROM (
  SELECT user_id, ROW_NUMBER() OVER (ORDER BY value_duration_ms ASC NULLS LAST) AS rank
  FROM entries WHERE list_id = $1 AND status = 'approved'
) r WHERE user_id = $2;

-- name: GetCurrentRankByDurationDesc :one
SELECT rank::INT AS rank FROM (
  SELECT user_id, ROW_NUMBER() OVER (ORDER BY value_duration_ms DESC NULLS LAST) AS rank
  FROM entries WHERE list_id = $1 AND status = 'approved'
) r WHERE user_id = $2;

-- name: GetCurrentRankByText :one
SELECT rank::INT AS rank FROM (
  SELECT user_id, ROW_NUMBER() OVER (ORDER BY manual_rank ASC NULLS LAST) AS rank
  FROM entries WHERE list_id = $1 AND status = 'approved'
) r WHERE user_id = $2;

-- ── moderation feed ──────────────────────────────────────────────────

-- name: GetPendingEntries :many
SELECT
  e.*,
  u.display_name,
  u.avatar_url
FROM entries e
JOIN users u ON u.id = e.user_id
WHERE e.list_id = $1 AND e.status = 'pending'
ORDER BY e.submitted_at ASC;

-- name: ApproveEntry :exec
UPDATE entries SET status = 'approved', updated_at = NOW()
WHERE id = $1 AND list_id = $2;

-- name: RejectEntry :exec
UPDATE entries SET status = 'rejected', updated_at = NOW()
WHERE id = $1 AND list_id = $2;

-- name: UpdateEntryManualRank :exec
UPDATE entries SET manual_rank = $2, updated_at = NOW() WHERE id = $1;
