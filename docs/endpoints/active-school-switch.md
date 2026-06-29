# PUT /api/v1/active-school

**Group:** Active School

**Auth:** Session cookie (`somo_sid`) required

**Description:** Sets or switches the active school for the authenticated user. This is used when a user belongs to multiple schools and needs to change their current working context.

**Request body:**

```json
{
  "school_id": "uuid (required)"
}
```

**Response `200`:**

```json
{
  "message": "active school updated"
}
```
