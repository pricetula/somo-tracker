# GET /api/v1/members

**Group:** Members

**Auth:** Session cookie (`somo_sid`) required

**Description:** Lists members (staff) of the active school filtered by role. Supports pagination and search by name/email. Used for populating teacher/nurse/finance dropdowns and staff directories.

**Query params:**

- `role` (string, required) - One of: `TEACHER`, `NURSE`, `FINANCE`
- `page` (integer, optional, default: 1)
- `per_page` (integer, optional, default: 50, max: 100)
- `search` (string, optional) - Search by name or email

**Response `200`:**

```json
{
  "members": [
    {
      "id": "uuid",
      "full_name": "John Doe",
      "email": "john@example.com",
      "role": "TEACHER"
    }
  ],
  "total": 42
}
```
