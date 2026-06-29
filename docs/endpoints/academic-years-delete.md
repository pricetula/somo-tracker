# DELETE /api/v1/academic-years/:id

**Group:** Academic Calendar

**Auth:** Admin role (`SCHOOL_ADMIN` or `SYSTEM_ADMIN`) required

**Description:** Deletes an academic year. Will fail with a `HAS_DEPENDENTS` conflict if the year has associated terms, classes, or assessment data.

**Response `204`:** No content on success.

**Error `409`:**

```json
{
  "code": "HAS_DEPENDENTS",
  "message": "Academic year has X dependents (terms, classes, assessments) and cannot be deleted."
}
```
