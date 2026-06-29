# DELETE /api/v1/academic-terms/:id

**Group:** Academic Calendar

**Auth:** Admin role (`SCHOOL_ADMIN` or `SYSTEM_ADMIN`) required

**Description:** Deletes an academic term. Will fail with a `HAS_DEPENDENTS` conflict if the term has associated classes, attendance records, or timetable slots.

**Response `204`:** No content on success.

**Error `409`:**

```json
{
  "code": "HAS_DEPENDENTS",
  "message": "Term has X dependents and cannot be deleted."
}
```
