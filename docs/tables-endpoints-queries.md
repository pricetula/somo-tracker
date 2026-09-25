# Somotracker Tables, Endpoints & Queries Inventory

> Generated from `backend/db/migrations/*.up.sql`, `backend/db/queries/*.sql` and `backend/internal/api/router.go`.
> Last updated: 2026-09-22

## Purpose

Central reference for all database tables in the Somotracker monorepo, the API endpoints that currently touch them, and the sqlc query files that exist. Use this to identify gaps for CRUD coverage, missing queries, and RLS considerations.

---

## Enums

| Enum | Values | Used by |
|------|--------|---------|
| `user_role` | ADMIN, TEACHER, GUARDIAN, FINANCE | school_memberships.role |
| `enrollment_status` | ACTIVE, PROMOTED, REPEATING, GRADUATED | student_class_enrollments.status |
| `substitution_status` | PENDING, ASSIGNED, COMPLETED, CANCELLED | timetable_substitutions.status |
| `room_type` | STANDARD, SCIENCE_LAB, COMPUTER_LAB, GYM | rooms.room_type |
| `timetable_attendance_status` | PRESENT, ABSENT, LATE, EXCUSED | timetable_attendance.status |
| `event_attendance_status` | PRESENT, ABSENT, EXCUSED | event_attendance.status |
| `student_gender` | M, F, OTHER | students.gender |

---

## Table Inventory

### Core Tenancy & Auth

#### tenants
Primary key: `id`
RLS: No
Notes: 1:1 with Stytch organization. Parent of users, schools, sessions.
Existing queries: `GetTenantByStytchOrgID` in `tenants.sql`, `GetTenantStytchOrgID` in `bulk_jobs.sql`
Existing endpoints: none direct; used via `/me`, `/schools`

#### users
Primary key: `id`
RLS: `users_tenant_isolation` on `tenant_id = app.current_tenant_id`
Notes: Per-tenant email uniqueness.
Existing queries: `users.sql`
Existing endpoints: `/api/me`, auth flows

#### sessions
Primary key: `id`
RLS: No
Notes: Opaque local token, Stytch session cached in Redis.
Existing queries: `sessions.sql`
Existing endpoints: auth middleware only

#### members
Primary key: `id`
RLS: No
Notes: B2B mirror of Stytch member.
Existing queries: `members.sql`
Existing endpoints: internal auth provisioning

### School Information System

#### countries
PK `id`, unique `country_code`
Existing queries: `countries.sql`
Endpoints: none yet

#### education_systems
PK `id`, FK `country_id`
Existing queries: none
Endpoints: none yet

#### grade_levels
PK `id`, unique `(education_system_id, country_id, sequence_index)`
Existing queries: `grade_levels.sql`
Endpoint: `GET /api/school/grades`

#### schools
PK `id`, FK `tenant_id, country_id, education_system_id`
RLS: via child tables
Existing queries: none direct
Endpoints: `POST /school/register`, `GET /schools`, `POST /school`, `POST /school/set-active`

#### school_memberships
PK `id`, unique `(school_id, user_id)`, partial unique active per user
RLS: No
Notes: invitation columns `invited_at, invited_by, accepted_at`
Existing queries: `school_memberships.sql`
Endpoints: `/admins`, `/teachers`, `/finance`, `/guardians` list/delete

#### students
PK `student_id`, unique `(school_id, admission_number)` + CI index
RLS: via `student_class_enrollments_tenant_isolation`
Notes: gender enum, metadata JSONB
Existing queries: `students.sql` – ListStudents, CountStudents, DeleteStudents, GetStudentSummary
Endpoints: `GET /students`, `GET /students/summary`, `DELETE /students`, `POST /students/add` bulk import

#### guardian_student_links
PK `id`, unique `(school_membership_id, student_id)`
Existing queries: none direct
Endpoints: implicit via guardians

#### streams
PK `id`, unique `(school_id, name)`
Existing queries: `streams.sql`
Endpoints: `GET /school/streams`, `POST /school/streams`, `GET /school/streams/:id`, `PATCH /school/streams/:id`, `POST /school/streams/delete`

### Curriculum Hierarchy

#### subjects
PK `id`, unique `(education_system_id, code)`, nullable `grade_level_id`
Existing queries: `subjects.sql`
Endpoints: `GET /subjects`, `GET /subjects/:id`

#### topics
PK `id`, unique `(subject_id, sequence_index)`
Existing queries: `topics.sql`
Endpoints: `GET /topics`, `GET /topics/:id`

#### sub_topics
PK `id`, unique `(topic_id, sequence_index)`
Existing queries: `sub_topics.sql`
Endpoints: `GET /sub-topics`, `GET /sub-topics/:id`

### Academic Calendar

#### academic_years
PK `id`, unique `(school_id, name)`
Existing queries: `academic_calendar.sql`, `academic_year_lookup.sql`
Endpoint: `POST /school/academic-period`

#### academic_terms
PK `id`, unique `(academic_year_id, name)`
Existing queries: `academic_calendar.sql`
Endpoints: none direct

#### public_holidays
PK `id`
Existing queries: `academic_calendar.sql`
Endpoints: none

#### school_events
PK `id`
Existing queries: `academic_calendar.sql`
Endpoint: `GET /events`

### Classrooms & Enrollments

#### class_rooms
PK `id`, unique `(school_id, academic_year_id, grade_level_id, NULLIF(stream,'') )`
Existing queries: `class_lookup.sql`
Endpoints: `GET /school/classes`, `GET /school/classes/:id`, `POST /school/classes`

#### student_class_enrollments
PK `id`, unique `(student_id, academic_year_id)`, unique term index
RLS: `student_class_enrollments_tenant_isolation`
Existing queries: none direct (joined in students)
Endpoints: implicit via students/classes

### Timetable & Scheduling

#### timetable_templates
PK `id`, unique `(school_id, name)`
RLS: tenant isolation
Existing queries: `timetable_templates.sql`
Endpoints: `GET /timetable/templates`, `GET /timetable/templates/:id`, `PATCH /timetable/templates/:id`, `POST /timetable/templates`

#### time_slots
PK `id`, unique `(timetable_template_id, sequence_index)`
RLS: tenant isolation
Existing queries: `time_slots.sql`
Endpoint: `GET /timetable/templates/:id/slots`

#### rooms
PK `id`, unique `(school_id, name)`
RLS: tenant isolation
Existing queries: none
Endpoints: none yet

#### class_timetable_slots
PK `id`, unique `(school_id, class_room_id, academic_term_id, day_of_week, time_slot_id)`
RLS: tenant isolation
Unique constraints for teacher clash and class clash
Existing queries: `class_timetable_slots.sql`
Endpoints: `POST /timetable/setup`, `GET /timetable/templates/:id/classes/:classId/slots`, `DELETE /timetable/class-timetable-slots/:id`

#### timetable_substitutions
PK `id`, unique `(class_timetable_slot_id, substitution_date)`
RLS: tenant isolation
Existing queries: none
Endpoints: none yet

### Attendance

#### timetable_attendance
PK `id`, unique `(student_id, class_timetable_slot_id, attendance_date)`
RLS: tenant isolation
Existing queries: `attendance.sql`
Endpoint: `POST /attendance`, `GET /attendance/sessions`

#### event_attendance
PK `id`, unique `(student_id, school_event_id, attendance_date)`
RLS: tenant isolation
Existing queries: `attendance.sql`
Endpoints: none direct

### Bulk Jobs

#### bulk_jobs
PK `id`, unique `(tenant_id, idempotency_key)`, unique `(school_id, idempotency_key)`
Existing queries: `bulk_jobs.sql` – CreateBulkJob, GetBulkJob, UpdateBulkJobStatus, IncrementBulkJobCounts, GetBulkJobByTenantAndIdempotency
Endpoints: admin/teacher/finance/guardian invitations, student import jobs

#### bulk_job_items
PK `id`, FK `job_id`
Existing queries: none direct
Endpoints: internal worker only

#### student_gender_counts
PK `school_id` denormalized
Existing queries: none
Endpoints: used via `GET /students/summary`

---

## Existing API Endpoints by Resource

### Auth
- `POST /api/auth/magic-link/send`
- `GET /api/auth/callback`
- `GET /api/auth/invite/callback`
- `POST /api/auth/logout`
- `GET /api/me`

### Schools
- `POST /api/school/register`
- `GET /api/schools`
- `POST /api/school`
- `POST /api/school/set-active`

### Academic Period
- `POST /api/school/academic-period`

### Streams
- `GET /api/school/streams`
- `POST /api/school/streams`
- `GET /api/school/streams/:id`
- `PATCH /api/school/streams/:id`
- `POST /api/school/streams/delete`

### Grades
- `GET /api/school/grades`

### Classes
- `GET /api/school/classes`
- `GET /api/school/classes/:id`
- `POST /api/school/classes`

### Timetable
- `GET /api/timetable/templates`
- `GET /api/timetable/templates/:id`
- `PATCH /api/timetable/templates/:id`
- `POST /api/timetable/templates`
- `POST /api/timetable/setup`
- `GET /api/timetable/templates/:id/slots`
- `GET /api/timetable/templates/:id/classes/:classId/slots`
- `DELETE /api/timetable/class-timetable-slots/:id`

### Attendance
- `POST /api/attendance`
- `GET /api/attendance/sessions`

### Memberships
- `GET /api/admins` / `DELETE /api/admins`
- `GET /api/teachers` / `GET /api/teachers/summary` / `DELETE /api/teachers`
- `GET /api/finance` / `DELETE /api/finance`
- `GET /api/guardians` / `GET /api/guardians/summary` / `DELETE /api/guardians`

### Invitations & Bulk Jobs
- Admin: `POST /api/admins/invitations`, `GET /api/admins/invitations/jobs/:job_id`, `POST /api/admins/invitations/jobs/:job_id/retry-failed`, `GET /api/admins/invitations/jobs/:job_id/events`
- Teacher: `POST /api/teachers/invitations` + jobs endpoints
- Finance: `POST /api/finance/invitations` + jobs endpoints
- Guardian: `POST /api/guardians/invitations` + jobs endpoints
- Students import: `POST /api/students/add`, `GET /api/students/jobs/:job_id`, `GET /api/students/jobs/:job_id/events`

### Students
- `GET /api/students`
- `GET /api/students/summary`
- `DELETE /api/students`

### Events
- `GET /api/events`

### Curriculum
- `GET /api/subjects` / `GET /api/subjects/:id`
- `GET /api/topics` / `GET /api/topics/:id`
- `GET /api/sub-topics` / `GET /api/sub-topics/:id`

---

## Existing SQLC Query Files

| File | Tables covered |
|------|----------------|
| `academic_calendar.sql` | academic_years, academic_terms, public_holidays, school_events |
| `academic_year_lookup.sql` | academic_years |
| `attendance.sql` | timetable_attendance, event_attendance |
| `auth_lookup.sql` | users, tenants |
| `auth_service.sql` | sessions, members |
| `bulk_jobs.sql` | bulk_jobs |
| `class_lookup.sql` | class_rooms, student_class_enrollments |
| `class_timetable_slots.sql` | class_timetable_slots |
| `countries.sql` | countries |
| `grade_levels.sql` | grade_levels |
| `guardians.sql` | school_memberships, guardian_student_links |
| `members.sql` | members |
| `school_memberships.sql` | school_memberships |
| `sessions.sql` | sessions |
| `streams.sql` | streams |
| `students.sql` | students, student_gender_counts |
| `sub_topics.sql` | sub_topics |
| `subjects.sql` | subjects |
| `teachers.sql` | school_memberships |
| `tenants.sql` | tenants |
| `time_slots.sql` | time_slots |
| `timetable_templates.sql` | timetable_templates |
| `topics.sql` | topics |
| `users.sql` | users |

---

## Gaps & Recommended Endpoints / Queries

### Missing CRUD
- **countries / education_systems / grade_levels**: read-only reference today. If admin seeding needed, add `POST /countries`, `POST /education-systems`.
- **rooms**: no API. Need `GET /school/rooms`, `POST /school/rooms`, `PATCH /school/rooms/:id`, `DELETE /school/rooms/:id`.
- **timetable_substitutions**: no API. Need list/create/update for teacher coverage.
- **public_holidays**: no API for admin to manage.
- **academic_terms**: only created via academic-period endpoint. Add `GET /school/academic-periods` listing.
- **student_class_enrollments**: enrollment create/patch endpoint missing; currently implicit.
- **guardian_student_links**: no direct API for linking/unlinking guardians to students.
- **event_attendance**: no create/list endpoints.
- **bulk_job_items**: no API for inspecting per-row results.

### Missing Queries
- `schools` table has no sqlc file; queries for list schools per tenant needed.
- `rooms`, `timetable_substitutions`, `event_attendance`, `class_rooms` create/update queries missing.
- `public_holidays` CRUD queries missing.
- `education_systems` queries missing.

### Recommendations
1. Add sqlc files for `schools`, `rooms`, `timetable_substitutions`.
2. Expose `GET /api/school/rooms` and timetable substitution endpoints.
3. Add enrollment endpoints: `POST /api/classes/:id/enrollments`, `PATCH /api/enrollments/:id`.
4. Standardize list endpoints with pagination params matching `ListStudents` pattern.
5. Ensure all RLS-enabled tables have tenant-scoped queries using `app.current_tenant_id`.

---

## Notes on RLS

Tables with `FORCE ROW LEVEL SECURITY`:
- users
- student_class_enrollments
- timetable_templates, time_slots, rooms, class_timetable_slots, timetable_substitutions
- timetable_attendance, event_attendance

All queries on these tables must run inside `database.WithTenantTx` which sets `app.current_tenant_id`.

---

*This document is auto-generated reference. Update when migrations or router change.*
