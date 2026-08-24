# Auth Flow Security Audit — Findings

**Date**: 2026-08-24  
**Auditor**: AI Agent  
**Scope**: `backend/internal/auth/` + `backend/internal/middleware/sessionresolver.go` + `backend/internal/auth/handler.go`

---

## Executive Summary

The authentication flow has **critical security vulnerabilities** that allow session token replay attacks, inconsistent API responses, and missing token blacklisting. The most severe issue (ISSUE-001) enables attackers to maintain access long after users log out by reusing stale session tokens that remain valid in Redis for up to 30 days.

---

## Critical Findings

### ISSUE-001: Session Token Replay Vulnerability (CRITICAL)

**Location**: `backend/internal/auth/service.go` → `loginExistingUser()`, `createUserInExistingTenant()`, `reconstructOrg()`

**Description**: When a user logs in (via magic link verification or invite acceptance), a new session token is created and cached in Redis. However, **the old session token's Redis entry is never deleted**. The old token remains valid in Redis until it naturally expires (30 days, `sessionTTL`).

**Impact**:
- Attackers who obtain a session token (via XSS, log leakage, network capture) can replay it indefinitely
- Users believe they've "logged out" but old tokens remain functional
- Violates principle of least privilege and session invalidation on re-authentication

**Code Evidence** (`service.go:340-360`):
```go
// loginExistingUser creates new session but never deletes old one
tokenBytes := make([]byte, 32)
rand.Read(tokenBytes)
sessionToken := hex.EncodeToString(tokenBytes)

sessionParams := CreateSessionParams{ Token: sessionToken, ... }
s.repo.CreateSessionOnly(ctx, sessionParams)

// Only adds NEW token to Redis — old token still exists!
s.rdb.Set(ctx, s.sessionKey(sessionToken), stytchSessionToken, sessionTTL)
```

**Same pattern in**:
- `createUserInExistingTenant()` (line 530)
- `reconstructOrg()` (line 1080)
- `Register()` (line 730) — new user registration creates session without cleaning old

**Reproduction**:
1. User A logs in → gets session token `T1` (cached in Redis)
2. User A logs in again (new device/magic link) → gets session token `T2`
3. Redis now contains BOTH `T1` and `T2` (both valid for 30 days)
4. Attacker with `T1` can access resources indefinitely

**Remediation**:
- Add `DeleteSessionsByUserID(ctx, userID, tenantID)` to repository interface
- Call it at start of `loginExistingUser`, `createUserInExistingTenant`, `reconstructOrg`
- Delete all Redis keys for old sessions via `SCAN` + `DEL` on pattern `session:*` + hash-based keys

---

### ISSUE-002: Missing Immediate Token Blacklisting (HIGH)

**Location**: `backend/internal/auth/service.go` → `Logout()`, `GetSession()`

**Description**: No mechanism exists to immediately invalidate a compromised session token. The only invalidation paths are:
1. Explicit `Logout()` call (deletes from Redis + Postgres)
2. Natural expiration (30 days)
3. `GetSession()` cache miss → Postgres lookup → finds deleted row → purges cache

**Impact**:
- Compromised tokens cannot be revoked without waiting 30 days
- No incident response capability for token theft
- `GetSession()` has 15-minute cache TTL — stale sessions survive up to 15 min after Postgres deletion

**Code Evidence** (`service.go:850-880`):
```go
func (s *Service) Logout(ctx context.Context, token string) error {
    s.repo.DeleteSession(ctx, token)  // Postgres only
    s.purgeSessionCacheKeys(ctx, token)  // Redis only for THIS token
    // No blacklist — token could be replayed if Redis delete fails
}
```

**Remediation**:
- Add Redis Set `revoked_sessions:{token_hash}` with TTL matching sessionTTL
- Check blacklist in `GetSession()` and `resolveSession()` before allowing access
- On logout/compromise: `SADD revoked_sessions:{hash} 1 EX sessionTTL`

---

### ISSUE-003: Inconsistent API Response Format (HIGH)

**Location**: `backend/internal/auth/service.go` → `Verify()` return types + `frontend/src/lib/api/generated.ts`

**Description**: The `Verify()` method returns fundamentally different response shapes based on user state:
- **Existing users**: `{ "session_token": "...", "role": "...", "email": "..." }`
- **New users**: `{ "session_ref": "...", "email": "..." }`

The frontend type definitions (`generated.ts`) confirm this inconsistency:
```typescript
export interface VerifyResponse {
    session_ref: string;  // Only for new users
}

export interface ExistingUserVerifyResponse {
    session_token: string;  // Only for existing users
    role: string;
    email: string;
}
```

**Impact**:
- Client code must handle two different response schemas
- Increases likelihood of bugs (accessing `session_token` on new-user response → undefined)
- Frontend registration flow depends on `session_ref` but login flow expects `session_token`
- Non-browser API clients calling POST `/api/auth/verify` directly will receive inconsistent shapes

**Code Evidence** (`service.go:140-180`):
```go
// Existing user path
return &VerifyResult{SessionToken: sessionToken, Role: session.Role, Email: email}

// New user path
return &VerifyResult{SessionRef: sessionRef, Email: email}
```

**Remediation**:
- Standardize on single response format: always return `session_token` (or always `session_ref`)
- For new users, create a provisional session immediately and return its token
- Or always return `session_ref` and have client call a unified "complete" endpoint

---

### ISSUE-004: Device Fingerprint Enforcement Gap (MEDIUM)

**Location**: `backend/internal/middleware/sessionresolver.go` → `enforceDeviceFingerprint()`

**Description**: Device fingerprint validation is **only enforced in production** when `cfg.EnforceDeviceFingerprint == true`. In development/staging or if flag is disabled, sessions can be replayed from any device.

**Impact**:
- Sessions stolen in non-prod environments are fully replayable
- Config flag creates accidental bypass in production
- Legacy v1 fingerprints (IP-inclusive) are accepted without re-verification

**Code Evidence** (`sessionresolver.go:130-160`):
```go
func enforceDeviceFingerprint(c *fiber.Ctx, cfg config.Config, sess *SessionInfo) error {
    if cfg.AppEnv != "production" {
        return nil  // BYPASSED in dev/staging
    }
    if !cfg.EnforceDeviceFingerprint {  // FLAG CAN DISABLE
        logger.Warn("device fingerprint mismatch (C5) — allowed because enforcement is disabled")
        return nil
    }
    return ErrDeviceFingerprintMismatch
}
```

**Remediation**:
- Always enforce fingerprint check in production (remove config flag or make it read-only true)
- Add fingerprint validation in non-prod with warn-only mode (log but don't block)
- Migrate all v1 sessions to v2 format on next login

---

### ISSUE-005: CSRF Token Not Bound to Session (MEDIUM)

**Location**: `backend/internal/auth/handler.go` → `generateCSRFToken()`, `setCSRFTokenCookie()`

**Description**: CSRF tokens are generated as random values and stored in a cookie, but **not cryptographically bound to the session token**. A CSRF token issued for session A can be used with session B.

**Impact**:
- CSRF protection can be bypassed by stealing any valid CSRF token
- Does not prevent cross-session request forgery
- Double-submit pattern requires token == cookie value, but no session binding

**Code Evidence** (`handler.go:340-360`):
```go
func generateCSRFToken() (string, error) {
    b := make([]byte, 32)
    rand.Read(b)
    return base64.RawURLEncoding.EncodeToString(b), nil  // No session binding
}

func (h *Handler) setCSRFTokenCookie(c *fiber.Ctx, token string) {
    c.Cookie(&fiber.Cookie{ Name: "csrf_token", Value: token, ... })
}
```

**Remediation**:
- Sign CSRF token with session token: `HMAC(session_token, "csrf")`
- Verify on mutating requests: recompute HMAC and compare
- Or store CSRF token hash in Redis keyed by session token

---

### ISSUE-006: Magic Link Token Reuse Not Explicitly Prevented (MEDIUM)

**Location**: `backend/internal/auth/service.go` → `Verify()` → `idp.AuthenticateDiscoveryToken()`

**Description**: The Stytch discovery token (IST) is validated via `AuthenticateDiscoveryToken()`, but there's no explicit check in our code that the token hasn't been used before. Relies entirely on Stytch's one-time-use guarantee.

**Impact**:
- If Stytch's one-time guarantee fails (bug, clock skew, replay), our system accepts reused tokens
- No defense-in-depth — single point of failure at identity provider

**Code Evidence** (`service.go:115-125`):
```go
ist, email, discoveredOrgs, err := s.idp.AuthenticateDiscoveryToken(ctx, token)
// No local tracking of used ISTs — trusts Stytch entirely
```

**Remediation**:
- Store hash of consumed IST in Redis with TTL > magic link expiry
- Check before calling `AuthenticateDiscoveryToken()`
- Return `ErrSessionRefExpired` if already seen

---

## Medium Findings

### ISSUE-007: Session Token Stored in Two Places (MEDIUM)

**Location**: `service.go` (Redis + Postgres), `repository.go` (Postgres), `sessionresolver.go` (Redis)

**Description**: Session tokens exist in both Redis (cache) and Postgres (authoritative). If Postgres is compromised, both raw token (via hash) and Stytch session token are exposed.

**Impact**:
- Larger attack surface — compromise of either store leaks session data
- Inconsistency risk if writes to one succeed but other fails
- `fn_resolve_session()` is SECURITY DEFINER — elevated privilege surface

**Remediation**:
- Consider storing only in Redis (with persistence) and removing from Postgres
- Or encrypt session tokens at rest in Postgres
- Audit `fn_resolve_session` for least-privilege

---

### ISSUE-008: No Session Concurrency Limit (MEDIUM)

**Location**: No code enforces max concurrent sessions per user

**Description**: Users can have unlimited simultaneous sessions. No mechanism to:
- Limit active sessions (e.g., max 5)
- Invalidate oldest on new login
- Notify user of new session creation

**Impact**:
- Account sharing undetected
- Stolen sessions persist alongside legitimate ones
- No visibility into active sessions for users

**Remediation**:
- Add `max_concurrent_sessions` config (default: 5)
- On new session creation, delete oldest if over limit
- Add `/api/auth/sessions` endpoint for user to view/revoke

---

## Low Findings

### ISSUE-009: Cookie Domain Configuration Risk (LOW)

**Location**: `handler.go` → `setSessionCookies()` uses `cfg.CookieDomain`

**Description**: If `CookieDomain` is misconfigured (e.g., set to parent domain), cookies may be sent to subdomains not controlled by the application.

**Impact**: Session cookie leakage to unrelated subdomains

**Remediation**: Validate `CookieDomain` at startup; reject if not exact match for API domain

---

### ISSUE-010: Session TTL Hardcoded to 30 Days (LOW)

**Location**: `service.go` line 12: `sessionTTL = 30 * 24 * time.Hour`

**Description**: 30-day session lifetime is excessive for a SaaS application handling educational data.

**Impact**: Extended exposure window for stolen tokens

**Remediation**: Reduce to 7-14 days; add refresh token pattern for long-lived access

---

### ISSUE-011: Frontend verifyToken Only Handles New User Response (MEDIUM)

**Location**: `frontend/src/lib/api/auth.ts` → `verifyToken()` + `frontend/src/lib/api/generated.ts`

**Description**: The frontend `verifyToken()` function is typed to return `VerifyResponse` which only contains `session_ref`. However, the backend's POST `/api/auth/verify` endpoint returns different shapes for existing vs new users. The frontend callback flow (GET `/api/auth/callback`) handles both via redirects, but direct API calls will fail type-checking or runtime parsing for existing users.

**Impact**:
- TypeScript type safety gap — existing user response doesn't match `VerifyResponse`
- Runtime errors if non-browser client calls verify endpoint directly
- Frontend code assumes only new-user flow (redirects to `/register?session_ref=...`)

**Code Evidence** (`generated.ts:45-55`):
```typescript
export interface VerifyResponse {
    session_ref: string;  // Missing session_token, role, email for existing users
}

export interface ExistingUserVerifyResponse {
    session_token: string;
    role: string;
    email: string;
}
```

**Remediation**:
- Update `VerifyResponse` to be a discriminated union or include all possible fields
- Or fix backend to return consistent shape (see ISSUE-003)

---

## Repository Interface Gaps

The following methods are **missing** from the `Repository` interface and needed for fixes:

```go
// Auth/Repository interface additions needed:
DeleteSessionsByUserID(ctx context.Context, userID, tenantID string) error
GetSessionsByUserID(ctx context.Context, userID string) ([]*UserSession, error)
```

---

## Remediation Priority Order

| Priority | Issue | Effort | Risk Reduction |
|----------|-------|--------|----------------|
| P0 | ISSUE-001: Session replay | Medium | Eliminates primary attack vector |
| P0 | ISSUE-002: Token blacklist | Low | Enables incident response |
| P1 | ISSUE-003: API consistency | Medium | Reduces client bugs |
| P1 | ISSUE-004: Fingerprint enforcement | Low | Closes bypass vector |
| P2 | ISSUE-005: CSRF binding | Medium | Strengthens CSRF |
| P2 | ISSUE-006: IST reuse tracking | Low | Defense in depth |
| P3 | ISSUE-007: Single token store | High | Architecture change |
| P3 | ISSUE-008: Concurrency limit | Medium | UX + security |
| P4 | ISSUE-009: Cookie domain | Low | Config hardening |
| P4 | ISSUE-010: TTL reduction | Low | Reduce exposure window |
| P3 | ISSUE-011: Frontend verifyToken types | Medium | Type safety + API consistency |

---

## Testing Recommendations

Add integration tests for:
1. **Token replay**: Login twice, verify first token rejected
2. **Blacklist**: Logout, verify token rejected immediately
3. **Fingerprint**: Login from device A, replay from device B → 401
4. **IST reuse**: Call `Verify()` twice with same token → second fails
5. **Concurrency**: Create 6 sessions, verify oldest deleted

---

## Appendix: Key Files Reviewed

| File | Purpose |
|------|---------|
| `backend/internal/auth/service.go` | Core auth business logic (login, register, invite, session mgmt) |
| `backend/internal/auth/handler.go` | HTTP endpoints (discover, verify, register, callback, logout, me) |
| `backend/internal/auth/repository.go` | Postgres data access (sessions, users, tenants, invitations) |
| `backend/internal/middleware/sessionresolver.go` | Per-request session resolution + device fingerprint enforcement |
| `backend/internal/auth/domain.go` | Domain types, errors, payloads, results |
| `backend/internal/auth/stytch.go` | Stytch identity provider interface + implementation |
| `frontend/src/lib/api/auth.ts` | Frontend auth API client functions |
| `frontend/src/lib/api/generated.ts` | Frontend TypeScript types for auth API |
| `frontend/src/hooks/use-auth.ts` | Frontend auth React Query hooks |
| `frontend/src/features/auth/components/register-form.tsx` | Frontend registration form component |

---

## Version History

| Version | Date | Author | Changes |
|---------|------|--------|---------|
| 1.0 | 2026-08-24 | AI Agent | Initial audit findings |