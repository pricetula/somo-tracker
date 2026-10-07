/**
 * School Event type matching the backend `school_events` table structure.
 *
 * Backend schema (from migration 000006):
 *   id (UUID) – auto-generated primary key
 *   school_id (UUID) – FK to schools(id), cascades on school delete
 *   title (text) – Event title (e.g., "Inter-House Sports Day")
 *   event_type (text) – Event type: SPORTS, ADMISSION, EXAM, etc.
 *   start_date (date) – First day of the event
 *   end_date (date) – Last day of the event
 *   requires_attendance (boolean) – Whether student attendance must be tracked
 *
 * The frontend Event type uses snake_case field names to match the backend
 * JSON response from `GET /events` and `POST /events`.
 */
export type Event = {
    id: string;
    title: string;
    event_type: string;
    start_date: string; // YYYY-MM-DD
    end_date: string; // YYYY-MM-DD
    requires_attendance: boolean;
};

/**
 * Request payload for creating a new school event.
 *
 * Backend endpoint: `POST /events` (rewritten via `/backend` proxy).
 */
export type CreateEventInput = {
    title: string;
    event_type: string;
    start_date: string;
    end_date: string;
    requires_attendance: boolean;
};

/**
 * Request payload for updating an existing school event.
 *
 * Backend endpoint: `PATCH /events/:id` (not yet implemented, planned).
 */
export type UpdateEventInput = Partial<CreateEventInput> & { id: string };
