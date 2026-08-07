# Attendance Endpoints

This document outlines the API endpoints related to attendance management, including sessions, records, and various summaries.

## Sessions

### 1. Create Attendance Session

*   **URL:** `/api/v1/attendance/sessions`
*   **Method:** `POST`
*   **Description:** Creates a new attendance session.
*   **Authentication:** Required
*   **Request Body (CreateSessionPayload):**
    *   `class_id` (string): The ID of the class for the session.
    *   `timetable_slot_id` (string): The ID of the timetable slot for the session.
    *   `date` (string, YYYY-MM-DD): The date of the session.
    *   `start_time` (string, HH:MM): The start time of the session.
    *   `end_time` (string, HH:MM): The end time of the session.
    *   `status` (string): The status of the session (e.g., "scheduled", "completed").
*   **Response (Session):**
    *   `id` (string): Unique identifier for the session.
    *   `class_id` (string): ID of the associated class.
    *   `timetable_slot_id` (string): ID of the associated timetable slot.
    *   `date` (string, YYYY-MM-DD): Date of the session.
    *   `start_time` (string, HH:MM): Start time of the session.
    *   `end_time` (string, HH:MM): End time of the session.
    *   `status` (string): Current status of the session.
    *   `created_at` (string, ISO 8601): Timestamp of creation.
    *   `updated_at` (string, ISO 8601): Timestamp of last update.

### 2. List Attendance Sessions

*   **URL:** `/api/v1/attendance/sessions`
*   **Method:** `GET`
*   **Description:** Retrieves a list of attendance sessions based on provided filters.
*   **Authentication:** Required
*   **Query Parameters (SessionFilter):**
    *   `timetable_slot_id` (string, optional): Filter by timetable slot ID.
    *   `date` (string, YYYY-MM-DD, optional): Filter by date.
    *   `status` (string, optional): Filter by session status.
    *   `class_id` (string, optional): Filter by class ID.
*   **Response (List of Session):** An array of `Session` objects.

### 3. Get Sessions for Class and Date

*   **URL:** `/api/v1/attendance/sessions/class/:class_id/date/:date`
*   **Method:** `GET`
*   **Description:** Retrieves all attendance sessions for a specific class on a given date.
*   **Authentication:** Required
*   **Path Parameters:**
    *   `class_id` (string): The ID of the class.
    *   `date` (string, YYYY-MM-DD): The date.
*   **Response (List of Session):** An array of `Session` objects.

### 4. Get Attendance Session by ID

*   **URL:** `/api/v1/attendance/sessions/:id`
*   **Method:** `GET`
*   **Description:** Retrieves a single attendance session by its ID.
*   **Authentication:** Required
*   **Path Parameters:**
    *   `id` (string): The unique identifier of the session.
*   **Response (Session):** A single `Session` object.

### 5. Update Attendance Session

*   **URL:** `/api/v1/attendance/sessions/:id`
*   **Method:** `PUT`
*   **Description:** Updates an existing attendance session.
*   **Authentication:** Required
*   **Path Parameters:**
    *   `id` (string): The unique identifier of the session to update.
*   **Request Body (UpdateSessionPayload):**
    *   `class_id` (string, optional): New class ID.
    *   `timetable_slot_id` (string, optional): New timetable slot ID.
    *   `date` (string, YYYY-MM-DD, optional): New date.
    *   `start_time` (string, HH:MM, optional): New start time.
    *   `end_time` (string, HH:MM, optional): New end time.
    *   `status` (string, optional): New status.
*   **Response (Session):** The updated `Session` object.

## Attendance Records

### 6. Batch Mark Attendance Records

*   **URL:** `/api/v1/attendance/records/batch`
*   **Method:** `POST`
*   **Description:** Marks multiple attendance records in a single request.
*   **Authentication:** Required
*   **Query Parameters:**
    *   `term_id` (string, optional): The academic term ID for these records.
*   **Request Body (BatchMarkPayload):**
    *   `session_id` (string): The ID of the attendance session.
    *   `records` (array of objects):
        *   `student_id` (string): The ID of the student.
        *   `status` (string): Attendance status (e.g., "present", "absent", "late").
        *   `remarks` (string, optional): Any additional remarks.
*   **Response (BatchMarkResult):**
    *   `created_count` (integer): Number of records created.
    *   `updated_count` (integer): Number of records updated.

### 7. List Attendance Records by Slot and Date

*   **URL:** `/api/v1/attendance/records/slot`
*   **Method:** `GET`
*   **Description:** Retrieves attendance records for a specific timetable slot and date.
*   **Authentication:** Required
*   **Query Parameters:**
    *   `timetable_slot_id` (string): The ID of the timetable slot.
    *   `date` (string, YYYY-MM-DD): The date.
*   **Response (List of Record):** An array of attendance `Record` objects.

### 8. List Attendance Records by Student and Term

*   **URL:** `/api/v1/attendance/records/student/:student_id`
*   **Method:** `GET`
*   **Description:** Retrieves attendance records for a specific student within a given term.
*   **Authentication:** Required
*   **Path Parameters:**
    *   `student_id` (string): The ID of the student.
*   **Query Parameters:**
    *   `term_id` (string, optional): The academic term ID.
*   **Response (List of Record):** An array of attendance `Record` objects.

### 9. List Attendance Records by Class and Date

*   **URL:** `/api/v1/attendance/records/class/:class_id/date/:date`
*   **Method:** `GET`
*   **Description:** Retrieves attendance records for a specific class on a given date.
*   **Authentication:** Required
*   **Path Parameters:**
    *   `class_id` (string): The ID of the class.
    *   `date` (string, YYYY-MM-DD): The date.
*   **Query Parameters:**
    *   `term_id` (string, optional): The academic term ID.
*   **Response (List of Record):** An array of attendance `Record` objects.

### 10. List Attendance Records

*   **URL:** `/api/v1/attendance/records`
*   **Method:** `GET`
*   **Description:** Retrieves a list of attendance records based on provided filters.
*   **Authentication:** Required
*   **Query Parameters (RecordFilter):
    *   `timetable_slot_id` (string, optional): Filter by timetable slot ID.
    *   `date` (string, YYYY-MM-DD, optional): Filter by date.
    *   `student_id` (string, optional): Filter by student ID.
    *   `class_id` (string, optional): Filter by class ID.
    *   `academic_term_id` (string, optional): Filter by academic term ID.
    *   `status` (string, optional): Filter by record status.
*   **Response (List of Record):** An array of attendance `Record` objects.

### 11. Update Attendance Record

*   **URL:** `/api/v1/attendance/records/:id`
*   **Method:** `PUT`
*   **Description:** Updates an existing attendance record.
*   **Authentication:** Required
*   **Path Parameters:**
    *   `id` (string): The unique identifier of the record to update.
*   **Request Body (UpdateRecordPayload):
    *   `session_id` (string, optional): New session ID.
    *   `student_id` (string, optional): New student ID.
    *   `status` (string, optional): New attendance status.
    *   `remarks` (string, optional): New remarks.
*   **Response (Record):** The updated `Record` object.

## Attendance Summaries

### 12. Get Student Term Summary

*   **URL:** `/api/v1/attendance/summaries/student/:student_id`
*   **Method:** `GET`
*   **Description:** Retrieves an attendance summary for a specific student within a given term.
*   **Authentication:** Required
*   **Path Parameters:**
    *   `student_id` (string): The ID of the student.
*   **Query Parameters:**
    *   `term_id` (string, optional): The academic term ID.
*   **Response (StudentTermSummary):
    *   `student_id` (string)
    *   `term_id` (string)
    *   `total_sessions` (integer)
    *   `present_count` (integer)
    *   `absent_count` (integer)
    *   `late_count` (integer)
    *   `present_percentage` (float)

### 13. Get Class Term Summary

*   **URL:** `/api/v1/attendance/summaries/class/:class_id`
*   **Method:** `GET`
*   **Description:** Retrieves an attendance summary for a specific class within a given term.
*   **Authentication:** Required
*   **Path Parameters:**
    *   `class_id` (string): The ID of the class.
*   **Query Parameters:**
    *   `term_id` (string, optional): The academic term ID.
*   **Response (ClassTermSummary):
    *   `class_id` (string)
    *   `term_id` (string)
    *   `total_sessions` (integer)
    *   `average_present_percentage` (float)
    *   `student_summaries` (array of StudentTermSummary)

### 14. Refresh Attendance Summaries

*   **URL:** `/api/v1/attendance/summaries/refresh`
*   **Method:** `POST`
*   **Description:** Triggers a refresh of all attendance summaries for a given term.
*   **Authentication:** Required
*   **Request Body:
    *   `term_id` (string): The academic term ID for which to refresh summaries.
*   **Response (RefreshSummaryResult):
    *   `status` (string): "success" or "failed"
    *   `message` (string): A descriptive message.

## Class Daily Attendance Summaries

### 15. Get Class Daily Summary

*   **URL:** `/api/v1/attendance/daily/class/:class_id/date/:date`
*   **Method:** `GET`
*   **Description:** Retrieves the daily attendance summary for a specific class on a given date.
*   **Authentication:** Required
*   **Path Parameters:**
    *   `class_id` (string): The ID of the class.
    *   `date` (string, YYYY-MM-DD): The date.
*   **Response (ClassDailySummary):
    *   `class_id` (string)
    *   `date` (string, YYYY-MM-DD)
    *   `total_students` (integer)
    *   `present_count` (integer)
    *   `absent_count` (integer)
    *   `late_count` (integer)
    *   `present_percentage` (float)
    *   `sessions` (array of Session, simplified details)
    *   `records` (array of Record, simplified details for students)

### 16. Refresh Class Daily Summary

*   **URL:** `/api/v1/attendance/daily/class/:class_id/date/:date/refresh`
*   **Method:** `POST`
*   **Description:** Triggers a refresh of the daily attendance summary for a specific class and date.
*   **Authentication:** Required
*   **Path Parameters:**
    *   `class_id` (string): The ID of the class.
    *   `date` (string, YYYY-MM-DD): The date.
*   **Response (RefreshSummaryResult):
    *   `status` (string): "success" or "failed"
    *   `message` (string): A descriptive message.

### 17. List Class Daily Summaries

*   **URL:** `/api/v1/attendance/daily/class/:class_id`
*   **Method:** `GET`
*   **Description:** Retrieves a list of daily attendance summaries for a specific class within a date range.
*   **Authentication:** Required
*   **Path Parameters:**
    *   `class_id` (string): The ID of the class.
*   **Query Parameters:**
    *   `start_date` (string, YYYY-MM-DD): The start date of the range.
    *   `end_date` (string, YYYY-MM-DD): The end date of the range.
*   **Response (List of ClassDailySummary):** An array of `ClassDailySummary` objects.

## Class Learning Area Term Summaries

### 18. List Class Learning Area Term Summaries

*   **URL:** `/api/v1/attendance/class-learning-area/class/:class_id/term/:term_id`
*   **Method:** `GET`
*   **Description:** Retrieves learning area term summaries for a class.
*   **Authentication:** Required
*   **Path Parameters:**
    *   `class_id` (string): The ID of the class.
    *   `term_id` (string): The ID of the academic term.
*   **Query Parameters:**
    *   `learning_area_id` (string, optional): Filter by a specific learning area ID.
*   **Response (List of ClassLearningAreaTermSummary):
    *   `class_id` (string)
    *   `term_id` (string)
    *   `learning_area_id` (string)
    *   `total_sessions` (integer)
    *   `average_present_percentage` (float)

### 19. Get Class Learning Area Term Summary

*   **URL:** `/api/v1/attendance/class-learning-area/class/:class_id/learning-area/:learning_area_id/term/:term_id`
*   **Method:** `GET`
*   **Description:** Retrieves a specific learning area term summary for a class.
*   **Authentication:** Required
*   **Path Parameters:**
    *   `class_id` (string): The ID of the class.
    *   `learning_area_id` (string): The ID of the learning area.
    *   `term_id` (string): The ID of the academic term.
*   **Response (ClassLearningAreaTermSummary):** A single `ClassLearningAreaTermSummary` object.

### 20. Refresh Class Learning Area Term Summary

*   **URL:** `/api/v1/attendance/class-learning-area/class/:class_id/term/:term_id/refresh`
*   **Method:** `POST`
*   **Description:** Triggers a refresh of the learning area term summary for a class.
*   **Authentication:** Required
*   **Path Parameters:**
    *   `class_id` (string): The ID of the class.
    *   `term_id` (string): The ID of the academic term.
*   **Response (RefreshSummaryResult):
    *   `status` (string): "success" or "failed"
    *   `message` (string): A descriptive message.

## Class Term Attendance Summaries

### 21. Get Class Term Attendance Summary

*   **URL:** `/api/v1/attendance/class-term/class/:class_id/term/:term_id`
*   **Method:** `GET`
*   **Description:** Retrieves the overall term attendance summary for a specific class.
*   **Authentication:** Required
*   **Path Parameters:**
    *   `class_id` (string): The ID of the class.
    *   `term_id` (string): The ID of the academic term.
*   **Response (ClassTermAttendanceSummary):
    *   `class_id` (string)
    *   `term_id` (string)
    *   `total_students` (integer)
    *   `average_daily_present_percentage` (float)
    *   `summary_by_date` (map[string]ClassDailySummary, where key is date YYYY-MM-DD)

### 22. List Class Term Attendance Summaries

*   **URL:** `/api/v1/attendance/class-term/term/:term_id`
*   **Method:** `GET`
*   **Description:** Retrieves overall term attendance summaries for multiple classes within a given term.
*   **Authentication:** Required
*   **Path Parameters:
    *   `term_id` (string): The ID of the academic term.
*   **Query Parameters:
    *   `class_id` (string, optional): Filter by a specific class ID.
*   **Response (List of ClassTermAttendanceSummary):** An array of `ClassTermAttendanceSummary` objects.

### 23. Refresh Class Term Attendance Summary

*   **URL:** `/api/v1/attendance/class-term/class/:class_id/term/:term_id/refresh`
*   **Method:** `POST`
*   **Description:** Triggers a refresh of the overall term attendance summary for a specific class.
*   **Authentication:** Required
*   **Path Parameters:**
    *   `class_id` (string): The ID of the class.
    *   `term_id` (string): The ID of the academic term.
*   **Response (RefreshSummaryResult):
    *   `status` (string): "success" or "failed"
    *   `message` (string): A descriptive message.

## Calendar Status

### 24. Get Calendar Status

*   **URL:** `/api/v1/attendance/calendar/status`
*   **Method:** `GET`
*   **Description:** Retrieves a monthly overview of attendance statuses for a given date range.
*   **Authentication:** Required
*   **Query Parameters:
    *   `start_date` (string, YYYY-MM-DD, required): The start date of the calendar view (max 62 days range).
    *   `end_date` (string, YYYY-MM-DD, required): The end date of the calendar view (max 62 days range).
*   **Response (CalendarStatus):
    *   `date` (string, YYYY-MM-DD): The date.
    *   `has_sessions` (boolean): True if there are any sessions on this date.
    *   `is_holiday` (boolean): True if the date is a holiday.
    *   `is_weekend` (boolean): True if the date is a weekend.
