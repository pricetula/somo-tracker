---
name: security-audit
description: Audit and harden REST/GraphQL/gRPC APIs for OWASP Top 10 vulnerabilities. Used before PR approval, during design, and for production deployment.

---

# Security Audit Workflow

## When to Run
This skill is triggered when:
- A PR touches API endpoints, auth middleware, or API gateway config
- User requests an "API security review", "pentest prep", "API audit", or similar
- Designing a new endpoint before implementation

## Step 1 — Threat Model the Endpoint
For each endpoint/resource in scope, answer:
1. **Object-level**: What entity is read/written, and by what ID?
2. **Authorization**: Who is supposed to call it, with what role?
3. **Worst-case**: What can an authenticated-but-unauthorized user do?
4. **External calls**: Does this endpoint call another service based on user input?
5. **Sensitive flows**: Is this a checkout/transfer/password-reset that could be abused?

### Fiber-Specific Threat Questions
6. **Context usage**: Does handler read `c.Locals("userID")` / `c.Locals("role")` / `c.Locals("tenantID")` set by auth middleware?
7. **Middleware order**: Is `auth.Required()` before route handler? (Check `backend/internal/routes/routes.go`)
8. **Error handling**: Does handler return canonical error via `errors.BadRequest()`, `errors.Forbidden()`, `errors.NotFound()`?
9. **DTO binding**: Does handler use dedicated request struct (not DB model) with `c.BodyParser(&req)`?

## Step 2 — OWASP API Top 10 Checklist
Walk through `reference/owasp-api-top10-2023.md` category by category:

| Category | Control to Verify | Fiber/Go Pattern to Check |
|----------|-------------------|---------------------------|
| **API1 — BOLA** | Every object-fetching endpoint filters by `owner_id`/scope, not just `is_logged_in` | `svc.Get(c.Locals("userID").(string), id)` — service receives userID from context |
| **API3 — BPLA** | Serializers respect field-level authorization (no mass assignment, no over-fetching) | Response DTOs (`UserResponse`) separate from DB models (`User`); request DTOs exclude `role`, `owner_id` |
| **API5 — BFLA** | Admin-only ops gated at handler level, not hidden from UI | Route group: `admin := app.Group("/admin", auth.RequireRole("admin"))` + handler check |
| **API4 — Unrestricted Consumption** | Rate limits, pagination caps, payload size limits on expensive operations | `limit := min(c.QueryInt("limit", 20), 100)`; `bodyparser.New(Config{Limit: 1<<20})` |
| **API6 — Unrestricted Flow** | Can this flow be automated/scripted at scale (scalping, spam)? | CAPTCHA on register/invite; idempotency keys on payments; business logic limits (max 5 invites/user) |
| **API7 — SSRF** | Any user-input URL/webhook is allowlisted, not denylisted | `allowlist.Contains(parseHost(url))`; custom `http.Transport` blocking private IPs |
| **API2 — Auth** | Proper auth flow (JWT/CSRF/RLS) for every endpoint | `auth.Required()` middleware sets `c.Locals("userID")`; token expiry ≤15min |
| **API8 — Misconfig** | Security headers, CSP, HSTS present | `helmet.New(Config{ContentSecurityPolicy: "default-src 'self'"})`; no debug routes in prod |
| **API9 — Inventory** | Endpoints documented in OpenAPI/schema | `swag` annotations on every handler; `make api-inventory` in CI |
| **API10 — Third-party** | External API calls validated for secrets/permissions | Secrets from `config.MustGet()`; webhook signatures verified; circuit breaker on external calls |

## Step 3 — Verify Runtime Behavior
- **Authorization tests**: For every object-scoped endpoint, assert user A cannot read/write user B's object by ID substitution
- **Schema/contract testing**: Use Schemathesis or similar to fuzz inputs against OpenAPI spec
- **Dynamic scanning**: Run OWASP ZAP against a running instance for misconfigurations
- **API inventory**: Confirm all endpoints are documented in the OpenAPI/GraphQL schema

### Fiber-Specific Runtime Checks
- **Middleware chain test**: `GET /api/v1/users/me` without token → 401; with token → 200 with user data
- **Role escalation test**: Regular user `POST /api/v1/admin/users` → 403 (not 404)
- **Tenant isolation test**: User in tenant A `GET /api/v1/teams?tenant_id=B` → empty/403
- **Rate limit test**: Burst 20 requests to `/auth/login` → 429 after limit
- **Error contract test**: All non-2xx responses match `{code, message, errors}` shape (see `backend/internal/api/errors.go`)
- **Body limit test**: POST 2MB JSON to default endpoint → 413

## Step 4 — Reporting
Structure findings as:
- **Critical** – BOLA, BFLA, SSRF, unrestricted flow
- **High** – Missing rate limits, missing auth on sensitive flows
- **Medium** – Missing security headers, incomplete input validation
- **Low** – Documentation gaps, minor misconfigurations

### Output Format: `security-report.md`
```markdown
# Security Audit Report — [PR #xxx / Feature: xxx]

**Date**: YYYY-MM-DD
**Auditor**: pi-agent (security-audit skill)
**Scope**: [endpoints/files reviewed]

## Findings

### Critical
- **[API1-BOLA]** `GET /api/v1/projects/:id` — Missing ownership check in `ProjectService.Get()`
  - **File**: `backend/internal/services/project.go:42`
  - **Fix**: Add `userID` parameter; filter by `owner_id = ?`

### High
- **[API4-Consumption]** `POST /api/v1/reports` — No rate limit, no pagination cap
  - **File**: `backend/internal/handlers/report.go:18`
  - **Fix**: Add `limiter.New(Config{Key: userID, Max: 10, Expiration: time.Hour})`

### Medium
- **[API8-Misconfig]** Missing CSP header on `/api/*` routes
  - **File**: `backend/main.go:35`
  - **Fix**: Add `helmet.New(Config{ContentSecurityPolicy: "default-src 'self'"})`

### Low
- **[API9-Inventory]** `GET /api/v1/health/detailed` not in OpenAPI spec
  - **File**: `backend/internal/handlers/health.go:22`
  - **Fix**: Add `// @Summary` annotations
```

## Integration Points
- Triggered by `/security` command
- Uses `pi.registerTool()` for security scanning tools (OWASP ZAP, etc.)
- Outputs findings to `security-report.md`
- Can be combined with `generate-api` skill for new endpoint hardening
- Runs automatically in CI on PRs touching `backend/internal/handlers/`, `backend/internal/routes/`, `backend/internal/middleware/auth.go`

## Prerequisites
- `reference/owasp-api-top10-2023.md` (exists in project)
- `reference/review-checklist.md` (exists in project)
- Access to running API instance for dynamic scanning

## Fiber/Go-Specific Reference Patterns

### Canonical Error Response (from `backend/internal/api/errors.go`)
```go
// All non-2xx responses MUST match this shape
type ErrorResponse struct {
    Code    string            `json:"code"`
    Message string            `json:"message"`
    Errors  map[string][]string `json:"errors"`  // never null, empty {} if none
}
```

### Auth Middleware Contract (`backend/internal/middleware/auth.go`)
```go
// Required() — validates JWT, sets Locals:
//   c.Locals("userID") = string
//   c.Locals("role") = string ("user" | "admin")
//   c.Locals("tenantID") = string (if multi-tenant)
// Returns 401 with code="unauthorized" if invalid/missing

// RequireRole(roles...) — checks c.Locals("role") against allowed
// Returns 403 with code="forbidden" if mismatch
```

### Rate Limiter Pattern (`backend/internal/middleware/ratelimit.go`)
```go
// Per-user: key = userID from context
// Per-IP: key = c.IP()
// Config: Max requests, Expiration window
limiter.New(limiter.Config{
    Key: func(c *fiber.Ctx) string { return c.Locals("userID").(string) },
    Max: 100,
    Expiration: time.Minute,
})
```

### Request DTO Pattern (prevents mass assignment)
```go
// GOOD: Dedicated request struct, pointers for optional fields
type UpdateProjectReq struct {
    Name        *string `json:"name,omitempty" validate:"omitempty,min=1,max=100"`
    Description *string `json:"description,omitempty"`
    // NO: OwnerID, Role, CreatedAt, DeletedAt
}

// BAD: Using DB model directly
// type UpdateProjectReq = models.Project
```

### Response DTO Pattern (prevents over-fetching)
```go
// GOOD: Explicit response struct, omits sensitive fields
type ProjectResponse struct {
    ID          string    `json:"id"`
    Name        string    `json:"name"`
    Description string    `json:"description"`
    OwnerID     string    `json:"owner_id"`
    CreatedAt   time.Time `json:"created_at"`
    // NO: DeletedAt, internal flags, Stripe IDs, etc.
}

// BAD: Returning DB model directly
// return c.JSON(project)  // leaks all fields
```

### SSRF-Safe HTTP Client (`backend/internal/httpclient/safe.go`)
```go
var SafeClient = &http.Client{
    Timeout: 10 * time.Second,
    Transport: &http.Transport{
        DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
            // Block private IPs (10.x, 172.16-31.x, 192.168.x, 127.x, 169.254.x, ::1)
            if isPrivateIP(addr) {
                return nil, errors.New("blocked: private IP")
            }
            return (&net.Dialer{Timeout: 5 * time.Second}).DialContext(ctx, network, addr)
        },
        TLSClientConfig: &tls.Config{InsecureSkipVerify: false}, // Enforce TLS
    },
}
```