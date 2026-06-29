# GET /api/v1/academic-years

**Group:** Academic Calendar

**Auth:** Session cookie (`somo_sid`) required

**Description:** Lists all academic years for the authenticated user's tenant. Returns a paginated collection of academic years with their start/end dates and current status.

**Query params:**

- `school_id` (string, optional) - Filter by school

**Response `200`:**

```json
{
  "data": [
    {
      "id": "uuid",
      "name": "2026",
      "start_date": "2026-01-01",
      "end_date": "2026-12-31",
      "is_current": true,
      "version": 1
    }
  ]
}
```
