# Contributing to Somotracker

## Getting Started

1. Clone the repository.
2. Install dependencies: `pnpm install` (root/workspaces) and `cd backend && go mod download`.
3. Configure secrets via Doppler — all runtime configuration is injected by Doppler; no `.env` values are committed.
4. Run backend tests: `cd backend && go test ./...`
5. Run frontend typecheck: `cd frontend && pnpm typecheck`

## Making Changes

1. Create a feature branch from `main`.
2. Implement changes following the per-directory `AGENTS.md` contracts.
3. Add unit tests for new backend functions and handlers.
4. Run the pre-commit checks (golangci-lint, eslint, tsc, go test).
5. Open a pull request against `main`.
6. GitHub CI runs on every push/PR to `main`.

## Code Review Checklist

- [ ] Backend: handler → service → store layer separation respected
- [ ] Backend: errors use canonical `apperr.Error` contract
- [ ] Backend: no fallback defaults in `config.go`; required values in `validate()`
- [ ] Backend: unit tests for new functions/handlers
- [ ] Frontend: one component per file, no div soup, no hardcoded hex colors
- [ ] Frontend: async handlers have try/catch
- [ ] Frontend: typecheck passes
- [ ] Docs: new features documented in `content/docs` or `docs/`
- [ ] Config: new env vars added to `docs/secrets.md`

## Secrets Management

All secrets are injected by Doppler at runtime. Never commit real secret values to `.env` (which is gitignored). To add a new secret:

1. Add it to Doppler (development + other environments).
2. Read it in `backend/internal/platform/config/config.go` via `firstNonEmpty("YOUR_VAR")` — no fallback default.
3. If required, add it to the `required` map in `config.validate()`.
4. Update `docs/secrets.md`.

## Docker

Docker is managed separately. Do not run `docker compose`, `docker build`, `docker run`, or any Docker commands locally.
