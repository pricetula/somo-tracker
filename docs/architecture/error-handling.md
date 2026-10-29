---
id: ARCH-ERROR-HANDLING
title: Error Handling & Request ID Contract
status: draft
depends_on: []
touches: [backend, frontend]
---

# Error Handling & Request ID Contract

## Overview

This document defines the canonical error response shape and request ID propagation for all API endpoints. It ensures consistent error handling across backend and frontend, enabling reliable debugging and user-facing error messages.

## Error Response Shape

All non-2xx HTTP responses **must** return this exact JSON shape:

```json
{
  "code": "BAD_REQUEST",
  "message": "human readable description",
  "errors": {
    "field_name": ["specific validation error"]
  },
  "request_id": "req_abc123xyz"
}
```

### Required Fields

| Field | Type | Description |
|-------|------|-------------|
| `code` | string | SCREAMING_SNAKE_CASE stable error code (see [Error Codes](#error-codes)) |
| `message` | string | Human-readable message for logging/user display |
| `errors` | object | Map of `field_name` → `string[]` for field-level validation errors. Empty object `{}` if none. |
| `request_id` | string | Unique request identifier for tracing (format: `req_` + 16-char random) |

### Current Implementation Gap

**Backend (current):**
```json
{"code":"INVALID_EMAIL","message":"invalid email address"}
```
Missing: `errors` map, `request_id`

**Target (after central handler):**
```json
{
  "code": "VALIDATION_FAILED",
  "message": "request validation failed",
  "errors": {
    "email": ["invalid email format"]
  },
  "request_id": "req_abc123xyz"
}
```

## Error Codes

| Code | HTTP Status | When to Use |
|------|-------------|-------------|
| `BAD_REQUEST` | 400 | Malformed request body, invalid JSON |
| `VALIDATION_FAILED` | 422 | Request body parsed but field validation failed |
| `UNAUTHENTICATED` | 401 | Missing/invalid/expired session cookie |
| `FORBIDDEN` | 403 | Authenticated but insufficient permissions |
| `NOT_FOUND` | 404 | Resource doesn't exist or not in tenant scope |
| `CONFLICT` | 409 | Idempotency key conflict, duplicate resource |
| `RATE_LIMITED` | 429 | Too many requests (include `Retry-After` header) |
| `INTERNAL_ERROR` | 500 | Unexpected server error (log full details, return generic message) |

## Request ID Propagation

### Generation
- Generated at **edge middleware** (first middleware in chain)
- Format: `req_` + 16-char URL-safe base64 (128-bit entropy)
- Stored in `c.Locals("request_id")` and `context.Context`

### Propagation
```
Request → Middleware generates request_id
  → Attached to all log entries (slog)
  → Passed to service layer via context
  → Returned in error response
  → Frontend includes in error reports
```

### Backend Implementation

```go
// In middleware (platform/httpserver/server.go or dedicated middleware)
func RequestIDMiddleware() fiber.Handler {
    return func(c fiber.Ctx) error {
        // Check for incoming request ID (e.g., from load balancer)
        requestID := c.Get("X-Request-ID")
        if requestID == "" {
            // Generate new
            b := make([]byte, 16)
            rand.Read(b)
            requestID = "req_" + base64.RawURLEncoding.EncodeToString(b)
        }
        
        // Set in locals for handlers
        c.Locals("request_id", requestID)
        
        // Add to response header for debugging
        c.Set("X-Request-ID", requestID)
        
        // Add to context for service layer
        ctx := context.WithValue(c.Context(), "request_id", requestID)
        c.SetContext(ctx)
        
        return c.Next()
    }
}
```

### Central Error Handler

```go
// In platform/httpserver/server.go
func customErrorHandler(c fiber.Ctx, err error) error {
    requestID := c.Locals("request_id")
    if requestID == nil {
        requestID = "req_unknown"
    }
    
    // Default values
    code := "INTERNAL_ERROR"
    status := fiber.StatusInternalServerError
    message := "Internal Server Error"
    errors := map[string][]string{}
    
    // Handle known error types
    switch e := err.(type) {
    case *fiber.Error:
        status = e.Code
        code = statusToCode(status)
        message = e.Message
    case *apperr.Error:
        status = codeToStatus(e.Code)
        code = string(e.Code)
        message = e.Message
    case validator.ValidationErrors:
        status = fiber.StatusUnprocessableEntity
        code = "VALIDATION_FAILED"
        message = "request validation failed"
        errors = formatValidationErrors(e)
    }
    
    // Log with request_id
    slog.Error("request error",
        "request_id", requestID,
        "code", code,
        "status", status,
        "path", c.Path(),
        "err", err,
    )
    
    return c.Status(status).JSON(fiber.Map{
        "code":       code,
        "message":    message,
        "errors":     errors,
        "request_id": requestID,
    })
}
```

### Frontend Integration

```typescript
// src/lib/api/client.ts
interface ApiErrorResponse {
  code: string;
  message: string;
  errors?: Record<string, string[]>;
  request_id?: string;
}

export async function apiFetch<T>(path: string, init?: RequestInit): Promise<T> {
  const res = await fetch(`/api/v1${path}`, {
    headers: { 'Content-Type': 'application/json', ...init?.headers },
    credentials: 'include',
    ...init,
  });
  
  if (!res.ok) {
    const body: ApiErrorResponse = await res.json().catch(() => ({}));
    // Attach request_id for debugging
    const requestId = res.headers.get('X-Request-ID') || body.request_id;
    throw { 
      status: res.status, 
      code: body.code, 
      message: body.message, 
      errors: body.errors,
      requestId 
    };
  }
  
  if (res.status === 204) return undefined as unknown as T;
  return res.json() as Promise<T>;
}
```

```typescript
// src/lib/errors.ts
export function getErrorMessage(err: unknown): string {
  if (typeof err === 'object' && err !== null && 'message' in err) {
    const e = err as { message?: string; code?: string; errors?: Record<string, string[]>; requestId?: string };
    if (e.errors) {
      // Flatten field errors for display
      return Object.entries(e.errors)
        .map(([field, msgs]) => `${field}: ${msgs.join(', ')}`)
        .join('; ');
    }
    return e.message || 'Unknown error';
  }
  return 'Unknown error';
}
```

## Logging Integration

All log entries must include `request_id` for traceability:

```go
slog.Info("request started",
    "request_id", requestID,
    "method", c.Method(),
    "path", c.Path(),
)

slog.Error("request failed",
    "request_id", requestID,
    "code", code,
    "status", status,
    "err", err,
)
```

## Implementation Plan

| Phase | Task | Status |
|-------|------|--------|
| 1 | Add RequestIDMiddleware to server | ⏳ |
| 2 | Implement customErrorHandler with full shape | ⏳ |
| 3 | Update all handlers to use apperr.Error types | ⏳ |
| 4 | Add validation error formatting (validator → errors map) | ⏳ |
| 5 | Frontend: update apiFetch to surface request_id | ⏳ |
| 6 | Frontend: error boundary shows request_id for support | ⏳ |
| 7 | Add integration test for error shape + request_id | ⏳ |

## Related

- [Error Contract Skill](.pi/skills/error-contract/SKILL.md)
- [Backend AGENTS.md BE-025](backend/AGENTS.md#api-contract)
- [ADR-0002 Auth and Roles](adr/0002-auth-and-roles.md)