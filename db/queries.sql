-- name: GetUser :one
SELECT * FROM users
WHERE id = $1 LIMIT 1;

-- name: CreateOrUpdateUser :one
INSERT INTO users (name, email, role, oauth_provider)
VALUES ($1, $2, $4, $3)
ON CONFLICT(email) DO UPDATE SET
name = excluded.name,
oauth_provider = excluded.oauth_provider,
updated_at = CURRENT_TIMESTAMP
RETURNING *;
