# DELETE /api/v1/schools/:schoolId/classes/:classId/teachers/:userId

**Group:** Teacher Assignments

**Auth:** Session cookie (`somo_sid`) required

**Description:** Removes a teacher's assignment from a class. If the teacher was assigned to a specific learning area, only that assignment is removed.

**URL params:**

- `schoolId` (string, required) - The school UUID
- `classId` (string, required) - The class UUID
- `userId` (string, required) - The teacher's user UUID

**Response `200`:**

```json
{
  "code": "deleted",
  "message": "teacher removed successfully"
}
```
