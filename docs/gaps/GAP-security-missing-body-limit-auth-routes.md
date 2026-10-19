# Gap: Missing Request Body Size Limit on Public Auth Endpoints

**Area:** backend
**Severity:** High
**Status:** Open
**Owner:** 
**Created:** 2025-10-19
**Related Issues:** #security-audit-2025-10-19

## Description
Public authentication endpoints (`POST /api/auth/magic-link/send`, `GET /api/auth/callback`, `POST /api/auth/logout`) are registered directly on the Fiber app (not in the protected route group), so they may not inherit the global body size limit middleware. This allows oversized request bodies that could cause memory exhaustion.

## Current Behavior
In `backend/internal/api/router.go:33-55`, public auth routes are registered on `app` directly:
```go
app.Post("/api/auth/magic-link/send", ..., r.Auth.sendMagicLink)
app.Get("/api/auth/callback", ..., r.Auth.callback)
app.Post("/api/auth/logout", ..., r.Auth.logout)
```

The global body parser middleware (if configured) is typically added to the protected group or app-wide, but auth routes are added before middleware groups in some configurations.

The `sendMagicLink` handler reads from both form value and JSON body:
```go
email := strings.TrimSpace(c.FormValue("email"))
if email == "" {
    var body struct { Email string `json:"email"` }
    if err := c.Bind().Body(&body); err == nil {
        email = strings.TrimSpace(body.Email)
    }
}
```

## Expected Behavior
All HTTP endpoints (public and protected) must enforce a request body size limit (default 1MB per security standard).

## Impact
- **DoS Risk**: Attacker can send large payloads (10MB+) to auth endpoints, consuming memory/CPU
- **Resource Exhaustion**: JSON parsing of large bodies allocates memory
- **Compliance**: Violates OWASP API4:2023 (Unrestricted Resource Consumption) and project security standard

## Reproduction Steps
1. Send POST to `/api/auth/magic-link/send` with 5MB JSON body: `{"email": "x" + "a".repeat(5000000) + "@test.com"}`
2. Observe if request is rejected with 413 Payload Too Large
3. If not rejected, server processes large body → memory pressure

## Proposed Solutions
- **Option A (Global middleware)**: Ensure `bodyparser.New(bodyparser.Config{Limit: 1<<20})` is registered on `app` before any routes in `main.go` or `newFiberApp`
- **Option B (Per-route)**: Add body limit middleware to each public auth route explicitly
- **Option C (Route group)**: Create a `publicAuth` group with body limit + rate limit + CAPTCHA, then register routes there

## Decision
Pending — recommend Option A (global) for defense in depth, plus Option C for clarity.

## Acceptance Criteria
- [ ] POST `/api/auth/magic-link/send` with >1MB body returns 413 Payload Too Large
- [ ] POST `/api/auth/logout` with >1MB body returns 413
- [ ] Global body limit middleware registered in `main.go` / `newFiberApp` before route registration
- [ ] Integration test added for body limit on public auth endpoints
- [ ] All other public endpoints (webhooks, callbacks) audited for body limits

## Links
- Security audit report: `security-report-test.md`
- OWASP API4:2023 reference: `docs/reference/owasp-api-top10-2023.md#api42023---unrestricted-resource-consumption`
- Fiber body parser docs: https://docs.gofiber.io/api/middleware/bodyparser
- Router registration: `backend/internal/api/router.go:33-55`