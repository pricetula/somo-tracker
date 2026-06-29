# GET /api/v1/invitations

**Group:** Invitations

**Auth:** Session cookie (`somo_sid`) required

**Description:** Lists pending and historical invitations for the active school. Supports filtering by email, status, role, and expiry. Used by school admins to track sent invitations and manage onboarding.

**Query params:**

- `page` (integer, optional, default: 1)
- `per_page` (integer, optional, default: 50, max: 100)
- `search` (string, optional) - Search by name or email
- `email` (string, optional) - Filter by exact email
- `status` (string, optional) - Filter by status (`PENDING`, `ACCEPTED`, `EXPIRED`, `REVOKED`)
- `role` (string, optional) - Filter by role
- `expired` (boolean, optional, default: false) - Show only expired invitations

**Response `200`:**

```json
{
  "invitations": [
    {
      "id": "uuid",
      "email": "teacher@example.com",
      "role": "TEACHER",
      "status": "PENDING",
      "expires_at": "2026-07-30T00:00:00Z",
      "created_at": "2026-06-30T00:00:00Z"
    }
  ],
  "total": 15
}
```
