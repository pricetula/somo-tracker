# Backend & Frontend Secrets Reference

All configuration is injected by **Doppler** at runtime. Nothing is read from the repo (`.env` is gitignored and contains no real values). Missing required values cause the backend to fail fast at startup — no silent fallbacks.

## Backend Environment Variables

| Variable | Required | Example | Description |
|----------|----------|---------|-------------|
| `APP_ENV` | Yes | `development` | `development`, `test`, or `production` |
| `LOG_LEVEL` | Yes | `debug` | `debug`, `info`, `warn`, `error` |
| `SERVER_PORT` | Yes | `3030` | HTTP port the backend listens on |
| `DATABASE_URL` | Yes | `postgres://somotracker:somotracker@postgres:5432/somotracker?sslmode=disable` | PostgreSQL connection string |
| `REDIS_URL` | Yes | `redis://redis:6379` | Redis connection string (Docker service name `redis`) |
| `BREVO_API_KEY` | Yes | `keys.your_brevo_key` | Brevo email provider API key |
| `BREVO_SENDER_EMAIL` | Yes | `noreply@somotracker.com` | From email address |
| `BREVO_SENDER_NAME` | Yes | `Somotracker` | From display name |
| `EMAIL_HMAC_KEY` | Yes | 32-byte HMAC key | Signs OTP codes in email links |
| `SESSION_COOKIE_NAME` | Yes | `session` (dev) / `__Host-session` (prod) | Session cookie name |
| `TURNSTILE_ENABLED` | Yes | `true` / `false` | Enable Cloudflare Turnstile bot protection |
| `TURNSTILE_SECRET_KEY` | Yes (if enabled) | `0x...` | Turnstile secret key for server-side verification |
| `CORS_ORIGINS` | Yes | `http://localhost:3000` | Allowed CORS origins |
| `SEED_REFERENCE_DATA` | No | `true` / `false` | Seed reference data on startup |

## Frontend Environment Variables

| Variable | Required | Example | Description |
|----------|----------|---------|-------------|
| `PUBLIC_APP_URL` | Yes | `http://localhost:3000` | Public frontend URL |
| `PUBLIC_API_URL` | Yes | `http://localhost:3030` | Public backend API URL |
| `NEXT_PUBLIC_TURNSTILE_SITE_KEY` | Yes (if enabled) | `0x...` | Turnstile site key for client-side widget |
| `NODE_ENV` | Yes | `development` | Node environment |
| `SESSION_COOKIE_NAME` | Yes | `session` | Must match backend |
| `APP_ENV` | Yes | `development` | Must match backend |
| `HOSTNAME` | Yes | `0.0.0.0` | Hostname for Next.js |
| `PORT` | Yes | `3000` | Frontend port |

## Docker Compose Service Variables (internal, not Doppler)

| Variable | Example | Description |
|----------|---------|-------------|
| `POSTGRES_DB` | `somotracker` | Postgres database name |
| `POSTGRES_USER` | `somotracker` | Postgres username |
| `POSTGRES_PASSWORD` | `somotracker` | Postgres password |

## Validation

The backend validates required values in `config.validate()` on startup. If any required variable is missing or empty, the app exits immediately with:

```
invalid configuration: missing required environment variables: DATABASE_URL, BREVO_API_KEY, ...
```

## Adding a New Secret

1. Add the variable to Doppler (development + any other environments).
2. Read it in `backend/internal/platform/config/config.go` via `firstNonEmpty("YOUR_VAR")` — **no fallback default**.
3. If it's required, add it to the `required` map in `config.validate()`.
4. Update this doc (`docs/secrets.md`).

## Troubleshooting

- `connect redis: connection refused` → `REDIS_URL` is empty or pointing at `localhost` instead of the `redis` service name.
- `missing required environment variables` → Doppler isn't injecting that variable; check your Doppler config.
- `invalid configuration` → same as above; the app won't start until all required vars are present.
