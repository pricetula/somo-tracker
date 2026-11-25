# ADR-0001: Authentication Architecture

## Metadata

| Field | Value |
|-------|-------|
| id | ADR-0001 |
| title | Authentication Architecture |
| status | proposed |
| date | 2026-11-25 |
| author | engineering |
| deciders | backend, frontend |
| relates_to | SPEC-001, specs/glossary.md, ADR-0003, ADR-0004 |

## Status

proposed

## Context

SPEC-001 defines the OTP-based login flow (request → verify → session) and references `auth-architecture-v2.md` §4.2, §4.3, §11, §12.2, which does not exist. The codebase implements email-based OTP with optional Cloudflare Turnstile bot protection, supporting both new-user signup (with onboarding) and existing-user signin.

Key forces:
- No password storage: users authenticate exclusively via time-limited OTP codes delivered by email.
- New users require onboarding (organization, school, display name) before full access.
- Temporary sessions (30-minute TTL) are issued for new users until onboarding completes; full sessions (30-day TTL) for existing users.
- Cookies must be HttpOnly and Secure in production, with `__Host-` prefix enforcement over HTTPS.
- Session tokens are SHA-256 hashed before storage; the raw token exists only in the browser cookie.
- OTP codes are HMAC-SHA256'd (never plaintext) and single-use.

The architecture needs a single source of truth covering: OTP request/verify, user/session creation, onboarding integration, cookie strategy, and error handling.

## Decision

Adopt an email-only OTP authentication architecture with the following characteristics:

1. **AuthN method**: Email OTP only (6-digit code, 10-minute validity, 5 max attempts, single use). No passwords.
2. **Challenge lifecycle**: `login_challenges` row + `email_outbox` row created in one DB transaction on request; HMAC-SHA256(`server_secret`, `challenge_id` || `code`) stored as `code_mac`.
3. **User resolution**: On verify, look up user by `normalized_email`. If not found, auto-create user + organization + placeholder school and mark `justCreated = true` (needs onboarding). If found, `justCreated = false`.
4. **Session types**: `temporary` (30 min TTL) for new users requiring onboarding; `full` (30-day TTL) for existing users or post-onboarding.
5. **School resolution**: Auto-select `active_school_id` if the user has exactly one active school; NULL if multiple (client selects via separate endpoint).
6. **Cookie strategy**: `__Host-session` (prod, Secure, HTTPS) or `session` (dev). HttpOnly, SameSite=Lax, Path=/. `needs_onboarding` flag cookie set for temporary sessions.
7. **Onboarding**: POST `/api/v1/auth/onboard` updates org name, user name, creates school, links user as owner (`user_schools` role=owner), upgrades temporary sessions to full, clears flag cookie.
8. **No account enumeration**: OTP request returns identical generic response regardless of whether the email exists.

## Alternatives Considered

| Alternative | Pros | Cons | Why rejected |
|-------------|------|------|--------------|
| Password + OTP (multi-factor) | Familiar, fallback channel | Increases complexity, password storage risk | Project requires passwordless OTP-only |
| Magic links (email URL) | No code entry required | URL-based links can be intercepted/forwarded, poorer mobile UX | OTP codes are simpler and more controllable |
| JWT sessions | Stateless, no DB lookup per request | Token revocation is harder, larger cookie size | Opaque tokens + DB-backed sessions allow instant revocation |
| Single session type (no temporary) | Simpler session model | New users could access dashboard before onboarding | Temporary sessions enforce onboarding completion |

## Consequences

### Positive

- Passwordless auth reduces credential-stuffing and password-reset support burden.
- Temporary sessions guarantee onboarding completion before dashboard access.
- Opaque, hashed session tokens enable instant revocation and limit token theft impact.
- Generic OTP responses prevent email enumeration.
- HMAC'd OTP codes mean plaintext codes never touch the database.

### Negative

- Email delivery latency affects login UX (user waits for code).
- If email provider (Brevo) is down, users cannot authenticate at all.
- Session lookups hit PostgreSQL on every authenticated request (mitigated by short TTLs and potential Redis caching in future).
- Onboarding adds friction for new users before they reach the product.

### Neutral / Risks

- OTP delivery depends on email reliability; consider SMS fallback as a future option.
- Rate limiting must be tuned to balance brute-force protection vs. legitimate resend behavior (see ADR-0005).
- Turnstile adds a frontend dependency on Cloudflare; local dev disables it via `TURNSTILE_ENABLED=false`.

## Links

- Spec: `docs/specs/001-auth-otp-request.md`
- Flow: `docs/auth-flow.md`
- Session ADR: `docs/adr/0003-session-management.md`
- Email outbox ADR: `docs/adr/0004-email-outbox-delivery.md`
- Rate limiting ADR: `docs/adr/0005-rate-limiting-strategy.md`
- Glossary: `docs/specs/glossary.md`
