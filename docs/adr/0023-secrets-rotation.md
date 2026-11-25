# ADR-0023: Secrets Rotation

## Metadata

| Field | Value |
|-------|-------|
| id | ADR-0023 |
| title | Secrets Rotation |
| status | proposed |
| date | 2026-11-25 |
| author | backend, platform |
| deciders | backend, platform, security |
| relates_to | ADR-0001, ADR-0004, ADR-0022 |

## Status

proposed

## Context

Somotracker relies on Doppler-injected configuration at runtime (`EMAIL_HMAC_KEY`, `BREVO_API_KEY`, `SESSION_COOKIE_NAME`, etc.). Per AGENTS.md, required values are validated at startup in `config.validate()` with no fallback defaults. However, there is no documented process for rotating secrets when they are compromised, when employees leave, or on a scheduled basis.

Without a rotation process, a compromised secret remains valid indefinitely, and there is no path to revoke access.

## Decision

Adopt a **Doppler-driven, zero-downtime secrets rotation model**:

1. **Secret inventory**: All secrets managed in Doppler. No secrets in git, `.env` files, or developer machines. Required secrets: `DATABASE_URL`, `BREVO_API_KEY`, `EMAIL_HMAC_KEY`, `SESSION_COOKIE_NAME`, `TURNSTILE_SECRET_KEY` (when enabled), backup object storage credentials, encryption keys.

2. **Rotation trigger**: Secrets are rotated on:
   - Compromise or suspected compromise (immediate)
   - Employee offboarding (within 24 hours)
   - Scheduled rotation (every 90 days for long-lived keys like `EMAIL_HMAC_KEY`)

3. **Zero-downtime rotation**: The application reads secrets from Doppler at startup via `firstNonEmpty` with no default. Doppler config changes are picked up on next application restart. For zero-downtime, a rolling deployment replaces old pods with new pods that read the new secret.

4. **`EMAIL_HMAC_KEY` rotation**: This key is used to HMAC OTP codes (`code_mac`). Rotating it invalidates all existing unverified OTP codes (they will fail HMAC verification). This is acceptable — OTP codes are short-lived (10 minutes). Existing sessions are unaffected (session tokens are SHA-256 hashes, not HMAC'd with this key).

5. **`BREVO_API_KEY` rotation**: Brevo API keys can be rotated in the Brevo dashboard, then updated in Doppler. No downtime — the application reads the new key on next OTP request.

6. **Backup object storage credentials**: Rotated per cloud provider's IAM process. Updated in Doppler; backup jobs pick up new credentials on next run.

7. **Revocation**: On employee offboarding, revoke all Doppler access and rotate any secrets the employee had access to.

## Alternatives Considered

| Alternative | Pros | Cons | Why rejected |
|-------------|------|------|--------------|
| No rotation (secrets live forever) | Simplest | Compromised secrets remain valid indefinitely | Rejected — security risk |
| Application reads secrets on every request | Immediate rotation without restart | Performance overhead, Doppler rate limits, complexity | Rejected — startup read + rolling deploy is simpler |
| Doppler-driven + rolling deploy (chosen) | Zero downtime, centralized secret management | Requires Doppler integration, rolling deploy infrastructure | Accepted — matches existing Doppler pattern |

## Consequences

### Positive

- Centralized secret management via Doppler.
- Zero-downtime rotation via rolling deployments.
- Clear rotation triggers and cadence.
- Revocation path for employee offboarding.

### Negative

- `EMAIL_HMAC_KEY` rotation invalidates all pending OTP codes (users must request new codes).
- Rolling deploy required for zero-downtime (infrastructure dependency).

### Neutral / Risks

- Doppler outage at startup prevents application startup (existing behavior, mitigated by Doppler's high availability).
- Rotation cadence (90 days) must be enforced — no automated enforcement yet.

## Implementation Notes

1. **Doppler config**: All secrets in Doppler project `somotracker`, environments `development`, `staging`, `production`.
2. **Startup validation**: `config.validate()` checks required secrets are present (existing behavior).
3. **Rolling deploy**: Kubernetes deployment with `maxUnavailable: 0`, `maxSurge: 1` for zero-downtime secret rotation.
4. **Audit log**: Log secret rotation events (who rotated what, when) — without logging the secret values themselves.

## Links

- Authentication: `docs/adr/0001-authentication-architecture.md`
- Email outbox: `docs/adr/0004-email-outbox-delivery.md`
- Observability: `docs/adr/0022-observability.md`
