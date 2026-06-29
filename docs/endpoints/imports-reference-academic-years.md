# GET /api/v1/academic/years

**Group:** Imports - Reference Data

**Auth:** Session cookie (`somo_sid`) required

**Description:** Lists academic years for use in the import wizard. Used when importing students to associate them with the correct academic year context.

**Response `200`:**

```json
{
  "data": [
    {
      "id": "uuid",
      "name": "2026",
      "is_current": true
    }
  ]
}
```
