# Somotracker — Root Agent Contract

## Monorepo Structure

```
├── .github/      # CI/CD workflows
├── backend/      # Go (Fiber) REST API
├── docs/         # Markdown templates and feature specs
├── frontend/     # Next.js (App Router) — multi-tenant educational dashboard
└── public/       # SvelteKit + shadcn-svelte — marketing website
```

**Strictly isolate all changes to their respective top-level directory.** Read that directory's `AGENTS.md` for rules before making any changes.

---

## Error Handling

### Core rule
Every error must be **returned up the call stack with context added**, OR **logged and acted upon**. Never both. Never neither.

### Canonical error response contract
Every non-2xx HTTP response from the backend MUST return this exact JSON body:

```json
{
  "code":    "snake_case_error_code",
  "message": "human readable message",
  "errors":  { "field_name": ["Specific field validation message"] }
}
```

- Backend reference: `internal/middleware/errors.go`
- Frontend reference: `src/lib/api/client.ts`

### Three universal forbidden patterns
1. **Empty catch** — `catch (e) {}` or `if err != nil { }` — never silently drop an error.
2. **Log-and-return** — logging an error and then also returning it up the stack duplicates the event. Log once at the handler/worker layer. Intermediate layers only wrap and return.
3. **Silent `_`** — `_ = someFunc()` discards the error without action. In non-test code this is forbidden.

### Layer-specific rules
- **Backend:** See `backend/AGENTS.md` — Error Handling section.
- **Frontend:** See `frontend/AGENTS.md` — Error Handling section.

### Version & ownership
- **Standard version:** 1.0.0 (June 2026)
- **Owner:** Platform team. Any changes to this standard must be reviewed by the platform team and propagated to both AGENTS.md files.

### Isolation Rule
> **Strictly edit only the AGENTS.md located in the top‑level directory that corresponds to the layer you are working on.**
> • `backend/AGENTS.md` – backend contracts only.
> • `frontend/AGENTS.md` – frontend contracts only.
> • `public/AGENTS.md` – marketing site contracts only.
> Do **not** modify other top‑level folders’ contracts.

### Quick‑Start Commands (project‑wide)

| Layer      | Command                     | Outcome                              |
|------------|----------------------------|--------------------------------------|
| Backend    | `make build-backend`        | Compiles Go binaries in `./backend` |
| Frontend   | `npm run build:frontend`    | Builds Next.js production bundle     |
| Docs       | `npm run docs:build`        | Generates MDX docs under `content/docs` |
| All Tests  | `npm run test:all`          | Runs Go interop + Jest + Playwright |

---

### Version & ownership
- **Standard version:** 1.0.0 (June 2026
- **Owner:** Platform team. Any changes to this standard must be reviewed by the platform team and propagated to both AGENTS.md files.
