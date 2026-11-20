# Authentication Flow: Signup & Signin

## Overview

This document describes the complete OTP-based authentication flow for the Somotracker application. The system uses email-based OTP (One-Time Password) verification with optional Turnstile bot protection, supporting both new user signup (with onboarding) and existing user signin.

---

## Architecture

```
┌─────────────┐     ┌─────────────┐     ┌─────────────┐
│   Frontend  │────▶│  BFF Proxy  │────▶│   Backend   │
│  (Next.js)  │     │  (Next.js)  │     │   (Go)      │
└─────────────┘     └─────────────┘     └─────────────┘
       │                                    │
       │  1. POST /api/auth/otp/request     │
       │  2. POST /api/auth/otp/verify      │
       │  3. POST /api/auth/onboard         │
       ▼                                    ▼
  ┌─────────┐                        ┌─────────────┐
  │ Browser │                        │  PostgreSQL │
  │ Cookies │                        │  + Redis    │
  └─────────┘                        └─────────────┘
```

---

## Flow 1: New User Signup (First Time)

### Step 1: Email Entry + Turnstile (Frontend: `/login`)

**User Action:** Enters email, completes Turnstile challenge, clicks "Send code"

**Frontend Request:**
```http
POST /api/auth/otp/request
Content-Type: application/json
Cookie: (none)

{
  "email": "user@example.com",
  "turnstile_token": "cf-turnstile-response-token"
}
```

**BFF Proxy** (`/api/auth/otp/request` → `/api/v1/auth/otp/request`):
- Forwards request to backend with `credentials: include`
- No session cookie exists yet

**Backend Processing** (`POST /api/v1/auth/otp/request`):
1. Normalize email (lowercase)
2. Validate email format
3. If `TURNSTILE_ENABLED=true`, verify Turnstile token with Cloudflare
4. Generate 6-digit OTP code
5. Create `login_challenges` record:
   - `challenge_id` (UUID v7)
   - `email_normalized`
   - `code_mac` = HMAC-SHA256(secret, challenge_id || code)
   - `expires_at` = now + 10 minutes
   - `max_attempts` = 5
6. Create `email_outbox` record (for async email sending)
7. Fire-and-forget: Send OTP via Brevo (goroutine)
8. Return challenge ID

**Backend Response:**
```json
{
  "ok": true,
  "message": "If an account exists for this email, a code has been sent.",
  "challenge_id": "018f1e8a-7b3c-4d2e-9f1a-2b3c4d5e6f7a"
}
```

**Frontend:** Stores `challenge_id`, switches UI to "Verify code" step

---

### Step 2: OTP Verification (Frontend: `/login` - OTP Step)

**User Action:** Enters 6-digit code, clicks "Verify"

**Frontend Request:**
```http
POST /api/auth/otp/verify
Content-Type: application/json
Cookie: (none)

{
  "challenge_id": "018f1e8a-7b3c-4d2e-9f1a-2b3c4d5e6f7a",
  "code": "123456"
}
```

**Backend Processing** (`POST /api/v1/auth/otp/verify`):
1. Look up challenge by `challenge_id`
2. Validate:
   - Not expired (`expires_at > now`)
   - Not used (`used_at IS NULL`)
   - Attempts < max_attempts
3. Compute expected HMAC: `HMAC(secret, challenge_id || provided_code)`
4. Constant-time compare with stored `code_mac`
5. If invalid: increment `attempt_count`, return error
6. If valid: mark challenge `used_at = now`
7. Look up user by `normalized_email`
8. **If user NOT found (new user):**
   - Create Organization: `org-{challenge_id[:8]}`
   - Create School: "School" (placeholder)
   - Create User: email verified, no name
   - `justCreated = true`
   - `sessionType = "temporary"` (30 min TTL)
   - `needsOnboarding = true`
9. **If user EXISTS:**
   - `justCreated = false`
   - `sessionType = "full"` (30 day TTL)
   - `needsOnboarding = false`
10. Generate session token (32 bytes hex)
11. Hash token: `SHA256(token)`
12. Create `sessions` record:
    - `token_hash`
    - `user_id`, `organization_id`
    - `active_school_id` (NULL if multiple schools)
    - `session_type` ("temporary" or "full")
    - `expires_at`
13. Set cookies on response:
    - `__Host-session` (or `session` in dev) = raw token, HttpOnly, Secure, SameSite=Lax
    - `needs_onboarding` = "true" (if temporary), HttpOnly

**Backend Response:**
```json
{
  "ok": true,
  "message": "Code verified successfully.",
  "session_token": "a1b2c3d4e5f6...",  // raw token (for reference)
  "needs_onboarding": true
}
```

**Frontend:** Receives response, cookies automatically set by browser via BFF proxy forwarding `Set-Cookie` headers. Checks `needs_onboarding`:
- If `true`: `router.replace('/onboarding')`
- If `false`: `router.replace('/dashboard')`

---

### Step 3: Onboarding (Frontend: `/onboarding`)

**Prerequisites:** Valid temporary session cookie + `needs_onboarding=true` cookie

**User Action:** Fills org name, school name, user name, clicks "Create organization & school"

**Frontend Request:**
```http
POST /api/auth/onboard
Content-Type: application/json
Cookie: session=<token>; needs_onboarding=true

{
  "organization_name": "Acme School District",
  "school_name": "Lincoln High School",
  "user_name": "Jane Doe"
}
```

**BFF Proxy:** Forwards to backend with session cookie

**Backend Processing** (`POST /api/v1/auth/onboard`):
1. Read `session` cookie → hash → lookup session
2. Validate: session exists, type = "temporary"
3. Update organization: name + `onboarding_completed_at = now()`
4. Update user: name = provided user_name
5. Create school with provided name
6. Upgrade all temporary sessions for user: `session_type = 'full'`
7. Clear `needs_onboarding` cookie (MaxAge=-1)

**Backend Response:**
```json
{
  "ok": true,
  "message": "Onboarding complete",
  "needs_onboarding": false
}
```

**Frontend:** `router.replace('/dashboard')`

---

### Step 4: Dashboard Access (Frontend: `/dashboard`)

**Proxy Check** (on every navigation):
1. Read `session` cookie
2. Read `needs_onboarding` cookie
3. **No session** → redirect `/login`
4. **Has session + needs_onboarding=true** → redirect `/onboarding`
5. **Has session + needs_onboarding=false** → allow access

---

## Flow 2: Existing User Signin

### Steps 1-2: Same as New User (Email → OTP Verify)

**Difference at Step 2:** User exists in database
- `justCreated = false`
- `sessionType = "full"` (30 day TTL)
- `needsOnboarding = false`
- No `needs_onboarding` cookie set

**Frontend:** After verify, `router.replace('/dashboard')` directly

---

## Flow 3: Returning User (Session Restore)

**Scenario:** User closes browser, returns later (or hard refresh)

**Request:** `GET /dashboard` (or any protected route)

**Proxy Check:**
1. Reads `session` cookie from request headers
2. Reads `needs_onboarding` cookie
3. **Cookie present, valid session** → allows access (no backend call)
4. **Cookie present, needs_onboarding=true** → redirects `/onboarding`
5. **No cookie** → redirects `/login`

**Backend Validation:** On every API call, backend middleware validates session:
- Hash token from cookie
- Lookup in `sessions` table
- Check `expires_at > now`, `revoked_at IS NULL`
- Return 401 if invalid

---

## Cookie Configuration

| Environment | Cookie Name | Secure | Notes |
|-------------|-------------|--------|-------|
| Development (HTTP) | `session` | `false` | No `__Host-` prefix |
| Production (HTTPS) | `__Host-session` | `true` | Requires HTTPS, host-only |

**Session Cookie Attributes:**
- `HttpOnly: true` (JS cannot read)
- `SameSite: Lax`
- `Path: /`
- `MaxAge`: 30 min (temporary) or 30 days (full)

**Onboarding Flag Cookie:**
- Name: `needs_onboarding`
- Value: `"true"` or cleared
- `HttpOnly: true`
- Same attributes as session cookie

---

## Database Schema (Relevant Tables)

```sql
-- Login challenge (OTP)
CREATE TABLE login_challenges (
  id UUID PRIMARY KEY DEFAULT uuidv7(),
  email_normalized TEXT NOT NULL,
  code_mac BYTEA NOT NULL,           -- HMAC(secret, id || code)
  expires_at TIMESTAMPTZ NOT NULL,
  attempt_count INT DEFAULT 0,
  max_attempts INT DEFAULT 5,
  used_at TIMESTAMPTZ,
  created_at TIMESTAMPTZ DEFAULT NOW()
);

-- Sessions
CREATE TABLE sessions (
  id UUID PRIMARY KEY DEFAULT uuidv7(),
  token_hash BYTEA NOT NULL UNIQUE,  -- SHA256(raw_token)
  user_id UUID NOT NULL REFERENCES users(id),
  organization_id UUID NOT NULL REFERENCES organizations(id),
  active_school_id UUID REFERENCES schools(id),
  session_type TEXT NOT NULL DEFAULT 'full' CHECK (session_type IN ('temporary','full')),
  created_at TIMESTAMPTZ DEFAULT NOW(),
  expires_at TIMESTAMPTZ NOT NULL,
  revoked_at TIMESTAMPTZ,
  revocation_reason TEXT
);

-- Users
CREATE TABLE users (
  id UUID PRIMARY KEY DEFAULT uuidv7(),
  organization_id UUID NOT NULL REFERENCES organizations(id),
  email TEXT NOT NULL,
  normalized_email TEXT NOT NULL UNIQUE,
  name TEXT,                          -- set during onboarding
  email_verified_at TIMESTAMPTZ,      -- set on OTP verify
  status user_status DEFAULT 'ACTIVE',
  created_at TIMESTAMPTZ DEFAULT NOW()
);

-- Organizations
CREATE TABLE organizations (
  id UUID PRIMARY KEY DEFAULT uuidv7(),
  name TEXT NOT NULL UNIQUE,
  onboarding_completed_at TIMESTAMPTZ,  -- set on onboarding
  status org_status DEFAULT 'ACTIVE'
);
```

---

## Error Responses

All errors follow standard format:
```json
{
  "code": "ERROR_CODE",
  "message": "Human readable message"
}
```

| Code | HTTP | Scenario |
|------|------|----------|
| `INVALID_EMAIL` | 400 | Malformed email |
| `TURNSTILE_REQUIRED` | 400 | Turnstile enabled but no token |
| `TURNSTILE_INVALID` | 400 | Turnstile verification failed |
| `INVALID_TOKEN` | 401 | Wrong/expired/used OTP code |
| `RATE_LIMITED` | 429 | Too many requests |
| `UNAUTHENTICATED` | 401 | No/missing session |
| `INVALID_REQUEST` | 400 | Malformed request body |

---

## Security Considerations

1. **Constant-time HMAC comparison** - prevents timing attacks
2. **Rate limiting** - per IP and per email (configured in backend)
3. **HMAC'd OTP storage** - code never stored in plaintext
4. **Session token hashing** - SHA256 stored, raw token only in cookie
5. **HttpOnly cookies** - XSS cannot steal session
6. **Secure cookies in prod** - `__Host-` prefix enforces HTTPS
7. **Turnstile bot protection** - optional, configurable
8. **Challenge expiration** - 10 min default
9. **Max attempts** - 5 default per challenge
10. **One-time use** - challenge marked used after verify

---

## Environment Variables

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `EMAIL_HMAC_KEY` | Yes | - | 32+ byte secret for OTP HMAC |
| `BREVO_API_KEY` | Yes | - | Brevo (Sendinblue) API key |
| `BREVO_SENDER_EMAIL` | No | noreply@somotracker.com | From email |
| `BREVO_SENDER_NAME` | No | Somotracker | From name |
| `TURNSTILE_ENABLED` | No | false | Enable Cloudflare Turnstile |
| `TURNSTILE_SECRET_KEY` | If enabled | - | Turnstile secret key |
| `SESSION_COOKIE_NAME` | No | session | Cookie name (dev: `session`, prod: `__Host-session`) |
| `OTP_VALIDITY_MINUTES` | No | 10 | Challenge expiration |
| `OTP_MAX_ATTEMPTS` | No | 5 | Max verify attempts |