# POST /api/auth/register

**Group:** Auth

**Auth:** None

**Description:** Registers a new user account after successful OTP verification. Collects additional profile information (name, role) and creates the user record in the system.

**Request body:**

```json
{
  "token": "string (required - verification token)",
  "full_name": "string (required)",
  "role": "string (required)"
}
```

**Response `201`:**

```json
{
  "user_id": "uuid",
  "tenant_id": "uuid"
}
```
