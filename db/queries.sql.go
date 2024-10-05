// Code generated by sqlc. DO NOT EDIT.
// versions:
//   sqlc v1.27.0
// source: queries.sql

package db

import (
	"context"

	"github.com/jackc/pgx/v5/pgtype"
)

const createLink = `-- name: CreateLink :one
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

const createOrUpdateUser = `-- name: CreateOrUpdateUser :one
INSERT INTO users (name, email, role, oauth_provider)
VALUES ($1, $2, $4, $3)
ON CONFLICT(email) DO UPDATE SET
name = excluded.name,
oauth_provider = excluded.oauth_provider,
updated_at = CURRENT_TIMESTAMP
RETURNING id, name, email, role, oauth_provider, created_at, updated_at
`

type CreateOrUpdateUserParams struct {
	Name          pgtype.Text
	Email         string
	OauthProvider pgtype.Text
	Role          string
}

func (q *Queries) CreateOrUpdateUser(ctx context.Context, arg CreateOrUpdateUserParams) (User, error) {
	row := q.db.QueryRow(ctx, createOrUpdateUser,
		arg.Name,
		arg.Email,
		arg.OauthProvider,
		arg.Role,
	)
	var i User
	err := row.Scan(
		&i.ID,
		&i.Name,
		&i.Email,
		&i.Role,
		&i.OauthProvider,
		&i.CreatedAt,
		&i.UpdatedAt,
	)
	return i, err
}

const findDuplicatesForUrl = `-- name: FindDuplicatesForUrl :one
WITH limited_links AS (
  SELECT short_code
  FROM links
  WHERE user_id = $1
    AND destination_url = $2
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

const getUser = `-- name: GetUser :one
SELECT id, name, email, role, oauth_provider, created_at, updated_at FROM users
WHERE id = $1 LIMIT 1
`

func (q *Queries) GetUser(ctx context.Context, id int32) (User, error) {
	row := q.db.QueryRow(ctx, getUser, id)
	var i User
	err := row.Scan(
		&i.ID,
		&i.Name,
		&i.Email,
		&i.Role,
		&i.OauthProvider,
		&i.CreatedAt,
		&i.UpdatedAt,
	)
	return i, err
}
