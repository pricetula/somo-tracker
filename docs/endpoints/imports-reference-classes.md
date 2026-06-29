# GET /api/v1/classes

**Group:** Imports - Reference Data

**Auth:** Session cookie (`somo_sid`) required

**Description:** Lists existing classes for the active school. Used by the import wizard to allow users to select target classes when importing students. Note: This is a different endpoint from `GET /api/v1/classes` in the Classes group (different path registration, same path). Returns data from the imports handler for wizard integration.

**Response `200`:**

```json
{
  "data": [
    {
      "id": "uuid",
      "grade_level": "Grade 1",
      "stream_name": "East"
    }
  ]
}
```
