# POST /api/auth/discover

**Group:** Auth

**Auth:** None

**Description:** Phase 1 of the authentication flow. Accepts an email or phone number, determines whether the user exists, and sends a one-time passcode (OTP) or magic link. Used for both sign-in and sign-up discovery.

**Request body:**

```json
{
  "email": "string (optional - email or phone required)",
  "phone": "string (optional - email or phone required)"
}
```

**Response `200`:**

```json
{
  "discovery_type": "signin|signup",
  "delivery_method": "email|sms"
}
```
