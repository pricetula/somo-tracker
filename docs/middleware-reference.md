# Middleware Reference

## Overview

Somotracker's Go Fiber v3 backend uses a **chain of middleware** for cross-cutting concerns: request identification, rate limiting, and session authentication with multi-tenant RLS context injection.

All middleware follows the project's error handling contract — returning canonical JSON errors with `code`, `message`, `errors` fields.

---

## Middleware Stack Order

```go
app.Use(RequestIDMiddleware)          // 1. Request ID + logger injection
app.Use(SessionMiddleware)            // 2. Session validation + tenant context (protected routes only)
app.Use(RateLimitMiddleware)          // 3. Per-client rate limiting (applied per-route-group)
```

> **Note:** Session middleware is applied only to the protected route group (`/api/*`), not to public auth endpoints.

---

## 1. Request ID Middleware

**File:** `internal/api/middleware/request_id.go`  
**Constructor:** `NewRequestIDHandler(baseLogger *zap.Logger) fiber.Handler`

### Purpose
- Extracts or generates `X-Request-ID` (UUIDv4)
- Injects request-scoped `*zap.Logger` with `request_id` field
- Propagates via both Fiber locals and `context.Context`

### Behavior
| Input | Output |
|-------|--------|
| `X-Request-ID` header present | Echoed in response header |
| Header missing | New UUIDv4 generated, set in response |

### Injected Values
```go
// Fiber locals
c.Locals("request_id", string)
c.Locals("logger", *zap.Logger)

// context.Context (for sqlc, services)
ctx = context.WithValue(ctx, requestIDKey, requestID)
ctx = context.WithValue(ctx, loggerKey, logger)
```

### Helper Functions
```go
middleware.GetRequestID(c)   // → string (from locals or context)
middleware.GetLogger(c)      // → *zap.Logger (from locals or context)
middleware.RequestID(ctx)    // → string (from context)
middleware.Logger(ctx)       // → *zap.Logger (from context)
```

### Error Handling
- Never returns errors — always calls `c.Next()`
- Panic-safe: if UUID parsing fails, generates new one

---

## 2. Session Middleware (Auth + RLS)

**File:** `internal/api/middleware/session/session_middleware.go`  
**Constructor:** `NewSessionMiddleware(client *redis.Client, logger *zap.Logger) fiber.Handler`

### Purpose
- Validates opaque `session_token` cookie against Redis
- Injects `user_id`, `tenant_id`, `stytch_session_id` into `c.Locals`
- Enforces multi-tenant RLS by enabling tenant-scoped downstream handlers

### Cookie Contract
| Attribute | Value |
|-----------|-------|
| Name | `session_token` |
| Value | 64-char hex (256-bit opaque token) |
| HttpOnly | `true` |
| Secure | `true` |
| SameSite | `Lax` |
| Path | `/` |
| Max-Age | ~7 days (rolling) |

### Redis Session Key Format
```
session:{opaque_token}
```

### Session Data (JSON in Redis)
```json
{
  "user_id": "uuid",
  "tenant_id": "uuid",
  "stytch_session_id": "stytch-session-uuid",
  "expires_at": "2024-12-31T23:59:59Z"
}
```

### Validation Logic
1. **Missing/empty cookie** → 401, log `missing_session_cookie`
2. **Redis GET** on `session:{token}`
   - `redis.Nil` → 401, clear cookie, log `session_not_found`
   - Redis error → 401, log `redis_error`
3. **JSON unmarshal** → 401 on parse failure, log `session_parse_error`
4. **Expiration check** (`expires_at > now`)
   - Expired → 401, clear cookie, log `session_expired`
5. **Required fields** (`user_id`, `tenant_id` non-empty)
   - Missing → 401, log `session_invalid`
6. **Success** → Inject locals, refresh Redis TTL (6h), `c.Next()`

### Injected Locals
```go
c.Locals("user_id", string)           // authenticated user UUID
c.Locals("tenant_id", string)         // organization/tenant UUID
c.Locals("stytch_session_id", string) // Stytch session reference
```

### Error Response (All 401)
```json
{
  "code": "unauthorized",
  "message": "Unauthorized. Please log in to continue.",
  "errors": {}
}
```
> **Security:** No internal details, token values, or Redis keys leaked.

### Graceful Degradation
- `client == nil` → logs warning, passes through (test mode)

### Logging
All failures logged via `zap` with structured fields:
```go
logger.Warn("session middleware: unauthorized",
    zap.String("reason", "session_expired|session_not_found|..."),
    zap.String("request_id", c.Get("X-Request-ID")),
    zap.String("remote_addr", c.IP()),
    zap.String("method", c.Method()),
    zap.String("path", c.Path()),
    // ...context-specific fields
)
```

---

## 3. Rate Limit Middleware

**File:** `internal/api/middleware/ratelimit/ratelimit.go`  
**Module:** `ratelimit.Module` (Fx)  
**Constructor:** `NewRateLimitMiddleware(limiter *redis_rate.Limiter, rate redis_rate.Limit, keyPrefix string) fiber.Handler`

### Purpose
Redis-backed sliding-window rate limiting using `github.com/go-redis/redis_rate/v10`.

### Configuration
```go
// Fx provides singleton limiter
ratelimit.NewLimiter(redisClient) → *redis_rate.Limiter

// Per-route limits
ratelimit.NewRateLimitMiddleware(limiter, redis_rate.PerMinute(10), "api:auth:magic-link")
```

### Current Limits
| Route Group | Limit | Window | Key Prefix |
|-------------|-------|--------|------------|
| `POST /api/auth/magic-link/send` | 10 | 1 min | `api:auth:magic-link` |
| `GET /api/auth/callback` | 10 | 1 min | `api:auth:callback` |

### Client ID Derivation (Priority Order)
1. `c.FormValue("email")` — login/password-reset forms
2. `c.FormValue("target_email")` — multi-tenant recipient ops
3. `c.FormValue("user_id")` — authenticated user via form
4. `c.Get("X-User-ID")` — authenticated user via header
5. `c.IP()` — remote address (Fiber unwraps X-Forwarded-For, CF-Connecting-IP)
6. `"unknown"` — fallback

> **Note:** After session middleware runs, handlers can enrich the key by setting `X-User-ID` header before rate limit middleware (not currently used; auth routes are public).

### Response Headers (Always Set)
```
X-RateLimit-Limit: 10
X-RateLimit-Remaining: 9
X-RateLimit-Reset: 1703980800000  // milliseconds since epoch
```

### 429 Response
```json
{
  "code": "rate_limit_exceeded",
  "message": "Too many requests. Please wait a few minutes before trying again.",
  "errors": {}
}
```
```
Retry-After: 45  // seconds
```

### Redis Failure Behavior
- Redis error → **allow request** (fail-open), log error
- Prevents infrastructure outage from blocking legitimate traffic

### Logging
Blocked requests logged with:
```go
logger.Warn("ratelimit: request blocked",
    zap.String("client_id", clientID),
    zap.String("key", key),
    zap.String("limit", rate.String()),
    zap.Int("remaining", remaining),
    zap.Duration("reset_after", resetAfter),
    zap.Time("blocked_at", time.Now()),
)
```

---

## Middleware Integration in Router

```go
func (r *Router) RegisterRoutes(app *fiber.App, redisClient *redis.Client, logger *zap.Logger) {
    // Protected group with session middleware
    protected := app.Group("/api", session.NewSessionMiddleware(redisClient, logger))

    // Public auth routes (rate limited, no session)
    public := app.Group("/api/auth")
    public.Post("/magic-link/send",
        ratelimit.NewRateLimitMiddleware(r.limiter, authRate, "api:auth:magic-link"),
        r.Auth.sendMagicLink,
    )
    public.Get("/callback",
        ratelimit.NewRateLimitMiddleware(r.limiter, authRate, "api:auth:callback"),
        r.Auth.callback,
    )

    // Protected endpoints (session + rate limit if needed)
    protected.Get("/users/:id", r.User.getByID)
    protected.Get("/users/email/:email", r.User.getByEmail)
    protected.Get("/tenants/slug/:slug", r.Tenant.getBySlug)
}
```

---

## Protected Handler Pattern

Handlers extract tenant context from locals:

```go
func (h *userHandler) getByID(c fiber.Ctx) error {
    tenantID, _ := c.Locals("tenant_id").(string)
    userID, _ := c.Locals("user_id").(string)

    if tenantID == "" {
        return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
            "code": "unauthorized",
            "message": "tenant context missing",
            "errors": fiber.Map{},
        })
    }

    user, err := h.svc.GetByID(c.Context(), tenantID, userID)
    // ...
}
```

Service layer uses `database.WithTenantTx` to set `app.current_tenant_id` GUC.

---

## Testing Strategy

| Middleware | Test File | Coverage |
|------------|-----------|----------|
| Request ID | `request_id_test.go` | Header echo, generation, context propagation |
| Rate Limit | `ratelimit/ratelimit_test.go` | Allow/block, headers, client ID extraction |
| Session | `session/session_integration_test.go` | Unauthorized, valid flow, expiry, revocation, nil client, concurrent |

Run with:
```bash
go test ./internal/api/... -tags=integration
```

---

## Security Summary

| Middleware | Threat Mitigated |
|------------|------------------|
| Request ID | Traceability, correlation across logs/metrics/traces |
| Session | Session hijacking, token leakage, tenant isolation bypass |
| Rate Limit | Brute force, credential stuffing, DoS on auth endpoints |

### Defense-in-Depth
1. **HttpOnly cookies** — XSS cannot steal session tokens
2. **RLS at DB layer** — Even compromised app code cannot cross tenant boundaries
3. **Fail-closed session validation** — Invalid/expired/missing → 401
4. **Fail-open rate limiting** — Redis outage never blocks users
5. **Sanitized errors** — No internal details in client responses