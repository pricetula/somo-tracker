# GET /api/v1/schools/:schoolId/attendance/periods/:periodId

**Group:** Attendance

**Auth:** Session cookie (`somo_sid`) required

**Description:** Retrieves an attendance period with all its attendance logs. Used to view attendance records for a specific period.

**URL params:**

- `schoolId` (string, required) - The school UUID
- `periodId` (string, required) - The attendance period UUID

**Response `200`:**

```json
{
  "period": {
    "id": "uuid",
    "class_id": "uuid",
    "date": "2026-06-30",
    "status": "OPEN|CLOSED"
  },
  "logs": [
    {
      "student_id": "uuid",
      "student_name": "Jane Doe",
      "status": "PRESENT",
      "submitted_by": "uuid",
      "submitted_at": "2026-06-30T08:00:00Z"
    }
  ]
}
```
