# Somotracker Makefile

.PHONY: help \
	dev dev-local dev-stop dev-logs \
	build build-backend build-frontend \
	test test-backend test-frontend test-coverage test-integration \
	generate generate-sqlc generate-swag generate-frontend-types \
	compliance lint vet fmt typecheck \
	docker-build docker-up docker-down docker-logs docker-ps \
	db-migrate-up db-migrate-down db-migrate-create db-seed \
	clean clean-all

# Default target
help:
	@echo "Somotracker - Development Commands"
	@echo ""
	@echo "Development:"
	@echo "  make dev           - Start full dev stack (docker compose with HMR)"
	@echo "  make dev-local     - Start DBs in docker, run backend/frontend locally"
	@echo "  make dev-stop      - Stop dev stack"
	@echo "  make dev-logs      - Follow dev stack logs"
	@echo ""
	@echo "Build:"
	@echo "  make build         - Build backend + frontend"
	@echo "  make build-backend - Build backend binary"
	@echo "  make build-frontend- Build frontend for production"
	@echo ""
	@echo "Test:"
	@echo "  make test              - Run all tests"
	@echo "  make test-backend      - Run backend tests"
	@echo "  make test-frontend     - Run frontend tests"
	@echo "  make test-coverage     - Run tests with coverage"
	@echo "  make test-integration  - Run integration tests (requires DB)"
	@echo ""
	@echo "Generate:"
	@echo "  make generate               - Generate all code"
	@echo "  make generate-sqlc          - Generate sqlc code"
	@echo "  make generate-swag          - Generate Swagger docs"
	@echo "  make generate-frontend-types- Generate frontend API types"
	@echo ""
	@echo "Quality:"
	@echo "  make compliance    - Run all compliance checks (vet, lint, typecheck, test)"
	@echo "  make lint          - Run linters (backend + frontend)"
	@echo "  make vet           - Run go vet"
	@echo "  make fmt           - Format code (backend + frontend)"
	@echo "  make typecheck     - Run TypeScript typecheck (frontend)"
	@echo ""
	@echo "Docker:"
	@echo "  make docker-build  - Build docker images"
	@echo "  make docker-up     - Start docker stack (detached)"
	@echo "  make docker-down   - Stop docker stack (with volumes)"
	@echo "  make docker-logs   - Follow docker logs"
	@echo "  make docker-ps     - Show docker containers"
	@echo ""
	@echo "Database:"
	@echo "  make db-migrate-up      - Run migrations up"
	@echo "  make db-migrate-down    - Run migrations down"
	@echo "  make db-migrate-create  - Create new migration (name=xxx)"
	@echo "  make db-seed            - Seed development data"
	@echo ""
	@echo "Clean:"
	@echo "  make clean      - Clean build artifacts"
	@echo "  make clean-all  - Clean everything including node_modules, docker volumes"

# =============================================================================
# Development
# =============================================================================

# Full docker dev stack with HMR (frontend: vite, backend: air)
dev:
	docker compose up --build

# Local dev: DBs in docker, backend/frontend run natively
# Requires: go, air, pnpm, node installed locally
dev-local:
	docker compose up -d postgres redis
	@echo "Waiting for DBs to be ready..."
	@sleep 3
	@echo "Starting backend (air)..."
	@cd backend && air -c .air.toml & \
	echo $$! > /tmp/somotracker-backend.pid
	@echo "Starting frontend (vite)..."
	@cd frontend && pnpm dev & \
	echo $$! > /tmp/somotracker-frontend.pid
	@echo ""
	@echo "Backend PID: $$(cat /tmp/somotracker-backend.pid)"
	@echo "Frontend PID: $$(cat /tmp/somotracker-frontend.pid)"
	@echo "Stop with: make dev-stop"

dev-stop:
	@if [ -f /tmp/somotracker-backend.pid ]; then kill $$(cat /tmp/somotracker-backend.pid) 2>/dev/null; rm /tmp/somotracker-backend.pid; fi
	@if [ -f /tmp/somotracker-frontend.pid ]; then kill $$(cat /tmp/somotracker-frontend.pid) 2>/dev/null; rm /tmp/somotracker-frontend.pid; fi
	docker compose down

dev-logs:
	docker compose logs -f

# =============================================================================
# Build
# =============================================================================

build: build-backend build-frontend

build-backend:
	cd backend && CGO_ENABLED=0 go build -o bin/api ./cmd/api
	@if [ -d ./backend/cmd/worker ]; then \
		cd backend && CGO_ENABLED=0 go build -o bin/worker ./cmd/worker; \
	fi

build-frontend:
	cd frontend && pnpm build

# =============================================================================
# Test
# =============================================================================

test: test-backend test-frontend

test-backend:
	cd backend && go test -v ./...

test-frontend:
	cd frontend && pnpm test

test-coverage:
	cd backend && go test -coverprofile=coverage.out ./...
	@cd backend && go tool cover -html=coverage.out -o coverage.html
	@echo "Backend coverage: backend/coverage.html"
	cd frontend && pnpm test -- --coverage

test-integration:
	@if [ -z "$$POSTGRES_DSN" ]; then \
		echo "POSTGRES_DSN not set. Using docker compose DB..."; \
		export POSTGRES_DSN="postgres://somotracker:somotracker@localhost:5432/somotracker?sslmode=disable"; \
	fi
	cd backend && go test -tags=integration -v ./...

# =============================================================================
# Generate
# =============================================================================

generate: generate-sqlc generate-swag generate-frontend-types

generate-sqlc:
	cd backend && sqlc generate

generate-swag:
	cd backend && swag init -g cmd/api/main.go -o docs

generate-frontend-types:
	cd frontend && pnpm generate:api

# =============================================================================
# Quality / Compliance
# =============================================================================

compliance: vet lint typecheck test-backend generate-check

vet:
	cd backend && go vet ./...

lint:
	@echo "=== Backend lint ==="
	cd backend && golangci-lint run ./...
	@echo "=== Frontend lint ==="
	cd frontend && pnpm lint

fmt:
	@echo "=== Backend fmt ==="
	cd backend && go fmt ./...
	@echo "=== Frontend fmt ==="
	cd frontend && pnpm format

typecheck:
	cd frontend && pnpm typecheck

generate-check: generate
	@if [ -n "$$(git status --porcelain)" ]; then \
		echo "Generated files have changes. Commit them."; \
		git diff --stat; \
		exit 1; \
	fi

# =============================================================================
# Docker
# =============================================================================

docker-build:
	docker compose build

docker-up:
	docker compose up -d

docker-down:
	docker compose down -v

docker-logs:
	docker compose logs -f

docker-ps:
	docker compose ps

# =============================================================================
# Database
# =============================================================================

# Requires goose: go install github.com/pressly/goose/v3/cmd/goose@latest
db-migrate-up:
	@if [ -z "$$POSTGRES_DSN" ]; then \
		echo "POSTGRES_DSN not set. Using default..."; \
		export POSTGRES_DSN="postgres://somotracker:somotracker@localhost:5432/somotracker?sslmode=disable"; \
	fi
	cd backend && goose -dir migrations postgres "$$POSTGRES_DSN" up

db-migrate-down:
	@if [ -z "$$POSTGRES_DSN" ]; then \
		echo "POSTGRES_DSN not set. Using default..."; \
		export POSTGRES_DSN="postgres://somotracker:somotracker@localhost:5432/somotracker?sslmode=disable"; \
	fi
	cd backend && goose -dir migrations postgres "$$POSTGRES_DSN" down

db-migrate-create:
	@if [ -z "$(name)" ]; then echo "Usage: make db-migrate-create name=add_users_table"; exit 1; fi
	cd backend && goose -dir migrations create $(name) sql

db-seed:
	@if [ -z "$$POSTGRES_DSN" ]; then \
		echo "POSTGRES_DSN not set. Using default..."; \
		export POSTGRES_DSN="postgres://somotracker:somotracker@localhost:5432/somotracker?sslmode=disable"; \
	fi
	@echo "Seeding database..."
	@cd backend && go run ./cmd/seed

# =============================================================================
# Clean
# =============================================================================

clean:
	rm -rf backend/bin backend/coverage.out backend/coverage.html
	rm -rf frontend/dist
	docker compose down

clean-all: clean
	rm -rf frontend/node_modules
	docker compose down -v
	docker system prune -f