# Security Overview

## Authentication

Somotracker uses **email-based OTP (One-Time Password)** authentication with optional Cloudflare Turnstile bot protection.

### OTP Flow

1. User submits email + Turnstile token to `POST /api/v1/auth/otp/request`
2. Backend verifies Turnstile token with Cloudflare (server-side only)
3. Backend generates a 6-digit OTP, stores `HMAC-SHA256(secret, challenge_id || code)` — **never plaintext**
4. Backend sends OTP via Brevo (fire-and-forget)
5. User submits challenge_id + code to `POST /api/v1/auth/otp/verify`
6. Backend verifies HMAC in constant time, creates a session, sets HttpOnly cookie

### Session Management

- Session token is 256-bit opaque random value
- Only `SHA-256(token)` is stored in the database
- Cookie is `HttpOnly`, `SameSite=Lax`, `Secure` in production
- Production uses `__Host-session` prefix (enforces HTTPS, host-only)
- Temporary sessions (new users) last 30 minutes; full sessions last 30 days

## Bot Protection (Cloudflare Turnstile)

- `turnstile_token` is **required** in the OTP request body (400 if missing)
- Verified with Cloudflare `siteverify` before any OTP generation, DB access, or email sending
- 5-second timeout for siteverify calls
- Upstream failure → 502 `BAD_GATEWAY` (fail closed)
- Failed verification → 403 `FORBIDDEN`
- Cloudflare `error-codes` are logged server-side only, never returned to the client
- Expected hostname check via `TURNSTILE_EXPECTED_HOST` (if configured)

**Turnstile is not rate limiting.** Rate limiting is implemented separately (see below).

## Rate Limiting

The OTP request endpoint is rate-limited per IP and per email using Redis atomic counters:

| Limit | Default | Scope |
|-------|---------|-------|
| `RateLimitOTPRequestIPPerMin` | 10 | Per IP address per minute |
| `RateLimitOTPRequestEmailPer15Min` | 3 | Per email per 15 minutes |

- Exceeding either limit → 429 `RATE_LIMITED`
- Generic error message (does not reveal which limit was hit)
- Uses `CF-Connecting-IP` header when behind Cloudflare, otherwise connection IP

## Account Enumeration Protection

- `POST /api/v1/auth/otp/request` returns the **same generic response** whether or not the email exists
- `POST /api/v1/auth/otp/verify` returns `INVALID_TOKEN` for any invalid challenge (expired, used, wrong code, not found) — no distinction
- `GET /api/v1/auth/session` returns `authenticated: false` for missing/invalid sessions (HTTP 200 either way)
- `POST /api/v1/auth/logout` returns 200 regardless of session state (idempotent)

## CORS

- Go API allows only the Next.js frontend origin (`CORS_ORIGINS`)
- Credentials enabled when origins are specified
- Wildcard `*` only when `CORS_ORIGINS` is empty

## Configuration Security

- All secrets injected by **Doppler** at runtime — no fallback defaults in `config.go`
- Required values validated at startup in `config.validate()` — app exits if missing
- `TURNSTILE_SECRET_KEY` is server-side only, never exposed to frontend
- Site key `0x4AAAAAAGHleNH8xjjhEUQL` is hard-coded in the frontend Turnstile component (public by design — it's meant to be embedded in the page)

## Data Protection

- Emails are normalized (lowercased, trimmed) before storage
- `normalized_email` is UNIQUE in the users table
- OTP codes are HMAC'd, never stored plaintext
- Session tokens are SHA-256 hashed before storage
- Soft deletes via `deleted_at` on relevant tables

## Known Gaps / Future Work

- **Server-side session revocation**: Logout clears cookies but does not revoke the token in the database. Token expires naturally via `expires_at`.
- **Resend OTP**: Not yet implemented (see SPEC-002 open questions).
- **Password-based auth**: Not applicable — OTP only.
- **MFA / 2FA**: Not yet implemented.
- **Audit logging**: Not yet implemented for auth events.
