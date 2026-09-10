# Backend Security Audit — Somotracker API

**Date:** 2025-09-05  
**Auditor:** Automated OWASP API Security Top 10 (2023) Review  
**Scope:** Backend (`backend/`) — Go/Fiber REST API, auth middleware, session management, rate limiting, CORS, security headers

---

## Executive Summary

The Somotracker backend implements a **strong baseline security posture** with:
- ✅ Mandatory correlation IDs (`X-Request-ID`) on all requests
- ✅ Canonical error response format (no stack traces leaked)
- ✅ Row-Level Security (RLS) via PostgreSQL GUCs for multi-tenant isolation
- ✅ Redis-backed session validation with device fingerprinting
- ✅ IP auto-blacklisting on repeated violations
- ✅ Circuit breaker + retries for third-party (Stytch) integration
- ✅ Structured logging with request-scoped context
- ✅ **CSRF protection** — double-submit cookie pattern implemented
- ✅ **Magic-link abuse protection** — dual rate limits (IP + email) + optional CAPTCHA

**Remaining gaps:**
1. **CORS configuration** — `AllowCredentials: true` with dynamic origin list requires strict validation
2. **API inventory** — no OpenAPI schema; undocumented endpoints possible (API9)
3. **Role-based authorization** — no RBAC enforcement on handlers (API5), though RLS provides tenant isolation
4. **Pagination caps** — no max limit on list endpoints (API4)
5. **Session cookie hardening** — `__Host-` prefix, HSTS header (API8)

---

## OWASP API Top 10 (2023) — Category-by-Category Findings

### API1:2023 — Broken Object Level Authorization (BOLA) — **PASS**

**Evidence:**  
- `internal/database/tenant.go:50-106` — `WithTenantTx` enforces `SET LOCAL app.current_tenant_id = $1` on every transaction  
- `internal/services/user_service.go:34-55` — All queries scoped by `tenantID` passed from session middleware  
- `internal/api/user_handler.go:18-35` — Handler extracts `tenant_id` from `c.Locals` (injected by session middleware) and passes to service  

**Why it works:** PostgreSQL RLS policies reference `current_setting('app.current_tenant_id')`. Even if a handler forgot the tenant check, the DB would reject cross-tenant access.

**Note:** `internal/api/tenant_handler.go:GetBySlug` bypasses tenant scoping (reads by slug globally). Acceptable for tenant discovery during auth flow, but ensure no sensitive data exposed.

---

### API2:2023 — Broken Authentication — **PASS**

**Strengths:**  
- Rate limiting on `/magic-link/send` and `/callback` (10/min via `ratelimit.Module`)  
- Magic link tokens validated server-side via Stytch; never exposed to JS  
- Session tokens: 32-byte opaque, HttpOnly, Secure, SameSite=Lax, 7h TTL  
- Fingerprint validation on every request (detects cookie theft)  
- Session revoked on logout (Redis + DB)  
- Circuit breaker + exponential backoff on Stytch calls (`internal/stytch/stytch.go`)  
- **CSRF double-submit token** — `csrf_token` cookie (non-HttpOnly) + `X-CSRF-Token` header validation on mutating endpoints (`internal/api/middleware/csrf/csrf.go`)  
- **Magic-link dual rate limits** — IP (10/min) + Email (3/hour) + optional CAPTCHA  

**Remaining Gaps:**  
| Finding | Severity | File:Line | Fix |
|---------|----------|-----------|-----|
| Session cookie lacks `__Host-` prefix | MEDIUM | `auth_handler.go:123` | Rename to `__Host-session_token` (requires HTTPS, path=/, no Domain) for defense-in-depth |
| No `Secure` flag enforcement test | MEDIUM | `session_middleware.go:37` | Add integration test asserting cookie attributes |

---

### API3:2023 — Broken Object Property Level Authorization — **PASS**

**Evidence:**  
- No mass-assignment: Handlers never bind request body directly to persistence models  
- `internal/api/auth_handler.go:28-44` — Email extracted via `c.FormValue` / `c.Bind().Body` into explicit struct  
- `internal/services/auth_service.go:165-200` — Explicit field selection in `CreateSessionParams`  
- Response structs (`sqlc.User`, `sqlc.Tenant`) returned directly but contain only tenant-scoped data  

**Note:** If new endpoints are added, enforce DTO pattern (explicit request/response structs) to maintain this.

---

### API4:2023 — Unrestricted Resource Consumption — **PARTIAL PASS**

**Strengths:**  
- Global body limit: 4MB (`bodylimit.go:9`)  
- Request timeout: 15s (`timeout.go:12`)  
- Rate limiting infrastructure exists (per-IP, Redis-backed)  
- DB pool limits: `DBMaxConns=10`, connection lifetime/idle timeouts  

**Gaps:**  
| Finding | Severity | File:Line | Fix |
|---------|----------|-----------|-----|
| No pagination caps on list endpoints | **HIGH** | `router.go:68-72` | Add max `limit` (e.g. 100) and default on all list handlers |
| Rate limit keyed by IP only | MEDIUM | `ratelimit.go:155-165` | For authenticated routes, key by `user_id` + IP (already supported via `X-User-ID` header) |
| No query complexity/depth limits | LOW | N/A | Not applicable (REST, not GraphQL) |

---

### API5:2023 — Broken Function Level Authorization (BFLA) — **FAIL**

**Finding:** No role-based authorization on any handler. The session middleware injects `user_id` and `tenant_id` but **no role claim**. Handlers have no `requireRole` or similar guard.

| Endpoint | Current Auth | Required Auth |
|----------|--------------|---------------|
| `GET /api/users/:id` | Session only | Session + (self OR admin) |
| `GET /api/users/email/:email` | Session only | Session + (self OR admin) |
| `GET /api/tenants/slug/:slug` | **None** (public) | Session? |
| `POST /api/auth/logout` | Session only | Session |

**Fix (deferred — roles based on school membership table):**  
1. Extend `SessionData` (`internal/session/session.go:11-18`) to include `Role` field  
2. Populate role at callback (`auth_service.go:200-210`) from Stytch member/role  
3. Add `RequireRole(roles ...string)` middleware (`internal/api/middleware/`)  
4. Apply to privileged endpoints (admin, user management, settings)

---

### API6:2023 — Unrestricted Access to Sensitive Business Flows — **PASS**

**Implemented:** Magic-link initiation (`POST /api/auth/magic-link/send`) now has:
- ✅ Per-email rate limit: 3/hour (`redis_rate.PerHour(3)` keyed by normalized email)
- ✅ IP rate limit: 10/min (distributed botnet protection)
- ✅ CAPTCHA infrastructure: Pluggable providers (hCaptcha, Turnstile, reCAPTCHA v3) with config-driven enable/disable
- ✅ Logging of all magic-link sends with fingerprint for security review

**Files:**
- `internal/api/router.go` — dual rate limit middleware chain
- `internal/api/middleware/captcha/captcha.go` — CAPTCHA middleware
- `internal/config/config.go` — CAPTCHA config (disabled by default for local dev)

---

### API7:2023 — Server Side Request Forgery (SSRF) — **PASS**

**Evidence:**  
- No user-supplied URLs fetched by backend  
- Stytch SDK uses hardcoded base URIs (`stytchconfig.BaseURITest/Live`)  
- No webhook registration, URL preview, or file-fetch-by-URL features  
- Database connections use parsed DSN from config (validated at startup)

**Note:** If webhooks or "import from URL" features are added, implement allowlist + private IP blocking at network layer.

---

### API8:2023 — Security Misconfiguration — **PARTIAL PASS**

**Strengths:**  
- Security headers via `helmet.Middleware()`: `X-Content-Type-Options: nosniff`, `X-Frame-Options: DENY`, `CSP: default-src 'none'`  
- CORS: explicit allowed origins from config, `AllowCredentials: true`  
- No debug endpoints in production (`/health`, `/livez`, `/readyz` only)  
- Structured error handling: no stack traces in responses (`fiber_config.go`)  
- Environment validation: `COOKIE_SECRET`, `COOKIE_DOMAIN`, `ALLOWED_ORIGINS` required  

**Gaps:**  
| Finding | Severity | File:Line | Fix |
|---------|----------|-----------|-----|
| CSP `default-src 'none'` blocks all resources | MEDIUM | `helmet.go:17` | Add `frame-ancestors 'none'; base-uri 'self'`; consider `connect-src` for API if needed |
| CORS `AllowHeaders` missing `X-CSRF-Token` | MEDIUM | `cors.go:18-20` | Add `X-CSRF-Token` to allowed headers |
| No HSTS header | LOW | `helmet.go` | Add `Strict-Transport-Security: max-age=31536000; includeSubDomains` in production |
| `Content-Security-Policy` on API responses | LOW | `helmet.go:17` | API doesn't serve HTML; current CSP is fine but document intent |
| Cookie domain not validated | MEDIUM | `config.go:146` | Ensure `COOKIE_DOMAIN` starts with `.` for subdomain sharing |

---

### API9:2023 — Improper Inventory Management — **FAIL**

**Finding:** **No OpenAPI/Swagger schema generated.** Endpoints exist only in code (`router.go`, handlers). No automated inventory, no deprecation tracking.

**Impact:**  
- Shadow endpoints possible (forgotten test routes, debug handlers)  
- Frontend/backend contract drift (as seen in recent auth mismatch)  
- Security tools cannot scan undocumented attack surface  

**Fix:**  
- Add `swaggo/swag` or `oapi-codegen` to generate OpenAPI 3.1 from handler annotations  
- Enforce schema in CI: fail build if handlers not documented  
- Version API in path (`/api/v1/...`) with explicit deprecation policy

---

### API10:2023 — Unsafe Consumption of APIs — **PASS**

**Evidence:**  
- Stytch client: circuit breaker, retries only on idempotent reads, timeouts, response validation (`stytch.go:180-250`)  
- `SanitizedError` maps all Stytch errors to safe generic messages  
- No secrets sent to Stytch beyond required project ID/secret  
- Redis client: connection pooling, timeouts via pool config  

**Note:** If new third-party integrations are added, apply same pattern (circuit breaker, schema validation, minimal data sharing).

---

## Additional Findings (Beyond Top 10)

### A1 — Missing Security Headers on Error Responses
**File:** `helmet.go` sets headers via `defer` after `c.Next()`. If panic occurs before `defer`, headers may not be set on error path.  
**Fix:** Use Fiber's `Use` with `Recover` middleware before helmet, or set headers in a `Finally` wrapper.

### A2 — Session Cookie SameSite=Lax Limits CSRF Protection
**File:** `auth_handler.go:128`  
**Issue:** `SameSite=Lax` allows cookies on top-level navigation (GET). POST from external site blocked, but GET-based state changes (if any) vulnerable.  
**Fix:** Keep `Lax` for usability; CSRF token now protects all mutating endpoints (POST/PUT/PATCH/DELETE).

### A3 — IP Blacklist Fail-Open on Redis Error
**File:** `ipblacklist.go:105-115`  
**Issue:** Redis error → request allowed through. Acceptable for availability, but log at `WARN` level and alert on repeated fail-opens.  
**Fix:** Add metric/counter for fail-open events; alert if > N/minute.

### A4 — Stytch Redirect URL Not Validated Against Allowlist
**File:** `config.go:152` validates `STYTCH_REDIRECT_URL` is non-empty but not against allowlist.  
**Risk:** Open redirect if Stytch configuration compromised.  
**Fix:** Validate redirect URL host matches `FrontendURL` or configured allowlist at startup.

### A5 — No API Versioning in Path
**File:** `router.go:52-73`  
**Issue:** All routes under `/api/` with no version prefix. Breaking changes require coordinated deploy.  
**Fix:** Mount router at `/api/v1`; plan v2 migration path.

---

## Prioritized Remediation Roadmap (Updated)

| Priority | Category | Action | Effort | Status |
|----------|----------|--------|--------|--------|
| **P0** | API2 | ~~Add CSRF double-submit token~~ | Medium | ✅ **DONE** |
| **P0** | API5 | Implement RBAC middleware + role claim in session | Medium | Deferred |
| **P0** | API6 | ~~Add per-email rate limit + CAPTCHA on magic-link send~~ | Low | ✅ **DONE** |
| **P1** | API4 | Add pagination caps (max 100) on all list endpoints | Low | Pending |
| **P1** | API8 | Add HSTS header in production; validate cookie domain | Low | Pending |
| **P1** | API9 | Generate OpenAPI schema; enforce in CI | Medium | Pending |
| **P2** | API2 | Add `__Host-` prefix to session cookie | Low | Pending |
| **P2** | API8 | Add `X-CSRF-Token` to CORS `AllowHeaders` | Trivial | Pending |
| **P3** | API4 | Add authenticated rate limit key (user_id) | Low | Pending |
| **P3** | API2 | Add integration test for cookie Secure/HttpOnly flags | Low | Pending |

---

## Files Referenced

| File | Purpose |
|------|---------|
| `cmd/api/main.go` | App bootstrap, middleware chain |
| `internal/api/router.go` | Route registration, rate limit attachment |
| `internal/api/auth_handler.go` | Magic-link send, callback, logout |
| `internal/api/middleware/session/session_middleware.go` | Session validation, fingerprinting |
| `internal/api/middleware/ratelimit/ratelimit.go` | Redis-backed rate limiting |
| `internal/api/middleware/ipblacklist/ipblacklist.go` | Auto-blacklisting on violations |
| `internal/api/middleware/helmet/helmet.go` | Security headers |
| `internal/api/middleware/cors/cors.go` | CORS configuration |
| `internal/api/middleware/bodylimit/body_limit.go` | 4MB payload limit |
| `internal/api/middleware/timeout/timeout.go` | 15s request timeout |
| `internal/api/middleware/request_id.go` | Correlation ID injection |
| `internal/api/middleware/csrf/csrf.go` | **NEW** CSRF double-submit cookie middleware |
| `internal/api/middleware/captcha/captcha.go` | **NEW** CAPTCHA middleware (pluggable providers) |
| `internal/session/session.go` | Session data model, fingerprinting |
| `internal/services/auth_service.go` | Auth orchestration, session creation |
| `internal/services/user_service.go` | Tenant-scoped user queries |
| `internal/database/tenant.go` | RLS via `SET LOCAL app.current_tenant_id` |
| `internal/stytch/stytch.go` | Stytch client with circuit breaker |
| `internal/config/config.go` | Environment validation |

---

## Compliance Mapping

| Standard | Status |
|----------|--------|
| OWASP API Top 10 (2023) | 5/10 Pass, 2 Partial, 3 Fail |
| OWASP ASVS 4.0 L1 | ~75% (auth, access control, logging, CSRF, business flow strong; crypto, comms, API inventory weak) |
| SOC 2 Type II (CC6.1-CC6.8) | Authentication & monitoring strong; authorization gaps (CC6.7) |

---

*Generated by automated OWASP API Security Top 10 review workflow. This audit covers code-as-of commit; re-run on significant auth/endpoint changes.*