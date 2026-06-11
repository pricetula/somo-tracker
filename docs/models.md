# Somotracker Architecture: Schema Specifications

This document defines the unified identity model, role hierarchies, system constraints, database layouts, and non-negotiable coding agent logic boundaries for Somotracker. These rules dictate multi-tenant isolation, backend API validation gates, and frontend dashboard layouts.

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

## 📐 Part 1: Unified Data Architecture Blueprint

### 1. Tenant Schema (`tenants` table)
Represents the core organization or corporate entity. Every business account maps to a record here.

| Field | Type | Constraints | Description |
| :--- | :--- | :--- | :--- |
| `id` | UUID | Primary Key | Unique internal identifier for the tenant. |
| `name` | String | Required | Display name of the parent organization/corporate group. |
| `slug` | String | Unique, Indexed | URL-friendly identifier (e.g., `acme-corp`) for custom routing. |
| `created_at` | Timestamp | Auto-generated | Timestamp when the tenant was created. |

### 2. User Schema (`users` table)
Every user identity in the system shares this core schema. Authentication states and login tokens are offloaded externally to Stytch.

| Field Name | Data Type | Nullable | Domain Rules & Database Constraints |
| --- | --- | --- | --- |
| `id` | `UUID` | No | Primary key. Globally unique system identifier. |
| `email` | `VARCHAR` | No | Normalized, unique communication key. Secondary unique index used for Stytch mapping. |
| `tenant_id` | `UUID` | Yes | Foreign key linking the user to their tenant. Must be NULL for `SYSTEM_ADMIN`, but strictly MANDATORY for all school-level users to drive Row-Level Security (RLS). |
| `first_name` | `VARCHAR` | No | Target for UI dashboard personalization. |
| `last_name` | `VARCHAR` | No | User profile family name. |
| `is_active` | `BOOLEAN` | No | Defaults to `true`. Setting to `false` instantly kills system access globally while keeping historical records intact for audit trails. |
| `external_auth_id` | `VARCHAR` | No | Unique, Indexed. The immutable Stytch User ID used for mapping active auth sessions. |
| `created_at` | `TIMESTAMP` | No | Timestamp when the user profile was initialized. |
| `updated_at` | `TIMESTAMP` | No | Timestamp of the most recent profile change. |

### 3. Education System Schema (`education_systems` table)
Represents the education system which a learning institution uses and determines grading styles, rubrics, and structural constraints for different curriculums.

| Field | Type | Constraints | Description |
| :--- | :--- | :--- | :--- |
| `id` | UUID | Primary Key | Unique internal identifier for the education system. |
| `name` | String | Required | Display name of the curriculum system (e.g., "Kenya CBC", "Cambridge IGCSE"). |
| `country_code` | String | Required | ISO country code where the education system is implemented (e.g., `KE`). |

### 4. Grade Level Schema (`grades` table)
Static lookup catalog defining the permanent academic learning stages within an education system blueprint.

| Field Name | Data Type | Nullable | Domain Rules & Database Constraints |
| --- | --- | --- | --- |
| `id` | `UUID` | No | Primary key. |
| `education_system_id` | `UUID` | No | Foreign key linking to the parent `education_systems` schema. |
| `name` | `VARCHAR` | No | Standardized blueprint name (e.g., "Grade 1", "Form 1", "Year 7"). |
| `sequence_order` | `INTEGER` | No | Numeric sequence tracking (e.g., 1 for PP1, 2 for PP2, 3 for Grade 1) to fuel automated promotions. |

### 5. School Schema (`schools` table)
Represents individual physical school campuses or academic sections operating under a corporate Tenant.

| Field | Type | Constraints | Description |
| :--- | :--- | :--- | :--- |
| `id` | UUID | Primary Key | Unique internal identifier for the specific school/campus. |
| `name` | String | Required | Display name of the school campus. |
| `education_system_id` | `UUID` | No | Foreign key linking to the active `education_systems` track. |
| `tenant_id` | `UUID` | No | Foreign key linking explicitly to the parent corporate `tenants` account. Driven by RLS. |

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

### 10. Subjects Schema (`subjects` table)
The master catalog of subjects, explicitly mapped to specific grade levels. Standardizing on 'subjects' keeps the schema universally compatible with international tracks like IGCSE, while seamlessly serving as the container for CBC 'Learning Areas'.

| Field Name | Data Type | Nullable | Domain Rules & Database Constraints |
| --- | --- | --- | --- |
| `id` | `UUID` | No | Primary key. |
| `grade_id` | `UUID` | No | Foreign key linking directly to the static `grades` blueprint table. |
| `name` | `VARCHAR` | No | Standardized name of the course or tracking domain (e.g., "Mathematics Activities", "Physics", "English Language"). |
| `code` | `VARCHAR` | No | Unique code shorthand for backend sorting and calculation loops (e.g., "CBC-MAT1", "IG-PHYS9"). |

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

### 14. Strands Schema (`strands` table)
Represents broad curriculum themes linked directly to a specific subject (used extensively in CBC and outcome-based tracks).

| Field Name | Data Type | Nullable | Domain Rules & Database Constraints |
| --- | --- | --- | --- |
| `id` | `UUID` | No | Primary key. |
| `subject_id` | `UUID` | No | Foreign key pointing to the master `subjects` table. |
| `name` | `VARCHAR` | No | Standardized name of the strand theme (e.g., "Numbers", "Geometry"). |

### 15. Sub-Strands Schema (`sub_strands` table)
Refined topical focus blocks nested underneath a parent curriculum strand.

| Field Name | Data Type | Nullable | Domain Rules & Database Constraints |
| --- | --- | --- | --- |
| `id` | `UUID` | No | Primary key. |
| `strand_id` | `UUID` | No | Foreign key linking back up to the parent `strands` row. |
| `name` | `VARCHAR` | No | Targeted topic or chapter title (e.g., "Fractions", "Time"). |

### 16. Learning Outcomes Schema (`learning_outcomes` table)
The atomic criteria leaf nodes. These represent the literal skill-based observations teachers look for when grading in a CBC classroom.

| Field Name | Data Type | Nullable | Domain Rules & Database Constraints |
| --- | --- | --- | --- |
| `id` | `UUID` | No | Primary key. |
| `sub_strand_id` | `UUID` | No | Foreign key pointing directly to the parent `sub_strands` block. |
| `description` | `TEXT` | No | The exact competency behavior being measured. |

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

### 18. Task Evaluations Schema (`task_evaluations` table)
The transactional ledger row where a specific student's performance is locked. In CBC, this tracks strict evaluation rubrics rather than percentages.

| Field Name | Data Type | Nullable | Domain Rules & Database Constraints |
| --- | --- | --- | --- |
| `id` | `UUID` | No | Primary key. Unique transaction tracking ID. |
| `formative_task_id` | `UUID` | No | Foreign key pointing to the parent `formative_tasks` record. |
| `student_id` | `UUID` | No | Foreign key tracking the assessed student. |
| `score_level` | `ENUM` | No | Must map strictly to CBC standard rubrics: `EE` (Exceeding Expectation), `ME` (Meeting Expectation), `AE` (Approaching Expectation), or `BE` (Below Expectation). |
| `teacher_remarks` | `TEXT` | Yes | Qualitative feedback notes, which are highly critical for statutory CBC reporting metrics. |
| `updated_at` | `TIMESTAMP` | No | System timestamp tracking modifications for audit validation chains. |

### 19. Summative Assessments Schema (`summative_assessments` table)
Represents formal terminal examinations or standardized block assessments managed at the institutional or national level for a given term (e.g., end-of-term exams, KPSEA layouts, or traditional style tests).

| Field Name | Data Type | Nullable | Domain Rules & Database Constraints |
| --- | --- | --- | --- |
| `id` | `UUID` | No | Primary key. |
| `school_id` | `UUID` | No | Foreign key mapping to the physical campus. Driven by RLS. |
| `academic_term_id` | `UUID` | No | Foreign key specifying the exact lock window for the evaluation. |
| `subject_id` | `UUID` | No | Foreign key linking directly to the parent `subjects` schema node. |
| `max_points` | `INTEGER` | No | Total achievable score raw boundary (e.g., `100` for traditional percentages, or `50` for localized statutory tests). |

### 20. Summative Scores Schema (`summative_scores` table)
Stores the physical scores achieved by individual students on formal terminal examinations.

| Field Name | Data Type | Nullable | Domain Rules & Database Constraints |
| --- | --- | --- | --- |
| `id` | `UUID` | No | Primary key. Unique transaction tracking ID. |
| `summative_assessment_id` | `UUID` | No | Foreign key targeting the parent examination metadata profile. |
| `student_id` | `UUID` | No | Foreign key targeting the explicit student profile record. |
| `raw_score` | `NUMERIC(5,2)` | No | The absolute numeric score earned by the student (e.g., `20.00`). Must be less than or equal to the parent assessment's `max_points`. |
| `teacher_remarks` | `TEXT` | Yes | Custom terminal textual notes for end-of-term review sheets. |
| `updated_at` | `TIMESTAMP` | No | System timestamp tracking modifications for audit validation chains. |

### 21. Attendance Periods Schema (`attendance_periods` table)
Represents a specific, scheduled lesson block or class period on the school timetable. Every attendance sheet must be anchored to an explicit subject lesson.

| Field Name | Data Type | Nullable | Domain Rules & Database Constraints |
| --- | --- | --- | --- |
| `id` | `UUID` | No | Primary key. |
| `school_id` | `UUID` | No | Foreign key mapping to the physical campus. Driven by RLS. |
| `academic_term_id` | `UUID` | No | Foreign key anchoring the sheet to the active calendar window. |
| `class_id` | `UUID` | No | Foreign key referencing the specific classroom group instance (`classes` table). |
| `subject_id` | `UUID` | No | Foreign key linking directly to the master `subjects` table. Strictly MANDATORY to drive lesson-by-lesson tracking. |
| `date_recorded` | `DATE` | No | The calendar day the lesson occurred. Indexed for fast dashboard aggregation. |

### 22. Attendance Logs Schema (`attendance_logs` table)
The transactional ledger row where every individual student's presence status is officially logged for a given lesson period.

| Field Name | Data Type | Nullable | Domain Rules & Database Constraints |
| --- | --- | --- | --- |
| `id` | `UUID` | No | Primary key. Unique transaction tracking ID. |
| `attendance_period_id` | `UUID` | No | Foreign key linking directly to the parent scheduled lesson period header. |
| `student_id` | `UUID` | No | Foreign key targeting the explicit student profile record. |
| `status` | `ENUM` | No | Must map strictly to tracking states: `PRESENT`, `ABSENT`, `LATE`, or `EXCUSED`. |
| `remarks` | `VARCHAR` | Yes | Qualitative notes explaining an absence or lateness (e.g., "Medical clearance", "Sports practice"). |
| `recorded_by` | `UUID` | No | Foreign key mapping to the `users` profile of the staff member clocking the register. |

### 23. Timetable Slots Schema (`timetable_slots` table)
Defines the scheduled academic and non-instructional blocks tracking which subjects are taught to which classes, by which teachers, and where. Fully accommodates variable, non-uniform day timings natively.

| Field Name | Data Type | Nullable | Domain Rules & Database Constraints |
| --- | --- | --- | --- |
| `id` | `UUID` | No | Primary key. |
| `school_id` | `UUID` | No | Foreign key mapping to the physical campus. Driven by RLS. |
| `class_id` | `UUID` | No | Foreign key pointing to the target room cohort (`classes` table). |
| `subject_id` | `UUID` | No | Foreign key linking to the master `subjects` catalog. |
| `teacher_id` | `UUID` | No | Foreign key mapping to the `users` profile instructing this specific block. |
| `room_id` | `UUID` | Yes | Foreign key pointing to a physical campus room structure. |
| `day_of_week` | `INTEGER` | No | Numeric index representation: `1` (Monday) through `7` (Sunday). |
| `start_time` | `TIME` | No | The official scheduled start time clock parameter. |
| `end_time` | `TIME` | No | The official scheduled end time clock parameter. |

### 24. Student Health Profiles Schema (`student_health_profiles` table)
Maintains baseline critical medical indicators and biological flags per individual student context.

| Field Name | Data Type | Nullable | Domain Rules & Database Constraints |
| --- | --- | --- | --- |
| `id` | `UUID` | No | Primary key. |
| `student_id` | `UUID` | No | Foreign key mapping 1:1 explicitly to the master `students` profile record. |
| `blood_group` | `VARCHAR` | Yes | Standard blood categorization syntax (e.g., `A+`, `O-`). |
| `allergies` | `TEXT[]` | Yes | Array listing diagnosed triggers for instant profile workspace flags. |
| `chronic_conditions` | `TEXT[]` | Yes | Array tracking long-term medical conditions (e.g., Asthma, Diabetes). |
| `emergency_instructions`| `TEXT` | Yes | Operational directive text for immediate execution during first-aid incidents. |

### 25. Medical Incidents Ledger (`medical_incidents` table)
The live transactional log record tracking individual health emergency events, sickbay visit transactions, or treatment administration logs.

| Field Name | Data Type | Nullable | Domain Rules & Database Constraints |
| --- | --- | --- | --- |
| `id` | `UUID` | No | Primary key. Unique ledger entry transaction index. |
| `student_id` | `UUID` | No | Foreign key identifying the treated student target profile. |
| `incident_timestamp` | `TIMESTAMP`| No | The precise date and time the medical emergency context occurred. |
| `symptoms` | `TEXT` | No | Text summary documenting client presentation details. |
| `action_taken` | `TEXT` | No | Explicit clinical logging steps applied (e.g., medications given, emergency referrals). |
| `logged_by` | `UUID` | No | Foreign key mapping directly to the `users` profile of the administrator or nurse. |

### 26. Fee Categories Schema (`fee_categories` table)
Acts as the global master catalog of billing items and invoicing dimensions across the school campus.

| Field Name | Data Type | Nullable | Domain Rules & Database Constraints |
| --- | --- | --- | --- |
| `id` | `UUID` | No | Primary key. |
| `school_id` | `UUID` | No | Foreign key mapping to the physical campus. Driven by RLS. |
| `name` | `VARCHAR` | No | Display name of the fee item type (e.g., "Tuition Fee"). |
| `is_mandatory` | `BOOLEAN` | No | Defaults to `true`. If false, represents an optional add-on subscription. |

### 27. Fee Templates Schema (`fee_templates` table)
The configuration layer where administrators define baseline fee structures for an explicit grade level and term block.

| Field Name | Data Type | Nullable | Domain Rules & Database Constraints |
| --- | --- | --- | --- |
| `id` | `UUID` | No | Primary key. |
| `school_id` | `UUID` | No | Foreign key mapping to the physical campus. Driven by RLS. |
| `academic_term_id` | `UUID` | No | Foreign key binding the structure to a specific billing calendar term. |
| `grade_id` | `UUID` | No | Foreign key linking to the static `grades` lookup table. |
| `fee_category_id` | `UUID` | No | Foreign key pointing to the target item type in `fee_categories`. |
| `amount` | `NUMERIC(12,2)`| No | The standard baseline cost defined for this item cohort structure. |
| *Constraint* | *Composite* | *No* | `UNIQUE(academic_term_id, grade_id, fee_category_id)` to prevent duplicate structural configuration rules. |

### 28. Student Invoices Schema (`student_invoices` table)
The live transactional billing row assigned to a student's ledger representing exact liabilities owed for a given term block.

| Field Name | Data Type | Nullable | Domain Rules & Database Constraints |
| --- | --- | --- | --- |
| `id` | `UUID` | No | Primary key. Unique transactional invoice tracker. |
| `student_id` | `UUID` | No | Foreign key targeting the explicit student profile record. **Composite Performance Index Active.** |
| `academic_term_id` | `UUID` | No | Foreign key locking the liability context to a specific operational term. **Composite Performance Index Active.** |
| `fee_category_id` | `UUID` | No | Foreign key pointing to the specific item being billed. |
| `amount_due` | `NUMERIC(12,2)`| No | The actual amount billed to this student (inherits from template, custom modifiable for scholarships). |
| `amount_paid` | `NUMERIC(12,2)`| No | The cumulative total of cleared payments applied against this invoice line. Defaults to 0.00. |
| *Constraint* | *Composite* | *No* | `UNIQUE(student_id, academic_term_id, fee_category_id)` to explicitly prevent duplicate line-item billing records. |


## 🛡️ Part 2: Non-Negotiable Coding Agent Guardrails

When drafting route handlers, database rules, or validation middleware, the coding agent must strictly satisfy this checklist:

### 🔐 Multi-Tenant & Campus Access Controls
**1. The RLS Tenant Wall**
Unless the user's role is explicitly evaluated as `SYSTEM_ADMIN`, the `tenant_id` context must be injected into every backend query pipeline. Cross-tenant leaks are blocked at the database level via Row-Level Security.

**2. The RLS School Wall**
Unless the user's role evaluates as `SYSTEM_ADMIN` or they carry a tenant-wide `SCHOOL_ADMIN` membership, the session's active `school_id` context must be injected into every backend query pipeline to block cross-campus data leaks.

**3. Auth Logic Separation**
Do not introduce local password hashing mechanisms or custom session tables into this schema. Authentication is managed entirely by Stytch, linking directly to the immutable `external_auth_id` and system `id` properties.

**4. Contextual Scope & Permission Validation**
Permissions must be validated against the active user context inside the `memberships` table for the requested `school_id`. A single user could carry a `TEACHER` scope at school campus A while holding a `SCHOOL_ADMIN` scope at school campus B.

**5. Teacher Roster Visibility**
Users operating under active `TEACHER` membership scopes are dynamically limited by backend REST handlers. They are authorized to interact only with student cohorts, subjects, and grade structures to which they are linked through active class roster join tables.

**10. Dual-Key Student RLS Matching**
When reading student records, the backend query pipeline must match the student's `tenant_id` against the transaction context, and validate that the requesting user has an active membership role at the specific `school_id` found in the student's current active row within `student_enrollments`.

### 📅 Academic Calendar & Enrollment Constraints
**6. The Assessment Lock Boundary**
Student progress tracks, competency rubrics, and attendance logs must always carry a foreign key pointing to `academic_term_id`. When an API request attempts to modify grades, the backend must verify that the term's `end_date` has not passed.

**7. Cascade Deactivations**
If an academic year is marked as closed (`is_current = false`), all child terms must instantly treat their active status as closed. This prevents teachers from accidentally uploading marks into a past calendar year.

**8. Single Active Term Constraint**
The database or backend middleware must enforce that only a single `academic_term` record can have `is_current = true` per `school_id`. Activating a new term must automatically toggle the previous term's status to false.

**9. Primary Class Teacher & Subject Uniqueness**
The database or backend middleware must enforce that only **one** record in `class_teachers` can be marked as `is_primary = true` per `class_id` where `subject_id IS NULL`. This ensures a class has exactly one main pastoral headteacher.

**11. Single Active Term Enrollment Constraint**
The database or middleware validation layer must ensure a student carries exactly **one** active enrollment row (`status = 'ACTIVE'`) per `academic_term_id`. A student cannot be simultaneously enrolled in multiple terms or across different sister campuses during the same calendar block.

### 📝 Evaluation & Score Capture Controls
**12. Teacher Scope Enforcement for Assignments**
When creating an entry in `formative_tasks`, the backend validation service must cross-reference the requesting `user_id` against the `class_teachers` table for that `class_id`. The transaction must be rejected unless the teacher is registered to teach that explicit class.

**13. The Historical Evaluation Lock**
Mutations to values within `task_evaluations` are governed strictly by the assessment window boundary. If the `end_date` found in the parent `academic_terms` table has elapsed, the backend API route must block all `UPDATE` or `DELETE` requests unless executed by an authorized `SCHOOL_ADMIN`.

**14. Summative Score Ceiling Validation**
The backend validation middleware must catch and block any attempt to save a record into `summative_scores` where the `raw_score` is greater than the corresponding `max_points` defined in the parent `summative_assessments` structural rule.

**15. Multi-Tenant Assessment Boundaries**
When looking up or inserting records into `summative_assessments`, the query path must validate that the target `subject_id` belongs to an education system active within the user's tenant scope, preventing cross-tenant curriculum modifications.

### ⏱️ Attendance & Timetable Anti-Clash Controls
**16. Roster-Scoped Attendance Verification**
When a user attempts to write rows to `attendance_logs`, the API middleware validation layer must verify that the `recorded_by` user has an active `class_teachers` assignment or a `SCHOOL_ADMIN` role for the targeted class context and subject.

**17. Past-Term Attendance Lock**
The database or backend middleware must block any `INSERT`, `UPDATE`, or `DELETE` statements on `attendance_periods` or its child log records if the corresponding `academic_term_id` has `is_current = false`.

**18-A. Class Conflict Prevention**
The backend validation service must block any transaction on `timetable_slots` if the requested time interval matches an existing row's timeframe for the same `class_id` on the same `day_of_week`. A class cohort cannot have two lessons simultaneously.

**18-B. Teacher Conflict Prevention**
The backend validation layer must check and reject adjustments if the assigned `teacher_id` is already scheduled to instruct another class group anywhere on the campus during an overlapping time interval on that specific `day_of_week`.

**18-C. Room Conflict Prevention**
When a physical space parameter (`room_id`) is submitted, the system must drop the operation if that exact location context is already booked by another group within the intersecting time boundaries on that day.

**20. Non-Instructional Attendance Bypass**
The backend attendance generation pipeline must instantly ignore any `timetable_slots` where the mapped `subject_id` corresponds to a system-defined non-instructional code wrapper (e.g., prefix `SYS-` for Lunch, Breaks, or Assemblies). No transactional rows inside `attendance_periods` or `attendance_logs` shall be initialized for non-academic tracking periods.

### 🏥 Medical & Health Data Security
**19. Health Profile Access Restriction**
The API routing stack must explicitly restrict queries targeting `student_health_profiles` and `medical_incidents`. Data hydration is strictly forbidden unless the requesting security context evaluates to a tenant-wide `SCHOOL_ADMIN`, a designated campus `SUPPORT_STAFF` profile, or a `TEACHER` who is actively linked to that student cohort through a class-roster ledger block.

### 💰 Financial & Invoicing Controls
**21. Strict Invoice Balance Ceiling**
The backend validation engine must intercept and block any transaction updating `student_invoices` if the modified `amount_paid` evaluates to a value greater than `amount_due`. A student line-item account cannot carry an overpaid status directly on an individual invoice node.

**22. Automated Mass Invoicing Trigger**
When an administrator executes a fee template application, the backend runtime must process bulk operations to safely insert transactional child rows into `student_invoices` for every student tracking as `status = 'ACTIVE'` inside `student_enrollments` matching that target `grade_id` and `academic_term_id`.