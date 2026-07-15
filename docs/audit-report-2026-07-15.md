# Somotracker Backend Audit Report

**Date:** 2026-07-15  
**Auditor:** Automated agent (Claude Code)  
**Scope:** All 18 packages under `internal/` + `cmd/api/main.go`  
**Reference:** `backend/AGENTS.md` (v1.0.0) + Root `AGENTS.md` (v1.0.0)

---

## 1. Executive Summary

| # | Finding | Severity | `AGENTS.md` Section |
|---|---------|----------|---------------------|
| 1 | **4 packages have sentinel errors that don't wrap `middleware.ErrNotFound`** — `auth`, `invitations`, `members`, `teachers` use bare `errors.New()` instead of `fmt.Errorf("...: %w", middleware.ErrNotFound)`, so `HTTPError`'s `errors.Is()` will never match them. | **Critical** | §6 Error Handling — Sentinel errors |
| 2 | **3 `init()` functions exist** — in `curriculum/seeding.go` (×2) and `utils/http_client.go` (×1). The contract explicitly bans `init()` functions. | **Critical** | §3 Dependency Injection — "No init() functions anywhere" |
| 3 | **Migration file `000003_attendance_updated_at_nonbreak.up.sql`** exists, violating the "Do NOT create new migration files" policy. Should have been an inline `ALTER TABLE … ADD COLUMN IF NOT EXISTS` in `000001`. | **Moderate** | §4 Database Migration Policy |
| 4 | **`students/repository.go` line 510 uses string comparison** (`err.Error() == "no rows in result set"`) instead of `errors.Is(err, pgx.ErrNoRows)`. Anti-pattern explicitly forbidden. | **Critical** | §6 — String comparison anti-pattern |
| 5 | **Widespread `return err` and `return nil, err` unwrapped** across handlers, services, and repositories (~50+ instances). Errors must be `fmt.Errorf`-wrapped with the naming convention at every layer boundary. | **Moderate** | §6 — Wrapping convention |
| 6 | **Auth handler bypasses `HTTPError()` for all validation failures** — uses `c.Status(422/400).JSON(...)` directly instead of routing through the centralized `middleware.HTTPError`. | **Moderate** | §6 — HTTPError centralization |
| 7 | **Attendance handler also bypasses `HTTPError()`** — uses `c.Status(400/422).JSON(...)` with non-standard `"code": "VALIDATION_ERROR"` instead of the canonical `"invalid_input"`. | **Moderate** | §6 — HTTPError centralization |
| 8 | **`LoadSession()` in `security.go` ignores `pgx.ErrNoRows` silently** — uncovered errors bubble up as bare `return nil, err` to callers. | **Moderate** | §6 — Error wrapping + sql.ErrNoRows mapping |
| 9 | **Auth middleware `RequireAuth` returns `nil` on failure** (because `c.JSON()` returns nil in Fiber), meaning `RequireRole` won't detect auth failure and may return 403 instead of 401. | **Critical** | §6 — Forbidden patterns |
| 10 | **9 packages missing both `service_test.go` and `repository_test.go`** — assessment, attendance, behavior, cbcstreams, cbctimetableslots, imports, parents, teachers, timetablestructure. | **Moderate** | §5 — Testing |
| 11 | **Fiber recover middleware registered twice** — in `main.go` `registerApp()` and again in `middleware/security.go` `Register()`. Redundant, not harmful. | **Minor** | §6 — Panics/goroutines |

---

## 2. Architecture & DI Findings

| File/Package | Issue | `AGENTS.md` Rule | Severity | Fix |
|---|---|---|---|---|
| `curriculum/seeding.go:39,73` | Two `init()` functions populate `gradeMapping` and `defaultCBCFS` | §3: "No init() functions anywhere" | **Critical** | Move to lazy init via `sync.Once` in `NewSeedingService` or compute `gradeMapping` inline |
| `utils/http_client.go:19` | `init()` initializes `privateCIDRs` slice | §3: "No init() functions anywhere" | **Critical** | Move to `sync.Once` inside `NewSafeClient` or compute lazily |
| `auth/handler.go:490` | `var _ = errors.Is` — unused import suppressor in production code | §6: "Any `_ = someFunc()` in non-test code" | **Minor** | Remove; if the import is needed for type assertions in tests, gate it in a test file |
| `cbcclasses/module.go` | `fx.Invoke` wires `*academicyears.Service` directly into handler — hard import path from cbcclasses to academicyears | §1: "Zero circular imports" / §2: Cross-domain joins via orchestrator or view | **Moderate** | Should use a consumer-side interface (like `cbcschools` does with `AcademicYearSeeder`) instead of direct import |
| `students/module.go` | Imports `academicyears` and `imports` packages directly for DI wiring | §1: Cross-domain imports | **Minor** | Similar to cbcclasses pattern. Marginally acceptable as DI wiring but could use interface boundary |
| `auth/domain.go` | `Repository` interface declared in domain.go alongside types — fine. `SchoolCreator` correctly defined as consumer-side interface ✅ | §3: "Interfaces at consumer side" | **OK** | — |
| `academicyears/domain.go` | `Repository` interface declared inside `domain.go` with `context.Context` and `Tx` types. Domain remains pure Go (no Fiber/SQL/driver imports) ✅ | §1: domain.go must be pure Go | **OK** | — |

### Module Registration Compliance

All 18 `module.go` files checked — **no duplicate constructor registrations** found in any `fx.Provide` block. All follow the single `fx.Annotate` pattern correctly. ✅

### Constructor signatures

- All constructors are `New…` functions ✅
- Most return `(T, error)` as required ✅
- One exception pattern: middleware and some handlers use `New…` returning only `T` (e.g. `NewHandler`) — acceptable since these never fail

---

## 3. Migration Policy Findings

| File | Issue | `AGENTS.md` Rule | Severity | Fix |
|---|---|---|---|---|
| `migrations/000003_attendance_updated_at_nonbreak.up.sql` | New migration file created for `updated_at` column + non-break constraint trigger | §4: "Do NOT create new migration files" | **Moderate** | Squash into `000001_initial_schema.up.sql` as `ALTER TABLE … ADD COLUMN IF NOT EXISTS` and inline trigger creation |
| `migrations/000003_attendance_updated_at_nonbreak.down.sql` | Rollback file also violates policy | §4: Same | **Moderate** | Remove alongside the up file |
| `migrations/000002_seed.up.sql` | Contains only data (populates education levels, assessment types) — no DDL ✅ | §4: "000002_seed.up.sql contains only data, no DDL" | **OK** | — |
| `000001_initial_schema.up.sql` | ~124KB monolithic file — confirming it uses `CREATE TABLE IF NOT EXISTS` and inline patterns ✅ | §4: "Add columns inline" | **OK** | — |

---

## 4. Error Handling Findings

### 4a. Sentinel Error Wrapping Failures

These packages define sentinel errors with bare `errors.New()` instead of `fmt.Errorf("...: %w", middleware.ErrNotFound)`. This means `middleware.HTTPError`'s `errors.Is()` will **never** match these errors — they will all fall through to the 500 `"internal_error"` default case.

| Package | Current Definition (representative) | Severity | Fix |
|---|---|---|---|
| `auth/domain.go` | `ErrNotFound = errors.New("not_found")` | **Critical** | Change all 6 sentinels: `fmt.Errorf("auth not found: %w", middleware.ErrNotFound)`, etc. Note: `ErrExpiredToken` correctly wraps `middleware.ErrUnauthorized` but the base sentinels don't. |
| `invitations/domain.go` | `ErrNotFound = errors.New("invitations not found")` | **Critical** | Change all 6 sentinels to wrap `middleware.Err*` |
| `members/domain.go` | `ErrNotFound = errors.New("members not found")` | **Critical** | Change all 6 sentinels to wrap `middleware.Err*` |
| `teachers/domain.go` | `ErrNotFound = errors.New("teachers not found")` | **Critical** | Change all 6 sentinels to wrap `middleware.Err*` |

**Additional auth sentinel issues:**
- `auth.ErrNotFound = errors.New("not_found")` — uses underscore not space, inconsistent with convention
- `auth.ErrAlreadyExists = errors.New("already exists")` — no module prefix
- `auth.ErrInternal = errors.New("internal_error")` — not in the required sentinel set

**14 other packages** correctly wrap `middleware.Err*` in their domain.go sentinels ✅

### 4b. sql.ErrNoRows / pgx.ErrNoRows Mapping

| File | Line | Issue | Severity | Fix |
|---|---|---|---|---|
| `students/repository.go` | 510 | `return err != nil && err.Error() == "no rows in result set"` — uses string comparison instead of `errors.Is` | **Critical** | Change to `errors.Is(err, pgx.ErrNoRows)` |
| `middleware/security.go` | 183 | `loadSession()` returns bare `nil, err` when `pgx.ErrNoRows` occurs — not mapped to any sentinel | **Moderate** | Map `pgx.ErrNoRows` → `auth.ErrNotFound` before returning |

All repository files correctly map `pgx.ErrNoRows` → `ErrNotFound` for direct `QueryRow` calls ✅ (except the two issues above).

### 4c. Bare `return err` / `return nil, err` (No Wrapping)

The contract requires `fmt.Errorf("<Package>.<Type>.<Method>: %w", err)` at every layer boundary. These instances return errors without wrapping:

| File | Lines | Pattern | Severity |
|---|---|---|---|
| `auth/service.go` | 115, 117, 125, 485, 606, 617 | `return nil, err` | **Moderate** |
| `billing/service.go` | 153, 240, 360 | `return nil, err` | **Moderate** |
| `billing/repository.go` | 447, 452, 457 | `return nil, err` | **Moderate** |
| `students/repository.go` | 250 | `return nil, err` | **Moderate** |
| `parents/repository.go` | 161 | `return nil, err` | **Moderate** |
| `timetablestructure/service.go` | 295 | `return nil, err` | **Moderate** |
| `middleware/security.go` | 183 | `return nil, err` (from loadSession) | **Moderate** |
| `attendance/handler.go` | 54, 83, 123, 158, 190, 220, 251 | `return err` (from middleware.HTTPError, which is acceptable) | ✅ Acceptable |
| `behavior/handler.go` | 55, 74, 97, 113, 139, 170, 185, 201 | `return err` (same) | ✅ Acceptable |

**Note:** Handler-level `return err` patterns that pass through `middleware.HTTPError` are correct — the contract says intermediate layers wrap, handler is the logging layer.

### 4d. HTTPError Centralization Violations

The contract states `middleware.HTTPError` is the **only** place HTTP status codes should be decided for domain errors.

**Auth handler** (`auth/handler.go`): All validation/parsing errors use `c.Status(...).JSON(...)` directly:

```
Line 131:  c.Status(fiber.StatusUnprocessableEntity).JSON({"code":"invalid_input","message":"invalid request body"})
Line 137:  c.Status(fiber.StatusUnprocessableEntity).JSON({"code":"invalid_input","message":"email is required"})
Line 154:  c.Status(fiber.StatusUnprocessableEntity).JSON({"code":"invalid_input","message":"token query parameter is required"})
Line 215:  c.Status(fiber.StatusBadRequest).JSON(...)
Line 261:  c.Status(fiber.StatusUnprocessableEntity).JSON(...)
Line 267:  c.Status(fiber.StatusUnprocessableEntity).JSON(...)
Line 303:  c.Status(fiber.StatusUnprocessableEntity).JSON(...)
Line 339:  c.Status(fiber.StatusUnauthorized).JSON(...)
```

All return `{"error": "...", "message": "..."}` instead of the canonical `{"code": "...", "message": "...", "errors": {...}}`.

**Attendance handler** (`attendance/handler.go`): Uses non-standard error codes:

```
Line 42:  c.Status(400).JSON({"code": "VALIDATION_ERROR", "message": "active school not set"})
Line 59:  c.Status(400).JSON({"code": "VALIDATION_ERROR", "message": "timetable_slot_id is required"})
Line 88:  c.Status(401).JSON({"code": "UNAUTHORIZED", "message": "user not authenticated"})
Line 96:  c.Status(422).JSON({"code": "VALIDATION_ERROR", "message": "invalid request body"})
Line 163: c.Status(400).JSON({"code": "VALIDATION_ERROR", ...})
Line 195: c.Status(400).JSON({"code": "VALIDATION_ERROR", ...})
Line 203: c.Status(422).JSON({"code": "VALIDATION_ERROR", ...})
Line 225: c.Status(400).JSON({"code": "VALIDATION_ERROR", ...})
Line 233: c.Status(400).JSON({"code": "VALIDATION_ERROR", ...})
Line 258: c.Status(422).JSON({"code": "VALIDATION_ERROR", ...})
Line 265: c.Status(400).JSON({"code": "VALIDATION_ERROR", ...})
```

Should be `"code": "invalid_input"` and routed through `middleware.HTTPError(c, middleware.ErrInvalidInput)`.

### 4e. Logging Assessment

- `log/slog` is used throughout ✅
- No `log.Println`, `fmt.Println`, or `log.Printf` found in production code ✅
- **Stytch adapter logs at the integration layer** (stytch.go: `s.logger.Error("Stytch sendDiscoveryEmail failed", ...)`) AND the caller also logs (`service.go: s.logger.Error("auth: discovery send failed", ...)`) — potential **duplicate logging** violation. However, the stytch.go logs the raw Stytch API error detail, and the service layer logs the domain-level event. This is borderline and could be tightened.
- `imports/module.go` correctly uses `slog` for worker lifecycle ✅

### 4f. Transaction Rollback Pattern

The contract requires a deferred rollback pattern with dual-error logging. Several repositories use `_ = tx.Rollback(ctx)` (silent):

| File | Line | Issue | Severity | Fix |
|---|---|---|---|---|
| `attendance/repository.go` | 168 | `_ = tx.Rollback(ctx)` | **Moderate** | Log rollback error with `slog.Warn` |
| `cbctimetableslots/repository.go` | 314 | `_ = tx.Rollback(ctx)` | **Moderate** | Log rollback error |
| `cbcclasses/repository.go` | 199, 267, 356, 542 | `_ = tx.Rollback(ctx)` | **Moderate** | Log rollback error (×4 occurrences) |

`auth/repository.go` uses the correct dual-error pattern with logging ✅

### 4g. Silent Error Drops (`_ = someFunc()`)

Contract explicitly forbids `_ = someFunc()` in non-test code.

| File | Line | Statement | Severity | Fix |
|---|---|---|---|---|
| `attendance/service.go` | 116, 123 | `_ = s.redis.Del(ctx, dedupKey).Err()` | **Moderate** | Log the error at WARN level |
| `auth/service.go` | 482, 603 | `_ = s.rdb.Del(ctx, s.sessionKey(token)).Err()` | **Moderate** | Log the error at WARN level |
| `auth/service.go` | 621 | `_ = s.rdb.Set(ctx, ...).Err()` | **Moderate** | Log the error at WARN level |
| `database/database.go` | 49 | `_ = rdb.Close()` | **Minor** | Log close error |
| `config/config.go` | 46 | `_ = logger.Sync()` | **Minor** | Log sync error |
| `attendance/handler.go` | 40 | `schoolID, _ = c.Locals(...).(string)` | **Minor** | Acceptable — type assertion, not function call |
| `behavior/handler.go` | 39 | Same pattern | **Minor** | Acceptable |
| `cbctimetableslots/handler.go` | 37 | Same pattern | **Minor** | Acceptable |
| `cbcstreams/handler.go` | 52 | Same pattern | **Minor** | Acceptable |

### 4h. Auth Middleware `c.Next()` After Failed Auth Check

**`middleware/auth.go` `RequireAuth`:**

```go
func RequireAuth(c *fiber.Ctx) error {
    session := GetSession(c)
    if session == nil {
        return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{...})
        // ↑ c.JSON() returns nil in Fiber!
    }
    c.Locals("tenant_id", session.TenantID)
    c.Locals("user_id", session.UserID)
    c.Locals("role", session.Role)
    return c.Next()
}
```

The response IS sent as 401, but `RequireAuth` returns `nil` because `c.JSON()` always returns `nil` in Fiber. When called from `RequireRole`:

```go
if err := RequireAuth(c); err != nil {  // err is nil even when auth fails
    return err  // never reached
}
// Proceeds to role check and returns 403 instead of 401
```

**Severity: Critical** — Fix: return a `*fiber.Error` or wrap in a sentinel so callers can detect auth failure.

### 4i. External API Error Leaking

- `auth/stytch.go`: Wraps Stytch errors into module-local sentinels correctly ✅
  - Example: `fmt.Errorf("%w: stytch send discovery email: %v", ErrInternal, err)` — the `%v` captures the raw Stytch message, but since it's wrapped with `%w` of a `ErrInternal`, the sentinel takes precedence at `errors.Is()` time. The raw detail is logged via `s.logger.Error` at the Stytch layer.
- Would be slightly stronger as `fmt.Errorf("%w: stytch send discovery email", ErrInternal)` dropping the `%v` detail, but current pattern is acceptable.

### 4j. Fiber Recover Middleware — Double Registration

Registered in two places:
1. `cmd/api/main.go` `registerApp()`: `app.Use(fiberrecover.New())` — registered after `ErrorHandler` config
2. `middleware/security.go` `Register()` Layer 1: `app.Use(fibermiddleware.New())` — registered inside security pipeline

**Severity: Minor** — redundant but harmless. Should consolidate into one place (security.go is the canonical pipeline location).

### 4k. Goroutine Without recover

`cmd/api/main.go` line ~155:
```go
go func() {
    if err := app.Listen(":" + cfg.Port); err != nil && !errors.Is(err, http.ErrServerClosed) {
        slog.Error("fiber listen fatal", "error", err)
    }
}()
```

No `defer recover()`. If `app.Listen` panics (e.g., port in use), the panic crashes the goroutine silently without recovery. The Fiber recover middleware won't catch this because the goroutine runs outside the Fiber handler chain.

**Severity: Moderate** — add `defer func() { if r := recover(); r != nil { slog.ErrorContext(ctx, "fiber listen panic", "recover", r) } }()` inside the goroutine.

---

## 5. Type Sanity Findings

### 5a. Primitive Obsession — ID Types

All IDs across all packages are bare `string`:
- `TenantID`, `SchoolID`, `StudentID`, `UserID`, `ClassID`, `StreamID`, `AcademicTermID`, `ParentID`, `BehaviorNoteID`, etc.

Nothing prevents passing a `StudentID` where a `ClassID` is expected at compile time. The `uuid` package is used only for validation, not type safety.

**Severity: Minor** — not required by contract, but "pure Go" domain.go would benefit from `type StudentID string`, `type TenantID string`, etc.

### 5b. Enums

| Package | What | Status |
|---|---|---|
| `attendance` | `AttendanceStatus` with `StatusPresent`, `StatusAbsent`, etc. | ✅ Proper typed constants |
| `imports` | `ImportJobType`, `ImportJobStatus`, `ImportStagingStatus`, `ImportChunkStatus`, `ImportFailureType` | ✅ Proper typed constants |
| `behavior` | `BehaviorNoteStatus` with `StatusPendingReview` etc. | ✅ Proper typed constants |
| `billing` | `PaymentStatus` with `PaymentStatusUnpaid` etc. | ✅ Proper typed constants |
| `assessment` | Session status validated via `ValidSessionStatuses` map | ✅ Acceptable (string-based switch with map) |
| `timetablestructure` | `DayMonday` through `DaySunday` constants | ✅ Proper typed constants |
| `auth` | Roles (`SYSTEM_ADMIN`, `SCHOOL_ADMIN`, `TEACHER`) used as bare strings in `middleware/auth.go` and `auth/repository.go` with hardcoded `CASE` expressions | **Moderate** — No type safety, magic strings scattered |
| `cbcclasses` | `GradeLevel` — bare `string` in domain model | **Minor** |
| `curriculum` | `GradeLevel`, `EducationLevel` — bare strings | **Minor** |

### 5c. Optional vs Required Fields

| Package | Example | Assessment |
|---|---|---|
| `attendance` | `Note *string` (nullable DB column) | ✅ Correct pointer usage |
| `assessment` | `RawScore *string`, `TeacherObservationNotes *string` | ✅ Correct |
| `students` | `DateOfBirth *string`, `UPINumber *string`, `AdmissionNumber *string` | ✅ Correct |
| `auth` | `MeInfo` has all non-pointer fields — looks correct (always present after successful query) | ✅ Correct |
| `auth` | `RegistrationPayload.SchoolName` is `string` (not pointer) — correct, always required | ✅ Correct |
| `behavior` | `BehaviorNote.ReviewedByID *string`, `ReviewedAt *time.Time` — correct, nullable columns | ✅ Correct |

### 5d. Money/Precision Fields

- Billing amounts: `string` type for all `Amount` fields (maps to DB `NUMERIC`) ✅ — avoids float64 entirely
- No `float64` money anywhere ✅

### 5e. Time Handling

- `academicyears/domain.go` defines `DateOnly` type with explicit `time.Time` wrapping and UTC formatting ✅
- Most `CreatedAt`/`UpdatedAt` fields use `time.Time` — consistent ✅
- Some packages use date strings (`"2006-01-02"`) instead of `DateOnly` — e.g., `assessment` sessions have `DateAdministered string` — **Minor** inconsistency
- `attendance` uses `string` for `Date` fields — should ideally reuse `DateOnly` from academicyears or define its own

### 5f. DTO vs Domain Separation

**Good pattern:** Most packages separate HTTP payloads (`CreateTermBody`, `CreateBlueprintPayload`) from domain structs (`AcademicTerm`, `AssessmentBlueprint`) ✅

**Mixed concern:** Domain structs carry both `json:"..."` and `db:"..."` tags:
```go
type AcademicYear struct {
    ID        string    `db:"id"         json:"id"`
    TenantID  string    `db:"tenant_id"  json:"-"`
    ...
}
```

This technically violates "pure Go" in domain.go, but is a pragmatic trade-off seen across 16/18 packages. Only `teachers/domain.go` and `members/domain.go` avoid DB/JSON tags on domain types — ironically making them the most "pure."

**Severity: Minor** — pragmatic deviates from ideal but not harmful.

### 5g. `any`/`interface{}` Usage

- `academicyears/domain.go:24` `func (d *DateOnly) Scan(src any) error` — legitimate `sql.Scanner` interface implementation ✅
- No `interface{}` in domain struct definitions ✅
- `imports/domain.go` uses `json.RawMessage` for opaque data — correct use of a well-typed wrapper, not bare `interface{}` ✅

### 5h. Cross-Package Type Duplication

`invitations/domain.go` defines its own `Invitation` struct which overlaps with `auth/domain.go`'s `Invitation` type. Both model the same `invitations` DB table:

| Field | `auth.Invitation` | `invitations.Invitation` |
|---|---|---|
| `ID` | ✅ | ✅ |
| `TenantID` | ✅ | ✅ |
| `SchoolID` | ✅ | ✅ |
| `Role` | ✅ | ✅ |
| `Email` | ✅ | ✅ |
| `FullName` | ✅ | ✅ (`*string` — nullable) |
| `Status` | ✅ | ✅ |
| `ExpiresAt` | ✅ | ✅ |
| `StytchMemberID` | ✅ | ❌ |
| `RegistrationNumber` | ✅ | ❌ |
| `CreatedAt` | ❌ | ✅ |

**Severity: Minor** — both serve different purposes (auth reads for invite acceptance, invitations manages CRUD). If they diverge further, unify into a shared read-model.

---

## 6. API Validation Findings

### 6a. Handler-Side Validation Bypassing HTTPError

As noted in §4d, both `auth/handler.go` and `attendance/handler.go` bypass `middleware.HTTPError` for validation errors. Consequences:

- Auth returns `{"error": "invalid_input", "message": "..."}` — **non-canonical** (should be `{"code": "invalid_input", ...}`)
- Attendance returns `{"code": "VALIDATION_ERROR", "message": "..."}` — **wrong snake_case, wrong code name** (should be `"invalid_input"`)
- Attendance returns `{"code": "UNAUTHORIZED"}` — **wrong** (should be `"unauthorized"` — lowercase)

### 6b. Validation Centralization

- `auth/domain.go` has a `RegistrationPayload.Validate()` method — centralized per-type ✅
- Most other packages do not have centralized validation — validation is ad-hoc in handlers or service methods
- No shared validation library or struct tags used consistently across packages

### 6c. Authorization Consistency

- `middleware.RequireRole("SCHOOL_ADMIN", "SYSTEM_ADMIN")` is used correctly across all handler registrations
- **But** the `RequireAuth` returning nil on failure (see §4h) means role-protected routes may return 403 instead of 401 when the session is missing — cross-cutting issue affecting every role-gated endpoint

### 6d. Missing Validation Patterns

- `cbcclasses/handler.go`: Receives `CreateClassPayload` with `GradeLevel`, `AcademicYearID`, etc. — are these validated? Need to check if empty strings or invalid UUIDs are caught before reaching the repository.
- `attendance/handler.go`: `BulkAttendancePayload.Date` — validated only as non-empty string, not as a valid date format
- No consistent pattern for UUID format validation across handlers

---

## 7. Test Coverage Findings

### Test File Matrix

| Package | `*_service_test.go` | `*_repository_test.go` | `handler_test.go` | Risk if Untested |
|---|---|---|---|---|
| `academicyears` | ✅ `service_test.go` | ❌ | ✅ | Integration: constraint/RLS for year/term CRUD |
| `assessment` | ✅ `service_test.go` | ❌ | ❌ | Complex state machine (session status transitions) untested |
| `attendance` | ❌ | ❌ | ❌ | **HIGH** — revenue-critical, slot/roster queries complex |
| `auth` | ✅ `service_test.go` | ❌ (`integration_test.go` exists but is end-to-end) | ✅ | Integration: session expiry, concurrent registration |
| `behavior` | ❌ | ❌ | ❌ | **HIGH** — multi-step approval workflow untested |
| `billing` | ✅ `service_test.go` | ❌ | ✅ | Integration: invoice generation, payment recording |
| `cbcclasses` | ✅ `service_test.go` | ❌ | ✅ | Integration: unique constraints, RLS for class/roster |
| `cbcschools` | ✅ `service_test.go` | ❌ | ✅ | Integration: school creation with curriculum seeding |
| `cbcstreams` | ❌ | ❌ | ❌ | **HIGH** — stream-name uniqueness, active-enrollment checks |
| `cbctimetableslots` | ❌ | ❌ | ❌ | Complex overlap constraints, teacher double-booking |
| `curriculum` | ✅ `service_test.go` | ❌ | ✅ | Integration: FK cascade, tree hierarchy |
| `imports` | ✅ `service_test.go` | ❌ (`integration_test.go` exists but covers different scope) | ❌ | **HIGH** — concurrency correctness, atomic chunk completion |
| `invitations` | ✅ `service_test.go` | ❌ | ✅ | Integration: bulk invite, expiry, idempotency |
| `members` | ✅ `service_test.go` | ❌ | ✅ | Integration: role ordering, pagination |
| `parents` | ✅ `service_test.go` | ❌ | ❌ | Integration: student linking, duplicate prevention |
| `students` | ❌ | ❌ | ✅ | **HIGH** — import pipeline, duplicate detection, enrollment |
| `teachers` | ❌ | ❌ | ❌ | **HIGH** — toggle-active, TSC number uniqueness |
| `timetablestructure` | ❌ | ❌ | ❌ | **HIGH** — overlap detection, day replication, shift-follow |
| `database` | ❌ | ✅ `migration_test.go` | N/A | Only migration tests |

### Packages with Zero Tests (6 packages)

- **attendance**, **behavior**, **cbcstreams**, **cbctimetableslots**, **teachers**, **timetablestructure**

### Unit Test Quality

- ✅ `auth/service_test.go` — uses mocks, no real DB
- ✅ `billing/service_test.go` — uses mocks
- ✅ `cbcclasses/service_test.go` — in-memory mocks
- ❌ Need to verify all service tests use zero I/O (run under `go test -short`)

### Integration Test Coverage Gaps

- **Every package** is missing `*_repository_test.go`. The only integration tests that exist are:
  - `database/migration_test.go` — migration execution test
  - `auth/integration_test.go` — end-to-end auth flow
  - `imports/integration_test.go` — import workflow

---

## 8. Prioritized Action List (Top 5 by Risk × Effort)

| # | Fix | Risk | Effort | Rationale |
|---|---|---|---|---|
| **1** | **Fix 4 packages' sentinel errors to wrap `middleware.Err*`** (auth, invitations, members, teachers) | **Critical** | **Low** (~30 min) | Without this, all errors from auth/invitations/members/teachers return 500 instead of proper 404/401/403/409. Auth is the entry point for every user. |
| **2** | **Fix `students/repository.go:510` string comparison** | **Critical** | **Low** (~5 min) | Forbidden pattern per contract. Fragile against driver changes. |
| **3** | **Fix `middleware/auth.go` `RequireAuth` returning nil on failure** | **Critical** | **Low** (~15 min) | Causes wrong HTTP status code (403 vs 401) across all role-protected routes. |
| **4** | **Remove `init()` functions from curriculum + utils; use `sync.Once`** | **Critical** | **Low** (~30 min) | Explicitly banned. Hinders testability. Surprising startup order. |
| **5** | **Centralize auth + attendance handler validation responses through `middleware.HTTPError`** | **Moderate** | **Medium** (~2 hr) | Non-canonical JSON shapes mean frontend `api/client.ts` may parse error codes incorrectly, causing silent UI failures. |

### Reject-on-Sight Items for Future Code Reviews

- ⛔ No duplicate constructor registrations in `fx.Provide` — none found ✅
- ⛔ No `init()` functions — **3 found, must be removed** ❌
- ⛔ No new migration files — **1 found (000003), must be squashed** ❌
- ⛔ No `err.Error() ==` string comparisons — **1 found in students/repository.go** ❌
- ⛔ No `_ = someFunc()` in non-test code — **multiple found** ❌

---

## Appendix: Files with Most Violations

| File | Critical | Moderate | Minor | Key Issues |
|---|---|---|---|---|
| `auth/handler.go` | 1 | 8 | 1 | HTTPError bypass, RequireAuth return nil, sentinel errors |
| `attendance/handler.go` | 0 | 11 | 1 | HTTPError bypass, non-standard error codes |
| `auth/service.go` | 0 | 4 | 2 | Bare return nil,err, silent redis errors |
| `billing/service.go` | 0 | 3 | 0 | Bare return nil,err |
| `students/repository.go` | 1 | 1 | 0 | String comparison, bare return nil,err |
| `curriculum/seeding.go` | 1 | 0 | 0 | Two init() functions |
| `utils/http_client.go` | 1 | 0 | 0 | init() function |
