# GET /api/v1/streams

**Group:** Streams (CBC)

**Auth:** Session cookie (`somo_sid`) required

**Description:** Lists all streams for the active school. Streams represent class divisions (e.g., "East", "West", "North").

**Response `200`:**

```json
{
  "data": [
    {
      "id": "uuid",
      "name": "East",
      "school_id": "uuid",
      "tenant_id": "uuid"
    }
  ]
}
```
