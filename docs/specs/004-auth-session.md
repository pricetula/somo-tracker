# SPEC-004: Get Session

Endpoint: `GET /api/v1/auth/session`

## Metadata

| Field | Value |
|-------|-------|
| id | SPEC-004 |
| title | Get Session Info |
| status | approved |
| depends_on | specs/glossary.md, specs/002-auth-otp-verify.md |
| api_endpoint | GET /api/v1/auth/session |
| owner | backend |

## Summary

Returns session info for the frontend to make redirect decisions. Reads the session cookie,
validates it against the backend, and returns authenticated status, session type, and onboarding
flag.

No `Set-Cookie` is returned on this endpoint — cookies are only set on verify/onboard and cleared
on logout.

## Request

```
GET /api/v1/auth/session
Cookie: session=<token>
```

No request body.

## Response (success — authenticated)

```
HTTP 200 OK
Content-Type: application/json

{
  "ok": true,
  "authenticated": true,
  "session_type": "full",
  "needs_onboarding": false
}
```

## Response (success — not authenticated)

```
HTTP 200 OK
Content-Type: application/json

{
  "ok": true,
  "authenticated": false
}
```

## Response (errors)

| Status | Code | Condition |
|--------|------|-----------|
| 500 | `INTERNAL_ERROR` | Unexpected server error |

Note: A missing or invalid session cookie does **not** return an error — it returns
`authenticated: false` with HTTP 200. The frontend proxy handles redirect decisions.

## Business rules

1. **No cookie**: Return `authenticated: false`, HTTP 200.
2. **Invalid/missing session**: If token hash not found, expired, or revoked, return
   `authenticated: false`, HTTP 200 (do not leak session state).
3. **Valid session**: Return `authenticated: true`, `session_type` ("temporary" or "full"), and
   `needs_onboarding` (true if temporary).
4. **Account enumeration**: Response does not reveal whether a session token is valid beyond
   the authenticated boolean.

## Tests required

| Test type | What to verify |
|-----------|----------------|
| Unit | Returns `authenticated: false` with no cookie |
| Unit | Returns `authenticated: false` with invalid cookie |
| Integration | Returns `authenticated: true` with valid session |
| Integration | Returns `session_type` and `needs_onboarding` for temporary session |
| Contract | GET `/api/v1/auth/session` returns 200 for both authenticated and unauthenticated |

## Acceptance criteria

Given no session cookie
When the client GETs `/api/v1/auth/session`
Then the response is HTTP 200 with `authenticated: false`

Given a valid full session cookie
When the client GETs `/api/v1/auth/session`
Then the response is HTTP 200 with `authenticated: true`, `session_type: "full"`, `needs_onboarding: false`

Given a valid temporary session cookie
When the client GETs `/api/v1/auth/session`
Then the response is HTTP 200 with `authenticated: true`, `session_type: "temporary"`, `needs_onboarding: true`
