# PATCH /api/v1/academic-years/:id

**Group:** Academic Calendar

**Auth:** Admin role (`SCHOOL_ADMIN` or `SYSTEM_ADMIN`) required

**Description:** Updates an academic year's metadata (name, dates). Uses optimistic locking via the `version` field. Returns the updated academic year.

**Request body:**

```json
{
  "name": "string (optional)",
  "start_date": "2026-01-01 (optional)",
  "end_date": "2026-12-31 (optional)",
  "version": "integer (required - optimistic lock)"
}
```

**Response `200`:**

```json
{
  "id": "uuid",
  "name": "string",
  "start_date": "2026-01-01",
  "end_date": "2026-12-31",
  "is_current": true,
  "version": 2
}
```
