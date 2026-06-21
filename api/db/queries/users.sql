-- name: GetUserByEmail :one
SELECT * FROM users WHERE email = $1;

-- name: UpsertUser :one
INSERT INTO users (email, name, avatar_url)
VALUES ($1, $2, $3)
ON CONFLICT (email) DO UPDATE
SET name = EXCLUDED.name, avatar_url = EXCLUDED.avatar_url
RETURNING *;
