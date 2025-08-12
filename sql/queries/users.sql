-- name: CreateUser :one
INSERT INTO users (id, email)
VALUES (
    gen_random_uuid(), $1
)
RETURNING *;

-- name: DeleteAll :exec
DELETE FROM users;
