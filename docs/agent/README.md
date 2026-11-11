# Phase 6 — Agent Knowledge

AGENTS.md lineage, reusable instructions, and context-loading rules for agents working on Somotracker.

## Status

draft — bootstrapped from existing AGENTS.md files.

## Context-loading rules (progressive)

1. **Start** with the root `AGENTS.md` (document-led workflow, commit policy, monorepo rules, security principles).
2. **Read** `docs/README.md` for the documentation structure and phase status.
3. **Read** only the phase documentation relevant to the current task (product, domain, architecture, requirements, specifications).
4. **Follow references** to authoritative documents (`docs/api/openapi.yaml`, migrations, per-app `AGENTS.md`).
5. **Never rely exclusively on conversation history** for project knowledge — record durable knowledge in `docs/`.
6. **Record** new durable knowledge in `docs/` when an approved specification changes behavior, domain model, or API contract.

## AGENTS.md chain

| File | Scope | Location |
|------|-------|----------|
| Root AGENTS.md | Repo-wide: document-led workflow, commit policy, monorepo rules, security principles, compliance commands | `/AGENTS.md` |
| Frontend AGENTS.md | TanStack Start rules (FE-001…FE-027, FE-AUTH-01…03) | `frontend/AGENTS.md` |
| Backend AGENTS.md | Go Fiber rules (BE-001…BE-045) | `backend/AGENTS.md` |
| Marketing AGENTS.md | Astro static site rules | `marketing/AGENTS.md` |

Application-specific rules do **not** override repository-wide requirements. Root AGENTS.md is authoritative for cross-application concerns.

## When to stop and ask

Per root AGENTS.md, stop and request human clarification when:

- Documentation conflicts with implementation or another authoritative decision.
- A specification is missing, incomplete, or contradictory.
- Behavior is unstated — it becomes an Open Question, not an assumption.
- Adding a new dependency, changing database schema/migrations, or changing API behavior.

## Keeping docs current

Documentation must evolve alongside approved implementation changes:

- Update affected specifications, architecture decisions (ADRs under `docs/architecture/adr/`), data models (`docs/domain/`), and API documentation (`docs/api/openapi.yaml`) when behavior changes.
- Distinguish verified facts, inferences, assumptions, and open questions.
- Avoid duplicate documentation: every important concept or business rule has one authoritative definition; link, don't copy.

## Commands relevant to agents

| Purpose | Command | Where |
|---------|---------|-------|
| Full compliance | `make compliance` | repo root |
| Backend vet / lint / tests | `cd backend && go vet ./...` / `golangci-lint run ./...` / `go test -short -v ./...` | repo root |
| Frontend lint / typecheck / tests | `cd frontend && pnpm lint` / `pnpm typecheck` / `pnpm test` | repo root |
| Generate API types | `cd frontend && pnpm generate:api` | repo root |
| Backend sqlc / swag | `cd backend && sqlc generate` / `swag init -g cmd/api/main.go -o docs` | repo root |
| Generate check (no drift) | `make generate-check` | repo root |

## Commit policy (agents)

- Conventional commits (`feat:`, `fix:`, `docs:`, `chore:`); reference FR/NFR IDs where requirements exist.
- Agents may `git add`/`git commit` locally; never push to `main` or `develop`; PRs target `develop`.
- All commits must pass `make compliance` and lefthook pre-commit checks.
