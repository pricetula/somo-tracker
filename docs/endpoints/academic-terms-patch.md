# PATCH /api/v1/academic-terms/:id

**Group:** Academic Calendar

**Auth:** Admin role (`SCHOOL_ADMIN` or `SYSTEM_ADMIN`) required

**Description:** Updates an academic term's metadata (name, dates). Uses optimistic locking. `term_number` and `is_current` cannot be modified via PATCH.

**Request body:**

```json
{
  "name": "string (optional)",
  "start_date": "2026-01-15 (optional)",
  "end_date": "2026-04-15 (optional)",
  "version": "integer (required - optimistic lock)"
}
```

**Response `200`:** Returns the updated term object with warnings if `term_number` or `is_current` were stripped.

**Error `422`:**

```json
{
  "code": "TERM_OUT_OF_YEAR_BOUNDS",
  "message": "Updated dates must fall within the academic year's date range."
}
```

```json
{
  "code": "TERM_DATE_OVERLAP",
  "message": "Updated dates overlap with existing term: Term 2",
  "conflicting_term": "Term 2"
}
```

**Error `409`:**

```json
{
  "code": "conflict",
  "message": "Resource was modified by another request. Fetch the latest version and retry."
}
```
