# pkm-atlas-api

A Go HTTP API for a Pokémon Dex Tracker, backed by PostgreSQL.

## Database migrations

Migrations live in `migrations/` as numbered [golang-migrate](https://github.com/golang-migrate/migrate)
up/down SQL files. Install the `migrate` CLI first (e.g.
`brew install golang-migrate`).

Start Postgres (host port `5433` → container `5432`):

```sh
docker compose up -d db
```

Apply all migrations:

```sh
migrate -path migrations -database "postgres://postgres:postgres@localhost:5433/pkm_tracker?sslmode=disable" up
```

Roll back the most recent migration:

```sh
migrate -path migrations -database "postgres://postgres:postgres@localhost:5433/pkm_tracker?sslmode=disable" down 1
```

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
