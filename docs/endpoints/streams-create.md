# POST /api/v1/streams

**Group:** Streams (CBC)

**Auth:** Session cookie (`somo_sid`) required

**Description:** Creates a new stream for the active school. Stream names must be unique within a school and cannot exceed 100 characters.

**Request body:**

```json
{
  "name": "string (required, max 100 chars)"
}
```

**Response `201`:**

```json
{
  "id": "uuid",
  "name": "East",
  "school_id": "uuid",
  "tenant_id": "uuid"
}
```
