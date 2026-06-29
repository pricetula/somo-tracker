# GET /api/v1/parents

**Group:** Imports - Reference Data

**Auth:** Session cookie (`somo_sid`) required

**Description:** Lists existing parents/guardians in the system. Used by the import wizard to allow users to select existing parents when importing students, avoiding duplicate parent records.

**Response `200`:**

```json
{
  "data": [
    {
      "id": "uuid",
      "full_name": "John Doe",
      "email": "john@example.com",
      "phone": "+254700000000"
    }
  ]
}
```
