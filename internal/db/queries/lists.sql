-- name: CreateList :one
INSERT INTO lists (
  owner_id, title, description, value_type, rank_order, is_public,
  category, telegram_link, whatsapp_link, discord_link
)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
RETURNING *;

-- name: GetListByID :one
SELECT * FROM lists WHERE id = $1;

-- GetUserLists returns one row per list the caller belongs to, including
-- their current rank if they have an entry. The rank is computed
-- in-line as `count_of_better + 1` to avoid an N+1 round-trip (one rank
-- query per board) from the handler — a single SELECT now drives the
-- entire home feed.
--
-- The CASE chain dispatches on the list's value_type / rank_order:
--   - number   asc : lower value wins
--   - number   desc: higher value wins
--   - duration asc : faster wins
--   - duration desc: slower wins
--   - text         : manual_rank wins (lower is better)
-- We treat the user as unranked (NULL own_rank) when their entry doesn't
-- exist OR is not approved.
-- name: GetUserLists :many
SELECT
  l.*,
  lm.role AS user_role,
  e.id AS own_entry_id,
  e.value_number AS own_entry_value_number,
  e.value_duration_ms AS own_entry_value_duration_ms,
  e.value_text AS own_entry_value_text,
  e.manual_rank AS own_entry_manual_rank,
  (SELECT COUNT(*) FROM list_members WHERE list_id = l.id) AS member_count,
  -- own_rank uses 0 as a sentinel for "no rank" (no entry, or pending/
  -- rejected entry). sqlc generates a non-nullable int from the
  -- expression below; the handler converts 0 -> null before serializing.
  --
  -- Using a sentinel rather than a NULL because sqlc 1.30 can't be made
  -- to emit pgtype.Int4 for a derived column (overrides don't bind to
  -- aliases), and we'd rather not hand-edit generated code.
  (CASE
    WHEN e.id IS NULL OR e.status <> 'approved' THEN 0
    ELSE 1 + (
      SELECT COUNT(*) FROM entries e2
      WHERE e2.list_id = l.id
        AND e2.status = 'approved'
        AND e2.user_id <> $1
        AND (
          (l.value_type = 'number'   AND l.rank_order = 'asc'  AND e2.value_number      < e.value_number)      OR
          (l.value_type = 'number'   AND l.rank_order = 'desc' AND e2.value_number      > e.value_number)      OR
          (l.value_type = 'duration' AND l.rank_order = 'asc'  AND e2.value_duration_ms < e.value_duration_ms) OR
          (l.value_type = 'duration' AND l.rank_order = 'desc' AND e2.value_duration_ms > e.value_duration_ms) OR
          (l.value_type = 'text'                               AND e2.manual_rank      < e.manual_rank)
        )
    )
  END)::INT AS own_rank
FROM lists l
JOIN list_members lm ON lm.list_id = l.id AND lm.user_id = $1
LEFT JOIN entries e ON e.list_id = l.id AND e.user_id = $1
ORDER BY l.updated_at DESC;

-- name: UpdateList :one
UPDATE lists
SET
  title         = COALESCE(sqlc.narg('title'),         title),
  description   = COALESCE(sqlc.narg('description'),   description),
  is_public     = COALESCE(sqlc.narg('is_public'),     is_public),
  locked        = COALESCE(sqlc.narg('locked'),        locked),
  category      = COALESCE(sqlc.narg('category'),      category),
  telegram_link = COALESCE(sqlc.narg('telegram_link'), telegram_link),
  whatsapp_link = COALESCE(sqlc.narg('whatsapp_link'), whatsapp_link),
  discord_link  = COALESCE(sqlc.narg('discord_link'),  discord_link),
  updated_at    = NOW()
WHERE id = sqlc.arg('id')
RETURNING *;

-- name: DeleteList :exec
DELETE FROM lists WHERE id = $1;

-- name: GetListByInviteToken :one
SELECT * FROM lists WHERE invite_token = $1;

-- name: GetInvitePreview :one
SELECT
  l.*,
  (SELECT COUNT(*) FROM list_members WHERE list_id = l.id) AS member_count
FROM lists l
WHERE l.invite_token = $1;

-- name: RegenerateInviteToken :one
UPDATE lists
SET invite_token = gen_random_uuid(), updated_at = NOW()
WHERE id = $1
RETURNING invite_token;

-- name: SearchPublicLists :many
SELECT
  l.*,
  (SELECT COUNT(*) FROM list_members WHERE list_id = l.id) AS member_count
FROM lists l
WHERE l.is_public = TRUE
  AND (sqlc.narg('q')::TEXT IS NULL
       OR l.title       ILIKE '%' || sqlc.narg('q')::TEXT || '%'
       OR l.description ILIKE '%' || sqlc.narg('q')::TEXT || '%')
  AND (sqlc.narg('category')::TEXT IS NULL OR l.category = sqlc.narg('category')::TEXT)
ORDER BY l.updated_at DESC
LIMIT 100;
