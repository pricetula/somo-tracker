# GET /api/v1/active-school

**Group:** Active School

**Auth:** Session cookie (`somo_sid`) required

**Description:** Returns the currently active school ID for the authenticated user. Used to determine which school the user is currently working in.

**Response `200`:**

```json
{
  "school_id": "uuid"
}
```
