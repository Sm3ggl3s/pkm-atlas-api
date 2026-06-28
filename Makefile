# Load .env (if present) so DATABASE_URL etc. are available to targets.
-include .env
export

POSTGRES_USER     ?= postgres
POSTGRES_PASSWORD ?= postgres
POSTGRES_DB       ?= pkm_tracker
MIGRATIONS_DIR    := migrations

# Host-oriented URL (only needed if you run the migrate CLI on the host directly).
DATABASE_URL ?= postgres://postgres:postgres@localhost:5433/pkm_tracker?sslmode=disable

# Migrations run INSIDE the db container (the migrate CLI is baked into the db
# image), so no extra container and no host migrate CLI are required. From inside
# that container Postgres is reachable on localhost:5432.
DB_URL_LOCAL := postgres://$(POSTGRES_USER):$(POSTGRES_PASSWORD)@localhost:5432/$(POSTGRES_DB)?sslmode=disable
MIGRATE_EXEC := docker compose exec -T db migrate -path=/migrations -database "$(DB_URL_LOCAL)"

.DEFAULT_GOAL := help

## ---- Go / app ----

.PHONY: run
run: ## Run with hot reload (air)
	air -c .air.toml

.PHONY: build
build: ## Build the API binary to ./tmp/api
	go build -o ./tmp/api ./cmd/api

.PHONY: test
test: ## Run all tests
	go test ./...

.PHONY: vet
vet: ## Run go vet
	go vet ./...

.PHONY: fmt
fmt: ## Format all Go files
	gofmt -w .

.PHONY: tidy
tidy: ## Tidy go.mod / go.sum
	go mod tidy

.PHONY: lint
lint: ## Run golangci-lint if installed
	@command -v golangci-lint >/dev/null 2>&1 && golangci-lint run || echo "golangci-lint not installed (brew install golangci-lint)"

## ---- Database / Docker ----

.PHONY: db-up
db-up: ## Start the Postgres container
	docker compose up -d db

.PHONY: db-down
db-down: ## Stop and remove containers
	docker compose down

.PHONY: db-psql
db-psql: ## Open a psql shell in the db container
	docker compose exec db psql -U $(POSTGRES_USER) -d $(POSTGRES_DB)

.PHONY: db-tables
db-tables: ## List tables in the database
	docker compose exec db psql -U $(POSTGRES_USER) -d $(POSTGRES_DB) -c "\dt"

.PHONY: up
up: ## Start the full stack (API + Postgres)
	docker compose up

.PHONY: down
down: ## Stop the full stack
	docker compose down

.PHONY: logs
logs: ## Follow container logs
	docker compose logs -f

## ---- Migrations (golang-migrate) ----

.PHONY: migrate-up
migrate-up: ## Apply all up migrations (inside the db container)
	$(MIGRATE_EXEC) up

.PHONY: migrate-down
migrate-down: ## Roll back the last migration (inside the db container)
	$(MIGRATE_EXEC) down 1

.PHONY: migrate-drop
migrate-drop: ## Drop everything in the database (DANGER)
	$(MIGRATE_EXEC) drop -f

.PHONY: migrate-version
migrate-version: ## Show the current migration version
	$(MIGRATE_EXEC) version

.PHONY: migrate-force
migrate-force: ## Force a migration version (usage: make migrate-force version=1)
	$(MIGRATE_EXEC) force $(version)

.PHONY: migrate-create
migrate-create: ## Create a new migration (usage: make migrate-create name=add_users)
	docker compose exec -T --user $$(id -u):$$(id -g) db \
		migrate create -ext sql -dir /migrations -seq $(name)

## ---- Help ----

.PHONY: help
help: ## Show this help
	@awk 'BEGIN {FS = ":.*## "} \
		/^## / {printf "\n\033[1m%s\033[0m\n", substr($$0, 4)} \
		/^[a-zA-Z0-9_-]+:.*## / {printf "  \033[36m%-16s\033[0m %s\n", $$1, $$2}' $(MAKEFILE_LIST)
