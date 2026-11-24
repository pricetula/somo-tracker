# SPEC-010: Rate Limiting Phase 0 — Trust Boundary & BFF Header Plumbing

Endpoint: `ALL /api/v1/*` (middleware)

## Metadata

| Field | Value |
|-------|-------|
| id | SPEC-010 |
| title | Rate Limiting Phase 0 — Trust Boundary & BFF Header Plumbing |
| status | draft |
| depends_on | docs/adr/0002-rate-limiting-strategy.md, specs/glossary.md |
| api_endpoint | ALL /api/v1/* |
| owner | backend |

## Summary

Implement the **mandatory BFF service credential verification** and **trusted client IP extraction** at the Go API edge. This is the foundational security layer that prevents direct public-Render-URL bypass and establishes the trust boundary between the BFF and the Go API. Without this, all downstream rate limiting can be spoofed.

## Components

### 1. BFF_SERVICE_SECRET Middleware (`internal/app/app.go`)

**Order (first after exempt paths):**
1. Exempt path check (`/healthz`, `/readyz`, signature-verified webhooks)
2. **BFF_SERVICE_SECRET verification** — reject with `403 Forbidden` before any session lookup
3. Trusted CIDR check (optional, `TRUSTED_BFF_CIDRS`)
4. Client IP extraction from `X-Client-IP` (verified by BFF)
5. Session resolution middleware (attaches `userID` to context)
6. Rate limit middleware (keys on `userID` when available)

### 2. BFF Header Forwarding (BFF Service — Next.js)

The BFF **must** perform these transformations on every proxied request:

| Action | Detail |
|--------|--------|
| Strip incoming `X-Client-IP` | Remove any client-supplied header |
| Extract real client IP | Parse `X-Forwarded-For` using `TRUSTED_PROXY_HOPS` (count from right) |
| Normalize IP | IPv4 → `/32`, IPv6 → `/64` prefix |
| Set `X-Client-IP` | Forward verified normalized IP to Go API |
| Set `X-BFF-Service-Secret` | Value from `BFF_SERVICE_SECRET` (Doppler) |
| Forward session token | Map `HttpOnly` session cookie → `Authorization: Bearer <token>` or forward cookie |

### 3. Configuration (Doppler)

```bash
# Required in production
BFF_SERVICE_SECRET=<32+ char random secret>      # Mandatory; API rejects without it
TRUSTED_PROXY_HOPS=1                             # Verify in staging; count from right of X-Forwarded-For
TRUSTED_BFF_CIDRS=                               # Optional CIDR allowlist (e.g., Render IP ranges)
RATE_LIMIT_KEY_HMAC_SECRET=<secret>              # For HMAC-SHA256(email) keys
```

## Request Flow

```
Browser → BFF (Next.js) → Go API
                │
                ├── Strip X-Client-IP
                ├── Parse X-Forwarded-For[TRUSTED_PROXY_HOPS from right]
                ├── Normalize to /32 or /64
                ├── Set X-Client-IP: <verified-ip>
                ├── Set X-BFF-Service-Secret: <secret>
                ├── Forward session cookie/token
                └── Proxy to Go API
```

```
Go API Middleware Chain:
1. ExemptPaths("/healthz", "/readyz", webhooks...)
2. VerifyBFFServiceSecret() → 403 if missing/invalid
3. (Optional) VerifyTrustedBFFCIDR() → 403 if not in range
4. ExtractClientIP() from X-Client-IP header
5. ResolveSession() → attach userID to context (if session present)
6. RateLimitMiddleware() → keys on userID > challengeID > email > IP
```

## Business Rules

1. **Exempt paths**: `/healthz`, `/readyz`, and signature-verified webhooks skip BFF secret check
2. **Constant-time comparison**: Use `crypto/subtle.ConstantTimeCompare` for `BFF_SERVICE_SECRET`
3. **Fail fast**: Invalid/missing `BFF_SERVICE_SECRET` → `403 Forbidden` immediately (no session lookup, no rate limit check)
4. **IP normalization**: IPv4 mask `/32`, IPv6 mask `/64` — applied at BFF, trusted at API
5. **No fallback**: If `X-Client-IP` missing after BFF secret verification → `400 Bad Request` (BFF misconfiguration)

## Response (Errors)

| Status | Code | Condition |
|--------|------|-----------|
| 400 | `INVALID_REQUEST` | `X-Client-IP` header missing after BFF secret verification |
| 403 | `FORBIDDEN` | Missing/invalid `X-BFF-Service-Secret` or IP not in `TRUSTED_BFF_CIDRS` |
| 500 | `INTERNAL_ERROR` | Unexpected server error |

## Implementation Files

| File | Responsibility |
|------|----------------|
| `internal/app/app.go` | Middleware chain wiring; exempt path list |
| `internal/platform/middleware/bffauth.go` | `VerifyBFFServiceSecret`, `VerifyTrustedBFFCIDR`, `ExtractClientIP` |
| `internal/platform/config/config.go` | Doppler parsing for `BFF_SERVICE_SECRET`, `TRUSTED_PROXY_HOPS`, `TRUSTED_BFF_CIDRS`, `RATE_LIMIT_KEY_HMAC_SECRET` |
| BFF: `apps/bff/middleware/proxy-headers.ts` | IP extraction, header injection, session forwarding |

## Tests Required

| Test Type | What to Verify |
|-----------|----------------|
| Unit | `VerifyBFFServiceSecret` uses constant-time compare |
| Unit | `ExtractClientIP` parses `X-Forwarded-For` with `TRUSTED_PROXY_HOPS=1,2,3` |
| Unit | IPv4 → `/32`, IPv6 → `/64` normalization |
| Unit | Dual-secret verification accepts both old and new secrets |
| Integration | Request without `X-BFF-Service-Secret` → `403` |
| Integration | Request with invalid secret → `403` |
| Integration | Request with valid secret but no `X-Client-IP` → `400` |
| Integration | Exempt paths (`/healthz`, `/readyz`) bypass secret check |
| Integration | `TRUSTED_BFF_CIDRS` blocks non-matching IPs |
| Integration | Valid secret + valid IP → middleware chain continues |
| Contract | All `/api/v1/*` routes require valid BFF secret |
| Contract | BFF forwards correct headers to Go API |
| Contract | `403` response uses RFC 9457 problem details (`code: forbidden`) |
| Contract | `400` response uses RFC 9457 problem details (`code: invalid_request`) |
| Contract | No rate limit headers on `403`/`400` responses (fail-fast) |
| Chaos | BFF secret rotation mid-load: dual-secret works, no 403 spikes |
| Chaos | `TRUSTED_PROXY_HOPS` change mid-load: IP extraction adapts |
| Security | Direct request to API public URL without BFF secret → `403` |
| Security | Spoofed `X-Forwarded-For` with `TRUSTED_PROXY_HOPS=1` → correct IP extracted |
| Security | Spoofed `X-Forwarded-For` with multiple hops → only trusted hop used |
| Security | Timing attack on secret verification → constant-time resistant |

## Acceptance Criteria

Given a request to `/api/v1/auth/otp/request` without `X-BFF-Service-Secret`
When the request reaches Go API
Then response is `403 Forbidden` with code `FORBIDDEN`
And no session lookup occurs
And no rate limit check occurs

Given a request with valid `X-BFF-Service-Secret` but missing `X-Client-IP`
When the request reaches Go API
Then response is `400 Bad Request` with code `INVALID_REQUEST`

Given `TRUSTED_PROXY_HOPS=2` and `X-Forwarded-For: 1.2.3.4, 10.0.0.1, 192.168.1.1`
When BFF extracts client IP
Then `X-Client-IP` = `10.0.0.1` (2nd from right)
And Go API receives normalized `/32` IPv4

Given a request to `/healthz`
When the request reaches Go API
Then response is `200 OK` regardless of BFF secret

## Open Questions

- Dual-secret rotation strategy for `BFF_SERVICE_SECRET` (zero-downtime)?
- Exact Render CIDR ranges for `TRUSTED_BFF_CIDRS`?
- Staging verification procedure for `TRUSTED_PROXY_HOPS`?

## Links

- ADR: `docs/adr/0002-rate-limiting-strategy.md` §1, §9
- Glossary: `specs/glossary.md` (entities: `bff_service_secret`, `client_ip`, `trusted_proxy_hops`)
- BFF Proxy Doc: `docs/bff-proxy.md`