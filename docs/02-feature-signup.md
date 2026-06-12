# Signup & Authentication Flow

This document describes the complete signup-to-session lifecycle for Somotracker, from a user entering their email to landing on their dashboard with a valid session cookie.

---

## Table of Contents

- [Architecture Overview](#architecture-overview)
- [Phase 1: Discovery — Sending the Magic Link](#phase-1-discovery--sending-the-magic-link)
- [Phase 2: Verification — Browser Callback & IST Caching](#phase-2-verification--browser-callback--ist-caching)
- [Phase 3: Registration — Org Creation & Session Issuance](#phase-3-registration--org-creation--session-issuance)
- [Post-Registration: Session Validation](#post-registration-session-validation)
- [Logout](#logout)
- [Data Stores](#data-stores)
  - [PostgreSQL Schema (auth tables)](#postgresql-schema-auth-tables)
  - [Redis Key Layout](#redis-key-layout)
  - [Cookie Layout](#cookie-layout)
- [Security Considerations](#security-considerations)
- [Idempotency & Edge Cases](#idempotency--edge-cases)
- [Error Taxonomy](#error-taxonomy)

---

## Architecture Overview

```
┌─────────────────┐     ┌──────────────────┐     ┌───────────────────┐
│  Next.js        │     │  Go Backend      │     │  Stytch B2B API   │
│  (Frontend)     │     │  (Fiber + Fx)    │     │                   │
└────────┬────────┘     └────────┬─────────┘     └────────┬──────────┘
         │                      │                        │
         │                      │                        │
         │     POST /api/auth/discover                    │
         │─────────────────────>│                        │
         │                      │  Discovery.Send()      │
         │                      │────────────────────────>│
         │<── 200 OK ───────────│                        │── Dispatches email
         │                      │                        │
         │    (User clicks magic link in email)           │
         │                      │                        │
         │                      │  GET /api/auth/callback │
         │                      │<──?token=xxx────────────│ (Stytch redirect)
         │                      │                        │
         │                      │  Discovery.Authenticate │
         │                      │────────────────────────>│
         │                      │<── IST ────────────────│
         │                      │                        │
         │                      │  ┌─ Redis: ist:{env}:{uuid}
         │                      │  │  (IST cached, 10min)
         │                      │  └─
         │                      │
         │  302 → /register     │
         │<──?session_ref=uuid──│ (csrf_token cookie set)
         │                      │
         │  POST /api/auth/register
         │  {session_ref,       │
         │   school_name,       │
         │   first_name,        │
         │   last_name}         │
         │─────────────────────>│
         │                      │  ┌─ Redis: GET+DEL ist:{env}:{uuid}
         │                      │  │  (one-time IST consumption)
         │                      │  └─
         │                      │
         │                      │  Organizations.Create()
         │                      │────────────────────────>│
         │                      │<── orgID ───────────────│
         │                      │                        │
         │                      │  IntermediateSessions
         │                      │  .Exchange(ist, orgID)
         │                      │────────────────────────>│
         │                      │<── StytchSessionToken───│
         │                      │                        │
         │                      │  ┌─ Postgres TX:
         │                      │  │  INSERT tenants + users + sessions
         │                      │  │  (single transaction)
         │                      │  └─
         │                      │
         │                      │  ┌─ Redis: session:{opaque} → StytchToken
         │                      │  │  (30-day TTL)
         │                      │  └─
         │                      │
         │<── Set-Cookie ───────│
         │   somo_sid (HttpOnly)│
         │   csrf_token         │
```

---

## Phase 1: Discovery — Sending the Magic Link

**Endpoint:** `POST /api/auth/discover`

**Request:**
```json
{
  "email": "teacher@school.com"
}
```

**Flow:**
1. Frontend sends the user's email address to the backend.
2. Backend calls `Service.Discover()` which invokes Stytch's `MagicLinks.Email.Discovery.Send()` with the email and a `DiscoveryRedirectURL` pointing to `GET /api/auth/callback` on the backend.
3. Stytch sends a discovery magic link email to the user.
4. Backend returns HTTP 200 OK (no content). The response is deliberately generic to prevent email enumeration.

**Stytch Parameters:**
| Parameter | Value |
|---|---|
| `EmailAddress` | User's email |
| `DiscoveryRedirectURL` | `http://localhost:3030/api/auth/callback` (configurable via `STYTCH_REDIRECT_URL` env) |

**Idempotency:** Calling discover multiple times with the same email is safe — Stytch handles deduplication and rate-limiting of email sends.

---

## Phase 2: Verification — Browser Callback & IST Caching

**Two paths** lead here depending on how the token arrives:

### Path A: Browser Redirect (Magic Link Click)

**Endpoint:** `GET /api/auth/callback?token=xxx&stytch_token_type=discovery`

When the user clicks the magic link in their email, Stytch redirects the browser directly to this endpoint. The handler:

1. Extracts `token` from query parameters.
2. Calls `Service.Verify()` which:
   - Calls Stytch `MagicLinks.Discovery.Authenticate()` with the token.
   - Stytch validates the token and returns an **Intermediate Session Token (IST)**.
   - Generates a UUID v4 `session_ref`.
   - Caches the IST in **Redis** at key `ist:{env}:{sessionRef}` with a **10-minute TTL**.
   - Returns the `session_ref`.
3. Generates a **CSRF token** (crypto-random base64) and sets it as a non-HttpOnly `csrf_token` cookie.
4. **Redirects** the browser to `http://localhost:3000/register?session_ref=<uuid>`.

**Redis save (IST isolation):**
```
Key:   ist:development:550e8400-e29b-41d4-a716-446655440000
Value: <raw IST string from Stytch>
TTL:   10 minutes
```

> **Security:** The IST is **never exposed to the browser**. The frontend only receives the `session_ref` UUID in the URL.

### Path B: API Call (Direct Token Verification)

**Endpoint:** `POST /api/auth/verify`

**Request:**
```json
{
  "token": "stytch-magic-link-token-xxx"
}
```

Returns the `session_ref` as JSON for programmatic consumption (e.g., the frontend extracts the token from the URL and POSTs it here instead of relying on the redirect).

**Response:**
```json
{
  "session_ref": "550e8400-e29b-41d4-a716-446655440000"
}
```

---

## Phase 3: Registration — Org Creation & Session Issuance

**Endpoint:** `POST /api/auth/register`

**Request:**
```json
{
  "session_ref": "550e8400-e29b-41d4-a716-446655440000",
  "school_name": "Green Valley Academy",
  "first_name": "Jane",
  "last_name": "Muthoni"
}
```

**Headers:** Must include `X-CSRF-Token` matching the `csrf_token` cookie (set during callback).

### Execution Steps (in order)

#### Step 1 — Payload Validation
Validates the `RegistrationPayload`:
- `school_name`: required, 2–100 chars, printable UTF-8 only, trimmed
- `session_ref`: required, must be a valid UUID v4

Returns HTTP 422 if validation fails.

#### Step 2 — Atomic IST Consumption (Redis)
Executes a **Lua script** that atomically reads and deletes the IST from Redis:
```
GET "ist:development:<sessionRef>"
DEL "ist:development:<sessionRef>"
```
If the IST doesn't exist (already consumed or expired), returns `ErrExpiredToken` (HTTP 401).

> **One-time use:** The IST is deleted from Redis immediately after reading. TTL expiry alone is not relied upon.

#### Step 3 — Idempotency Check (Tenant Name)
Checks if a tenant already exists with the given `school_name` in Postgres. If it exists, the org creation is skipped (see Idempotency section below).

#### Step 4 — Stytch Organization Creation
Calls Stytch `Organizations.Create()` with the school name. Stytch creates a new organization and returns a unique `organization_id`.

If the org already exists in Stytch (e.g., from a previous partially-successful registration), this call will fail. See [Edge Cases](#idempotency--edge-cases).

#### Step 5 — IST Exchange
Calls Stytch `Discovery.IntermediateSessions.Exchange()` with the IST and the new `organization_id`:
- Stytch validates the IST, creates a member profile, and returns a **Stytch session token** + `MemberAuthenticated` boolean.
- If `MemberAuthenticated` is `false` (MFA required), the flow is **blocked** and returns HTTP 401 `mfa_required`.

#### Step 6 — Opaque Session Token Generation
Generates a 32-byte cryptographically random token via `crypto/rand`, hex-encoded:
```
Example: "a1b2c3d4e5f6..."
```
This is the opaque key — the value the browser holds in its cookie.

#### Step 7 — Postgres Transaction
A single database transaction inserts three records:

```sql
BEGIN;

INSERT INTO tenants (name, slug, stytch_org_id)
VALUES ('Green Valley Academy', 'green-valley-academy', 'stytch-org-xxx')
RETURNING id;  -- → tenantID

INSERT INTO users (email, tenant_id, first_name, last_name, external_auth_id)
VALUES ('', tenantID, 'Jane', 'Muthoni', 'stytch-member-xxx')
RETURNING id;  -- → userID

INSERT INTO sessions (token, user_id, tenant_id, stytch_member_id,
                      stytch_org_id, stytch_session_token, device_fingerprint, expires_at)
VALUES (opaqueToken, userID, tenantID, 'stytch-member-xxx',
        'stytch-org-xxx', 'stytch-session-xxx', fingerprint, now() + 30d);

COMMIT;
```

If any insert fails, the **entire transaction rolls back**. If the Postgres write fails after a successful Stytch org creation, a `WARN` log is emitted with the `stytch_org_id` for manual reconciliation.

#### Step 8 — Redis Session Cache
Persists the mapping:
```
Key:   session:a1b2c3d4e5f6...    (the opaque key)
Value: stytch-session-xxx         (the actual Stytch session token)
TTL:   30 days
```

#### Step 9 — Cookie Issuance
Two cookies are set in the response:

| Cookie | Value | HttpOnly | Secure | SameSite | Max-Age |
|---|---|---|---|---|---|
| `somo_sid` | Opaque token (32-byte hex) | ✅ Yes | ✅ (prod) | Lax | 30 days |
| `csrf_token` | Crypto-random base64 | ❌ No | ✅ (prod) | Lax | 30 days |

Response: **HTTP 204 No Content** (no body).

---

## Post-Registration: Session Validation

### `GET /api/auth/me`

Reads the `somo_sid` cookie, validates the session, and returns user context.

**Flow:**
1. Extracts opaque token from `somo_sid` cookie.
2. Checks Redis for key `session:{opaqueToken}` (fast existence check).
3. If missing → HTTP 401 `expired_token`.
4. Cross-references Postgres for full session data (user_id, tenant_id).
5. If Postgres has no matching record but Redis does → deletes stale Redis key → HTTP 401.
6. Returns user context:
```json
{
  "user_id": "550e8400-e29b-41d4-a716-446655440000",
  "tenant_id": "550e8400-e29b-41d4-a716-446655440001"
}
```

---

## Logout

### `DELETE /api/auth/session`

1. Reads `somo_sid` cookie.
2. Deletes session from Postgres `sessions` table.
3. Deletes `session:{opaqueToken}` from Redis.
4. Clears `somo_sid` cookie (Max-Age = -1).
5. Clears `csrf_token` cookie (Max-Age = -1).
6. Returns HTTP 204 No Content.

---

## Data Stores

### PostgreSQL Schema (auth tables)

```sql
-- ============================================================
-- tenants
-- ============================================================
CREATE TABLE tenants (
    id            UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    name          VARCHAR(255) NOT NULL,
    slug          VARCHAR(255) NOT NULL UNIQUE,
    stytch_org_id VARCHAR(255) NOT NULL UNIQUE,
    created_at    TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

-- ============================================================
-- users
-- ============================================================
CREATE TABLE users (
    id               UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    email            VARCHAR(255) NOT NULL,
    tenant_id        UUID        REFERENCES tenants(id),
    first_name       VARCHAR(255) NOT NULL DEFAULT '',
    last_name        VARCHAR(255) NOT NULL DEFAULT '',
    is_active        BOOLEAN     NOT NULL DEFAULT TRUE,
    external_auth_id VARCHAR(255) NOT NULL,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX idx_users_email ON users (email);
CREATE UNIQUE INDEX idx_users_external_auth_id ON users (external_auth_id);

-- ============================================================
-- sessions
-- ============================================================
CREATE TABLE sessions (
    id                  UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    token               VARCHAR(128) NOT NULL UNIQUE,
    user_id             UUID        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    tenant_id           UUID        NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    stytch_member_id    VARCHAR(255) NOT NULL,
    stytch_org_id       VARCHAR(255) NOT NULL,
    stytch_session_token VARCHAR(512) NOT NULL DEFAULT '',
    device_fingerprint  VARCHAR(128) NOT NULL DEFAULT '',
    expires_at          TIMESTAMPTZ  NOT NULL,
    created_at          TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);
```

### Redis Key Layout

| Key Pattern | Value | TTL | Purpose |
|---|---|---|---|
| `ist:{env}:{uuid}` | Raw IST string from Stytch | 10 min | One-time IST cache (deleted on read) |
| `session:{opaqueKey}` | Stytch session token | 30 days | Opaque key → Stytch token mapping |
| `ratelimit:{ip}` | Sorted set of timestamps | 1 min | Sliding-window rate limiter |

### Cookie Layout

| Cookie | HttpOnly | Secure | SameSite | Max-Age | Readable by JS |
|---|---|---|---|---|---|
| `somo_sid` | ✅ | ✅ (prod) | Lax | 30 days | ❌ |
| `csrf_token` | ❌ | ✅ (prod) | Lax | 30 days | ✅ |

---

## Security Considerations

### 1. IST Isolation
The Intermediate Session Token (IST) from Stytch is **never exposed to the browser**. It is cached in Redis with the key pattern `ist:{env}:{uuid}` and deleted immediately after the first read via an atomic Lua script. The frontend receives only a UUID reference (`session_ref`).

### 2. Opaque Session Tokens
The session token stored in the `somo_sid` cookie is a cryptographically random 32-byte hex string generated by the backend via `crypto/rand`. It is not a Stytch JWT or session token. The actual Stytch session token is stored server-side as the **value** of the Redis key `session:{opaqueKey}`.

### 3. CSRF Double-Submit Cookie
- On every state-changing request (POST, PUT, DELETE, PATCH), the CSRF middleware compares the `csrf_token` cookie against the `X-CSRF-Token` request header.
- The cookie is non-HttpOnly so the frontend JS can read it, but it is SameSite=Lax and Secure in production.
- Comparison uses `subtle.ConstantTimeCompare` to prevent timing attacks.

### 4. Device Fingerprinting
Every request is fingerprinted by the middleware (SHA-256 of `IP|User-Agent|Accept-Language`) and stored as `c.Locals("device_fingerprint")`. This is persisted in the `sessions` table `device_fingerprint` column for session binding.

### 5. MFA Enforcement
The Stytch exchange response includes a `MemberAuthenticated` flag. If `false`, the registration flow is blocked with HTTP 401 `mfa_required` — no Postgres writes occur.

### 6. Rate Limiting
A Redis-based sliding window rate limiter (60 requests per minute per IP) is applied globally. Returns HTTP 429 with `Retry-After` header when exceeded. Fail-open when Redis is unavailable.

### 7. Security Headers
- `X-Content-Type-Options: nosniff`
- `X-Frame-Options: DENY`
- `Content-Security-Policy: default-src 'self'` (API routes only)

---

## Idempotency & Edge Cases

### Duplicate Registration (Same School Name)
If `TenantExistsByName(schoolName)` returns `true` (tenant already exists in Postgres), the flow still proceeds through Stytch org creation. Stytch handles deduplication by name — if the org already exists, `Organizations.Create()` fails, and the current implementation treats this as an internal error.

**Future improvement:** Add `GetTenantByStytchOrgID` to the repository and look up the existing org ID instead of attempting org creation.

### Duplicate Registration (Same `session_ref`)
The IST is atomically read-and-deleted from Redis. If the same `session_ref` is reused, the Lua script returns `nil` and the flow immediately returns `ErrExpiredToken` (HTTP 401). This prevents replay attacks.

### Postgres Failure After Stytch Org Creation
If the Postgres transaction fails after a successful `Organizations.Create()` in Stytch, the `stytch_org_id` is logged at `WARN` level for manual reconciliation. Stytch has no two-phase commit, so this inconsistency cannot be automatically recovered.

### Tenant Name Conflicts
`school_name` is validated at the payload level (2–100 chars, printable UTF-8). A slug is generated from the name and stored as `UNIQUE` in the `tenants` table. If two tenants with the same name exist in different envs, they get different Stytch org IDs.

---

## Error Taxonomy

| Error Sentinel | HTTP Status | `error` field | When |
|---|---|---|---|
| `ErrInvalidInput` | 422 | `invalid_input` | Payload validation failure |
| `ErrExpiredToken` | 401 | `expired_token` | Magic link expired, IST consumed/expired, session expired |
| `ErrMFARequired` | 401 | `mfa_required` | Stytch exchange returned `MemberAuthenticated: false` |
| `ErrOrgAlreadyExists` | 409 | `org_already_exists` | Tenant name conflicts |
| `ErrNotFound` | 404 | `not_found` | Resource lookup failure |
| `ErrInternal` | 500 | `internal_error` | All other errors (Stytch failures, Redis/Postgres errors) |

**All error responses follow the same shape:**
```json
{
  "error": "expired_token",
  "message": "session expired or invalid"
}
```

---

## Environment Variables

| Variable | Default | Description |
|---|---|---|
| `STYTCH_PROJECT_ID` | — | Stytch B2B project ID |
| `STYTCH_SECRET` | — | Stytch B2B secret |
| `STYTCH_REDIRECT_URL` | `http://localhost:3030/api/auth/callback` | Where Stytch redirects after magic link click |
| `STYTCH_ENV` | `test` | Stytch environment (`test` or `live`) |
| `ALLOWED_ORIGINS` | `http://localhost:3000` | CORS allowed origins |
| `COOKIE_DOMAIN` | `localhost` | Domain attribute for cookies |
| `APP_ENV` | `development` | Controls `Secure` flag on cookies |
| `DATABASE_URL` | local Postgres | PostgreSQL connection string |
| `REDIS_URL` | `redis:6379` | Redis server address |

---

## Files

| File | Role |
|---|---|
| `backend/internal/auth/handler.go` | HTTP handlers for all auth endpoints |
| `backend/internal/auth/service.go` | Core business logic (Discover, Verify, Register, GetSession, Logout) |
| `backend/internal/auth/stytch.go` | Stytch B2B SDK adapter (only file importing Stytch) |
| `backend/internal/auth/domain.go` | Domain types, interfaces (IdentityProvider, Repository), errors |
| `backend/internal/auth/repository.go` | PostgreSQL implementation of Repository interface |
| `backend/internal/auth/module.go` | Uber fx module wiring |
| `backend/internal/auth/service_test.go` | Unit tests with in-memory mocks |
| `backend/internal/middleware/security.go` | CSRF, rate limiter, fingerprint middleware |
| `backend/internal/config/config.go` | Environment configuration |
| `backend/internal/database/migrations/` | SQL migrations for auth tables |
