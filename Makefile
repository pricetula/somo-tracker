.PHONY: up up-d down ps logs-down logs-api logs-postgres logs-redis logs-frontend restart-api rebuild-api build-api sqlc-gen generate-swagger generate-api-types lint vet test test-short test-integration test-verbose test-all help migrated migrate-down migrate-range migrate-verify test-db-up test-db-down test-db-status test-db-logs test-db-reset

# ─── Docker Compose shortcuts ────────────────────────────────────────────────

up:    ## Start all services (detached)
	doppler run -p somotracker-backend -c dev -- \
	doppler run -p somotracker-frontend -c dev -- \
	docker compose up -d

up-d:  ## Start all services (foreground, live logs)
	doppler run -p somotracker-backend -c dev -- \
	doppler run -p somotracker-frontend -c dev -- \
	docker compose up

down:  ## Stop and remove all containers
	docker compose down

ps:    ## Show running service status
	docker compose ps

# ─── Logs ────────────────────────────────────────────────────────────────────

logs-down:  ## Tail logs for all services
	docker compose logs -f

logs-api:   ## Tail API (Go/Fiber) logs
	docker compose logs -f somotracker_api

logs-postgres: ## Tail Postgres logs
	docker compose logs -f somotracker_postgres

logs-redis: ## Tail Redis logs
	docker compose logs -f somotracker_redis

logs-frontend: ## Tail Next.js frontend logs
	docker compose logs -f somotracker_frontend

# ─── Service lifecycle ────────────────────────────────────────────────────────

restart-api:  ## Restart the API service (triggers Air hot-reload)
	docker compose restart somotracker_api

rebuild-api:  ## Rebuild and recreate the API container
	docker compose up -d --force-recreate --no-deps somotracker_api

build-api:    ## Build the API service (no start)
	docker compose build somotracker_api

# ─── Quality ────────────────────────────────────────────────────────────────

lint:  ## Run golangci-lint (backend)
	cd backend && golangci-lint run

vet:   ## Run go vet (backend)
	cd backend && go vet ./...

# ─── Tests ───────────────────────────────────────────────────────────────────

GOTESTSUM := go run gotest.tools/gotestsum@latest

test: test-short  ## Run unit tests (short mode, skips integration)

test-short:  ## Run unit tests only, race-detected, junit output
	cd backend && $(GOTESTSUM) --junitfile ../test-results/unit.xml -- -race -short -count=1 ./...

test-integration:  ## Run integration tests (requires Docker)
	cd backend && $(GOTESTSUM) --junitfile ../test-results/integration.xml -- -race -count=1 ./...

# Shared long-lived test database.
# `make test-db-up` starts the postgres:16-alpine instance on port 5433 with
# tmpfs storage. `make test` (above) then runs -tags=integration tests
# against it without spinning up containers per test.
TEST_DSN ?= postgres://somo_admin:somo_secure_password@127.0.0.1:5433/somotracker_test?sslmode=disable

test-db-up:  ## Start the test Postgres (port 5433, tmpfs, no data persistence)
	docker compose -f docker-compose.test.yml up -d
	@echo "Waiting for Postgres to become ready..."
	@for i in $$(seq 1 30); do \
		if docker exec somotracker_postgres_test pg_isready -U somo_admin -d somotracker_test >/dev/null 2>&1; then \
			echo "Test Postgres is ready on port 5433."; \
			exit 0; \
		fi; \
		sleep 1; \
	done; \
	echo "Test Postgres failed to become ready within 30s. Check: docker compose -f docker-compose.test.yml logs"; \
	exit 1

test-db-down:  ## Tear down the test Postgres
	docker compose -f docker-compose.test.yml down

test-db-status:  ## Print test Postgres container status
	docker compose -f docker-compose.test.yml ps

test-db-logs:  ## Tail test Postgres logs
	docker compose -f docker-compose.test.yml logs -f

test-db-reset:  ## Recreate the test Postgres (drops all data)
	docker compose -f docker-compose.test.yml down
	docker compose -f docker-compose.test.yml up -d
	@echo "Waiting for Postgres to become ready..."
	@for i in $$(seq 1 30); do \
		if docker exec somotracker_postgres_test pg_isready -U somo_admin -d somotracker_test >/dev/null 2>&1; then \
			echo "Test Postgres is ready on port 5433."; \
			exit 0; \
		fi; \
		sleep 1; \
	done; \
	echo "Test Postgres failed to become ready within 30s."; \
	exit 1

test-verbose:  ## Run all tests with verbose output
	cd backend && $(GOTESTSUM) --format standard-verbose -- -race -count=1 ./...

test-coverage:  ## Run tests with coverage report
	cd backend && go test -race -short -count=1 -coverprofile=../coverage.out -covermode=atomic ./...
	cd backend && go tool cover -html=../coverage.out -o ../coverage.html
	@echo "Coverage report: coverage.html"

test-all: test-short test-integration  ## Run unit + integration tests

# ─── Code generation ─────────────────────────────────────────────────────────

sqlc-gen:  ## Run sqlc code generation (requires queries in db/queries/)
	cd backend && sqlc generate

generate-swagger:  ## Generate Swagger/OpenAPI docs from Go annotations
	cd backend && swag init -g cmd/api/main.go --output docs --parseDependency --parseInternal

generate-api-types: generate-swagger  ## Generate TypeScript types from swagger.json
	cd frontend && pnpm generate:api

generate: sqlc-gen generate-api-types  ## Run all code generation (sqlc + swagger + TS types)

# ─── Help ─────────────────────────────────────────────────────────────────────

help:  ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) \
		| sort \
		| awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}'

setup-hooks:
	lefthook install

# ─── Database migrations (golang-migrate) ───────────────────────────────────

MIGRATE_DIR := backend/db/migrations
MIGRATE_URL := $(DATABASE_URL)

migrated:  ## Apply pending migrations against the dev/postgres database
	migrate -database "$(MIGRATE_URL)" -path $(MIGRATE_DIR) up

migrate-down:  ## Roll back one migration
	migrate -database "$(MIGRATE_URL)" -path $(MIGRATE_DIR) down 1

migrate-range:  ## Migrate to a specific version (e.g. 1, 2, ...)
	@migrate -database "$(MIGRATE_URL)" -path $(MIGRATE_DIR) goto $(VERSION)

migrate-verify:  ## Force version check / status
	migrate -database "$(MIGRATE_URL)" -path $(MIGRATE_DIR) version
