-- name: CreateUser :one
INSERT INTO users (id, email, hashed_password)
VALUES (
    gen_random_uuid(), $1, $2
)
RETURNING id, email, created_at, updated_at;

-- name: GetUser :one
SELECT * FROM users
WHERE email = $1;

-- name: DeleteAll :exec
DELETE FROM users;
