# Error Handling Reference

## Overview

Somotracker enforces a **strict, uniform error handling contract** across all layers (HTTP, services, database). Every non-2xx response returns the same canonical JSON structure. Errors are **never silently dropped** — they are either returned up the call stack with added context, or logged and acted upon at the boundary.

---

## Canonical Error Response

Every HTTP error (4xx, 5xx) returns:

```json
{
  "code": "snake_case_error_code",
  "message": "Human readable message",
  "errors": { "field_name": ["Specific validation message"] }
}
```

| Field | Type | Purpose |
|-------|------|---------|
| `code` | string | Machine-readable, stable identifier for client handling |
| `message` | string | User-facing description (never leaks internals) |
| `errors` | object | Optional field-level validation details (empty object `{}` if none) |

### Examples

**400 Bad Request — Validation:**
```json
{
  "code": "invalid_uuid",
  "message": "invalid user id",
  "errors": { "id": ["must be a valid UUID"] }
}
```

**401 Unauthorized:**
```json
{
  "code": "unauthorized",
  "message": "Unauthorized. Please log in to continue.",
  "errors": {}
}
```

**429 Rate Limited:**
```json
{
  "code": "rate_limit_exceeded",
  "message": "Too many requests. Please wait a few minutes before trying again.",
  "errors": {}
}
```

**500 Internal Error:**
```json
{
  "code": "internal_error",
  "message": "An unexpected error occurred",
  "errors": {}
}
```

---

## Error Code Registry

### Authentication & Authorization
| Code | HTTP Status | Description |
|------|-------------|-------------|
| `unauthorized` | 401 | Missing/invalid/expired session |
| `invalid_request` | 400 | Malformed auth request (missing token) |
| `missing_email` | 400 | Email field required |
| `missing_token` | 400 | Magic link token required |

### Validation
| Code | HTTP Status | Description |
|------|-------------|-------------|
| `invalid_uuid` | 400 | UUID format invalid |
| `invalid_tenant` | 400 | Tenant ID required/format invalid |

### Resources
| Code | HTTP Status | Description |
|------|-------------|-------------|
| `not_found` | 404 | Resource not found |

### Rate Limiting
| Code | HTTP Status | Description |
|------|-------------|-------------|
| `rate_limit_exceeded` | 429 | Too many requests |

### System
| Code | HTTP Status | Description |
|------|-------------|-------------|
| `internal_error` | 500 | Unexpected server error |
| `database_unavailable` | 503 | PostgreSQL unreachable |

---

## Layer Responsibilities

### 1. HTTP Handlers (Delivery Layer)
**File pattern:** `*_handler.go`

- **Only place** that constructs HTTP responses
- Maps service errors → canonical JSON via `mapServiceError()` / `mapAuthError()`
- **Never** exposes internal error details to clients
- Returns 500 with `internal_error` for unmapped errors

```go
func mapServiceError(c fiber.Ctx, err error) error {
    switch {
    case errors.Is(err, services.ErrInvalidUUID):
        return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
            "code": "invalid_uuid", "message": "invalid user id",
            "errors": fiber.Map{"id": []string{"must be a valid UUID"}},
        })
    case errors.Is(err, services.ErrNotFound):
        return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
            "code": "not_found", "message": "user not found", "errors": fiber.Map{},
        })
    default:
        // Log internally, return sanitized 500
        logger.Error("unhandled service error", zap.Error(err))
        return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
            "code": "internal_error", "message": "An unexpected occurred", "errors": fiber.Map{},
        })
    }
}
```

### 2. Services (Business Logic Layer)
**File pattern:** `*_service.go`

- **Never** constructs HTTP responses
- Returns **sentinel errors** (`ErrNotFound`, `ErrInvalidUUID`, `ErrTenantRequired`)
- Wraps database errors with context: `fmt.Errorf("service.method: %w", err)`
- Uses `context.Context` for cancellation/timeout

```go
var (
    ErrTenantRequired = errors.New("tenant required")
    ErrInvalidUUID    = errors.New("invalid uuid")
    ErrNotFound       = errors.New("not found")
)

func (s *userService) GetByID(ctx context.Context, tenantID, id string) (User, error) {
    parsed, err := uuid.Parse(id)
    if err != nil {
        return User{}, fmt.Errorf("%w: id", ErrInvalidUUID)  // wrap with field context
    }
    // ...
}
```

### 3. Database Layer
**Files:** `database/*.go`, `internal/database/sqlc/*.go`

- Uses `pgx` errors; wraps with operation context
- **Never** handles HTTP concerns
- Transaction wrappers (`WithTenantTx`, `WithTx`) use **named return + deferred rollback** pattern:

```go
func WithTenantTx(ctx, pool, logger, tenantID, fn) (err error) {
    tx, err := pool.BeginTx(ctx, pgx.TxOptions{...})
    if err != nil {
        return fmt.Errorf("database.WithTenantTx: begin: %w", err)
    }

    defer func() {
        if rbErr := tx.Rollback(ctx); rbErr != nil && !errors.Is(rbErr, pgx.ErrTxClosed) {
            logger.Error("rollback failed",
                zap.String("tenant_id", tenantID),
                zap.String("original_error", errString(err)),
                zap.String("rollback_error", rbErr.Error()),
            )
            if err == nil {
                err = fmt.Errorf("database.WithTenantTx: rollback: %w", rbErr)
            }
        }
    }()

    if _, err := tx.Exec(ctx, "SET LOCAL app.current_tenant_id = $1", tenantID); err != nil {
        return fmt.Errorf("database.WithTenantTx: set tenant: %w", err)
    }

    if err := fn(ctx, tx); err != nil {
        return fmt.Errorf("database.WithTenantTx: callback: %w", err)
    }

    return tx.Commit(ctx)
}
```

### 4. Middleware
**Files:** `internal/api/middleware/*/`

- Session middleware: returns 401 with canonical JSON on any validation failure
- Rate limit middleware: returns 429 with canonical JSON; **fail-open** on Redis error
- Request ID middleware: never returns errors

---

## The Three Forbidden Patterns

### ❌ 1. Empty Catch / Silent Drop
```go
// BAD
if err != nil { }
// BAD
catch (e) {}
```
**Rule:** Every error must be returned with context OR logged and acted upon.

### ❌ 2. Log-and-Return (Duplicate Logging)
```go
// BAD
logger.Error("failed", zap.Error(err))
return err  // Will be logged again upstream
```
**Rule:** Log **once** at the boundary (handler/worker). Intermediate layers only wrap and return.

### ❌ 3. Silent `_` Assignment
```go
// BAD
_ = someFunc()  // Discards error without action
```
**Rule:** In non-test code, never discard errors. Use `_ =` only when error is intentionally ignored (documented).

---

## Error Wrapping Convention

Use `fmt.Errorf` with `%w` for wrapping, adding **operation context** at each layer:

```
database layer:     "database.WithTenantTx: set tenant context: %w"
service layer:      "userService.GetByID: %w"
handler layer:      (maps to canonical JSON, no further wrapping)
```

This produces a readable chain:
```
userService.GetByID: database.WithTenantTx: set tenant context: pgx: syntax error
```

---

## Sentinel Errors

Defined in `internal/services/errors.go` (or service files):

```go
var (
    ErrTenantRequired = errors.New("tenant required")
    ErrInvalidUUID    = errors.New("invalid uuid")
    ErrNotFound       = errors.New("not found")
)
```

- **Immutable** — never wrap these, only return them directly
- Checked via `errors.Is(err, services.ErrNotFound)` in handlers
- Allow callers to distinguish error classes without string parsing

---

## Logging Errors

### When to Log
- **At the boundary** where the error is acted upon (handler, worker, cron)
- **Never** in intermediate wrapping layers

### What to Log
```go
logger.Error("operation failed",
    zap.String("operation", "GetUserByID"),
    zap.String("user_id", userID),
    zap.String("tenant_id", tenantID),
    zap.Error(err),           // full error chain via %w
    zap.String("request_id", middleware.GetRequestID(c)),
)
```

### What NOT to Log
- Internal error details in HTTP responses
- Raw session tokens, passwords, PII
- Stack traces in production (use `zap.Error` which includes stack in development)

---

## HTTP Error Handler (Fiber)

Configured in `cmd/api/main.go`:

```go
ErrorHandler: func(c fiber.Ctx, err error) error {
    if errors.Is(err, fiber.ErrNotFound) || err.Error() == "Not Found" {
        return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
            "code": "not_found", "message": "Resource not found", "errors": fiber.Map{},
        })
    }

    logger.Error("unhandled error", zap.Error(err), zap.String("env", cfg.Environment))

    return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
        "code": "internal_error", "message": "An unexpected error occurred", "errors": fiber.Map{},
    })
}
```

- Catches panics and unhandled errors
- Normalizes 404 to canonical format
- Logs full error with request context internally
- Returns sanitized 500 to client

---

## Testing Errors

### Unit Tests (Service Layer)
```go
func TestGetByID_InvalidUUID(t *testing.T) {
    _, err := svc.GetByID(ctx, "tenant-1", "not-a-uuid")
    assert.ErrorIs(t, err, services.ErrInvalidUUID)
}
```

### Integration Tests (HTTP Layer)
```go
func TestGetByID_Unauthorized(t *testing.T) {
    resp := sendRequest(app, "GET", "/api/users/123", nil)  // no cookie
    assert.Equal(t, fiber.StatusUnauthorized, resp.StatusCode)
    assert.Equal(t, "unauthorized", readJSONField(resp, "code"))
}
```

### Error Mapping Tests
```go
func TestMapServiceError_NotFound(t *testing.T) {
    err := mapServiceError(c, services.ErrNotFound)
    assert.Equal(t, fiber.StatusNotFound, resp.StatusCode)
    assert.Equal(t, "not_found", body["code"])
}
```

---

## Client-Side Handling Guide

### By Error Code
```javascript
switch (error.code) {
    case 'unauthorized':
        // Redirect to login, clear local session
        break;
    case 'rate_limit_exceeded':
        // Show retry-after timer, disable submit button
        break;
    case 'invalid_uuid':
    case 'invalid_tenant':
        // Show field-level validation from error.errors
        break;
    case 'not_found':
        // Show "not found" UI, offer create action
        break;
    case 'internal_error':
    case 'database_unavailable':
        // Show generic "try again later", log to error tracking
        break;
}
```

### Retry Logic
| Code | Retry | Backoff |
|------|-------|---------|
| `rate_limit_exceeded` | Yes | `Retry-After` header (seconds) |
| `database_unavailable` | Yes | Exponential (2s, 4s, 8s...) |
| `internal_error` | Yes | Exponential |
| `unauthorized` | No | Redirect to login |
| `not_found` | No | N/A |

---

## Migration from Legacy Patterns

If you encounter old code violating these rules:

| Legacy Pattern | Fix |
|----------------|-----|
| `log.Error(err); return err` | Remove log, let handler log |
| `return c.Status(500).JSON(fiber.Map{"error": err.Error()})` | Map to canonical, return sanitized |
| `if err != nil { return err }` without wrap | Add `fmt.Errorf("context: %w", err)` |
| `_ = db.Query(...)` | Handle error or document intentional ignore |

---

## Quick Reference: Error Flow

```
┌─────────────┐     ┌─────────────┐     ┌──────────────┐     ┌─────────────┐
│  Database   │────▶│  Service    │────▶│   Handler    │────▶│   Client    │
│  (pgx err)  │     │ (sentinel)  │     │ (map to JSON)│     │ (code-based)│
└─────────────┘     └─────────────┘     └──────────────┘     └─────────────┘
       │                   │                    │                    │
  wrap with ctx      return sentinel        map to canonical      switch on
  "db.op: %w"        or wrap with ctx       log once at boundary   error.code
```