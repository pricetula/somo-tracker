# ADR-0005: Rate Limiting Strategy

## Metadata

| Field | Value |
|-------|-------|
| id | ADR-0005 |
| title | Rate Limiting Strategy |
| status | proposed |
| date | 2026-11-25 |
| author | engineering |
| deciders | backend, platform |
| relates_to | ADR-0001, ADR-0003, ADR-0004 |

## Status

proposed

## Context

SPEC-001 requires rate limiting on OTP request (per normalized email, per client IP, and global) without revealing which limit was hit. The codebase implements rate limiting across phases 0-4 (see prior feature branch `feat/rate-limiting-phases-0-4`) using Redis shared atomic counters with a token-bucket-like allow/deny decision.

Key forces:
- OTP request must be rate limited per email and per IP to prevent abuse and email bombing.
- OTP verify must be rate limited per challenge to prevent brute-force on 6-digit codes.
- Onboarding must be rate limited per user (expensive operation).
- Limits must not reveal whether an email exists (generic error responses).
- Redis is the shared atomic counter store across API instances.
- A fallback/local limiter exists for when Redis is unavailable.
- Bot defense (Turnstile) raises limits for verified humans and lowers for bots.
- Integration tests use `postgres:18` and Redis 7 via testcontainers; PostgreSQL 18 is the standard everywhere.

Key decisions needed:
- Limit thresholds per endpoint/scope.
- Redis key structure and TTL strategy.
- Fallback behavior when Redis is down.
- Interaction with Turnstile bot defense.
- Observability (headers, logging, alerts).

## Decision

Adopt Redis-backed shared atomic rate limiting with per-key scoped limits and graceful fallback:

1. **Redis store**: Shared atomic counters (`INCR` + `EXPIRE` on first increment) keyed by `{scope}:{identifier}` (e.g. `otp_request:email:{normalized}`, `otp_request:ip:{ip}`, `otp_request:global`, `otp_verify:challenge:{id}`, `onboard:user:{id}`).
2. **Limit scopes**:
   - OTP request: per normalized email, per client IP, and global budget.
   - OTP verify: per challenge ID (prevents brute-force on a single challenge).
   - Onboard: per user ID (onboard is expensive).
3. **Decision model**: `Allow(key, limit, window)` returns `Allowed` boolean plus `Remaining` and `Reset` for headers. If any scope is exceeded, return 429 `RATE_LIMITED` with a generic message.
4. **Fallback**: When Redis is unavailable, use a local in-memory limiter (per-process) with the same limits; degrade gracefully rather than blocking all traffic.
5. **Bot defense**: Cloudflare Turnstile verification (when `TURNSTILE_ENABLED=true`) influences limits — verified humans get higher limits; failed/missing Turnstile gets stricter limits or 400 `TURNSTILE_REQUIRED`.
6. **Headers**: `X-RateLimit-Limit`, `X-RateLimit-Remaining`, `X-RateLimit-Reset` set on responses when rate limiter is active.
7. **Configuration**: All limits injected via Doppler (`RateLimitOtpRequestEmailPerMin`, `RateLimitOtpRequestIPPerMin`, `RateLimitOtpRequestGlobalPerMin`, `RateLimitOtpVerifyPerMin`, `RateLimitOnboardUserPerMin`, etc.). No fallback defaults in `config.go`.
8. **Validation**: Required config values validated at startup in `config.validate()`.

## Alternatives Considered

| Alternative | Pros | Cons | Why rejected |
|-------------|------|------|--------------|
| In-memory only | Simple, no Redis dependency | Per-instance limits don't aggregate across replicas; reset on restart | Rejected — need shared state across API instances |
| Lua scripts in Redis | Atomic check-and-increment in one round-trip | More complex, harder to debug, Redis version constraints | Simple INCR+EXPIRE is sufficient for current scale |
| Token bucket (Redis) | Smoother burst allowance | More state per key, complex implementation | Fixed-window counters are simpler and adequate |
| API gateway rate limiting | Offloads from application | Requires gateway infrastructure; less flexible per-endpoint logic | Application-level control preferred |

## Consequences

### Positive

- Shared counters across all API instances via Redis.
- Generic 429 responses prevent limit-based enumeration.
- Graceful fallback keeps the system available during Redis outages.
- Turnstile integration reduces abuse from bots while easing limits for humans.
- All limits are configurable via Doppler without code changes.

### Negative

- Redis dependency for accurate cross-instance rate limiting (fallback mitigates outage).
- Fixed-window counters allow burst at window boundaries (acceptable trade-off).
- Local fallback limits are per-process; under Redis outage with N instances, effective limit is N× configured limit.

### Neutral / Risks

- Fallback limiter must be kept in sync with Redis limits when config changes.
- No distributed rate-limit dashboard yet; limits observable only via headers/logs.
- Global OTP budget (`RateLimitOtpRequestGlobalPerMin`) could block all OTP traffic during a targeted attack; needs alerting.
- Outbox email delivery retries (ADR-0004) are internal attempts and do NOT count against the global OTP request rate limit. Only client-initiated OTP requests consume rate-limit budget.

## Links

- Spec: `docs/specs/001-auth-otp-request.md` (§Rate limits)
- Auth ADR: `docs/adr/0001-authentication-architecture.md`
- Session ADR: `docs/adr/0003-session-management.md`
- Email outbox ADR: `docs/adr/0004-email-outbox-delivery.md`
- Config: `backend/internal/platform/config/config.go` (`firstNonEmpty`, `validate()`)
- Rate limit platform: `backend/internal/platform/ratelimit/`
- Alerts: `backend/monitoring/alerts/ratelimit.yaml`
- Integration test standard: `backend/internal/auth/service/service_integration_test.go` (uses `postgres:18`)
