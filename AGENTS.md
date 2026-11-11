# Somotracker — Root Agent Contract

**Project:** School management system for East African schools.
**Repository:** Monorepo with three applications: `frontend/` (TanStack Start), `backend/` (Go Fiber v3), `marketing/` (Astro static site).

Before any implementation work, determine whether Knowledge Bootstrap Mode is required. If the project's foundational documentation is missing or insufficient, prioritize establishing the knowledge baseline before proceeding with feature development.

## Document-led workflow

1. Human writes/approves a specification in `docs/`.
2. Agents implement only specifications with `status: approved`. Do not implement draft or in-progress specs.
3. Code follows documentation; never the reverse. Do not silently redefine documented behavior.
4. Verify changes with `make compliance` and resolve review feedback before considering work complete.
5. Status moves `implemented → verified by human`.

## Read before acting

| Task | Read first |
|------|------------|
| New feature | Root AGENTS.md (this file), the relevant application AGENTS.md, `docs/api/openapi.yaml` (if API-related) |
| New API | `docs/api/openapi.yaml`, `backend/AGENTS.md` |
| New UI | `frontend/AGENTS.md` |
| Domain change | `docs/domain/` (if it exists) |
| Architecture | `docs/adr/` (if it exists) |
| Docs update | `docs/README.md` (if it exists) |

> Note: Documentation is currently being reworked. Many `docs/` subdirectories referenced in older records do not yet exist. If a referenced file or directory is missing, treat it as an open question and ask the human before proceeding. Do not invent specifications or create new architecture. Link to specifications; do not copy them.

## Commit policy

- Agents may run `git add` and `git commit` locally using conventional commits (e.g., `feat:`, `fix:`, `docs:`, `chore:`).
- Never push to `main` or `develop`. Pull requests target `develop`.
- All commits must pass `make compliance` and the lefthook pre-commit checks.

## Monorepo rules (apply to all applications)

- **One fact lives in one place.** Link to specifications; do not copy their contents into AGENTS.md or source code.
- Change only what the task requires. Do not refactor unrelated code.
- Add no new dependency without explicit approval. Prefer dependencies already installed.
- Reference FR/NFR IDs in PRs, commits, and tests where requirements exist. Do not build Phase 2 / Later features during MVP work.
- When a specification is missing, incomplete, or contradictory, raise an Open Question rather than assuming. Document the unresolved decision.
- Agents must not push directly to `main` or `develop`. PRs target `develop`.

## Application boundaries

- `frontend/` — TanStack Start + Tailwind + shadcn/ui + Base UI. See `frontend/AGENTS.md`.
- `backend/` — Go (Fiber v3, pgx, sqlc, PostgreSQL, Redis, Asynq). See `backend/AGENTS.md`.
- `marketing/` — Astro static site. See `marketing/AGENTS.md`. Not part of `docker-compose.yml`.

## Cross-application changes

API contract changes are coordinated through `docs/api/openapi.yaml` (the source of truth, backend-owned):
1. Backend updates `docs/api/openapi.yaml` and its handler/DTOs.
2. Backend runs `make generate-swag` to regenerate `backend/docs/swagger.*`.
3. Frontend runs `pnpm generate:api` (from `frontend/`) to regenerate `src/features/auth/api/generated/types.ts`.
4. Verify no generated-file drift with `make generate-check`.

Database schema and migration changes:
1. Migrations live in `backend/migrations/` and are managed with goose (`make db-migrate-create name=xxx`, `make db-migrate-up`, `make db-migrate-down`).
2. Every new migration must be verified with `migrator_integration_test.go` (see `backend/AGENTS.md` BE-037).
3. `sqlc generate` regenerates `backend/internal/*/store/gen/`.

## Commands

| Purpose | Command | Where |
|---------|---------|-------|
| Full compliance | `make compliance` | repo root |
| Backend vet | `cd backend && go vet ./...` | repo root |
| Backend lint | `cd backend && golangci-lint run ./...` | repo root |
| Backend tests | `cd backend && go test -short -v ./...` | repo root |
| Frontend lint | `cd frontend && pnpm lint` | repo root |
| Frontend typecheck | `cd frontend && pnpm typecheck` | repo root |
| Frontend tests | `cd frontend && pnpm test` | repo root |
| Frontend generate API types | `cd frontend && pnpm generate:api` | repo root |
| Backend generate sqlc | `cd backend && sqlc generate` | repo root |
| Backend generate swag | `cd backend && swag init -g cmd/api/main.go -o docs` | repo root |
| Generate all | `make generate` | repo root |
| Generate check (no drift) | `make generate-check` | repo root |
| Marketing lint | `cd marketing && pnpm lint` | repo root |
| Marketing typecheck | `cd marketing && pnpm typecheck` | repo root |

## Security and domain principles (preserved)

- Document-led development; only approved specifications authorize implementation.
- Backend owns tenant and active-school context; frontend never sends a school identifier or `X-Active-School` header.
- Database-enforced invariants and tenant isolation (RLS); every query is tenant-scoped.
- Generated API contracts and generated code ownership — never hand-edit generated files.
- Existing security, privacy, authorization, and testing requirements are preserved; do not weaken them.
- Kenyan school-management domain terminology (CBC: learning area, strand, sub-strand; KES money in minor units).


Knowledge-First Development
---------------------------

### 1\. Documentation as the Source of Truth

The docs/ directory is the central knowledge base for Somotracker.

All agents MUST use and maintain it throughout development.

Before starting a task:

*   Read the applicable AGENTS.md files.
    
*   Inspect docs/ and identify relevant documentation.
    
*   Understand existing requirements, domain rules and architectural decisions.
    

### 2\. Documentation Bootstrap

If docs/ is empty or incomplete, agents MUST establish the documentation foundation before implementing new features.

The bootstrap process MUST:

1.  Inspect the existing repository, including frontend, backend, database, API contracts and infrastructure.
    
2.  Identify existing product capabilities, domain concepts, business rules and architecture.
    
3.  Organize findings into the agreed documentation structure.
    
4.  Distinguish verified facts, inferences, assumptions and unresolved questions.
    
5.  Document gaps, contradictions and decisions requiring human input.
    
6.  Create a docs/README.md explaining the documentation structure and navigation.
    

Do not invent business rules, requirements or architectural decisions.

Existing code describes current implementation; it does not automatically establish intended business behaviour.

### 3\. Feature Specification Workflow

Before implementing a new feature or substantial change, agents MUST:

1.  Understand the requested outcome.
    
2.  Inspect relevant existing documentation and implementation.
    
3.  Identify dependencies, missing information and conflicts.
    
4.  Ask necessary clarification questions.
    
5.  Define functional and non-functional requirements.
    
6.  Create or update the relevant feature specifications.
    
7.  Define acceptance criteria and verification requirements.
    
8.  Present the specifications for human review.
    

Only explicitly approved specifications authorize implementation.

Agents MUST NOT approve their own specifications.

### 4\. Knowledge Maintenance

Documentation MUST evolve alongside approved implementation changes.

Agents MUST update affected specifications, architecture decisions, data models and API documentation when necessary.

Maintain traceability between requirements, specifications, domain rules and implementation.

Avoid duplicate documentation. Every important concept or business rule MUST have one authoritative definition.

### 5\. AI Agent Context Management

Agents MUST use progressive context loading:

*   Start with repository instructions and docs/README.md.
    
*   Read only documentation relevant to the current task.
    
*   Follow references to related authoritative documents.
    
*   Never rely exclusively on conversation history for project knowledge.
    
*   Record durable project knowledge in docs/.
    

### 6\. Scope and Restrictions

Documentation bootstrap does not authorize application code changes.

Agents MUST NOT:

*   Implement draft or unapproved specifications.
    
*   Introduce unapproved architectural decisions.
    
*   Change database schemas, migrations or API behaviour without authorization.
    
*   Refactor unrelated application code.
    
*   Present assumptions as verified facts.
    

When documentation conflicts with implementation or another authoritative decision, stop and request human clarification.
