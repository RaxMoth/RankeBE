-- name: CreateRefreshToken :one
INSERT INTO refresh_tokens (user_id, token, expires_at)
VALUES ($1, $2, $3)
RETURNING *;

-- name: GetValidRefreshToken :one
SELECT * FROM refresh_tokens
WHERE token = $1 AND revoked = FALSE AND expires_at > NOW();

-- GetRefreshTokenAny finds a refresh token by value regardless of
-- revoked / expired state. Used by the reuse-detection path: if a
-- caller presents an already-revoked token, that's a strong signal of
-- token theft, and we revoke every refresh token for that user.
-- name: GetRefreshTokenAny :one
SELECT * FROM refresh_tokens WHERE token = $1;

-- name: RevokeRefreshToken :exec
UPDATE refresh_tokens SET revoked = TRUE WHERE id = $1;

-- name: RevokeAllUserRefreshTokens :exec
UPDATE refresh_tokens SET revoked = TRUE WHERE user_id = $1;
