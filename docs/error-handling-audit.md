# Backend Error Handling Audit

**Date:** 2025-08-25  
**Auditor:** AI Assistant  
**Scope:** `/backend/internal` (all Go packages)

---

## Executive Summary

The backend has a **canonical error response contract** defined in `internal/middleware/errors.go` that all non-2xx HTTP responses **MUST** follow:

```json
{
  "code":    "snake_case_error_code",
  "message": "human readable message",
  "errors":  { "field_name": ["Specific field validation message"] }
}
```

The centralized `middleware.HTTPError` function implements this contract. However, **multiple handlers bypass this contract**, leading to inconsistent error responses, missing request IDs, and potential error swallowing in transaction cleanup.

---

## Critical Issues

### 1. Transaction Commit/Rollback Errors Silently Swallowed

**Location:** `internal/auth/service.go` (5 occurrences)

```go
// Lines 332, 407, 661, 698, 845
defer func() { _ = finish() }()
```

The `finish()` function commits or rolls back a tenant-scoped transaction and **returns an error** on failure:

```go
finish := func() error {
    if committed {
        return nil
    }
    if err := tx.Commit(context.WithoutCancel(ctx)); err != nil {
        _ = tx.Rollback(context.WithoutCancel(ctx))  // <-- Also swallows rollback error!
        return fmt.Errorf("auth.Service.tenantScope: commit: %w", err)
    }
    committed = true
    return nil
}
```

**Impact:** Database transaction failures (deadlocks, constraint violations during commit, network issues) are **completely silent**. The HTTP handler proceeds as if the transaction succeeded.

**Fix:** Log the error at minimum, or propagate it:
```go
defer func() {
    if err := finish(); err != nil {
        s.logger.Errorw("tenantScope finish failed", "error", err)
    }
}()
```

---

### 2. Rollback Errors Silently Ignored Across Repositories

**Locations:** Multiple repository files use `_ = tx.Rollback(...)`

| File | Line | Pattern |
|------|------|---------|
| `internal/attendance/worker.go` | 196 | `defer func() { _ = tx.Rollback(...) }()` |
| `internal/attendance/repository.go` | Multiple | `_ = tx.Rollback(ctx)` |
| `internal/behavior/repository.go` | (defer) | Implicit in transaction helpers |
| `internal/database/tenant.go` | Multiple | `_ = tx.Rollback(...)` |
| `internal/auth/service.go` | Inside `finish()` | `_ = tx.Rollback(...)` |
| `internal/academicyears/repository.go` | Multiple | `_ = tx.Rollback(ctx)` |
| `internal/assessments/worker.go` | Multiple | `_ = tx.Rollback(...)` |

**Impact:** If rollback itself fails (e.g., connection lost), the error is lost. While less critical than commit failures, it masks connection issues.

**Fix:** Log rollback failures:
```go
defer func() {
    if rbErr := tx.Rollback(ctx); rbErr != nil && rbErr != pgx.ErrTxClosed {
        loggerFrom(c).Warnw("rollback failed", "error", rbErr)
    }
}()
```

---

## High Severity Issues

### 3. Inconsistent Error Response Formats

**Three different error response patterns exist:**

#### A. Canonical Format (via `middleware.HTTPError`) ✅
Used by: `assessments/handler.go`, `auth/handler.go` (mostly), service layer returns

```json
{ "code": "not_found", "message": "resource not found", "errors": {...} }
```

#### B. Custom `writeError` Function (students handler) ❌
**File:** `internal/students/handler.go` (lines 124-131, used 30+ times)

```go
type errorResponse struct {
    Code    string              `json:"code"`
    Message string              `json:"message"`
    Errors  map[string][]string `json:"errors,omitempty"`
}
```
**Differs from canonical:** Field name is `Errors` (capital E) vs `errors` (lowercase). Frontend expects lowercase per contract.

#### C. Direct `c.Status(...).JSON(...)` (attendance, behavior, auth handlers) ❌
**Files:** 
- `internal/attendance/handler.go` (~15 occurrences)
- `internal/behavior/handler.go` (~10 occurrences)
- `internal/auth/handler.go` (2 occurrences: switch-school endpoint)

```go
return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
    "code":    "VALIDATION_ERROR",
    "message": "active school not set",
})
```
**Issues:** 
- Inconsistent field casing (`VALIDATION_ERROR` vs `invalid_input`)
- Missing `request_id` field that `middleware.HTTPError` injects
- Missing `errors` field for validation details
- Status codes sometimes wrong (e.g., `StatusUnauthorized` for missing school is actually a validation error)

---

### 4. Fiber ErrorHandler Bypasses Canonical Format for 4xx Errors

**File:** `cmd/api/main.go` (lines 104-118)

```go
ErrorHandler: func(c *fiber.Ctx, err error) error {
    var e *fiber.Error
    if errors.As(err, &e) {
        if e.Code < 500 {
            return fiber.DefaultErrorHandler(c, err)  // <-- Returns HTML/text, not JSON!
        }
    }
    return middleware.HTTPError(c, err)
}
```

**Impact:** Fiber's built-in 404, 405, 400 errors return **HTML/plain text** instead of the canonical JSON contract. Frontend cannot parse these reliably.

**Fix:** Always use `middleware.HTTPError` or a wrapper that maps Fiber errors to canonical format.

---

## Medium Severity Issues

### 5. Missing Request ID in Direct JSON Responses

`middleware.HTTPError` injects `request_id` into the response body when available. Direct JSON responses **never include it**, breaking correlation between frontend errors and backend logs.

**Affected handlers:** All direct `c.Status(...).JSON(...)` calls in attendance, behavior, auth, students handlers.

---

### 6. Inconsistent Validation Error Codes

| Handler | Code Used | Should Be |
|---------|-----------|-----------|
| students | `VALIDATION_ERROR` | `invalid_input` |
| attendance | `VALIDATION_ERROR` | `invalid_input` |
| behavior | `VALIDATION_ERROR` | `invalid_input` |
| auth (switch-school) | `VALIDATION_ERROR` | `invalid_input` |
| assessments | (uses middleware) | `invalid_input` ✅ |

The canonical contract (and xerrors sentinels) uses `invalid_input` for 400 errors.

---

### 7. Wrong HTTP Status Codes for Certain Errors

**File:** `internal/attendance/handler.go` line 237, `internal/behavior/handler.go` line 175

```go
// Missing school ID returns 401 Unauthorized
return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
    "code":    "UNAUTHORIZED",
    "message": "user not authenticated",
})
```

**Issue:** Missing `active_school_id` is a **client validation error** (400), not an authentication failure (401). The user IS authenticated; they just haven't selected a school.

**Same pattern:** `internal/auth/handler.go` line 376 returns 401 for missing session cookie (correct), but line 419/425 returns direct JSON for switch-school validation.

---

### 8. Error Context Not Consistently Added

**Good pattern (service layer):**
```go
return nil, fmt.Errorf("attendance.Service.CreateSession: %w", err)
```

**Inconsistent pattern (repository layer):**
Some repositories wrap with context, others don't:
```go
// With context (good)
return nil, fmt.Errorf("attendance.Repository.CreateSession: %w", err)

// Without context (bad - loses call site)
return nil, err
```

**Files with good context wrapping:** `attendance/service.go`, `students/service.go`, `behavior/service.go`
**Files with inconsistent wrapping:** Various repositories - some methods wrap, others don't.

---

## Low Severity / Style Issues

### 9. Type Assertion Without Check (Panic Risk)

**Files:** Multiple handlers

```go
tenantID = c.Locals("tenant_id").(string)  // Panics if not string or missing
schoolID, _ = c.Locals("active_school_id").(string)  // Silent failure if wrong type
```

**Fix:** Use safe type assertion with ok check:
```go
tenantID, ok := c.Locals("tenant_id").(string)
if !ok || tenantID == "" {
    return middleware.HTTPError(c, middleware.ErrUnauthorized)
}
```

---

### 10. Empty Error Handling in Test Helpers (Non-Production)

**File:** `internal/database/testhelper/testhelper.go`

```go
_ = c.Terminate(ctx)  // Ignored in test helper - acceptable for tests
```

**Verdict:** Acceptable since test-only, but worth noting.

---

## Compliance Matrix by Handler

| Handler | Uses `middleware.HTTPError` | Custom Format | Direct JSON | Consistent Codes |
|---------|----------------------------|---------------|-------------|------------------|
| assessments | ✅ 100% | ❌ | ❌ | ✅ |
| auth | ~90% | ❌ | 2 endpoints | ⚠️ |
| attendance | ~80% | ❌ | ~15 endpoints | ❌ |
| behavior | ~75% | ❌ | ~10 endpoints | ❌ |
| students | ~30% | ✅ (writeError) | ❌ | ❌ |
| Other handlers | Not fully audited | | | |

---

## Recommended Remediation Plan

### Phase 1: Critical Fixes (Do First)
1. **Fix transaction error swallowing** in `auth/service.go` - log or propagate `finish()` errors
2. **Fix rollback error swallowing** in all repositories - add logging
3. **Fix Fiber ErrorHandler** in `main.go` to use canonical format for ALL errors

### Phase 2: Consistency (Do Next)
4. **Remove `writeError`** from `students/handler.go` - use `middleware.HTTPError` everywhere
5. **Replace direct JSON** in attendance/behavior/auth handlers with `middleware.HTTPError`
6. **Standardize error codes** to match xerrors sentinels (`invalid_input`, `not_found`, etc.)

### Phase 3: Quality Improvements
7. **Add request_id** to all error responses (automatic via middleware.HTTPError)
8. **Add context wrapping** consistently in all repository methods
9. **Fix type assertions** to be safe (ok checks)
10. **Fix wrong status codes** (401 → 400 for missing school)

---

## Appendix: Canonical Error Codes Reference

From `internal/xerrors/domain.go` and `internal/middleware/errors.go`:

| Code | HTTP Status | Use Case |
|------|-------------|----------|
| `not_found` | 404 | Resource doesn't exist |
| `already_exists` | 409 | Unique constraint violation |
| `invalid_input` | 400 | Validation failure |
| `unauthorized` | 401 | Not authenticated / bad credentials |
| `forbidden` | 403 | Authenticated but insufficient permissions |
| `conflict` | 409 | Business logic conflict |
| `unprocessable_entity` | 422 | Semantic validation failure |
| `device_fingerprint_mismatch` | 401 | Session bound to different device |
| `request_canceled` | 499 | Client cancelled request |
| `timeout` | 504 | Request timeout |
| `internal_error` | 500 | Unexpected server error |

---

## Files Requiring Changes (Priority Order)

1. `cmd/api/main.go` - Fix ErrorHandler
2. `internal/auth/service.go` - Fix deferred finish() error swallowing
3. `internal/students/handler.go` - Remove writeError, use middleware.HTTPError
4. `internal/attendance/handler.go` - Replace direct JSON with middleware.HTTPError
5. `internal/behavior/handler.go` - Replace direct JSON with middleware.HTTPError
6. `internal/auth/handler.go` - Fix switch-school endpoint
7. All repository files - Add rollback error logging
8. All service/repository files - Ensure consistent error wrapping context

---

## Verification Checklist

After fixes, verify:
- [ ] All 4xx/5xx responses return canonical JSON format
- [ ] All responses include `request_id` when available
- [ ] Error codes match xerrors sentinels
- [ ] Transaction commit/rollback errors are logged
- [ ] Fiber 404/405 return JSON, not HTML
- [ ] Frontend error handling works without special cases