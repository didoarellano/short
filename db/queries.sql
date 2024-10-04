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

-- name: CreateLink :one
INSERT INTO links (user_id, short_code, destination_url, title, notes)
VALUES ($1, $2, $3, $4, $5)
RETURNING *;

-- name: FindDuplicatesForURL :one
WITH limited_links AS (
  SELECT short_code
  FROM links
  WHERE user_id = $1
    AND destination_url = $2
  LIMIT sqlc.arg('limit')
)
SELECT
  ARRAY_AGG(short_code)::text[] AS short_codes,
  GREATEST((SELECT COUNT(*)
              FROM links  As l
              WHERE l.user_id = $1
                AND l.destination_url = $2) - sqlc.arg('limit'), 0)::int AS remaining_count
FROM limited_links;
