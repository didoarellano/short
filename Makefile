include .env

check_atlas:
	@command -v atlas > /dev/null 2>&1 || { echo "Atlas is not installed. Install it from https://atlasgo.io"; exit 1; }

check_sqlc:
	@command -v sqlc > /dev/null 2>&1 || { echo "sqlc is not installed. Install it from https://sqlc.dev/"; exit 1; }

# Why not an atlas.hcl config file? .hcl files can't read from .env files.
# A makefile or another tool would've been needed to set env vars before calling atlas
# so just consolidate all tasks here (builds, tests, etc.).
ENV ?= dev
migrate: check_atlas
	@DB_URL=$$( [ "$(ENV)" = "prod" ] && echo "$(PROD_HOST_DB_URL)" || echo "$(DEV_HOST_DB_URL)" ); \
	atlas schema apply --url $$DB_URL --to "file://db/schema.sql" --dev-url "docker://postgres/16/dev"

db/db.go db/models.go db/queries.sql.go: db/schema.sql db/queries.sql | check_sqlc
	sqlc generate

db: db/db.go db/models.go db/queries.sql.go

test:
	go test ./...

.PHONY: check_atlas check_sqlc migrate db test
