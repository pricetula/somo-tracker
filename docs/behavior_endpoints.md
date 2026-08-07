# Behavior Endpoints

This document outlines the API endpoints related to behavior management, including categories, notes, and student summaries.

## Behavior Categories (Admin-managed)

### 1. List Behavior Categories

*   **URL:** `/api/v1/behavior/categories`
*   **Method:** `GET`
*   **Description:** Retrieves a list of behavior categories.
*   **Authentication:** Required
*   **Query Parameters:**
    *   `active_only` (boolean, optional): If `true`, only returns active categories.
*   **Response (List of Category):
    *   `items` (array of objects):
        *   `id` (string): Unique identifier for the category.
        *   `name` (string): Name of the category.
        *   `description` (string, optional): Description of the category.
        *   `is_active` (boolean): Whether the category is active.
        *   `tenant_id` (string)
        *   `school_id` (string)
        *   `created_at` (string, ISO 8601)
        *   `updated_at` (string, ISO 8601)
    *   `total` (integer): Total number of categories.

### 2. Create Behavior Category

*   **URL:** `/api/v1/behavior/categories`
*   **Method:** `POST`
*   **Description:** Creates a new behavior category.
*   **Authentication:** Required
*   **Request Body (CreateCategoryPayload):
    *   `name` (string): The name of the category.
    *   `description` (string, optional): A description for the category.
    *   `is_active` (boolean, optional): Whether the category is active. Defaults to true.
*   **Response (Category):** The newly created `Category` object.

### 3. Get Behavior Category by ID

*   **URL:** `/api/v1/behavior/categories/:id`
*   **Method:** `GET`
*   **Description:** Retrieves a single behavior category by its ID.
*   **Authentication:** Required
*   **Path Parameters:**
    *   `id` (string): The unique identifier of the category.
*   **Response (Category):** A single `Category` object.

### 4. Update Behavior Category

*   **URL:** `/api/v1/behavior/categories/:id`
*   **Method:** `PUT`
*   **Description:** Updates an existing behavior category.
*   **Authentication:** Required
*   **Path Parameters:**
    *   `id` (string): The unique identifier of the category to update.
*   **Request Body (UpdateCategoryPayload):
    *   `name` (string, optional): New name for the category.
    *   `description` (string, optional): New description for the category.
    *   `is_active` (boolean, optional): New active status for the category.
*   **Response (Category):** The updated `Category` object.

## Behavior Notes

### 5. Create Behavior Note

*   **URL:** `/api/v1/behavior/notes`
*   **Method:** `POST`
*   **Description:** Creates a new behavior note for a student.
*   **Authentication:** Required
*   **Request Body (CreateNotePayload):
    *   `student_id` (string): The ID of the student the note is about.
    *   `category_id` (string): The ID of the behavior category.
    *   `description` (string): Detailed description of the behavior.
    *   `incident_date` (string, YYYY-MM-DD): The date of the incident.
    *   `status` (string): Initial status of the note (e.g., "pending", "approved").
*   **Response (Note):
    *   `id` (string)
    *   `student_id` (string)
    *   `category_id` (string)
    *   `description` (string)
    *   `incident_date` (string, YYYY-MM-DD)
    *   `status` (string)
    *   `author_id` (string)
    *   `reviewer_id` (string, optional)
    *   `reviewed_at` (string, ISO 8601, optional)
    *   `created_at` (string, ISO 8601)
    *   `updated_at` (string, ISO 8601)

### 6. List Behavior Notes (Authored by User)

*   **URL:** `/api/v1/behavior/notes`
*   **Method:** `GET`
*   **Description:** Retrieves a list of behavior notes authored by the authenticated user.
*   **Authentication:** Required
*   **Response (List of Note):** An array of `Note` objects.

### 7. Get Pending Behavior Notes Queue

*   **URL:** `/api/v1/behavior/notes/queue`
*   **Method:** `GET`
*   **Description:** Retrieves a list of behavior notes that are pending review.
*   **Authentication:** Required
*   **Response (List of Note):** An array of `Note` objects, typically with `status: "pending"`.

### 8. Get Behavior Note by ID

*   **URL:** `/api/v1/behavior/notes/:id`
*   **Method:** `GET`
*   **Description:** Retrieves a single behavior note by its ID.
*   **Authentication:** Required
*   **Path Parameters:**
    *   `id` (string): The unique identifier of the note.
*   **Response (Note):** A single `Note` object.

### 9. Update Behavior Note

*   **URL:** `/api/v1/behavior/notes/:id`
*   **Method:** `PUT`
*   **Description:** Updates the description of an existing behavior note.
*   **Authentication:** Required
*   **Path Parameters:**
    *   `id` (string): The unique identifier of the note to update.
*   **Request Body:
    *   `description` (string): The new description for the note.
*   **Response:
    *   `message` (string): "Behavior note updated"

### 10. Delete Behavior Note

*   **URL:** `/api/v1/behavior/notes` (Note: The route definition is `notes.Delete("/", middleware.RequireAuth, h.DeleteNote)` but the handler uses a payload with `ID` which suggests it should be `DELETE /api/v1/behavior/notes/:id` or expect a body. Based on the payload, it should ideally be `DELETE /api/v1/behavior/notes` with a body.)
*   **Method:** `DELETE`
*   **Description:** Deletes a behavior note.
*   **Authentication:** Required
*   **Request Body:
    *   `id` (string): The ID of the note to delete.
*   **Response:** `204 No Content` on success.

### 11. Review Behavior Note

*   **URL:** `/api/v1/behavior/notes/:id/review`
*   **Method:** `POST`
*   **Description:** Reviews a behavior note, changing its status (e.g., approve or reject).
*   **Authentication:** Required
*   **Path Parameters:**
    *   `id` (string): The unique identifier of the note to review.
*   **Request Body (ReviewDecisionPayload):
    *   `decision` (string): The review decision (e.g., "approved", "rejected").
    *   `remarks` (string, optional): Any remarks from the reviewer.
*   **Response:
    *   `message` (string): "Behavior note reviewed"
    *   `decision` (string): The decision made.

## Student Behavior Term Summaries

### 12. List Student Behavior Term Summaries

*   **URL:** `/api/v1/behavior/summaries`
*   **Method:** `GET`
*   **Description:** Retrieves a list of student behavior term summaries. Requires `term_id`. Can be filtered by `student_id`.
*   **Authentication:** Required
*   **Query Parameters:
    *   `term_id` (string, required): The ID of the academic term.
    *   `student_id` (string, optional): Filter by a specific student ID.
*   **Response (List of StudentBehaviorTermSummary):
    *   `student_id` (string)
    *   `term_id` (string)
    *   `total_incidents` (integer)
    *   `positive_incidents` (integer)
    *   `negative_incidents` (integer)
    *   `incidents_by_category` (map[string]integer, where key is category name)

### 13. Get Student Behavior Term Summary

*   **URL:** `/api/v1/behavior/summaries/:student_id`
*   **Method:** `GET`
*   **Description:** Retrieves a single student behavior term summary for a specific student and term.
*   **Authentication:** Required
*   **Path Parameters:
    *   `student_id` (string, required): The ID of the student.
*   **Query Parameters:
    *   `term_id` (string, required): The ID of the academic term.
*   **Response (StudentBehaviorTermSummary):** A single `StudentBehaviorTermSummary` object.
