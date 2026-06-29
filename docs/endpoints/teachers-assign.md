# POST /api/v1/schools/:schoolId/classes/:classId/teachers

**Group:** Teacher Assignments

**Auth:** Session cookie (`somo_sid`) required

**Description:** Assigns a teacher to a class, optionally for a specific learning area. Used to manage class-teacher mappings for the school timetable.

**URL params:**

- `schoolId` (string, required) - The school UUID
- `classId` (string, required) - The class UUID

**Request body:**

```json
{
  "user_id": "uuid (required)",
  "learning_area_id": "uuid (optional)",
  "teacher_role": "string (required - e.g., CLASS_TEACHER, SUBJECT_TEACHER)"
}
```

**Response `201`:**

```json
{
  "code": "created",
  "message": "teacher assigned successfully"
}
```
