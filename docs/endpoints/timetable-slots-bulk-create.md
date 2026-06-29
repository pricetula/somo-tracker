# POST /api/v1/schools/:schoolId/timetable/slots/bulk

**Group:** Timetable

**Auth:** Session cookie (`somo_sid`) required

**Description:** Creates or updates timetable slots in bulk for a school. Accepts an array of slot definitions including day of week, time range, class, teacher, and subject/learning area.

**URL params:**

- `schoolId` (string, required) - The school UUID

**Request body:**

```json
{
  "slots": [
    {
      "day_of_week": 1,
      "start_time": "08:00",
      "end_time": "08:40",
      "class_id": "uuid",
      "teacher_id": "uuid",
      "subject": "Mathematics"
    }
  ]
}
```

**Response `201`:**

```json
{
  "code": "created",
  "message": "timetable slots saved successfully"
}
```
