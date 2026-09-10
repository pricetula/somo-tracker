# Database Schema Reference

> **Purpose:** Plain-text description of all tables, columns, and relationships in the Somotracker PostgreSQL database.
> This document describes shape only — no SQL is included.

---

## Multi-Tenant Layer

These tables form the foundation of the multi-tenant identity system, backed by Row-Level Security (RLS).

### `tenants`
Maps 1:1 to a Stytch OIDC organization. All Somotracker data is scoped under exactly one tenant.

| Field          | Type       | Description                                                    |
|----------------|------------|----------------------------------------------------------------|
| `id`           | UUID (PK)  | Auto-generated primary key. Opaque — treat as internal.        |
| `name`         | VARCHAR    | Human-readable organization display name (e.g. "Acme Corp").  |
| `slug`         | VARCHAR    | URL-safe, lowercase, hyphen-separated identifier. Globally unique. Used for subdomains and admin routing. |
| `stytch_org_id`| VARCHAR    | Stytch OIDC organization ID. The authoritative identity anchor for SSO/SAML membership. Globally unique. |
| `created_at`   | TIMESTAMPTZ| UTC timestamp of row creation.                                 |

---

### `users`
Per-tenant user accounts. Rows are scoped to exactly one tenant via a foreign key. Users are identified by email within a tenant scope; the same email may appear in different tenants.

| Field              | Type       | Description                                                    |
|--------------------|------------|----------------------------------------------------------------|
| `id`               | UUID (PK)  | Auto-generated primary key. Opaque.                            |
| `email`            | VARCHAR    | Canonical email address. Lowercase enforced at the DB layer. Unique per tenant (not globally). |
| `tenant_id`        | UUID (FK)  | Foreign key to `tenants(id)`. Every user belongs to exactly one tenant. Deleting the tenant cascades this row. |
| `full_name`        | VARCHAR    | Display name chosen by the user. May be empty.                 |
| `is_active`        | BOOLEAN    | Soft-disable flag. Inactive users cannot authenticate but rows are retained for audit. |
| `external_auth_id` | VARCHAR    | Stytch user ID or equivalent auth-provider subject. Enables linking the local row to the external identity without querying by email. Unique. |
| `created_at`       | TIMESTAMPTZ| UTC timestamp of row creation.                                 |
| `updated_at`       | TIMESTAMPTZ| UTC timestamp of last modification (updated by application code, not triggers by default). |

RLS is enabled and forced on this table. All row-level operations are gated by comparing the row's `tenant_id` with the session's `app.current_tenant_id` GUC variable.

---

### `sessions`
Server-issued opaque session tokens. The raw Stytch session token is NOT stored here — only a local opaque token that maps to the cached Stytch token in Redis. This keeps the DB harmless even if leaked.

| Field              | Type       | Description                                                    |
|--------------------|------------|----------------------------------------------------------------|
| `id`               | UUID (PK)  | Auto-generated primary key.                                     |
| `token`            | VARCHAR    | Opaque 256-bit token, hex-encoded (64 chars). Never exposed in URLs or JavaScript. Unique. |
| `stytch_session_id`| VARCHAR    | Opaque Stytch session ID used for revocation and validation against Stytch. |
| `user_id`          | UUID (FK)  | Foreign key to `users(id)`. Deleting the user cascades all sessions. |
| `tenant_id`        | UUID (FK)  | Foreign key to `tenants(id)`. Used for fast tenant-scoped session lookup. |
| `expires_at`       | TIMESTAMPTZ| Rolling expiry timestamp. Default 7 days; refreshed on each valid request. |
| `created_at`       | TIMESTAMPTZ| UTC timestamp of session creation.                             |
| `last_seen_at`     | TIMESTAMPTZ| UTC timestamp of last validated request. Auto-updated via trigger on row update. Used for rolling expiry refresh. |

---

### `members`
Mirrors Stytch's B2B Member model. Created and updated atomically with the `users` row during magic-link provisioning. The member is the B2B identity; the user row is the Somotracker application identity.

| Field               | Type       | Description                                                    |
|---------------------|------------|----------------------------------------------------------------|
| `id`                | UUID (PK)  | Auto-generated primary key.                                     |
| `stytch_member_id`  | VARCHAR    | Stytch B2B Member.member_id. Unique across the entire system. |
| `user_id`           | UUID (FK)  | Foreign key to `users(id)`. The Somotracker application identity. |
| `tenant_id`         | UUID (FK)  | Foreign key to `tenants(id)`. The B2B organization.            |
| `stytch_member_raw` | JSONB      | Cached Stytch member object for audit and debugging. Sensitive metadata fields are stripped before storage. |
| `created_at`        | TIMESTAMPTZ| UTC timestamp of member creation.                              |
| `updated_at`        | TIMESTAMPTZ| UTC timestamp of last modification. Auto-updated via trigger.  |

---

## School Information System (SIS) Layer

### Enums

- **`user_role`** — Role for `school_memberships`: `ADMIN`, `TEACHER`, `GUARDIAN`, `FINANCE`.
- **`enrollment_status`** — Student enrollment lifecycle: `ACTIVE`, `PROMOTED`, `REPEATING`, `GRADUATED`.
- **`substitution_status`** — Substitution lifecycle: `PENDING`, `ASSIGNED`, `COMPLETED`, `CANCELLED`.
- **`room_type`** — Physical room category: `STANDARD`, `SCIENCE_LAB`, `COMPUTER_LAB`, `GYM`.
- **`timetable_attendance_status`** — Attendance state for timetable-linked lessons: `PRESENT`, `ABSENT`, `LATE`, `EXCUSED`.
- **`event_attendance_status`** — Attendance state for school-wide events: `PRESENT`, `ABSENT`, `EXCUSED`.

---

### Reference Tables

These tables are shared, non-tenant-scoped lookups used throughout the SIS.

#### `countries`
Reference table of countries supported by the SIS. Country codes follow ISO 3166-1 alpha-2.

| Field           | Type       | Description                                        |
|-----------------|------------|----------------------------------------------------|
| `id`            | UUID (PK)  | Auto-generated primary key.                        |
| `country_name`  | VARCHAR    | Human-readable country name (e.g. "Kenya").        |
| `country_code`  | VARCHAR    | ISO 3166-1 alpha-2 code (e.g. "KE"). Unique.      |
| `created_at`    | TIMESTAMPTZ| UTC timestamp of row creation.                     |
| `updated_at`    | TIMESTAMPTZ| UTC timestamp of last modification.                |

#### `education_systems`
Reference table of education systems (e.g. Competency-Based Education / CBE). Scoped to a country.

| Field           | Type       | Description                                        |
|-----------------|------------|----------------------------------------------------|
| `id`            | UUID (PK)  | Auto-generated primary key.                        |
| `country_id`    | UUID (FK)  | Foreign key to `countries(id)`. Cascades on delete. |
| `system_name`   | VARCHAR    | Human-readable system name.                        |
| `description`   | TEXT       | Optional free-text description of the system.      |
| `created_at`    | TIMESTAMPTZ| UTC timestamp of row creation.                     |
| `updated_at`    | TIMESTAMPTZ| UTC timestamp of last modification.                |

#### `grade_levels`
Grade levels scoped to a specific education system and country. A grade level's sequence position must be unique within a system+country to guarantee deterministic chronological ordering.

| Field                | Type       | Description                                                           |
|----------------------|------------|-----------------------------------------------------------------------|
| `id`                 | UUID (PK)  | Auto-generated primary key.                                           |
| `education_system_id`| UUID (FK)  | Foreign key to `education_systems(id)`. Cascades on delete.           |
| `country_id`         | UUID (FK)  | Foreign key to `countries(id)`. Cascades on delete.                   |
| `tier_stage`         | VARCHAR    | Broad stage bucket: `pre_primary`, `primary`, `lower_secondary`, `upper_secondary`. |
| `local_label`         | VARCHAR    | Local label used in that country/system (e.g. "PP1", "Grade 7", "Senior 1"). |
| `sequence_index`     | INTEGER    | Integer for chronological sorting. Unique within education_system + country. |
| `created_at`         | TIMESTAMPTZ| UTC timestamp of row creation.                                        |
| `updated_at`         | TIMESTAMPTZ| UTC timestamp of last modification.                                   |

---

### Operational Tables

#### `schools`
Schools operated by a tenant. Each school belongs to exactly one tenant, one country, and one education system.

| Field                 | Type       | Description                                                           |
|-----------------------|------------|-----------------------------------------------------------------------|
| `id`                  | UUID (PK)  | Auto-generated primary key.                                           |
| `tenant_id`           | UUID (FK)  | Foreign key to `tenants(id)`. Cascades on tenant delete.              |
| `school_name`         | VARCHAR    | Human-readable school name.                                           |
| `country_id`          | UUID (FK)  | Foreign key to `countries(id)`. Cascades on delete.                   |
| `education_system_id` | UUID (FK)  | Foreign key to `education_systems(id)`. Cascades on delete.           |
| `created_at`          | TIMESTAMPTZ| UTC timestamp of row creation.                                        |
| `updated_at`          | TIMESTAMPTZ| UTC timestamp of last modification.                                   |

#### `school_memberships`
Links a user to a school with a role. A user may have at most one membership record per school (enforced by a unique constraint on `school_id` + `user_id`).

| Field       | Type       | Description                                                     |
|-------------|------------|----------------------------------------------------------------|
| `id`        | UUID (PK)  | Auto-generated primary key.                                     |
| `school_id` | UUID (FK)  | Foreign key to `schools(id)`. Cascades on school delete.        |
| `user_id`   | UUID (FK)  | Foreign key to `users(id)`. Cascades on user delete.           |
| `role`      | user_role  | Role within the school: `ADMIN`, `TEACHER`, `GUARDIAN`, `FINANCE`. |
| `is_active` | BOOLEAN    | Soft-disable flag. Defaults to `FALSE`.                        |
| `created_at`| TIMESTAMPTZ| UTC timestamp of row creation.                                  |
| `updated_at`| TIMESTAMPTZ| UTC timestamp of last modification.                             |

#### `students`
Student records scoped to a school. The `admission_number` is unique within a school.

| Field             | Type       | Description                                                           |
|-------------------|------------|-----------------------------------------------------------------------|
| `student_id`      | UUID (PK)  | Auto-generated primary key.                                           |
| `school_id`       | UUID (FK)  | Foreign key to `schools(id)`. Cascades on school delete.              |
| `admission_number`| VARCHAR    | School-scoped admission number. Unique per school.                    |
| `full_name`       | VARCHAR    | Student full name (first and last).                                  |
| `date_of_birth`   | DATE       | Date of birth.                                                        |
| `gender`          | VARCHAR    | Gender (free-text for international flexibility).                     |
| `metadata`        | JSONB      | Flexible external identifiers (e.g. NEMIS, KICD tracking codes). Defaults to an empty object. |
| `created_at`      | TIMESTAMPTZ| UTC timestamp of row creation.                                        |
| `updated_at`      | TIMESTAMPTZ| UTC timestamp of last modification.                                    |

#### `guardian_student_links`
Links a guardian (via their `school_membership`) to a student with a relationship type. Enforces that a guardian cannot be linked to the same student twice via the same membership record.

| Field                  | Type       | Description                                                           |
|------------------------|------------|-----------------------------------------------------------------------|
| `id`                   | UUID (PK)  | Auto-generated primary key.                                           |
| `school_membership_id` | UUID (FK)  | Foreign key to `school_memberships(id)`. Cascades on membership delete. |
| `student_id`           | UUID (FK)  | Foreign key to `students(student_id)`. Cascades on student delete.    |
| `relationship_type`     | VARCHAR    | Relationship type (e.g. "Parent", "Legal Guardian", "Sponsor").       |
| `created_at`           | TIMESTAMPTZ| UTC timestamp of row creation.                                        |
| `updated_at`           | TIMESTAMPTZ| UTC timestamp of last modification.                                    |

---

## Curriculum Layer (Subject Hierarchy)

#### `subjects`
Subjects offered within a specific education system. A subject code must be unique within an education system.

| Field                | Type       | Description                                                      |
|----------------------|------------|------------------------------------------------------------------|
| `id`                 | UUID (PK)  | Auto-generated primary key.                                      |
| `education_system_id`| UUID (FK)  | Foreign key to `education_systems(id)`. Cascades on delete.     |
| `name`               | VARCHAR    | Human-readable subject name (e.g. "Mathematics").               |
| `code`               | VARCHAR    | Short subject code (e.g. "MAT", "ENG"). Unique within an education system. |
| `type`               | VARCHAR    | Subject type: "Core", "Optional", "Elective", etc.              |
| `created_at`         | TIMESTAMPTZ| UTC timestamp of row creation.                                  |
| `updated_at`         | TIMESTAMPTZ| UTC timestamp of last modification.                             |

#### `topics`
Topics within a subject, ordered chronologically by `sequence_index`. A topic's sequence position must be unique within a subject.

| Field           | Type       | Description                                                      |
|-----------------|------------|------------------------------------------------------------------|
| `id`            | UUID (PK)  | Auto-generated primary key.                                      |
| `subject_id`    | UUID (FK)  | Foreign key to `subjects(id)`. Cascades on subject delete.       |
| `name`          | VARCHAR    | Topic name (e.g. "Fractions and Decimals").                     |
| `sequence_index`| INTEGER    | Integer for chronological sorting. Unique within a subject.     |
| `created_at`    | TIMESTAMPTZ| UTC timestamp of row creation.                                   |
| `updated_at`    | TIMESTAMPTZ| UTC timestamp of last modification.                              |

#### `sub_topics`
Sub-topics within a topic, ordered by `sequence_index`. A sub-topic's sequence position must be unique within a topic.

| Field           | Type       | Description                                                      |
|-----------------|------------|------------------------------------------------------------------|
| `id`            | UUID (PK)  | Auto-generated primary key.                                      |
| `topic_id`      | UUID (FK)  | Foreign key to `topics(id)`. Cascades on topic delete.           |
| `name`          | VARCHAR    | Sub-topic name (e.g. "Addition of Fractions").                   |
| `sequence_index`| INTEGER    | Integer for chronological sorting. Unique within a topic.        |
| `created_at`    | TIMESTAMPTZ| UTC timestamp of row creation.                                   |
| `updated_at`    | TIMESTAMPTZ| UTC timestamp of last modification.                              |

---

## Academic Calendar Layer

#### `academic_years`
Academic years scoped to a school. A school may have multiple academic years over time. The `name` is a human-readable identifier; `start_date`/`end_date` define the calendar boundaries.

| Field        | Type       | Description                                                      |
|--------------|------------|------------------------------------------------------------------|
| `id`         | UUID (PK)  | Auto-generated primary key.                                      |
| `school_id`  | UUID (FK)  | Foreign key to `schools(id)`. Cascades on school delete.         |
| `name`       | VARCHAR    | Human-readable year identifier (e.g. "2026", "2026-2027"). Unique per school. |
| `start_date` | DATE       | First day of the academic year.                                  |
| `end_date`   | DATE       | Last day of the academic year.                                    |
| `created_at` | TIMESTAMPTZ| UTC timestamp of row creation.                                   |
| `updated_at` | TIMESTAMPTZ| UTC timestamp of last modification.                              |

#### `academic_terms`
Academic terms (semester, quarter, term 1/2/3) scoped to an academic year. Each term has its own date range for scheduling and grading purposes. Term name is unique within an academic year.

| Field              | Type       | Description                                                      |
|--------------------|------------|------------------------------------------------------------------|
| `id`               | UUID (PK)  | Auto-generated primary key.                                      |
| `academic_year_id` | UUID (FK)  | Foreign key to `academic_years(id)`. Cascades on academic year delete. |
| `name`             | VARCHAR    | Term name (e.g. "Term 1", "Semester 1"). Unique within an academic year. |
| `start_date`       | DATE       | First day of the term.                                           |
| `end_date`         | DATE       | Last day of the term.                                            |
| `created_at`       | TIMESTAMPTZ| UTC timestamp of row creation.                                   |
| `updated_at`       | TIMESTAMPTZ| UTC timestamp of last modification.                              |

#### `public_holidays`
Public holidays declared at the country level. These apply nationwide (e.g. Madaraka Day, Jamhuri Day, Labour Day). Stored as `month`/`day` to handle recurring annual holidays.

| Field        | Type       | Description                                                      |
|--------------|------------|------------------------------------------------------------------|
| `id`         | UUID (PK)  | Auto-generated primary key.                                      |
| `country_id` | UUID (FK)  | Foreign key to `countries(id)`. Cascades on country delete.       |
| `name`       | VARCHAR    | Holiday name (e.g. "Madaraka Day", "Labour Day").               |
| `month`       | SMALLINT   | Month of the holiday (1–12).                             |
| `day`         | SMALLINT   | Day of the month (1–31).                                |
| `created_at`  | TIMESTAMPTZ| UTC timestamp of row creation.                          |
| `updated_at`  | TIMESTAMPTZ| UTC timestamp of last modification.                     |

#### `school_events`
School-specific events such as sports days, admission days, exams, etc. Events may span multiple days and may optionally require student attendance tracking.

| Field                | Type       | Description                                                      |
|----------------------|------------|------------------------------------------------------------------|
| `id`                 | UUID (PK)  | Auto-generated primary key.                                      |
| `school_id`          | UUID (FK)  | Foreign key to `schools(id)`. Cascades on school delete.         |
| `title`              | VARCHAR    | Event title (e.g. "Inter-House Sports Day").                     |
| `event_type`         | VARCHAR    | Event type: "SPORTS", "ADMISSION", "EXAM", etc.                  |
| `start_date`         | DATE       | First day of the event.                                          |
| `end_date`           | DATE       | Last day of the event.                                           |
| `requires_attendance`| BOOLEAN    | Whether student attendance must be tracked for this event. Defaults to `FALSE`. |
| `created_at`         | TIMESTAMPTZ| UTC timestamp of row creation.                                   |
| `updated_at`         | TIMESTAMPTZ| UTC timestamp of last modification.                              |

---

## Classrooms & Enrollments Layer

### `streams`
School-scoped stream definitions (e.g. "Blue", "Yellow", "A", "B") that may be referenced by class rooms. Created per school; stream names must be unique within a school.

| Field      | Type       | Description                                                      |
|------------|------------|------------------------------------------------------------------|
| `id`       | UUID (PK)  | Auto-generated primary key.                                      |
| `school_id`| UUID (FK)  | Foreign key to `schools(id)`. Cascades on school delete.         |
| `name`     | VARCHAR(64)| Stream identifier (e.g. "Blue"). Unique per school.             |
| `color`    | VARCHAR(32)| Optional display color (e.g. hex code, named color).             |
| `created_at`| TIMESTAMPTZ| UTC timestamp of row creation.                                   |
| `updated_at`| TIMESTAMPTZ| UTC timestamp of last modification.                               |

**Constraints / Indexes (from SQL):**
- `UNIQUE (school_id, name)` (`streams_school_name_uniq`).
- Index `streams_school_id_idx`.

---

#### `class_rooms`
Operational classroom container for a specific academic year and stream. Represents a grade-level + stream combination (e.g. "Class 1 Blue", "Class 3 Yellow") within a given academic year. Each year creates new `class_room` entries, preserving historical context for past enrollments.

| Field              | Type       | Description                                                      |
|--------------------|------------|------------------------------------------------------------------|
| `id`               | UUID (PK)  | Auto-generated primary key.                                      |
| `school_id`        | UUID (FK)  | Foreign key to `schools(id)`. Cascades on school delete.         |
| `academic_year_id` | UUID (FK)  | Foreign key to `academic_years(id)`. Links the class_room to a specific academic year. Cascades on delete. |
| `grade_level_id`   | UUID (FK)  | Foreign key to `grade_levels(id)`. The grade level for this class_room (e.g. "Grade 1", "Grade 3"). Cascades on delete. |
| `name`             | VARCHAR    | Human-readable class_room name (e.g. "Class 1 Blue", "Grade 3 Yellow"). |
| `stream`           | VARCHAR    | Optional stream identifier within the grade (e.g. "Blue", "Yellow", "A", "B"). NULL if no streams. |
| `created_at`       | TIMESTAMPTZ| UTC timestamp of row creation.                                   |
| `updated_at`       | TIMESTAMPTZ| UTC timestamp of last modification.                              |

The `(school_id, academic_year_id, grade_level_id, NULLIF(stream, ''))` combination is unique — a school cannot have duplicate class_rooms for the same year/grade/stream combination.

#### `student_class_enrollments`
Historical mapping of a student to a class_room for a specific academic term or year. Each enrollment is immutable once status reaches a terminal state (`GRADUATED`), preserving attendance registers, assessments, and report cards permanently.

| Field              | Type             | Description                                                           |
|--------------------|------------------|-----------------------------------------------------------------------|
| `id`               | UUID (PK)        | Auto-generated primary key.                                           |
| `school_id`        | UUID (FK)        | Foreign key to `schools(id)`. Enforces multi-tenant isolation via RLS. Cascades on school delete. |
| `student_id`       | UUID (FK)        | Foreign key to `students(student_id)`. The enrolled student. Cascades on student delete. |
| `class_room_id`    | UUID (FK)        | Foreign key to `class_rooms(id)`. The operational class_room for this enrollment. Cascades on delete. |
| `academic_year_id` | UUID (FK)        | Foreign key to `academic_years(id)`. The academic year of enrollment. Cascades on delete. |
| `academic_term_id` | UUID (FK, NULL)  | Foreign key to `academic_terms(id)`. Optional; NULL for year-level enrollments (e.g. final year without term splits). Cascades on delete. |
| `status`           | enrollment_status| Enrollment lifecycle: `ACTIVE`, `PROMOTED`, `REPEATING`, `GRADUATED`. Defaults to `ACTIVE`. Controls promotion, repetition, and graduation workflows. |
| `enrolled_at`      | TIMESTAMPTZ      | UTC timestamp when the student was enrolled in this class_room. Defaults to NOW(). |
| `completed_at`     | TIMESTAMPTZ      | UTC timestamp when enrollment reached a terminal status (`PROMOTED`, `REPEATING`, `GRADUATED`). NULL for `ACTIVE`. |
| `metadata`         | JSONB            | Flexible enrollment metadata (e.g. previous school, transfer notes). Defaults to an empty object. |
| `created_at`       | TIMESTAMPTZ      | UTC timestamp of row creation.                                        |
| `updated_at`       | TIMESTAMPTZ      | UTC timestamp of last modification.                                   |

RLS is enabled and forced on this table. All row-level operations are gated by a subquery against `schools` validating that the school's `tenant_id` matches the session's `app.current_tenant_id` GUC variable.

---

## Timetable & Scheduling Layer

### `timetable_templates`
Parent container for a distinct bell schedule configuration per school. A school can define multiple schedule patterns (e.g. "Standard 6-Period Day", "Morning Shift 3-Lesson", "Half-Day Schedule").

| Field                | Type       | Description                                                      |
|----------------------|------------|------------------------------------------------------------------|
| `id`                 | UUID (PK)  | Auto-generated primary key.                                      |
| `school_id`          | UUID (FK)  | Foreign key to `schools(id)`. Cascades on school delete.         |
| `name`               | VARCHAR(255) | Human-readable template name (e.g. "Standard 6-Period Day"). Unique per school. |
| `description`        | TEXT       | Optional details about when or who uses this template.            |
| `created_at`         | TIMESTAMPTZ| UTC timestamp of row creation.                                   |
| `updated_at`         | TIMESTAMPTZ| UTC timestamp of last modification.                               |

---

### `time_slots`
Individual periods or breaks belonging to a timetable template. Each slot represents a fixed time window (e.g. "Period 1" 08:00–08:40, "Morning Break" 10:00–10:15) and can be instructional (`is_instructional`) or a non-instructional break.

| Field                | Type       | Description                                                      |
|----------------------|------------|------------------------------------------------------------------|
| `id`                 | UUID (PK)  | Auto-generated primary key.                                      |
| `school_id`          | UUID (FK)  | Foreign key to `schools(id)`. Cascades on school delete.         |
| `timetable_template_id` | UUID (FK) | Foreign key to `timetable_templates(id)`. Cascades on delete.    |
| `name`               | VARCHAR(255) | Slot label (e.g. "Period 1", "Morning Break").                  |
| `start_time`         | TIME       | Slot start time (e.g. 08:00:00).                                 |
| `end_time`           | TIME       | Slot end time (e.g. 08:40:00). Must be after `start_time`.       |
| `sequence_index`     | INTEGER    | Order of the slot within the template. Unique per template.      |
| `is_instructional`   | BOOLEAN    | `TRUE` for instructional periods; `FALSE` for breaks/recess.     |
| `created_at`         | TIMESTAMPTZ| UTC timestamp of row creation.                                   |
| `updated_at`         | TIMESTAMPTZ| UTC timestamp of last modification.                               |

---

### `rooms`
Physical facilities and campus locations to prevent room overbooking. Distinct from `class_rooms` (operational containers per year/stream) — a `room` is a concrete space (e.g. "Lab A", "Room 204", "Gymnasium").

| Field                | Type       | Description                                                      |
|----------------------|------------|------------------------------------------------------------------|
| `id`                 | UUID (PK)  | Auto-generated primary key.                                      |
| `school_id`          | UUID (FK)  | Foreign key to `schools(id)`. Cascades on school delete.         |
| `name`               | VARCHAR(255) | Room identifier or name (e.g. "Lab A"). Unique per school.      |
| `capacity`           | INTEGER    | Maximum student capacity the room can hold. Optional (nullable). |
| `room_type`          | room_type  | Category: `STANDARD`, `SCIENCE_LAB`, `COMPUTER_LAB`, `GYM`.      |
| `created_at`         | TIMESTAMPTZ| UTC timestamp of row creation.                                   |
| `updated_at`         | TIMESTAMPTZ| UTC timestamp of last modification.                               |

---

### `class_timetable_slots`
Maps a `class_room` to an academic term, assigning a subject, teacher (`school_membership`), and optional physical `room` to a `time_slot` for a specific day of the week. This is the master weekly recurring timetable.

| Field                     | Type       | Description                                                      |
|---------------------------|------------|------------------------------------------------------------------|
| `id`                      | UUID (PK)  | Auto-generated primary key.                                      |
| `school_id`               | UUID (FK)  | Foreign key to `schools(id)`. Cascades on school delete.         |
| `class_room_id`           | UUID (FK)  | Foreign key to `class_rooms(id)`. Cascades on delete.             |
| `academic_term_id`        | UUID (FK)  | Foreign key to `academic_terms(id)`. Cascades on delete.         |
| `day_of_week`              | INTEGER    | Day index: `1` = Monday through `7` = Sunday.                    |
| `time_slot_id`             | UUID (FK)  | Foreign key to `time_slots(id)`. Cascades on delete.             |
| `subject_id`               | UUID (FK)  | Foreign key to `subjects(id)`. Cascades on delete.               |
| `teacher_membership_id`    | UUID (FK)  | Foreign key to `school_memberships(id)`. The assigned teacher. Cascades on delete. |
| `room_id`                  | UUID (FK)  | Foreign key to `rooms(id)`. Optional physical location. Cascades on delete (set NULL). |
| `created_at`               | TIMESTAMPTZ| UTC timestamp of row creation.                                   |
| `updated_at`               | TIMESTAMPTZ| UTC timestamp of last modification.                               |

**Constraints / Indexes (from SQL):**
- `UNIQUE (teacher_membership_id, academic_term_id, day_of_week, time_slot_id)` — a teacher cannot be double-booked in the same slot across classes in the same term (`class_timetable_slots_teacher_no_clash`).
- `UNIQUE (class_room_id, academic_term_id, day_of_week, time_slot_id)` — a class room has at most one subject per time slot per day (`class_timetable_slots_class_day_time_uniq`).
- RLS enabled and forced; tenant isolation via `schools.tenant_id`.

---

### `timetable_substitutions`
Handles emergency or planned teacher absences on specific calendar dates without modifying the master weekly recurring timetable (`class_timetable_slots`). A substitution overrides a single timetable slot instance on a given date.

| Field                        | Type              | Description                                                      |
|------------------------------|-------------------|------------------------------------------------------------------|
| `id`                         | UUID (PK)         | Auto-generated primary key.                                      |
| `school_id`                  | UUID (FK)         | Foreign key to `schools(id)`. Cascades on school delete.         |
| `class_timetable_slot_id`    | UUID (FK)         | Foreign key to `class_timetable_slots(id)`. The overridden slot. Cascades on delete. |
| `substitution_date`           | DATE              | The precise calendar date of the absence/coverage.               |
| `original_teacher_membership_id` | UUID (FK)     | The teacher who is away (`school_memberships.id`). Cascades on delete. |
| `substitute_teacher_membership_id` | UUID (FK)  | The covering teacher. Optional (`NULL` if unassigned). Cascades on delete (set NULL). |
| `status`                     | `substitution_status` | `PENDING`, `ASSIGNED`, `COMPLETED`, `CANCELLED`. Default `PENDING`. |
| `reason`                     | TEXT              | Optional explanation (e.g. "Medical leave", "Training").          |
| `created_at`                 | TIMESTAMPTZ       | UTC timestamp of row creation.                                   |
| `updated_at`                 | TIMESTAMPTZ       | UTC timestamp of last modification.                               |

**Constraints / Indexes (from SQL):**
- `UNIQUE (class_timetable_slot_id, substitution_date)` — only one substitution per slot per date (`timetable_substitutions_slot_date_uniq`).
- RLS enabled and forced; tenant isolation via `schools.tenant_id`.

---

## Attendance Tracking Layer

### `timetable_attendance`
Primary attendance tracking table linking to specific timetable slot instances on calendar dates. A subject teacher records presence (`PRESENT`, `ABSENT`, `LATE`, `EXCUSED`) for each student during an instructional period (`is_instructional = TRUE`).

| Field                     | Type                  | Description                                                      |
|---------------------------|-----------------------|------------------------------------------------------------------|
| `id`                      | UUID (PK)             | Auto-generated primary key.                                      |
| `school_id`               | UUID (FK)             | Foreign key to `schools(id)`. Cascades on school delete.         |
| `student_id`              | UUID (FK)             | Foreign key to `students(student_id)`. Cascades on delete.      |
| `class_timetable_slot_id` | UUID (FK)             | Foreign key to `class_timetable_slots(id)`. Cascades on delete.  |
| `attendance_date`         | DATE                  | The calendar date of the lesson instance.                        |
| `status`                  | `timetable_attendance_status` | `PRESENT`, `ABSENT`, `LATE`, `EXCUSED`.               |
| `remarks`                 | TEXT                  | Optional notes (e.g. "Left early due to illness").             |
| `recorded_by_membership_id` | UUID (FK)          | Foreign key to `school_memberships(id)`. The recording teacher. Cascades on delete. |
| `created_at`              | TIMESTAMPTZ           | UTC timestamp of row creation.                                   |
| `updated_at`              | TIMESTAMPTZ           | UTC timestamp of last modification.                               |

**Constraints / Indexes (from SQL):**
- `UNIQUE (student_id, class_timetable_slot_id, attendance_date)` — one attendance entry per student per slot per date (`timetable_attendance_uniq_student_slot_date`).
- RLS enabled and forced; tenant isolation via `schools.tenant_id`.

---

### `event_attendance`
Attendance tracking for special school-wide activities (sports days, symposia, admission days) where the regular timetable is suspended. Links to `school_events` rather than `class_timetable_slots`.

| Field                | Type              | Description                                                      |
|----------------------|-------------------|------------------------------------------------------------------|
| `id`                 | UUID (PK)         | Auto-generated primary key.                                      |
| `school_id`          | UUID (FK)         | Foreign key to `schools(id)`. Cascades on school delete.         |
| `student_id`         | UUID (FK)         | Foreign key to `students(student_id)`. Cascades on delete.       |
| `school_event_id`    | UUID (FK)         | Foreign key to `school_events(id)`. Cascades on delete.          |
| `attendance_date`    | DATE              | The calendar date of the event.                                  |
| `status`             | `event_attendance_status` | `PRESENT`, `ABSENT`, `EXCUSED`.                        |
| `remarks`            | TEXT              | Optional details about event attendance.                          |
| `created_at`         | TIMESTAMPTZ       | UTC timestamp of row creation.                                   |
| `updated_at`         | TIMESTAMPTZ       | UTC timestamp of last modification.                               |

**Constraints / Indexes (from SQL):**
- `UNIQUE (student_id, school_event_id, attendance_date)` — one attendance entry per student per event per date (`event_attendance_uniq_student_event_date`).
- RLS enabled and forced; tenant isolation via `schools.tenant_id`.

---

## Summary of Relationships

```
tenants
  └── users            (FK tenant_id → tenants.id, RLS-isolated)
        ├── sessions   (FK user_id → users.id, FK tenant_id → tenants.id)
        ├── members    (FK user_id → users.id, FK tenant_id → tenants.id)
        └── school_memberships (FK user_id → users.id)
              └── guardian_student_links (FK school_membership_id → school_memberships.id)
                    └── students (FK student_id → students.student_id)

schools               (FK tenant_id → tenants.id, FK country_id → countries.id, FK education_system_id → education_systems.id)
  ├── streams         (FK school_id → schools.id)
  ├── students        (FK school_id → schools.id)
  ├── academic_years  (FK school_id → schools.id)
  │     └── academic_terms (FK academic_year_id → academic_years.id)
  ├── school_events   (FK school_id → schools.id)
  │     └── event_attendance (FK school_event_id → school_events.id, FK student_id → students.student_id, RLS-isolated)
  ├── class_rooms     (FK school_id → schools.id, FK academic_year_id → academic_years.id, FK grade_level_id → grade_levels.id)
  │     └── student_class_enrollments (FK class_room_id → class_rooms.id, FK student_id → students.student_id, FK academic_year_id → academic_years.id, FK academic_term_id → academic_terms.id, RLS-isolated)
  ├── timetable_templates (FK school_id → schools.id, RLS-isolated)
  │     └── time_slots (FK timetable_template_id → timetable_templates.id, RLS-isolated)
  ├── rooms          (FK school_id → schools.id, RLS-isolated)
  └── class_timetable_slots (FK class_room_id → class_rooms.id, FK academic_term_id → academic_terms.id, FK time_slot_id → time_slots.id, FK subject_id → subjects.id, FK teacher_membership_id → school_memberships.id, FK room_id → rooms.id, RLS-isolated)
        ├── timetable_attendance (FK class_timetable_slot_id → class_timetable_slots.id, FK student_id → students.student_id, RLS-isolated)
        └── timetable_substitutions (FK class_timetable_slot_id → class_timetable_slots.id, FK original_teacher_membership_id → school_memberships.id, FK substitute_teacher_membership_id → school_memberships.id, RLS-isolated)

countries
  ├── grade_levels    (FK country_id → countries.id, FK education_system_id → education_systems.id)
  └── public_holidays (FK country_id → countries.id)

education_systems
  ├── grade_levels    (FK education_system_id → education_systems.id)
  └── subjects        (FK education_system_id → education_systems.id)
        └── topics    (FK subject_id → subjects.id)
              └── sub_topics (FK topic_id → topics.id)
```

---

## Cross-Cutting Conventions

- **Primary keys** are auto-generated UUIDs via `gen_random_uuid()` (from the `pgcrypto` extension). Treat as opaque.
- **Timestamps** are stored as `TIMESTAMPTZ` in UTC. All tables include `created_at` and (except `sessions`) `updated_at`. The `set_updated_at()` trigger maintains `updated_at` on row updates for all SIS, curriculum, calendar, classroom, timetable, and attendance tables.
- **Cascade behavior:** Foreign keys cascade on delete unless otherwise noted (e.g. `members` and `sessions` cascade from `users` and `tenants`).
- **Multi-tenant isolation** is enforced via RLS on `users`, `student_class_enrollments`, `timetable_templates`, `time_slots`, `rooms`, `class_timetable_slots`, `timetable_attendance`, and `event_attendance`. All application transactions run inside `database.WithTenantTx`, which sets `app.current_tenant_id` via `SET LOCAL`. Raw queries outside a tenant-scoped transaction return zero rows (fail-closed).
- **Unique constraints** typically combine the parent FK with a business identifier (e.g. `(school_id, admission_number)`, `(education_system_id, code)`, `(tenant_id, email)`) to enforce scoping without relying on global uniqueness.
- **Soft-delete** is used only on `users.is_active`; other tables use hard deletes with cascading FKs.