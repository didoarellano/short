-- name: GetUser :one
SELECT * FROM users
WHERE id = $1 LIMIT 1;

-- name: GetUserByEmail :one
SELECT id, name, email
FROM users
WHERE email = $1;

-- name: CreateUser :one
INSERT INTO users (name, email, role, oauth_provider)
VALUES ($1, $2, $4, $3)
RETURNING id, name, email;

-- name: CreateLink :one
INSERT INTO links (user_id, short_code, destination_url, title, notes)
VALUES ($1, $2, $3, $4, $5)
RETURNING *;

-- name: GetDestinationUrl :one
SELECT destination_url
FROM links
WHERE short_code = $1
LIMIT 1;

-- name: GetLinkForUser :one
SELECT short_code, destination_url, title, notes, created_at, updated_at
FROM links
WHERE user_id = $1
AND short_code = $2
LIMIT 1;

-- name: GetPaginatedLinksForUser :one
WITH paginated_links AS (
  SELECT short_code, destination_url, title, notes
  FROM links
  WHERE user_id = $1
  ORDER BY created_at DESC
  LIMIT $2
  OFFSET $3
)
SELECT
  (SELECT COUNT(*)
    FROM links l
    WHERE l.user_id = $1
  ) as total_count,
  ARRAY_AGG(
    jsonb_build_object(
      'short_code', short_code,
      'destination_url', destination_url,
      'title', title,
      'notes', notes
    )
  ) as links
FROM paginated_links;

-- name: FindDuplicatesForUrl :one
WITH limited_links AS (
  SELECT short_code
  FROM links
  WHERE user_id = $1
    AND destination_url = $2
  ORDER BY created_at DESC
  LIMIT sqlc.arg('limit')
)
SELECT
  ARRAY_AGG(short_code)::text[] AS short_codes,
  GREATEST((SELECT COUNT(*)
              FROM links  As l
              WHERE l.user_id = $1
                AND l.destination_url = $2) - sqlc.arg('limit'), 0)::int AS remaining_count
FROM limited_links;
