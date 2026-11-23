# Somotracker — Structural Baseline

**Status:** Current as of the OTP/auth rework. This document describes the actual codebase state.

**Purpose:** Verified facts extracted from source code. This is a scratchpad, not a governing spec. No assumptions, no invented rules.

## Monorepo layout

```
/somo-tracker
├── backend/               Go, Fiber v3, pgx, sqlc, PostgreSQL, Redis, Brevo
├── frontend/              Next.js 16.4.6, React 19.3.1, App Router, Tailwind + shadcn/ui
├── marketing/             Astro static site
├── docker-compose.yml     Postgres 18, Redis 7, backend
├── Makefile
└── .env                   (Doppler-injected secrets at runtime, no .env committed)
```

## Backend structure (`backend/`)

```
backend/
├── cmd/api/main.go                    # entry point
├── internal/
│   ├── app/
│   │   └── app.go                     # composition root, manual DI, route registration, middleware chain
│   ├── auth/
│   │   ├── handler/handler.go         # OTP request/verify, onboard, session, logout handlers
│   │   ├── handler/handler_test.go    # unit tests for auth handlers
│   │   ├── service/service.go         # OTP, session, onboarding business logic
│   │   ├── service/service_test.go    # unit tests for auth service
│   │   ├── service/service_integration_test.go # integration tests
│   │   └── store/
│   │       ├── store.go               # DB access via sqlc-generated queries
│   │       ├── store_interface.go     # StoreInterface definition
│   │       ├── queries/auth.sql       # raw SQL compiled by sqlc
│   │       └── gen/                   # sqlc-generated code
│   └── platform/                      # infrastructure only
│       ├── apperr/                    # typed errors (incl. BAD_GATEWAY)
│       ├── brevo/                     # email provider (CodeSender interface)
│       ├── config/                    # typed env config (incl. Turnstile fields)
│       ├── db/                        # pgx pool
│       ├── httpserver/                # Fiber server, CORS, error handler (502 mapping)
│       ├── logger/                    # slog JSON
│       ├── ratelimit/                 # Redis-backed rate limiter (NEW)
│       ├── ratelimit/ratelimit_test.go # unit tests for rate limiter
│       ├── redis/                     # Redis client wrapper with Increment/Get
│       └── turnstile/                 # Cloudflare Turnstile verification (NEW)
│           ├── turnstile.go           # Verifier interface + siteverify client
│           └── turnstile_test.go      # unit tests (success/failure/timeout/malformed/hostname)
├── migrations/                        # goose SQL
└── sqlc.yaml
```

## Auth flow (current)

1. **POST /api/v1/auth/otp/request** — email + `turnstile_token` → Turnstile verify → rate limit → OTP → email via Brevo → challenge ID
2. **POST /api/v1/auth/otp/verify** — challenge_id + code → HMAC verify → session → cookie
3. **POST /api/v1/auth/onboard** — org/school/user name → upgrade session → clear onboarding cookie
4. **GET /api/v1/auth/session** — session info
5. **POST /api/v1/auth/logout** — clear cookies

## Rate limiting

- OTP request endpoint rate-limited per IP (per minute) and per email (per 15 min).
- Config: `RateLimitOTPRequestIPPerMin` (default 10), `RateLimitOTPRequestEmailPer15Min` (default 3).
- Backed by Redis atomic counters.

## Turnstile

- OTP request requires `turnstile_token` in body (400 if missing).
- Verified with Cloudflare siteverify before OTP generation, DB access, or email sending.
- Upstream failure → 502 `BAD_GATEWAY`; failed verification → 403 `FORBIDDEN`.
- Config: `TURNSTILE_SECRET_KEY` (required), `TURNSTILE_EXPECTED_HOST` (recommended).

## Configuration (Doppler-injected)

Required: `DATABASE_URL`, `REDIS_URL`, `BREVO_API_KEY`, `EMAIL_HMAC_KEY`, `SESSION_COOKIE_NAME`, `TURNSTILE_SECRET_KEY`, `CORS_ORIGINS`.
Recommended: `TURNSTILE_EXPECTED_HOST`.

## Database migrations (`backend/migrations/`)

Goose-managed SQL migrations including: `login_challenges`, `email_outbox`, `sessions`, `users`, `organizations`, `schools`.

## Rules

- **DO NOT RUN DOCKER** — Docker is managed separately.
- No fallback defaults in `config.go`.
- Required config validated at startup in `config.validate()`.
- Handler → Service → Store layer separation enforced.
