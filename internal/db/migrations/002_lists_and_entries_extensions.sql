-- 002_lists_and_entries_extensions.sql
--
-- Aligns the schema with the mobile app's data model:
--
--   1. Lists gain editable metadata used by the create/edit flows:
--      `category` (for Discover search), `locked` (freeze submissions),
--      and three optional chat links (Telegram / WhatsApp / Discord).
--
--   2. Entries gain moderation state (`status`) and a live `previous_rank`
--      so the leaderboard can show rank-change deltas without a separate
--      history table. New rows default to `approved` so existing data and
--      lists that don't require moderation keep working untouched.
--
--   3. Two supporting indexes:
--        - entries(list_id, status)     → cheap pending-queue queries
--        - lists(is_public, category)   → cheap discover/search filters

-- ── lists ────────────────────────────────────────────────────────────
ALTER TABLE lists
  ADD COLUMN IF NOT EXISTS category       TEXT,
  ADD COLUMN IF NOT EXISTS locked         BOOLEAN NOT NULL DEFAULT FALSE,
  ADD COLUMN IF NOT EXISTS telegram_link  TEXT,
  ADD COLUMN IF NOT EXISTS whatsapp_link  TEXT,
  ADD COLUMN IF NOT EXISTS discord_link   TEXT;

CREATE INDEX IF NOT EXISTS idx_lists_public_category
  ON lists (is_public, category)
  WHERE is_public = TRUE;

-- ── entries ──────────────────────────────────────────────────────────
ALTER TABLE entries
  ADD COLUMN IF NOT EXISTS status        TEXT NOT NULL DEFAULT 'approved'
    CHECK (status IN ('pending', 'approved', 'rejected')),
  ADD COLUMN IF NOT EXISTS previous_rank INT;

CREATE INDEX IF NOT EXISTS idx_entries_list_status
  ON entries (list_id, status);
