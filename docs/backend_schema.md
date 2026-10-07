# Backend Schema Summary

This document provides a comprehensive overview of the Somotracker backend database schema, organized by table with descriptions of purpose, columns, and relationships.

## Core Tenancy & Identity

### `tenants`
- **Purpose**: Represents multi-tenant organizations (Stytch OIDC organizations). Each tenant is a distinct school/organization boundary.
- **Key Columns**:
  - `id` (UUID) – primary key
  - `name` – human‑readable organization name
  - `slug` – URL‑safe, globally unique identifier for routing
  - `stytch_org_id` – Stytch OIDC organization ID (authoritative SSO anchor)
  - `created_at` – row creation timestamp
- **Relationships**: One‑to‑many with `users` (each user belongs to exactly one tenant).

### `users`
- **Purpose**: Per‑tenant user accounts. Users are scoped to a tenant and share an email address within that tenant.
- **Key Columns**:
  - `id` (UUID) – primary key
  - `email` – unique email within the tenant (enforced via `UNIQUE (tenant_id, email)`)
  - `tenant_id` – foreign key to `tenants` (cascade delete)
  - `full_name` – display name
  - `is_active` – soft‑disable flag
  - `external_auth_id` – Stytch user ID for SSO linking
  - `created_at` / `updated_at` – timestamps
- **Relationships**: Many‑to‑one with `tenants`; many‑to‑many with `sessions` (via `user_id`), `members` (via `user_id`), and `school_memberships` (via `user_id`).

### `sessions`
- **Purpose**: Server‑issued opaque session tokens. Raw Stytch tokens are cached only in Redis; the DB stores an internal opaque token.
- **Key Columns**:
  - `id` (UUID) – primary key
  - `token` – 64‑char opaque token (never exposed to clients)
  - `stytch_session_id` – reference to the Stytch session
  - `user_id` – FK to `users` (deleted when user is removed)
  - `tenant_id` – FK to `tenants` (enforced via RLS)
  - `expires_at` – rolling expiration (default 7 days)
  - `created_at` / `last_seen_at` – timestamps
- **Relationships**: Many‑to‑one with `users`; many‑to‑one with `sessions`.

### `members`
- **Purpose**: B2B member identities (Stytch `member_id`) linked to users. Acts as the bridge between the application user and the external Stytch identity.
- **Key Columns**:
  - `id` (UUID) – primary key
  - `stytch_member_id` – canonical Stytch member ID (unique per tenant)
  - `user_id` – FK to `users`
  - `tenant_id` – FK to `tenants`
  - `stytch_member_raw` – JSONB cache of raw Stytch member data
  - `created_at` / `updated_at`
- **Relationships**: Many‑to‑one with `users`; many‑to‑one with `schools` (via `user_id` → `schools`).

## Curriculum Hierarchy

### `subjects`
- **Purpose**: Curriculum subjects scoped to an education system.
- **Key Columns**:
  - `id` (UUID) – primary key
  - `education_system_id` – FK to `education_systems`
  - `name` – subject name
  - `code` – short code (unique per education system)
  - `type` – subject type (Core, Optional, Elective, …)
  - `created_at` / `updated_at`
- **Relationships**: Many‑to‑one with `education_systems`.

### `topics`
- **Purpose**: Topics within a subject, ordered chronologically.
- **Key Columns**:
  - `id` (UUID) – primary key
  - `subject_id` – FK to `subjects`
  - `name` – topic name
  - `sequence_index` – order within the subject
  - `created_at` / `updated_at`
- **Relationships**: Many‑to‑one with `subjects`.

### `sub_topics`
- **Purpose**: Sub‑topics within a topic (granular curriculum units).
- **Key Columns**:
  - `id` (UUID) – primary key
  - `topic_id` – FK to `topics`
  - `name` – sub‑topic name
  - `sequence_index` – order within the topic
  - `created_at` / `updated_at`
- **Relationships**: Many‑to‑one with `topics`.

### `grade_levels`
- **Purpose**: Grade levels (e.g., PP1, Grade 7, Junior Sec) scoped to an education system and country.
- **Key Columns**:
  - `id` (UUID) – primary key
  - `education_system_id` – FK to `education_systems`
  - `country_id` – FK to `countries`
  - `tier_stage` – broad stage (pre‑primary, primary, lower_secondary, …)
  - `local_label` – human‑readable label (e.g., “PP1”)
  - `sequence_index` – ordering within a system+country
- **Relationships**: Many‑to‑one with `education_systems` and `countries`.

## Schools & Scheduling

### `schools`
- **Purpose**: Schools operated by a tenant (the B2B organization that runs the institution).
- **Key Columns**:
  - `id` (UUID) – primary key
  - `tenant_id` – FK to `tenants`
  - `school_name` – human‑readable name
  - `country_id` – FK to `countries`
  - `education_system_id` – FK to `education_systems`
  - `created_at` / `updated_at`
- **Relationships**: Many‑to‑one with `tenants`; many‑to‑one with `schools` (via `student_class_enrollments`, `guardian_student_links`, etc.), `class_rooms`, `school_memberships`, `streams`.

### `streams`
- **Purpose**: Academic streaming (e.g., Grade 1, Grade 3, Senior 1).
- **Key Columns**:
  - `id` (UUID) – primary key
  - `school_id` – FK to `schools`
  - `name` – stream identifier
- **Relationships**: Many‑to‑one with `schools`.

### `class_rooms`
- **Purpose**: Operational classroom containers per academic year and stream.
- **Key Columns**:
  - `id` (UUID) – primary key
  - `school_id` – FK to `schools`
  - `academic_year_id` – FK to `academic_years`
  - `grade_level_id` – FK to `grade_levels`
  - `name` – human‑readable room name
  - `stream` – optional grade‑specific stream identifier
  - `created_at` / `updated_at`
- **Relationships**: Many‑to‑one with `schools`, `academic_years`, `grade_levels`.

### `student_class_enrollments`
- **Purpose**: Historical mapping of a student to a class room for a given academic term/year.
- **Key Columns**:
  - `id` (UUID) – primary key
  - `school_id` – FK to `schools`
  - `student_id` – FK to `students`
  - `class_room_id` – FK to `class_rooms`
  - `academic_year_id` – FK to `academic_years`
  - `academic_term_id` – FK to `academic_terms` (optional)
  - `status` – `ACTIVE`, `PROMOTED`, `REPEATING`, `GRADUATED`
  - `enrolled_at` / `completed_at` – timestamps
- **Relationships**: Many‑to‑one with `schools`, `students`, `class_rooms`, `academic_years`, `academic_terms`.

### `class_timetable_slots`
- **Purpose**: Assigns a class room to a specific time slot within a timetable template.
- **Key Columns**:
  - `id` (UUID) – primary key
  - `school_id` – FK to `schools`
  - `class_room_id` – FK to `class_rooms`
  - `academic_term_id` – FK to `academic_terms`
  - `day_of_week` – integer (1‑7)
  - `time_slot_id` – FK to `time_slots`
  - `subject_id` – FK to `subjects`
  - `teacher_membership_id` – FK to `school_memberships`
  - `room_id` – FK to `rooms` (optional physical location)
  - `created_at` / `updated_at`
- **Relationships**: Many‑to‑one with `schools`, `class_rooms`, `academic_terms`, `subjects`, `school_memberships`, `rooms`.

### `timetable_substitutions`
- **Purpose**: Emergency or planned teacher‑absence coverings for a scheduled slot.
- **Key Columns**:
  - `id` (UUID) – primary key
  - `school_id` – FK to `schools`
  - `class_timetable_slot_id` – FK to `class_timetable_slots`
  - `substitution_date` – calendar date
  - `original_teacher_membership_id` – FK to `school_memberships`
  - `substitute_teacher_membership_id` – FK to `school_memberships` (NULL if none)
  - `status` – `PENDING`, `ASSIGNED`, `COMPLETED`, `CANCELLED`
  - `reason` – free‑text justification
- **Relationships**: Many‑to‑one with `schools` and `class_timetable_slots`.

### `timetable_attendance`
- **Purpose**: Records whether a student was present, absent, late, or excused for a timetable slot on a given date.
- **Key Columns**:
  - `id` (UUID) – primary key
  - `school_id` – FK to `schools`
  - `student_id` – FK to `students`
  - `class_timetable_slot_id` – FK to `class_timetable_slots`
  - `attendance_date` – calendar date
  - `status` – `PRESENT`, `ABSENT`, `LATE`, `EXCUSED`
  - `remarks` – optional notes
- **Relationships**: Many‑to‑one with `schools`, `students`, `class_timetable_slots`.

### `event_attendance`
- **Purpose**: Attendance tracking for school‑wide events (sports days, symposia, etc.).
- **Key Columns**:
  - `id` (UUID) – primary key
  - `school_id` – FK to `schools`
  - `student_id` – FK to `students`
  - `school_event_id` – FK to `school_events`
  - `attendance_date` – calendar date
  - `status` – `PRESENT`, `ABSENT`, `EXCUSED`
  - `remarks` – optional notes
- **Relationships**: Many-to-one with `schools` and `students`.

## Bulk Operations

### `bulk_jobs`
- **Purpose**: Generic framework for large-scale data ingestion (admin invitations, student imports, exam results, staff imports, etc.).
- **Key Columns**:
  - `id` (UUID) - primary key
  - `job_type` - restricted to `ADMIN_INVITATION`, `STUDENT_IMPORT`, `TEACHER_INVITATION`, `GUARDIAN_INVITATION`, `FINANCE_INVITATION`
  - `idempotency_key` - prevents duplicate submissions per tenant
  - `school_id` - FK to `schools`
  - `tenant_id` - FK to `tenants`
  - `created_by` - FK to `users` (SET NULL on delete to preserve audit trail)
  - `status` - `QUEUED`, `PROCESSING`, `COMPLETED`, `COMPLETED_WITH_ERRORS`, `FAILED`
  - `total_records`, `succeeded_count`, `failed_count`, `deferred_count` - counters
  - `metadata` - JSONB for job-type-specific summary info
  - `created_at` / `updated_at` - timestamps
- **Relationships**: Many-to-one with `schools`, `tenants`, `users`; one-to-many with `bulk_job_items`.

### `bulk_job_items`
- **Purpose**: Individual row-level records for each bulk ingestion job.
- **Key Columns**:
  - `id` (UUID) - primary key
  - `job_id` - FK to `bulk_jobs` (cascade delete)
  - `row_index` - original array position in submitted payload
  - `payload` - JSONB of submitted row data
  - `result` - JSONB of output on success
  - `status` - `PENDING`, `PROCESSING`, `SUCCEEDED`, `FAILED`, `DEFERRED`
  - `attempt_count` - number of processing attempts
  - `last_error` - human-readable error from last failure
  - `created_at` / `updated_at` - timestamps
- **Relationships**: Many-to-one with `bulk_jobs`.

### `student_gender_counts`
- **Purpose**: Denormalized gender counts per school for fast reporting (updated via trigger after bulk imports and student CRUD).
- **Key Columns**:
  - `school_id` (UUID) - primary key, FK to `schools`
  - `male_count`, `female_count`, `other_count`, `total_count` - integer counters
  - `updated_at` - last recompute timestamp
- **Relationships**: One-to-one with `schools`.

## Reference & Lookup Tables

### `countries`
- **Purpose**: ISO 3166-1 alpha-2 country reference.
- **Key Columns**: `id`, `country_name`, `country_code` (unique), timestamps.
- **Relationships**: Referenced by `education_systems`, `grade_levels`, `schools`, `public_holidays`.

### `education_systems`
- **Purpose**: Education systems (e.g., Competency-Based Education) scoped to a country.
- **Key Columns**: `id`, `country_id` (FK), `system_name`, `description`, timestamps.
- **Relationships**: Many-to-one with `countries`; one-to-many with `grade_levels`, `subjects`, `schools`.

### `school_memberships`
- **Purpose**: Links a user to a school with a role (`ADMIN`, `TEACHER`, `GUARDIAN`, `FINANCE`). A user can have at most one membership per school, and at most one *active* membership across all schools.
- **Key Columns**:
  - `id` (UUID) - primary key
  - `school_id` - FK to `schools`
  - `user_id` - FK to `users`
  - `role` - `user_role` enum
  - `is_active` - boolean (enforced by partial unique index)
  - `invited_at`, `invited_by`, `accepted_at` - invitation tracking
  - `created_at` / `updated_at` - timestamps
- **Relationships**: Many-to-one with `schools`, `users`; referenced by `guardian_student_links`, `class_timetable_slots`, `timetable_substitutions`, `timetable_attendance` (recorded_by).

### `students`
- **Purpose**: Student records scoped to a school. `admission_number` is unique per school.
- **Key Columns**:
  - `student_id` (UUID) - primary key
  - `school_id` - FK to `schools`
  - `admission_number` - school-scoped identifier
  - `full_name`, `date_of_birth`, `gender` (`student_gender` enum: `M`, `F`, `OTHER`)
  - `metadata` - JSONB for external IDs (NEMIS, KICD, etc.)
  - `created_at` / `updated_at`
- **Relationships**: Many-to-one with `schools`; one-to-many with `guardian_student_links`, `student_class_enrollments`, `timetable_attendance`, `event_attendance`.

### `guardian_student_links`
- **Purpose**: Links a guardian (via their `school_membership`) to a student with a relationship type (Parent, Legal Guardian, Sponsor, ...).
- **Key Columns**:
  - `id` (UUID) - primary key
  - `school_membership_id` - FK to `school_memberships`
  - `student_id` - FK to `students`
  - `relationship_type` - free text
  - `created_at` / `updated_at`
- **Relationships**: Many-to-one with `school_memberships` and `students`.

### `academic_years`
- **Purpose**: Academic years scoped to a school (e.g., 2026-2027).
- **Key Columns**: `id`, `school_id` (FK), `name`, `start_date`, `end_date`, timestamps.
- **Relationships**: Many-to-one with `schools`; one-to-many with `academic_terms`, `class_rooms`, `student_class_enrollments`.

### `academic_terms`
- **Purpose**: Terms (semesters, quarters) within an academic year.
- **Key Columns**: `id`, `academic_year_id` (FK), `name`, `start_date`, `end_date`, timestamps.
- **Relationships**: Many-to-one with `academic_years`; referenced by `student_class_enrollments`, `class_timetable_slots`.

### `public_holidays`
- **Purpose**: National public holidays at the country level.
- **Key Columns**: `id`, `country_id` (FK), `name`, `month`, `day`, timestamps.
- **Relationships**: Many-to-one with `countries`.

### `school_events`
- **Purpose**: School-specific events (sports, admissions, exams, etc.).
- **Key Columns**: `id`, `school_id` (FK), `title`, `event_type`, `start_date`, `end_date`, `requires_attendance`, timestamps.
- **Relationships**: Many-to-one with `schools`; referenced by `event_attendance`.

### `timetable_templates`
- **Purpose**: Bell-schedule configurations (e.g., "Standard 6-Period Day").
- **Key Columns**: `id`, `school_id` (FK), `name`, `description`, timestamps.
- **Relationships**: Many-to-one with `schools`; one-to-many with `time_slots`.

### `time_slots`
- **Purpose**: Individual periods/breaks belonging to a timetable template.
- **Key Columns**: `id`, `school_id`, `timetable_template_id`, `name`, `start_time`, `end_time`, `sequence_index`, `is_instructional`, timestamps.
- **Relationships**: Many-to-one with `schools` and `timetable_templates`; referenced by `class_timetable_slots`.

### `rooms`
- **Purpose**: Physical facilities (lab, gym, classroom) to prevent double-booking.
- **Key Columns**: `id`, `school_id` (FK), `name`, `capacity`, `room_type` (enum), timestamps.
- **Relationships**: Many-to-one with `schools`; referenced by `class_timetable_slots`.

## Enums

| Enum | Values | Used By |
|------|--------|---------|
| `user_role` | `ADMIN`, `TEACHER`, `GUARDIAN`, `FINANCE` | `school_memberships.role` |
| `enrollment_status` | `ACTIVE`, `PROMOTED`, `REPEATING`, `GRADUATED` | `student_class_enrollments.status` |
| `substitution_status` | `PENDING`, `ASSIGNED`, `COMPLETED`, `CANCELLED` | `timetable_substitutions.status` |
| `room_type` | `STANDARD`, `SCIENCE_LAB`, `COMPUTER_LAB`, `GYM` | `rooms.room_type` |
| `timetable_attendance_status` | `PRESENT`, `ABSENT`, `LATE`, `EXCUSED` | `timetable_attendance.status` |
| `event_attendance_status` | `PRESENT`, `ABSENT`, `EXCUSED` | `event_attendance.status` |
| `student_gender` | `M`, `F`, `OTHER` | `students.gender` |

## Relationship Summary (ER-style)

```
tenants
  ├── users (1:N)
  │     ├── sessions (1:N)
  │     ├── members (1:N)
  │     └── school_memberships (1:N)
  └── schools (1:N)
        ├── streams (1:N)
        ├── grade_levels (via education_systems)
        ├── academic_years (1:N)
        │     └── academic_terms (1:N)
        ├── class_rooms (1:N)
        │     └── student_class_enrollments (1:N)
        │           └── students (N:1)
        ├── school_memberships (1:N)
        │     ├── guardian_student_links (1:N)
        │     │     └── students (N:1)
        │     ├── class_timetable_slots (teacher_membership_id)
        │     └── timetable_substitutions (teacher references)
        ├── subjects (via education_systems)
        │     ├── topics (1:N)
        │     │     └── sub_topics (1:N)
        │     └── class_timetable_slots (subject_id)
        ├── rooms (1:N)
        ├── school_events (1:N)
        │     └── event_attendance (1:N)
        │           └── students (N:1)
        ├── public_holidays (via countries)
        ├── bulk_jobs (1:N)
        │     └── bulk_job_items (1:N)
        └── student_gender_counts (1:1)
```

## Row-Level Security (RLS)

All tenant-scoped tables (`users`, `school_memberships`, `students`, `class_rooms`, `student_class_enrollments`, `timetable_*`, `attendance`, `bulk_jobs`, etc.) enforce multi-tenant isolation via RLS policies that compare `school_id`/`tenant_id` against the session variable `app.current_tenant_id`. This ensures queries only return rows belonging to the current tenant.

## Triggers

- `updated_at` columns are maintained via the shared `set_updated_at()` trigger on virtually every table.
- `sessions.last_seen_at` is refreshed via `update_last_seen()`.
- `student_gender_counts` are recomputed via `recompute_student_gender_counts_for_school()` on student `INSERT/UPDATE/DELETE`.