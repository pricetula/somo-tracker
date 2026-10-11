# Architecture Deep Dive & Remediation Plan

**Date:** 2025-10-11  
**Scope:** `backend/`, `frontend/`, `public/`, monorepo contracts  
**Owner:** Platform team

## Executive Summary

The monorepo has a clear, intentional architecture: Go Fiber backend with service layer + sqlc, Next.js App Router frontend with feature-module isolation, SvelteKit marketing site. Core contracts – error response shape, academic period resolver, handler thinness, proxy auth guard – are documented and partially implemented.

The biggest risks are contract drift: error handling is inconsistent and string-based, handlers leak DB access, workers silently drop errors, and the Next.js proxy export is incorrect. These undermine reliability, testability and the “never commit” safety net.

## Architecture Overview

```
somo-tracker/
├─ backend/          Go Fiber REST API
│  ├─ internal/api/  transport adapters, router, middleware
│  ├─ internal/services/ business logic, academic period resolver
│  ├─ internal/worker/ Asynq bulk invitation / import processors
│  └─ internal/database/sqlc/ generated queries
├─ frontend/         Next.js App Router
│  ├─ src/app/ routing only
│  ├─ src/features/ feature modules with components/hooks/services/types/index.ts
│  ├─ src/lib/api/client.ts canonical ApiError client
│  └─ src/proxy.ts auth guard
└─ public/           SvelteKit + shadcn-svelte marketing
```

## Strengths

* Monorepo isolation per AGENTS.md, feature-module boundaries respected in frontend.
* Service layer separation: handlers should be thin, services own business logic and sqlc.
* Academic Period Resolver implements `active | break | expired | none` with strict backend resolution, no term/year IDs exposed to clients.
* Canonical error contract documented in `backend/internal/api/errors.go` and `frontend/src/lib/api/client.ts`; correlation id, global 401 redirect implemented.
* Observability: request id, zap logging, Prometheus metrics, tracing, Fiber error handler.
* Asynq workers for bulk invitations / student import with Redis progress publishing.
* Next.js proxy auth guard is UX-only; backend session + fingerprint is source of truth.
* Migration tests required per `backend/AGENTS.md`.

## Issues Found

### 1. Error handling contract drift
* Services return `fmt.Errorf("bad_request: …")` / `internal_error: …`. Handlers parse with `strings.Contains(err.Error(), "bad_request:")` and strip prefix.  
  Files: `internal/api/rooms_handler.go`, `academic_period_handler.go`, `school_handler.go`, 15+ handlers.
* Some handlers use `WriteError`/`ErrBadRequest`, most use `fiber.NewError`. Error handler always returns empty `errors` map → field validation lost.
* Success responses return `"errors": fiber.Map{}` – inconsistent with error shape.
* Workers silently discard errors: `_ = p.svc.UpdateJobStatus(...)` in `admin/teacher/finance/guardian_invitation_worker.go`.
* Raw rollback error ignored: `defer func(){ _ = tx.Rollback(ctx) }()` in services.

### 2. Handlers not thin
* `SchoolHandler` holds `*pgxpool.Pool` and runs raw SQL: `SELECT stytch_org_id FROM tenants` inside handler. Updates Stytch org directly in handler.
* Session store mutated in handler.
* Violates backend AGENTS.md: handlers must be thin, no pgxpool/sqlc.

### 3. Duplicate code
* Invitation stack duplicated 4x: admin/teacher/finance/guardian workers, services, handlers share >95% logic.
* Generic invitation service uses raw SQL for bulk jobs instead of sqlc.

### 4. Frontend proxy bug
* `frontend/src/proxy.ts` exports named `function proxy`. AGENTS requires `export default function proxy(request: NextRequest, event?: NextFetchEvent)`.
* Next.js 16 proxy adapter will not load, auth guard silently disabled.

### 5. Frontend hygiene
* Hard-coded hex colours `#0050d1` in `features/streams/components/stream-form.tsx` and `features/school/components/create-streams.tsx`. Pure-shadcn rule forbids custom colours.
* Empty `className=""` in `import-orchestrator.tsx` and `school-switcher.tsx`.
* `useEffect` redirect in `features/auth/components/register-form.tsx`.

### 6. Validation stub
* `internal/api/validator.go` is a stub returning nil. Struct validation done ad-hoc.

## Code Smells / Anti-patterns

* String-based error codes → fragile parsing.
* Log-and-return duplication in handlers.
* Silent `_ =` discards.
* Raw SQL in transport layer.
* Copy-paste workers/services.
* Inconsistent response shapes.
* Export shape mismatch for Next.js proxy.

## Remediation Plan

### Phase 0 – Stop the bleeding [DONE]
✅ Proxy export fixed to `export default function proxy(request: NextRequest, event?: NextFetchEvent)`
✅ CI guard `scripts/check_blank_assign.sh` added, no `_ =` discards remain
✅ Silent discards in workers/services replaced with logged errors
✅ Audit of `strings.Contains(err.Error(), "bad_request:")` completed

### Phase 1 – Error handling standardization [DONE]
✅ Service-level typed errors introduced via `services/errors.go` ServiceError with `BadRequest/Forbidden/Internal`
✅ Handlers migrated to `WriteError` and type-assert `*services.ServiceError` – no more string prefix parsing
✅ All 9 flagged handlers now return canonical `{code, message, errors}` shape
✅ Curriculum pagination implemented, removing unused param discards

### Phase 2 – Handler thinness [2-3 days]
1. Remove `*pgxpool.Pool` from `SchoolHandler`. Move Stytch org update and session `UpdateActiveSchool` into `SchoolService`.
2. Refactor all handlers: no imports of `pgxpool`, `sqlc`. Lint rule.
3. Confirm session mutation is service-side only.

### Phase 3 – Deduplicate invitation code [4-6 days] [DONE]
✅ 1. Create generic `BulkInvitationProcessor` with job type param.
   - `backend/internal/worker/bulk_invitation_processor.go` created
   - `cmd/api/main.go` updated to use `NewBulkInvitationProcessor` for admin/teacher/finance/guardian
✅ 2. Consolidate `AdminInvitationService` into `GenericInvitationService` wrapper
   - `backend/internal/services/admin_invitation_service.go` now embeds `*GenericInvitationService` with jobType `ADMIN_INVITATION` / role `ADMIN`
   - Teacher/Finance/Guardian services already embed `GenericInvitationService`
✅ 3. Update Asynq mux registration to use generic processor.
   - `cmd/api/main.go` registers all four invitation queues with `NewBulkInvitationProcessor`
✅ 4. Remove duplicate worker implementations
   - Deleted `admin_invitation_worker.go`, `teacher_invitation_worker.go`, `finance_invitation_worker.go`, `guardian_invitation_worker.go`
   - Shared types `RedisPublisher`, `InvitationBatchPayload`, `InvitationRetryPayload`, `InvitationPayload` moved to `bulk_invitation_processor.go`
✅ 5. Migrate tests – completed
   - `test/integration/admin_invitation_e2e_test.go` updated to use `NewBulkInvitationProcessor`
   - Unit tests updated, integration tests passing with generic processor

### Phase 4 – Validation & hygiene [2 days]
1. Implement `ValidateRequest` with `go-playground/validator`.
2. Add struct tags to request DTOs, use validator in handlers.
3. Remove hard-coded hex colours, replace with CSS variables / semantic tokens.
4. Remove empty `className=""`, replace `useEffect` redirect with server redirect.

### Phase 5 – Observability & tests
1. Add contract tests for error shape and request_id echo.
2. Add migration tests for new generic bulk job schema changes.
3. Add Playwright test for proxy auth guard.

## Definition of Done

* No `strings.Contains(err.Error(), "bad_request:")` remains.
* `internal/api/*` imports no `pgxpool` or `sqlc`.
* Proxy file exports default function with correct signature.
* All 400 responses include field-level `errors` map.
* Invitation code duplication reduced to one generic processor.
* CI passes `pnpm lint`, `go vet`, and contract tests.

## Risks

* Refactoring error handling touches many handlers – coordinate with frontend to avoid breaking `ApiError` parsing.
* Generic invitation refactor must preserve existing job IDs and Redis progress channels.

## Next Steps

### Phase 2 – Handler thinness [2-3 days]
1. Remove `*pgxpool.Pool` from `SchoolHandler`. Move Stytch org update and session `UpdateActiveSchool` into `SchoolService`.
2. Refactor all handlers: no imports of `pgxpool`, `sqlc`. Add lint rule.
3. Confirm session mutation is service-side only.

### Phase 3 – Deduplicate invitation code [4-6 days] [DONE]
✅ 1. Create generic `BulkInvitationProcessor` with job type param.
   - `backend/internal/worker/bulk_invitation_processor.go` created
   - `cmd/api/main.go` updated to use `NewBulkInvitationProcessor` for admin/teacher/finance/guardian
✅ 2. Consolidate `AdminInvitationService` into `GenericInvitationService` wrapper
   - `backend/internal/services/admin_invitation_service.go` now embeds `*GenericInvitationService` with jobType `ADMIN_INVITATION` / role `ADMIN`
   - Teacher/Finance/Guardian services already embed `GenericInvitationService`
✅ 3. Update Asynq mux registration to use generic processor.
   - `cmd/api/main.go` registers all four invitation queues with `NewBulkInvitationProcessor`
✅ 4. Remove duplicate worker implementations
   - Deleted `admin_invitation_worker.go`, `teacher_invitation_worker.go`, `finance_invitation_worker.go`, `guardian_invitation_worker.go`
   - Shared types `RedisPublisher`, `InvitationBatchPayload`, `InvitationRetryPayload`, `InvitationPayload` moved to `bulk_invitation_processor.go`
✅ 5. Migrate tests – completed
   - `test/integration/admin_invitation_e2e_test.go` updated to use `NewBulkInvitationProcessor`
   - Unit tests updated, integration tests passing with generic processor

### Phase 4 – Validation & hygiene [2 days]
1. Implement `ValidateRequest` with `go-playground/validator`.
2. Add struct tags to request DTOs.
3. Remove hard-coded hex colours, replace with CSS variables / semantic tokens.
4. Remove empty `className=""`, replace `useEffect` redirect with server redirect.

### Phase 5 – Observability & tests
1. Add contract tests for error shape and request_id echo.
2. Add migration tests for generic bulk job schema changes.
3. Add Playwright test for proxy auth guard.

---
**References:** `AGENTS.md`, `backend/AGENTS.md`, `frontend/AGENTS.md`, `backend/internal/api/errors.go`, `frontend/src/lib/api/client.ts`, `backend/internal/api/router.go`.
