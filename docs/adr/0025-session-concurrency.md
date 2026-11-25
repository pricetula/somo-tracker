# ADR-0025: Session Concurrency

## Metadata

| Field | Value |
|-------|-------|
| id | ADR-0025 |
| title | Session Concurrency |
| status | proposed |
| date | 2026-11-25 |
| author | backend |
| deciders | backend, product |
| relates_to | ADR-0001, ADR-0003 |

## Status

proposed

## Context

Somotracker issues opaque sessions after OTP verification (ADR-0003). Sessions are stored in PostgreSQL with `token_hash`, `user_id`, `organization_id`, `active_school_id`, `session_type`, `expires_at`, and `revoked_at`. There is no policy on whether a user can have multiple active sessions across devices or browsers.

Without a session concurrency policy, it is unclear whether logging in on a new device invalidates existing sessions, whether concurrent sessions are allowed, and how session revocation (logout) affects other sessions.

## Decision

Adopt a **multiple concurrent sessions allowed** model:

1. **Multiple sessions per user**: A user can have multiple active sessions simultaneously (e.g. logged in on a laptop and a phone). Each session is an independent row in the `sessions` table with its own `token_hash`, `expires_at`, and `active_school_id`.

2. **No session invalidation on new login**: Logging in on a new device does NOT invalidate existing sessions. Each OTP verification creates a new session row.

3. **Per-session revocation**: Logging out revokes only the current session (`revoked_at` set on that session's `token_hash`). Other sessions remain active.

4. **Global revocation (future)**: A "revoke all sessions" action (e.g. password compromise, account recovery) can be added later by setting `revoked_at` on all sessions for a user. Not implemented in MVP.

5. **Session limits**: No limit on the number of concurrent sessions in MVP. If abuse is detected (e.g. one user creating hundreds of sessions), rate limiting (ADR-0005) and session cleanup (ADR-0003) mitigate the risk.

6. **Session type per session**: Each session can be `temporary` (30 min, pre-onboarding) or `full` (30 days). A user upgrading from temporary to full gets the new full session; old temporary sessions expire on their own 30-minute TTL.

## Alternatives Considered

| Alternative | Pros | Cons | Why rejected |
|-------------|------|------|--------------|
| Single session (new login invalidates old) | Simple security model, no session proliferation | Frustrates users who log in on multiple devices | Rejected — teachers often use phone + computer |
| Multiple concurrent sessions (chosen) | User-friendly, matches real-world usage | Session table grows; no single "kill all" in MVP | Accepted — UX priority, cleanup job handles growth |
| Session limit (e.g. max 5) | Prevents session proliferation | Arbitrary limit, may block legitimate use | Rejected — defer until abuse is observed |

## Consequences

### Positive

- Users can be logged in on multiple devices simultaneously.
- Logging out on one device does not disrupt other devices.
- Simple mental model: each login = one session row.

### Negative

- Session table grows with each login (mitigated by TTL expiration and cleanup job).
- No "revoke all sessions" in MVP — a lost device session remains active until TTL expiry.

### Neutral / Risks

- Session cleanup job (ADR-0003) is required to prevent unbounded table growth.
- Stolen session tokens remain valid until expiry — no way to invalidate a specific stolen session without knowing its token hash.

## Implementation Notes

1. **Session creation**: Each OTP verification inserts a new `sessions` row (ADR-0003 Step 10).
2. **Logout**: Sets `revoked_at` + `revocation_reason = 'logout'` on the current session's `token_hash`.
3. **Session validation**: Middleware checks `revoked_at IS NULL AND expires_at > NOW()` (ADR-0003 Step 5).
4. **Cleanup**: Periodic job deletes sessions where `expires_at < NOW()` or `revoked_at IS NOT NULL`.

## Links

- Authentication: `docs/adr/0001-authentication-architecture.md`
- Session management: `docs/adr/0003-session-management.md`
- Rate limiting: `docs/adr/0005-rate-limiting-strategy.md`
