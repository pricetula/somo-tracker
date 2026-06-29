# POST /api/v1/academic-terms

**Group:** Academic Calendar

**Auth:** Admin role (`SCHOOL_ADMIN` or `SYSTEM_ADMIN`) required

**Description:** Creates a new academic term within an existing academic year. Validates that the term dates fall within the year's bounds and do not overlap with existing terms. Term numbers must be unique within the year.

**Request body:**

```json
{
  "name": "string (required)",
  "term_number": "integer (required)",
  "start_date": "2026-01-15 (required)",
  "end_date": "2026-04-15 (required)",
  "academic_year_id": "uuid (required)"
}
```

**Response `201`:** Returns the created term object.

**Error `422`:**

```json
{
  "code": "TERM_OUT_OF_YEAR_BOUNDS",
  "message": "Term dates must fall within the academic year's date range."
}
```

```json
{
  "code": "TERM_DATE_OVERLAP",
  "message": "Term dates overlap with existing term: Term 2",
  "conflicting_term": "Term 2"
}
```

**Error `409`:**

```json
{
  "code": "TERM_NUMBER_EXISTS",
  "message": "Term number 1 already exists for this academic year."
}
```
