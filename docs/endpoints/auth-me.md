# GET /api/auth/me

**Group:** Auth

**Auth:** Session cookie (`somo_sid`) required

**Description:** Returns the currently authenticated user's profile information, including tenant affiliation, role, and associated school.

**Response `200`:**

```json
{
  "user_id": "uuid",
  "tenant_id": "uuid",
  "role": "SYSTEM_ADMIN|SCHOOL_ADMIN|TEACHER|NURSE|FINANCE",
  "school_id": "uuid",
  "school_name": "string",
  "full_name": "string",
  "email": "string"
}
```
