# Somotracker Architecture: Schema Specifications

This document defines the unified identity model, role hierarchies, and system constraints for Somotracker. These definitions dictate both backend API routing privileges and frontend dashboard layouts.

---

## 👥 System Roles & Permissions Matrix

The platform operates on a strict, top-down data hierarchy to protect multi-tenant integrity and simplify navigation. Role scopes are bounded by the `user_role` enum.

| Role (Enum Value) | Access Scope | Operational Domain & Core Focus |
| :--- | :--- | :--- |
| `SYSTEM_ADMIN` | Global (Cross-Tenant) | Platform infrastructure, multi-tenant onboarding, billing, and global health metrics. |
| `SCHOOL_ADMIN` | Tenant-Wide | Full institutional oversight, staff/student user configuration, and school-wide historical performance analytics. |
| `TEACHER` | Roster-Scoped | Classroom management, grade inputs, attendance tracking, and localized subject trends. |
| `SUPPORT_STAFF` | Cohort-Scoped | Early warning indicators, action log tracking, student academic matrices, and clinical/health logs. |

---

## 📐 Data Architecture

### 1. Tenant Schema (`tenants` table)
Represents the organization or corporate entity. Every business account maps to a record here.

| Field | Type | Constraints | Description |
| :--- | :--- | :--- | :--- |
| `id` | UUID | Primary Key | Unique internal identifier for the tenant. |
| `name` | String | Required | Display name of the parent organization/corporate group. |
| `slug` | String | Unique, Indexed | URL-friendly identifier (e.g., `acme-corp`) for custom routing. |
| `created_at` | Timestamp | Auto-generated | Timestamp when the tenant was created. |

---

### 2. User Schema (`users` table)
Every user identity in the system shares this core schema. Authentication states and login tokens are offloaded externally (Stytch).

| Field Name | Data Type | Nullable | Domain Rules & Database Constraints |
| --- | --- | --- | --- |
| `id` | `UUID` | No | Primary key. Globally unique system identifier. |
| `email` | `VARCHAR` | No | Normalized, unique communication key. Secondary unique index used for Stytch mapping. |
| `tenant_id` | `UUID` | **Yes** | Foreign key linking the user to their tenant. **Must be NULL for `SYSTEM_ADMIN`**, but strictly **MANDATORY** for all school-level users to drive Row-Level Security (RLS). |
| `first_name` | `VARCHAR` | No | Target for UI dashboard personalization. |
| `last_name` | `VARCHAR` | No | User profile family name. |
| `is_active` | `BOOLEAN` | No | Defaults to `true`. Setting to `false` instantly kills system access globally while keeping historical records intact for audit trails. |
| `external_auth_id` | `VARCHAR` | No | Unique, Indexed. The immutable Stytch User ID used for mapping active auth sessions. |
| `created_at` | `TIMESTAMP` | No | Timestamp when the user profile was initialized. |
| `updated_at` | `TIMESTAMP` | No | Timestamp of the most recent profile change. |

---

### 3. Education System Schema (`education_systems` table)
Represents the education system which a learning institution uses and will determine grading styles, rubrics, and structural constraints for different curriculums.

| Field | Type | Constraints | Description |
| :--- | :--- | :--- | :--- |
| `id` | UUID | Primary Key | Unique internal identifier for the education system. |
| `name` | String | Required | Display name of the curriculum system (e.g., "Kenya CBC", "Cambridge IGCSE"). |
| `country_code` | String | Required | ISO country code where the education system is implemented (e.g., `KE`). |

---

### 4. Grade Level Schema (`grades` table)
Static lookup catalog defining the permanent academic learning stages within an education system blueprint.

| Field Name | Data Type | Nullable | Domain Rules & Database Constraints |
| --- | --- | --- | --- |
| `id` | `UUID` | No | Primary key. |
| `education_system_id` | `UUID` | No | Foreign key linking to the parent `education_systems` schema. |
| `name` | `VARCHAR` | No | Standardized blueprint name (e.g., "Grade 1", "Grade 2", "Form 1", "Year 7"). |
| `sequence_order` | `INTEGER` | No | Numeric sequence tracking (e.g., 1 for PP1, 2 for PP2, 3 for Grade 1) to fuel automated promotions. |

---

### 5. School Schema (`schools` table)
Represents individual physical school campuses or academic sections operating under a corporate Tenant.

| Field | Type | Constraints | Description |
| :--- | :--- | :--- | :--- |
| `id` | UUID | Primary Key | Unique internal identifier for the specific school/campus. |
| `name` | String | Required | Display name of the school campus. |
| `education_system_id` | `UUID` | No | Foreign key linking to the active `education_systems` track. |
| `tenant_id` | `UUID` | No | Foreign key linking explicitly to the parent corporate `tenants` account. Driven by RLS. |

---

### 6. Membership Schema (`memberships` table)
Maps users to specific schools with specific granular access roles. This enables multi-school assignments across a single tenant profile.

| Field Name | Data Type | Nullable | Domain Rules & Database Constraints |
| --- | --- | --- | --- |
| `id` | `UUID` | No | Primary key. Globally unique identifier for this specific contextual relationship. |
| `role` | `ENUM` | No | Bounded by `user_role`. Determines UI dashboard layouts and API middleware access for this specific campus context. |
| `user_id` | `UUID` | No | Foreign key linking directly to the `users` table identity record. |
| `school_id` | `UUID` | No | Foreign key linking directly to the target `schools` record. |
| `is_active` | `BOOLEAN` | No | Defaults to `true`. Setting to `false` revokes access to this explicit school without destroying historical activity data or log trails. |
| `created_at` | `TIMESTAMP` | No | Timestamp when this membership assignment was initialized. |

---

### 7. Academic Year Schema (`academic_years` table)
Defines the overarching boundary for a school's calendar cycle.

| Field Name | Data Type | Nullable | Domain Rules & Database Constraints |
| --- | --- | --- | --- |
| `id` | `UUID` | No | Primary key. Globally unique identifier for this specific calendar cycle. |
| `school_id` | `UUID` | No | Foreign key linking to the specific school. Driven by RLS. |
| `name` | `VARCHAR` | No | Display name of the academic year (e.g., "2026", "2026/2027"). |
| `start_date` | `DATE` | No | The calendar start day of the academic cycle. |
| `end_date` | `DATE` | No | The calendar end day of the academic cycle. |
| `is_current` | `BOOLEAN` | No | Defaults to `false`. Only one year per school can be true at a time. Drives active dashboard fallback filters. |

---

### 8. Academic Term Schema (`academic_terms` table)
Splits the academic year into actionable grading, tracking, and reporting windows.

| Field Name | Data Type | Nullable | Domain Rules & Database Constraints |
| --- | --- | --- | --- |
| `id` | `UUID` | No | Primary key. Globally unique identifier for this term. |
| `academic_year_id` | `UUID` | No | Foreign key linking directly to the parent `academic_years` record. |
| `name` | `VARCHAR` | No | Display name of the academic term (e.g., "Term 1", "Term 2", "Semester 1"). |
| `start_date` | `DATE` | No | The calendar start day of the term. |
| `end_date` | `DATE` | No | The calendar end day of the term. |
| `is_current` | `BOOLEAN` | No | Defaults to `false`. Only one term per school can be true at any given time to drive current assessment pipelines. |

---

### 9. Student Profile Schema (`students` table)
Represents the core master record for a student identity inside the tenant database ecosystem.

| Field Name | Data Type | Nullable | Domain Rules & Database Constraints |
| --- | --- | --- | --- |
| `id` | `UUID` | No | Primary key. Globally unique system identifier. |
| `tenant_id` | `UUID` | No | Foreign key linking explicitly to the parent corporate account. Driven by RLS. |
| `first_name` | `VARCHAR` | No | Student's legal first name. |
| `middle_name` | `VARCHAR` | Yes | Student's middle name. |
| `last_name` | `VARCHAR` | No | Student's family surname. |
| `gender` | `VARCHAR` | No | Demographics categorization for institutional statutory metrics. |
| `date_of_birth` | `DATE` | No | Date of birth for tracking baseline registration limits and age-cohort validation. |
| `is_active` | `BOOLEAN` | No | Defaults to `true`. Flipping to `false` terminates current operational status while keeping history locked for log validation. |
| `created_at` | `TIMESTAMP` | No | System timestamp when profile row initialized. |

---

### 10. Subjects Schema (`subjects` table)
The master catalog of subjects, explicitly mapped to specific grade levels. Standardizing on 'subjects' keeps the schema universally compatible with international tracks like IGCSE, while seamlessly serving as the container for CBC 'Learning Areas'.

| Field Name | Data Type | Nullable | Domain Rules & Database Constraints |
| --- | --- | --- | --- |
| `id` | `UUID` | No | Primary key. |
| `grade_id` | `UUID` | No | Foreign key linking directly to the static `grades` blueprint table. |
| `name` | `VARCHAR` | No | Standardized name of the course or tracking domain (e.g., "Mathematics Activities", "Physics", "English Language"). |
| `code` | `VARCHAR` | No | Unique code shorthand for backend sorting and calculation loops (e.g., "CBC-MAT1", "IG-PHYS9"). |

---

### 11. Classes Schema (`classes` table)
Represents the physical classroom group instance (e.g., "Grade 1 East") processing together during an explicit annual cycle.

| Field Name | Data Type | Nullable | Domain Rules & Database Constraints |
| --- | --- | --- | --- |
| `id` | `UUID` | No | Primary key. Globally unique identifier for this class instance. |
| `school_id` | `UUID` | No | Foreign key linking to the campus. Driven by RLS. |
| `academic_year_id` | `UUID` | No | Foreign key binding the classroom group to an explicit annual iteration. |
| `grade_id` | `UUID` | No | Foreign key referencing the structural layout rules from the static `grades` lookup table. |
| `name` | `VARCHAR` | No | Explicit structural naming indicator for this section grouping context (e.g., "East", "West", "Alpha"). |
| `created_at` | `TIMESTAMP` | No | Internal timestamp creation flag. |

---

### 12. Class Teacher Schema (`class_teachers` table)
A join table handling administrative, pastoral care, and specialized subject accountability assignments for explicit annual classes.

| Field Name | Data Type | Nullable | Domain Rules & Database Constraints |
| --- | --- | --- | --- |
| `id` | `UUID` | No | Primary key. |
| `class_id` | `UUID` | No | Foreign key connecting directly into the active annual `classes` layout. |
| `user_id` | `UUID` | No | Foreign key targeting the explicit `users` profile matching teacher context via `memberships`. |
| `subject_id` | `UUID` | Yes | Foreign key pointing to the master `subjects` catalog. If `NULL`, this user is the primary classroom manager for all general subjects inside that class. |
| `is_primary` | `BOOLEAN` | No | Defaults to `true`. Restricts context allocations ensuring unique tracking accountability anchors. |
| `created_at` | `TIMESTAMP` | No | Timestamp tracking execution. |

---

### 13. Student Enrollment Schema (`student_enrollments` table)
Tracks chronological progress, active tracking states, and student groupings for individual school terms.

| Field Name | Data Type | Nullable | Domain Rules & Database Constraints |
| --- | --- | --- | --- |
| `id` | `UUID` | No | Primary key. Unique tracking indicator. |
| `student_id` | `UUID` | No | Foreign key linking directly to the structural `students` identity table. |
| `school_id` | `UUID` | No | Foreign key linking to the parent physical campus context. Driven by RLS. |
| `academic_year_id` | `UUID` | No | Foreign key binding the tracking cycle to a specific annual configuration. |
| `academic_term_id` | `UUID` | No | Foreign key targeting precise operational assessment terms. |
| `class_id` | `UUID` | Yes | Foreign key mapping the student directly to their assigned classroom cohort (`classes` table) for this window. |
| `status` | `ENUM` | No | State values bounded by `enrollment_status` values (`ACTIVE`, `SUSPENDED`, `TRANSFERRED`, `GRADUATED`). |
| `created_at` | `TIMESTAMP` | No | Timestamp flag. |

---

### 14. Strands Schema (`strands` table)
Represents broad curriculum themes linked directly to a specific subject.

| Field Name | Data Type | Nullable | Domain Rules & Database Constraints |
| --- | --- | --- | --- |
| `id` | `UUID` | No | Primary key. |
| `subject_id` | `UUID` | No | Foreign key pointing to the master `subjects` table. |
| `name` | `VARCHAR` | No | Standardized name of the strand theme (e.g., "Numbers", "Geometry"). |

---

### 15. Sub-Strands Schema (`sub_strands` table)
Refined topical focus blocks nested underneath a parent curriculum strand.

| Field Name | Data Type | Nullable | Domain Rules & Database Constraints |
| --- | --- | --- | --- |
| `id` | `UUID` | No | Primary key. |
| `strand_id` | `UUID` | No | Foreign key linking back up to the parent `strands` row. |
| `name` | `VARCHAR` | No | Targeted topic or chapter title (e.g., "Fractions", "Time"). |

---

### 16. Learning Outcomes Schema (`learning_outcomes` table)
The atomic criteria leaf nodes. These represent the literal skill-based observations teachers look for when grading in a CBC classroom.

| Field Name | Data Type | Nullable | Domain Rules & Database Constraints |
| --- | --- | --- | --- |
| `id` | `UUID` | No | Primary key. |
| `sub_strand_id` | `UUID` | No | Foreign key pointing directly to the parent `sub_strands` block. |
| `description` | `TEXT` | No | The exact competency behavior being measured. |

---

## 📝 Daily Grade Capture: Formative Evaluation Tables

Once the blueprint structure exists, teachers need transactional tables to record actual day-to-day evaluations.

### 17. Formative Tasks Schema (`formative_tasks` table)
An assignment, practical exercise, or observation event created by a teacher in their workspace.

| Field Name | Data Type | Nullable | Domain Rules & Database Constraints |
| --- | --- | --- | --- |
| `id` | `UUID` | No | Primary key. |
| `class_id` | `UUID` | No | Foreign key pointing to the targeted annual `classes` room instance. |
| `academic_term_id` | `UUID` | No | Foreign key defining the active term window constraints. |
| `learning_outcome_id`| `UUID` | No | Foreign key linking this specific exercise to the competency outcome it validates. |
| `title` | `VARCHAR` | No | Display name of the assignment (e.g., "Market Day Counting Exercise"). |
| `date_administered` | `DATE` | No | The calendar day the observation or task took place. |
| `created_by` | `UUID` | No | Foreign key mapping to the `users` profile of the grading teacher. |

---

### 18. Task Evaluations Schema (`task_evaluations` table)
The transactional ledger row where a specific student's performance is locked. In CBC, this does not track numbers out of 100, but rather strict assessment levels.

| Field Name | Data Type | Nullable | Domain Rules & Database Constraints |
| --- | --- | --- | --- |
| `id` | `UUID` | No | Primary key. Unique transaction tracking ID. |
| `formative_task_id` | `UUID` | No | Foreign key pointing to the parent `formative_tasks` record. |
| `student_id` | `UUID` | No | Foreign key tracking the assessed student. |
| `score_level` | `ENUM` | No | Must map strictly to CBC standard rubrics: `EE` (Exceeding Expectation), `ME` (Meeting Expectation), `AE` (Approaching Expectation), or `BE` (Below Expectation). |
| `teacher_remarks` | `TEXT` | Yes | Qualitative feedback notes, which are highly critical for statutory CBC reporting metrics. |
| `updated_at` | `TIMESTAMP` | No | System timestamp tracking modifications for audit validation chains. |

---

## 🛡️ Non-Negotiable Coding Agent Guardrails

When drafting route handlers or database queries, the coding agent must strictly enforce these architectural guardrails:

**1. The RLS Tenant Wall**
Unless the user's role is explicitly evaluated as `SYSTEM_ADMIN`, the `tenant_id` context must be injected into every backend query pipeline. Cross-tenant leaks are blocked at the PostgreSQL level via strict Row-Level Security.

**2. The RLS School Wall**
Unless the user's role evaluates as `SYSTEM_ADMIN` or they carry a tenant-wide `SCHOOL_ADMIN` membership, the session's active `school_id` context must be injected into every backend query pipeline to block cross-campus data leaks at the database level.

**3. Auth Logic Separation**
Do not introduce local password hashing mechanisms, magic tokens, or session tables into this schema. Authentication is managed entirely by Stytch, linking directly to the immutable `external_auth_id` and system `id` properties.

**4. Contextual Scope & Permission Validation**
Permissions must be validated against the active user context inside the `memberships` table for the requested `school_id`. A single user could carry a `TEACHER` scope at school campus A while holding a `SCHOOL_ADMIN` scope at school campus B.

**5. Teacher Roster Visibility**
Users operating under active `TEACHER` membership scopes are dynamically limited by backend REST handlers. They are authorized to interact only with student cohorts, subjects, and grade structures to which they are linked through active class roster join tables.

**6. The Assessment Lock Boundary**
Student progress tracks, competency rubrics, and attendance logs must always carry a foreign key pointing to `academic_term_id`. When an API request attempts to modify grades, the backend must verify that the term's `end_date` has not passed.

**7. Cascade Deactivations**
If an academic year is marked as closed (`is_current = false`), all child terms must instantly treat their active status as closed. This prevents teachers from accidentally uploading marks into a past calendar year.

**8. Single Active Term Constraint**
The database or backend middleware must enforce that only a single `academic_term` record can have `is_current = true` per `school_id`. Activating a new term must automatically toggle the previous term's `is_current` status to false.

**9. Primary Class Teacher & Subject Uniqueness**
The database or backend middleware must enforce that only **one** record in `class_teachers` can be marked as `is_primary = true` per `class_id` where `subject_id IS NULL`. This ensures a class has exactly one main pastoral headteacher, while still allowing multiple specialized subject teachers to be attached to the same class.

**10. Dual-Key Student RLS Matching**
When reading student records, the backend query pipeline must match the student's `tenant_id` against the transaction context, and validate that the requesting user has an active membership role at the specific `school_id` found in the student's current active row within `student_enrollments`.

**11. Single Active Term Enrollment Constraint**
The database or middleware validation layer must ensure a student carries exactly **one** active enrollment row (`status = 'ACTIVE'`) per `academic_term_id`. A student cannot be simultaneously enrolled in multiple terms or across different sister campuses during the same calendar block.

**12. Teacher Scope Enforcement for Assignments**
When creating an entry in `formative_tasks`, the backend validation service must cross-reference the requesting `user_id` against the `class_teachers` table for that `class_id`. The transaction must be rejected unless the teacher is registered to teach that explicit class (either globally or for the specific parent `subject_id` bound to that learning outcome).

**13. The Historical Evaluation Lock**
Mutations to values within `task_evaluations` are governed strictly by the assessment window boundary. If the `end_date` found in the parent `academic_terms` table has elapsed, the backend API route must block all `UPDATE` or `DELETE` requests unless executed by an authorized `SCHOOL_ADMIN`.