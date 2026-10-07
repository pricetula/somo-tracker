# Backend Improvements Tracker

Generated from audit against `backend/AGENTS.md`, `internal/api/router.go`, `errors.go`, and handler review.

---

## 1. Canonical Error Contract Enforcement

**Contract:** Every non-2xx response MUST use `WriteError(c, APIError)` returning exact JSON:
```json
{ "code": "snake_case", "message": "...", "errors": { "field": ["msg"] }, "request_id": "..." }
```

**Violations:**
- `timetable_handler.go` — inline `c.Status(...).JSON(fiber.Map{...})` (no `request_id`, wrong `Errors` type)
- `streams_handler.go`, `classes_handler.go`, `grades_handler.go`, `rooms_handler.go`, `enrollments_handler.go`, `attendance_handler.go`, `subjects_handler.go`, `topics_handler.go`, `sub_topics_handler.go`, `subjects_detail_handler.go`, `topics_detail_handler.go`, `admin_invitation_handler.go`, `finance_handler.go`, `guardians_handler.go`, `teachers_handler.go`, `admins_handler.go`, `finance_invitation_handler.go`, `guardian_invitation_handler.go`, `teacher_invitation_handler.go`, `students_handler.go`, `students_import_handler.go`, `school_handler.go`, `school_create_handler.go`, `auth_handler.go`, `me_handler.go` — likely similar

**Action:** Replace all inline JSON errors with `WriteError(c, ErrBadRequest/ErrNotFound/ErrInternal/ErrUnauthorized/ErrForbidden)`.

---

## 2. Handler/Service Separation — `events_handler.go` uses `pgxpool` directly

**Contract:** Handlers must be thin transport adapters. No `pgxpool`, no raw SQL, no `sqlc` calls in `internal/api/*.go`. All DB access via `internal/services` using `*sqlc.Queries`.

**Violation:** `events_handler.go` injects `*pgxpool.Pool`, runs `pool.Query/QueryRow/Exec` directly. No `events_service.go` exists.

**Action:**
1. Create `internal/services/events_service.go` wrapping `*sqlc.Queries`
2. Move all SQL to service methods
3. Update `EventsHandler` to depend on `services.EventsService` interface
4. Update `router.go` `NewRouter` signature and wiring

---

## 3. Structured Input Validation

**Gap:** No `validator` dependency. Manual `if req.Field == ""` checks scattered.

**Action:**
1. Add `github.com/go-playground/validator/v10` to `go.mod`
2. Tag request structs with `validate:"required,email,max=255"` etc.
3. Write `ValidateRequest(c, &req) APIError` helper returning `ErrBadRequest` with per-field `errors` map
4. Apply to all handlers

---

## 4. Migration Integration Test Coverage

**Contract (`AGENTS.md`):** Every new migration SQL file MUST have `TestMigrator_<FeatureName>` in `internal/database/migrator_integration_test.go`.

**Action:** Audit `db/migrations/`; for each migration without a test, add `func TestMigrator_<Name>(t *testing.T)` using `testdb.DB(t)` with `information_schema` assertions only.

---

## 5. Router Initialization — Nil Handler Risk

**Issue:** `NewRouter` sets `AdminInvitation`, `TeacherInvitation`, `FinanceInvitation`, `GuardianInvitation` to `nil`. They get session stores only in `RegisterRoutes`. If routes hit before full init → nil dereference.

**Action:** Initialize invitation handlers fully in `NewRouter` (pass required deps), or add nil guards in `RegisterRoutes`.

---

## 6. Protected Group Rate Limiting

**Gap:** Only auth endpoints have `redis_rate` tiers (`authRateIP`, `bulkInviteRate`). Protected `/api` routes have no global rate limit.

**Action:** Add default protected-group limiter (e.g., 120/min per session/user) in `RegisterRoutes` before the protected group.

---

## 7. Term / Year Resolution — Strict Backend Resolution

**Contract (`AGENTS.md`):** "Never expose `term_id` or `academic_year_id` as query/body parameters to clients. Never pass term/year IDs from the frontend; resolution belongs to the API layer."

**Action:** Audit all endpoints referencing academic periods; ensure resolution uses `GetCurrentAcademicTermBySchool` / `GetLatestAcademicTermBySchool` inside service/handler, not in request params.

---

## 8. Swagger / Endpoint URI Verification

**Contract:** Before writing `swaggo` annotations, verify full URI = router group prefix + route string. Verify frontend `lib/api/*.ts` matches. After `swag init`, confirm `swagger.json` path matches.

**Gap:** `subjectsListHandler`, `topicsListHandler`, `subTopicsListHandler` are closures using `curriculumSvc` with no swagger annotations verified.

**Action:** For each, confirm full URI in `router.go`, verify frontend client, add annotations, regenerate.

---

## 9. Logging / Observability Discipline

**Contract:** Every error returned up the call stack with context added, OR logged and acted upon. Never both. Never neither. No empty catch, no log-and-return, no silent `_ =`.

**Violations:**
- Some handlers log error then return it (duplicate)
- Some discard errors with `_ = someFunc()`
- Some use `h.logger.Error` but others don't log at all

**Action:** Enforce request-scoped `zap.Logger`; log once at handler layer; intermediate layers only wrap and return; never `_ =` in non-test code.

---

## Quick Verification Commands

```bash
# Files not using WriteError
grep -L 'WriteError' backend/internal/api/*.go

# Handlers importing pgxpool (should be 0)
grep 'pgxpool' backend/internal/api/*.go

# Handlers with raw SQL
grep -n 'pool\.\|Queries\.' backend/internal/api/*.go

# Empty catch / silent discard
grep -n '_ =' backend/internal/api/*.go
grep -n 'catch.*{}' backend/internal/api/*.go
```

---

## Execution Order (suggested)

1. **Error contract** — fastest win, touches all handlers
2. **Events service extraction** — architectural fix
3. **Input validation** — improves all future handlers
4. **Migration tests** — compliance
5. **Router nil guards** — stability
6. **Rate limiting** — security
7. **Term resolution audit** — contract compliance
8. **Swagger verification** — API consistency
9. **Logging discipline** — observability

---

*Last updated: 2025-10-07*