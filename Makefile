# Somotracker Makefile

.PHONY: help dev build test test-short test-coverage test-integration generate compliance lint vet fmt clean

# Default target
help:
	@echo "Available targets:"
	@echo "  dev              - Start local development (requires docker compose up -d postgres redis)"
	@echo "  build            - Build all services"
	@echo "  build-backend    - Build backend binary"
	@echo "  build-frontend   - Build frontend"
	@echo "  test             - Run all tests"
	@echo "  test-short       - Run unit tests only"
	@echo "  test-coverage    - Run tests with coverage"
	@echo "  test-integration - Run integration tests (requires postgres)"
	@echo "  generate         - Generate code (sqlc, swag, frontend types)"
	@echo "  compliance       - Run compliance checks"
	@echo "  lint             - Run linters"
	@echo "  vet              - Run go vet"
	@echo "  fmt              - Format code"
	@echo "  clean            - Clean build artifacts"

# Development
dev:
	docker compose up -d postgres redis
	@echo "Starting backend..."
	cd backend && go run ./cmd/api &
	@echo "Starting frontend..."
	cd frontend && pnpm dev

# Build
build: build-backend build-frontend

build-backend:
	cd backend && CGO_ENABLED=0 go build -o bin/api ./cmd/api
	cd backend && CGO_ENABLED=0 go build -o bin/worker ./cmd/worker

build-frontend:
	cd frontend && pnpm build

# Testing
test: test-short test-integration

test-short:
	cd backend && go test -short ./...

test-coverage:
	cd backend && go test -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html

test-integration:
	cd backend && go test -tags=integration ./...

# Code Generation
generate: generate-sqlc generate-swag generate-frontend-types

generate-sqlc:
	@echo 'sqlc generate skipped (scaffolding)'

generate-swag:
	@echo 'swag init skipped (scaffolding)'

generate-frontend-types:
	cd frontend && pnpm generate:api

# Compliance & Quality
compliance: vet lint test-short generate-check

vet:
	cd backend && go vet ./...

lint:
	cd backend && golangci-lint run ./...
	cd frontend && pnpm lint

fmt:
	cd backend && go fmt ./...
	cd frontend && pnpm format

generate-check: generate
	@if [ -n "$$(git status --porcelain)" ]; then \
		echo "Generated files have changes. Commit them."; \
		git diff --stat; \
		exit 1; \
	fi

# Docker
docker-build:
	docker compose build

docker-up:
	docker compose up -d

docker-down:
	docker compose down -v

# Database
migrate-up:
	cd backend && goose -dir migrations postgres "$(POSTGRES_DSN)" up

migrate-down:
	cd backend && goose -dir migrations postgres "$(POSTGRES_DSN)" down

migrate-create:
	cd backend && goose -dir migrations create $(name) sql

# Clean
clean:
	rm -rf backend/bin backend/coverage.out backend/coverage.html
	rm -rf frontend/dist frontend/node_modules
	docker compose down -v