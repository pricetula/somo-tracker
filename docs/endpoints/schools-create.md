# POST /api/v1/schools

**Group:** Schools (CBC)

**Auth:** Session cookie (`somo_sid`) required

**Description:** Creates a new school under the authenticated user's tenant. Returns the new school's UUID. Schools are the primary organisational unit below tenant level.

**Request body:**

```json
{
  "name": "string (required)"
}
```

**Response `201`:**

```json
{
  "id": "uuid"
}
```
