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
