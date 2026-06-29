# POST /api/auth/verify

**Group:** Auth

**Auth:** None

**Description:** Phase 2 of the authentication flow. Verifies the one-time passcode (OTP) submitted by the user. On success, establishes a session and returns a session reference.

**Request body:**

```json
{
  "token": "string (required - the OTP code)"
}
```

**Response `200`:**

```json
{
  "session_ref": "string"
}
```
