# POST /api/v1/schools/:schoolId/attendance

**Group:** Attendance

**Auth:** Session cookie (`somo_sid`) required

**Description:** Records attendance for a class/period. Opens the attendance period if not already open and submits the attendance records for listed students. Used for daily attendance taking.

**URL params:**

- `schoolId` (string, required) - The school UUID

**Request body:**

```json
{
  "class_id": "uuid (required)",
  "period_id": "uuid (required - attendance period)",
  "date": "2026-06-30 (required)",
  "records": [
    {
      "student_id": "uuid",
      "status": "PRESENT|ABSENT|LATE|SICK|PERMISSION"
    }
  ]
}
```

**Response `200`:**

```json
{
  "code": "success",
  "message": "attendance recorded successfully"
}
```
