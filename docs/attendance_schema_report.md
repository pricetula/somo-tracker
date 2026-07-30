# SQL Schema Report: Attendance Module

This document outlines the SQL schema pertaining to the attendance module within the SomoTracker platform, as defined in `backend/internal/database/migrations/000001_initial_schema.up.sql` and subsequent migrations.

---

## 1. Enums

### `attendance_status`
Defines the possible states for a student's attendance in a specific timetable slot.

*   `PRESENT`
*   `ABSENT`
*   `LATE`
*   `EXCUSED`

---

## 2. Tables

### `attendance_records`

*   **Purpose**: Stores individual attendance marks for each student for a specific timetable slot on a given date. This table is the granular source of truth for all attendance data.
*   **Columns**:
    *   `id` (UUID, Primary Key, Default: `gen_random_uuid()`)
    *   `tenant_id` (UUID, NOT NULL, Foreign Key to `tenants.id`)
    *   `school_id` (UUID, NOT NULL)
    *   `student_id` (UUID, NOT NULL, Foreign Key to `cbc_students.id`)
    *   `timetable_slot_id` (UUID, NOT NULL, Foreign Key to `cbc_timetable_slots.id`)
    *   `academic_term_id` (UUID, NOT NULL)
    *   `date` (DATE, NOT NULL)
    *   `status` (`attendance_status`, NOT NULL)
    *   `marked_by` (UUID, NOT NULL, Foreign Key to `users.id`)
    *   `marked_at` (TIMESTAMPTZ, NOT NULL, Default: `NOW()`)
    *   `note` (TEXT, NULL) - Optional short free text for specific attendance notes (e.g., "left early, picked up by parent").
    *   `created_at` (TIMESTAMPTZ, NOT NULL, Default: `NOW()`)
    *   `updated_at` (TIMESTAMPTZ, NOT NULL, Default: `NOW()`) - Tracks modifications to the record (e.g., status or note updates).
    *   `attendance_session_id` (UUID, NULL, Foreign Key to `cbc_attendance_sessions.id`) - Optional reference to the `cbc_attendance_sessions` row, particularly relevant when a session is marked as `SKIPPED`.

*   **Constraints & Indexes**:
    *   `uq_attendance_student_slot_date`: `UNIQUE (student_id, timetable_slot_id, date)` - Ensures only one attendance record exists for a given student in a specific timetable slot on a particular date.
    *   `fk_attendance_tenant_student`: `FOREIGN KEY (tenant_id, student_id) REFERENCES cbc_students(tenant_id, id) ON DELETE CASCADE`
    *   `fk_attendance_timetable_slot`: `FOREIGN KEY (timetable_slot_id) REFERENCES cbc_timetable_slots(id) ON DELETE CASCADE`
    *   `fk_attendance_tenant_term`: `FOREIGN KEY (tenant_id, school_id, academic_term_id) REFERENCES academic_terms(tenant_id, school_id, id) ON DELETE CASCADE`
    *   `fk_attendance_marked_by`: `FOREIGN KEY (tenant_id, marked_by) REFERENCES users(tenant_id, id) ON DELETE RESTRICT`
    *   Indexes: `idx_attendance_slot_date`, `idx_attendance_student_term`, `idx_attendance_tenant`, `idx_attendance_school`, `idx_attendance_records_session`.

*   **Triggers**:
    *   `trg_attendance_records_updated_at`: `BEFORE UPDATE` - Automatically sets `updated_at` to `NOW()`.
    *   `trg_attendance_check_non_break_slot`: `BEFORE INSERT OR UPDATE` - Enforces that attendance records can only be created for timetable slots that are *not* marked as `is_break = true` in `timetable_structures`.

---

### `cbc_attendance_sessions`

*   **Purpose**: Tracks whether a scheduled timetable slot actually occurred or was intentionally skipped (e.g., due to a public holiday, teacher absence, or school assembly). Skipped sessions are excluded from overall attendance percentage calculations to avoid penalizing students for cancelled lessons.
*   **Columns**:
    *   `id` (UUID, Primary Key, Default: `gen_random_uuid()`)
    *   `tenant_id` (UUID, NOT NULL, Foreign Key to `tenants.id`)
    *   `school_id` (UUID, NOT NULL, Foreign Key to `cbc_schools.id`)
    *   `timetable_slot_id` (UUID, NOT NULL, Foreign Key to `cbc_timetable_slots.id`)
    *   `date` (DATE, NOT NULL)
    *   `status` (VARCHAR(20), NOT NULL, Default: `'SUBMITTED'`) - Values: `SUBMITTED` (lesson held as scheduled) or `SKIPPED` (lesson did not hold).
    *   `skip_reason` (TEXT, NULL) - Teacher-provided reason when `status` is `SKIPPED` (e.g., "School Assembly", "Teacher Absence").
    *   `created_at` (TIMESTAMPTZ, NOT NULL, Default: `NOW()`)
    *   `updated_at` (TIMESTAMPTZ, NOT NULL, Default: `NOW()`) - Tracks changes to the session status or skip reason.

*   **Constraints & Indexes**:
    *   `chk_cbc_attendance_session_status`: `CHECK (status IN ('SUBMITTED', 'SKIPPED'))`
    *   `uq_cbc_attendance_sessions_slot_date`: `UNIQUE (school_id, timetable_slot_id, date)` - Ensures one session entry per school, timetable slot, and date.
    *   `fk_cbc_attendance_sessions_tenant_school`: `FOREIGN KEY (tenant_id, school_id) REFERENCES cbc_schools(tenant_id, id) ON DELETE CASCADE`
    *   Indexes: `idx_cbc_attendance_sessions_slot_date`, `idx_cbc_attendance_sessions_tenant`, `idx_cbc_attendance_sessions_school`, `idx_cbc_attendance_sessions_status`.

*   **Triggers**:
    *   `trg_cbc_attendance_sessions_updated_at`: `BEFORE UPDATE` - Automatically sets `updated_at` to `NOW()`.

---

### `attendance_term_summaries`

*   **Purpose**: A materialised (pre-calculated) rollup of attendance data for efficient reporting. It summarizes attendance metrics per student, per academic term, and per learning area. While this table provides fast access to summary data, `attendance_records` remains the authoritative source of truth for all attendance calculations.
*   **Columns**:
    *   `id` (UUID, Primary Key, Default: `gen_random_uuid()`)
    *   `tenant_id` (UUID, NOT NULL)
    *   `school_id` (UUID, NOT NULL)
    *   `student_id` (UUID, NOT NULL, Foreign Key to `cbc_students.id`)
    *   `academic_term_id` (UUID, NOT NULL)
    *   `learning_area_id` (UUID, NOT NULL, Foreign Key to `cbc_learning_areas.id`)
    *   `periods_total` (INT, NOT NULL) - Total expected instructional periods for the given student, term, and learning area.
    *   `periods_present` (INT, NOT NULL)
    *   `periods_absent` (INT, NOT NULL)
    *   `periods_late` (INT, NOT NULL)
    *   `periods_excused` (INT, NOT NULL)
    *   `attendance_percentage` (NUMERIC(5,2), NOT NULL) - Calculated as `(periods_present / periods_total) * 100`. Stored as a decimal with two fractional digits (e.g., 92.50).
    *   `last_refreshed_at` (TIMESTAMPTZ, NOT NULL, Default: `NOW()`) - Timestamp indicating when the summary was last recalculated/refreshed.
    *   `updated_at` (TIMESTAMPTZ, NOT NULL, Default: `NOW()`)

*   **Constraints & Indexes**:
    *   `uq_summary_student_term_area`: `UNIQUE (student_id, academic_term_id, learning_area_id)` - Ensures only one summary entry per student, academic term, and learning area.
    *   `fk_summaries_tenant_student`: `FOREIGN KEY (tenant_id, student_id) REFERENCES cbc_students(tenant_id, id) ON DELETE CASCADE`
    *   `fk_summaries_tenant_term`: `FOREIGN KEY (tenant_id, school_id, academic_term_id) REFERENCES academic_terms(tenant_id, school_id, id) ON DELETE CASCADE`
    *   `fk_summaries_learning_area`: `FOREIGN KEY (learning_area_id) REFERENCES cbc_learning_areas(id) ON DELETE CASCADE`
    *   Indexes: `idx_att_summaries_student_term`, `idx_att_summaries_tenant`, `idx_att_summaries_school`.

*   **Triggers**:
    *   `trg_attendance_term_summaries_updated_at`: `BEFORE UPDATE` - Automatically sets `updated_at` to `NOW()`.

*   **Additional Columns (from `000005_extend_summaries_and_daily.up.sql`)**:
    *   `academic_year_id` (UUID, NOT NULL) - Foreign key to `academic_years(id)`. Backfilled from `academic_terms` and made `NOT NULL`.
    *   `created_at` (TIMESTAMPTZ, NOT NULL, Default: `NOW()`) - Added to track when the summary was first created.
    *   Index: `idx_att_summaries_academic_year` - Added for year-scoped queries.

---

### `class_daily_attendance_summaries`

*   **Source Migration**: `000005_extend_summaries_and_daily.up.sql`
*   **Purpose**: A materialised rollup of attendance records per class per date. It provides a daily snapshot of attendance metrics for an entire class, enabling efficient reporting at the class level.
*   **Columns**:
    *   `id` (UUID, Primary Key, Default: `gen_random_uuid()`)
    *   `tenant_id` (UUID, NOT NULL)
    *   `school_id` (UUID, NOT NULL)
    *   `class_id` (UUID, NOT NULL, Foreign Key to `cbc_classes.id`)
    *   `academic_term_id` (UUID, NOT NULL)
    *   `date` (DATE, NOT NULL)
    *   `total_enrolled` (INT, NOT NULL) - Total number of enrolled students for the class on the given date.
    *   `present_count` (INT, NOT NULL)
    *   `absent_count` (INT, NOT NULL)
    *   `late_count` (INT, NOT NULL)
    *   `excused_count` (INT, NOT NULL)
    *   `daily_attendance_rate` (NUMERIC(5,2), NOT NULL) - Calculated as `(present_count / (present_count + absent_count + late_count + excused_count)) * 100`, stored as a decimal with two fractional digits (e.g., 94.60).
    *   `last_refreshed_at` (TIMESTAMPTZ, NOT NULL, Default: `NOW()`) - Timestamp indicating when the summary was last recalculated/refreshed.
    *   `created_at` (TIMESTAMPTZ, NOT NULL, Default: `NOW()`)
    *   `updated_at` (TIMESTAMPTZ, NOT NULL, Default: `NOW()`)

*   **Constraints & Indexes**:
    *   `uq_class_daily_attendance`: `UNIQUE (class_id, date)` - Ensures only one summary entry per class per date.
    *   `fk_class_daily_tenant_class`: `FOREIGN KEY (tenant_id, class_id) REFERENCES cbc_classes(tenant_id, id) ON DELETE CASCADE`
    *   `fk_class_daily_tenant_term`: `FOREIGN KEY (tenant_id, school_id, academic_term_id) REFERENCES academic_terms(tenant_id, school_id, id) ON DELETE CASCADE`
    *   Indexes: `idx_class_daily_tenant`, `idx_class_daily_school`, `idx_class_daily_class_date`, `idx_class_daily_academic_term`.

*   **Triggers**:
    *   `trg_class_daily_attendance_summaries_updated_at`: `BEFORE UPDATE` - Automatically sets `updated_at` to `NOW()`.

*   **Notes**:
    *   Populated by incremental background tasks triggered when all attendance for a class-date is marked (or on a timeout).
    *   `total_enrolled` is derived from distinct students who have `attendance_records` rows that day, not from `cbc_student_enrollments.status`. This is a documented workaround because enrollment status has no effective date within a term — a student suspended on day 50 would otherwise vanish from every earlier day's calculation too.

---

### `teacher_delivery_summaries`

*   **Source Migration**: `000013_create_teacher_delivery_summaries.up.sql`
*   **Purpose**: An incrementally updated summary of teacher lesson delivery metrics per term. While not a pure attendance table, it is derived from attendance data (`attendance_records` and `cbc_attendance_sessions`) and provides teacher-level delivery insights.
*   **Columns**:
    *   `id` (UUID, Primary Key, Default: `gen_random_uuid()`)
    *   `tenant_id` (UUID, NOT NULL)
    *   `school_id` (UUID, NOT NULL)
    *   `user_id` (UUID, NOT NULL, Foreign Key to `users.id`)
    *   `academic_term_id` (UUID, NOT NULL)
    *   `total_assigned_slots` (INT, NOT NULL, Default: 0) - Total number of timetable slot occurrences assigned to this teacher during the term.
    *   `marked_slots` (INT, NOT NULL, Default: 0) - Number of assigned slot occurrences where attendance records exist (attendance was taken).
    *   `missed_slots` (INT, NOT NULL, Default: 0) - Number of assigned slot occurrences where the lesson was marked SKIPPED.
    *   `sessions_created` (INT, NOT NULL, Default: 0) - Number of `cbc_attendance_sessions` records associated with this teacher's slots in the term.
    *   `sessions_approved` (INT, NOT NULL, Default: 0) - Number of sessions where status = `SUBMITTED`.
    *   `on_time_submission_rate` (NUMERIC(5,2), NULL) - Percentage of assigned slots that were either marked or skipped: `(marked_slots + missed_slots) / total_assigned_slots * 100`.
    *   `last_refreshed_at` (TIMESTAMPTZ, NOT NULL, Default: `NOW()`)
    *   `created_at` (TIMESTAMPTZ, NOT NULL, Default: `NOW()`)
    *   `updated_at` (TIMESTAMPTZ, NOT NULL, Default: `NOW()`)

*   **Constraints & Indexes**:
    *   `uq_teacher_delivery_term`: `UNIQUE (user_id, academic_term_id)` - Ensures one summary entry per teacher per term.
    *   `fk_teacher_delivery_tenant_school`: `FOREIGN KEY (tenant_id, school_id) REFERENCES cbc_schools(tenant_id, id) ON DELETE CASCADE`
    *   `fk_teacher_delivery_tenant_user`: `FOREIGN KEY (tenant_id, user_id) REFERENCES users(tenant_id, id) ON DELETE CASCADE`
    *   `fk_teacher_delivery_term`: `FOREIGN KEY (tenant_id, school_id, academic_term_id) REFERENCES academic_terms(tenant_id, school_id, id) ON DELETE CASCADE`
    *   Indexes: `idx_teacher_delivery_tenant`, `idx_teacher_delivery_school`, `idx_teacher_delivery_user`, `idx_teacher_delivery_term`.

*   **Triggers**:
    *   `trg_teacher_delivery_summaries_updated_at`: `BEFORE UPDATE` - Automatically sets `updated_at` to `NOW()`.

*   **Functions**:
    *   `fn_compute_teacher_delivery_summaries(target_term_id UUID)`: Batch-computes `teacher_delivery_summaries` for all teachers with timetable slots in the given term. Uses `attendance_records` and `cbc_attendance_sessions` to calculate delivery metrics.

*   **Notes**:
    *   Grain: `(user_id, academic_term_id)`.
    *   Updated via triggers on `attendance_records` INSERT and `cbc_attendance_sessions` status changes.
    *   Slot ownership is resolved via `cbc_timetable_slots.teacher_id`.
    *   Has Row-Level Security (RLS) enabled with `tenant_isolation_policy`.

---

## 3. Related Tables (Contextual for Attendance)

The attendance module relies on several other core tables for its functionality and data integrity:

*   **`academic_terms`**: Defines the academic terms (e.g., Term 1, Term 2) within an academic year, providing the temporal scope for attendance records and summaries.
*   **`cbc_students`**: Stores comprehensive student profiles, providing the `student_id` referenced in all attendance tables.
*   **`cbc_timetable_slots`**: Defines specific teaching periods in the school's timetable, linking an `attendance_record` to a particular lesson.
*   **`timetable_structures`**: Defines the master grid of a school day (lessons, breaks). It's indirectly used by `attendance_records` via `cbc_timetable_slots` to enforce that attendance cannot be marked during break periods.
*   **`cbc_learning_areas`**: Represents subjects taught in the curriculum, providing the `learning_area_id` for `attendance_term_summaries`.
*   **`users`**: Contains user profiles, allowing `attendance_records` to track which user (`marked_by`) recorded the attendance.
*   **`tenants`**: Provides the multi-tenancy context, ensuring data isolation across different organizations.
*   **`cbc_schools`**: Provides the school context, linking attendance data to a specific school.

---

## 4. Row-Level Security (RLS)

All attendance-related tables (`attendance_records`, `cbc_attendance_sessions`, `attendance_term_summaries`) have Row-Level Security enabled with a `tenant_isolation_policy`. This policy ensures that queries only return rows belonging to the `tenant_id` set in the current session (`app.current_tenant_id`), preventing cross-tenant data leaks at the database level.
