# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Overview

`pkm-atlas-api` is a Go 1.25 HTTP API. It uses the **standard library only** for HTTP (`net/http` + `http.ServeMux`) — there is no web framework. PostgreSQL is accessed via `pgx/v5` (`pgxpool`). Config is loaded from a `.env` file via `godotenv`.

## Commands

- **Run with hot reload (host):** `air -c .air.toml` — rebuilds on changes to `.go`/`.html`/`.env` files.
- **Run full stack (API + Postgres) in Docker:** `docker-compose up`
- **Build:** `go build -o ./tmp/api ./cmd/api`
- **Test:** `go test ./...` (no tests exist yet)
- **Lint:** `golangci-lint run` (config in `.golangci.yml`; install via `brew install golangci-lint`)

## Environment

Copy `.env.example` to `.env` before running. `DATABASE_URL` is **required** — the app errors on startup without it. `PORT` defaults to `8081`.

Gotchas around ports/URLs (intentional, don't "fix"):
- The container **always listens on 8080**; `docker-compose.yml` maps host `PORT` (default 8081) → container 8080.
- In Docker the API reaches Postgres via the `db` service host; on the host it connects to `localhost:5433` (see `.env.example`).

## Database migrations

Use **golang-migrate** with numbered up/down SQL files in `migrations/` (e.g. `003_*.up.sql` / `003_*.down.sql`). The schema follows the richer species/forms/types design (migrations 003/004 transition to it), not a single `pokemon` table.

## Git workflow

Branch off `development` and PR back into it; `main` is the release branch.
