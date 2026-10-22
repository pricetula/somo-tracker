# ADR-001: Docs-First / OpenAPI-First Development Workflow

**Status:** Accepted
**Date:** 2026-10-15
**Deciders:** Platform Team
**Related:** `docs/CONTRIBUTING.md`, `docs/templates/adr.md`, `docs/openapi.yaml`

## Context
This project uses a multi-layer monorepo (Go backend, Next.js frontend, marketing site). Historically, specs were implicit — code was authored, then documentation was written (or skipped). The result was spec drift between backend handlers, frontend clients, and feature specs.

We need a single source of truth that drives implementation rather than following it.

## Decision Drivers
- Multi-team (backend, frontend, QA, product) must agree on contracts before coding
- Backend uses Fiber; frontend uses Next.js App Router; both need shared types
- Feature specs must be reviewable by non-engineers
- Changes must be traceable through PRs without relying on external tools (Confluence, etc.)

## Options Considered
1. **Confluence / Notion first, then code**
   - Pros: Rich UI for product; easy collaboration; no repo access needed
   - Cons: Not version-controlled; links rot; no CI enforcement; hard to keep in sync with API specs
2. **Code-first + generated docs**
   - Pros: Always accurate; minimal overhead
   - Cons: Specs are hidden in annotations; not readable as standalone documents; hard for product/QA to review
3. **Markdown/MDX in repo (this choice)**
   - Pros: Version-controlled; PR-reviewed; can build static sites; integrates with CI; readable by non-developers
   - Cons: Requires discipline; needs lint/build pipeline

## Decision Outcome
We choose **Option 3 (Markdown/MDX in repo)** with these sub-decisions:
- All specs in `docs/features/FEATURE-*.md`
- All ADRs in `docs/adr/ADR-*.md`
- All APIs described in `docs/openapi.yaml`; backend generates `swagger.json` from annotations; frontend generates `src/lib/api/generated.ts`
- Component docs in `.mdx` files next to components (live demo pattern)
- Every PR that touches a layer must reference or update the corresponding doc file

## Consequences
### Positive
- Contract-first: backend and frontend agree before code is written (`openapi.yaml` drives both)
- Non-developers can review specs via static docs (built from `docs/`)
- Changes are traceable via git blame / PR links
- No external dependency for documentation hosting (static site builds from repo)

### Negative
- Requires `docs` script / linting in CI (must enforce, not suggest)
- Slight overhead for every new endpoint / feature (must update spec first)
- Risk of "docs just for show" if CI doesn't enforce updates

## Implementation Notes
- See `docs/CONTRIBUTING.md` for full workflow (Gaps → Planning → Features → Implementation)
- `docs/openapi.yaml` is the canonical contract; `backend/docs/swagger.json` is derived from Go annotations via `swag init`
- Frontend uses `pnpm generate:api` (calls `openapi-typescript`) to generate types from `swagger.json`
- Component docs (`.mdx`) use shadcn components for live interaction examples; see checklist item 3

## References
- `docs/openapi.yaml`
- `backend/docs/swagger.json`
- `frontend/src/lib/api/generated.ts`
- `docs/CONTRIBUTING.md`
- `backend/AGENTS.md` (Swagger / Endpoint URI Verification)
