# SPEC-001: Request Login OTP

Endpoint: `POST /api/v1/auth/otp/request`

## Metadata

| Field | Value |
|-------|-------|
| id | SPEC-001 |
| title | Request Login OTP |
| status | draft |
| depends_on | specs/glossary.md, auth-architecture-v2.md |
| api_endpoint | POST /api/v1/auth/otp/request |
| owner | backend |

## Summary

Accepts an email address, validates it, creates a login challenge and an
encrypted email-outbox record in one transaction, and returns a generic
success response that does not reveal whether the account exists.

The frontend calls the Next.js BFF API route, which proxies to the Go
backend. Go sets no cookie on this endpoint (no session exists yet).

## Request

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

## Response (errors)

| Status | Code | Condition |
|--------|------|-----------|
| 400 | `INVALID_REQUEST` | Malformed JSON or missing email |
| 400 | `INVALID_EMAIL` | Email fails syntax validation |
| 429 | `RATE_LIMITED` | Per-email, per-IP, or global limit exceeded |
| 400 | `TURNSTILE_REQUIRED` | `TURNSTILE_ENABLED=true` but token missing |
| 400 | `TURNSTILE_INVALID` | Turnstile server-side verification failed |
| 500 | `INTERNAL_ERROR` | Unexpected server error |

## Business rules

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
| Contract | POST `/api/v1/auth/otp/request` returns 200 with generic body |
| Contract | POST `/api/v1/auth/otp/request` returns 400 on missing email |

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

## Open questions

- What are the exact rate limit thresholds (requests per minute per email/IP)?
- Does the email-outbox worker run in the same Go process or a separate worker?
- What is the Brevo template ID for OTP emails?

## Links

- Architecture: `auth-architecture-v2.md` §4.2, §4.3, §11, §12.2
- Glossary: `specs/glossary.md` (entities: `login_challenge`, `email_outbox`, `user`)
