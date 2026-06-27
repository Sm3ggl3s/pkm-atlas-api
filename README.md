# pkm-atlas-api

A Go HTTP API for a Pokémon Dex Tracker, backed by PostgreSQL.

Common tasks are wrapped in a `Makefile` — run `make help` to list them
(e.g. `make db-up`, `make migrate-up`, `make test`, `make run`).

## Database migrations

Migrations live in `migrations/` as numbered [golang-migrate](https://github.com/golang-migrate/migrate)
up/down SQL files.

The migrations run **inside the db container** — the `migrate` CLI is baked into
the custom db image (`Dockerfile.db`), so no extra container and no host
`migrate` CLI are needed, just Docker.

Start Postgres (host port `5433` → container `5432`) and apply all migrations:

```sh
make db-up        # builds the custom db image (first run) and starts Postgres
make migrate-up   # runs `migrate` inside the db container
```

Roll back the most recent migration:

```sh
make migrate-down
```

Other helpers: `make migrate-version`, `make migrate-create name=add_users`,
`make migrate-drop`. Run `make help` for the full list.

> **Optional (host CLI):** if you prefer running migrations from the host,
> `brew install golang-migrate`, then:
> `migrate -path migrations -database "postgres://postgres:postgres@localhost:5433/pkm_tracker?sslmode=disable" up`

Inspect the schema:

```sh
# List tables
docker compose exec db psql -U postgres -d pkm_tracker -c "\dt"

# Describe a table
docker compose exec db psql -U postgres -d pkm_tracker -c "\d pokemon_species"
```

### Schema overview

The catalogue models the Pokémon Dex without user data yet:

- `pokemon_generations` — generation metadata (number, name, region).
- `pokemon_species` — National Dex species (carries no type columns).
- `pokemon_forms` — forms/variants of a species (default, regional, mega, etc.).
- `pokemon_types` — the 18 Pokémon types.
- `pokemon_form_types` — joins forms to their type(s), with slot order.

Types belong to **forms**, not species, because forms of the same species can
have different typings (e.g. Mega Charizard X is Fire/Dragon).

Migration `000002` seeds reference data (the 9 generations and 18 types) only.
