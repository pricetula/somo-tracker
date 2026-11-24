# Local Development Runbook

## Prerequisites

- **Go** (see `backend/go.mod`)
- **Node.js** + **pnpm** (see `frontend/package.json`)
- **Docker** (for PostgreSQL + Redis — managed externally, do not run `docker compose` from the agent)
- **Doppler** (for secrets injection)

> **Rule**: The agent must not run Docker commands. Docker is managed separately. The agent only writes code, runs tests, and analyzes files.

## Step 1: Infrastructure

Start PostgreSQL 18 and Redis 7 via Docker Compose (managed externally):

```bash
# From repo root — run this yourself, not via the agent
docker compose up -d postgres redis
```

Verify they are healthy:
- PostgreSQL: `pg_isready -U somotracker -d somotracker` on port 5432
- Redis: `redis-cli ping` on port 6379

## Step 2: Secrets (Doppler)

All configuration is injected by Doppler at runtime. No `.env` values are committed.

Required backend variables:
```
DATABASE_URL=postgres://somotracker:somotracker@localhost:5432/somotracker?sslmode=disable
REDIS_URL=redis://localhost:6379
BREVO_API_KEY=<from Doppler>
EMAIL_HMAC_KEY=<32-byte secret from Doppler>
SESSION_COOKIE_NAME=session
TURNSTILE_SECRET_KEY=<from Doppler>
TURNSTILE_EXPECTED_HOST=localhost:3000
CORS_ORIGINS=http://localhost:3000
APP_ENV=development
PORT=3030
```

Required frontend variables:
```
API_URL=http://localhost:3030
APP_URL=http://localhost:3000
SESSION_COOKIE_NAME=session
APP_ENV=development
NODE_ENV=development
PORT=3000
HOSTNAME=0.0.0.0
```

## Step 3: Run Backend

```bash
cd backend
go mod download
go test ./...           # run unit tests
go run ./cmd/api        # start API server
```

Backend listens on port 3030 (or `$PORT`).

## Step 4: Run Frontend

```bash
cd frontend
pnpm install
pnpm typecheck          # verify types
pnpm dev                # start Next.js dev server
```

Frontend listens on port 3000 (or `$PORT`).

Open `http://localhost:3000/login` to test the OTP flow.

## Step 5: Run Tests

```bash
# Backend unit tests
cd backend && go test ./...

# Backend vet + lint
cd backend && go vet ./...
cd backend && golangci-lint run ./...

# Frontend typecheck
cd frontend && pnpm typecheck

# Frontend lint
cd frontend && pnpm lint

# Frontend tests
cd frontend && pnpm test
```

## Step 6: Integration Tests (Local Only)

Integration tests use testcontainers (requires Docker). Run locally only — not in CI:

```bash
cd backend && go test -tags=integration ./...
```

Startup integration test:
```bash
cd backend && go test -tags=integration -v -run TestStartup ./internal/app/
```

## Step 7: Verify End-to-End

1. Open `http://localhost:3000/login`
2. Enter an email → Turnstile widget appears → click "Send code"
3. Check that the request reaches the Go backend (look at backend logs)
4. Since email is fire-and-forget via Brevo, check the `email_outbox` table for a `pending` record
5. The frontend switches to "Verify code" step
6. Enter a 6-digit code → session cookie is set → redirect to `/onboarding` or `/dashboard`

## Troubleshooting

| Issue | Likely Cause | Fix |
|-------|-------------|-----|
| `connect redis: connection refused` | `REDIS_URL` empty or wrong host | Set `REDIS_URL=redis://localhost:6379` |
| `missing required environment variables` | Doppler not injecting a required var | Add var to Doppler, restart |
| `invalid configuration` | Missing `TURNSTILE_SECRET_KEY`, `SESSION_COOKIE_NAME`, etc. | Add to Doppler config |
| CORS error in browser | `CORS_ORIGINS` doesn't include frontend origin | Set `CORS_ORIGINS=http://localhost:3000` |
| Turnstile widget not loading | Site key mismatch or ad blocker | Check `0x4AAAAAAGHleNH8xjjhEUQL` is hard-coded |
| 403 on OTP request | Turnstile token missing or verification failed | Ensure token is sent; check `TURNSTILE_SECRET_KEY` matches site key |

## Database

Migrations are managed by goose under `backend/migrations/`. They run automatically on backend startup via `app.New()`.

```bash
# Check migration status (manual, not via agent)
cd backend && goose postgres "$DATABASE_URL" status
```

## Ports

| Service | Port | Notes |
|---------|------|-------|
| PostgreSQL | 5432 | Via Docker |
| Redis | 6379 | Via Docker |
| Go Backend | 3030 | `$PORT` |
| Next.js Frontend | 3000 | `$PORT` |
