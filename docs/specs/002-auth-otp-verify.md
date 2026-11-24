# SPEC-002: Verify Login OTP

Endpoint: `POST /api/v1/auth/otp/verify`

## Metadata

| Field | Value |
|-------|-------|
| id | SPEC-002 |
| title | Verify Login OTP → Session |
| status | approved |
| depends_on | specs/glossary.md, specs/001-auth-otp-request.md |
| api_endpoint | POST /api/v1/auth/otp/verify |
| owner | backend |

## Summary

Accepts a `challenge_id` and 6-digit OTP code, verifies the HMAC, resolves or creates the user,
creates an opaque session, and returns `needs_onboarding`. The handler sets the session cookie
(`__Host-session` prod / `session` dev) and the onboarding flag cookie directly on the response.

The frontend calls the Next.js BFF API route, which proxies to the Go backend and forwards
`Set-Cookie` headers to the browser.

## Request

```
POST /api/v1/auth/otp/verify
Content-Type: application/json
Cookie: (none)

{
  "challenge_id": "uuid",
  "code": "123456"
}
```

| Field | Type | Required | Notes |
|-------|------|----------|-------|
| challenge_id | string (uuid) | Yes | ID from OTP request response |
| code | string | Yes | Exactly 6 digits |

## Response (success)

```
HTTP 200 OK
Content-Type: application/json
Set-Cookie: __Host-session=<raw_token>; HttpOnly; Secure; SameSite=Lax; Path=/
Set-Cookie: needs_onboarding=true; HttpOnly; Secure; SameSite=Lax; Path=/   (only if new user)

{
  "ok": true,
  "message": "Code verified successfully.",
  "needs_onboarding": false
}
```

- `session_token` is intentionally **not** returned in the JSON body — it is delivered only via the
  HttpOnly cookie.
- `needs_onboarding` is `true` for brand-new users (just-in-time account provisioned), `false` for
  existing users.

## Response (errors)

| Status | Code | Condition |
|--------|------|-----------|
| 400 | `INVALID_REQUEST` | Malformed JSON or missing/invalid fields |
| 401 | `INVALID_TOKEN` | Invalid, expired, used, or exceeded-attempts challenge |
| 500 | `INTERNAL_ERROR` | Unexpected server error |

## Business rules

1. **Challenge lookup**: find `login_challenge` by ID; return 401 if not found.
2. **Expiry check**: reject if `expires_at < now`.
3. **Attempt limit**: reject if `attempt_count >= max_attempts` (default 5).
4. **Single use**: reject if `used_at IS NOT NULL`.
5. **HMAC verification**: compute `HMAC-SHA256(EMAIL_HMAC_KEY, challenge_id || code)` and compare
   constant-time with stored `code_mac`. Never store plaintext code.
6. **Mark used**: set `used_at = NOW()` on successful verification.
7. **User resolution**: look up user by `challenge.email_normalized`.
   - **If not found (new user)**: just-in-time provision organization (`org-{challenge_id[:8]}`),
     school ("School"), and user with email-verified flag. `session_type = "temporary"` (30 min TTL),
     `needs_onboarding = true`.
   - **If found**: `session_type = "full"` (30 day TTL), `needs_onboarding = false`.
8. **School resolution**: get user's schools. If exactly one, auto-select as `active_school_id`.
   If multiple, `active_school_id = NULL` and client selects via separate endpoint.
9. **Session creation**: generate 256-bit opaque token, SHA-256 hash it, insert `sessions` row with
   `token_hash`, `user_id`, `organization_id`, `active_school_id`, `expires_at`.
10. **Cookies**: handler sets `__Host-session` (raw token, HttpOnly, Secure in prod, SameSite=Lax)
    and `needs_onboarding=true` (if temporary session). No `Domain` attribute (`__Host-` forbids it).

## Tests required

| Test type | What to verify |
|-----------|----------------|
| Unit | HMAC verification rejects wrong code |
| Unit | Constant-time comparison (no timing leak) |
| Integration | Rejects expired challenge |
| Integration | Rejects used challenge |
| Integration | Rejects exceeded max attempts |
| Integration | Creates session row on success |
| Integration | Sets `__Host-session` cookie on response |
| Integration | Auto-selects single school as active_school_id |
| Integration | Leaves active_school_id NULL for multiple schools |
| Integration | JIT-provisions user/org/school for new email |
| Contract | POST `/api/v1/auth/otp/verify` returns session cookie |
| Contract | `session_token` not in JSON body |

## Acceptance criteria

Given a valid challenge ID and correct 6-digit code
When the client POSTs `/api/v1/auth/otp/verify`
Then the response is HTTP 200 with `ok: true`
And a `sessions` row exists with the token hash
And the `__Host-session` cookie is set on the response

Given an expired challenge
When the client POSTs `/api/v1/auth/otp/verify`
Then the response is HTTP 401 `INVALID_TOKEN`

Given a code that has already been used
When the client POSTs `/api/v1/auth/otp/verify`
Then the response is HTTP 401 `INVALID_TOKEN`

Given a user with exactly one active school
When OTP verification succeeds
Then `active_school_id` is set to that school in the session

Given a user with multiple active schools
When OTP verification succeeds
Then `active_school_id` is NULL in the session

Given an email that has never signed up
When OTP verification succeeds
Then a user, organization, and school are JIT-provisioned
And `needs_onboarding` is `true`
And session type is `temporary` (30 min TTL)
