# OWASP API Security Top 10 (2023) — Reference for Somotracker

> **Purpose:** Concrete verification controls for each category, mapped to Somotracker's Go/Fiber backend patterns.
> **Usage:** Referenced by `.pi/skills/security-audit/SKILL.md` Step 2.

---

## API1:2023 — Broken Object Level Authorization (BOLA)

**Risk:** Attackers manipulate object IDs to access unauthorized resources.

### Somotracker Patterns to Verify

| Check | Pass Criteria | Fiber Code Pattern |
|-------|---------------|---------------------|
| Every `GET /api/v1/:resource/:id` filters by ownership | Handler calls service with `userID` from context, not just `id` param | `svc.Get(c.Locals("userID").(string), id)` |
| Every `PUT/PATCH/DELETE /api/v1/:resource/:id` validates ownership | Service returns `ErrNotFound` (not `ErrForbidden`) when user lacks access | `if obj.OwnerID != userID { return ErrNotFound }` |
| No endpoint accepts `user_id`/`owner_id` from request body for writes | Request DTOs exclude ownership fields; service injects from context | `type CreateReq struct { Name string `json:"name"` }` |
| List endpoints (`GET /api/v1/:resource`) scope to user | Query uses `WHERE owner_id = ?` or RLS policy | `db.Where("owner_id = ?", userID).Find(&items)` |

### Red Flags
- `WHERE id = ?` without ownership check
- `c.Params("id")` passed directly to DB without user context
- Admin checks only in UI, not handler (`if c.Locals("role") != "admin"` missing)

---

## API2:2023 — Broken Authentication

**Risk:** Compromised tokens, weak auth flows, credential stuffing.

### Somotracker Patterns to Verify

| Check | Pass Criteria | Fiber Code Pattern |
|-------|---------------|---------------------|
| JWT validation on all protected routes | Middleware parses/validates token; sets `userID`, `role` in `c.Locals()` | `auth.Required()` middleware on route group |
| Token expiry enforced (access ≤ 15min, refresh ≤ 7d) | `exp` claim checked; refresh rotation implemented | `jwt.ParseWithClaims(..., func(t *jwt.Token) (interface{}, error) { return secret, nil })` |
| No secrets in URL/query params | Tokens only in `Authorization: Bearer` header or secure `HttpOnly` cookie | `c.Cookie("access_token")` with `Secure: true, HttpOnly: true, SameSite: "Strict"` |
| Password reset/email change requires re-auth | Sensitive actions require current password or recent MFA | `svc.ChangePassword(userID, currentPass, newPass)` |
| Rate limit on auth endpoints | `POST /auth/login`, `/auth/register`, `/auth/refresh` limited (e.g., 5/min/IP) | `limiter.New(limiter.Config{Max: 5, Expiration: time.Minute})` |

### Red Flags
- `c.Get("user")` without middleware guarantee
- Long-lived access tokens (>1hr)
- Passwords logged or returned in responses

---

## API3:2023 — Broken Object Property Level Authorization (BPLA)

**Risk:** Over-fetching (reading sensitive fields) or mass assignment (writing protected fields).

### Somotracker Patterns to Verify

| Check | Pass Criteria | Fiber Code Pattern |
|-------|---------------|---------------------|
| Response DTOs separate from DB models | `UserResponse` excludes `password_hash`, `totp_secret`, `stripe_customer_id` | `type UserResponse struct { ID string `json:"id"` Email string `json:"email"` }` |
| Create/Update DTOs exclude internal fields | No `role`, `owner_id`, `created_at`, `deleted_at` in request structs | `type UpdateUserReq struct { Name *string `json:"name,omitempty"` }` |
| Field-level authorization for sensitive reads | `GET /users/me` returns more fields than `GET /users/:id` | Separate handlers or `if requesterID == targetID { return detailed }` |
| PATCH merges safely (no nil-overwrite) | Pointer fields + `omitempty` + service merges non-nil only | `if req.Name != nil { user.Name = *req.Name }` |

### Red Flags
- Returning `User` model directly: `return c.JSON(user)`
- `c.BodyParser(&user)` where `user` is DB model
- Admin-only fields writable by non-admin

---

## API4:2023 — Unrestricted Resource Consumption

**Risk:** DoS via unbounded pagination, large payloads, expensive queries.

### Somotracker Patterns to Verify

| Check | Pass Criteria | Fiber Code Pattern |
|-------|---------------|---------------------|
| Pagination hard limits | `limit` capped at 100; `offset`/`cursor` required for large sets | `limit := min(c.QueryInt("limit", 20), 100)` |
| Request body size limit | Fiber `BodyLimit` middleware (e.g., 1MB default, 10MB for uploads) | `app.Use(bodyparser.New(bodyparser.Config{Limit: 1024 * 1024}))` |
| Expensive operations rate-limited per user | Report generation, exports, analytics capped (e.g., 10/hr/user) | `limiter.New(limiter.Config{Key: userID, Max: 10, Expiration: time.Hour})` |
| Query complexity limits | No unbounded `JOIN`s, recursive CTEs, or `SELECT *` on wide tables | `db.Select("id, name, email").Where(...)` |
| Timeout on external calls | HTTP clients have `Timeout` (5-30s); circuit breaker for 3rd party | `http.Client{Timeout: 10 * time.Second}` |

### Red Flags
- `c.QueryInt("limit", 1000)` — no cap
- `db.Find(&items)` without `LIMIT`
- Synchronous PDF generation in request path

---

## API5:2023 — Broken Function Level Authorization (BFLA)

**Risk:** Regular users accessing admin/privileged endpoints.

### Somotracker Patterns to Verify

| Check | Pass Criteria | Fiber Code Pattern |
|-------|---------------|---------------------|
| Admin routes gated by middleware | Separate route group with `auth.RequireRole("admin")` | `admin := app.Group("/admin", auth.RequireRole("admin"))` |
| Role checks in handler (defense in depth) | Handler asserts `c.Locals("role") == "admin"` before mutating | `if c.Locals("role") != "admin" { return errors.Forbidden("admin required") }` |
| No "hidden" endpoints (security by obscurity) | All endpoints in OpenAPI spec; none rely on unknown paths | `swagger` docs generated from code |
| Tenant isolation in multi-tenant endpoints | `tenant_id` from context, not request; cross-tenant queries blocked | `db.Where("tenant_id = ?", c.Locals("tenantID"))` |

### Red Flags
- `if strings.HasPrefix(c.Path(), "/admin")` string checks
- Role only checked in frontend route guards
- `tenant_id` accepted from query param

---

## API6:2023 — Unrestricted Access to Sensitive Business Flows

**Risk:** Automatable flows (signup, invite, reward, checkout) abused at scale.

### Somotracker Patterns to Verify

| Check | Pass Criteria | Fiber Code Pattern |
|-------|---------------|---------------------|
| Signup/invite rate-limited per IP/device | `POST /auth/register`, `POST /invites` limited (5/hr/IP) | `limiter.New(limiter.Config{Key: c.IP(), Max: 5, Expiration: time.Hour})` |
| CAPTCHA/challenge on sensitive flows | Turnstile/hCaptcha on register, password reset, invite send | `if !verifyTurnstile(c.FormValue("cf-turnstile-response")) { return errors.BadRequest("captcha failed") }` |
| Idempotency keys for non-idempotent ops | `POST /payments`, `/invites` require `Idempotency-Key` header | `idempotency.CheckAndSet(c.Get("idempotency_key"), userID)` |
| Business logic limits (not just rate) | Max 5 active invites/user; max 100 teams/user | `if count > 5 { return errors.Conflict("invite limit reached") }` |
| Audit logging on sensitive actions | All mutations log `actor_id`, `action`, `target_id`, `timestamp` | `audit.Log(c.Locals("userID"), "invite.created", inviteID)` |

### Red Flags
- No CAPTCHA on public registration
- Invite codes guessable (short, sequential)
- No idempotency on payment/webhook endpoints

---

## API7:2023 — Server-Side Request Forgery (SSRF)

**Risk:** User-supplied URLs fetched by server, accessing internal metadata/cloud APIs.

### Somotracker Patterns to Verify

| Check | Pass Criteria | Fiber Code Pattern |
|-------|---------------|---------------------|
| No user-supplied URLs fetched directly | Webhooks, imports, avatars use allowlisted domains only | `if !allowlist.Contains(parseHost(url)) { return errors.BadRequest("domain not allowed") }` |
| Allowlist (not denylist) for outbound HTTP | Config-driven allowlist; deny by default | `var allowlist = []string{"api.github.com", "avatars.githubusercontent.com"}` |
| No `file://`, `gopher://`, `dict://` schemes | URL parser rejects non-http(s) schemes | `u, _ := url.Parse(input); if u.Scheme != "http" && u.Scheme != "https" { reject }` |
| Internal metadata endpoints blocked | `169.254.169.254`, `localhost`, `127.0.0.1`, `::1` blocked at HTTP client level | Custom `http.Transport` with `DialContext` that rejects private IPs |
| Timeout + size limit on fetches | Max 10s, max 5MB response | `http.Client{Timeout: 10*time.Second}` + `io.LimitReader(resp.Body, 5<<20)` |

### Red Flags
- `http.Get(userProvidedURL)` anywhere
- Webhook URL stored and called later without re-validation
- Image proxy fetches arbitrary URLs

---

## API8:2023 — Security Misconfiguration

**Risk:** Default configs, verbose errors, missing headers, exposed debug endpoints.

### Somotracker Patterns to Verify

| Check | Pass Criteria | Fiber Code Pattern |
|-------|---------------|---------------------|
| Security headers on all responses | `helmet` equivalent: CSP, HSTS, X-Frame-Options, Referrer-Policy | `app.Use(helmet.New(helmet.Config{ContentSecurityPolicy: "default-src 'self'"}))` |
| No debug/pprof endpoints in prod | `pprof`, `/debug`, `/healthz` (detailed) behind auth or disabled | `if config.Env == "production" { app.RemoveRoute("GET", "/debug/pprof/*") }` |
| Error responses don't leak stack traces | Custom error handler returns canonical error JSON (no `error` field with stack) | `app.Use(errorhandler.New(errorhandler.Config{Handler: customErrorHandler}))` |
| CORS restricted to known origins | No `Access-Control-Allow-Origin: *` with credentials | `cors.New(cors.Config{AllowOrigins: config.AllowedOrigins, AllowCredentials: true})` |
| TLS enforced | `app.Use(redirectToHTTPS)` in prod; HSTS header | `app.Use(redirectToHTTPS)` |

### Red Flags
- `app.Use(logger.New())` logging request bodies in prod
- `c.SendStatus(500)` with `err.Error()` in response
- `Access-Control-Allow-Origin: *` + `AllowCredentials: true`

---

## API9:2023 — Improper Inventory Management

**Risk:** Shadow APIs, deprecated versions, undocumented endpoints.

### Somotracker Patterns to Verify

| Check | Pass Criteria | Fiber Code Pattern |
|-------|---------------|---------------------|
| OpenAPI spec generated from code | `swag init` or `go-swagger` runs in CI; spec committed | `// @title Somotracker API @version 1.0 @BasePath /api/v1` |
| Every route in spec | No "hidden" routes; middleware routes documented | `router.Get("/health", health.Check).Name("health")` |
| Deprecated endpoints marked | `Deprecated: true` in spec; `Sunset` header on responses | `// @deprecated` in handler comment |
| Versioning strategy documented | `/api/v1/` in path; `Accept: application/vnd.somotracker.v1+json` supported | `v1 := app.Group("/api/v1")` |
| API inventory review quarterly | Automated script lists all routes vs. spec | `make api-inventory` |

### Red Flags
- Routes added without OpenAPI annotations
- `/api/internal/` or `/api/debug/` not in spec
- Multiple versions without deprecation policy

---

## API10:2023 — Unsafe Consumption of Third-Party APIs

**Risk:** Supply chain attacks, credential leakage, trusting external responses.

### Somotracker Patterns to Verify

| Check | Pass Criteria | Fiber Code Pattern |
|-------|---------------|---------------------|
| Third-party secrets in vault/env, not code | Stripe, SendGrid, GitHub tokens from `config.Secret()` | `stripe.Key = config.MustGet("STRIPE_SECRET_KEY")` |
| Outbound calls validate TLS certs | No `InsecureSkipVerify: true` | `http.Client{Transport: &http.Transport{TLSClientConfig: &tls.Config{}}}` |
| Response validation before use | JSON schema / struct validation on webhook payloads | `var evt stripe.Event; if err := json.Unmarshal(payload, &evt); err != nil { reject }` |
| Rate limit respected on external APIs | Backoff + retry with `Retry-After`; circuit breaker | `breaker.Execute(func() error { return httpClient.Do(req) })` |
| Webhook signatures verified | Stripe/GitHub/Slack signatures checked before processing | `webhook.VerifySignature(payload, sigHeader, secret)` |

### Red Flags
- `http.DefaultClient` used for external calls
- Webhook handler processes before signature verification
- Third-party API keys in repo history

---

## Quick Reference: Fiber Middleware Stack Order

```go
// Correct order (outer to inner):
app.Use(recover.New())           // 1. Panic recovery (no stack leaks)
app.Use(requestid.New())         // 2. Request ID for tracing
app.Use(logger.New(...))         // 3. Structured logging (no bodies in prod)
app.Use(helmet.New(...))         // 4. Security headers
app.Use(cors.New(...))           // 5. CORS (before auth)
app.Use(bodyparser.New(...))     // 6. Body parsing (size limits)
app.Use(limiter.New(...))        // 7. Global rate limit
app.Use(auth.Required())         // 8. Authentication (sets Locals)
app.Use(auth.RequireRole(...))   // 9. Authorization (per route group)
```

---

## Checklist Template for PR Reviews

```markdown
## Security Audit — [PR #xxx]

### API1 BOLA
- [ ] All object endpoints filter by ownership (userID from context)
- [ ] No ownership fields in request DTOs
- [ ] List endpoints scoped to user

### API2 Auth
- [ ] JWT middleware on all protected routes
- [ ] Token expiry ≤ 15min (access), ≤ 7d (refresh)
- [ ] Auth endpoints rate-limited

### API3 BPLA
- [ ] Response DTOs separate from models
- [ ] Create/Update DTOs exclude internal fields
- [ ] Field-level auth for sensitive reads

### API4 Consumption
- [ ] Pagination capped at 100
- [ ] Body limit configured (1MB default)
- [ ] Expensive ops rate-limited per user

### API5 BFLA
- [ ] Admin routes in separate group with role middleware
- [ ] Handler-level role checks (defense in depth)
- [ ] Tenant isolation uses context, not request

### API6 Business Flows
- [ ] Signup/invite rate-limited + CAPTCHA
- [ ] Idempotency keys on non-idempotent POSTs
- [ ] Business logic limits enforced
- [ ] Audit logging on mutations

### API7 SSRF
- [ ] No user-supplied URLs fetched directly
- [ ] Allowlist for outbound HTTP
- [ ] Private IPs blocked at transport level

### API8 Misconfig
- [ ] Security headers (CSP, HSTS, etc.)
- [ ] No debug endpoints in prod
- [ ] Error handler returns canonical JSON
- [ ] CORS restricted to known origins

### API9 Inventory
- [ ] OpenAPI spec updated
- [ ] All routes documented
- [ ] Deprecated endpoints marked

### API10 Third-Party
- [ ] Secrets from vault/env only
- [ ] TLS verification enabled
- [ ] Webhook signatures verified
- [ ] Circuit breaker on external calls
```

---

## Related Files

- `.pi/skills/security-audit/SKILL.md` — Workflow that uses this reference
- `backend/internal/api/errors.go` — Canonical error response format
- `backend/internal/middleware/auth.go` — Auth middleware patterns
- `docs/reference/backend_schema.md` — Database schema reference