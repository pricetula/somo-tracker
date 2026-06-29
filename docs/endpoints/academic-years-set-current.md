# POST /api/v1/academic-years/:id/set-current

**Group:** Academic Calendar

**Auth:** Admin role (`SCHOOL_ADMIN` or `SYSTEM_ADMIN`) required

**Description:** Designates the specified academic year as the current/active year. All other academic years for the same school will be marked as non-current.

**Response `200`:**

```json
{
  "message": "Academic year set as current."
}
```
