# GET /api/v1/classes

**Group:** Classes (CBC)

**Auth:** Session cookie (`somo_sid`) required

**Description:** Lists classes for the active school, filtered by academic year and term. Supports pagination and optional filtering by grade level and stream. Used to view the class roster for a given term.

**Query params:**

- `academic_year_id` (string, required)
- `academic_term_id` (string, required)
- `school_id` (string, optional)
- `grade_level` (string, optional) - Filter by grade (e.g., "Grade 1", "Grade 2")
- `stream_id` (string, optional) - Filter by stream
- `page` (integer, optional, default: 1)
- `limit` (integer, optional, default: 50, max: 200)

**Response `200`:**

```json
{
  "data": [
    {
      "id": "uuid",
      "grade_level": "Grade 1",
      "stream_id": "uuid",
      "stream_name": "East",
      "academic_year_id": "uuid",
      "academic_term_id": "uuid",
      "student_count": 35,
      "students": ["uuid", "uuid"]
    }
  ],
  "total": 8,
  "page": 1,
  "limit": 50
}
```
