# Auth & Session Security — Deep Audit Report

**Auditor:** AI Agent  
**Date:** September 2025  
**Status:** 🟡 Review in Progress  
**Confidence:** High

---

## Table of Contents

1. [Architecture Overview](#1-architecture-overview)
2. [Auth Flows](#2-auth-flows)
3. [Session Management](#3-session-management)
4. [Security Middleware Stack](#4-security-middleware-stack)
5. [Frontend Auth Layer](#5-frontend-auth-layer)
6. [Error Handling & Contracts](#6-error-handling--contracts)
7. [Security Properties by Threat](#7-security-properties-by-threat)
8. [Findings & Risk Ratings](#8-findings--risk-ratings)
9. [Gap Analysis](#9-gap-analysis)
10. [Recommendations](#10-recommendations)

---

## 1. Architecture Overview

### 1.1 Technology Stack

| Layer | Technology | Purpose |
|-------|-----------|---------|
| Identity Provider | **Stytch B2B** | Magic-link authentication, org/member management, MFA |
| Database | **PostgreSQL** (with RLS) | Multi-tenant persistence, session storage |
| Cache | **Redis** | Session cache, IST cache, rate limiting |
| Backend Framework | **Go + Fiber** | REST API, middleware pipeline |
| Frontend | **Next.js App Router** | Dashboard, auth pages |
| Cookie Storage | **Browser HttpOnly cookies** | Opaque session token |

### 1.2 Package Map

```
backend/internal/
├── auth/          — Core auth logic: service, handler, repository, Stytch adapter
├── middleware/     — Security pipeline: session resolver, CSRF, rate limits, CORS, device fingerprint
├── database/      — RLS setup, tenant transaction scoping
├── xerrors/       — Structured error types (DomainError)
└── config/        — Configuration loading (Stytch, Redis, cookie secrets)

frontend/src/
├── lib/api/auth.ts    — API client functions for auth endpoints
├── lib/api/client.ts  — Base fetch wrapper, ApiError, global 401/403 handlers
├── hooks/use-auth.ts   — React Query hooks (useMe, useDiscover, useVerifyToken, useRegister, useLogout)
├── proxy.ts           — Route-level auth guard (UX guard only)
└── features/auth/     — Login page, register form, components
```

### 1.3 Key Design Decisions

- **Stytch B2B Magic Links** are the sole authentication mechanism — no passwords, no OAuth, no WebAuthn.
- **Opaque session tokens** (32 random bytes, hex-encoded) stored in an HttpOnly cookie (`somo_sid`). Stytch session tokens are never sent to the browser.
- **Multi-tenant isolation** via PostgreSQL Row-Level Security (RLS) with tenant-scoped transactions set via `app.current_tenant_id` GUC.
- **CSRF protection** via the Double-Submit Cookie pattern: `csrf_token` (non-HttpOnly) read by JS, sent as `X-CSRF-Token` header on mutating requests.
- **Device fingerprinting** via `sha256(User-Agent + "|" + Accept-Language)` stored at session creation, enforced in production only.
- **Redis is not the session authority** — Postgres is. Redis is a cache. Cache misses always fall through to Postgres.

---

## 2. Auth Flows

### 2.1 Magic-Link Discovery Flow (Login / New User)

```
Browser                     Frontend                   Backend                    Stytch
   │                             │                         │                          │
   │  POST /api/auth/discover    │                         │                          │
   │  { email }                 │                         │                          │
   │────────────────────────────►│                         │                          │
   │                             │  SendDiscoveryEmail     │                          │
   │                             │────────────────────────►│  Magic link email        │
   │                             │                         │─────────────────────────►│
   │                             │                         │                          │
   │  User clicks magic link     │                         │                          │
   │  GET /api/auth/callback     │                         │                          │
   │  ?token=XXX                │                         │                          │
   │────────────────────────────►│                         │                          │
   │                             │  AuthenticateDiscoveryToken                      │
   │                             │────────────────────────►│                          │
   │                             │  (returns IST, email,   │  Validate token          │
   │                             │   discovered orgs)       │─────────────────────────►│
   │                             │◄────────────────────────│                          │
   │                             │                         │                          │
   │  [No existing orgs]         │                         │                          │
   │  Cache IST + email in       │                         │                          │
   │  Redis (10 min TTL)         │                         │                          │
   │                             │                         │                          │
   │  302 → /register           │                         │                          │
   │  ?session_ref=UUID         │                         │                          │
   │◄────────────────────────────│                         │                          │
   │                             │                         │                          │
   │  [Existing orgs found]     │                         │                          │
   │  Exchange IST → session     │                         │                          │
   │  Create DB session          │                         │                          │
   │  Cache in Redis             │                         │                          │
   │                             │                         │                          │
   │  302 → / (dashboard)       │                         │                          │
   │  + somo_sid cookie         │                         │                          │
   │◄────────────────────────────│                         │                          │
```

### 2.2 Registration Flow

```
Browser                     Frontend                   Backend                    Stytch
   │                             │                         │                          │
   │  POST /api/auth/register    │                         │                          │
   │  { school_name,            │                         │                          │
   │    session_ref,            │                         │                          │
   │    full_name }             │                         │                          │
   │────────────────────────────►│                         │                          │
   │                             │  Read + DELETE IST      │                          │
   │                             │  from Redis (atomic     │                          │
   │                             │  Lua script)            │                          │
   │                             │                         │                          │
   │                             │  CreateOrganization      │                          │
   │                             │  (if new tenant)        │                          │
   │                             │────────────────────────►│  Create org              │
   │                             │                         │─────────────────────────►│
   │                             │                         │                          │
   │                             │  CreateMember           │                          │
   │                             │────────────────────────►│  Add member to org       │
   │                             │                         │─────────────────────────►│
   │                             │                         │                          │
   │                             │  ExchangeIntermediateSession                    │
   │                             │────────────────────────►│  Full session token      │
   │                             │                         │─────────────────────────►│
   │                             │                         │                          │
   │                             │  CreateTenantUserSession                       │
   │                             │  (DB transaction)       │                          │
   │                             │  CreateSchool (via      │                          │
   │                             │  cbcschools.Service)    │                          │
   │                             │  CreateMembership       │                          │
   │                             │                         │                          │
   │                             │  Cache session          │                          │
   │                             │  in Redis              │                          │
   │                             │                         │                          │
   │  204 No Content            │                         │                          │
   │  + somo_sid cookie        │                         │                          │
   │  + somo_role cookie        │                         │                          │
   │  + somo_school_id cookie   │                         │                          │
   │  + csrf_token cookie       │                         │                          │
   │◄────────────────────────────│                         │                          │
   │                             │                         │                          │
   │  302 → / (dashboard)       │                         │                          │
   │◄────────────────────────────│                         │                          │
```

### 2.3 Invite Acceptance Flow

```
Browser                     Backend                    Stytch
   │                             │                          │
   │  User clicks invite link     │                          │
   │  GET /api/auth/invite/      │                          │
   │  callback?token=XXX         │                          │
   │────────────────────────────►│                          │
   │                             │  AuthenticateMagicLink    │
   │                             │────────────────────────►│
   │                             │  (org-scoped endpoint,   │
   │                             │   NOT discovery)         │
   │                             │                         │
   │                             │  [MFA check]            │
   │                             │                         │
   │                             │  Lookup invitation      │
   │                             │  by email               │
   │                             │                         │
   │                             │  CreateInvitedUserSession│
   │                             │  (DB transaction)       │
   │                             │                         │
   │                             │  Cache session          │
   │                             │  in Redis              │
   │                             │                         │
   │  302 → / (dashboard)       │                         │
   │  + all session cookies     │                         │
   │◄────────────────────────────│                         │
```

### 2.4 Logout Flow

```
Browser                     Backend
   │                             │
   │  DELETE /api/auth/session  │
   │  (with somo_sid cookie)    │
   │────────────────────────────►│
   │                             │  DeleteSession (Postgres)
   │                             │  Del Redis (BOTH keys):
   │                             │    - session:{token}
   │                             │    - session:{sha256(token)}
   │                             │
   │  Clear all cookies         │
   │  (somo_sid, somo_role,     │
   │   somo_school_id, csrf)    │
   │◄────────────────────────────│
   │                             │
   │  204 No Content            │
```

### 2.5 MFA Flow

```
User has MFA enabled in Stytch
   │
   ▼
Magic-link click → AuthenticateDiscoveryToken
   │
   ├── MemberAuthenticated = true → normal login
   │
   └── MemberAuthenticated = false
            │
            ▼
       Return 401 with code "mfa_required"
            │
            ▼
       Frontend shows MFA prompt
            │
            ▼
       User completes MFA via Stytch
            │
            ▼
       ExchangeIntermediateSession
       (resumes the cached IST)
```

---

## 3. Session Management

### 3.1 Session Lifecycle

| Phase | Token | Storage | TTL |
|-------|-------|---------|-----|
| Post-magic-link, pre-registration | Stytch IST | Redis (`ist:{env}:{session_ref}`) | 10 minutes |
| Post-magic-link, pre-registration (MFA) | Stytch IST | Redis | 10 minutes |
| Active authenticated session | Opaque hex token (64 chars) | Postgres + Redis (`session:{sha256(token)}`) | 30 days |

### 3.2 Session Resolution (Per-Request)

```
Request arrives
       │
       ▼
Middleware: NewSessionResolver
       │
       ├── Check Redis cache
       │   Key: session:{sha256(raw_token)}
       │   └── Hit → Return SessionInfo (role, school_id, schools[])
       │
       └── Miss → Load from Postgres
                    │
                    ├── Query fn_resolve_session(token_hash)
                    │   Returns: user_id, tenant_id, role, school_id, schools[], device_fingerprint
                    │   RLS bypass: SECURITY DEFINER
                    │
                    ├── B3 check: role == NULL → 403 Forbidden
                    │   (valid session but zero memberships)
                    │
                    └── Cache in Redis (15 min TTL)
                        Negative cache: invalid tokens → "INVALID" sentinel (30s TTL)
```

### 3.3 School Context (C2)

```
somo_school_id cookie (unsigned, client-controlled)
         │
         ├── Is cookie value in sess.Schools[]?
         │   YES → Use cookie value as active_school_id
         │   NO  → Use DB active_school_id
         │
         └── If no schools at all → active_school_id = ""
```

The cookie can only select a school the user already has an active membership for.

### 3.4 Session Cache Invalidation (B2)

On logout, **both** Redis key formats are purged:

```go
keys := []string{
    "session:" + rawToken,           // Legacy: auth service format
    "session:" + sha256(rawToken),  // Current: session resolver format
}
```

This ensures logout invalidates the session in all cache layers.

### 3.5 Redis Double-Write (B2 Risk)

The auth service writes Redis keys in two formats:

1. **`session:{rawToken}`** — written by `auth/service.go` on login/register
2. **`session:{sha256(rawToken)}`** — written by `middleware/sessionresolver.go` on cache miss

Both formats must be invalidated on logout. The `purgeSessionCacheKeys` method does this correctly.

**Risk:** If a new cache write format is added in the future without updating `purgeSessionCacheKeys`, logout could leave stale sessions.

---

## 4. Security Middleware Stack

### 4.1 Middleware Registration Order (`middleware/register.go`)

```
1.  WithLogger          — Inject zap logger into c.Locals
2.  NewPanicRecover     — Catch panics, return 500
3.  NewRequestID        — Generate / honor X-Request-ID
4.  NewCORS             — Origin/credentials/headers policy
5.  NewSecurityHeaders   — HSTS, CSP, X-Frame-Options, etc.
6.  NewCSRFGuard        — Double-submit cookie validation
7.  NewRateLimiter      — Coarse IP-based throttle (300 req/min)
8.  NewDeviceFingerprinter — Compute fingerprint → c.Locals
9.  NewSessionResolver  — Load session from Redis/Postgres
10. WithTenantContext   — Open RLS-scoped transaction
11. NewAccessLog        — Log request with user context
12. NewRateLimiter      — Fine per-user throttle (100 req/min)
```

### 4.2 CSRF Guard (Double-Submit Cookie)

- Ignores: `GET`, `HEAD`, `OPTIONS`
- Ignores paths: `/api/auth/discover`, `/api/auth/verify`, `/api/auth/register` (these are pre-session)
- **Origin Header Check:** If `Origin` header is present and `AllowedOrigins` is configured, the origin must match.
- **Token Match:** `X-CSRF-Token` header must equal `csrf_token` cookie (constant-time comparison).

**Status:** `NewCSRFGuard` is now applied globally in `middleware/register.go` (Sept 2025). Pre-session auth routes are excluded via `ignoredPrefixes`. All other mutating routes are protected.

### 4.3 Device Fingerprint (C5)

| Version | Format | Inputs | Notes |
|---------|--------|--------|-------|
| v1 (legacy) | `{hex}` (no prefix) | `IP + "|" + User-Agent + "|" + Accept-Language` | Cannot be re-verified after migration |
| v2 (current) | `v2:{hex}` | `User-Agent + "|" + Accept-Language` | IP intentionally omitted |

Enforcement is gated by `cfg.EnforceDeviceFingerprint` (production only).
**Status:** As of Sept 2025, `config.Load()` panics in non-development if `EnforceDeviceFingerprint` is false, making enforcement mandatory in production.

**Risk:** v1 sessions continue in production with no enforcement. After the v1→v2 migration window closes, ensure legacy sessions are flushed.

### 4.4 Rate Limiting

| Limiter | Key | Limit | Window | Notes |
|---------|-----|-------|--------|-------|
| Coarse IP | `ratelimit:ip_coarse:{ip}` | 300 req | 1 min | Exempts Stytch callbacks |
| Auth routes | `ratelimit:ip_limiter:{ip}` | 10 req | 1 min | Per-route, applied in handler |
| Per-user | `ratelimit:user_fine:{user_id}` | 100 req | 1 min | Fail-open on Redis error |

### 4.5 CORS Configuration

```go
AllowOrigins:     cfg.AllowedOrigins
AllowMethods:     "GET,POST,PUT,DELETE,PATCH,OPTIONS"
AllowHeaders:     "Origin,Content-Type,Accept,Authorization,X-Requested-With,X-CSRF-Token,X-Request-ID"
AllowCredentials: true
MaxAge:           86400
```

**Note:** `AllowCredentials: true` requires `AllowOrigins` to be specific (no `*`). The config must not be empty in production.

---

## 5. Frontend Auth Layer

### 5.1 Route Guard (`proxy.ts`)

The Next.js proxy performs a **UX guard only**:

```
Request path
     │
     ├── Is public route? (login, register, logout, unauthorized)
     │   YES → Allow
     │
     ├── Protected prefix? (/students, /classes, /, etc.)
     │   YES → Has somo_sid cookie?
     │          ├── YES → Allow
     │          └── NO  → Redirect to /login
     │
     └── Not protected → Allow
```

**Critical:** The proxy only checks for the **presence** of the cookie, not its validity. Valid session verification is done by the backend on each API request.

### 5.2 API Client (`lib/api/client.ts`)

| Feature | Implementation |
|---------|---------------|
| Global 401 handler | `window.location.href = "/logout"` (unless `skipGlobal401Handler`) |
| Global 403 handler | `window.location.href = "/unauthorized"` (only on GET `/api/auth/me`) |
| CSRF header | `X-CSRF-Token` from `csrf_token` cookie on POST/PUT/PATCH/DELETE |
| Credentials | `credentials: "include"` (sends cookies to API) |
| Correlation ID | `X-Request-ID` (per-page-load UUID) |
| Error shape | `{ code, message, errors, request_id }` |

### 5.3 Frontend Auth Hooks

| Hook | Purpose |
|------|---------|
| `useMe()` | Fetch current user session; returns `null` on error (not throwing) |
| `useDiscover()` | Send magic link (Phase 1) |
| `useVerifyToken()` | Verify token, handle expired/session_ref_expired codes |
| `useRegister()` | Complete registration; handles 401 redirect on expired session_ref |
| `useLogout()` | Clear session; invalidates React Query cache |

### 5.4 Auth Pages

| Page | Route | Auth State |
|------|-------|-----------|
| Login | `/login` | Unauthenticated only |
| Register | `/register` | Unauthenticated with `?session_ref=` param |
| Log| Logout | `/logout` | Unauthenticated, always allowed |
| Unauthorized | `/unauthorized` | Unauthenticated, always allowed |

---

## 6. Error Handling & Contracts

### 6.1 Canonical Error Response

Every non-2xx response from backend MUST return:

```json
{
  "code": "snake_case_error_code",
  "message": "human readable message",
  "errors": { "field_name": ["Specific field validation message"] },
  "request_id": "correlation-id"
}
```

Reference: `backend/internal/middleware/errors.go`

### 6.2 Auth-Specific Error Codes

| Code | HTTP | Meaning | Frontend Action |
|------|------|---------|---------------|
| `expired_token` | 401 | Magic-link expired | Show "Link expired", request new |
| `session_ref_expired` | 401 | IST already consumed | Show "Link already used", request new |
| `mfa_required` | 401 | User needs MFA | Prompt for MFA verification |
| `invalid_input` | 400 | Validation failure | Show field errors via `form.setError` |
| `unauthorized` | 401 | No session / bad cookie | Redirect to `/logout` |
| `forbidden` | 403 | Zero memberships (B3) / bad role | Redirect to `/unauthorized` |
| `not_found` | 404 | Session/user/tenant missing | Log out |
| `internal_error` | 500 | Unexpected failure | Log, show generic message |

### 6.3 Session Cache Errors

- `ErrNotFound` from DB → `ErrExpiredToken` (401) to prevent information leakage
- Redis error → Log at Warn, fail open for rate limiter; fail closed for session resolution
- Cache miss in session resolver → Always load from DB (`fn_resolve_session` with SECURITY DEFINER)

---

## 7. Security Properties by Threat

### 7.1 Threat Matrix

| Threat | Control | Status | Notes |
|--------|---------|--------|-------|
| Session hijacking (cookie theft) | HttpOnly cookie, 30d TTL, device fingerprint (mandatory in prod), session resolution validates DB | ✅ | Fingerprint enforcement required in non-dev via config panic |
| CSRF attack | Double-submit cookie, Origin header check, ignored on pre-session endpoints | ✅ | `NewCSRFGuard` now global in middleware pipeline (2025-09) |
| Replay attack (magic link reuse) | IST read-and-delete via atomic Lua script (GET + DEL) | ✅ | One-time use enforced |
| Session fixation | New opaque token on every login/register | ✅ | Old session invalidated via DB delete + Redis purge |
| Brute force | IP rate limiter (10 req/min on auth), coarse IP throttle (300/min), per-user throttle (100/min) | ✅ | Fail-open on Redis outage (logged) |
| Session replay after logout | Both Redis key formats purged | ✅ | Potential gap if new format added |
| Cross-site scripting (XSS) | HttpOnly session cookie, CSP `default-src 'self'`, `X-XSS-Protection: 0` | ✅ | Non-HttpOnly cookies (role, school) readable by JS |
| Session injection (fake cookie) | Postgres `token_hash` verification + RLS | ✅ | Cookie alone is not sufficient |
| Man-in-the-middle | HTTPS + HSTS (only when `https` or `X-Forwarded-Proto: https`) | ⚠️ | HSTS only with TLS termination |
| Privilege escalation | Role from DB memberships (not cookie), B3 rejects zero memberships | ✅ | `somo_role` cookie only for routing; DB is authority |
| School scoping (C2) | Cookie only selects from `sess.Schools[]`; DB active school is fallback | ✅ | Forged cookie can never scope to unauthorized school |
| Device replay (C5) | Fingerprint comparison, production-only enforcement (mandatory) | ✅ | `EnforceDeviceFingerprint` enforced via config panic in non-dev |

---

## 8. Findings & Risk Ratings

### 8.1 Critical (Red) — RESOLVED

| # | Finding | Evidence | Impact | Status |
|---|---------|----------|--------|--------|
| **C-1** | **CSRF guard is not global** — `NewCSRFGuard` registered only on auth routes via `auth/module.go` / handler group, not in `middleware/register.go`. Mutating endpoints (`PATCH`, `DELETE`, `PUT` on other features) have no CSRF protection unless individually configured. | `middleware/csrf.go`, `auth/handler.go` line 90-95 | Cross-site request forgery on non-auth endpoints | ✅ RESOLVED 2025-09 — `NewCSRFGuard(cfg)` now in `middleware/register.go` pipeline; pre-session routes ignored via `ignoredPrefixes` |
| **C-2** | **Cookie secret fallback is insecure** — `COOKIE_SECRET` defaults to `"dev-insecure-change-in-production"` if not set. In production without explicit env, the HMAC on `somo_role` is forgeable. | `backend/internal/config/config.go` | Role cookie forgery, privilege escalation | ✅ RESOLVED 2025-09 — `config.Load()` panics in non-dev if `COOKIE_SECRET` missing or equals dev default |

### 8.2 High (Orange)

| # | Finding | Evidence | Impact | Status |
|---|---------|----------|--------|--------|
| **H-1** | **Device fingerprint enforcement disabled** — `EnforceDeviceFingerprint` defaults to `false`. Mismatches are logged but allowed (`logger.Warnw`). In production with mobile/NAT, this is a risk signal only. | `config/config.go`, `middleware/sessionresolver.go` | Session cookie replay from other devices allowed | ✅ RESOLVED 2025-09 — `config.Load()` panics in non-dev if `ENFORCE_DEVICE_FINGERPRINT` is false; enforcement now mandatory in production |
| **H-2** | **Proxy only checks cookie presence** — `proxy.ts` redirects based solely on `cookies.has("somo_sid")`, not session validity. A stale/expired cookie passes through until API fail. | `frontend/src/proxy.ts` | UX delay before error; not a security failure | Open |
| **H-3** | **Stytch SDK dependency in auth adapter** — Only `auth/stytch.go` should import Stytch SDK (design rule), but the adapter is a single file with all SDK methods. Any SDK vulnerability affects auth exclusively. | `auth/stytch.go`, 600+ lines | Large attack surface, hard to audit | Open |
| **H-4** | **Partial wipe recovery is complex** — `reconstructFromStytch`, `handleExistingUser`, `createUserInExistingTenant` have multiple recovery paths with different session/token outcomes. Potential for inconsistent state. | `auth/service.go` 400+ lines | Data inconsistency, session duplication | Open |

### 8.3 Medium (Yellow)

| # | Finding | Evidence | Impact |
|---|---------|----------|--------|
| **M-1** | **Redis negative cache (30s)** — Invalid tokens cached as `"INVALID"`. A token invalidated shortly after creation could be wrongly rejected within 30s if clock desync. | `middleware/sessionresolver.go` | Temporary false-negative (safe failure) |
| **M-2** | **Rate limiter fail-open** — Redis outage skips throttle (`c.Next()` after `warnRateLimitDegraded`). No fallback throttle. | `middleware/ratelimiter.go` | Abuse possible during Redis outage |
| **M-3** | **v1 device fingerprint legacy** — Old sessions with unprefixed fingerprint continue; cannot be re-verified. Migration window unclear. | `middleware/sessionresolver.go` | Session continuity vs security tradeoff |
| **M-4** | **Cookie domain empty by default** — `CookieDomain` defaults to `""`. Cookies are domain-bound to the exact host; subdomain sharing may break. | `config/config.go` | UX issue, not security |

### 8.4 Low (Green / Informational)

| # | Finding | Evidence | Impact |
|---|---------|----------|--------|
| **L-1** | **Session token 32 bytes** — 256 bits of entropy; sufficient for session security. | `auth/service.go` |
| **L-2** | **Atomic IST consumption** — Lua `GET+DEL` prevents TOCTOU race on session ref. | `auth/service.go` `readAndDeleteIST` |
| **L-3** | **Postgres is session authority** — Redis miss always loads from DB (`fn_resolve_session`). | `middleware/sessionresolver.go` |

---

## 9. Gap Analysis

### 9.1 Verified Working

- ✅ Magic-link flow (discover → verify → register/login)
- ✅ Session creation via opaque token + HttpOnly cookie
- ✅ Postgres as session source of truth
- ✅ RLS tenant isolation with `fn_resolve_session`
- ✅ CSRF double-submit globally (pre-session routes excluded)
- ✅ Rate limiting (IP + per-user + per-email for discover)
- ✅ Device fingerprint computation (v2) + mandatory enforcement in production
- ✅ Redirection after login/logout
- ✅ Cookie clearing on logout (all 4 cookies)

### 9.2 Potential Gaps

- ✅ **Global CSRF applied** — `NewCSRFGuard` now registered globally in `middleware/register.go` (2025-09); pre-session auth routes excluded via ignored prefixes
- ❓ **HSTS not enforced** — Only when `https` or `X-Forwarded-Proto: https`; if terminated at load balancer without header, HSTS missing
- ✅ **Cookie secret enforced** — `config.Load()` now panics in non-development if `COOKIE_SECRET` is unset or equals dev default (2025-09)
- ✅ **Session TTL configurable** — `SessionTTL` now in config via `SESSION_TTL` env, overridden in `auth/service.go` (2025-09)
- ✅ **Session revocation API** — `DELETE /api/auth/sessions/:token` with `RequireAuth` + `RequireRole("admin")` added (2025-09)
- ❓ **No audit log for auth events** — Login, register, logout not written to audit table (only zap logs) [skipped per request]
- ✅ **Per-email brute-force tracking** — Redis per-email throttle 5/15min on `/api/auth/discover` (2025-09)

---

## 10. Recommendations

### 10.1 Immediate (Before Production) — UPDATED

1. ~~**Enable `EnforceDeviceFingerprint`** in production environment config (`APP_ENV=production`).~~ ✅ Resolved 2025-09 — config panics if disabled in non-dev.
2. ~~**Set `COOKIE_SECRET` explicitly** in all environments; remove default from `.env` and config.~~ ✅ Resolved 2025-09 — config panics if missing/default.
3. ~~**Apply CSRF globally** — Add `NewCSRFGuard(cfg)` to `middleware/register.go` for all non-public routes, or apply to all feature route groups.~~ ✅ Resolved 2025-09 — global middleware active.
4. **Verify `AllowedOrigins`** is specific (not `*`) when `AllowCredentials: true` is set. [Still required]

### 10.2 Short Term

5. **Audit `auth/stytch.go`** — Ensure no secrets (StytchSecret) leaked in logs (currently logged at initialization: `zap.String("project_id", ...)` is safe; verify `StytchSecret` never logged).
6. ~~**Add audit table** — Log auth events (login, register, logout) with user_id, timestamp, IP, device fingerprint.~~ Skipped per request
7. ~~**Add admin session revocation** — Endpoint to kill specific sessions by token/user.~~ ✅ Resolved 2025-09 — `DELETE /api/auth/sessions/:token` with admin guard
8. ~~**Make session TTL configurable** — Via config/env rather than hardcoded constant.~~ ✅ Resolved 2025-09 — `SESSION_TTL` config

### 10.3 Long Term

9. **Consider session rotation** — Rotate session token on privilege change (role upgrade, school switch).
10. **Add MFA enforcement option** — Make MFA required for certain roles beyond Stytch configuration.
11. **Implement session idle timeout** — In addition to 30-day absolute TTL, add inactivity timeout.
12. **Consider WebAuthn / passkey integration** — As alternative to magic links for high-security roles.

---

## Appendices

### A. Key Files Referenced

| File | Purpose |
|------|---------|
| `backend/internal/auth/module.go` | fx module definition |
| `backend/internal/auth/domain.go` | Errors, interfaces, models |
| `backend/internal/auth/service.go` | Business logic, session management |
| `backend/internal/auth/handler.go` | HTTP endpoints |
| `backend/internal/auth/stytch.go` | Stytch SDK adapter |
| `backend/internal/auth/repository.go` | DB access |
| `backend/internal/middleware/auth.go` | RequireAuth, RequireRole |
| `backend/internal/middleware/sessionresolver.go` | Session load + cache |
| `backend/internal/middleware/register.go` | Middleware pipeline order |
| `backend/internal/middleware/csrf.go` | Double-submit cookie |
| `backend/internal/middleware/fingerprint.go` | Device fingerprint |
| `backend/internal/middleware/ratelimiter.go` | Sliding window throttle |
| `frontend/src/proxy.ts` | Route auth guard |
| `frontend/src/lib/api/client.ts` | API client with error handling |
| `frontend/src/hooks/use-auth.ts` | React Query auth hooks |

### B. Document Information

- **Created by:** AI Agent (audit task)
- **Based on:** Source code audit of `backend/`, `frontend/src/`, `docs/`
- **Methodology:** Static code review + architecture analysis + threat modeling
- **Limitations:** Dynamic testing (pen-testing, load testing, runtime fuzzing) not performed; only static analysis
- **Status:** 🟡 Review in Progress — findings C-1, C-2, H-1 resolved 2025-09; Session TTL, admin revocation, per-email brute-force resolved 2025-09; audit log skipped per request
