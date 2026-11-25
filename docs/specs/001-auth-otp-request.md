# SPEC-001: Request Login OTP

Endpoint: `POST /api/v1/auth/otp/request`

## Metadata

| Field | Value |
|-------|-------|
| id | SPEC-001 |
| title | Request Login OTP (and Verify OTP → Session) |
| status | approved |
| depends_on | specs/glossary.md, docs/adr/0001-authentication-architecture.md |
| api_endpoint | POST /api/v1/auth/otp/request, POST /api/v1/auth/otp/verify |
| owner | backend |

## Summary

Accepts an email address, validates it, creates a login challenge and an
encrypted email-outbox record in one transaction, and returns a generic
success response that does not reveal whether the account exists.

On OTP verification, resolves the user identity, creates an opaque session,
and returns a session token that the handler sets as the `__Host-session`
cookie.

The frontend calls the Next.js BFF API route, which proxies to the Go
backend. Go sets the session cookie directly on the response that flows
through Next.js to the browser.

## Request — RequestOTP

```
POST /api/v1/auth/otp/request
Content-Type: application/json

{
  "email": "string",
  "turnstile_token": "string | null"
}
```

| Field | Type | Required | Notes |
|-------|------|----------|-------|
| email | string | Yes | Must be valid email syntax |
| turnstile_token | string | No | Required when `TURNSTILE_ENABLED=true` |

## Response (success — always generic)

```
HTTP 200 OK
Content-Type: application/json

{
  "ok": true,
  "message": "If an account exists for this email, a code has been sent."
}
```

Always returns 200 with the same body regardless of whether the email
exists. Never distinguishes registered vs. unregistered addresses.

## Request — VerifyOTP

```
POST /api/v1/auth/otp/verify
Content-Type: application/json

{
  "challenge_id": "uuid",
  "code": "6-digit string"
}
```

## Response — VerifyOTP (success)

```
HTTP 200 OK
Content-Type: application/json

{
  "ok": true,
  "message": "Code verified successfully.",
  "session_token": "opaque-random-token"
}
```

## Response — VerifyOTP (errors)

| Status | Code | Condition |
|--------|------|-----------|
| 400 | `INVALID_REQUEST` | Malformed JSON or missing fields |
| 401 | `INVALID_TOKEN` | Invalid, expired, used, or exceeded-attempts challenge |
| 500 | `INTERNAL_ERROR` | Unexpected server error |

## Business rules

### RequestOTP

1. **Email normalization**: lowercase + trim before any lookup or storage.
2. **Email syntax**: must pass `validate:"required,email"`.
3. **Turnstile**: when `TURNSTILE_ENABLED=true`, verify server-side via Cloudflare Siteverify. When `false` (local dev), skip verification.
4. **Rate limits**: per normalized email, per client IP, and global. Use Redis (shared atomic counters). Do not reveal which limit was hit.
5. **No account enumeration**: response and timing must be identical for existing and non-existing emails.
6. **Transaction**: create `login_challenge` row + `email_outbox` row in one DB transaction. Commit both or neither.
7. **Challenge properties**: six-digit code, 10-minute validity, 5 max attempts, purpose=`login`, single use.
8. **Code storage**: store `HMAC-SHA256(server_secret, challenge_id || code)` as `code_mac`. Never store plaintext code.
9. **HMAC key**: loaded from Doppler, never in repo.
10. **Outbox payload**: encrypted with key outside DB. Contains email + code + expiry.
11. **Resend behavior**: not part of this spec (see SPEC-002).

### VerifyOTP

1. **Challenge lookup**: find `login_challenge` by ID; return 401 if not found.
2. **Expiry check**: reject if `expires_at < now`.
3. **Attempt limit**: reject if `attempt_count >= max_attempts`.
4. **Single use**: reject if `used_at IS NOT NULL`.
5. **HMAC verification**: compute `HMAC-SHA256(server_secret, challenge_id || code)` and compare constant-time with stored `code_mac`.
6. **Mark used**: set `used_at = NOW()` on successful verification.
7. **User resolution**: look up user by `challenge.email_normalized` in `users` table; return 401 if no user exists.
8. **Organization resolution**: get user's `organization_id` from `users` table.
9. **School resolution**: get user's active `user_schools` rows; if exactly one school, auto-select it as `active_school_id`; if multiple, return `active_school_id = null` and let client select via separate endpoint.
10. **Session creation**: generate 256-bit opaque token, SHA-256 hash it, insert `sessions` row with `token_hash`, `user_id`, `organization_id`, `active_school_id`, `expires_at` (30 days).
11. **Cookie**: handler sets `__Host-session` cookie with the raw opaque token, `HttpOnly`, `Secure` (HTTPS), `SameSite=Lax`, `Path=/`.

## Tests required

| Test type | What to verify |
|-----------|----------------|
| Unit | Email normalization (lowercase, trim) |
| Unit | Email syntax validation (valid/invalid formats) |
| Unit | Rate limiter returns 429 after threshold |
| Unit | Generic response body is identical for existing/non-existing email |
| Integration | Creates `login_challenge` + `email_outbox` in same transaction |
| Integration | `code_mac` is HMAC, not plaintext |
| Integration | Turnstile skipped when `TURNSTILE_ENABLED=false` |
| Integration | Turnstile required when `TURNSTILE_ENABLED=true` and no token |
| Integration | VerifyOTP rejects expired challenge |
| Integration | VerifyOTP rejects used challenge |
| Integration | VerifyOTP rejects wrong code (HMAC mismatch) |
| Integration | VerifyOTP creates session row on success |
| Integration | VerifyOTP auto-selects single school as active_school_id |
| Integration | VerifyOTP leaves active_school_id NULL for multiple schools |
| Contract | POST `/api/v1/auth/otp/request` returns 200 with generic body |
| Contract | POST `/api/v1/auth/otp/verify` returns session_token on valid code |
| Contract | POST `/api/v1/auth/otp/verify` sets `__Host-session` cookie |

## Acceptance criteria (Given/When/Then)

Given a valid email and (when enabled) a valid Turnstile token
When the client POSTs `/api/v1/auth/otp/request`
Then the response is HTTP 200 with the generic success body
And a `login_challenge` row exists with `code_mac` set
And an `email_outbox` row exists with `status='pending'`
And the challenge expires 10 minutes after creation

Given an email that does not exist in the system
When the client POSTs `/api/v1/auth/otp/request`
Then the response is identical to an existing email (same status, same body)

Given `TURNSTILE_ENABLED=true` and no Turnstile token
When the client POSTs `/api/v1/auth/otp/request`
Then the response is HTTP 400 `TURNSTILE_REQUIRED`

Given rate limit exceeded for the email
When the client POSTs `/api/v1/auth/otp/request`
Then the response is HTTP 429 `RATE_LIMITED`

Given a valid challenge ID and correct 6-digit code
When the client POSTs `/api/v1/auth/otp/verify`
Then the response is HTTP 200 with `ok: true` and `session_token`
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

## Open questions

- What are the exact rate limit thresholds (requests per minute per email/IP)?
- Does the email-outbox worker run in the same Go process or a separate worker?
- What is the Brevo template ID for OTP emails?

## Links

- Architecture: `docs/adr/0001-authentication-architecture.md`
- Session management: `docs/adr/0003-session-management.md`
- Email outbox: `docs/adr/0004-email-outbox-delivery.md`
- Rate limiting: `docs/adr/0005-rate-limiting-strategy.md`
- Glossary: `specs/glossary.md` (entities: `login_challenge`, `email_outbox`, `user`, `session`)
