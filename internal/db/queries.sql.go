// Code generated by sqlc. DO NOT EDIT.
// versions:
//   sqlc v1.27.0
// source: queries.sql

package db

import (
	"context"

	"github.com/jackc/pgx/v5/pgtype"
)

const addBasicSubscription = `-- name: AddBasicSubscription :one
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
SELECT us.status, s.name, s.max_links_per_month, s.can_customise_path, s.can_create_duplicates
FROM user_sub us
JOIN subscriptions s
ON us.subscription_id = s.id
`

type AddBasicSubscriptionRow struct {
	Status              string
	Name                string
	MaxLinksPerMonth    int32
	CanCustomisePath    bool
	CanCreateDuplicates bool
}

func (q *Queries) AddBasicSubscription(ctx context.Context, userID int32) (AddBasicSubscriptionRow, error) {
	row := q.db.QueryRow(ctx, addBasicSubscription, userID)
	var i AddBasicSubscriptionRow
	err := row.Scan(
		&i.Status,
		&i.Name,
		&i.MaxLinksPerMonth,
		&i.CanCustomisePath,
		&i.CanCreateDuplicates,
	)
	return i, err
}

const createLink = `-- name: CreateLink :one
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
RETURNING id, user_id, short_code, destination_url, title, notes, created_at, updated_at
`

type CreateLinkParams struct {
	UserID         int32
	ShortCode      string
	DestinationUrl string
	Title          pgtype.Text
	Notes          pgtype.Text
}

func (q *Queries) CreateLink(ctx context.Context, arg CreateLinkParams) (Link, error) {
	row := q.db.QueryRow(ctx, createLink,
		arg.UserID,
		arg.ShortCode,
		arg.DestinationUrl,
		arg.Title,
		arg.Notes,
	)
	var i Link
	err := row.Scan(
		&i.ID,
		&i.UserID,
		&i.ShortCode,
		&i.DestinationUrl,
		&i.Title,
		&i.Notes,
		&i.CreatedAt,
		&i.UpdatedAt,
	)
	return i, err
}

const createUser = `-- name: CreateUser :one
INSERT INTO users (name, email, oauth_provider)
VALUES ($1, $2, $3)
RETURNING id, name, email
`

type CreateUserParams struct {
	Name          pgtype.Text
	Email         string
	OauthProvider pgtype.Text
}

type CreateUserRow struct {
	ID    int32
	Name  pgtype.Text
	Email string
}

func (q *Queries) CreateUser(ctx context.Context, arg CreateUserParams) (CreateUserRow, error) {
	row := q.db.QueryRow(ctx, createUser, arg.Name, arg.Email, arg.OauthProvider)
	var i CreateUserRow
	err := row.Scan(&i.ID, &i.Name, &i.Email)
	return i, err
}

const findDuplicatesForUrl = `-- name: FindDuplicatesForUrl :one
WITH limited_links AS (
  SELECT short_code
  FROM links
  WHERE user_id = $1
    AND destination_url = $2
  ORDER BY created_at DESC
  LIMIT $3
)
SELECT
  ARRAY_AGG(short_code)::text[] AS short_codes,
  GREATEST((SELECT COUNT(*)
              FROM links  As l
              WHERE l.user_id = $1
                AND l.destination_url = $2) - $3, 0)::int AS remaining_count
FROM limited_links
`

type FindDuplicatesForUrlParams struct {
	UserID         int32
	DestinationUrl string
	Limit          int32
}

type FindDuplicatesForUrlRow struct {
	ShortCodes     []string
	RemainingCount int32
}

func (q *Queries) FindDuplicatesForUrl(ctx context.Context, arg FindDuplicatesForUrlParams) (FindDuplicatesForUrlRow, error) {
	row := q.db.QueryRow(ctx, findDuplicatesForUrl, arg.UserID, arg.DestinationUrl, arg.Limit)
	var i FindDuplicatesForUrlRow
	err := row.Scan(&i.ShortCodes, &i.RemainingCount)
	return i, err
}

const getDestinationUrl = `-- name: GetDestinationUrl :one
SELECT destination_url
FROM links
WHERE short_code = $1
LIMIT 1
`

func (q *Queries) GetDestinationUrl(ctx context.Context, shortCode string) (string, error) {
	row := q.db.QueryRow(ctx, getDestinationUrl, shortCode)
	var destination_url string
	err := row.Scan(&destination_url)
	return destination_url, err
}

const getLinkByShortCode = `-- name: GetLinkByShortCode :one
SELECT user_id, short_code from links
WHERE short_code = $1
LIMIT 1
`

type GetLinkByShortCodeRow struct {
	UserID    int32
	ShortCode string
}

func (q *Queries) GetLinkByShortCode(ctx context.Context, shortCode string) (GetLinkByShortCodeRow, error) {
	row := q.db.QueryRow(ctx, getLinkByShortCode, shortCode)
	var i GetLinkByShortCodeRow
	err := row.Scan(&i.UserID, &i.ShortCode)
	return i, err
}

const getLinkForUser = `-- name: GetLinkForUser :one
SELECT short_code, destination_url, title, notes, created_at, updated_at
FROM links
WHERE user_id = $1
AND short_code = $2
LIMIT 1
`

type GetLinkForUserParams struct {
	UserID    int32
	ShortCode string
}

type GetLinkForUserRow struct {
	ShortCode      string
	DestinationUrl string
	Title          pgtype.Text
	Notes          pgtype.Text
	CreatedAt      pgtype.Timestamp
	UpdatedAt      pgtype.Timestamp
}

func (q *Queries) GetLinkForUser(ctx context.Context, arg GetLinkForUserParams) (GetLinkForUserRow, error) {
	row := q.db.QueryRow(ctx, getLinkForUser, arg.UserID, arg.ShortCode)
	var i GetLinkForUserRow
	err := row.Scan(
		&i.ShortCode,
		&i.DestinationUrl,
		&i.Title,
		&i.Notes,
		&i.CreatedAt,
		&i.UpdatedAt,
	)
	return i, err
}

const getPaginatedLinksForUser = `-- name: GetPaginatedLinksForUser :one
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
FROM paginated_links
`

type GetPaginatedLinksForUserParams struct {
	UserID int32
	Limit  int32
	Offset int32
}

type GetPaginatedLinksForUserRow struct {
	TotalCount int64
	Links      interface{}
}

func (q *Queries) GetPaginatedLinksForUser(ctx context.Context, arg GetPaginatedLinksForUserParams) (GetPaginatedLinksForUserRow, error) {
	row := q.db.QueryRow(ctx, getPaginatedLinksForUser, arg.UserID, arg.Limit, arg.Offset)
	var i GetPaginatedLinksForUserRow
	err := row.Scan(&i.TotalCount, &i.Links)
	return i, err
}

const getUser = `-- name: GetUser :one
SELECT id, name, email, oauth_provider, created_at, updated_at FROM users
WHERE id = $1 LIMIT 1
`

func (q *Queries) GetUser(ctx context.Context, id int32) (User, error) {
	row := q.db.QueryRow(ctx, getUser, id)
	var i User
	err := row.Scan(
		&i.ID,
		&i.Name,
		&i.Email,
		&i.OauthProvider,
		&i.CreatedAt,
		&i.UpdatedAt,
	)
	return i, err
}

const getUserByEmail = `-- name: GetUserByEmail :one
SELECT id, name, email
FROM users
WHERE email = $1
`

type GetUserByEmailRow struct {
	ID    int32
	Name  pgtype.Text
	Email string
}

func (q *Queries) GetUserByEmail(ctx context.Context, email string) (GetUserByEmailRow, error) {
	row := q.db.QueryRow(ctx, getUserByEmail, email)
	var i GetUserByEmailRow
	err := row.Scan(&i.ID, &i.Name, &i.Email)
	return i, err
}

const getUserCurrentUsage = `-- name: GetUserCurrentUsage :one
SELECT links_created
FROM user_monthly_usage
WHERE user_id = $1
  AND cycle_start_date <= CURRENT_DATE
  AND cycle_end_date > CURRENT_DATE
`

func (q *Queries) GetUserCurrentUsage(ctx context.Context, userID int32) (int32, error) {
	row := q.db.QueryRow(ctx, getUserCurrentUsage, userID)
	var links_created int32
	err := row.Scan(&links_created)
	return links_created, err
}

const getUserSubscription = `-- name: GetUserSubscription :one
SELECT us.status, s.name, s.max_links_per_month, s.can_customise_path, s.can_create_duplicates
FROM user_subscriptions us
JOIN subscriptions s
ON us.subscription_id=s.id
WHERE us.user_id=$1
`

type GetUserSubscriptionRow struct {
	Status              string
	Name                string
	MaxLinksPerMonth    int32
	CanCustomisePath    bool
	CanCreateDuplicates bool
}

func (q *Queries) GetUserSubscription(ctx context.Context, userID int32) (GetUserSubscriptionRow, error) {
	row := q.db.QueryRow(ctx, getUserSubscription, userID)
	var i GetUserSubscriptionRow
	err := row.Scan(
		&i.Status,
		&i.Name,
		&i.MaxLinksPerMonth,
		&i.CanCustomisePath,
		&i.CanCreateDuplicates,
	)
	return i, err
}

const recordVisit = `-- name: RecordVisit :exec
INSERT INTO analytics (short_code, user_agent_data, referrer_url)
VALUES ($1, $2, $3)
`

type RecordVisitParams struct {
	ShortCode     string
	UserAgentData []byte
	ReferrerUrl   pgtype.Text
}

func (q *Queries) RecordVisit(ctx context.Context, arg RecordVisitParams) error {
	_, err := q.db.Exec(ctx, recordVisit, arg.ShortCode, arg.UserAgentData, arg.ReferrerUrl)
	return err
}
