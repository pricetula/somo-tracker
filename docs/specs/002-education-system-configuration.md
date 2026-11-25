# SPEC-002: Multi-Country Education System Configuration

Endpoint: N/A (domain configuration domain)

## Metadata

| Field | Value |
|-------|-------|
| id | SPEC-002 |
| title | Multi-Country Education System Configuration |
| status | proposed |
| depends_on | specs/glossary.md, specs/001-auth-otp-request.md |
| api_endpoint | N/A (backend domain configuration) |
| owner | backend |

## Summary

Somotracker must support the current primary-to-high-school education structures of Kenya, Rwanda, Tanzania, and Uganda within a single codebase. Instead of hardcoding country-specific grades, terms, or subjects, the system uses **table-driven configuration**:

- **Global seed data** (countries, education systems) is seeded on server start and is not owned by any school.
- **Per-school seed data** (curriculum, grade years, subjects, academic periods) is generated when a school is created, based on the education system the school selects.
- All configuration uses **normalized relational rows**. JSONB is strictly prohibited.
- Scope is **primary to high school only**. Early childhood and tertiary are out of scope.
- No cross-border student transfers or certificate equivalency are supported.
- Language of instruction defaults to English; schools may override.
- Assessments are local and teacher-entered only. No integration with national exam boards (KNEC, NESA, NECTA, UNEB, etc.).
- Grade placement is by level. Age is informational only and never affects placement decisions.

## Global Configuration (seeded on server start)

These tables are managed at the platform level and apply across all schools.

### `countries`

Registry of active countries.

| Column | Type | Notes |
|--------|------|-------|
| id | UUID | Primary key |
| name | TEXT | e.g. Kenya, Rwanda, Tanzania, Uganda |
| code | TEXT | ISO-like code, unique |

### `education_systems`

One row per country representing its **current** education system. Legacy systems (e.g. Kenya 8-4-4) are not modeled.

| Column | Type | Notes |
|--------|------|-------|
| id | UUID | Primary key |
| country_id | UUID | References `countries.id` |
| name | TEXT | e.g. "CBC", "9-Year Basic", "7-4-2-3", "7-4-2" |
| years_primary | INT | Number of years in primary level |
| years_lower_secondary | INT | Number of years in lower secondary level |
| years_upper_secondary | INT | Number of years in upper secondary level |
| default_language | TEXT | Default language of instruction (default: "English") |

### `education_levels`

Canonical levels shared across all systems.

| Column | Type | Notes |
|--------|------|-------|
| id | UUID | Primary key |
| name | TEXT | e.g. "Primary", "Lower Secondary", "Upper Secondary" |
| sort_order | INT | Ordering within the system |

### `education_system_levels`

Concrete mapping of years per level within a specific education system.

| Column | Type | Notes |
|--------|------|-------|
| id | UUID | Primary key |
| education_system_id | UUID | References `education_systems.id` |
| education_level_id | UUID | References `education_levels.id` |
| year_count | INT | Number of years this level spans in this system |

### `education_system_grading_scales`

Default grading scale bands for an education system. Normalized rows only.

| Column | Type | Notes |
|--------|------|-------|
| id | UUID | Primary key |
| education_system_id | UUID | References `education_systems.id` |
| name | TEXT | e.g. "CBC Scale", "8-Point Scale" |
| is_default | BOOL | Whether this is the system default |

### `education_system_grading_bands`

Individual grade bands within a scale. Supports both percentage-based and rubric-based (competency) grading. Normalized rows only — no JSONB.

| Column | Type | Notes |
|--------|------|-------|
| id | UUID | Primary key |
| grading_scale_id | UUID | References `education_system_grading_scales.id` |
| grade | TEXT | e.g. "A", "B", "C", "Level 4", "EE1", "ME1" |
| scale_type | TEXT | `percentage` or `rubric`. Default `percentage`. |
| rubric_level | TEXT | When `scale_type='rubric'`: e.g. `EE1` (Exceeds Expectations), `ME1` (Meets Expectations), `AE1` (Approaching Expectations), `BE1` (Below Expectations). NULL for percentage scales. |
| min_score | DECIMAL | Inclusive lower bound (percentage). For rubric bands, maps the rubric level to a percentage floor. |
| max_score | DECIMAL | Inclusive upper bound (percentage). For rubric bands, maps the rubric level to a percentage ceiling. |
| sort_order | INT | Ordering within the scale |

### `education_system_subjects`

Default subject catalog per education system per level.

| Column | Type | Notes |
|--------|------|-------|
| id | UUID | Primary key |
| education_system_id | UUID | References `education_systems.id` |
| education_level_id | UUID | References `education_levels.id` |
| subject_code | TEXT | e.g. "MATH", "ENG", "SCI" |
| subject_name | TEXT | e.g. "Mathematics", "English", "Science" |
| is_compulsory_default | BOOL | System-level default for compulsory flag |

### `education_system_term_structures`

Default term names and count per education system.

| Column | Type | Notes |
|--------|------|-------|
| id | UUID | Primary key |
| education_system_id | UUID | References `education_systems.id` |
| term_name | TEXT | e.g. "Term 1", "Term 2", "Term 3" |
| term_sort_order | INT | Order within the academic year |

---

## School-Level Configuration (seeded when a school is created)

When a new school is created, it selects an `education_system_id`. The system then seeds the school-specific configuration tables based on that education system's defaults. A school admin may later override these settings.

### `schools`

| Column | Type | Notes |
|--------|------|-------|
| id | UUID | Primary key |
| name | TEXT | School name |
| registration_number | TEXT | Local registration identifier |
| country_id | UUID | References `countries.id` |
| education_system_id | UUID | References `education_systems.id` (selected at creation) |
| school_type | TEXT | e.g. "public", "private", "boarding", "day" |
| language_of_instruction | TEXT | Override of system default; defaults to system default |

### `school_grading_scales`

School-specific override of the education system default grading scale.

| Column | Type | Notes |
|--------|------|-------|
| id | UUID | Primary key |
| school_id | UUID | References `schools.id` |
| name | TEXT | e.g. "Acme Custom Scale" |
| is_active | BOOL | Whether this is the school's active scale |

### `school_grading_bands`

Normalized grade bands for a school's active scale. Supports both percentage-based and rubric-based (competency) grading. No JSONB.

| Column | Type | Notes |
|--------|------|-------|
| id | UUID | Primary key |
| school_grading_scale_id | UUID | References `school_grading_scales.id` |
| grade | TEXT | e.g. "A", "B+", "EE1", "ME1" |
| scale_type | TEXT | `percentage` or `rubric`. Default `percentage`. |
| rubric_level | TEXT | When `scale_type='rubric'`: e.g. `EE1` (Exceeds Expectations), `ME1` (Meets Expectations), `AE1` (Approaching Expectations), `BE1` (Below Expectations). NULL for percentage scales. |
| min_score | DECIMAL | Inclusive lower bound (percentage). For rubric bands, maps the rubric level to a percentage floor. |
| max_score | DECIMAL | Inclusive upper bound (percentage). For rubric bands, maps the rubric level to a percentage ceiling. |
| sort_order | INT | Ordering within the scale |

### `school_subjects`

Subjects available at this school, per level. Seeded from `education_system_subjects`; school admin may mark compulsory/optional or deactivate subjects.

| Column | Type | Notes |
|--------|------|-------|
| id | UUID | Primary key |
| school_id | UUID | References `schools.id` |
| education_level_id | UUID | References `education_levels.id` |
| subject_code | TEXT | e.g. "MATH" |
| subject_name | TEXT | e.g. "Mathematics" |
| is_compulsory | BOOL | Per-school override of system default |
| is_active | BOOL | Whether the school offers this subject |

### `school_academic_years`

| Column | Type | Notes |
|--------|------|-------|
| id | UUID | Primary key |
| school_id | UUID | References `schools.id` |
| year_label | TEXT | e.g. "2025", "2026" |
| start_date | DATE | Academic year start |
| end_date | DATE | Academic year end |
| is_active | BOOL | Current active year |

### `school_academic_terms`

| Column | Type | Notes |
|--------|------|-------|
| id | UUID | Primary key |
| school_id | UUID | References `schools.id` |
| academic_year_id | UUID | References `school_academic_years.id` |
| term_name | TEXT | e.g. "Term 1" |
| start_date | DATE | Term start date (school-set) |
| end_date | DATE | Term end date (school-set) |
| sort_order | INT | Order within the year |

### `school_grade_years`

Concrete grade/year names within a level for this school. Seeded from `education_system_levels` + `education_system_subjects` context; e.g. "Grade 1", "P1", "Standard 1", "Form 1".

| Column | Type | Notes |
|--------|------|-------|
| id | UUID | Primary key |
| school_id | UUID | References `schools.id` |
| education_level_id | UUID | References `education_levels.id` |
| grade_name | TEXT | Local name: "Grade 4", "P4", "Standard 4", "S1" |
| grade_number | INT | Numeric sequence within the level |
| is_active | BOOL | Whether this grade year is offered |

---

## Shared Core Domain

### `streams`

A class/stream within a grade year (e.g. "Grade 4 East", "Form 2 Blue").

| Column | Type | Notes |
|--------|------|-------|
| id | UUID | Primary key |
| school_id | UUID | References `schools.id` |
| grade_year_id | UUID | References `school_grade_years.id` |
| stream_name | TEXT | e.g. "East", "Blue", "A" |
| academic_year_id | UUID | References `school_academic_years.id` |

### `teachers`

| Column | Type | Notes |
|--------|------|-------|
| id | UUID | Primary key |
| school_id | UUID | References `schools.id` |
| employee_number | TEXT | Unique within school |
| full_name | TEXT | |

### `students`

| Column | Type | Notes |
|--------|------|-------|
| id | UUID | Primary key |
| school_id | UUID | References `schools.id` |
| admission_number | TEXT | Unique within school |
| full_name | TEXT | |
| date_of_birth | DATE | **Informational only.** Never used for grade placement or level assignment. |

### `enrollments`

Links a student to a stream for an academic year.

| Column | Type | Notes |
|--------|------|-------|
| id | UUID | Primary key |
| student_id | UUID | References `students.id` |
| stream_id | UUID | References `streams.id` |
| academic_year_id | UUID | References `school_academic_years.id` |
| status | TEXT | e.g. "active", "transferred", "graduated" |

---

## Local Assessments (teacher-entered, no external APIs)

### `assessment_types`

| Column | Type | Notes |
|--------|------|-------|
| id | UUID | Primary key |
| name | TEXT | e.g. "Quiz", "Term Exam", "Final Year Exam" |

### `assessments`

| Column | Type | Notes |
|--------|------|-------|
| id | UUID | Primary key |
| stream_id | UUID | References `streams.id` |
| subject_id | UUID | References `school_subjects.id` |
| assessment_type_id | UUID | References `assessment_types.id` |
| academic_term_id | UUID | References `school_academic_terms.id` |
| title | TEXT | e.g. "Mid-Term Mathematics" |
| max_score | DECIMAL | |
| date_assessed | DATE | |
| created_by_teacher_id | UUID | References `teachers.id` |

### `assessment_results`

| Column | Type | Notes |
|--------|------|-------|
| id | UUID | Primary key |
| assessment_id | UUID | References `assessments.id` |
| student_id | UUID | References `students.id` |
| score | DECIMAL | Raw score obtained |
| grade | TEXT | Derived from school's active grading scale bands |
| entered_by_teacher_id | UUID | References `teachers.id` |
| entered_at | TIMESTAMPTZ | |

No `external_exam_board` or `national_results_api` tables exist per scope constraints.

---

## Business Rules

1. **Single education system per school.** A school adopts exactly one education system at creation. Changing it later is a data migration, not a toggle.
2. **School-level overrides.** Grading scales, subjects (compulsory/optional), term dates, and language of instruction are configurable per school after creation.
3. **Normalized relationships only.** All grading scales, bands, subjects, terms, and grade years are stored as normalized rows with foreign keys. JSONB is not used for configuration data.
4. **Primary to high school only.** Configuration covers Primary, Lower Secondary, and Upper Secondary levels only.
5. **No cross-border transfers.** Students and schools belong to one country context. No equivalency or migration logic between Kenyan, Rwandan, Tanzanian, or Ugandan certificates.
6. **English default.** Language of instruction defaults to the education system default (English). Schools may override.
7. **Age is informational.** `students.date_of_birth` is stored but never used to determine grade level, stream assignment, or progression.
8. **Manual assessments only.** Quizzes, term exams, and final-year results are created and scored by teachers within the school. No API integration with national examination bodies.
9. **Seeding on school creation.** When a school is created with an `education_system_id`, the system automatically inserts rows into `school_subjects`, `school_grade_years`, `school_academic_terms` (defaults), and `school_grading_scales` (based on system default), which the school admin may then edit.

---

## Acceptance Criteria

Given a super-admin creates a school and selects the Kenyan CBC education system:
- When the school is saved, then `school_subjects` contains Mathematics, English, Kiswahili, Science, and Social Studies for the primary level (seeded from system defaults).
- When the school is saved, then `school_grade_years` contains Grade 1 through Grade 6 for Primary, Grade 7–9 for Lower Secondary, Grade 10–12 for Upper Secondary.
- When the school is saved, then `school_grading_scales` contains one active scale seeded from the education system default.
- When a school admin edits a subject's `is_compulsory` flag, then the change persists for that school only and does not affect other schools using the same education system.

Given a teacher creates a quiz for Stream "Grade 4 East":
- When the teacher enters scores for each student, then `assessment_results` stores the raw score and the derived grade based on the school's active grading scale bands.
- When the teacher views the results, then grades are displayed per the school's configured scale (not another country's scale).

Given a student record with a date of birth:
- When the student is enrolled in a stream, then the date of birth is stored but does not influence which stream or grade year the student may join.

---

## Open Questions

1. **Changing education systems:** If a school wants to switch education systems after creation, is that supported as a data migration, or must a new school be created?
2. **Promotion workflow:** Who triggers end-of-year promotion (class teacher, headteacher, system admin)? Is there a formal "close academic year" process?
3. **Grading scale migration:** If a school changes its active grading scale mid-year, are historical results recalculated or left with the original grade letter?
4. **Schools table migration:** The core `schools` table (migration `00000000000002_core_domain.sql`) does not yet have `country_id`, `education_system_id`, `school_type`, `language_of_instruction`, or `registration_number`. A new migration must add these columns before a school can reference `education_system_id`.
5. **Education config migrations:** All education configuration tables (`countries`, `education_systems`, `education_levels`, `education_system_levels`, `education_system_grading_scales`, `education_system_grading_bands`, `education_system_subjects`, `education_system_term_structures`, `school_grading_scales`, `school_grading_bands`, `school_subjects`, `school_academic_years`, `school_academic_terms`, `school_grade_years`, `streams`, `teachers`, `students`, `enrollments`, `assessment_types`, `assessments`, `assessment_results`) exist only in this spec — no migration files exist yet. Migrations must be created, seeded, and tested before implementation.

---

## Links

- Glossary: `docs/specs/glossary.md`
- Auth flow: `docs/auth-flow.md`
- Auth architecture: `docs/adr/0001-authentication-architecture.md`
- Session management: `docs/adr/0003-session-management.md`
