# POST /api/v1/imports/students

**Group:** Imports - Students

**Auth:** Session cookie (`somo_sid`) required

**Description:** Starts an asynchronous student import process. Accepts student enrolment data (name, grade, parent info) and queues it for processing. Students are created and assigned to appropriate classes.

**Request body:**

```json
{
  "students": [
    {
      "full_name": "Jane Doe",
      "admission_number": "2026/001",
      "grade_level": "Grade 1",
      "stream_id": "uuid",
      "parent_name": "John Doe",
      "parent_email": "john@example.com",
      "parent_phone": "+254700000000"
    }
  ]
}
```

**Response `202`:**

```json
{
  "import_id": "uuid",
  "status": "PENDING",
  "total": 50
}
```
