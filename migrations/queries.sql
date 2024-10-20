-- name: GetUser :one
SELECT * FROM users
WHERE id = $1 LIMIT 1;

-- name: GetUserByEmail :one
SELECT id, name, email
FROM users
WHERE email = $1;

-- name: CreateUser :one
INSERT INTO users (name, email, oauth_provider)
VALUES ($1, $2, $3)
RETURNING id, name, email;

-- name: GetUserSubscription :one
SELECT us.status, s.name, s.max_links_per_month, s.can_customise_path, s.can_create_duplicates, s.can_view_analytics
FROM user_subscriptions us
JOIN subscriptions s
ON us.subscription_id=s.id
WHERE us.user_id=$1;

-- name: AddBasicSubscription :one
WITH user_sub AS (
  INSERT INTO user_subscriptions (user_id, subscription_id, end_date)
  VALUES ($1, (SELECT id FROM subscriptions WHERE name = 'basic'), 'infinity')
  RETURNING status, subscription_id
),
user_usage AS (
  INSERT INTO user_monthly_usage (user_id, cycle_start_date, cycle_end_date)
  VALUES (
    $1, CURRENT_DATE, CURRENT_DATE + INTERVAL '1 month'
  )
)
SELECT us.status, s.name, s.max_links_per_month, s.can_customise_path, s.can_create_duplicates, s.can_view_analytics
FROM user_sub us
JOIN subscriptions s
ON us.subscription_id = s.id;

-- name: GetUserCurrentUsage :one
SELECT links_created
FROM user_monthly_usage
WHERE user_id = $1
  AND cycle_start_date <= CURRENT_DATE
  AND cycle_end_date > CURRENT_DATE;

-- name: CreateLink :one
WITH updated_usage AS (
  UPDATE user_monthly_usage
  SET links_created = links_created + 1,
      updated_at = CURRENT_TIMESTAMP
  WHERE user_id = $1
    AND cycle_start_date <= CURRENT_DATE
    AND cycle_end_date > CURRENT_DATE
)
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

-- name: GetLinkByShortCode :one
SELECT user_id, short_code from links
WHERE short_code = $1
LIMIT 1;

-- name: RecordVisit :exec
INSERT INTO analytics (short_code, user_agent_data, geo_data, referrer_url)
VALUES ($1, $2, $3, $4);

-- name: GetVisitDataForShortcode :many
SELECT user_agent_data, geo_data, referrer_url, recorded_at
FROM analytics
WHERE short_code = $1
ORDER BY created_at DESC;
