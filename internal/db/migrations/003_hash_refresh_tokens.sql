-- 003_hash_refresh_tokens.sql
--
-- Refresh tokens are now stored SHA-256-hashed at rest (see
-- internal/service/auth.go). The raw token is only ever returned to the
-- client; the database keeps a hash, so a DB leak no longer yields
-- directly-replayable sessions.
--
-- This cutover invalidates every pre-existing session: rows written before
-- this migration hold raw-hex tokens that can never match a hashed lookup.
-- Rather than let them linger until they expire, we drop them outright so
-- the table only ever contains hashes. Affected users simply log in again.
--
-- The column type is unchanged: SHA-256 hex is 64 chars, same width as the
-- previous raw-hex token, so `token TEXT UNIQUE` still fits.

TRUNCATE TABLE refresh_tokens;
