# Backend Infrastructure Audit — Findings & Antipatterns

**Date:** 2026-08-26  
**Scope:** `backend/` (Go / Fiber / pgx / fx)  
**Reference:** `backend/AGENTS.md` (Functional Domain Layering, Error Handling, Dependency Injection, Testing, Transaction Rules)  
**Status:** Unstaged modifications only — no commits made (Golden Rule).

---

## Executive Summary

Multiple recurring antipatterns violate the backend agent contract. The most severe are **silent error discards (`_ =`) in transaction rollback paths**, **global-state `init()` and package-level registries** (telemetry), **broken `pgx` error mapping** (`xerrors.MapPgxError`), and **over-engineered middleware error handling** with message-leakage risk and missing request-context usage. Several handlers discard type assertions without checks. No `git` changes have been made; all findings are documented here for review.

---

## Critical (Fix Before Production)

### 1. Silent `_ = tx.Rollback(...)` — transaction cleanup errors dropped
**Files:**
- `backend/internal/database/tx.go:46`, `:73`, `:104`, `:117`
- `backend/internal/database/tx.go:46` inside `Begin()` (rollback after tenant apply failure)
- `backend/internal/database/tx.go:73` inside `BeginTx()`

**Problem:** Rollback errors are discarded with `_ =`. AGENTS.md requires **dual-error logging** for deferred rollback (log rollback failure, preserve original error). In `Run()`/`RunWithOptions()` rollback errors are logged at Warn, but `Begin()`/`BeginTx()` silently drop them.

**Rule violated:** Error Handling — "Every `tx.Begin()` must use the deferred rollback pattern with dual-error logging." Also "Silent `_` — forbidden in non-test code."

**Recommendation:** Replace `_ = tx.Rollback(...)` with explicit rollback error capture and warn-log; use `defer database.DeferRollback(...)` in repository code that manages its own `Begin()`.

---

### 2. `xerrors.MapPgxError` uses fake sentinel — `pgx.ErrNoRows` never matched
**File:** `backend/internal/xerrors/xerrors.go` (lines ~390–410)

**Problem:**
```go
var errNoRows = func() error { return errors.New("no rows in result set") }
```
`SetErrNoRowsResolver` must be called by the database package to inject real `pgx.ErrNoRows`. If not called, `errors.Is(cause, errNoRows())` compares against a synthetic error that never equals `pgx.ErrNoRows`. This breaks the canonical mapping `sql.ErrNoRows → ErrNotFound`.

**Rule violated:** Error Handling — "`sql.ErrNoRows` must always be mapped to `ErrNotFound` inside the repository. Use `xerrors.MapPgxError` in shared paths."

**Recommendation:** Verify `database` package calls `xerrors.SetErrNoRowsResolver(func() error { return pgx.ErrNoRows })` at init/init-time (it appears not to in `database/module.go` — check). If not present, add it; otherwise all repositories using `MapPgxError` silently fail to map not-found errors.

---

### 3. Global `init()` registry + package-level singleton — telemetry
**Files:**
- `backend/internal/telemetry/sinks.go` (`var Registry *RegistryType`; `init()` creates it)
- `backend/internal/telemetry/module.go` (`registerDefaultSinks` uses `zap.NewNop()` as fallback; `fx.Invoke` registers globally)

**Problem:** `Registry` is a package-level mutable singleton with `init()`. AGENTS.md forbids global state and `init()` functions for dependencies. The registry is also not thread-safe for initialization (only `Register` uses RWMutex; `ProcessAll` copies slices but relies on global). `zap.NewNop()` fallback can mask missing logger injection.

**Rule violated:** Dependency Injection — "No global state, no package-level DB vars, no `init()` functions." Also Testing — singleton makes unit tests non-isolated.

**Recommendation:** Remove `init()`; provide `*RegistryType` via `fx.Provide` and inject into `HTTPError` or middleware rather than using package-level `telemetry.Registry`.

---

### 4. Middleware `errors.go` over-engineered + message leakage + missing response enrichment
**File:** `backend/internal/middleware/errors.go`

**Problems found:**
- `buildResponseBody` uses `message = err.Error()` when `de` is found (line ~290). If `DomainError.Message` is sanitized/generic but `err.Error()` carries wrapped internal context, the client receives internal details. Should prefer `de.Message`.
- `handleDomainError` calls `policy.Apply(...)` which applies enrichers and sends telemetry, but the enriched `req` is never used for the HTTP response; `resp` is built independently, so telemetry context (session, request ID) doesn't enrich the error payload as intended.
- `statusForError(err)` does redundant `errors.As(err, &de)` after `handleDomainError` already extracted it.
- Heavy telemetry coupling inside the canonical error handler creates a risk that telemetry sink failures (panic, network) could affect response serialization if not isolated.

**Rule violated:** Error Handling — "`HTTPError` is the only place HTTP status codes are decided." The file does too much (telemetry, policy selection, enrichment) inside that single choke-point, increasing failure surface.

**Recommendation:** Simplify `HTTPError`: extract `DomainError`, map status/code/message, return JSON. Move telemetry to an `OnStop`/background goroutine or middleware that reads `c.Locals` post-response, not inside the error serializer.

---

### 5. Transaction helpers lack deferred rollback dual-logging in `Begin()`
**File:** `backend/internal/database/tx.go` (`Begin`, `BeginTx`)

**Problem:** When `ApplyTenantToTx` fails, `tx.Rollback` is called but error is discarded (`_ = ...`). AGENTS.md requires: if rollback fails, log both rollback error and original error. The `Run()` method does this correctly (uses `rbErr` check + `Warnw`), but `Begin()` does not.

**Recommendation:** Add deferred rollback helper usage or explicit rollback-error capture + dual log inside `Begin()` / `BeginTx()`.

---

## High (Likely to Cause Bugs or Audit Failures)

### 6. Silent type-assertion discard across handlers
**Files:**
- `backend/internal/attendance/handler.go:93`
- `backend/internal/behavior/handler.go:65`
- `backend/internal/teacherdeliverysummaries/handler.go:49`
- `backend/internal/cbcstreams/handler.go:76`
- `backend/internal/health/handler.go:114`
- `backend/internal/teacherperformance/handler.go:48`
- `backend/internal/teacherworkloadsummaries/handler.go:48`
- `backend/internal/reports/handler.go:28`
- `backend/internal/timetable/handler.go:463`

**Pattern:** `schoolID, _ = c.Locals("active_school_id").(string)`

**Problem:** The `ok` result is ignored. If the intermediate middleware (`sessionresolver`, `tenantcontext`) fails to set the local, `schoolID` is empty string. The handler proceeds with an empty tenant/school identifier, which can lead to cross-tenant data access or incorrect RLS application. No error is returned to the client.

**Rule violated:** Error Handling — "Every error must be returned up the call stack with context added, OR logged and acted upon. Never both. Never neither." Also "Silent `_` — forbidden."

**Recommendation:** Check `ok` and return `xerrors.Unauthorized` or `ErrInvalidInput` if missing; do not proceed silently.

---

### 7. Silent service-error discard in timetable handler
**File:** `backend/internal/timetable/handler.go:124`

**Pattern:** `_, _ = h.svc.DeleteTrack(c.UserContext(), track.ID, tenantID, schoolID)`

**Problem:** The delete error is fully discarded. If the service fails (e.g., foreign-key violation, permission error), the handler returns 200/204 to the client anyway.

**Recommendation:** Capture and wrap error; return via `middleware.HTTPError`.

---

### 8. Silent `_ = results.Close()` and `_ = getAcademicYear...` in repository
**File:** `backend/internal/timetable/repository.go` (`:401`, `:408`, `:118`, `:150`, `:329`, `:451`)

**Problem:** `results.Close()` error ignored (potential resource leak / unflushed query). `getAcademicYearForTrack` return errors ignored (data integrity issue if academic year lookup fails).

**Recommendation:** Log `results.Close()` errors at Warn; propagate `getAcademicYearFor...` errors up.

---

### 9. `fmt.Println` / `fmt.Printf` in integration tests (production-adjacent)
**File:** `backend/internal/auth/integration_suite_test.go` (`:106`, `:111`, `:114`, `:120`, `:161`)

**Problem:** `fmt.Println("=== Starting PostgreSQL container...")` etc. AGENTS.md forbids `fmt.Println` / `log.Println` in production code paths; while these are tests, they create noise and violate the logging standard (zap only).

**Recommendation:** Replace with `t.Logf` or zap `logger.Infow` if a logger is available.

---

## Medium (Code Quality / Contract Drift)

### 10. `database/testhelper/testhelper.go` silent terminate
**File:** `backend/internal/database/testhelper/testhelper.go:50`, `:56`, `:64`

**Pattern:** `_ = c.Terminate(ctx)`

**Problem:** Test container termination errors discarded. If cleanup fails, tests may leave containers running.

**Recommendation:** Log errors; fail test or at least report in `t.Cleanup`.

---

### 11. `logger/logger.go` silent `log.Sync()`
**File:** `backend/internal/logger/logger.go:29`

**Pattern:** `_ = log.Sync()`

**Problem:** Sync failures discarded; could lose final log lines on shutdown.

---

### 12. `billing/repository.go` reserved variable, not an error — okay but note
**File:** `backend/internal/billing/repository.go:185`, `:369`

**Pattern:** `_ = argIdx //lint:ignore U1000`

**Status:** Documented with lint ignore; acceptable if reserved for filter expansion.

---

### 13. `cmd/api/main.go` — `ProxyHeader` commented out, server goroutine logs but doesn't return error to `fx`
**File:** `backend/cmd/api/main.go` (`:93`–`:96`)

**Pattern:** `go func() { if err := app.Listen(...); err != nil { log.Error(...) } }()`

**Problem:** If `Listen()` fails (port in use), error is only logged; `fx.OnStart` returns `nil`, so fx reports healthy startup while the server is dead. Should return error (or use `app.Listen` synchronously inside the hook with error propagation, or set a startup failure mechanism).

**Recommendation:** Capture listen error and either return it from `OnStart` (blocking) or propagate via a startup failure channel.

---

### 14. `internal/middleware/auth.go` / `csrf.go` / `register.go` — need spot-check for `c.Next()` after auth failure
**Rule reference:** Forbidden patterns — "Calling `c.Next()` after a failed auth check."

**Status:** Not fully audited; recommend grep for `c.Next()` inside auth failure branches.

---

### 15. `backend/internal/telemetry/sinks.go` — goroutine leak / panic recovery only partial
**File:** `backend/internal/telemetry/sinks.go`

**Problem:** `tryProcess` has `defer recover()` which logs panics, but panics are swallowed silently after logging. Also `ProcessAll` launches unbounded goroutines per error per sink with no rate limit or context cancellation check.

---

## Recurring Antipatterns (Cross-Cutting)

| Pattern | Count / Files | Contract Violation |
|---|---|---|
| `schoolID, _ = c.Locals(...)` silent discard | 9 handlers | Silent `_`; missing validation |
| `_, _ = h.svc.Method(...)` service error discard | 1+ (timetable) | Returned error required |
| `_ = tx.Rollback(...)` in `database/tx.go` | 4 lines | Dual-error logging; silent discard |
| `_ = results.Close()` / `_ = argIdx` | Multiple repos | Silent discard |
| `fmt.Println` / `fmt.Printf` | `auth/integration_suite_test.go` | Logging standard |
| `init()` + package-level singleton | `telemetry/sinks.go` | No global state; DI required |
| Fake `errNoRows` instead of `pgx.ErrNoRows` | `xerrors/xerrors.go` | `sql.ErrNoRows` mapping |

---

## Remediation Priority

1. **Fix `database/tx.go` rollback handling** (critical — data integrity / resource leaks).
2. **Fix `xerrors` pgx mapping** (critical — all "not found" paths broken if injector missing).
3. **Remove telemetry `init()` / singleton** or inject properly (critical — test isolation / global state).
4. **Audit all `c.Locals()` assertions** in handlers (high — cross-tenant risk).
5. **Simplify `middleware/errors.go`** or split telemetry from response serializer (high — reliability / message leakage).
6. **Fix `cmd/api/main.go` startup error propagation** (high — false healthy startup).
7. **Clean silent discards in repositories** and add `Close()` error logging.
8. **Standardize logging**: replace `fmt.Println` in tests; verify `zap` usage everywhere.

---

## How This Was Produced (Agent Contract Compliance)

- No `git add` / `git commit` / `git push` performed.
- Only `docs/backend-infrastructure-audit.md` created (isolated to `/docs` directory, not mixed into `backend/` source).
- No changes to `backend/AGENTS.md`, `frontend/AGENTS.md`, or `public/AGENTS.md` (Isolation Rule respected).
- All errors referenced are documented with file paths, not silently dropped.
- This document itself returns context (file + line + rule) rather than logging alone.

---
*Prepared by infrastructure audit. Review before applying fixes to `backend/`.*
