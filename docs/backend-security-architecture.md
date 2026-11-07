# Backend Security Architecture — Somotracker (2026)

> **Purpose:** Share with AI (ChatGPT/Anthropic) for security review. Covers Go server, Redis, middleware, session management, rate limiting, and current threat mitigations.
>
> **Status:** Living document — update as architecture evolves.
>
> **Note:** Secrets managed via Doppler (no `.env` in repo). This doc describes *structure*, not secret values.

---

## 1. High-Level Stack

| Layer | Technology | Version | Notes |
|-------|------------|---------|-------|
| **Runtime** | Go | 1.22+ | `go.mod` pinned |
| **Web Framework** | Fiber v3 | v3.0.0 | `github.com/gofiber/fiber/v3` |
| **Database** | PostgreSQL | 16+ | `pgx/v5` driver, sqlc for type-safe queries |
| **Cache/Rate-Limit** | Redis | 7+ | `redis/go-redis/v9` |
| **Auth Provider** | Stytch B2B | — | Magic-link (Discovery), org/member management |
| **Secrets** | Doppler | — | Injected at deploy/runtime; no `.env` files |
| **Logging** | `slog` (structured) | — | Request ID correlation, no PII in logs |
| **Migrations** | Goose | — | SQL migrations in `migrations/` |

---

## 2. Request Flow & Middleware Chain

Order matters — **first listed = outermost** (executes first on request, last on response).

```
Request
  │
  ├─▶ 1. RequestID()           ← Generates/validates X-Request-ID (8-64 alphanum/_-)
  ├─▶ 2. AccessLog()           ← Single completion log line (method, route, status, latency, bytes)
  ├─▶ 3. recover.New()         ← Panic → 500 via ErrorHandler (stack trace logged)
  ├─▶ 4. helmet.New()          ← Security headers (see §3)
  ├─▶ 5. Custom security headers middleware (see §3)
  ├─▶ 6. CORS (configured origins, credentials=true)
  ├─▶ 7. logger.New()          ← Fiber access log (dev format)
  │
  ├─▶ /healthz, /readyz        ← Public, no auth
  │
  ├─▶ /api/v1 (public)         ← Identity routes (magic link, token exchange, onboarding)
  │
  ├─▶ /api/v1 (protected)      ← RequireAuth() → TenantMiddleware()
  │
  └─▶ /api/v1 (mutations)      ← RequireAuth() → TenantMiddleware() → CSRFMiddleware()
```

**Key Files:**
- `internal/platform/httpserver/server.go` — `New()`, middleware chain, `ErrorHandler`
- `internal/app/app.go` — App composition, route groups, middleware wiring
- `internal/identity/middleware/auth.go` — `RequireAuth`, `OptionalAuth`, `CSRFMiddleware`, `TenantMiddleware`

---

## 3. Security Headers (Helmet + Custom)

Applied to **all responses** via `app.Use()` in `app.go`:

| Header | Value | Notes |
|--------|-------|-------|
| `X-Content-Type-Options` | `nosniff` | Prevent MIME sniffing |
| `X-Frame-Options` | `DENY` | No framing |
| `Referrer-Policy` | `strict-origin-when-cross-origin` | |
| `Permissions-Policy` | `geolocation=(), microphone=(), camera=()` | |
| `Strict-Transport-Security` | `max-age=31536000` | **Only in production + HTTPS** |
| `Content-Security-Policy-Report-Only` | Configurable via `CSP_REPORT_ONLY_POLICY` + `CSP_REPORT_ONLY_URI` | Report-only rollout; not enforced yet |
| `Cache-Control` | `no-store` | On **all error responses** (in `ErrorHandler`) |

**CSP Status:** Report-only mode. Policy defaults to `default-src 'self'`. Configure via env:
- `CSP_REPORT_ONLY_ENABLED=true`
- `CSP_REPORT_ONLY_URI=https://your-report-endpoint`
- `CSP_REPORT_ONLY_POLICY="default-src 'self'; script-src 'self' ..."`

---

## 4. Authentication & Session Management

### 4.1 Stytch Integration (B2B Discovery Flow)
- **Magic Link** → `POST /api/v1/auth/magic-link` (email + PKCE)
- **Token Exchange** → `POST /api/v1/auth/callback` (token + PKCE verifier from cookie)
- **Onboarding Ticket** (10-min TTL, HMAC-SHA256 hash stored) — bridges Stytch IST to local session
- **Provisioning** → Creates Stytch org+member + local tenant/user/school/membership in **single DB transaction**

### 4.2 Local App Sessions (Post-Provisioning)
| Property | Implementation |
|----------|----------------|
| **Token** | 32 bytes (256-bit) cryptographically random (`crypto/rand`) |
| **Storage** | `app_sessions` table: `token_hash = SHA256(hex(token))`, `user_id`, `tenant_id`, `expires_at`, `revoked_at` |
| **Cookie** | `session_token_<env>` (dev/test) or `__Secure-session_token` (prod) |
| **Cookie Attributes** | `HttpOnly`, `SameSite=Lax`, `Secure` (prod only), `Path=/`, `Max-Age=30d` |
| **Lookup** | Middleware hashes cookie value → `SELECT ... WHERE token_hash = ? AND revoked_at IS NULL AND expires_at > NOW()` |
| **Rotation** | Not implemented (fixed 30-day expiry) |
| **Revocation** | `DELETE` on logout + Stytch member session revoke (best-effort) |
| **Concurrent Sessions** | Unlimited (no session limit per user) |

### 4.3 Session Middleware (`RequireAuth`)
```go
// internal/identity/middleware/auth.go
1. Read cookie (name varies by env)
2. SHA256(token) → lookup in app_sessions
3. Validate: not revoked, not expired
4. Fetch role via `GetTenantMembershipRole(user_id, tenant_id)`
5. Store in `c.Locals`: session_id, user_id, tenant_id, role
6. Store in context for service layer
```

---

## 5. Authorization (RBAC)

### 5.1 Roles (from `memberships.role`)
| Role | Scope | Typical Permissions |
|------|-------|---------------------|
| `tenant_owner` | Tenant | Full tenant admin, billing, user management |
| `school_admin` | School | School settings, teachers, students, fee catalogue |
| `finance_officer` | School | Fees, invoices, payments, reports |
| `teacher` | School/Class | Attendance, welfare notes for assigned classes |
| `guardian` | Student | Read-only own children's data |

### 5.2 Enforcement Points
- **Tenant isolation**: `TenantMiddleware()` extracts `tenant_id` from session → adds to context → all sqlc queries scoped by `tenant_id` (composite FKs)
- **Role checks**: Not yet centralized — handlers must check `c.Locals("role")` manually
- **Cross-tenant protection**: `GetUserByEmailAndTenant`, composite FKs `(tenant_id, id)` on all core tables

---

## 6. CSRF Protection

**Double-submit not used** — relies on **Origin header validation** (per ADR-0002: Go is sole security authority).

```go
// internal/identity/middleware/auth.go → CSRFMiddleware()
- Applies to: POST, PUT, DELETE, PATCH (not GET/HEAD/OPTIONS)
- Validates: `Origin` header against `CORSOrigins` config
- Dev fallback: allows `http://localhost:3000`
- Prod fallback: `defaultProductionOrigin` (must be overridden)
- Also checks `Sec-Fetch-Site: same-origin` if Origin missing
```

**Gap:** No CSRF token for non-browser clients (API tokens, mobile). Consider `SameSite=Strict` + token for mutating endpoints.

---

## 7. Rate Limiting (Redis-backed)

**Implementation:** `internal/platform/ratelimit/ratelimit.go` — Lua script for atomic `INCR + EXPIRE`.

| Endpoint / Action | Limit | Window | Key |
|-------------------|-------|--------|-----|
| Magic Link (IP) | `RATE_LIMIT_MAGIC_LINK_IP_PER_MIN` (default 10) | 1 min | `ip:<client_ip>` |
| Magic Link (Email) | `RATE_LIMIT_MAGIC_LINK_EMAIL_PER_15_MIN` (default 3) | 15 min | `email:HMAC_SHA256(email, EMAIL_HMAC_KEY)` |
| Token Exchange (IP) | `RATE_LIMIT_TOKEN_EXCHANGE_IP_PER_MIN` (default 10) | 1 min | `ip:<client_ip>` |
| Global Email Provider | `EMAIL_PROVIDER_HOURLY_CAP` (default 1000) | 1 hour | `global:email_provider` |
| Global Email Provider | `EMAIL_PROVIDER_DAILY_CAP` (default 10000) | 24 hours | `global:email_provider` |

**Email HMAC:** `HMAC-SHA256(lowercase(email), EMAIL_HMAC_KEY)` — prevents email enumeration in Redis keys.

**Gaps:**
- No rate limit on login callback (`/auth/callback`)
- No rate limit on password reset (not implemented)
- No adaptive/challenge-based limiting (e.g., CAPTCHA after N failures)
- No distributed rate limit sync across instances (single Redis = OK for now)

---

## 8. Input Validation & Sanitization

| Layer | Mechanism |
|-------|-----------|
| **Request Body** | `go-playground/validator/v10` — struct tags (`validate:"required,email,max=255"`) |
| **Error Contract** | Standardized `ErrorResponse` (code, message, errors map, request_id) — `internal/platform/apperr` |
| **JSON Parsing** | Fiber default + custom `ErrorHandler` catches `SyntaxError`, `UnmarshalTypeError` |
| **SQL Injection** | sqlc generates parameterized queries — no raw SQL in handlers |
| **XSS** | API-only (no server-rendered HTML); CSP report-only; `Content-Type: application/json` enforced |

---

## 9. Database Security

| Control | Implementation |
|---------|----------------|
| **Connection** | `pgx` pool, TLS required in production (managed PG) |
| **Credentials** | Doppler-injected `DATABASE_URL` |
| **RLS / Tenant Isolation** | Composite FKs `(tenant_id, id)` + `tenant_id` in all queries via context |
| **Migrations** | Goose, versioned SQL, run on startup (`internal/platform/migrator`) |
| **Soft Deletes** | `revoked_at`, `voided_at`, `deleted_at` columns — no hard deletes |
| **Audit** | `created_by`, `updated_by`, `voided_by`, `approved_by` on financial tables (fee module) |

---

## 10. Logging & Observability

- **Structured JSON** (`slog`) with `request_id` on every line
- **No PII** in logs — emails hashed in rate-limit keys, tokens never logged
- **Error levels:** 4xx = Info, 429 = Warn, 5xx = Error + stack
- **Access log:** Single line per request (method, route pattern, status, latency_ms, bytes)
- **Health/Ready:** `/healthz` (liveness), `/readyz` (DB + Redis ping)

---

## 11. Current Threat Mitigations (2026 Baseline)

| Threat | Mitigation | Gap / TODO |
|--------|------------|------------|
| **Credential stuffing / bot login** | Rate limit magic link by IP + email HMAC | No CAPTCHA, no device fingerprinting, no WAF |
| **Session hijacking** | `HttpOnly`, `Secure`, `SameSite=Lax`, SHA256 token hash, short TTL (30d) | No rotation, no concurrent session limit, no IP/user-agent binding |
| **CSRF** | Origin header validation on mutations | No token for SPA/mobile; `SameSite=Lax` not `Strict` |
| **XSS** | API-only, CSP report-only, `nosniff` | CSP not enforced; no output encoding needed (JSON) |
| **SQL Injection** | sqlc parameterized queries | ✅ Covered |
| **Tenant data leakage** | Composite FKs, context-scoped `tenant_id`, RLS-ready | No PG RLS policies enabled yet |
| **Secrets leakage** | Doppler, no `.env`, `slog` redacts sensitive fields | ✅ Covered |
| **DoS / Resource exhaustion** | Body limit (Fiber default), rate limits, timeouts | No request size limit per endpoint, no slowloris protection |
| **Supply chain** | `go.mod` pinned, `go.sum` verified | No `govulncheck` in CI, no SBOM |
| **Stytch token replay** | PKCE (S256) on magic link flow | ✅ Covered |
| **Onboarding ticket theft** | 10-min TTL, SHA256 hash, single-use, encrypted IST | IST stored plaintext (TODO: encrypt) |

---

## 12. Known Gaps & Hardening Priorities

### 🔴 Critical (Do Before Launch)
1. **Enable PG Row-Level Security** — policies on all tenant-scoped tables
2. **Encrypt Onboarding Ticket IST** — currently stored plaintext in `onboarding_tickets.encrypted_ist`
3. **Session Rotation / Refresh Tokens** — 30-day static token is risky
4. **Concurrent Session Limit** — e.g., max 5 per user, revoke oldest
5. **IP/User-Agent Binding** — optional flag on session, invalidate on mismatch
6. **Enforce CSP** — move from report-only to enforced after testing

### 🟠 High (Post-Launch Sprint)
7. **Centralized RBAC Middleware** — `RequireRole("finance_officer", "school_admin")` instead of manual `c.Locals` checks
8. **Adaptive Rate Limiting** — CAPTCHA/honeypot after N failures, exponential backoff
9. **Request Body Size Limits** — per-endpoint (e.g., 1MB default, 10MB for uploads)
10. **Security Headers Audit** — `helmet` defaults + custom; verify `Permissions-Policy`, `Cross-Origin-Opener-Policy`
11. **Audit Log Tamper-Proofing** — append-only table, hash chaining, or external SIEM

### 🟡 Medium (Continuous)
12. **Dependency Scanning** — `govulncheck` in CI, `go list -m -json all | nancy sleuth`
13. **Pen Test / SAST** — integrate `gosec`, `staticcheck` in CI
14. **Session Anomaly Detection** — geo-velocity, new device alerts
15. **API Versioning & Deprecation** — prevent stale clients
16. **Backup Encryption & Restore Testing** — quarterly

---

## 13. Configuration Reference (Env Vars via Doppler)

| Variable | Required | Default | Notes |
|----------|----------|---------|-------|
| `APP_ENV` | Yes | `development` | `development\|test\|production` |
| `DATABASE_URL` | Yes | — | Postgres DSN (TLS in prod) |
| `REDIS_ADDR` / `REDIS_URL` | Yes | `localhost:6379` | Managed Redis in prod |
| `PORT` | No | `8080` | |
| `LOG_LEVEL` | No | `info` | `debug\|info\|warn\|error` |
| `STYTCH_ENV` | Yes | `test` | `test\|live` |
| `STYTCH_PROJECT_ID` | Yes | — | |
| `STYTCH_SECRET` | Yes | — | |
| `EMAIL_HMAC_KEY` | **Prod only** | — | 32+ byte random, for email rate-limit HMAC |
| `CORS_ORIGINS` | Yes | `PUBLIC_APP_URL` | Comma-separated, e.g., `https://app.example.com,https://admin.example.com` |
| `COOKIE_DOMAIN` | Prod | — | e.g., `.example.com` for subdomain sharing |
| `CSP_REPORT_ONLY_ENABLED` | No | `false` | `true\|false` |
| `CSP_REPORT_ONLY_URI` | No | — | Report endpoint |
| `CSP_REPORT_ONLY_POLICY` | No | `default-src 'self'` | |
| `EMAIL_PROVIDER_HOURLY_CAP` | No | `1000` | |
| `EMAIL_PROVIDER_DAILY_CAP` | No | `10000` | |
| `RATE_LIMIT_MAGIC_LINK_IP_PER_MIN` | No | `10` | |
| `RATE_LIMIT_MAGIC_LINK_EMAIL_PER_15_MIN` | No | `3` | |
| `RATE_LIMIT_TOKEN_EXCHANGE_IP_PER_MIN` | No | `10` | |

---

## 14. Files to Review for AI Context

```
backend/
├── internal/platform/httpserver/server.go       # Middleware chain, ErrorHandler, RequestID
├── internal/app/app.go                          # App composition, security headers, CORS, route groups
├── internal/identity/middleware/auth.go         # RequireAuth, CSRF, TenantMiddleware
├── internal/identity/service/service.go         # Stytch flow, session create/validate, provisioning
├── internal/platform/ratelimit/ratelimit.go     # Redis rate limiting (IP, email HMAC, global)
├── internal/platform/config/config.go           # Config validation, env loading
├── internal/platform/apperr/                    # Error codes, contract shape
├── internal/identity/store/gen/                 # sqlc-generated queries (parameterized)
└── migrations/                                  # Goose SQL migrations (schema, constraints)
```

---

## 15. Quick "Ask the AI" Prompts

> Copy-paste these into ChatGPT/Anthropic with this doc attached.

1. **Session Hardening:** "Review our session design (256-bit random token, SHA256 hash in DB, 30-day TTL, HttpOnly/SameSite=Lax cookie). What 2026 attacks (token theft, replay, fixation) are we vulnerable to? Propose rotation + binding strategy compatible with Fiber/Go."

2. **Rate Limiting Gaps:** "Our Redis rate limits use fixed windows (Lua INCR+EXPIRE). What bypasses exist (distributed clients, header spoofing, email enumeration via timing)? Suggest sliding-window log or token bucket with Redis sorted sets."

3. **CSRF for SPA + Mobile:** "We validate Origin header on mutations. Is this sufficient for a React SPA + future React Native app? Should we add double-submit cookie or custom header (`X-CSRF-Token`) for non-browser clients?"

4. **Tenant Isolation Verification:** "We use composite FKs `(tenant_id, id)` and context-scoped `tenant_id` in all queries. No PG RLS enabled. What queries could leak cross-tenant data if a handler forgets the context? Write a sqlc linter rule or test pattern."

5. **Stytch Dependency Risk:** "Our auth depends entirely on Stytch B2B (magic link, org/member, session). What happens if Stytch is down, compromised, or deprecates Discovery? Design a local fallback or migration path."

6. **2026 Bot Mitigation:** "Magic link endpoint has IP + email rate limits. Modern bots use residential proxies, credential stuffing lists, and AI CAPTCHA solvers. What layered defenses (device fingerprint, behavioral, honeypot, WAF) can we add without blocking legitimate Kenyan school admins on 3G?"

7. **Audit & Compliance:** "Kenya Data Protection Act 2019 requires audit trails, breach notification, data subject access. Our fee module has `created_by`/`voided_by` on financial tables. What's missing for full compliance? Map to DPA sections."

---

## 16. Version History

| Date | Author | Change |
|------|--------|--------|
| 2026-11-07 | pi | Initial draft from codebase inspection |

---

*Delete this file after AI review. Do not commit secrets.*