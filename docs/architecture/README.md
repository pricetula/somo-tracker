# Phase 3 — System Architecture

Components, boundaries, integrations, and decisions. Verified sources: `docker-compose.yml`, `backend/internal/app/app.go`, backend/frontend/marketing `AGENTS.md` files, `docs/api/openapi.yaml`, `Makefile`.

## Status

draft — bootstrapped from code. No human sign-off yet.

## Components

### Monorepo applications

| Application | Stack | Role |
|-------------|-------|------|
| `frontend/` | TanStack Start + Tailwind + shadcn/ui + Base UI | Web application (TanStack Router, React Query) |
| `backend/` | Go, Fiber v3, pgx, sqlc, PostgreSQL, Redis, Asynq | HTTP API + background worker |
| `marketing/` | Astro static site | Public landing site; not part of `docker-compose.yml` |

Source: root `AGENTS.md`, per-app `AGENTS.md` files, `docker-compose.yml`.

### Backend layout (`BE-001`)

```
cmd/<binary>/        — entry points (api, worker)
internal/app/        — composition root; manual DI; wires handler/service/store
internal/platform/<capability>/ — infrastructure, no business logic
internal/<domain>/{handler,service,store}/ — one domain per business capability
```

Dependency direction (`BE-002`): `handler` → `service`; `store` → `service`; `internal/app` wires all three. `service` never imports `store`, `handler`, sqlc, pgx, or Fiber. Domains never import each other; cross-domain needs are declared as small interfaces in the consumer domain and wired in `app`.

Forbidden: `utils`, `helpers`, `common`, `misc` packages; DI frameworks, globals, `init()` side effects, service locators (`BE-002`, `BE-005`).

### Identity domain (implemented)

`internal/identity/{handler,service,store/}` + `internal/identity/middleware/`. Handlers: magic-link request, token authenticate, logout, onboarding get/update/complete. Service: session + membership resolution, invitation/session logic. Store: sqlc-generated queries for `app_session`, `onboarding_ticket`, `pending_onboarding`, `pkce_codes`, `core_domain` tables.

Middleware: `RequireAuth`, `TenantMiddleware`, RBAC middleware, CSRF middleware.

**Authorization:** `platform/authz` defines permissions and the `Require(perm)` / `ScopeFor()` primitives required by BE-020/BE-021. The permission matrix is data in `platform/authz` and must be implemented (it is not a placeholder to be left unimplemented). No protected domain handlers are registered yet (see route groups below).


### Platform packages

`apperr` (typed errors), `authz` (permissions; matrix data to be implemented), `config` (typed env config), `db` (pgx pool), `httpserver` (Fiber server, rate-limit middleware, security headers, access log), `logger` (slog JSON, redaction), `migrator` (goose wrapper), `ratelimit` (Redis with in-memory fallback), `redis`, `stytch` (SDK client adapter).

### Frontend architecture (`frontend/AGENTS.md`)

- Feature modules under `src/features/<name>/`; import only via `features/<name>/index.ts` outside the feature (`FE-001`).
- Routes in `routes/` export TanStack Router `Route` with `loader` and `component`; pages compose features via `index.ts`, hold no business logic or data access (`FE-002`).
- API types generated from `docs/api/openapi.yaml` into `src/features/auth/api/generated/types.ts`; generated wire types are snake_case; map to camelCase at the API-function layer (`FE-008`).
- HTTP via native `fetch` wrapped once in `src/lib/api/client.ts`; base path `/api/v1`; never send school ID or `X-Active-School` (`FE-007`).
- App shell: shadcn `sidebar-07`; header has `SidebarTrigger`, `Separator`, dynamic `Breadcrumb`; page title and primary actions in shared `<PageHeader>` (`FE-013`).
- Add/edit/view use intercepting + parallel routes in `Dialog`/`Sheet` (`FE-014`).

## Boundaries

### API contract boundary

Backend owns `docs/api/openapi.yaml`. Frontend regenerates types with `pnpm generate:api` after any API contract change; CI regenerates and diffs (`make generate-check`). Frontend consumes generated types only; no generated client SDK (`FE-008`).

### Tenant context boundary

Backend owns tenant and active-school context; frontend never sends a school identifier (`root AGENTS.md` security principles, `BE-017`, `FE-011`).

### Swagger vs OpenAPI

`swag` generates `backend/docs/swagger.*` from `@swaggo` annotations in handlers — that artifact is not the frontend type source (`BE-023`).

### Route groups (from `app.go`)

- Public routes: `/api/v1` + public rate limit; identity handler routes (auth, onboarding). These are the only registered routes.
- Protected routes: `/api/v1` + `RequireAuth` + `TenantMiddleware` + RLS GUC + protected rate limit. **No protected domain handlers registered yet** — deferred to a later specification (the group exists in `app.go` as a TODO).
- Mutation routes: protected + CSRF middleware + mutation rate limit. **No mutation handlers registered yet** — deferred to a later specification (the group exists in `app.go` as a TODO).
- Health: `GET /healthz`, `GET /readyz` at app root, no auth/rate limit. Per `BE-026`, health/metrics/spec must never be served under `/api`.

## Integrations

| Integration | Direction | Details |
|-------------|-----------|---------|
| PostgreSQL 18 | Backend → DB | `POSTGRES_DSN`; app role non-owner; FORCE RLS; goose migrations |
| Redis 7 | Backend → cache/queue | rate limits, short-lived tokens, cache, asynq queue; `noeviction` |
| Stytch | Backend → SaaS | magic links, organizations, members; only at login/logout; via `identity/service` interface + `platform/stytch` adapter |
| Doppler | Secrets | `doppler run -- ...`; secrets never in repo, no `.env` files (`BE-033`, `README.md`) |
| Email provider | Backend → SaaS | hourly/daily caps (`EMAIL_PROVIDER_HOURLY_CAP`, `DAILY_CAP`) |
| Sentry (EU) | Backend → SaaS | panics and 5xx only; `SendDefaultPII` off; scrub bodies/headers/cookies/query/user data (`BE-032`) |
| Vite proxy | Frontend → Backend | `BACKEND_URL`/`PUBLIC_API_URL`; trusted forwarded client IP hop only (`BE-026`) |

## Decisions

| Decision | Source | Status |
|----------|--------|--------|
| Keyset pagination (limit/cursor/sort); no OFFSET, no total count | `BE-014`, `FE-009` | implemented in rules; identity endpoints only |
| uuidv7 defaults, never gen_random_uuid | `BE-038` | implemented in migrations |
| Soft delete via deleted_at + partial index | `BE-041` | implemented in migrations |
| RLS + FORCE RLS on all tables; GUC `app.current_tenant_id` | `BE-016`, `BE-040` | implemented in migration 06 |
| Manual DI in `internal/app` | `BE-005` | implemented (`app.go`) |
| One transaction per service call via `TxManager.Run` | `BE-012` | rule stated |
| API v1 unreleased; breaking changes allowed | `BE-022` | current state (version 0.1.0) |
| Cookie session `__Secure-session_token`, Domain=domain.com | `BE-019` | rule stated |
| Academic hierarchy: subject → topic → sub-topic | migrations `00000000000005` | implemented in schema |
| Authorization permission matrix | `BE-020`, `BE-021` | to be implemented via specification; not a placeholder |
| Protected / mutation route handlers | `app.go` TODOs | deferred to later specifications |
| Guardian in MVP scope | product decision | in scope |

### Recorded gaps (decisions pending)

- `platform/authz` permission matrix: primitives exist but the matrix data and RBAC wiring are not implemented. Must be implemented via specification (BE-020/BE-021) before any protected-domain feature is built.
- `app.go` has TODO comments for protected domain handlers, finance routes with RBAC, and asynq worker server (`<-ctx.Done()` stub). No worker handlers implemented yet. Deferred to later specifications.
- Migrations run on app startup via `migrator.Up`; note in `app.go` says production should run migrations as a separate deploy step with a privileged role while the app connects as a non-owner role.

## Architecture decision records

Directory `docs/architecture/adr/` is empty. ADRs should be added when an architectural decision is made or reversed (link from here; one fact in one place).
