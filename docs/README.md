# Somotracker Documentation

Somotracker uses a **spec-driven development (SDD)** process. Documentation is the source of truth; application code implements approved specifications.

> Per root `AGENTS.md`: agents implement only specifications with `status: approved`. Code follows documentation; never the reverse. When documentation conflicts with implementation, stop and request human clarification — do not invent specifications or architecture.

## Documentation phases (in order)

| # | Phase | Directory | Content | Status |
|---|-------|-----------|---------|--------|
| 1 | Product knowledge | `docs/product/` | Vision, users, capabilities, terminology | draft |
| 2 | Domain knowledge | `docs/domain/` | Entities, relationships, business rules, workflows | draft |
| 3 | System architecture | `docs/architecture/` | Components, boundaries, integrations, decisions | draft |
| 4 | Requirements | `docs/requirements/` | Functional and non-functional requirements (FR/BE, FE IDs) | draft |
| 5 | Specifications | `docs/specifications/` | Implementable feature specifications, status-gated | empty |
| 6 | Agent knowledge | `docs/agent/` | AGENTS.md lineage, instructions, context-loading rules | draft |
| — | API contract | `docs/api/openapi.yaml` | Backend-owned OpenAPI 3.1 spec; frontend type source | maintained |

## Document-led workflow (from root AGENTS.md)

1. Human writes/approves a specification in `docs/specifications/` with `status: approved`.
2. Agents implement only approved specifications. Draft / in-progress specs are not implemented.
3. Code follows documentation; never the reverse.
4. Verify changes with `make compliance` and resolve review feedback.
5. Status moves `implemented → verified by human`.

Only explicitly approved specifications authorize implementation. Agents must not approve their own specifications.

## Knowledge bootstrap status

`docs/` previously contained only `api/openapi.yaml`. This documentation set was **bootstrapped from the existing repository**: root and per-app `AGENTS.md` files, backend migrations, `backend/internal/app/app.go`, `backend/internal/platform/authz`, `docker-compose.yml`, `Makefile`, and `docs/api/openapi.yaml`.

Every statement in these docs is classified:

- **Verified fact** — observable in code, migrations, tests, AGENTS.md, or the OpenAPI contract.
- **Inference** — derived from verified facts; marked as such.
- **Assumption** — not yet confirmed; requires human sign-off before dependents are written.
- **Open question** — a contradiction or gap requiring human clarification.

See `docs/agent/context-loading.md` for how agents should load this knowledge.

## Conventions

- Requirement IDs: `BE-001`…`BE-045` (backend), `FE-001`…`FE-027` plus `FE-AUTH-01`…`FE-AUTH-03` (frontend). See `docs/requirements/`.
- **One fact lives in one place.** Link to specifications; do not copy their contents into AGENTS.md or source code.
- Never hand-edit generated files: `backend/docs/swagger.*`, `backend/internal/*/store/gen/`, `frontend/src/features/auth/api/generated/types.ts`.
- API contract changes are coordinated through `docs/api/openapi.yaml` (backend-owned source of truth).
- Do not build Phase 2 / Later features during MVP work. Unstated behavior becomes an Open Question, not an assumption.
