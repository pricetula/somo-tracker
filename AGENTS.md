# Somotracker — Root Agent Contract

> **AI Coding Directive:** Before modifying any file, identify the layer (backend/frontend/public/docs) and load the matching `.pi/skills/`. Use `/skill:project-context` first to understand constraints. Use `/review` before committing work.

## Commit Policy: Allow Commits, No Push to Protected Branches

The AI agent **may** run `git add` and `git commit` with clear, conventional commit messages to create history locally.

**Never** push to protected branches: `main` or `develop`.

- Commits are allowed on feature branches.
- Pushes to `main`/`develop` are forbidden; PRs must be opened for human review.
- All commits must pass `make compliance` and lefthook pre-commit checks.

---

## AI Coding Setup (Late 2026 Edition)

This project is optimized for the **pi coding agent** framework (`.pi/` directory). All AI work must respect these contracts.

### Skills (Procedural Workflows)
Load from `.pi/skills/` when the task matches:
- `craft-docs` — Documentation (MDX creation, tooltip sync)
- `generate-api` — Backend endpoint scaffolding
- `pure-shadcn` — Frontend component generation
- `security-audit` — OWASP API security review (before PR approval)
- `migration-testing` — Migration integration tests
- `error-contract` — Canonical error validation
- `feature-module` — Feature architecture validation
- `agendas-compliance` — Cross-layer contract audit
- `project-context` — Load all contracts + docs for new work

### Bot Factory (Executable Enforcement)
Skills are now backed by CI bots in `.github/workflows/bots.yml`:
- `docs-bot` — `pnpm docs:lint` / audit:docs / link validation
- `api-bot` — `make generate` idempotency, swagger/router match
- `security-bot` — OWASP API Top 10, BOLA/BFLA, error contract
- `migration-bot` — Every migration SQL has `TestMigrator_*` integration test
- `ui-bot` — pure-shadcn rules, no hex colors, no back buttons, `pnpm lint`

Run locally: `make compliance`. PRs require bot green before merge.

### Prompt Templates (Reusable Commands)
- `/review` — Contract review across all layers
- `/skill:<name>` — Force-load any skill

### Project Context Loading Order
Every AI task must load in this sequence:
1. `AGENTS.md` (this file + layer file) — constraints
2. `docs/REVAMP-PLAN.md` — current process improvements
3. `docs/features/` or `docs/gaps/` — spec / known issues
4. `.pi/skills/<matching>` — procedural instructions
5. `docs/CONTRIBUTING.md` — workflow conventions

---

### Monorepo Structure
```
├── .github/      # CI/CD workflows
├── backend/      # Go (Fiber) REST API — Core business logic & analytics engine
├── docs/         # Document-led workflow. See docs/CONTRIBUTING.md
│   ├── reference/   # immutable reference info
│   ├── planning/    # pre-spec analysis, roadmaps, checklists
│   ├── features/    # formal feature specs
│   ├── gaps/        # known missing / broken work
│   ├── templates/   # gap, feature-spec, ADR templates
│   ├── adr/        # architecture decision records
│   └── seed/        # seeding / authoring data
├── frontend/     # Next.js (App Router) — Multi-tenant educational dashboard
└── public/       # Astro + shadcn-svelte — Marketing website
```

**Strictly isolate all changes to their respective top-level directory.** Read that directory's `AGENTS.md` for rules before making any changes.

---

### Error Handling (All Layers)
Every non-2xx HTTP response from the backend MUST return this exact JSON body (see `backend/internal/api/errors.go`):

```json
{
  "code":    "snake_case_error_code",
  "message": "human readable message",
  "errors":  { "field_name": ["Specific field validation message"] }
}
```

Every AI-generated error response must:
- Include a `code` (snake_case) and `message` (human-readable)
- Return `errors` as a non-nil map (empty `{}` if none, never null/undefined)
- Match the frontend contract (`frontend/src/lib/api/client.ts`)
- Be validated by `/skill:error-contract`

### Three Universal Forbidden Patterns (AI Must Check)
1. **Empty catch** — `catch (e) {}` or `if err != nil { }` — never silently drop an error.
2. **Log-and-return** — logging an error and then also returning it up the stack duplicates the event. Log once at handler/worker layer. Intermediate layers only wrap and return.
3. **Silent `_`** — `_ = someFunc()` discards the error without action. In non-test code this is forbidden.

### Version & ownership
- **Standard version:** 1.0.0 (June 2026)
- **Owner:** Platform team. Any changes to this standard must be reviewed by the platform team and propagated to both AGENTS.md files.

### Page Navigation — No Dumb Back Buttons
No `ArrowLeft` / icon-only back buttons, "Back to" links, or breadcrumb navigation inside `src/app/` pages or feature containers (`features/*/components/`). Navigation belongs in the app shell (`layout.tsx`); pages are content panes only. If you need a back path, rely on browser back or global shell navigation — never add a button that says "Back" or uses a left-arrow icon.

### Isolation Rule
> **Strictly edit only the AGENTS.md located in the top-level directory that corresponds to the layer you are working on.**
> • `backend/AGENTS.md` – backend contracts only.
> • `frontend/AGENTS.md` – frontend contracts only.
> • `public/AGENTS.md` – marketing site contracts only.
> Do **not** modify other top-level folders' contracts.

### Quick-Start Commands (project-wide)

| Layer      | Command                     | Outcome                              |
|------------|----------------------------|--------------------------------------|
| Backend    | `make build-backend`        | Compiles Go binaries in `./backend`   |
| Frontend   | `npm run build:frontend`    | Builds Next.js production bundle     |
| Docs       | `pnpm run docs:build`       | Generates MDX docs under `content/docs`|
| All Tests  | `pnpm run test:all`          | Runs Go interop + Jest + Playwright   |

### AI Skill Integration Points
- **New feature:** Load `feature-module` → create docs with `craft-docs` → scaffold backend with `generate-api`
- **API change:** Load `generate-api` → load `security-audit` → validate with `error-contract`
- **Migration:** Load `migration-testing` → write test → run `test-integration`
- **UI update:** Load `pure-shadcn` → use `feature-module` → validate `frontend/AGENTS.md`
- **Review:** Load `agendas-compliance` → run `/review` → check all contracts
- **Doc work:** Load `craft-docs` → use templates → audit with `pnpm run audit:docs`

---

### Version & ownership
- **Standard version:** 1.0.0 (June 2026)
- **Owner:** Platform team. Any changes to this standard must be reviewed by the platform team and propagated to both AGENTS.md files.