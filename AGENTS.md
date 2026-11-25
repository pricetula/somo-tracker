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

## Secret Management & Config

- **Doppler injects all configuration at runtime.** No fallback defaults in `config.go` — every value comes from the environment (`firstNonEmpty` with no default string).
- Required values (`DATABASE_URL`, `BREVO_API_KEY`, `EMAIL_HMAC_KEY`, `SESSION_COOKIE_NAME`) are validated at startup in `config.validate()`.
- Do not add real secret values to `.env` (which is gitignored anyway).

## Integration Tests

- Integration tests run **locally only** (not in CI) via testcontainers, which spins up real PostgreSQL and Redis containers.
- Run with: `cd backend && go test -tags=integration ./...`
- Requires Docker to be available locally.

## Database Version Standard

- **PostgreSQL 18** is the standard version everywhere: `docker-compose.yml`, integration tests, and any local dev DB.
- Integration tests in `backend/internal/auth/service/service_integration_test.go` must use `postgres:18` (testcontainers), matching `docker-compose.yml`.

## Git & Release Workflow

- **Conventional commits**: All commits must use conventional commit format (`feat:`, `fix:`, `docs:`, `refactor:`, `test:`, `chore:`, etc.). Examples:
  - `feat: add Cloudflare Turnstile protection to OTP request flow`
  - `fix: add challengeId dependency to verifyOTP useCallback`
  - `docs: remove stale BASELINE docs`
- **No direct pushes to `main` or `develop`**: Always create a feature branch from `develop`, then open a pull request against `develop`. Never push directly to `main` or `develop`.
- **Small, well-documented PRs**: Keep commits focused on one logical change. PRs must be small, well-documented, and target `develop` (not `main`).
- **PRs against `develop`**: All feature work is merged into `develop` first. `main` is updated via release PRs from `develop`.
- **Pre-push compliance**: A lefthook pre-push hook runs `go vet`, `golangci-lint`, ESLint, `tsc`, and `go test -short`. All must pass before pushing.
- **CI on push/PR**: GitHub Actions runs unit tests on every push and PR to `main`. Do not push broken code.
- **Run tests before committing**: Always run `cd backend && go test ./...` and `cd frontend && pnpm typecheck` before committing.
- **No secrets in git**: Never commit real secret values to `.env` (gitignored) or any tracked file. Secrets are injected by Doppler at runtime.
- **Push to main**: After local verification and pre-push hook success, push to `origin/main`. GitHub CI will validate.

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