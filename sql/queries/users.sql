
-- name: CreateUser :one
INSERT INTO users(id, created_at, updated_at, email, hashed_password)
VALUES(gen_random_uuid(), NOW(), NOW(), $1, $2)
RETURNING *;


-- name: ClearAllUsers :exec
DELETE FROM users;

-- name: Lookup_user_by_email :one
SELECT *
FROM users
WHERE email = $1;