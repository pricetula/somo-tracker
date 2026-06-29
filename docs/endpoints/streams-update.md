# PUT /api/v1/streams/:id

**Group:** Streams (CBC)

**Auth:** Session cookie (`somo_sid`) required

**Description:** Updates a stream's name. The new name must be unique within the school.

**Request body:**

```json
{
  "name": "string (required, max 100 chars)"
}
```

**Response `200`:** Returns the updated stream object.
