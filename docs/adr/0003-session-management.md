# ADR-0003: Session Management

## Metadata

| Field | Value |
|-------|-------|
| id | ADR-0003 |
| title | Session Management |
| status | proposed |
| date | 2026-11-25 |
| author | engineering |
| deciders | backend, frontend |
| relates_to | ADR-0001, ADR-0002, ADR-0005 |

## Status

proposed

## Context

The authentication flow issues sessions after OTP verification. Sessions control access to protected routes and must be invalidatable, time-bounded, and resistant to token theft. The codebase currently supports two session types:

- `temporary`: 30-minute TTL, issued to new users who have not completed onboarding.
- `full`: 30-day TTL, issued to existing users or after onboarding completes.

Key forces:
- Session tokens are opaque 256-bit values; SHA-256 hash is stored in PostgreSQL, raw token only in the browser cookie.
- Cookies use `HttpOnly`, `SameSite=Lax`, `Path=/`. Production uses `__Host-session` with `Secure` (HTTPS only); dev uses `session` without Secure.
- A `needs_onboarding` flag cookie accompanies temporary sessions so the frontend proxy can route without a backend call.
- Sessions are upgraded from `temporary` to `full` during onboarding (`UpdateSessionType`).
- Redis is available but sessions are stored in PostgreSQL; Redis is used for rate limiting and ephemeral data.

Key decisions needed:
- Where to store sessions (PostgreSQL vs Redis vs hybrid).
- How to handle session revocation and rotation.
- Cookie prefix and attribute strategy across environments.
- Temporary session upgrade path.

## Decision

Adopt PostgreSQL-backed opaque sessions with the following characteristics:

1. **Storage**: Sessions stored in the `sessions` table in PostgreSQL. Columns: `id`, `token_hash` (SHA-256 of raw token), `user_id`, `organization_id`, `active_school_id`, `session_type`, `created_at`, `last_seen_at`, `expires_at`, `revoked_at`, `revocation_reason`.
2. **Token lifecycle**: Generate 256-bit opaque token on verify; store SHA-256 hash; set raw token as cookie. Hash comparison prevents token extraction if DB is read.
3. **Session types**: `temporary` (30 min TTL) for pre-onboarding users; `full` (30-day TTL, `SessionAbsoluteTimeout()`) for all others. Configurable via config.
4. **Cookie strategy**:
   - Production: `__Host-session`, Secure, HttpOnly, SameSite=Lax, Path=/.
   - Development: `session`, HttpOnly, SameSite=Lax, Path=/ (Secure=false for HTTP localhost).
   - `needs_onboarding` flag cookie for temporary sessions; cleared on onboarding completion.
5. **Validation**: On each authenticated request, hash the cookie token, lookup `sessions` where `revoked_at IS NULL AND expires_at > NOW()`. Return 401 if missing/invalid/expired.
6. **Upgrade**: Onboarding calls `UpdateSessionType` to set all temporary sessions for the user to `full`.
7. **Revocation**: Soft revocation via `revoked_at` + `revocation_reason`; hard delete not required.
8. **Redis role**: Redis is NOT used for session storage (PostgreSQL is source of truth). Redis is used for rate limiting counters and ephemeral OTP/challenge data.

## Alternatives Considered

| Alternative | Pros | Cons | Why rejected |
|-------------|------|------|--------------|
| Redis-backed sessions | Fast lookups, built-in TTL | Eviction risk under memory pressure, no durable revocation log | PostgreSQL preferred for durability and revocation history |
| JWT (self-contained) | No DB lookup per request | Hard to revoke, larger cookies, key rotation complexity | Opaque tokens + DB allow instant revocation |
| Short-lived JWT + refresh token | Reduced DB load, rotation possible | Adds token refresh endpoint complexity | Single opaque session is simpler for current scale |
| Redis cache in front of PostgreSQL | Faster hot-session lookups | Cache invalidation complexity, stale session risk | Not needed yet; PostgreSQL performance is adequate |

## Consequences

### Positive

- Instant session revocation via `revoked_at`.
- Durable session audit trail (created/expired/revoked timestamps).
- Simple mental model: cookie → hash → DB lookup.
- `__Host-` prefix in production enforces HTTPS and prevents cookie injection by subdomains.

### Negative

- Every authenticated request incurs a PostgreSQL session lookup (mitigated by short-lived connections and indexing on `token_hash`).
- Session table growth over time; requires periodic cleanup of expired/revoked sessions (not yet implemented).
- No distributed session cache; horizontal scaling relies on shared PostgreSQL (already the case).

### Neutral / Risks

- Session cleanup job is not yet implemented; expired sessions accumulate until manually pruned.
- `last_seen_at` is selected but not yet updated on each request (future enhancement for idle timeout).
- If PostgreSQL is unavailable, all authenticated requests fail (no fallback cache).

## Links

- Auth ADR: `docs/adr/0001-authentication-architecture.md`
- Multi-tenancy ADR: `docs/adr/0002-multi-tenancy-model.md`
- Rate limiting ADR: `docs/adr/0005-rate-limiting-strategy.md`
- Store: `backend/internal/auth/store/store.go` (`CreateSession`, `GetActiveSession`, `GetSessionByTokenHash`, `UpdateSessionType`)
- Migration: `backend/migrations/00000000000004_onboarding.sql` (`sessions.session_type`)
- Flow: `docs/auth-flow.md` (Step 2, Step 4)
