# Somotracker — Docs Agent Contract

## Commit Policy: Allow Commits, No Push to Protected Branches

The AI agent may run `git add` and `git commit` with clear conventional messages.

Never push to `main` or `develop`. All changes must go through PR for human review.

---

## Docs-First Workflow

### Linting & Validation
- **Codegen first, then docs**: `make generate` generates swagger.json; `pnpm generate:api` creates `frontend/src/lib/api/generated.ts`
- **Docs audit**: `pnpm docs:build` runs `pnpm audit:docs` and validates doc links, frontmatter, and includes.
- **markdownlint**: Validate `*.md` and `*.mdx` in `docs/` folder (respects nolint comments)
- **Link validation**: Ensure all docs/ references are relative (`docs/`) and point to existing targets

### Reference Updates
- When any backend schema changes, update `docs/reference/backend_schema.md`
- Every API endpoint change must be reflected in:
  1. `docs/openapi.yaml` (contract) <--- NOT backend/docs/swagger.json
  2. `backend/docs/swagger.json` (derived via `swag init`)
  3. `frontend/src/lib/api/generated.ts` (types)
- Sync is triggered by CI: a broken link or missing doc causes `pnpm docs:build` to fail

### Feature Specs
- All feature work starts with a `docs/features/FEATURE-*.md` from template
- Before any coding touches an API or UI component, create/modify the appropriate `*.mdx` file next to it
- Example: `frontend/src/features/students/components/student-detail.mdx` explains the intended UI and design
- PRs must reference the related FEATURE- and ADR- files

### Architecture Decision Records (ADRs)
- Create ADRs in `docs/adr/` using `docs/templates/adr.md`
- Every architectural change (e.g., new API pattern, design system, background workers) has an ADR
- ADRs are consulted by all layers (backend, frontend, QA)

### Gap-Driven Backlog
- Missing functionality lives in `docs/gaps/` as GAP- files
- Planning docs in `docs/planning/` link related gaps and features
- Docs teams track progress: `docs/features/FEATURE-*.md` status evolves: Draft → Proposed → Accepted → In Progress → Done

## Principles
- Document first, code second (this is the primary workflow)
- Every error must be returned up the call stack with context added OR logged and acted upon. Never both, never neither.
- No dumb back buttons in app pages (see root AGENTS.md)

## Documentation Structure
- `docs/` is self-explanatory by name alone (see `docs/CONTRIBUTING.md`)
- `docs/reference/` contains immutable, non‑changing reference information
- `docs/features/` holds formal feature specifications (implementation targets)
- `docs/gaps/` holds known missing / broken work
- `docs/adr/` holds architecture decision records
- `docs/templates/` defines gap, feature-spec, and adr templates
- `docs/planning/` holds pre-spec analysis, roadmaps, checklists
- `docs/seed/` is for static data and sample files, not documentation

## Sync Automation
- CI runs `pnpm docs:build`
- `docs:build` enforces:
  1. Generated files are up‑to‑date (type generation matches spec)
  2. All docs/ path references exist
  3. Template adherence (naming, required frontmatter)
  4. Feature spec completeness (API contracts, acceptance criteria)
- On mismatch, CI fails and requires human review of `docs/` changes

## Documentation Build

Currently building docs involves:

1. **Static site generation**: (to be implemented) - converts MDX to static HTML for docs portal
2. **API contract sync**: Ensure `docs/openapi.yaml` is in sync with runtime API
3. **Accessibility audit**: Run axe-core on generated documentation (helps reviewers catch missing alt-text)
4. **Cross-reference validation**: Ensure all `[[link]]` and `[[ref]]` syntax (if used) resolves correctly

## Testing Docs

For the AI agent:

- `pnpm docs:build` should succeed before any PR targeting code changes.
- If you modify an API endpoint, update `docs/openapi.yaml` and re‑run generation.
- If you tweak a UI component, update the adjacent `.mdx` file.
- For documentation improvements, ensure `pnpm docs:build` still passes (no link breaks).

## Docs-First Implementation Workflow

### When adding a new feature

1. **Gap** → `docs/gaps/GAP-xxx.md` (if this is the first mention)
2. **Planning** → `docs/planning/` analysis (size, complexity, risk)
3. **ADR** (if architecture changes) → `docs/adr/` using template
4. **Feature spec** → `docs/features/FEATURE-xxx.md` from `feature-spec.md` template
5. **API contract** → `docs/openapi.yaml` (reference in feature spec)
6. **Component docs** → `frontend/src/features/*/components/*.mdx` alongside UI code
7. **Implementation** → code following spec (auto‑generated swagger types, docs audit)

### When modifying an existing feature

- Update `docs/features/FEATURE-*.md` with a `Changelog` section
- If API contract changes, update `docs/openapi.yaml`
- Update adjacent `.mdx` files if UI behavior changes
- Run `pnpm docs:build` and fix any errors before submitting PR

### Breaking changes

- Create NEW `FEATURE-*.md` (even if similar to old)
- Ensure `docs/openapi.yaml` defines the new contract
- Update ADRs if this changes architecture
- Break/fix paths in `docs/` only (don't delete history)

## Continuous Improvement

**DO**
- Use relative docs links: `docs/features/FEATURE-xxx.md` (always start with `docs/`)
- Reference dependencies in frontmatter: `depends_on: []`
- Update `docs/reference/backend_schema.md` after schema migrations

**DO NOT**
- Place any data files in `docs/` except `docs/seed/` (use root `frontend/data/`, `backend/db/`, etc.)
- Hardcode descriptive text inside UI markup or labels (use `<FeatureHelp>` with `content/docs/*.mdx` frontmatter)
- Directly edit `docs/reference/backend_schema.md` without using db migration scripts

**Audit the docs** before each completion:

```bash
pnpm docs:build
pnpm docs:lint
```

The docs must compile and pass all validation before a PR targeting code changes is accepted.

---

### Version & ownership
- **Standard version:** 1.0.0 (June 2026
- **Owner:** Platform team. Any changes to this standard must be reviewed by the platform team and propagated to all AGENTS.md files.
