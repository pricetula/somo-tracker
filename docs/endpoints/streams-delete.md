# DELETE /api/v1/streams/:id

**Group:** Streams (CBC)

**Auth:** Session cookie (`somo_sid`) required

**Description:** Deletes a stream. Will fail with `STREAM_IN_USE` if the stream is assigned to one or more classes.

**Response `204`:** No content on success.

**Error `409`:**

```json
{
  "error": "STREAM_IN_USE",
  "message": "Stream is in use by one or more classes and cannot be deleted."
}
```
