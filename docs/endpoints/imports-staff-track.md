# GET /api/v1/imports/staff/track/:id

**Group:** Imports - Staff

**Auth:** Session cookie (`somo_sid`) required

**Description:** Polls the progress of a staff import by its ID. Returns the current status, number of processed items, and any failures.

**URL params:**

- `id` (string, required) - The import UUID

**Response `200`:**

```json
{
  "import_id": "uuid",
  "status": "PROCESSING|COMPLETED|FAILED",
  "total": 10,
  "processed": 7,
  "failed": 3
}
```
