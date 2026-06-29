# GET /api/v1/imports/staff/:id/failures

**Group:** Imports - Staff

**Auth:** Session cookie (`somo_sid`) required

**Description:** Lists failed invitation records for a specific staff import. Provides details on why each invitation failed (e.g., invalid email, duplicate, service error).

**URL params:**

- `id` (string, required) - The import UUID

**Response `200`:**

```json
{
  "failures": [
    {
      "email": "invalid@",
      "reason": "invalid email address",
      "attempted_at": "2026-06-30T10:00:00Z"
    }
  ]
}
```
