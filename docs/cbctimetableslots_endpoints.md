# Timetable Slots Endpoints

This document outlines the API endpoints related to timetable slot management.

## Timetable Slots

### 1. List Timetable Slots

*   **URL:** `/api/v1/timetable/slots`
*   **Method:** `GET`
*   **Description:** Retrieves a list of timetable slots based on provided filters. `academic_year_id` is required.
*   **Authentication:** Required
*   **Query Parameters (SlotFilter):
    *   `academic_year_id` (string, required): The ID of the academic year.
    *   `structure_id` (string, optional): Filter by timetable structure ID (time block).
    *   `class_id` (string, optional): Filter by class ID.
    *   `teacher_id` (string, optional): Filter by teacher ID.
    *   `room_identifier` (string, optional): Filter by room identifier.
*   **Response (List of Slot):** An array of `Slot` objects.
    *   `id` (string)
    *   `academic_year_id` (string)
    *   `structure_id` (string)
    *   `class_id` (string)
    *   `learning_area_id` (string)
    *   `teacher_id` (string)
    *   `room_identifier` (string)
    *   `day_of_week` (integer, 0=Sunday, 6=Saturday)
    *   `start_time` (string, HH:MM)
    *   `end_time` (string, HH:MM)
    *   `is_active` (boolean)
    *   `created_at` (string, ISO 8601)
    *   `updated_at` (string, ISO 8601)

### 2. List Enriched Timetable Slots

*   **URL:** `/api/v1/timetable/slots/enriched`
*   **Method:** `GET`
*   **Description:** Retrieves a list of enriched timetable slots (likely including related details like class name, teacher name, etc.) based on provided filters. `academic_year_id` is required.
*   **Authentication:** Required
*   **Query Parameters (SlotFilter):
    *   `academic_year_id` (string, required): The ID of the academic year.
    *   `structure_id` (string, optional): Filter by timetable structure ID (time block).
    *   `class_id` (string, optional): Filter by class ID.
    *   `teacher_id` (string, optional): Filter by teacher ID.
    *   `room_identifier` (string, optional): Filter by room identifier.
    *   `date` (string, YYYY-MM-DD, optional): Filter by a specific date.
*   **Response (List of EnrichedSlot):** An array of `EnrichedSlot` objects. (Assuming `EnrichedSlot` includes `Slot` fields plus additional related data).

### 3. Get Timetable Slot by ID

*   **URL:** `/api/v1/timetable/slots/:id`
*   **Method:** `GET`
*   **Description:** Retrieves a single timetable slot by its ID.
*   **Authentication:** Required
*   **Path Parameters:**
    *   `id` (string): The unique identifier of the timetable slot.
*   **Response (Slot):** A single `Slot` object.

### 4. Create Timetable Slot

*   **URL:** `/api/v1/timetable/slots`
*   **Method:** `POST`
*   **Description:** Creates a new timetable slot.
*   **Authentication:** Required
*   **Request Body (CreateSlotPayload):
    *   `academic_year_id` (string, required): The academic year this slot belongs to.
    *   `structure_id` (string, required): The timetable structure (time block) ID.
    *   `class_id` (string, required): The class ID for this slot.
    *   `learning_area_id` (string, required): The learning area (subject) ID.
    *   `teacher_id` (string, required): The teacher ID assigned to this slot.
    *   `room_identifier` (string, optional): The room where the class takes place.
    *   `day_of_week` (integer, required): Day of the week (0 for Sunday, 6 for Saturday).
    *   `start_time` (string, HH:MM, required): Start time of the slot.
    *   `end_time` (string, HH:MM, required): End time of the slot.
*   **Response (Slot):** The newly created `Slot` object.

### 5. Batch Create Timetable Slots

*   **URL:** `/api/v1/timetable/slots/batch`
*   **Method:** `POST`
*   **Description:** Creates multiple timetable slots in a single request.
*   **Authentication:** Required
*   **Request Body (BatchCreateSlotsPayload):
    *   `slots` (array of CreateSlotPayload): An array of `CreateSlotPayload` objects, each representing a slot to create.
*   **Response (BatchCreateSlotsResult):
    *   `created_count` (integer): Number of slots successfully created.
    *   `failed_count` (integer): Number of slots that failed to create.
    *   `errors` (array of objects, optional): Details of failures, if any. (e.g., `{"index": 0, "error": "conflict"}`)

### 6. Update Timetable Slot

*   **URL:** `/api/v1/timetable/slots/:id`
*   **Method:** `PUT`
*   **Description:** Updates an existing timetable slot.
*   **Authentication:** Required
*   **Path Parameters:**
    *   `id` (string): The unique identifier of the slot to update.
*   **Request Body (UpdateSlotPayload):
    *   `academic_year_id` (string, optional)
    *   `structure_id` (string, optional)
    *   `class_id` (string, optional)
    *   `learning_area_id` (string, optional)
    *   `teacher_id` (string, optional)
    *   `room_identifier` (string, optional)
    *   `day_of_week` (integer, optional)
    *   `start_time` (string, HH:MM, optional)
    *   `end_time` (string, HH:MM, optional)
    *   `is_active` (boolean, optional)
*   **Response (Slot):** The updated `Slot` object.

### 7. Delete Timetable Slot

*   **URL:** `/api/v1/timetable/slots`
*   **Method:** `DELETE`
*   **Description:** Deletes a timetable slot.
*   **Authentication:** Required
*   **Request Body:
    *   `id` (string): The ID of the slot to delete.
*   **Response:
    *   `code` (string): "ok"
    *   `message` (string): "Slot removed successfully"
*   **Note:** This endpoint expects the slot ID in the request body for a DELETE operation on the base path. A more conventional RESTful approach would be `DELETE /api/v1/timetable/slots/:id`.
