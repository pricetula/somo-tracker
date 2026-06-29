# POST /api/v1/classes

**Group:** Classes (CBC)

**Auth:** Session cookie (`somo_sid`) required

**Description:** Creates a new class for a specific academic year, term, grade level, and stream. Optionally assigns a list of students to the new class.

**Request body:**

```json
{
  "grade_level": "string (required)",
  "academic_year_id": "uuid (required)",
  "academic_term_id": "uuid (required)",
  "stream_id": "uuid (required)",
  "student_ids": ["uuid", "uuid"] (optional)
}
```

**Response `201`:** Returns the created class object.
