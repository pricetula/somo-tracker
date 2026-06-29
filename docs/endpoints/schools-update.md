# PUT /api/v1/schools/:id

**Group:** Schools (CBC)

**Auth:** Session cookie (`somo_sid`) required

**Description:** Updates a school's metadata including name, location fields, identification codes, and active status. Only fields provided in the request body are updated.

**Request body:**

```json
{
  "name": "string (optional)",
  "county": "string (optional)",
  "sub_county": "string (optional)",
  "ward": "string (optional)",
  "knec_school_code": "string (optional)",
  "nemis_code": "string (optional)",
  "school_type": "PUBLIC|PRIVATE (optional)",
  "is_active": true (optional)
}
```

**Response `200`:** No body on success.
