# GET /api/v1/students

**Group:** Imports - Reference Data

**Auth:** Session cookie (`somo_sid`) required

**Description:** Lists existing student records for the active school. Used by the import wizard to check for duplicate students before importing new ones, and to allow parent linking.

**Response `200`:**

```json
{
  "data": [
    {
      "id": "uuid",
      "full_name": "Jane Doe",
      "admission_number": "2026/001",
      "grade_level": "Grade 1"
    }
  ]
}
```
