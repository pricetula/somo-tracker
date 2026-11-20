# Somotracker — Structural Baseline

**Purpose:** Verified facts extracted from source code before all `AGENTS.md` files and `docs/` were deleted (branch `rework`). This is a scratchpad, not a governing spec. No assumptions, no invented rules.

## Monorepo layout

```
/somo-tracker
├── backend/               Go, Fiber v3, pgx, sqlc, PostgreSQL, Redis, Asynq, Stytch
├── frontend/              Next.js 16.4.6, React 19.3.1, App Router, Tailwind + shadcn/ui
├── marketing/             Astro static site
├── docker-compose.yml     Postgres 18, Redis 7, backend
├── Makefile
└── .env                   (Doppler-injected secrets at runtime, no .env committed)
```

## Backend structure (`backend/`)

```
backend/
├── cmd/api/main.go                    # entry point
├── internal/
│   ├── app/
│   │   └── app.go                     # composition root, manual DI, route registration, middleware chain
│   ├── identity/                      # auth domain (sessions, magic link, onboarding tickets)
│   │   ├── handler/handler.go         # auth + onboarding HTTP handlers
│   │   ├── middleware/auth.go         # auth middleware
│   │   ├── service/service.go         # business logic (Stytch, sessions, onboarding state)
│   │   ├── store/store.go             # DB access via sqlc-generated queries
│   │   └── store/queries/*.sql        # raw SQL compiled by sqlc
│   ├── schools/                       # school domain (handler exists, NOT registered in app.go)
│   └── students/                      # student domain (handler exists, NOT registered in app.go)
│   └── platform/                      # infrastructure only
│       ├── apperr/                    # typed errors
│       ├── authz/                     # permissions, Require(perm), ScopeFor()
│       ├── config/                    # typed env config
│       ├── db/                        # pgx pool, TxManager, RLS GUC
│       ├── httpserver/                # Fiber server, rate limiting, access log
│       ├── logger/                    # slog JSON redaction
│       ├── migrator/                  # goose migration wrapper
│       ├── redis/
│       ├── seed/
│       ├── stytch/
│       └── audit/
├── migrations/                        # goose SQL (01–06)
└── sqlc.yaml
```

### Backend route registration (current `app.go`)

Only `/api/v1/auth/*` and `/api/v1/onboarding/*` are registered. Protected and mutation route groups exist but are empty TODOs. Schools/students handlers exist but are **not wired**.

## Frontend structure (`frontend/`)

```
frontend/
├── src/
│   ├── app/
│   │   ├── login/page.tsx             # public
│   │   ├── auth/callback/page.tsx     # public, Stytch redirect (GET)
│   │   ├── (authorized)/dashboard/page.tsx
│   │   ├── (authorized)/students/page.tsx
│   │   └── api/
│   │       ├── auth/request-magic-link/route.ts   # BFF proxy → Go
│   │       ├── auth/authenticate-token/route.ts   # BFF proxy → Go
│   │       └── auth/logout/route.ts               # BFF proxy → Go
│   └── lib/go-client.ts               # forwards cookies to Go
```

### Cookie / BFF architecture

Browser → Next.js API routes → Go backend. Cookies `__Host-session_token` (prod) / `session_token_<env>` (dev), HttpOnly, 30 days.

## Database migrations (`backend/migrations/`)

| Migration | Tables |
|-----------|--------|
| `00000000000001_extensions.sql` | pgcrypto (uuidv7) |
| `00000000000002_auth_domain.sql` | `pending_onboarding`, `app_session`, `onboarding_ticket` |
| `00000000000003_core_domain.sql` | `countries`, `education_systems`, `class_levels`, `tenants`, `users`, `schools`, `memberships` |
| `00000000000004_student_domain.sql` | `students`, `guardian_student_links` |
| `00000000000005_academic_domain.sql` | `subjects`, `topics`, `subtopics`, `streams`, `classrooms`, `academic_years`, `terms`, `enrollments` |
| `00000000000006_rls_and_security.sql` | RLS policies, `updated_at` triggers, soft delete `deleted_at` |

## Auth & onboarding flow — current state

| Step | Endpoint | Status |
|------|----------|--------|
| 1 | `POST /api/v1/auth/request-magic-link` | ✅ Implemented |
| 2 | `POST /api/v1/auth/authenticate-token` | ✅ Implemented |
| 3 | `GET /api/v1/onboarding/current` | ✅ Implemented |
| 4 | `PUT /api/v1/onboarding/current` | ✅ Implemented |
| 5 | `POST /api/v1/onboarding/complete` | ❌ **Missing** — no handler, no service method, no route registered |
| 6 | `POST /api/v1/auth/logout` | ✅ Implemented |

### Critical gap

Steps 1–4 work. Step 5 (tenant/school/membership provisioning) does **not exist** in code. `pending_onboarding` can never reach `status = completed`, so new users cannot finish onboarding. `ErrProvisioningFailed` is defined in `service.go` but never used.

## Roles

`OWNER`, `ADMIN`, `HEADTEACHER`, `TEACHER`, `GUARDIAN`, `FINANCE`, `PLATFORMOPERATOR` (enum `user_role` in migration 03).

## Swagger / API contract

- `@swaggo` annotations exist in backend handler code (e.g., `handler.go`).
- Root `docs/api/openapi.yaml` was deleted along with `docs/`.
- `backend/docs/` is a separate directory and may still contain generated `swagger.*` artifacts.
- Frontend type generation from OpenAPI contract needs to be re-established (deferred discussion).
