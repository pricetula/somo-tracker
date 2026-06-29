# GET /api/v1/schools

**Group:** Schools (CBC)

**Auth:** Session cookie (`somo_sid`) required

**Description:** Lists all schools belonging to the authenticated user's tenant. Returns school details including location and identification codes.

**Response `200`:**

```json
{
  "schools": [
    {
      "id": "uuid",
      "name": "Moi Primary School",
      "county": "Nairobi",
      "sub_county": "Westlands",
      "ward": "Parklands",
      "knec_school_code": "123456",
      "nemis_code": "789012",
      "school_type": "PUBLIC",
      "is_active": true
    }
  ],
  "total": 3
}
```
