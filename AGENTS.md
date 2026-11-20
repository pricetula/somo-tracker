# Agent Rules

## RULE #1: DO NOT RUN DOCKER

**Never run `docker compose`, `docker build`, `docker run`, or any Docker commands.**

This includes:
- `docker compose up`
- `docker compose down`
- `docker compose build`
- `docker compose logs`
- `docker compose config`
- Any `docker` command

**Reason:** Docker is managed separately. Agent should only write code, run tests, and analyze files.

---

## Database Version Standard

- **PostgreSQL 18** is the standard version everywhere: `docker-compose.yml`, integration tests, and any local dev DB.
- Integration tests in `backend/internal/auth/service/service_integration_test.go` must use `postgres:18` (testcontainers), matching `docker-compose.yml`.

## Other Rules

- Run backend tests with: `cd backend && go test ./...`
- Run frontend typecheck with: `cd frontend && pnpm typecheck`
- Run integration tests with: `cd backend && go test -tags=integration ./...`
- Run startup integration test with: `cd backend && go test -tags=integration -v -run TestStartup ./internal/app/`
- Make changes via `edit`/`write` tools only
- No manual container management

## Startup Test Coverage

`backend/internal/app/app_startup_integration_test.go` (`TestStartup_AppCreatesSuccessfully`) verifies the full backend startup path:

1. Real PostgreSQL 18 via testcontainers + goose migrations
2. Real Redis 7 via testcontainers + `PING`
3. `app.New(cfg)` construction (the path that previously failed with `failed to create app: connect redis: connection refused`)
4. `/readyz` dependencies reachable (DB ping + Redis ping)
5. Graceful shutdown

Run it before/after changes to config, DB, or Redis startup logic.