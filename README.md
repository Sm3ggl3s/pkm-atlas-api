# pkm-altas-api

A small Go 1.25 HTTP API for a Pokémon Dex Tracker, built on the standard
library (`net/http`) with PostgreSQL via `pgx/v5`.

## Running

```bash
# Hot reload on the host (requires Air)
air -c .air.toml

# Full stack (API + Postgres) in Docker
docker compose up --build
```

The container always listens on port `8080`; the host reaches it on
`localhost:8081` (see `docker-compose.yml`). Copy `.env.example` to `.env`
before running — `DATABASE_URL` is required.

## Testing

Unit tests use only the standard library and do **not** require Postgres or
Docker, so they are safe to run anywhere:

```bash
go test ./...
```

### Integration tests (optional)

Database integration tests are skipped unless `TEST_DATABASE_URL` is set. To run
them against the dockerized Postgres (start it first with `docker compose up`):

```bash
TEST_DATABASE_URL="postgres://postgres:postgres@localhost:5433/pkm_tracker?sslmode=disable" go test ./...
```

## Linting

Linting uses [golangci-lint](https://golangci-lint.run/) v2 (config in
`.golangci.yml`):

```bash
golangci-lint run
```

Install it with `brew install golangci-lint`. If you don't want a local install,
you can run the same version CI uses with:

```bash
go run github.com/golangci/golangci-lint/v2/cmd/golangci-lint@latest run
```

## CI

GitHub Actions (`.github/workflows/ci.yml`, "Go CI") runs `go test ./...` and
`golangci-lint run` on pushes to `development` and on PRs targeting it.

## Git workflow

Branch off `development` and PR back into it; `main` is the release branch.
