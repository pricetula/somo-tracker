# Authentication, Registration & Invitation Flow

> **Last updated:** 2026-06-23
> **Owner:** Platform team

This document describes how authentication, school registration, and staff invitation work across the Somotracker platform (backend Go/Fiber + frontend Next.js).

---

## Table of Contents

1. [Architecture Overview](#architecture-overview)
2. [Cookie Architecture](#cookie-architecture)
3. [Discovery & Registration Flow](#discovery--registration-flow)
4. [Session Validation](#session-validation)
5. [Logout](#logout)
6. [Invitation Flow](#invitation-flow)
7. [Frontend Proxy & Route Protection](#frontend-proxy--route-protection)
8. [Error Handling](#error-handling)
9. [FAQ & Troubleshooting](#faq--troubleshooting)

---

## Architecture Overview

A user signs in by entering their email on the login page. The system sends a **Stytch B2B magic link** to that email. Clicking the link authenticates the user and produces an **Intermediate Session Token (IST)**, which is cached in Redis for 10 minutes. On the registration page, the user provides their school name and personal details; the backend exchanges the IST for a full session and persists the tenant, user, and membership in Postgres.

```
┌──────────┐       ┌──────────────┐       ┌──────────┐       ┌──────────────┐
│  Browser │       │  Next.js FE  │       │ Go API   │       │   Stytch     │
│          │       │  (proxy.ts)  │       │ (Fiber)  │       │   (B2B)      │
└────┬─────┘       └──────┬───────┘       └────┬──────┘       └──────┬───────┘
     │                    │                     │                     │
     │  GET /login        │                     │                     │
     │───────────────────>│                     │                     │
     │                    │                     │                     │
     │  Enter email       │                     │                     │
     │───────────────────>│  POST /api/auth     │                     │
     │                    │  /discover          │  Send magic link    │
     │                    │────────────────────>│────────────────────>│
     │                    │                     │                     │
     │  ─── Email with magic link ────────────────────────────────────│
     │                    │                     │                     │
     │  Click link        │                     │                     │
     │───────────────────>│  GET /api/auth      │  Authenticate token │
     │                    │  /callback?token=X  │────────────────────>│
     │                    │<────────────────────│<────────────────────│
     │                    │                     │  Return IST + email │
     │  Redirect to       │                     │                     │
     │  /register?        │                     │                     │
     │  session_ref=UUID  │                     │                     │
     │<───────────────────│                     │                     │
     │                    │                     │                     │
     │  Fill form +       │                     │                     │
     │  submit            │  POST /api/auth     │  Create org,        │
     │───────────────────>│  /register          │  create member,     │
     │                    │────────────────────>│  exchange IST,      │
     │                    │                     │  persist in PG,     │
     │                    │                     │  set cookies        │
     │                    │<────────────────────│                     │
     │  204 + Set-Cookie  │                     │                     │
     │<───────────────────│                     │                     │
```

### Key components

| Component | File | Responsibility |
|---|---|---|
| **Backend handler** | `backend/internal/auth/handler.go` | HTTP route handlers for /api/auth/* |
| **Backend service** | `backend/internal/auth/service.go` | Business logic: verify, register, accept invite, logout |
| **Backend repository** | `backend/internal/auth/repository.go` | Postgres persistence (Pgx + sqlc patterns) |
| **Stytch adapter** | `backend/internal/auth/stytch.go` | Stytch B2B API client (the only file importing the Stytch SDK) |
| **Identity provider interface** | `backend/internal/auth/domain.go` | `IdentityProvider` contract for Stytch abstraction |
| **Repository interface** | `backend/internal/auth/domain.go` | Persistence contract |
| **Members service** | `backend/internal/members/service.go` | Invitation creation and bulk invite logic |
| **Members handler** | `backend/internal/members/handler.go` | HTTP routes for /api/v1/members and /api/v1/invitations |
| **Frontend API client** | `frontend/src/lib/api/client.ts` | `fetch` wrapper with credentials, CSRF, and global 401 eviction |
| **Frontend auth utilities** | `frontend/src/lib/auth.ts` | Session/role cookie constants and role-to-route mappings |
| **Frontend server auth** | `frontend/src/lib/auth-server.ts` | HMAC verification for Node.js runtime (server components) |
| **Frontend proxy** | `frontend/src/proxy.ts` | Edge runtime middleware that guards routes using signed role cookies |
| **Frontend auth hooks** | `frontend/src/hooks/use-auth.ts` | React Query mutations for discover, verify, register, logout |
| **Frontend login page** | `frontend/src/features/auth/components/login-page.tsx` | Email input form |
| **Frontend register form** | `frontend/src/features/auth/components/register-form.tsx` | School + name registration form |
| **Error middleware** | `backend/internal/middleware/errors.go` | Canonical HTTPError helper |
| **Schema migration** | `backend/internal/database/migrations/000001_initial_schema.up.sql` | tenants, users, sessions, memberships, invitations tables |

---

## Cookie Architecture

The system uses a **three-cookie** strategy:

### `somo_sid` — HttpOnly session token

| Attribute | Value |
|---|---|
| Name | `somo_sid` |
| Content | 64-character hex string (32 random bytes) |
| HttpOnly | ✅ Yes |
| Secure | ✅ Yes (except development) |
| SameSite | `Lax` |
| Path | `/` |
| Max-Age | 30 days |

- The opaque token is the primary authentication credential.
- It is **never readable by JavaScript** (`HttpOnly`).
- It is stored in both Postgres (`sessions` table) and Redis (for fast validation).
- Redis key pattern: `session:{token}`.

### `somo_role` — Signed role cookie

| Attribute | Value |
|---|---|
| Name | `somo_role` |
| Content | `{role}.{hmac_sha256_signature}` |
| HttpOnly | ❌ No (readable by JS proxy) |
| Secure | ✅ Yes (except development) |

- **Not a security credential** — it is a routing hint for the Next.js Edge proxy.
- Signed with HMAC-SHA256 using `COOKIE_SECRET`.
- Format: `value.hexsignature` — frontend splits on the last `.` and verifies.
- The actual security gate is always the backend CSRF + session validation.

### `csrf_token` — Double-submit CSRF token

| Attribute | Value |
|---|---|
| Name | `csrf_token` |
| Content | 32 random bytes, base64url-encoded |
| HttpOnly | ❌ No |
| Secure | ✅ Yes (except development) |

- The frontend reads this cookie and includes its value as the `X-CSRF-Token` header on all mutating requests (`POST`, `PUT`, `PATCH`, `DELETE`).
- The backend verifies the header matches the cookie value.
- Set on successful registration, magic-link callback, and invite acceptance.

### Cookie lifecycle summary

```
Discovery flow (new user):
  GET /api/auth/callback?token=...  →  Set csrf_token cookie
  POST /api/auth/register           →  Set somo_sid + somo_role + csrf_token

Invite acceptance (invited user):
  GET /api/auth/invite/callback?token=...  →  Set somo_sid + somo_role + csrf_token

Logout:
  DELETE /api/auth/session  →  Clear all three cookies (Max-Age: -1)
```

---

## Discovery & Registration Flow

The registration flow has **three phases**, each corresponding to a frontend page and backend endpoint.

### Phase 1: Discovery — `POST /api/auth/discover`

**Frontend:** `/login` page — email input form.

1. User enters their email and submits.
2. `useDiscover` mutation calls `POST /api/auth/discover` with `{ "email": "..." }`.
3. Backend `Handler.Discover` → `Service.Discover` → `StytchAdapter.SendDiscoveryEmail`.
   - `StytchAdapter` calls `api.MagicLinks.Email.Discovery.Send()` with the email and a `DiscoveryRedirectURL` pointing to the backend's `/api/auth/callback` endpoint.
4. On success, the frontend shows a toast: "Magic link sent! Check your inbox."
5. On failure, the frontend shows a toast with the error message.

### Phase 2: Magic Link Callback — `GET /api/auth/callback?token=...`

**Trigger:** User clicks the magic link in their email, which points to:
```
{backendURL}/api/auth/callback?token={stytch_token}
```

1. `Handler.MagicLinkCallback` extracts the `token` query parameter.
2. `Service.Verify` → `StytchAdapter.AuthenticateDiscoveryToken`
   - Calls `api.MagicLinks.Discovery.Authenticate()` with the token.
   - Stytch returns an **Intermediate Session Token (IST)** and the verified **email address**.
   - If the token is expired, `ErrExpiredToken` is returned → 401.
3. The backend generates a **UUID v4 `session_ref`** and caches the IST + email in **Redis**:
   - Key: `ist:{appEnv}:{sessionRef}`
   - Value: JSON `{ "ist": "...", "email": "..." }`
   - TTL: **10 minutes**
4. A CSRF token cookie is set on the response (non-HttpOnly).
5. The browser is **redirected** to:
   ```
   {frontendURL}/register?session_ref={uuid}
   ```

### Phase 3: Registration — `POST /api/auth/register`

**Frontend:** `/register?session_ref=...` page — school name, first name, last name form.

The form is guarded: if no `session_ref` query param is present, the user is redirected to `/login`.

**Backend steps (in order):**

1. **Validate payload** — school name must be 2–100 printable UTF-8 chars; `session_ref` must be a valid UUID v4.

2. **Read and delete IST from Redis** — uses a **Lua script** (`redis.call("GET", ...) + redis.call("DEL", ...)`) for atomic read-and-delete. If the key is missing (already consumed or expired), returns `ErrExpiredToken` → 401.

3. **Check tenant existence by school name** — queries Postgres `SELECT EXISTS(SELECT 1 FROM tenants WHERE name = $1)`.
   - **Existing tenant:** retrieves the tenant's Stytch org ID (no org creation needed).
   - **New tenant:** creates a Stytch organization via `StytchAdapter.CreateOrganization` (calls `api.Organizations.Create()`).

4. **Create member in Stytch org** — `StytchAdapter.CreateMember` must be called **before** IST exchange, otherwise the exchange fails with `email_jit_provisioning_not_allowed`.

5. **Exchange IST** — `StytchAdapter.ExchangeIntermediateSession` calls `api.Discovery.IntermediateSessions.Exchange()`.
   - Must check `result.MemberAuthenticated` (MFA enforcement). If `false`, returns `ErrMFARequired`.

6. **Check tenant by org ID** — second idempotency check after exchange (in case of race conditions).

7. **Generate opaque session token** — 32 random bytes, hex-encoded → 64-character string.

8. **Persist to Postgres** — two transaction paths:
   - **Fresh tenant:** `CreateTenantUserSession` — creates tenant + user + session in one transaction.
   - **Existing tenant:** `CreateUserSession` — creates only user + session (no tenant insert).
   - Both use the **deferred rollback** pattern with structured logging.

9. **Create school and membership** — `CreateSchool` inserts a `cbc_schools` row, then `CreateMembership` links the user to that school:
   - First user of a fresh tenant → role `SCHOOL_ADMIN`
   - Subsequent users of an existing tenant → role `TEACHER`

10. **Cache session in Redis** — key `session:{token}` → Stytch session token, TTL 30 days.

11. **Set cookies** — `somo_sid` (HttpOnly), `somo_role` (signed), `csrf_token` (non-HttpOnly).

12. Return **204 No Content**.

### State diagram

```
[User visits /login]
        │
        ▼
[Enter email → POST /api/auth/discover]
        │
        ▼
[Email sent with magic link] ← Stytch B2B
        │
        ▼
[User clicks magic link]
        │
        ▼
[GET /api/auth/callback?token=...]
        │
        ├── Token expired → 401 → Login page
        │
        ▼
[IST + email cached in Redis (10 min TTL)]
        │
        ▼
[Redirect to /register?session_ref=UUID]
        │
        ▼
[User fills school name, first name, last name]
        │
        ▼
[POST /api/auth/register]
        │
        ├── IST missing (consumed/expired) → 401 → /login
        ├── School name already exists → reuses existing tenant
        ├── Validation error → 400 with field errors
        ├── MFA not completed → 403 → block
        ├── Success → 204 + Set-Cookie → /
        │
        ▼
[Dashboard: authenticated session active (30 days)]
```

---

## Session Validation

### `GET /api/auth/me`

Called on every authenticated page load (including `/` root route).

1. **Extract cookie:** `c.Cookies("somo_sid")`. If empty → 401.
2. **Redis fast path:** checks `EXISTS` on `session:{token}`. If missing → 401.
3. **Postgres query:** `GetMeInfo` joins `sessions`, `users`, `memberships`, `cbc_schools`.
   - If session not found in PG → deletes stale Redis entry → 401.
4. **Returns:**

```json
{
  "user_id":     "uuid",
  "tenant_id":   "uuid",
  "role":        "SCHOOL_ADMIN",
  "school_id":   "uuid",
  "school_name": "Lincoln High School",
  "first_name":  "Jane",
  "last_name":   "Doe",
  "email":       "jane@school.edu"
}
```

### `GET /api/auth/me` (via Frontend)

The `useMe` hook (`frontend/src/hooks/use-auth.ts`) wraps this endpoint with React Query. It returns `null` when the user is not authenticated (non-throwing), so components can branch on `isLoading` / `data`.

---

## Logout

### `DELETE /api/auth/session`

1. **Extract `somo_sid` cookie** — if empty, no-op (return 204).
2. **Delete from Postgres** — `DELETE FROM sessions WHERE token = $1`.
   - If session not found, still proceed (best-effort cleanup).
3. **Delete from Redis** — `DEL session:{token}`.
4. **Clear cookies** — reset `somo_sid` + `somo_role` + `csrf_token` with `MaxAge: -1`.

**Frontend:** `useLogout` mutation → on success, calls `queryClient.clear()` (wipes all React Query cache) and redirects to `/login`.

---

## Invitation Flow

The invitation flow allows existing school administrators to invite new staff members (TEACHER, NURSE, FINANCE, SCHOOL_ADMIN) to join their school.

### Creating Invitations

#### `POST /api/v1/invitations` (per-invite roles)

**Auth required:** Any authenticated user with a valid session.

```json
{
  "invites": [
    { "email": "newteacher@school.edu", "first_name": "John", "last_name": "Smith", "role": "TEACHER" }
  ]
}
```

Backend steps:
1. Validate each invite: email required, role must be `TEACHER`, `NURSE`, `FINANCE`, or `SCHOOL_ADMIN`.
2. Check for existing membership (user already in school).
3. Check for existing pending invitation (duplicate guard).
4. Create `invitations` row with status `pending`, `expires_at = now + 7 days`.
5. Return response:

```json
{
  "sent": 1,
  "failed": 0,
  "errors": []
}
```

#### `POST /api/v1/members/invite` (shared role)

Similar to above, but `role` is shared across all invites. After persisting the invitation locally, it also **sends a Stytch invite email** via `StytchAdapter.InviteMemberByEmail`, which calls `api.MagicLinks.Email.Invite()` with redirect URL `{backendURL}/api/auth/invite/callback`.

If the Stytch call fails, the invitation is still persisted locally (so it can be retried later).

### Accepting an Invitation

#### `GET /api/auth/invite/callback?token=...`

1. User clicks the magic link in the invite email.
2. `Handler.AcceptInvite` → `Service.AcceptInvite`:
   - **Authenticate token:** `StytchAdapter.AuthenticateInviteToken` → returns IST + email.
   - **Look up invitation:** `Repository.GetInvitationByEmail` — queries `SELECT ... FROM invitations WHERE email = $1 AND status = 'pending' AND expires_at > NOW()`.
     - If no invitation → `ErrExpiredToken` → 401.
   - **Resolve Stytch org ID:** `Repository.GetTenantStytchOrgID` — fetches `stytch_org_id` from the `tenants` table.
   - **Exchange IST:** `StytchAdapter.ExchangeInviteSession` — enforces MFA (`MemberAuthenticated == true`), returns Stytch session token.
   - **Create user/session/membership in single transaction** (`CreateInvitedUserSession`):
     - Creates `users` row
     - Creates `sessions` row
     - Creates `memberships` row
     - Updates `invitations` → `status = 'accepted'`, `accepted_at = NOW()`
   - **Cache session in Redis:** `SET session:{token} → stytchSessionToken, TTL 30 days`.
   - **Set cookies:** `somo_sid` + `somo_role` + `csrf_token`.
   - **Redirect** to `{frontendURL}/dashboard`.

### Invitation lifecycle

```
[School admin creates invitation(s)]
        │
        ▼
[Invitation row: status = pending, expires_at = 7 days]
        │
        ├── [User clicks invite magic link]
        │       │
        │       ▼
        │   [GET /api/auth/invite/callback?token=...]
        │       │
        │       ├── Token expired/invalid → 401
        │       ├── No pending invitation → 401
        │       └── Success → session created, redirect to /dashboard
        │
        ├── [7 days pass]
        │       │
        │       ▼
        │   [Invitation expires naturally (WHERE expires_at > NOW())]
        │
        └── [Admin revokes invitation]
                │
                ▼
            [status = 'revoked']
```

### Invitation database schema

```sql
CREATE TABLE IF NOT EXISTS invitations (
    id                  UUID              PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id           UUID              NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    school_id           UUID              NOT NULL,
    email               VARCHAR(255)      NOT NULL,
    role                user_role         NOT NULL,
    status              invitation_status NOT NULL DEFAULT 'pending',
    invited_by          UUID              REFERENCES users(id) ON DELETE SET NULL,
    token               TEXT              NOT NULL,
    expires_at          TIMESTAMPTZ       NOT NULL,
    accepted_at         TIMESTAMPTZ       NULL,
    first_name          VARCHAR(255)      NULL,
    last_name           VARCHAR(255)      NULL,
    phone               VARCHAR(50)       NULL,
    registration_number VARCHAR(100)      NULL,
    stytch_member_id    VARCHAR(255)      NULL,
    import_job_id       UUID              NULL,
    error_message       TEXT              NULL,
    attempt_count       INT               NOT NULL DEFAULT 0,
    created_at          TIMESTAMPTZ       NOT NULL DEFAULT NOW(),

    CONSTRAINT fk_invitations_tenant_school
        FOREIGN KEY (tenant_id, school_id) REFERENCES cbc_schools(tenant_id, id) ON DELETE CASCADE,
    CONSTRAINT fk_invitations_import_job
        FOREIGN KEY (import_job_id) REFERENCES import_jobs(id) ON DELETE SET NULL
);
```

---

## Frontend Proxy & Route Protection

The Next.js Edge middleware (`frontend/src/proxy.ts`) intercepts all requests and enforces authentication and role-based access.

### Auth state detection

The proxy determines the current auth stage from cookies and query params:

| Stage | `somo_sid` cookie | `somo_role` cookie | `session_ref` query param |
|---|---|---|---|
| **Not authenticated** | ❌ | ❌ | ❌ |
| **IST stage** | ❌ | ❌ | ✅ (on /register) |
| **Authenticated** | ✅ | ✅ | — |

### Route protection matrix

| Route | Guard | Action |
|---|---|---|
| `/` (root dashboard) | Requires both cookies + valid role with dashboard access | Allow or redirect to `/login` |
| `/dashboard/*` | Requires both cookies + valid role with dashboard access | Allow or redirect to `/login` |
| `/settings/*` | Requires both cookies + valid role | Allow or redirect to `/login` |
| `/admin/*` | Requires both cookies + SYSTEM_ADMIN or SCHOOL_ADMIN | Allow, redirect to `/login`, or `/unauthorized` |
| `/schools/*` | Requires both cookies + SYSTEM_ADMIN or SCHOOL_ADMIN | Allow, redirect to `/login`, or `/unauthorized` |
| `/login` | Both cookies present → redirect to `/` | Otherwise allow |
| `/register` | Has `somo_sid` → redirect to `/`. No `session_ref` → redirect to `/login` | Otherwise allow |
| `/logout` | Always allowed | Destroys session |
| `/unauthorized` | Always allowed | Shows forbidden message |

### Role-based route access

Defined in `frontend/src/lib/auth.ts` `ROLE_ROUTES`:

| Role | Allowed route prefixes |
|---|---|
| `SYSTEM_ADMIN` | `/admin`, `/admins`, `/dashboard`, `/settings`, `/schools`, `/docs` |
| `SCHOOL_ADMIN` | `/admin`, `/admins`, `/dashboard`, `/settings`, `/schools`, `/docs` |
| `TEACHER` | `/dashboard`, `/docs` |
| `NURSE` | `/dashboard`, `/docs` |
| `FINANCE` | `/dashboard`, `/docs` |

### Cookie signature verification

The proxy verifies the `somo_role` cookie signature using the Web Crypto API:

```typescript
// proxy.ts (Edge runtime)
const key = await crypto.subtle.importKey(
  "raw", encoder.encode(COOKIE_SECRET),
  { name: "HMAC", hash: "SHA-256" }, false, ["verify"]
);
const isValid = await crypto.subtle.verify("HMAC", key, sigBytes, valueBytes);
```

For server components (Node.js runtime), `frontend/src/lib/auth-server.ts` uses `crypto.timingSafeEqual` for constant-time comparison.

---

## Error Handling

### Canonical response contract

Every non-2xx HTTP response returns:

```json
{
  "code":    "snake_case_error_code",
  "message": "human readable message",
  "errors":  { "field_name": ["Specific field validation message"] }
}
```

### Error code to HTTP status mapping

| Error code | HTTP status | Description |
|---|---|---|
| `not_found` | 404 | Resource not found |
| `already_exists` | 409 | Duplicate resource |
| `invalid_input` | 400 | Validation failure (with optional `errors` map) |
| `unauthorized` | 401 | Missing or invalid session |
| `expired_token` | 401 | Magic link or IST expired |
| `mfa_required` | 403 | MFA not completed |
| `forbidden` | 403 | Insufficient permissions |
| `conflict` | 409 | Concurrent modification |
| `request_canceled` | 499 | Client canceled request |
| `timeout` | 504 | Request timeout |
| `internal_error` | 500 | Unexpected error (logged, generic message) |

### Global 401 eviction

The frontend API client (`client.ts`) has a global interceptor: if **any** API request returns 401 (and `skipGlobal401Handler` is not set), it immediately redirects to `/logout`, which clears all cookies and redirects to `/login`.

### Expired token handling

- **Magic link expired:** Stytch returns `magic_link_token_expired` → mapped to `ErrExpiredToken` → 401.
- **IST expired:** Redis key TTL (10 minutes) expires → `GET` returns nil → mapped to `ErrExpiredToken` → 401.
- **Session expired:** Redis key TTL (30 days) expires → `EXISTS` returns 0 → mapped to `ErrExpiredToken` → 401.

### Frontend error display

| Error scenario | User-facing message |
|---|---|
| Magic link sent successfully | Toast: "Magic link sent! Check your inbox." |
| Magic link expired (callback) | "This magic link has expired. Please request a new one." |
| IST expired (registration) | "This registration session has expired. Please request a new magic link." |
| Registration field error | Inline form validation: "School name must be at least 2 characters" |
| Registration success | Toast: "Account created! Welcome to Somotracker." → redirect to `/` |
| Invite token expired | 401 → Global eviction → `/logout` |
| Logout success | Toast: "Logged out" → redirect to `/login` |
| Any 401 | Auto-redirect to `/logout` |

---

## FAQ & Troubleshooting

### "Why do I keep getting redirected to /login when I try to access the dashboard?"

Possible causes:

1. **Session expired** — sessions last 30 days. Log in again via magic link.
2. **Cookies blocked** — the system requires both `somo_sid` and `somo_role` cookies. Check that third-party cookies are not blocked.
3. **COOKIE_SECRET mismatch** — if `COOKIE_SECRET` was rotated, all existing `somo_role` cookies have invalid signatures. Users must log in again.
4. **Role not in ROLE_ROUTES** — if a new role was added without updating `ROLE_ROUTES` in `auth.ts`, the proxy denies access.

### "I clicked the magic link but nothing happens."

1. Check the link hasn't expired (Stytch magic links have a default TTL — check Stytch dashboard settings).
2. Check the backend logs for `auth: magic link callback verify failed`.
3. Ensure `StytchRedirectURL` in config matches the backend's `GET /api/auth/callback` URL.

### "My school name is already taken."

The system checks `tenants.name` for duplicates. If your school already exists, you are added as a new **TEACHER** member (not SCHOOL_ADMIN). Contact your school's admin to upgrade your role if needed.

### "I can't invite someone — it says 'user is already a member'."

Each email can only have one active membership per school. If the user was previously invited but hasn't accepted, there may be a pending invitation. Revoke the old invitation first.

### "The invitation email didn't arrive."

1. Check spam/junk folder.
2. The invitation is stored locally even if the Stytch email send fails — the admin can retry or check the invitation list for error details.
3. Verify the Stytch project is configured correctly (valid email sender, not in test mode).

### "I see 'MFA required' error."

Your Stytch organization has Multi-Factor Authentication enabled. Complete the MFA challenge in your Stytch authentication flow before attempting registration or invite acceptance.

---

## Related files

| File | Purpose |
|---|---|
| `backend/internal/auth/domain.go` | Domain models, sentinel errors, `IdentityProvider` and `Repository` interfaces |
| `backend/internal/auth/handler.go` | HTTP handlers: Discover, Verify, Register, MagicLinkCallback, AcceptInvite, Me, Logout |
| `backend/internal/auth/service.go` | Business logic: IST caching, registration orchestration, invite acceptance |
| `backend/internal/auth/repository.go` | Pgx-backed Postgres repository |
| `backend/internal/auth/stytch.go` | Stytch B2B API adapter |
| `backend/internal/middleware/errors.go` | `HTTPError` — the single error-to-HTTP mapper |
| `backend/internal/middleware/auth.go` | `RequireRole` middleware for route-level RBAC |
| `backend/internal/members/domain.go` | Member and invitation domain models |
| `backend/internal/members/handler.go` | HTTP handlers for /api/v1/members and /api/v1/invitations |
| `backend/internal/members/service.go` | Invitation creation and bulk invite business logic |
| `frontend/src/lib/api/client.ts` | API fetch wrapper with CSRF and 401 eviction |
| `frontend/src/lib/auth.ts` | Cookie constants and role-to-route mappings |
| `frontend/src/lib/auth-server.ts` | Server-side HMAC verification (Node.js) |
| `frontend/src/proxy.ts` | Edge middleware — route protection and cookie verification |
| `frontend/src/hooks/use-auth.ts` | React Query hooks for auth mutations |
| `frontend/src/features/auth/components/login-page.tsx` | Login page UI |
| `frontend/src/features/auth/components/register-form.tsx` | Registration form UI |
