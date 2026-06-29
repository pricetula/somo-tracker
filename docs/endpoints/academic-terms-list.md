# GET /api/v1/academic-terms

**Group:** Academic Calendar

**Auth:** Session cookie (`somo_sid`) required

**Description:** Lists all academic terms for the authenticated user's tenant. Optionally filtered by academic year.

**Query params:**

- `academic_year_id` (string, optional) - Filter by academic year

**Response `200`:**

```json
{
  "data": [
    {
      "id": "uuid",
      "name": "Term 1",
      "term_number": 1,
      "start_date": "2026-01-15",
      "end_date": "2026-04-15",
      "is_current": false,
      "academic_year_id": "uuid",
      "version": 1
    }
  ]
}
```
