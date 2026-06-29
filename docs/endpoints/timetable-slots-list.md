# GET /api/v1/schools/:schoolId/timetable/slots

**Group:** Timetable

**Auth:** Session cookie (`somo_sid`) required

**Description:** Lists timetable slots for a school, optionally filtered by class, teacher, or term. Used to display the school timetable.

**URL params:**

- `schoolId` (string, required) - The school UUID

**Query params:**

- `class_id` (string, optional) - Filter by class
- `teacher_id` (string, optional) - Filter by teacher
- `term_id` (string, required) - Filter by academic term

**Response `200`:**

```json
{
  "data": [
    {
      "id": "uuid",
      "day_of_week": 1,
      "start_time": "08:00",
      "end_time": "08:40",
      "class_id": "uuid",
      "teacher_id": "uuid",
      "subject": "Mathematics",
      "learning_area_id": "uuid"
    }
  ]
}
```
