# POST /api/v1/imports/staff

**Group:** Imports - Staff

**Auth:** Session cookie (`somo_sid`) required

**Description:** Starts an asynchronous staff import process. Accepts an array of staff member entries (with name, email, role) and queues them for processing. Staff members are invited via email to join the school.

**Request body:**

```json
{
  "staff_members": [
    {
      "full_name": "John Doe",
      "email": "john@example.com",
      "role": "TEACHER"
    }
  ]
}
```

**Response `202`:**

```json
{
  "import_id": "uuid",
  "status": "PENDING",
  "total": 10
}
```
