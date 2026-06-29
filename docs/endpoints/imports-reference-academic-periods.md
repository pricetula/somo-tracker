# GET /api/v1/academic/periods

**Group:** Imports - Reference Data

**Auth:** Session cookie (`somo_sid`) required

**Description:** Lists academic periods (terms) for use in the import wizard. Used when importing students to determine which term they should be enrolled in.

**Response `200`:**

```json
{
  "data": [
    {
      "id": "uuid",
      "name": "Term 1",
      "academic_year_id": "uuid"
    }
  ]
}
```
