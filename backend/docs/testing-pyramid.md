# Backend Test Pyramid

## Overview

The backend follows a three-layer test pyramid aligned with the project contract:

- **Unit → pure logic, no DB**
- **Integration → sqlc queries, services with test DB**
- **E2E → HTTP handlers + router + auth middleware**

All tests must follow the canonical error response contract and error handling rules in the root `AGENTS.md`.

## Layer Definitions

### 1. Unit Tests
**Scope:** Pure logic, no DB, no external services.

**Location:** `internal/*/ *_unit_test.go` or `*_test.go` with `t.Parallel()` and mocks.

**Characteristics:**
- No Docker, no Postgres, no testdb.
- Mock all dependencies via interfaces.
- Run fast: `make test-short` includes these.
- Tag: none (short mode).

**Examples:**
- `internal/api/admin_invitation_handler_unit_test.go` – handler validation with mock service.
- `internal/worker/admin_invitation_worker_unit_test.go` – Stytch error classification, circuit breaker.

**Rules:**
- Use `testify` assert/require.
- Never hit real DB or network.
- Keep tests `< 100ms`.

### 2. Integration Tests
**Scope:** sqlc queries, services with test DB.

**Location:** `internal/*/*_integration_test.go` and `test/integration/*`.

**Characteristics:**
- Real Postgres via `internal/testdb` Docker pool.
- Migrations applied: `testdb.Setup(t)` runs migrations.
- Tagged `//go:build integration` where needed.
- Uses `testdb.DB(t)` and `testdb.BeginTx(t)` for isolation.
- Run with `make test-integration`.

**Examples:**
- `internal/database/migrator_integration_test.go`
- `internal/services/admin_invitation_service_integration_test.go`
- `test/integration/admin_invitation_integration_test.go`

**Rules:**
- Never use `t.Parallel()` across shared DB without isolation.
- Every new migration SQL must have a corresponding test in `internal/database/migrator_integration_test.go`.
- Assert schema via `information_schema`, not complex joins.

### 3. E2E Tests
**Scope:** HTTP handlers + router + auth middleware.

**Location:** `internal/api/*_handler_test.go` with Fiber app, `test/integration/*`.

**Characteristics:**
- Full request/response cycle using `app.Test`.
- Inject locals for auth: `active_school_id`, `tenant_id`, `user_id`.
- Test middleware chain: auth, CORS, rate limit, etc.
- Can use test DB for persistence assertions.

**Examples:**
- `internal/api/academic_period_handler_test.go` – handler + router with middleware injection.
- `internal/services/auth_service_e2e_test.go` – service-level E2E with real DB.

**Rules:**
- Verify canonical error JSON shape for non-2xx.
- Never add back buttons – not applicable to backend.
- Auth middleware must be exercised via locals or real session.

## Running Tests

```bash
# Unit only
make test-short

# Integration (requires Docker)
make test-db-up
make test-integration
make test-db-down

# All
make test-all
```

## Test Data Management

- `internal/testdb/testdb.go` – Docker Postgres 16-alpine per test run.
- `internal/testdb/seeds.go` – Load JSON seed files from `db/seeds/`.
- Use `tx.Rollback()` for test isolation.
- Do not share mutable state between parallel tests.

## Naming Conventions

- Unit: `*_unit_test.go`
- Integration: `*_integration_test.go`
- E2E: `*_e2e_test.go` or handler tests `*_handler_test.go`
- Migration: `migrator_integration_test.go`

## Pyramid Ratio Target

- Unit: ~70%
- Integration: ~20%
- E2E: ~10%

Keep unit tests fast and numerous, integration tests focused on sqlc/service boundaries, E2E tests covering critical happy paths and auth flows.
