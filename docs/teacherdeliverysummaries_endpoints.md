# Teacher Delivery Summaries Endpoints

This document outlines the API endpoints related to teacher delivery summaries.

## Teacher Delivery Summaries

### 1. Refresh Teacher Delivery Summaries

*   **URL:** `/api/v1/teacher-delivery-summaries/refresh`
*   **Method:** `POST`
*   **Description:** Triggers a refresh computation of teacher delivery summaries for a given academic term.
*   **Authentication:** Required (SCHOOL_ADMIN or SYSTEM_ADMIN roles)
*   **Request Body (RefreshRequest):
    *   `academic_term_id` (string, required): The ID of the academic term for which to refresh summaries.
*   **Response (RefreshResponse):
    *   `message` (string): "Teacher delivery summaries refreshed"
    *   `term_id` (string): The academic term ID that was refreshed.

### 2. List Teacher Delivery Summaries by Term

*   **URL:** `/api/v1/teacher-delivery-summaries`
*   **Method:** `GET`
*   **Description:** Retrieves a list of all teacher delivery summaries for a specific academic term.
*   **Authentication:** Required
*   **Query Parameters:**
    *   `term_id` (string, required): The ID of the academic term.
*   **Response (List of TeacherDeliverySummary):** An array of `TeacherDeliverySummary` objects.
    *   `teacher_id` (string)
    *   `academic_term_id` (string)
    *   `total_classes_taught` (integer)
    *   `total_learning_areas_taught` (integer)
    *   `average_class_attendance` (float)
    *   `average_assessment_submission` (float)
    *   `last_refreshed_at` (string, ISO 8601)

### 3. List Teacher Delivery Summaries by Teacher

*   **URL:** `/api/v1/teacher-delivery-summaries/teacher/:user_id`
*   **Method:** `GET`
*   **Description:** Retrieves a list of teacher delivery summaries for a specific teacher across different academic terms.
*   **Authentication:** Required
*   **Path Parameters:**
    *   `user_id` (string, required): The ID of the teacher (user).
*   **Query Parameters:**
    *   `term_id` (string, required): The ID of the academic term.
*   **Response (List of TeacherDeliverySummary):** An array of `TeacherDeliverySummary` objects (similar structure to List by Term, but focused on a single teacher potentially across multiple terms if `term_id` was optional, but here it's required for a single term).

### 4. Get Single Teacher Delivery Summary

*   **URL:** `/api/v1/teacher-delivery-summaries/:user_id`
*   **Method:** `GET`
*   **Description:** Retrieves a single teacher delivery summary for a specific teacher and academic term.
*   **Authentication:** Required
*   **Path Parameters:**
    *   `user_id` (string, required): The ID of the teacher (user).
*   **Query Parameters:**
    *   `term_id` (string, required): The ID of the academic term.
*   **Response (TeacherDeliverySummary):** A single `TeacherDeliverySummary` object.
