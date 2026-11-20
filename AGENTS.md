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

## Other Rules

- Run backend tests with: `cd backend && go test ./...`
- Run frontend typecheck with: `cd frontend && pnpm typecheck`
- Run integration tests with: `cd backend && go test -tags=integration ./...`
- Make changes via `edit`/`write` tools only
- No manual container management