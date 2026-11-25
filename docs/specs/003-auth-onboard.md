# SPEC-003: Onboard New User

Endpoint: `POST /api/v1/auth/onboard`

## Metadata

| Field | Value |
|-------|-------|
| id | SPEC-003 |
| title | Complete Organization Onboarding |
| status | approved |
| depends_on | specs/glossary.md, specs/002-auth-otp-verify.md |
| api_endpoint | POST /api/v1/auth/onboard |
| owner | backend |

## Summary

Completes organization onboarding for a new user with a temporary session. Sets org name, school
name, and user name, marks onboarding complete, and upgrades the session from temporary to full.

Requires a valid temporary session cookie (`__Host-session` prod / `session` dev).

## Request

```
POST /api/v1/auth/onboard
Content-Type: application/json
Cookie: session=<token>; needs_onboarding=true

{
  "organization_name": "Acme School District",
  "school_name": "Lincoln High School",
  "user_name": "Jane Doe"
}
```

| Field | Type | Required | Notes |
|-------|------|----------|-------|
| organization_name | string | Yes | Organization display name |
| school_name | string | Yes | School display name |
| user_name | string | Yes | User's display name |

## Response (success)

```
HTTP 200 OK
Content-Type: application/json
Set-Cookie: needs_onboarding=; Max-Age=-1; Path=/   (clear onboarding flag)

{
  "ok": true,
  "message": "Onboarding complete",
  "needs_onboarding": false
}
```

## Response (errors)

| Status | Code | Condition |
|--------|------|-----------|
| 401 | `UNAUTHENTICATED` | No session cookie or invalid session |
| 400 | `INVALID_REQUEST` | Session is not temporary (already onboarded) |
| 400 | `VALIDATION_FAILED` | Missing or invalid organization_name/school_name/user_name |
| 500 | `INTERNAL_ERROR` | Unexpected server error |

## Business rules

1. **Session validation**: Read `session` cookie, hash token (SHA-256), look up session.
   Session must exist, not be revoked, and not be expired.
2. **Temporary session required**: Session `session_type` must be `temporary`. If `full`, return 400.
3. **Update organization**: Set organization name and `onboarding_completed_at = NOW()`.
4. **Update user**: Set user's display name.
5. **Create school**: Create school under the organization with the provided name.
6. **Upgrade sessions**: Update all temporary sessions for the user to `session_type = 'full'`.
7. **Clear cookie**: Clear `needs_onboarding` cookie (Max-Age=-1).
8. **Account enumeration**: The response does not reveal whether the session belongs to a new or
   existing user beyond what `needs_onboarding` already indicates.

## Tests required

| Test type | What to verify |
|-----------|----------------|
| Unit | Rejects request without session cookie |
| Unit | Rejects non-temporary (full) session |
| Integration | Updates organization name and marks onboarding complete |
| Integration | Updates user name |
| Integration | Creates school under organization |
| Integration | Upgrades temporary sessions to full |
| Integration | Clears `needs_onboarding` cookie |
| Contract | POST `/api/v1/auth/onboard` returns `needs_onboarding: false` |

## Acceptance criteria

Given a valid temporary session cookie
When the client POSTs `/api/v1/auth/onboard` with org/school/user names
Then the response is HTTP 200 with `ok: true` and `needs_onboarding: false`
And the organization name is updated
And the user name is updated
And a school is created under the organization
And all temporary sessions for the user are upgraded to `full`
And the `needs_onboarding` cookie is cleared

Given no session cookie
When the client POSTs `/api/v1/auth/onboard`
Then the response is HTTP 401 `UNAUTHENTICATED`

Given a full (non-temporary) session
When the client POSTs `/api/v1/auth/onboard`
Then the response is HTTP 400 `INVALID_REQUEST`
