# Somotracker Backend SQL Schema

This document describes the PostgreSQL schema used by the Somotracker backend. It is derived from the migration files in `backend/db/migrations/`. Tables are multi-tenant with Row-Level Security enforced via `app.current_tenant_id` session variable.

## Enums

### user_role
Role enum for school_memberships: ADMIN, TEACHER, GUARDIAN, FINANCE.

### enrollment_status
Student enrollment lifecycle status: ACTIVE (currently enrolled), PROMOTED (moved to next grade), REPEATING (retained in same grade), GRADUATED (completed final grade / alumni).

### substitution_status
Lifecycle state for substitutions: PENDING (awaiting assignment), ASSIGNED (cover teacher confirmed), COMPLETED (coverage done), CANCELLED (substitution revoked).

### room_type
Category of room: STANDARD (regular classroom), SCIENCE_LAB, COMPUTER_LAB, GYM.

### timetable_attendance_status
Attendance state for timetable-linked lessons: PRESENT, ABSENT, LATE, or EXCUSED.

### event_attendance_status
Attendance state for special school events: PRESENT, ABSENT, or EXCUSED.

## Core Tenancy & Auth

### tenants
Maps 1:1 to a Stytch OIDC organization. All Somotracker data is scoped under exactly one tenant row.

| Column | Type | Comment |
|--------|------|---------|
| id | UUID PK | Auto-generated UUID primary key. No external meaning — treat as opaque. |
| name | VARCHAR(255) NOT NULL | Human-readable organization display name (e.g. "Acme Corp"). |
| slug | VARCHAR(255) NOT NULL UNIQUE | URL-safe, lowercase, hyphen-separated identifier. Used for subdomains and admin routing. Must be globally unique. |
| stytch_org_id | VARCHAR(255) NOT NULL UNIQUE | The Stytch OIDC organization ID. This is the authoritative identity anchor for SSO / SAML membership. Unique. |
| created_at | TIMESTAMPTZ NOT NULL | UTC timestamp of row creation. |

Relations: parents of users, schools, sessions.

### users
Per-tenant user accounts. Rows are scoped to exactly one tenant via the foreign key on tenant_id. Users are identified by email within a tenant scope; the same email may appear in different tenants.

| Column | Type | Comment |
|--------|------|---------|
| id | UUID PK | Auto-generated UUID primary key. No external meaning — treat as opaque. |
| email | VARCHAR(255) NOT NULL | Canonical email address. Lowercase (enforced by CHECK), unique within a tenant but not globally via users_tenant_email_uniq. |
| tenant_id | UUID NOT NULL FK tenants(id) ON DELETE CASCADE | Foreign key to tenants(id). Every user must belong to exactly one tenant. Deleting the tenant cascades this row. |
| full_name | VARCHAR(255) NOT NULL DEFAULT '' | Display name chosen by the user. May be empty. |
| is_active | BOOLEAN NOT NULL DEFAULT TRUE | Soft-disable flag. Inactive users cannot authenticate but their rows are retained for audit purposes. |
| external_auth_id | VARCHAR(255) UNIQUE | Stytch user ID or equivalent auth-provider subject. Enables linking the local row to the external identity without querying by email. |
| created_at | TIMESTAMPTZ NOT NULL | UTC timestamp of row creation. |
| updated_at | TIMESTAMPTZ NOT NULL | UTC timestamp of last modification (updated by application code, not triggers by default). |

RLS: users_tenant_isolation on tenant_id = current_setting('app.current_tenant_id').

Relations: tenant_id → tenants, users 1:* sessions, members, school_memberships, bulk_jobs(created_by).

### sessions
Server-issued opaque session tokens. Tokens are stored in HttpOnly cookies; the raw Stytch session token is cached only in Redis.

| Column | Type | Comment |
|--------|------|---------|
| id | UUID PK | Auto-generated UUID primary key. |
| token | VARCHAR(64) NOT NULL UNIQUE | Opaque 256-bit token, hex-encoded. Never exposed to JavaScript or URLs. |
| stytch_session_id | VARCHAR(255) NOT NULL | Opaque Stytch session ID used for revocation / validation against Stytch. |
| user_id | UUID NOT NULL FK users(id) ON DELETE CASCADE | FK to users(id). Deleting the user cascades all sessions. |
| tenant_id | UUID NOT NULL FK tenants(id) ON DELETE CASCADE | FK to tenants(id). Used for fast tenant-scoped session lookup. |
| expires_at | TIMESTAMPTZ NOT NULL | Rolling expiry timestamp. Default 7 days. |
| created_at | TIMESTAMPTZ NOT NULL | UTC timestamp of session creation. |
| last_seen_at | TIMESTAMPTZ NOT NULL | UTC timestamp of last validated request. Used for rolling expiry refresh. |

### members
B2B member identity mirroring Stytch. Created/updated atomically with users during magic-link provisioning.

| Column | Type | Comment |
|--------|------|---------|
| id | UUID PK | Auto-generated UUID primary key. |
| stytch_member_id | VARCHAR(255) NOT NULL UNIQUE | Stytch B2B Member.member_id. Unique across the entire system. |
| user_id | UUID NOT NULL FK users(id) ON DELETE CASCADE | FK to users(id). The Somotracker application identity. |
| tenant_id | UUID NOT NULL FK tenants(id) ON DELETE CASCADE | FK to tenants(id). The B2B organization. |
| stytch_member_raw | JSONB | Cached Stytch member object (JSONB) for audit/debugging. Sensitive metadata fields are stripped before storage. |
| created_at | TIMESTAMPTZ NOT NULL | UTC timestamp of member creation. |
| updated_at | TIMESTAMPTZ NOT NULL | UTC timestamp of last modification. |

## School Information System

### countries
Reference table of countries supported by the SIS (ISO 3166-1 alpha-2 codes).

| Column | Type | Comment |
|--------|------|---------|
| id | UUID PK | Auto-generated UUID primary key. |
| country_name | VARCHAR(255) NOT NULL | Human-readable country name (e.g. Kenya). |
| country_code | VARCHAR(2) NOT NULL UNIQUE | ISO 3166-1 alpha-2 code (e.g. KE). Unique. |
| created_at | TIMESTAMPTZ NOT NULL | UTC timestamp of row creation. |
| updated_at | TIMESTAMPTZ NOT NULL | UTC timestamp of last modification. |

### education_systems
Reference table of education systems (e.g. Competency-Based Education / CBE). Each system belongs to one country.

| Column | Type | Comment |
|--------|------|---------|
| id | UUID PK | Auto-generated UUID primary key. |
| country_id | UUID NOT NULL FK countries(id) ON DELETE CASCADE | FK to countries(id). The country this education system belongs to. Cascades on country delete. |
| system_name | VARCHAR(255) NOT NULL | Human-readable system name (e.g. Competency-Based Education / CBE). |
| description | TEXT | Optional free-text description of the education system. |
| created_at | TIMESTAMPTZ NOT NULL | UTC timestamp of row creation. |
| updated_at | TIMESTAMPTZ NOT NULL | UTC timestamp of last modification. |

### grade_levels
Grade levels scoped to an education system and country (e.g. PP1, Grade 7, Senior 1).

| Column | Type | Comment |
|--------|------|---------|
| id | UUID PK | Auto-generated UUID primary key. |
| education_system_id | UUID NOT NULL FK education_systems(id) ON DELETE CASCADE | FK to education_systems(id). Cascades on education system delete. |
| country_id | UUID NOT NULL FK countries(id) ON DELETE CASCADE | FK to countries(id). Cascades on country delete. |
| tier_stage | VARCHAR(64) NOT NULL | Broad stage bucket: pre_primary, primary, lower_secondary, upper_secondary. |
| local_label | VARCHAR(64) NOT NULL | Local label used in that country/system (e.g. PP1, Grade 7, Senior 1). |
| sequence_index | INTEGER NOT NULL | Integer for chronological sorting; unique within system+country. |
| created_at | TIMESTAMPTZ NOT NULL | UTC timestamp of row creation. |
| updated_at | TIMESTAMPTZ NOT NULL | UTC timestamp of last modification. |

### schools
Schools operated by a tenant. Each school belongs to one tenant, country, and education system.

| Column | Type | Comment |
|--------|------|---------|
| id | UUID PK | Auto-generated UUID primary key. |
| tenant_id | UUID NOT NULL FK tenants(id) ON DELETE CASCADE | FK to tenants(id). The B2B organization that operates the school. Cascades on tenant delete. |
| school_name | VARCHAR(255) NOT NULL | Human-readable school name. |
| country_id | UUID NOT NULL FK countries(id) ON DELETE CASCADE | FK to countries(id). Cascades on country delete. |
| education_system_id | UUID NOT NULL FK education_systems(id) ON DELETE CASCADE | FK to education_systems(id). Cascades on education system delete. |
| created_at | TIMESTAMPTZ NOT NULL | UTC timestamp of row creation. |
| updated_at | TIMESTAMPTZ NOT NULL | UTC timestamp of last modification. |

### school_memberships
Links a user to a school with a role. A user has at most one membership per school, and at most one active membership across all schools.

| Column | Type | Comment |
|--------|------|---------|
| id | UUID PK | Auto-generated UUID primary key. |
| school_id | UUID NOT NULL FK schools(id) ON DELETE CASCADE | FK to schools(id). Cascades on school delete. |
| user_id | UUID NOT NULL FK users(id) ON DELETE CASCADE | FK to users(id). Cascades on user delete. |
| role | user_role NOT NULL | Role within the school (user_role enum). |
| is_active | BOOLEAN NOT NULL DEFAULT FALSE | Whether this membership is the user's currently active school. Enforced by partial unique index: only one active per user. |
| invited_at | TIMESTAMPTZ | Added via migration 000011. |
| invited_by | UUID FK users(id) ON DELETE SET NULL | Added via migration 000011. |
| accepted_at | TIMESTAMPTZ | Added via migration 000011. |
| created_at | TIMESTAMPTZ NOT NULL | UTC timestamp of row creation. |
| updated_at | TIMESTAMPTZ NOT NULL | UTC timestamp of last modification. |

### students
Student records scoped to a school. admission_number is unique per school. metadata stores flexible external identifiers (NEMIS, KICD, etc.).

| Column | Type | Comment |
|--------|------|---------|
| student_id | UUID PK | Auto-generated UUID primary key. |
| school_id | UUID NOT NULL FK schools(id) ON DELETE CASCADE | FK to schools(id). Cascades on school delete. |
| admission_number | VARCHAR(255) NOT NULL | School-scoped admission number (unique per school). |
| full_name | VARCHAR(255) NOT NULL | Student full name (first + last). |
| date_of_birth | DATE NOT NULL | Date of birth. |
| gender | VARCHAR(32) NOT NULL | Gender (free-text for international flexibility). |
| metadata | JSONB NOT NULL DEFAULT '{}' | JSONB for flexible external identifiers (e.g. NEMIS, KICD tracking codes). |
| created_at | TIMESTAMPTZ NOT NULL | UTC timestamp of row creation. |
| updated_at | TIMESTAMPTZ NOT NULL | UTC timestamp of last modification. |

### guardian_student_links
Links a guardian (via school_membership) to a student with a relationship type.

| Column | Type | Comment |
|--------|------|---------|
| id | UUID PK | Auto-generated UUID primary key. |
| school_membership_id | UUID NOT NULL FK school_memberships(id) ON DELETE CASCADE | FK to school_memberships(id). Cascades on membership delete. |
| student_id | UUID NOT NULL FK students(student_id) ON DELETE CASCADE | FK to students(student_id). Cascades on student delete. |
| relationship_type | VARCHAR(64) NOT NULL | Relationship type (e.g. Parent, Legal Guardian, Sponsor). |
| created_at | TIMESTAMPTZ NOT NULL | UTC timestamp of row creation. |
| updated_at | TIMESTAMPTZ NOT NULL | UTC timestamp of last modification. |

### streams
Optional stream definitions per school.

| Column | Type | Comment |
|--------|------|---------|
| id | UUID PK | Auto-generated UUID primary key. |
| school_id | UUID NOT NULL FK schools(id) ON DELETE CASCADE |  |
| name | VARCHAR(64) NOT NULL |  |
| color | VARCHAR(32) |  |
| created_at | TIMESTAMPTZ NOT NULL |  |
| updated_at | TIMESTAMPTZ NOT NULL |  |

## Curriculum Hierarchy

### subjects
Subjects offered within an education system (e.g. Mathematics, English). Scoped to education_system_id.

| Column | Type | Comment |
|--------|------|---------|
| id | UUID PK | Auto-generated UUID primary key. |
| education_system_id | UUID NOT NULL FK education_systems(id) ON DELETE CASCADE | FK to education_systems(id). Cascades on education system delete. |
| name | VARCHAR(255) NOT NULL | Human-readable subject name (e.g. Mathematics). |
| code | VARCHAR(64) NOT NULL | Short subject code (e.g. MAT, ENG). Unique within an education system. Expanded to VARCHAR(64) in 000014. |
| type | VARCHAR(32) NOT NULL | Subject type: Core, Optional, Elective, etc. |
| grade_level_id | UUID FK grade_levels(id) ON DELETE CASCADE | FK to grade_levels(id). The grade level this subject belongs to. Cascades on grade level delete. Nullable for legacy subjects. Added 000013. |
| color | VARCHAR(32) | Added 000016 for visual identification. |
| created_at | TIMESTAMPTZ NOT NULL | UTC timestamp of row creation. |
| updated_at | TIMESTAMPTZ NOT NULL | UTC timestamp of last modification. |

### topics
Topics within a subject, ordered by sequence_index for curriculum sequencing.

| Column | Type | Comment |
|--------|------|---------|
| id | UUID PK | Auto-generated UUID primary key. |
| subject_id | UUID NOT NULL FK subjects(id) ON DELETE CASCADE | FK to subjects(id). Cascades on subject delete. |
| name | VARCHAR(255) NOT NULL | Topic name (e.g. Fractions and Decimals). |
| sequence_index | INTEGER NOT NULL | Integer for chronological sorting; unique within a subject. |
| created_at | TIMESTAMPTZ NOT NULL | UTC timestamp of row creation. |
| updated_at | TIMESTAMPTZ NOT NULL | UTC timestamp of last modification. |

### sub_topics
Sub-topics within a topic, ordered by sequence_index for granular curriculum sequencing.

| Column | Type | Comment |
|--------|------|---------|
| id | UUID PK | Auto-generated UUID primary key. |
| topic_id | UUID NOT NULL FK topics(id) ON DELETE CASCADE | FK to topics(id). Cascades on topic delete. |
| name | VARCHAR(255) NOT NULL | Sub-topic name (e.g. Addition of Fractions). |
| sequence_index | INTEGER NOT NULL | Integer for chronological sorting; unique within a topic. |
| created_at | TIMESTAMPTZ NOT NULL | UTC timestamp of row creation. |
| updated_at | TIMESTAMPTZ NOT NULL | UTC timestamp of last modification. |

## Academic Calendar

### academic_years
Academic years scoped to a school (e.g., 2026, 2026-2027). Used for scheduling, grading periods.

| Column | Type | Comment |
|--------|------|---------|
| id | UUID PK | Auto-generated UUID primary key. |
| school_id | UUID NOT NULL FK schools(id) ON DELETE CASCADE | FK to schools(id). Cascades on school delete. |
| name | VARCHAR(64) NOT NULL | Human-readable year identifier (e.g., "2026", "2026-2027"). Unique per school. |
| start_date | DATE NOT NULL | First day of the academic year. |
| end_date | DATE NOT NULL | Last day of the academic year. |
| created_at | TIMESTAMPTZ NOT NULL | UTC timestamp of row creation. |
| updated_at | TIMESTAMPTZ NOT NULL | UTC timestamp of last modification. |

### academic_terms
Academic terms (semester, quarter, term 1/2/3) within an academic year.

| Column | Type | Comment |
|--------|------|---------|
| id | UUID PK | Auto-generated UUID primary key. |
| academic_year_id | UUID NOT NULL FK academic_years(id) ON DELETE CASCADE | FK to academic_years(id). Cascades on academic year delete. |
| name | VARCHAR(64) NOT NULL | Term name (e.g., "Term 1", "Semester 1"). Unique within an academic year. |
| start_date | DATE NOT NULL | First day of the term. |
| end_date | DATE NOT NULL | Last day of the term. |
| created_at | TIMESTAMPTZ NOT NULL | UTC timestamp of row creation. |
| updated_at | TIMESTAMPTZ NOT NULL | UTC timestamp of last modification. |

### public_holidays
National public holidays at the country level (e.g., Madaraka Day, Labour Day).

| Column | Type | Comment |
|--------|------|---------|
| id | UUID PK | Auto-generated UUID primary key. |
| country_id | UUID NOT NULL FK countries(id) ON DELETE CASCADE | FK to countries(id). Cascades on country delete. |
| name | VARCHAR(255) NOT NULL | Holiday name (e.g., "Madaraka Day", "Labour Day"). |
| month | SMALLINT NOT NULL | Month of the holiday (1-12). |
| day | SMALLINT NOT NULL | Day of the month (1-31). |
| created_at | TIMESTAMPTZ NOT NULL | UTC timestamp of row creation. |
| updated_at | TIMESTAMPTZ NOT NULL | UTC timestamp of last modification. |

### school_events
School-specific events (sports, exams, admission days, etc.) with optional attendance tracking.

| Column | Type | Comment |
|--------|------|---------|
| id | UUID PK | Auto-generated UUID primary key. |
| school_id | UUID NOT NULL FK schools(id) ON DELETE CASCADE | FK to schools(id). Cascades on school delete. |
| title | VARCHAR(255) NOT NULL | Event title (e.g., "Inter-House Sports Day"). |
| event_type | VARCHAR(64) NOT NULL | Event type: SPORTS, ADMISSION, EXAM, etc. |
| start_date | DATE NOT NULL | First day of the event. |
| end_date | DATE NOT NULL | Last day of the event. |
| requires_attendance | BOOLEAN NOT NULL DEFAULT FALSE | Whether student attendance must be tracked for this event. |
| created_at | TIMESTAMPTZ NOT NULL | UTC timestamp of row creation. |
| updated_at | TIMESTAMPTZ NOT NULL | UTC timestamp of last modification. |

## Classrooms & Enrollments

### class_rooms
Operational classroom container per academic year and stream (e.g., Class 1 Blue, Class 3 Yellow). Each academic year creates new class_room entries.

| Column | Type | Comment |
|--------|------|---------|
| id | UUID PK | Auto-generated UUID primary key. |
| school_id | UUID NOT NULL FK schools(id) ON DELETE CASCADE | FK to schools(id). Cascades on school delete. |
| academic_year_id | UUID NOT NULL FK academic_years(id) ON DELETE CASCADE | FK to academic_years(id). Links the class_room to a specific academic year. |
| grade_level_id | UUID NOT NULL FK grade_levels(id) ON DELETE CASCADE | FK to grade_levels(id). The grade level for this class_room (e.g., Grade 1, Grade 3). |
| name | VARCHAR(255) NOT NULL | Human-readable class_room name (e.g., "Class 1 Blue", "Grade 3 Yellow"). |
| stream | VARCHAR(64) | Optional stream identifier within the grade (e.g., "Blue", "Yellow", "A", "B"). NULL if no streams. |
| created_at | TIMESTAMPTZ NOT NULL | UTC timestamp of row creation. |
| updated_at | TIMESTAMPTZ NOT NULL | UTC timestamp of last modification. |

### student_class_enrollments
Historical mapping of a student to a class_room for a specific academic term or year. Preserves attendance, assessments, and report cards permanently.

| Column | Type | Comment |
|--------|------|---------|
| id | UUID PK | Auto-generated UUID primary key. |
| school_id | UUID NOT NULL FK schools(id) ON DELETE CASCADE | FK to schools(id). Enforces multi-tenant isolation via RLS. Cascades on school delete. |
| student_id | UUID NOT NULL FK students(student_id) ON DELETE CASCADE | FK to students(student_id). The enrolled student. |
| class_room_id | UUID NOT NULL FK class_rooms(id) ON DELETE CASCADE | FK to class_rooms(id). The operational class_room for this enrollment. |
| academic_year_id | UUID NOT NULL FK academic_years(id) ON DELETE CASCADE | FK to academic_years(id). The academic year of enrollment. |
| academic_term_id | UUID FK academic_terms(id) ON DELETE CASCADE | FK to academic_terms(id). Optional; NULL for year-level enrollments (e.g., final year without term splits). |
| status | enrollment_status NOT NULL DEFAULT 'ACTIVE' | Enrollment status (enrollment_status enum). Controls promotion, repetition, and graduation workflows. |
| enrolled_at | TIMESTAMPTZ NOT NULL DEFAULT NOW() | UTC timestamp when the student was enrolled in this class_room. |
| completed_at | TIMESTAMPTZ | UTC timestamp when enrollment reached a terminal status (PROMOTED, REPEATING, GRADUATED). NULL for ACTIVE. |
| metadata | JSONB NOT NULL DEFAULT '{}' | JSONB for flexible enrollment metadata (e.g., previous school, transfer notes). |
| created_at | TIMESTAMPTZ NOT NULL | UTC timestamp of row creation. |
| updated_at | TIMESTAMPTZ NOT NULL | UTC timestamp of last modification. |

RLS enabled with tenant_isolation policy.

## Timetable & Scheduling

### timetable_templates
Parent container for a distinct bell schedule configuration (e.g., "Standard 6-Period Day", "Morning Shift 3-Lesson").

| Column | Type | Comment |
|--------|------|---------|
| id | UUID PK | Auto-generated UUID primary key. |
| school_id | UUID NOT NULL FK schools(id) ON DELETE CASCADE | FK to schools(id). Cascades on school delete. |
| name | VARCHAR(255) NOT NULL | Name of the template (e.g., "Primary Schedule", "Standard 6-Period Day"). |
| description | TEXT | Optional details about when or who uses this template. |
| created_at | TIMESTAMPTZ NOT NULL | UTC timestamp of row creation. |
| updated_at | TIMESTAMPTZ NOT NULL | UTC timestamp of last modification. |

### time_slots
Individual periods or breaks belonging to a specific template (e.g., Period 1, Morning Break).

| Column | Type | Comment |
|--------|------|---------|
| id | UUID PK | Auto-generated UUID primary key. |
| school_id | UUID NOT NULL FK schools(id) ON DELETE CASCADE | FK to schools(id). Cascades on school delete. |
| timetable_template_id | UUID NOT NULL FK timetable_templates(id) ON DELETE CASCADE | FK to timetable_templates(id). Cascades on template delete. |
| name | VARCHAR(255) NOT NULL | Period label (e.g., Period 1, Morning Break). |
| start_time | TIME NOT NULL | Slot start time (e.g., 08:00:00). |
| end_time | TIME NOT NULL | Slot end time (e.g., 08:40:00). |
| sequence_index | INTEGER NOT NULL | Order of the slot within the template (1, 2, 3...). |
| is_instructional | BOOLEAN NOT NULL DEFAULT TRUE | True for classes, False for breaks/recess. Defaults to TRUE. |
| created_at | TIMESTAMPTZ NOT NULL | UTC timestamp of row creation. |
| updated_at | TIMESTAMPTZ NOT NULL | UTC timestamp of last modification. |

### rooms
Physical facilities and campus locations to prevent room overbooking (e.g., Lab A, Room 204). Distinct from class_rooms.

| Column | Type | Comment |
|--------|------|---------|
| id | UUID PK | Auto-generated UUID primary key. |
| school_id | UUID NOT NULL FK schools(id) ON DELETE CASCADE | FK to schools(id). Cascades on school delete. |
| name | VARCHAR(255) NOT NULL | Room identifier or name (e.g., Lab A, Room 204). Unique per school. |
| capacity | INTEGER | Maximum student capacity the room can hold. Optional. |
| room_type | room_type NOT NULL DEFAULT 'STANDARD' | Category of room (STANDARD, SCIENCE_LAB, COMPUTER_LAB, GYM). |
| created_at | TIMESTAMPTZ NOT NULL | UTC timestamp of row creation. |
| updated_at | TIMESTAMPTZ NOT NULL | UTC timestamp of last modification. |

### class_timetable_slots
Maps a classroom to an academic term, assigning subjects, teachers, and rooms to a template time slots for a specific day.

| Column | Type | Comment |
|--------|------|---------|
| id | UUID PK | Auto-generated UUID primary key. |
| school_id | UUID NOT NULL FK schools(id) ON DELETE CASCADE | FK to schools(id). Cascades on school delete. |
| class_room_id | UUID NOT NULL FK class_rooms(id) ON DELETE CASCADE | FK to class_rooms(id). The operational classroom. Cascades on class_room delete. |
| academic_term_id | UUID NOT NULL FK academic_terms(id) ON DELETE CASCADE | FK to academic_terms(id). The academic term. Cascades on term delete. |
| day_of_week | INTEGER NOT NULL CHECK 1-7 | Day index (1 = Monday through 7 = Sunday). |
| time_slot_id | UUID NOT NULL FK time_slots(id) ON DELETE CASCADE | FK to time_slots(id). The period within the day. Cascades on slot delete. |
| subject_id | UUID NOT NULL FK subjects(id) ON DELETE CASCADE | FK to subjects(id). The taught subject. Cascades on subject delete. |
| teacher_membership_id | UUID NOT NULL FK school_memberships(id) ON DELETE CASCADE | FK to school_memberships(id) for the assigned teacher. Cascades on membership delete. |
| room_id | UUID FK rooms(id) ON DELETE SET NULL | FK to rooms(id). Optional physical location constraint. |
| created_at | TIMESTAMPTZ NOT NULL | UTC timestamp of row creation. |
| updated_at | TIMESTAMPTZ NOT NULL | UTC timestamp of last modification. |

Constraints:
- class_timetable_slots_teacher_no_clash UNIQUE (teacher_membership_id, academic_term_id, day_of_week, time_slot_id)
- class_timetable_slots_class_day_time_uniq UNIQUE (class_room_id, academic_term_id, day_of_week, time_slot_id)
- class_timetable_slots_unique_assignment UNIQUE (school_id, class_room_id, academic_term_id, day_of_week, time_slot_id) added 000015

### timetable_substitutions
Handles emergency or planned teacher absences on specific calendar dates without modifying the master weekly recurring timetable.

| Column | Type | Comment |
|--------|------|---------|
| id | UUID PK | Auto-generated UUID primary key. |
| school_id | UUID NOT NULL FK schools(id) ON DELETE CASCADE | FK to schools(id). Cascades on school delete. |
| class_timetable_slot_id | UUID NOT NULL FK class_timetable_slots(id) ON DELETE CASCADE | FK to class_timetable_slots(id) being overridden. Cascades on slot delete. |
| substitution_date | DATE NOT NULL | The precise calendar date of the absence/coverage. |
| original_teacher_membership_id | UUID NOT NULL FK school_memberships(id) ON DELETE CASCADE | FK to school_memberships(id) for the teacher who is away. |
| substitute_teacher_membership_id | UUID FK school_memberships(id) ON DELETE SET NULL | FK to school_memberships(id) for the covering teacher. NULL if unassigned. |
| status | substitution_status NOT NULL DEFAULT 'PENDING' | Lifecycle state: PENDING, ASSIGNED, COMPLETED, or CANCELLED. |
| reason | TEXT | Optional explanation (e.g., Medical leave). |
| created_at | TIMESTAMPTZ NOT NULL | UTC timestamp of row creation. |
| updated_at | TIMESTAMPTZ NOT NULL | UTC timestamp of last modification. |

## Attendance

### timetable_attendance
Primary attendance tracking table linking to specific timetable slot instances on calendar dates, allowing subject teachers to record presence during instructional periods.

| Column | Type | Comment |
|--------|------|---------|
| id | UUID PK | Auto-generated UUID primary key. |
| school_id | UUID NOT NULL FK schools(id) ON DELETE CASCADE | FK to schools(id). Cascades on school delete. |
| student_id | UUID NOT NULL FK students(student_id) ON DELETE CASCADE | FK to students(student_id). Cascades on student delete. |
| class_timetable_slot_id | UUID NOT NULL FK class_timetable_slots(id) ON DELETE CASCADE | FK to class_timetable_slots(id). Cascades on slot delete. |
| attendance_date | DATE NOT NULL | The specific calendar date of the lesson instance. |
| status | timetable_attendance_status NOT NULL | Attendance state: PRESENT, ABSENT, LATE, or EXCUSED. |
| remarks | TEXT | Optional notes (e.g., "Left early due to illness"). |
| recorded_by_membership_id | UUID NOT NULL FK school_memberships(id) ON DELETE CASCADE | FK to school_memberships(id) for the teacher who took the register. Cascades on membership delete. |
| created_at | TIMESTAMPTZ NOT NULL | UTC timestamp of row creation. |
| updated_at | TIMESTAMPTZ NOT NULL | UTC timestamp of last modification. |

Constraint: timetable_attendance_uniq_student_slot_date UNIQUE (student_id, class_timetable_slot_id, attendance_date)

### event_attendance
Attendance tracking for special school-wide activities (sports days, symposia) where the regular timetable is suspended.

| Column | Type | Comment |
|--------|------|---------|
| id | UUID PK | Auto-generated UUID primary key. |
| school_id | UUID NOT NULL FK schools(id) ON DELETE CASCADE | FK to schools(id). Cascades on school delete. |
| student_id | UUID NOT NULL FK students(student_id) ON DELETE CASCADE | FK to students(student_id). Cascades on student delete. |
| school_event_id | UUID NOT NULL FK school_events(id) ON DELETE CASCADE | FK to school_events(id). Cascades on event delete. |
| attendance_date | DATE NOT NULL | The calendar date of the event. |
| status | event_attendance_status NOT NULL | Attendance state: PRESENT, ABSENT, or EXCUSED. |
| remarks | TEXT | Optional details about event attendance. |
| created_at | TIMESTAMPTZ NOT NULL | UTC timestamp of row creation. |
| updated_at | TIMESTAMPTZ NOT NULL | UTC timestamp of last modification. |

Constraint: event_attendance_uniq_student_event_date UNIQUE (student_id, school_event_id, attendance_date)

## Bulk Ingestion

### bulk_jobs
Generic bulk ingestion jobs. Reusable across admin invitations, student imports, exam results, staff imports, and future bulk operations.

| Column | Type | Comment |
|--------|------|---------|
| id | UUID PK | Auto-generated UUID primary key. |
| job_type | TEXT NOT NULL CHECK IN ('ADMIN_INVITATION') | Bulk operation category. Extendable via CHECK constraint or lookup table migration. |
| idempotency_key | TEXT NOT NULL | Client-supplied idempotency token to prevent duplicate submissions. |
| school_id | UUID NOT NULL FK schools(id) ON DELETE CASCADE | School scope for the bulk operation. |
| tenant_id | UUID NOT NULL FK tenants(id) ON DELETE CASCADE | Tenant isolation anchor. |
| created_by | UUID NOT NULL FK users(id) ON DELETE CASCADE | User who submitted the bulk job. |
| status | TEXT NOT NULL DEFAULT 'QUEUED' CHECK IN ('QUEUED','PROCESSING','COMPLETED','COMPLETED_WITH_ERRORS','FAILED') | Processing lifecycle: QUEUED, PROCESSING, COMPLETED, COMPLETED_WITH_ERRORS, FAILED. |
| total_records | INTEGER NOT NULL | Total rows submitted in the payload. |
| succeeded_count | INTEGER NOT NULL DEFAULT 0 | Rows successfully processed. |
| failed_count | INTEGER NOT NULL DEFAULT 0 | Rows that failed permanently. |
| deferred_count | INTEGER NOT NULL DEFAULT 0 | Rows deferred (e.g. retry queued). |
| metadata | JSONB NOT NULL DEFAULT '{}' | Job-type-specific summary info, e.g. source file name. |
| created_at | TIMESTAMPTZ NOT NULL | UTC timestamp of row creation. |
| updated_at | TIMESTAMPTZ NOT NULL | UTC timestamp of last modification. |

### bulk_job_items
Individual row-level records for each bulk ingestion job.

| Column | Type | Comment |
|--------|------|---------|
| id | UUID PK | Auto-generated UUID primary key. |
| job_id | UUID NOT NULL FK bulk_jobs(id) ON DELETE CASCADE | Parent bulk job reference. |
| row_index | INTEGER NOT NULL | Original array position in the submitted payload; maps errors back to client. |
| payload | JSONB NOT NULL | Full submitted row, e.g. {"email":"...","full_name":"...","role":"..."}. |
| result | JSONB | Output data on success, e.g. {"stytch_invite_id":"...","stytch_member_id":"..."}. |
| status | TEXT NOT NULL DEFAULT 'PENDING' CHECK IN ('PENDING','PROCESSING','SUCCEEDED','FAILED','DEFERRED') | Per-row processing lifecycle. |
| attempt_count | INTEGER NOT NULL DEFAULT 0 | Number of processing attempts made. |
| last_error | TEXT | Human-readable or structured error message from the last failed attempt. |
| created_at | TIMESTAMPTZ NOT NULL | UTC timestamp of row creation. |
| updated_at | TIMESTAMPTZ NOT NULL | UTC timestamp of last modification. |

## Relationships Summary

- tenants 1:* users, schools, sessions, members, bulk_jobs
- users 1:* sessions, members, school_memberships, bulk_jobs(created_by)
- schools 1:* school_memberships, students, academic_years, timetable_templates, rooms, class_timetable_slots, timetable_substitutions, timetable_attendance, event_attendance, bulk_jobs
- countries 1:* education_systems, grade_levels, public_holidays
- education_systems 1:* grade_levels, subjects
- grade_levels 1:* class_rooms, subjects
- academic_years 1:* academic_terms, class_rooms
- class_rooms 1:* student_class_enrollments, class_timetable_slots
- subjects 1:* topics, class_timetable_slots
- topics 1:* sub_topics
- class_timetable_slots 1:* timetable_substitutions, timetable_attendance
- school_events 1:* event_attendance

All school-scoped tables enforce multi-tenant isolation via RLS policies referencing schools.tenant_id = app.current_tenant_id.
