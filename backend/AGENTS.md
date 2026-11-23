# Backend Agent Rules

## Architecture
- **Layer separation**: Handlers → Services → Store → sqlc-generated queries. Handlers must not call the DB directly; they delegate to services.
- **Error contract**: All non-2xx responses use the canonical `apperr.Error` type with a stable `Code` and human-readable `Message`. The Fiber error handler maps codes to HTTP status codes.
- **Dependency injection**: `app.New(cfg)` is the composition root. All dependencies (store, service, handler, verifier) are constructed manually there.
- **Configuration**: All values come from environment variables via `firstNonEmpty()` with no fallback defaults. Required values are validated in `config.validate()` at startup.

## Required Configuration
| Variable | Required | Description |
|----------|----------|-------------|
| `DATABASE_URL` | Yes | PostgreSQL connection string |
| `REDIS_URL` | Yes | Redis connection string |
| `BREVO_API_KEY` | Yes | Brevo email provider API key |
| `EMAIL_HMAC_KEY` | Yes | HMAC key for OTP code signing |
| `SESSION_COOKIE_NAME` | Yes | Session cookie name (`session` dev, `__Host-session` prod) |
| `TURNSTILE_SECRET_KEY` | Yes | Cloudflare Turnstile secret key (server-side only) |
| `TURNSTILE_EXPECTED_HOST` | Recommended | Expected hostname in Turnstile verification response |
| `CORS_ORIGINS` | Yes | Comma-separated allowed CORS origins |

## Testing
- Unit tests: `cd backend && go test ./...`
- Integration tests (local only, needs Docker): `cd backend && go test -tags=integration ./...`
- Startup integration test: `cd backend && go test -tags=integration -v -run TestStartup ./internal/app/`

## Rate Limiting
- OTP request endpoint is rate-limited per IP and per email using Redis atomic counters.
- Config: `RateLimitOTPRequestIPPerMin`, `RateLimitOTPRequestEmailPer15Min`.
- Rate limited responses return `apperr.RATE_LIMITED` → HTTP 429.

## Turnstile
- OTP request requires `turnstile_token` in the body (400 if missing).
- Verified with Cloudflare siteverify before generating an OTP, touching the DB, or sending email.
- Upstream failure → 502 `BAD_GATEWAY`; failed verification → 403 `FORBIDDEN`.
- Cloudflare `error-codes` are logged server-side only, never returned to the client.

## Rules
- **DO NOT RUN DOCKER** — Docker is managed separately.
- No fallback defaults in `config.go`.
- Migration testing required for every new migration SQL file.
- Run `go vet ./...` and `golangci-lint run ./...` before committing.
