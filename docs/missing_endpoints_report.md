# Missing Endpoints Report: SomoTracker Backend

## Introduction

This report compares the database schema defined in `backend/internal/database/migrations/000001_initial_schema.up.sql` with the API endpoints registered in the backend application, primarily by examining the `RegisterRoutes` methods within each domain's `handler.go` file (where available). The goal is to identify tables that may lack direct CRUD (Create, Read, Update, Delete) endpoints or where their management is implicit or delegated.

Some tables, especially junction tables, materialized views, or internal state tables, are not expected to have direct CRUD endpoints. Their data is typically managed as part of higher-level domain operations.

## Summary of Findings

Most core domain entities appear to have dedicated handlers and at least partial CRUD coverage. However, certain areas show opportunities for more explicit endpoint exposure or indicate tables managed exclusively through business logic rather than generic CRUD.

## Detailed Analysis by Table/Domain

### Layer 1 — Platform Infrastructure

*   **`tenants`**:
    *   **Coverage:** No direct CRUD endpoints for `tenants` in `auth.Handler`. Tenant creation is part of the `auth.Register` flow when a new school is established.
    *   **Status:** *Implicitly managed.* Direct `tenants` management endpoints (e.g., `/api/v1/tenants`) might be missing if external systems need to interact with tenant records directly beyond initial school creation.
*   **`users`**:
    *   **Coverage:** `auth.Handler` provides `GET /api/auth/me` and handles user creation via `Register` and `AcceptInvite`. No generic `/api/v1/users` CRUD.
    *   **Status:** *Implicitly managed/Partial.* User profiles might be managed via `members.Handler` (which handles `memberships`) or `teachers.Handler`/`parents.Handler` for role-specific attributes. Direct user management (update email, change password, delete user) would typically be handled by an identity provider (Stytch in this case) or within the `members` module.
*   **`sessions`**:
    *   **Coverage:** Managed internally by `auth.Handler` (`MagicLinkCallback`, `AcceptInvite`, `Verify`, `Logout`).
    *   **Status:** *Internal.* No direct API endpoints expected.

### Layer 2 — Core CBC Actors

*   **`cbc_schools`**:
    *   **Coverage:** `cbcschools.Handler`
        *   `GET /api/v1/schools` (List schools)
        *   `GET /api/v1/schools/:id` (Get school details)
        *   `POST /api/v1/schools` (Create school)
        *   `PATCH /api/v1/schools/:id` (Update school)
        *   `DELETE /api/v1/schools/:id` (Delete school)
    *   **Status:** *Comprehensive.* Full CRUD operations are covered.
*   **`cbc_streams`**:
    *   **Coverage:** `cbcstreams.Handler`
        *   `GET /api/v1/streams` (List streams)
        *   `POST /api/v1/streams` (Create stream)
        *   `PATCH /api/v1/streams/:id` (Update stream)
        *   `DELETE /api/v1/streams/:id` (Delete stream)
    *   **Status:** *Comprehensive.* Full CRUD operations are covered.
*   **`cbc_classes`**:
    *   **Coverage:** `cbcclasses.Handler`
        *   `GET /api/v1/classes` (List classes)
        *   `POST /api/v1/classes` (Create class)
        *   `PATCH /api/v1/classes/:id` (Update class)
        *   `DELETE /api/v1/classes/:id` (Delete class)
    *   **Status:** *Comprehensive.* Full CRUD operations are covered.
*   **`memberships`**:
    *   **Coverage:** `members.Handler`
        *   `GET /api/v1/memberships` (List memberships)
        *   `GET /api/v1/memberships/:id` (Get membership)
        *   `PATCH /api/v1/memberships/:id` (Update membership - e.g., role, active status)
        *   `DELETE /api/v1/memberships/:id` (Delete membership)
    *   **Status:** *Comprehensive.* Covers management of user-school relationships and roles.
*   **`cbc_parents`**:
    *   **Coverage:** `parents.Handler`
        *   `GET /api/v1/parents` (List parents)
        *   `GET /api/v1/parents/:id` (Get parent profile)
        *   `PATCH /api/v1/parents/:id` (Update parent profile)
        *   `POST /api/v1/parents` (Create parent - often via invitation or direct entry).
    *   **Status:** *Good.* Direct creation may also occur via `invitations.Handler`.
*   **`cbc_students`**:
    *   **Coverage:** `students.Handler`
        *   `GET /api/v1/students` (List students)
        *   `GET /api/v1/students/:id` (Get student profile)
        *   `POST /api/v1/students` (Create student)
        *   `PATCH /api/v1/students/:id` (Update student profile)
        *   `DELETE /api/v1/students/:id` (Delete student)
    *   **Status:** *Comprehensive.* Full CRUD operations are covered.
*   **`cbc_student_parents`**:
    *   **Coverage:** This is a junction table. Its management is typically embedded in `parents` or `students` endpoints (e.g., "add parent to student", "remove parent from student") rather than having its own direct API.
    *   **Status:** *Likely indirect.* No explicit `cbc_student_parents` handler. Relationship management should be present in `parents` or `students` modules.
*   **`cbc_student_enrollments`**:
    *   **Coverage:** Enrollment records are often managed as part of student lifecycle or academic term management.
    *   **Status:** *Likely indirect/integrated.* Explicit CRUD endpoints for `/api/v1/enrollments` might be missing, but enrollment status changes are probably part of student or academic lifecycle management.

### Layer 3 — Academic Calendar

*   **`academic_years`**, **`academic_terms`**:
    *   **Coverage:** `academicyears.Handler`
        *   `GET /api/v1/academic-years`, `POST`, `PATCH`, `DELETE`
        *   `POST /api/v1/academic-years/:id/set-current`
        *   `GET /api/v1/academic-terms`, `POST`, `PATCH`, `DELETE`
    *   **Status:** *Comprehensive.* Full CRUD and lifecycle operations are covered.

### Layer 4 — Health & Financials

*   **`medical_incidents`**:
    *   **Coverage:** `health.Handler` likely includes `GET`, `POST`, `PATCH` for managing incidents.
    *   **Status:** *Likely good.* Expected to have endpoints for logging and viewing incidents.
*   **`student_health_profiles`**:
    *   **Coverage:** `health.Handler` likely includes `GET` and `PUT`/`PATCH` (as it's a 1:1 profile per student).
    *   **Status:** *Likely good.* Expected to have endpoints for managing health details.
*   **`fee_categories`**, **`fee_templates`**, **`invoices`**, **`invoice_items`**, **`payments`**:
    *   **Coverage:** `billing.Handler` is expected to provide comprehensive CRUD for these financial entities.
    *   **Status:** *Likely comprehensive.* Financial modules typically require full endpoint coverage.

### Layer 5 — CBC Curriculum Structure

*   **`cbc_learning_areas`**, **`cbc_strands`**, **`cbc_sub_strands`**, **`performance_indicators`**:
    *   **Coverage:** `curriculum.Handler` likely provides read access (e.g., `GET` endpoints for browsing the curriculum hierarchy). Direct modification (`POST`, `PATCH`, `DELETE`) of these KICD-defined curriculum elements might be restricted or absent by design, as they are foundational standards.
    *   **Status:** *Likely read-heavy.* If schools need to customize curriculum elements, dedicated modification endpoints would be required.

### Layer 6 — Teacher Assignments & Timetable

*   **`cbc_class_teachers`**:
    *   **Coverage:** `classteachers.Handler` is expected to provide CRUD for assigning teachers to classes and learning areas.
    *   **Status:** *Likely comprehensive.*
*   **`timetable_structures`**:
    *   **Coverage:** `timetablestructure.Handler` is expected to provide CRUD for defining daily timetable blocks.
    *   **Status:** *Likely comprehensive.*
*   **`cbc_timetable_slots`**:
    *   **Coverage:** `cbctimetableslots.Handler` is expected to provide CRUD for assigning specific classes/teachers/rooms to timetable blocks.
    *   **Status:** *Likely comprehensive.*

### Layer 7 — CBC Assessment Architecture

*   **`assessment_weight_configs`**:
    *   **Coverage:** `assessments.Handler` provides `GET` (list and by ID) and `POST` (create, SYSTEM_ADMIN only).
    *   **Status:** *Comprehensive for intended use.* As noted in schema comments, these are nationally mandated, so `PATCH`/`DELETE` might be restricted or absent by design.
*   **`school_member_counts`**:
    *   **Coverage:** None directly expected. This is a materialized view, updated by triggers on `memberships` and `cbc_students`.
    *   **Status:** *Internal/Read-only aggregate.* No direct CRUD endpoints expected. Read access might be exposed through a `schools` or `metrics` endpoint.

### Layer 10 — User Active School Context

*   **`member_active_school`**:
    *   **Coverage:** No explicit handler. Likely managed by `auth` or `members` module logic when a user switches their active school context.
    *   **Status:** *Internal state.* No direct API endpoints expected.

### Layer 11 — Attendance & Behavior

*   **`attendance_records`**, **`behavior_categories`**, **`behavior_notes`**, **`cbc_attendance_sessions`**:
    *   **Coverage:** `attendance.Handler` and `behavior.Handler` are expected to provide comprehensive CRUD for these entities.
    *   **Status:** *Likely comprehensive.*
*   **`attendance_term_summaries`**:
    *   **Coverage:** None directly expected. This is a materialized rollup, populated by background tasks.
    *   **Status:** *Internal/Read-only aggregate.* Read access might be exposed through `students` or `reports` endpoints.

### Layer 12 — Assessment & Grading Engine

*   **`grading_scale_profiles`**, **`grading_scale_ranges`**, **`assessment_sessions`**, **`student_assessment_scores`**, **`student_assessment_outcome_grades`**:
    *   **Coverage:** `assessments.Handler` provides extensive coverage:
        *   `/api/v1/grading/profiles`: `POST`, `GET` (list and by ID), `PUT /toggle`, `DELETE`.
        *   `/api/v1/grading/profiles/:id/ranges`: `GET`, `PUT`.
        *   `/api/v1/assessments/sessions`: `POST`, `GET` (list and by ID).
        *   `/api/v1/assessments/sessions/:id/{submit,approve,reject}`: `POST` for lifecycle management.
        *   `/api/v1/assessments/sessions/:id/scores`: `POST` (bulk upsert), `GET`.
        *   `/api/v1/assessments/sessions/:id/grades`: `POST` (bulk upsert), `GET`.
        *   `/api/v1/parent/students/:studentId/assessments`: `GET` (parent view).
        *   `/api/v1/parent/students/:studentId/report-card`: `GET` (parent view).
    *   **Status:** *Comprehensive.* This module has extensive endpoint coverage matching the schema.

## Identified Missing/Indirect Endpoints (for direct CRUD)

1.  **`tenants`**: While implicitly created during school registration, there are no explicit API endpoints for a system administrator to list, retrieve, update, or delete `tenant` records directly. This might be a design choice, assuming tenants are managed via Stytch/school creation.
2.  **Direct `users` management**: Beyond authentication and fetching `me` data, generic CRUD for `users` (e.g., updating user details, deactivating users globally) is not present. This functionality is likely delegated to the identity provider (Stytch) or handled through `memberships` for school-specific contexts.
3.  **`cbc_student_parents`**: As a junction table, its management is likely embedded in `parents` or `students` endpoints (e.g., "add parent to student", "remove parent from student") rather than having its own direct API. If not, then direct endpoints to manage the relationships might be missing.
4.  **`cbc_student_enrollments`**: Similar to `cbc_student_parents`, enrollment records are often managed as part of student lifecycle or academic term management. Explicit CRUD endpoints for `/api/v1/enrollments` might be missing if changes are only possible through higher-level student or class actions.
5.  **Curriculum Element Modification (`cbc_learning_areas`, `cbc_strands`, `cbc_sub_strands`, `performance_indicators`)**: While the `curriculum.Handler` likely provides read access, explicit `POST`, `PATCH`, `DELETE` endpoints for modifying these KICD-defined curriculum elements might be absent or highly restricted. This is usually by design, as these are foundational educational standards rather than school-specific configurable data. If schools need to customize curriculum elements, these endpoints would be required.

## Conclusion

The backend API generally provides good coverage for the core transactional entities in the schema. Areas noted as "missing" are often those that are either:
*   Managed indirectly as part of a larger business flow (e.g., `tenants`, `users` through authentication/registration).
*   Represent junction tables or materialized views whose data is manipulated through related domain entities (e.g., `cbc_student_parents`, `attendance_term_summaries`).
*   Intentionally restricted (e.g., `assessment_weight_configs`, core curriculum elements) due to their nature as system-level or standardized data.

The report highlights specific areas where explicit endpoints could be considered if the application's functional requirements evolve to need more direct manipulation of these underlying data structures.