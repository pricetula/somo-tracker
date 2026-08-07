# Parents Endpoints

This document outlines the API endpoints related to parent management, including creation, listing, updating, deleting, and linking students.

## Parent Management

### 1. Create Parent

*   **URL:** `/api/v1/parents`
*   **Method:** `POST`
*   **Description:** Creates a new parent profile.
*   **Authentication:** Required
*   **Request Body (CreateParentPayload):
    *   `email` (string, required): The parent's email address.
    *   `full_name` (string, required): The parent's full name.
    *   `phone_number` (string, required): The parent's phone number.
    *   `address` (string, optional): The parent's address.
*   **Response (CreateParentResponse):
    *   `id` (string): The ID of the newly created parent.

### 2. List Parents

*   **URL:** `/api/v1/parents`
*   **Method:** `GET`
*   **Description:** Retrieves a paginated list of parent profiles, with optional search and filtering.
*   **Authentication:** Required
*   **Query Parameters (ListFilter):
    *   `page` (integer, optional): The page number (default: 1).
    *   `limit` (integer, optional): The number of items per page (default: 50, max: 100).
    *   `search` (string, optional): Search term to filter parents by name or email.
    *   `student_id` (string, optional): Filter parents who are linked to a specific student ID.
    *   `education_level` (array of strings, optional): Filter by education level(s) of linked students.
    *   `grade_level` (array of strings, optional): Filter by grade level(s) of linked students.
*   **Response (ListParentsResponse):
    *   `items` (array of objects): A list of parent objects.
        *   `id` (string)
        *   `full_name` (string)
        *   `email` (string)
        *   `phone_number` (string)
        *   `is_active` (boolean)
        *   `created_at` (string, ISO 8601)
        *   `updated_at` (string, ISO 8601)
    *   `total` (integer): Total number of parents matching the filter.
    *   `page` (integer): Current page number.
    *   `limit` (integer): Items per page.

### 3. Get My Parent Profile

*   **URL:** `/api/v1/parents/me`
*   **Method:** `GET`
*   **Description:** Retrieves the profile of the authenticated parent, including linked children.
*   **Authentication:** Required
*   **Response (ParentDetailResponse):
    *   `data` (object): A detailed parent object.
        *   `id` (string)
        *   `full_name` (string)
        *   `email` (string)
        *   `phone_number` (string)
        *   `address` (string, optional)
        *   `is_active` (boolean)
        *   `user_id` (string, optional): Associated user account ID.
        *   `linked_students` (array of objects):
            *   `id` (string)
            *   `full_name` (string)
            *   `grade_level` (string)
            *   `class_name` (string)
        *   `created_at` (string, ISO 8601)
        *   `updated_at` (string, ISO 8601)

### 4. Get Parent Detail by ID

*   **URL:** `/api/v1/parents/:id`
*   **Method:** `GET`
*   **Description:** Retrieves a single parent profile by its ID, including linked children.
*   **Authentication:** Required
*   **Path Parameters:**
    *   `id` (string): The unique identifier of the parent.
*   **Response (ParentDetailResponse):
    *   `data` (object): A detailed parent object (same as `Get My Parent Profile`).

### 5. Update Parent

*   **URL:** `/api/v1/parents/:id`
*   **Method:** `PUT`
*   **Description:** Updates an existing parent profile. At least one of `phone_number` or `is_active` is required.
*   **Authentication:** Required
*   **Path Parameters:**
    *   `id` (string): The unique identifier of the parent to update.
*   **Request Body (UpdateParentPayload):
    *   `phone_number` (string, optional): New phone number.
    *   `is_active` (boolean, optional): New active status.
    *   `address` (string, optional): New address.
*   **Response:** `200 OK` on success.

### 6. Delete Parent

*   **URL:** `/api/v1/parents`
*   **Method:** `DELETE`
*   **Description:** Deletes a parent profile.
*   **Authentication:** Required
*   **Request Body:
    *   `id` (string): The ID of the parent to delete.
*   **Response:** `204 No Content` on success.
*   **Note:** This endpoint expects the parent ID in the request body for a DELETE operation on the base path. A more conventional RESTful approach would be `DELETE /api/v1/parents/:id`.

### 7. Bulk Invite Parents

*   **URL:** `/api/v1/parents/invite`
*   **Method:** `POST`
*   **Description:** Initiates an asynchronous job to send bulk invitations to parents.
*   **Authentication:** Required
*   **Request Body (BulkInviteRequest):
    *   `rows` (array of objects): An array of parent invitation details.
        *   `full_name` (string): Full name of the parent.
        *   `email` (string): Email address of the parent.
        *   `phone_number` (string, optional): Phone number of the parent.
        *   `student_ids` (array of strings, optional): IDs of students to link.
*   **Response (BulkInviteResponse):
    *   `job_id` (string): The ID of the asynchronous import job.
    *   `total_records` (integer): Total number of invitation records processed.
    *   `total_chunks` (integer): Number of chunks the job was split into.
    *   `status` (string): Current status of the job (e.g., "pending", "in_progress").
    *   `is_replay` (boolean): Indicates if this is a replay of a previous job.

### 8. Bulk Import Parents (Deprecated)

*   **URL:** `/api/v1/parents/import`
*   **Method:** `POST`
*   **Description:** This endpoint is deprecated and not implemented. Use `/api/v1/parents/invite` instead.
*   **Authentication:** Required
*   **Response:** `501 Not Implemented`

## Student Linking

### 9. Link Student to Parent

*   **URL:** `/api/v1/parents/:parent_id/students`
*   **Method:** `POST`
*   **Description:** Links a student to a parent profile.
*   **Authentication:** Required
*   **Path Parameters:**
    *   `parent_id` (string): The ID of the parent.
*   **Request Body (LinkStudentPayload):
    *   `student_id` (string): The ID of the student to link.
*   **Response:** `200 OK` on success.

### 10. Unlink Student from Parent

*   **URL:** `/api/v1/parents/student-link`
*   **Method:** `DELETE`
*   **Description:** Unlinks a student from a parent profile.
*   **Authentication:** Required
*   **Request Body:
    *   `parent_id` (string): The ID of the parent.
    *   `student_id` (string): The ID of the student to unlink.
*   **Response:** `204 No Content` on success.
*   **Note:** This endpoint expects both `parent_id` and `student_id` in the request body for a DELETE operation. A more conventional RESTful approach would be `DELETE /api/v1/parents/:parent_id/students/:student_id`.