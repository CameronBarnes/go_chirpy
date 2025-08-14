-- name: AddChirp :one
INSERT INTO chirps (id, body, user_id)
VALUES (gen_random_uuid(), $1, $2)
RETURNING *;

-- name: AllChirps :many
SELECT * FROM chirps
ORDER BY created_at ASC;

-- name: AllChirpsFromUser :many
SELECT * FROM chirps
WHERE user_id = $1
ORDER BY created_at ASC;

-- name: GetChirp :one
SELECT * FROM chirps
WHERE id = $1;

-- name: DeleteChirp :one
DELETE FROM chirps
WHERE id = $1 AND user_id = $2
RETURNING id;
