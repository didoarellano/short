include .env

# Why not an atlas.hcl config file? .hcl files can't read from .env files.
# A makefile or another tool would've been needed to set env vars before calling atlas
# so just consolidate all tasks here (builds, tests, etc.).
ENV ?= dev
migrate:
	@DB_URL=$$( [ "$(ENV)" = "prod" ] && echo "$(PROD_HOST_DB_URL)" || echo "$(DEV_HOST_DB_URL)" ); \
	atlas schema apply --url $$DB_URL --to "file://db/schema.sql" --dev-url "docker://postgres/16/dev"

.PHONY: migrate
