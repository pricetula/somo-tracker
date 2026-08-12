# Backend Auth Module Audit — `internal/auth`

Date: audit + fix pass
Scope: `backend/internal/auth`, plus the auth-adjacent middleware (`backend/internal/middleware/sessionresolver.go`) and app wiring (`backend/cmd/api/main.go`).

Focus areas requested: (1) auth error gaps, (2) existing accounts on Stytch but missing from our Postgres DB.

---

## 🔴 A. "Existing account on Stytch, not on DB" — flows

### A1. Tenant exists in DB but user row is missing → 500 lockout
`service.go` `handleExistingUser` — the org loop silently continues when `GetUserByEmailAndTenant` returns `ErrNotFound`, even though the org **does** have a tenant. When no org matches both conditions, control falls to `reconstructFromStytch`, which blindly picks `discoveredOrgs[0]` and calls `CreateTenantUserSession` → `INSERT INTO tenants` → **unique violation** on `tenants.styctch_org_id` / `tenants.name` → 500. User permanently locked out.

**Fix (implemented):** when an org has a tenant but the user is missing, create the user + session in the **existing tenant** (`CreateUserSession`), WARN-log that the user has no membership (needs admin enrollment), and do NOT fall through to tenant reconstruction.

### A2. Multi-org Stytch members: only `discoveredOrgs[0]` is rebuilt
`reconstructFromStytch` rebuilds a single org. A member of 2 orgs (2 schools) with a wiped DB loses the second school forever.

**Fix (implemented):** iterate **all** discovered orgs; reconstruct each one that has no tenant; return the session for the first successful reconstruction; failures are logged and skipped.

### A3. `reconstructFromStytch` picks a non-MFA-authenticated org and bails
Picks `orgs[0]` even when a later org is `MemberAuthenticated`, causing a needless `ErrMFARequired`.

**Fix (implemented):** prefer the first `MemberAuthenticated` org; if none is authenticated, cache the IST and return a `SessionRef` so the flow can resume after MFA (see B8).

### A4. `Register` with existing tenant + email already a Stytch member → 500 `duplicate_member_email`
`CreateMember` wraps all Stytch errors as `ErrInternal`. The recovery path (`IdentityProvider.GetMemberByEmail`) exists in the interface but is **dead code** — never called in production.

**Fix (implemented):** `CreateMember` detects `duplicate_member_email` and recovers the existing member ID via `GetMemberByEmail`.

### A5. `Register` consumes the IST first; any failure after that kills retry
`readAndDeleteIST` deletes the IST before org/member/exchange/Postgres work. Retry on the same `session_ref` returns `expired_token` — misleading.

**Fix (implemented):** distinct sentinel `ErrSessionRefExpired` (code `session_ref_expired`, 401) so the frontend can distinguish "link already used" from "link expired". Full retry recovery still requires a fresh magic link (the Stytch org now exists, so re-login reconstructs — covered by A1/A2/A6 fixes).

### A6. `reconstructFromStytch` has no tenant-exists pre-check → race collision
Two concurrent magic-link clicks reconstruct the same org → unique violation.

**Fix (implemented):** per-org `TenantExists` pre-check before inserting.

### A7. `Register` for an existing user in an existing tenant → duplicate user insert → 500
`CreateUserSession` always inserts a user; `idx_users_tenant_email` (tenant_id, LOWER(email)) rejects the second insert. This is the register-side variant of "existing on Stytch, not on DB".

**Fix (implemented):** in `Register`, when the tenant exists, look up the user first; if found, reuse it (pass `sessionParams.UserID`) and skip school creation (the user already has membership context); return their real role from `GetUserRoleInTenant`.

---

## 🔴 B. Auth error gaps

### B1. `GetMe` treats a Redis cache-miss as session-expired → mass logout on Redis restart/eviction
`GetMe` returns `ErrExpiredToken` when `session:<raw>` is absent **without checking Postgres**. `GetSession` has the correct DB-fallback pattern; `GetMe` doesn't. Frontend hard-redirects to /logout on 401.

**Fix (implemented):** `GetMe` always cross-checks `repo.GetMeInfo`; Redis is no longer a gate. Stale cache keys are removed when the session is gone from Postgres.

### B2. Logout doesn't invalidate the middleware session cache → logged-out sessions valid up to 15 min
Two incompatible caches coexist:
- Service: `session:<raw token>` → Stytch token, 30-day TTL (`service.go`)
- Middleware `resolveSession`: `session:<sha256(token)>` → JSON `SessionInfo`, 15-min TTL (`middleware/sessionresolver.go`)

`Logout` deletes only the raw key + DB row; the middleware cache key survives.

**Fix (implemented):** `Logout` and the stale-cleanup paths in `GetMe`/`GetSession` delete **both** key formats. Canonical key derivation lives in `middleware.SessionCacheKey` (single source of truth).

**Verification note:** the B2 refactor initially also passed the `session:`-prefixed cache key into `loadSessionFromDB`, where the query matches `sessions.token_hash` — every DB hit missed, so all sessions resolved to 401/500. Fixed: `resolveSession` uses the bare SHA-256 digest (`sessionTokenHash`) for the DB lookup and the prefixed key only for Redis. Covered by `middleware/sessionresolver_test.go` integration subtests.

### B3. Middleware grants `TEACHER` to users with zero memberships
`loadSessionFromDB` `COALESCE(role, 'TEACHER')` — a user with no active memberships is silently authenticated as TEACHER.

**Fix (implemented):** no default role. Session valid + no membership → `ErrForbidden` (403). Session row missing → translated to `ErrUnauthorized` (401) at the resolver boundary (was 404/500).

### B4. Invite path: expired magic link → 500, not 401
`AuthenticateMagicLink` doesn't map `magic_link_token_expired`.

**Fix (implemented):** maps expired tokens → `ErrExpiredToken` (401).

### B5. Invite path: `MemberAuthenticated=true` with empty `SessionToken` is not validated
**Fix (implemented):** `AuthenticateMagicLink` rejects authenticated-but-empty session tokens.

### B6. `FullName` is never validated
`RegistrationPayload.Validate()` checks SchoolName + SessionRef only.

**Fix (implemented):** FullName required, ≤ 255 chars, printable UTF-8.

### B7. Invite acceptance email lookup is case-sensitive
`GetInvitationByEmail` `email = $1`; Stytch normalizes emails to lowercase but the importer stores raw spreadsheet case.

**Fix (implemented):** `LOWER(email) = LOWER($1)`.

### B8. MFA for existing users is a dead end — IST discarded on `ErrMFARequired`
`handleExistingUser`/`reconstructFromStytch`/`Register` return 401 and drop the IST.

**Fix (implemented, backend):** on MFA-required, cache the IST + email in Redis under a fresh `session_ref` and return a `VerifyResult{SessionRef}` — the flow becomes resumable. Combined with A4 + A7, re-submitting registration with the same school name completes the session after MFA. Frontend MFA completion UI remains a follow-up.

---

## 🟠 C. Security / hardening

### C1. Session token stored in plaintext
`sessions.token` populated on every insert alongside `token_hash`; a DB leak = session hijack.

**Fix (implemented):** new session inserts no longer write the raw token (column stays NULL; only the hash is stored). `idx_sessions_token` becomes vestigial — safe to drop in a future migration.

### C2. `somo_school_id` cookie is unsigned and trusted into `Locals`
Handlers (`attendance`, `behavior`, `parents`) default school context from the cookie. Frontend does not read the cookie, but handlers must scope school IDs by tenant/membership.

**Fix (implemented, centralized):** the session resolver no longer trusts the cookie. `loadSessionFromDB` now returns the user's **authoritative school context** in one query — the DB active school (`member_active_school`, falling back to the first membership) plus the full set of school IDs the user has active memberships in, scoped to the session's tenant. The resolver then: honors the cookie only when it names one of those membership schools, otherwise falls back to the DB active school, and populates `active_school_id`/`school_id` locals exclusively from the result. A forged cookie can never scope a handler to an unowned school, and every consumer is covered without per-handler changes. Covered by `TestSessionResolver/C2:*` integration subtests.

### C3. Rate limiter (10/min/IP) on public group wraps Stytch redirect callbacks
Schools sit behind NAT; an entire school shares one IP and each magic-link click counts against 10/min → self-DoS login.

**Fix (implemented):** `/callback` and `/invite/callback` moved off the per-IP limiter (the one-time magic-link token is the auth; the global coarse limiter still applies when wired).

**Verification note:** group middleware (`public.Use(h.ipLimiter)`) prefix-matches **all** routes under `/api/auth` regardless of subgroup — moving the callback routes to the parent group alone did not un-throttle them (caught by `TestHandler_Callback_NotBlockedByIPLimiter`). The limiter is therefore attached **per-route** (`auth.Post("/discover", h.ipLimiter, h.Discover)`), which scopes it to exactly the three public POST endpoints.

### C4. `Discover` accepts any string as email — no format check
**Fix (implemented):** basic email format + trim validation.

### C5. Device fingerprint is collected but never enforced
Column stored, never compared. Cookie theft works from any device.

**Fix (implemented, production-only):** the device fingerprint is enforced in `APP_ENV=production` only — a resumed session whose presented fingerprint (IP + User-Agent + Accept-Language, computed by `NewDeviceFingerprinter`) differs from the one recorded at session creation is rejected with 401 (`device_fingerprint_mismatch`). Outside production the comparison is skipped entirely, so a single dev machine can act as multiple users. Legacy sessions with an empty stored fingerprint are allowed through (no forced mass logout on rollout). Covered by `TestSessionResolver/C5:*` integration subtests.

⚠️ **Formula caveat:** the existing fingerprint includes the client IP, so in production a user whose network egress changes (DHCP, mobile, Wi-Fi roaming) will be logged out until they re-authenticate. If device-only binding is preferred, drop `c.IP()` from `NewDeviceFingerprinter` — product decision.

---

## 🟡 D. App wiring (found during fix pass)

### D1. Global middleware + error handler not wired in `cmd/api/main.go`
`middleware.Register` (session resolver, CSRF, rate limiters, fingerprint) is never mounted, and `fiber.New()` has no custom error handler. Consequences: `middleware.RequireAuth` always 401s (`/me`, `/switch-school` broken in the running app), CSRF/rate limiting inactive, and middleware errors fall to Fiber's default 500.

**Fix (implemented):** wire `middleware.Register(app, pools, cfg)` and a global error handler that delegates to `middleware.HTTPError` (preserving Fiber's 404/405 defaults). The frontend already sends `X-CSRF-Token`.

### D2. Test harness for existing-user path is stale
`service_test.go` `verifyExistingUserViaMocks` replicates the OLD fallback-to-registration logic; the production code now uses `reconstructFromStytch`. Two tests asserted behavior the real code never had.

**Fix (implemented):** added direct tests against the real `handleExistingUser` / `reconstructFromStytch` / `Register` methods; repurposed the stale assertions.

---

## Status tracker

| ID | Area | Status |
|----|------|--------|
| A1 | Tenant exists, user missing → 500 lockout | ✅ fixed |
| A2 | Multi-org reconstruction drops orgs | ✅ fixed |
| A3 | Reconstruct picks non-authenticated org | ✅ fixed |
| A4 | Register duplicate Stytch member → 500 | ✅ fixed |
| A5 | IST consumed too early / misleading retry error | ✅ fixed |
| A6 | Reconstruct race collision | ✅ fixed |
| A7 | Register existing user → duplicate insert 500 | ✅ fixed |
| B1 | GetMe cache-miss = logout | ✅ fixed |
| B2 | Logout leaves middleware cache alive | ✅ fixed |
| B3 | Zero-membership → TEACHER | ✅ fixed |
| B4 | Expired invite link → 500 | ✅ fixed |
| B5 | Empty session token not validated | ✅ fixed |
| B6 | FullName unvalidated | ✅ fixed |
| B7 | Invite email case-sensitive | ✅ fixed |
| B8 | MFA dead end (backend plumbing) | ✅ fixed (frontend MFA UI: follow-up) |
| C1 | Plaintext session token | ✅ fixed |
| C2 | Unsigned school cookie trusted | ✅ fixed (resolver-scoped to memberships) |
| C3 | IP limiter on Stytch callbacks | ✅ fixed |
| C4 | Discover email unvalidated | ✅ fixed |
| C5 | Device fingerprint not enforced | ✅ fixed (production-only; formula caveat above) |
| D1 | Middleware/error-handler wiring | ✅ fixed |
| D2 | Stale test harness | ✅ fixed |
