# PUT /api/v1/classes/:id

**Group:** Classes (CBC)

**Auth:** Session cookie (`somo_sid`) required

**Description:** Updates a class's grade level, stream, term, and student roster. Replaces the full student list with the provided IDs. Will fail with `CLASS_LOCKED` if the class has assessment records.

**Request body:**

```json
{
  "grade_level": "string (required)",
  "stream_id": "uuid (required)",
  "academic_term_id": "uuid (required)",
  "student_ids": ["uuid", "uuid"] (optional)
}
```

**Response `200`:** Returns the updated class object.

**Error `409`:**

```json
{
  "error": "CLASS_LOCKED",
  "message": "This class has assessment records and cannot be modified."
}
```
