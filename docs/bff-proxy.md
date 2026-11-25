# BFF / Proxy Architecture

## Overview

Somotracker uses a **Backend-for-Frontend (BFF)** pattern. The Next.js frontend does **not**
call the Go backend directly from the browser. Instead:

```
Browser ──▶ Next.js API Routes (BFF) ──▶ Go Backend API
                                               │
                                               ▼
                                        PostgreSQL + Redis
```

All frontend API calls go through Next.js route handlers under `frontend/src/app/api/`, which
proxy to the Go backend at `backend/internal/app/`. The Go backend sets session cookies directly
on responses that flow through Next.js to the browser.

## Why BFF

1. **HttpOnly cookies**: Session tokens must never be accessible to JavaScript. The Go backend
   sets `HttpOnly` cookies on the HTTP response, and Next.js forwards `Set-Cookie` headers to
   the browser.
2. **Secret isolation**: `TURNSTILE_SECRET_KEY`, `BREVO_API_KEY`, `EMAIL_HMAC_KEY`, etc. never
   leave the Go backend.
3. **Single CORS surface**: CORS is configured on the Go API to allow only the Next.js origin.
4. **Cookie forwarding**: Next.js route handlers forward the browser's session cookie to the Go
   backend on every mutating request.

## Request Flow

### OTP Request (public, no session)

```
Browser
  │ POST /api/auth/otp/request  { email, turnstile_token }
  ▼
Next.js route: frontend/src/app/api/auth/otp/request/route.ts
  │ forwards to: POST /api/v1/auth/otp/request
  ▼
Go handler: backend/internal/auth/handler/handler.go (RequestOTP)
  │ rate limit → turnstile verify → OTP → Brevo email
  ▼
Go response: { ok, message, challenge_id }
  │ forwards response (no cookies to set)
  ▼
Browser
```

### OTP Verify (public, sets cookies)

```
Browser
  │ POST /api/auth/otp/verify  { challenge_id, code }
  ▼
Next.js route: frontend/src/app/api/auth/otp/verify/route.ts
  │ forwards to: POST /api/v1/auth/otp/verify
  ▼
Go handler: VerifyOTP
  │ verify HMAC → create session → set cookies
  ▼
Go response: { ok, message, needs_onboarding }
  + Set-Cookie: __Host-session=<token>, needs_onboarding=true
  ▼
Next.js forwards Set-Cookie headers to browser
  ▼
Browser stores HttpOnly cookies
```

### Authenticated API Call (has session)

```
Browser
  │ GET /api/auth/session   Cookie: session=<token>
  ▼
Next.js route
  │ forwards cookie to Go backend
  ▼
Go handler: Session
  │ validate session → return status
  ▼
Browser receives JSON (no new cookies)
```

## Route Mapping

| Frontend Route | Go Endpoint | Method | Auth Required |
|----------------|-------------|--------|---------------|
| `/api/auth/otp/request` | `/api/v1/auth/otp/request` | POST | No |
| `/api/auth/otp/verify` | `/api/v1/auth/otp/verify` | POST | No |
| `/api/auth/onboard` | `/api/v1/auth/onboard` | POST | Yes (temporary session) |
| `/api/auth/session` | `/api/v1/auth/session` | GET | No (returns auth status) |
| `/api/auth/logout` | `/api/v1/auth/logout` | POST | No (idempotent) |

## Cookie Forwarding

Next.js route handlers read the session cookie from `next/headers` `cookies()` and forward it
to the Go backend as a `Cookie` header. `Set-Cookie` headers from Go responses are forwarded
back to the browser (except when handled specially by `api-client.ts`).

## Environment Variables (Frontend)

| Variable | Required | Description |
|----------|----------|-------------|
| `API_URL` | Yes | Go backend API URL (e.g. `http://localhost:3030`) |
| `APP_URL` | Yes | Public frontend URL (e.g. `http://localhost:3000`) |
| `SESSION_COOKIE_NAME` | Yes | Must match backend |
| `APP_ENV` | Yes | `development` or `production` |
| `NODE_ENV` | Yes | Node environment |
| `PORT` | Yes | Frontend port (3000) |
| `HOSTNAME` | Yes | Hostname for Next.js |

## Environment Variables (Go)

| Variable | Required | Description |
|----------|----------|-------------|
| `CORS_ORIGINS` | Yes | Comma-separated allowed origins (Next.js frontend origin only) |
| `SESSION_COOKIE_NAME` | Yes | Must match frontend |
| `TURNSTILE_SECRET_KEY` | Yes | Turnstile secret key |
| `TURNSTILE_EXPECTED_HOST` | Recommended | Expected hostname in Turnstile response |

## CORS

The Go API configures CORS to allow only the Next.js frontend origin (from `CORS_ORIGINS`).
Credentials are enabled when origins are specified. Wildcard `*` is used only when
`CORS_ORIGINS` is empty.

## Session Cookie

| Environment | Cookie Name | Secure | Notes |
|-------------|-------------|--------|-------|
| Development | `session` | false | HTTP localhost |
| Production | `__Host-session` | true | Requires HTTPS, host-only |

Attributes: `HttpOnly`, `SameSite=Lax`, `Path=/`, no `Domain` attribute.
