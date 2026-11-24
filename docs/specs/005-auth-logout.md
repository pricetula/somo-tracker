# SPEC-005: Logout

Endpoint: `POST /api/v1/auth/logout`

## Metadata

| Field | Value |
|-------|-------|
| id | SPEC-005 |
| title | Logout User |
| status | approved |
| depends_on | specs/glossary.md, specs/002-auth-otp-verify.md |
| api_endpoint | POST /api/v1/auth/logout |
| owner | backend |

## Summary

Logs out the user by clearing the session cookie and onboarding flag cookie. Does not revoke the
session token server-side (session expires naturally via `expires_at`).

## Request

```
POST /api/v1/auth/logout
Cookie: session=<token>; needs_onboarding=true
```

No request body.

## Response (success)

```
HTTP 200 OK
Content-Type: application/json
Set-Cookie: session=; Max-Age=-1; Path=/
Set-Cookie: needs_onboarding=; Max-Age=-1; Path=/

{
  "ok": true,
  "message": "logged out"
}
```

## Response (errors)

| Status | Code | Condition |
|--------|------|-----------|
| 500 | `INTERNAL_ERROR` | Unexpected server error |

Logout always returns 200 even if no session cookie was present — it is idempotent and does not
reveal session state.

## Business rules

1. **Clear session cookie**: Set `session` (or `__Host-session`) cookie with Max-Age=-1.
2. **Clear onboarding cookie**: Set `needs_onboarding` cookie with Max-Age=-1.
3. **Idempotent**: Returns 200 regardless of whether a session existed.
4. **No server-side revocation**: The session token is not revoked in the database; it expires
   naturally via `expires_at`. (Future work: add token revocation.)

## Tests required

| Test type | What to verify |
|-----------|----------------|
| Unit | Clears session cookie with Max-Age=-1 |
| Unit | Clears onboarding cookie with Max-Age=-1 |
| Unit | Returns 200 even with no session cookie |
| Contract | POST `/api/v1/auth/logout` returns `{ ok: true, message: "logged out" }` |

## Acceptance criteria

Given a session cookie
When the client POSTs `/api/v1/auth/logout`
Then the response is HTTP 200 with `{ ok: true, message: "logged out" }`
And the session cookie is cleared (Max-Age=-1)
And the `needs_onboarding` cookie is cleared

Given no session cookie
When the client POSTs `/api/v1/auth/logout`
Then the response is HTTP 200 (idempotent, no error)
