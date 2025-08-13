-- name: CreateRefreshToken :one
INSERT INTO refresh_tokens(token, user_id, expires_at)
VALUES ($1, $2, $3)
RETURNING *;

-- name: GetRefreshToken :one
SELECT * FROM refresh_tokens
WHERE token = $1;

-- name: GetUserTokens :many
SELECT * FROM refresh_tokens
WHERE user_id = $1;

-- name: ExpireToken :exec
UPDATE refresh_tokens
SET updated_at = NOW(), revoked_at = NOW()
WHERE token = $1;

-- name: ExpireAllForUser :exec
UPDATE refresh_tokens
SET updated_at = NOW(), revoked_at = NOW()
WHERE user_id = $1;

-- name: GetUserFromRefreshToken :one
SELECT users.id, users.created_at, users.updated_at, users.email, refresh_tokens.token FROM users
INNER JOIN refresh_tokens ON users.id = refresh_tokens.user_id
WHERE refresh_tokens.token = $1 AND refresh_tokens.revoked_at IS NULL AND refresh_tokens.expires_at > NOW();
