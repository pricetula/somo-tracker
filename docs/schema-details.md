# Somotracker Database Schema

This document outlines the complete database schema for the Somotracker platform, including all tables, fields, relationships, enumerations, functions, triggers, and constraints.

## Extensions

The following PostgreSQL extensions are required:
- `btree_gist`: Supports GiST indexes on scalar data types, used for exclusion constraints
- `pgcrypto`: Provides cryptographic functions, used for `gen_random_uuid()` function

## Custom Functions

### `fn_set_updated_at()`
```sql
CREATE OR REPLACE FUNCTION fn_set_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;
```
Used as a trigger function to automatically update the `updated_at` column on record updates.

### `fn_timerange(day_of_week INT, start_time TIME, end_time TIME)`
```sql
CREATE OR REPLACE FUNCTION fn_timerange(day_of_week INT, start_time TIME, end_time TIME)
RETURNS tsrange AS $$
    SELECT tsrange(
        ('2024-01-01'::DATE + (day_of_week - 1)) + start_time,
        ('2024-01-01'::DATE + (day_of_week - 1)) + end_time,
        '[)'
    );
$$ LANGUAGE sql IMMUTABLE;
```
Maps day_of_week (1=Mon…7=Sun) onto base week 2024-01-01 so GiST exclusion constraints only conflict within the same day.

### `fn_validate_term_dates_within_year()`
```sql
CREATE OR REPLACE FUNCTION fn_validate_term_dates_within_year()
RETURNS TRIGGER AS $$
DECLARE
    v_year_start DATE;
    v_year_end   DATE;
BEGIN
    SELECT start_date, end_date INTO v_year_start, v_year_end
    FROM academic_years
    WHERE id = NEW.academic_year_id;

    IF (NEW.start_date < v_year_start OR NEW.end_date > v_year_end) THEN
        RAISE EXCEPTION 'Term dates (% to %) must fall within parent Academic Year bounds (% to %)',
            NEW.start_date, NEW.end_date, v_year_start, v_year_end;
    END IF;

    RETURN NEW;
END;
$$ LANGUAGE plpgsql;
```
Validates that term dates fall within the parent academic year bounds.

### `fn_current_tenant_id()`
```sql
CREATE OR REPLACE FUNCTION fn_current_tenant_id()
RETURNS UUID
LANGUAGE plpgsql
STABLE
AS $$
BEGIN
    RETURN NULLIF(current_setting('app.current_tenant_id', TRUE), '')::UUID;
EXCEPTION WHEN OTHERS THEN
    RETURN NULL;
END;
$$;
```
Returns the `tenant_id` set via `app.current_tenant_id` for the current session. Returns `NULL` if not set (which causes RLS policies to filter out ALL rows — safe by default). The application must `SET LOCAL app.current_tenant_id` before each request.

### `fn_resolve_session(p_token_hash TEXT)`
`SECURITY DEFINER` function that resolves a session by its SHA-256 `token_hash` **before the tenant is known** (the tenant is read from the session row itself, so it cannot run under tenant-scoped RLS). Returns `user_id`, `tenant_id`, `device_fingerprint`, highest-priority active `role`, active `school_id` (from `member_active_school` falling back to the highest-priority membership), and the full list of active `schools` (UUIDs as TEXT). Read-only and keyed by the unguessable token hash.

### `fn_pending_invitation_by_email(p_email TEXT)`
`SECURITY DEFINER` function that looks up a pending, non-expired invitation by case-insensitive email **before the tenant is known** (invite acceptance flow). Returns the invitation row fields needed to complete acceptance.

### `fn_rls_tenant_policy()`
Returns a `WHERE` clause fragment (`tenant_id = <current tenant>`) used to build RLS policies for tables without a `tenant_id` column.

### `fn_check_non_break_slot()`
```sql
CREATE OR REPLACE FUNCTION fn_check_non_break_slot()
RETURNS TRIGGER AS $$
BEGIN
    IF EXISTS (
        SELECT 1 FROM cbc_timetable_slots ts
        JOIN timetable_structures tstr ON tstr.id = ts.structure_id
        WHERE ts.id = NEW.timetable_slot_id
          AND tstr.is_break = true
    ) THEN
        RAISE EXCEPTION 'Cannot create attendance record for a break period (timetable_slot_id: %)', NEW.timetable_slot_id
            USING ERRCODE = 'P0001';
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;
```
Trigger function (wired as `trg_attendance_check_non_break_slot` on `attendance_records`) that prevents attendance records from referencing timetable slots whose structure has `is_break = true`.

### `max_points_check(session_id UUID, raw_score NUMERIC)`
```sql
CREATE OR REPLACE FUNCTION max_points_check(session_id UUID, raw_score NUMERIC)
RETURNS BOOLEAN
LANGUAGE sql
STABLE
AS $$
    SELECT raw_score <= COALESCE((SELECT max_points FROM assessment_sessions WHERE id = session_id), raw_score);
$$;
```
Helper used by the `chk_score_range` check constraint on `student_assessment_scores` to enforce `raw_score <= max_points` of the parent session.

### `fn_block_grading_scale_range_modification()`
BEFORE UPDATE OR DELETE trigger function on `grading_scale_ranges`. Raises error `P0002` when a range is modified or deleted while its profile is referenced by any `assessment_sessions` row (write-once semantics).

### `fn_block_assessment_max_points_update()`
BEFORE UPDATE trigger function on `assessment_sessions` (fired only when `max_points` changes, via `trg_assessment_sessions_max_points_immutable`). Raises error `P0002` when `max_points` is changed after any `student_assessment_scores` rows exist for the session.

### Invoice payment status sync — `fn_sync_invoice_payment_status_insert()`, `fn_sync_invoice_payment_status_delete()`, `fn_sync_invoice_payment_status_update()`
Three statement-level AFTER trigger functions on `payments` (INSERT, DELETE, and UPDATE respectively — split so each trigger only accesses the transition table available to it). Recompute `invoices.amount_paid` (sum of confirmed payments) and `invoices.payment_status`. The `WAIVED` status is never overwritten; otherwise status is `UNPAID` (0 paid), `PAID` (paid >= amount_due), or `PARTIAL`.

### School member-count sync — `fn_sync_school_staff_counts_insert()`, `fn_sync_school_staff_counts_delete()`, `fn_sync_school_staff_counts_update()`
Three statement-level AFTER trigger functions on `memberships` that upsert per-school admin/teacher/nurse/finance/parent counts into `school_member_counts` (counting only `is_active = true` memberships).

### School student-count sync — `fn_sync_school_student_counts_insert()`, `fn_sync_school_student_counts_delete()`, `fn_sync_school_student_counts_update()`
Three statement-level AFTER trigger functions on `cbc_students` that upsert per-school active student counts into `school_member_counts`.

### Assessment summary rollups (batch / on-demand)
- `fn_refresh_term_subject_summary_for_session(session_id UUID)` — recomputes `student_term_subject_summaries` for all students in a session, blending quantitative percentages with rubric outcome grades (mapped via `grading_scale_ranges.default_percentage_mapping`). Called automatically when a session is published.
- `fn_compute_term_overall_summaries_for_term(term_id UUID)` — computes `student_term_overall_summaries` for all enrolled students. Applies KNEC weighting formulas from `assessment_weight_configs` when the term is a final exam term (G6→KPSEA, G9→KJSEA, G12→KSSEA); otherwise a plain average across subjects.
- `fn_compute_single_student_term_overall_summary(student_id UUID, term_id UUID)` — convenience wrapper computing the overall summary for a single student+term.
- `fn_compute_cohort_positions_for_term(term_id UUID)` — batch-computes `student_cohort_position_summaries` (class/grade ranks, percentiles, averages) using window functions. NEVER updated incrementally.
- `fn_refresh_subject_strand_summary_for_session(session_id UUID)` — recomputes `student_subject_strand_summaries` for RUBRIC sessions only (no-op for QUANTITATIVE).
- `fn_compute_performance_projections_for_term(term_id UUID)` — batch-computes `student_performance_projections` (linear-regression momentum, projected score, risk level) from the last 2–3 terms.
- `fn_refresh_student_behavior_term_summary(student_id UUID, term_id UUID)` — idempotently recomputes the `student_behavior_term_summaries` row for a student+term.
- `fn_refresh_student_behavior_term_summary_for_note()` — trigger function that resolves the student+term from a `behavior_notes` insert/update and refreshes the behavior term summary.
- `fn_compute_teacher_subject_performance_summaries(term_id UUID)` — batch-computes `teacher_subject_performance_summaries` for all SUBJECT_TEACHER assignments.
- `fn_compute_teacher_delivery_summaries(term_id UUID)` — batch-computes `teacher_delivery_summaries` (marked/missed slots, session counts, on-time submission rate).
- `fn_compute_teacher_workload_summaries(year_id UUID)` — batch-computes `teacher_workload_summaries` (periods, subjects, classes, utilization, overcapacity flag).
- `fn_assessment_sessions_after_publish()` — AFTER UPDATE trigger function on `assessment_sessions` (status → PUBLISHED) that calls the subject-summary and strand-summary refresh functions.

### `member_counts` maintenance — `trg_update_student_count()` and `trg_update_membership_count()` (migration 000063_member_counts)
Trigger functions that keep the single-row global `member_counts` aggregate in sync:
- `trg_update_student_count()` — AFTER INSERT/UPDATE/DELETE on `cbc_students`, increments/decrements `students` when `is_active` toggles.
- `trg_update_membership_count()` — AFTER INSERT/UPDATE/DELETE on `memberships`, increments/decrements per-role counters (`admins`, `teachers`, `nurses`, `parents`, `finance`; `SYSTEM_ADMIN` is not counted), handling role changes, activation/deactivation, and deletes without double-counting.

## Enumerations

### User Roles
- `user_role`: SYSTEM_ADMIN, SCHOOL_ADMIN, TEACHER, NURSE, FINANCE, PARENT

### Invitation Status
- `invitation_status`: pending, accepted, expired, revoked, invite_failed

### Gender Type
- `gender_type`: M, F

### CBC Enrollment Status
- `cbc_enrollment_status`: ACTIVE, SUSPENDED, TRANSFERRED, COMPLETED_CYCLE

### CBC Grade Level
- `cbc_grade_level`: PP1, PP2, G1, G2, G3, G4, G5, G6, G7, G8, G9, G10, G11, G12

### CBC Education Level
- `cbc_education_level`: Early_Years, Upper_Primary, Junior_Secondary, Senior_School

### Teacher Role
- `teacher_role`: PRIMARY_CLASS_TEACHER, SUBJECT_TEACHER, SUBSTITUTE_TEACHER

### School Type
- `cbc_school_type`: Public, Private, Special_Needs_School

### Learning Pathway
- `cbc_learning_pathway`: Age_Based, Stage_Based

### Invoice Payment Status
- `invoice_payment_status`: UNPAID, PARTIAL, PAID, WAIVED

### Import Job Status
- `import_job_status`: pending, processing, completed, failed, cancelled, completed_with_errors, cancelling

### Import Job Type
- `import_job_type`: STAFF_INVITE, STUDENT_IMPORT, PARENT_INVITE

### Import Staging Status
- `import_staging_status`: pending, succeeded, failed

### Import Chunk Status
- `import_chunk_status`: pending, processing, completed, cancelled

### Block Type
- `block_type`: Lesson, Break, Assembly, ExtraCurricular

### Import Failure Type
- `import_failure_type`: SCHEMA_VALIDATION, DATABASE_CONSTRAINT, BUSINESS_RULE_VIOLATION, DUPLICATE_EMAIL, INVALID_EMAIL_FORMAT, STYTCH_API_ERROR, INVITATION_INSERT_FAILED

### Payment Method Type
- `payment_method_type`: MPESA, CASH, BANK_TRANSFER, CHEQUE, OTHER

### Parent Relationship Type
- `parent_relationship_type`: FATHER, MOTHER, GUARDIAN, OTHER

### Behavior Category Type
- `behavior_category_type`: COMMENDATION, DISCIPLINARY, OTHER

### Attendance Status
- `attendance_status`: PRESENT, ABSENT, LATE, EXCUSED

### Behavior Note Status
- `behavior_note_status`: PENDING_REVIEW, APPROVED, REJECTED, INCLUDED_IN_REPORT

### Behavior Severity
- `behavior_severity`: MINOR, NEEDS_FOLLOW_UP

### CBC Performance Level
- `cbc_performance_level`: EE (Exceeding Expectation), ME (Meeting Expectation), AE (Approaching Expectation), BE (Below Expectation)

### Assessment Session Status
- `assessment_session_status`: DRAFT, PENDING_APPROVAL, PUBLISHED

### Assessment Evaluation Method
- `assessment_evaluation_method`: QUANTITATIVE, RUBRIC

## Core Tables

### Tenants
- `id`: UUID (Primary Key) - Default: gen_random_uuid()
- `name`: VARCHAR(255) NOT NULL
- `slug`: VARCHAR(255) NOT NULL UNIQUE
- `stytch_org_id`: VARCHAR(255) NOT NULL UNIQUE
- `created_at`: TIMESTAMPTZ NOT NULL DEFAULT NOW()

**Comments:**
- No table or column comments defined

**Indexes:**
- `idx_tenants_slug` (slug)
- `idx_tenants_stytch_org_id` (stytch_org_id)

### Users
- `id`: UUID (Primary Key) - Default: gen_random_uuid()
- `email`: VARCHAR(255) NOT NULL
- `tenant_id`: UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE
- `full_name`: VARCHAR(255) NOT NULL DEFAULT ''
- `is_active`: BOOLEAN NOT NULL DEFAULT TRUE
- `external_auth_id`: VARCHAR(255) UNIQUE
- `tsc_number`: VARCHAR(15) NULL
- `knec_panel_assessor_id`: VARCHAR(20) NULL
- `created_at`: TIMESTAMPTZ NOT NULL DEFAULT NOW()
- `updated_at`: TIMESTAMPTZ NOT NULL DEFAULT NOW()

**Comments:**
- COLUMN users.tsc_number: 'Teachers Service Commission registration number. Populated only for users with the TEACHER role. Required for TSC portal access and official deployment.'
- COLUMN users.knec_panel_assessor_id: 'Assigned ONLY to teachers formally appointed to KNEC national exam panels (KPSEA, KJSEA, KSSEA invigilation or marking). NOT required for classroom SBA delivery — all SBA uploads use the school knec_school_code, not teacher IDs.'

**Indexes:**
- `idx_users_tenant_email` UNIQUE (tenant_id, LOWER(email)) - 'Per-tenant, case-insensitive unique constraint on email. Replaces the old global idx_users_email which prevented multi-tenant accounts and treated case variants as distinct. Added in 000003_fix.'
- `idx_users_tenant` (tenant_id)
- `idx_users_tsc_number` UNIQUE (tsc_number) WHERE tsc_number IS NOT NULL
- `idx_users_knec_panel_assessor_id` UNIQUE (knec_panel_assessor_id) WHERE knec_panel_assessor_id IS NOT NULL

**Triggers:**
- `trg_users_updated_at` BEFORE UPDATE ON users FOR EACH ROW EXECUTE FUNCTION fn_set_updated_at()

### Sessions
- `id`: UUID (Primary Key) - Default: gen_random_uuid()
- `token`: VARCHAR(128) NULL
- `token_hash`: TEXT NULL
- `user_id`: UUID NOT NULL
- `tenant_id`: UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE
- `stytch_member_id`: VARCHAR(255) NOT NULL
- `stytch_org_id`: VARCHAR(255) NOT NULL
- `stytch_session_token`: VARCHAR(512) NOT NULL DEFAULT ''
- `device_fingerprint`: VARCHAR(128) NOT NULL DEFAULT ''
- `expires_at`: TIMESTAMPTZ NOT NULL
- `created_at`: TIMESTAMPTZ NOT NULL DEFAULT NOW()

**Comments:**
- COLUMN sessions.token: 'DEPRECATED and will be removed in a future migration. This column is now nullable. All lookups should use token_hash instead. New sessions will insert NULL here.'
- COLUMN sessions.token_hash: 'SHA-256 hash of the session token (hex-encoded). Use this for token lookups instead of the raw token column.'
- COLUMN sessions.stytch_session_token: 'TODO: stytch_session_token is a third-party session token from Stytch, not one this schema issues. Hashing strategy for Stytch tokens requires app-team sign-off — do not implement hashing for this column without a reviewed design doc.'

**Indexes:**
- `idx_sessions_token` (token)
- `idx_sessions_user_id` (user_id)
- `idx_sessions_tenant_id` (tenant_id)
- `idx_sessions_stytch_session_token` (stytch_session_token)
- `idx_sessions_token_hash` UNIQUE (token_hash)

**Triggers:**
- No specific updated_at trigger (uses token_hash instead of token for lookups)

### CBC Schools
- `id`: UUID (Primary Key) - Default: gen_random_uuid()
- `tenant_id`: UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE
- `name`: VARCHAR(255) NOT NULL
- `knec_school_code`: VARCHAR(15) NULL
- `nemis_institution_code`: VARCHAR(20) NULL
- `county`: VARCHAR(50) NOT NULL
- `sub_county`: VARCHAR(50) NOT NULL
- `ward`: VARCHAR(50) NULL
- `school_type`: cbc_school_type NOT NULL
- `is_active`: BOOLEAN NOT NULL DEFAULT true
- `created_at`: TIMESTAMPTZ NOT NULL DEFAULT NOW()
- `updated_at`: TIMESTAMPTZ NOT NULL DEFAULT NOW()

**Comments:**
- COLUMN cbc_schools.knec_school_code: 'Official KNEC center code (8–10 digit numeric string). Used as the school login username on the CBA portal at cba.knec.ac.ke. Required before any SBA score uploads can be submitted to KNEC.'
- COLUMN cbc_schools.nemis_institution_code: 'National Education Management System institution code. Assigned by the Ministry of Education. Used for MoE reporting and NEMIS data synchronisation.'
- TABLE cbc_streams: 'Named streams within a school (e.g. "Blue", "Red", "Green"). A stream is the second axis of class identity alongside grade_level. Streams themselves cannot be deleted while any cbc_classes row references them (ON DELETE RESTRICT via fk_cbc_classes_stream). Streams with no class references may be deleted via the API.'
- CONSTRAINT fk_cbc_streams_school ON cbc_streams: 'Composite FK (tenant_id, school_id) enforces tenant-scoped referential integrity. ON DELETE CASCADE — streams are removed when their school is deleted.'

**Indexes:**
- `idx_cbc_schools_knec_code` UNIQUE (knec_school_code) WHERE knec_school_code IS NOT NULL
- `idx_cbc_schools_nemis_code` UNIQUE (nemis_institution_code) WHERE nemis_institution_code IS NOT NULL
- `idx_cbc_schools_tenant_id` (tenant_id)

**Triggers:**
- `trg_cbc_schools_updated_at` BEFORE UPDATE ON cbc_schools FOR EACH ROW EXECUTE FUNCTION fn_set_updated_at()

### Academic Years
- `id`: UUID (Primary Key) - Default: gen_random_uuid()
- `tenant_id`: UUID NOT NULL
- `school_id`: UUID NOT NULL
- `name`: VARCHAR(50) NOT NULL
- `start_date`: DATE NOT NULL
- `end_date`: DATE NOT NULL
- `is_current`: BOOLEAN NOT NULL DEFAULT false
- `version`: INTEGER NOT NULL DEFAULT 1
- `created_at`: TIMESTAMPTZ NOT NULL DEFAULT NOW()
- `updated_at`: TIMESTAMPTZ NOT NULL DEFAULT NOW()
- `created_by`: UUID NOT NULL
- `updated_by`: UUID NOT NULL

**Constraints:**
- `chk_year_dates` CHECK (start_date < end_date)
- `uq_academic_years_tenant` UNIQUE (tenant_id, id)
- `fk_academic_years_tenant_created_by` FOREIGN KEY (tenant_id, created_by) REFERENCES users(tenant_id, id)
- `fk_academic_years_tenant_updated_by` FOREIGN KEY (tenant_id, updated_by) REFERENCES users(tenant_id, id)
- `uq_academic_years_tenant_school_id` UNIQUE (tenant_id, school_id, id)
- `fk_academic_years_tenant_school` FOREIGN KEY (tenant_id, school_id) REFERENCES cbc_schools(tenant_id, id) ON DELETE CASCADE
- `EXCL_academic_years_no_overlap` EXCLUDE USING gist (
    school_id WITH =,
    daterange(start_date, end_date, '[]') WITH &&
)

**Comments:**
- No specific column comments beyond what's implied by constraints

**Indexes:**
- `idx_academic_years_tenant_id` (tenant_id)
- `idx_academic_years_school_id` (school_id)
- `idx_one_current_year_per_school` UNIQUE (school_id) WHERE is_current = TRUE

**Triggers:**
- `trg_academic_years_updated_at` BEFORE UPDATE ON academic_years FOR EACH ROW EXECUTE FUNCTION fn_set_updated_at()

### Academic Terms
- `id`: UUID (Primary Key) - Default: gen_random_uuid()
- `tenant_id`: UUID NOT NULL
- `school_id`: UUID NOT NULL
- `academic_year_id`: UUID NOT NULL
- `name`: VARCHAR(100) NOT NULL
- `term_number`: SMALLINT NOT NULL
- `start_date`: DATE NOT NULL
- `end_date`: DATE NOT NULL
- `is_current`: BOOLEAN NOT NULL DEFAULT false
- `is_final`: BOOLEAN NOT NULL DEFAULT false
- `version`: INTEGER NOT NULL DEFAULT 1
- `created_at`: TIMESTAMPTZ NOT NULL DEFAULT NOW()
- `updated_at`: TIMESTAMPTZ NOT NULL DEFAULT NOW()
- `created_by`: UUID NOT NULL
- `updated_by`: UUID NOT NULL

**Constraints:**
- `chk_term_dates` CHECK (start_date < end_date)
- `chk_term_number` CHECK (term_number BETWEEN 1 AND 3)
- `uq_academic_terms_tenant` UNIQUE (tenant_id, id)
- `fk_academic_terms_tenant_created_by` FOREIGN KEY (tenant_id, created_by) REFERENCES users(tenant_id, id)
- `fk_academic_terms_tenant_updated_by` FOREIGN KEY (tenant_id, updated_by) REFERENCES users(tenant_id, id)
- `uq_academic_terms_tenant_school` UNIQUE (tenant_id, school_id, id)
- `fk_academic_terms_tenant_school` FOREIGN KEY (tenant_id, school_id) REFERENCES cbc_schools(tenant_id, id) ON DELETE CASCADE
- `fk_academic_terms_tenant_year` FOREIGN KEY (tenant_id, school_id, academic_year_id) REFERENCES academic_years(tenant_id, school_id, id) ON DELETE CASCADE
- `EXCL_academic_terms_no_overlap` EXCLUDE USING gist (
    school_id WITH =,
    daterange(start_date, end_date, '[]') WITH &&
)

**Comments:**
- COLUMN academic_terms.term_number: 'Kenya CBC operates a 3-term academic year. term_number enforces this: 1 = Term 1, 2 = Term 2, 3 = Term 3.'
- COLUMN academic_terms.is_final: 'Marks the last term of the academic year before a national KNEC exam cycle (KPSEA at end of G6, KJSEA at end of G9, KSSEA at end of G12). The application uses this flag to lock SBA submissions and trigger KNEC sync workflows. Set to TRUE only on Term 3 of an exam year.'

**Indexes:**
- `idx_academic_terms_tenant_id` (tenant_id)
- `idx_academic_terms_school_id` (school_id)
- `idx_academic_terms_year_id` (academic_year_id)
- `idx_one_current_term_per_school` UNIQUE (school_id) WHERE is_current = TRUE
- `idx_unique_term_number_per_year` UNIQUE (academic_year_id, term_number)

**Triggers:**
- `trg_academic_terms_updated_at` BEFORE UPDATE ON academic_terms FOR EACH ROW EXECUTE FUNCTION fn_set_updated_at()
- `trg_validate_term_dates` BEFORE INSERT OR UPDATE ON academic_terms FOR EACH ROW EXECUTE FUNCTION fn_validate_term_dates_within_year()

### CBC Streams
- `id`: UUID (Primary Key) - Default: gen_random_uuid()
- `tenant_id`: UUID NOT NULL
- `school_id`: UUID NOT NULL
- `name`: VARCHAR(100) NOT NULL
- `color`: VARCHAR(50) NOT NULL DEFAULT ''
- `created_at`: TIMESTAMPTZ NOT NULL DEFAULT NOW()
- `updated_at`: TIMESTAMPTZ NOT NULL DEFAULT NOW()

**Constraints:**
- `fk_cbc_streams_school` FOREIGN KEY (tenant_id, school_id) REFERENCES cbc_schools(tenant_id, id) ON DELETE CASCADE
- `uq_cbc_streams_tenant_school_name` UNIQUE (tenant_id, school_id, name)
- `uq_cbc_streams_tenant` UNIQUE (tenant_id, id)

**Comments:**
- TABLE cbc_streams: 'Named streams within a school (e.g. "Blue", "Red", "Green"). A stream is the second axis of class identity alongside grade_level. Streams themselves cannot be deleted while any cbc_classes row references them (ON DELETE RESTRICT via fk_cbc_classes_stream). Streams with no class references may be deleted via the API.'
- CONSTRAINT fk_cbc_streams_school ON cbc_streams: 'Composite FK (tenant_id, school_id) enforces tenant-scoped referential integrity. ON DELETE CASCADE — streams are removed when their school is deleted.'

**Indexes:**
- `idx_cbc_streams_school_id` (school_id)
- `idx_cbc_streams_tenant_id` (tenant_id)

**Triggers:**
- `trg_cbc_streams_updated_at` BEFORE UPDATE ON cbc_streams FOR EACH ROW EXECUTE FUNCTION fn_set_updated_at()

### CBC Classes
- `id`: UUID (Primary Key) - Default: gen_random_uuid()
- `tenant_id`: UUID NOT NULL
- `school_id`: UUID NOT NULL
- `academic_year_id`: UUID NOT NULL
- `grade_level`: cbc_grade_level NOT NULL
- `stream_id`: UUID NOT NULL
- `is_active`: BOOLEAN NOT NULL DEFAULT true
- `created_at`: TIMESTAMPTZ NOT NULL DEFAULT NOW()
- `updated_at`: TIMESTAMPTZ NOT NULL DEFAULT NOW()

**Constraints:**
- `uq_cbc_classes_tier_stream` UNIQUE (school_id, academic_year_id, grade_level, stream_id)
- `fk_cbc_classes_tenant_school` FOREIGN KEY (tenant_id, school_id) REFERENCES cbc_schools(tenant_id, id) ON DELETE CASCADE
- `fk_cbc_classes_tenant_academic_year` FOREIGN KEY (tenant_id, academic_year_id) REFERENCES academic_years(tenant_id, id) ON DELETE CASCADE
- `fk_cbc_classes_stream` FOREIGN KEY (tenant_id, stream_id) REFERENCES cbc_streams(tenant_id, id) ON DELETE RESTRICT
- `uq_cbc_classes_tenant` UNIQUE (tenant_id, id)

**Comments:**
- COLUMN cbc_classes.grade_level: 'Official KNEC grade designation. Determines which assessment instruments, SBA projects, and KNEC portal upload windows apply to the class. Values match KNEC CBA portal grade codes: PP1–PP2 (Pre-Primary), G1–G12.'

**Indexes:**
- `idx_cbc_classes_tenant_id` (tenant_id)
- `idx_cbc_classes_school_id` (school_id)
- `idx_cbc_classes_academic_year_id` (academic_year_id)
- `idx_cbc_classes_grade_level` (grade_level)
- `idx_cbc_classes_school_year_grade_stream` (school_id, academic_year_id, grade_level, stream_id)

**Triggers:**
- `trg_cbc_classes_updated_at` BEFORE UPDATE ON cbc_classes FOR EACH ROW EXECUTE FUNCTION fn_set_updated_at()

### Memberships
- `id`: UUID (Primary Key) - Default: gen_random_uuid()
- `tenant_id`: UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE
- `user_id`: UUID NOT NULL
- `school_id`: UUID NOT NULL
- `role`: user_role NOT NULL
- `is_active`: BOOLEAN NOT NULL DEFAULT true
- `created_at`: TIMESTAMPTZ NOT NULL DEFAULT NOW()
- `updated_at`: TIMESTAMPTZ NOT NULL DEFAULT NOW()

**Constraints:**
- `fk_memberships_tenant_school` FOREIGN KEY (tenant_id, school_id) REFERENCES cbc_schools(tenant_id, id) ON DELETE CASCADE
- `fk_memberships_tenant_user` FOREIGN KEY (tenant_id, user_id) REFERENCES users(tenant_id, id) ON DELETE CASCADE
- `unique_user_school_membership` UNIQUE (user_id, school_id)

**Indexes:**
- `idx_memberships_tenant_id` (tenant_id)
- `idx_memberships_user_id` (user_id)
- `idx_memberships_school_id` (school_id)

**Triggers:**
- `trg_memberships_updated_at` BEFORE UPDATE ON memberships FOR EACH ROW EXECUTE FUNCTION fn_set_updated_at()

### Import Jobs
- `id`: UUID (Primary Key) - Default: gen_random_uuid()
- `tenant_id`: UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE
- `school_id`: UUID NOT NULL
- `job_type`: import_job_type NOT NULL
- `role`: user_role NULL
- `created_by`: UUID NULL
- `status`: import_job_status NOT NULL DEFAULT 'pending'
- `total_records`: INT NOT NULL DEFAULT 0
- `processed_records`: INT NOT NULL DEFAULT 0
- `success_count`: INT NOT NULL DEFAULT 0
- `failed_count`: INT NOT NULL DEFAULT 0
- `idempotency_key`: TEXT NULL
- `payload_hash`: TEXT NULL
- `total_chunks`: INT NOT NULL DEFAULT 0
- `processed_chunks`: INT NOT NULL DEFAULT 0
- `metadata`: JSONB NOT NULL DEFAULT '{}'::jsonb
- `created_at`: TIMESTAMPTZ NOT NULL DEFAULT NOW()
- `started_at`: TIMESTAMPTZ NULL
- `completed_at`: TIMESTAMPTZ NULL
- `last_progress_at`: TIMESTAMPTZ NULL
- `updated_at`: TIMESTAMPTZ NOT NULL DEFAULT NOW()

**Constraints:**
- `fk_import_jobs_tenant_school` FOREIGN KEY (tenant_id, school_id) REFERENCES cbc_schools(tenant_id, id) ON DELETE CASCADE
- `fk_import_jobs_tenant_created_by` FOREIGN KEY (tenant_id, created_by) REFERENCES users(tenant_id, id) ON DELETE SET NULL (created_by)
- `chk_import_jobs_role_required_for_staff` CHECK (
    (job_type IN ('STAFF_INVITE', 'PARENT_INVITE') AND role IS NOT NULL)
    OR (job_type NOT IN ('STAFF_INVITE', 'PARENT_INVITE'))
  )

**Indexes:**
- `idx_import_jobs_tenant_id` (tenant_id)
- `idx_import_jobs_school_id` (school_id)
- `idx_import_jobs_created_by` (created_by)
- `idx_import_jobs_status` (status)
- `uq_import_jobs_tenant_idempotency` UNIQUE (tenant_id, idempotency_key) WHERE idempotency_key IS NOT NULL
- `uq_import_jobs_tenant` UNIQUE (tenant_id, id)
- `uq_import_jobs_one_active_per_school` UNIQUE (school_id) WHERE status IN ('processing'::import_job_status, 'cancelling'::import_job_status)

**Triggers:**
- No specific updated_at trigger mentioned in SQL (has updated_at column with DEFAULT NOW())

### Import Job Chunks
- `id`: UUID (Primary Key) - Default: gen_random_uuid()
- `job_id`: UUID NOT NULL REFERENCES import_jobs(id) ON DELETE CASCADE
- `chunk_index`: INT NOT NULL
- `status`: import_chunk_status NOT NULL DEFAULT 'pending'
- `row_start`: INT NOT NULL DEFAULT 0
- `row_end`: INT NOT NULL DEFAULT 0
- `claimed_at`: TIMESTAMPTZ NULL
- `completed_at`: TIMESTAMPTZ NULL
- `created_at`: TIMESTAMPTZ NOT NULL DEFAULT NOW()
- `updated_at`: TIMESTAMPTZ NOT NULL DEFAULT NOW()

**Constraints:**
- `uq_import_job_chunks_job_chunk` UNIQUE (job_id, chunk_index)

**Indexes:**
- `idx_import_job_chunks_job_id` (job_id)
- `idx_import_job_chunks_status` (job_id, status)

### Import Job Failures
- `id`: BIGSERIAL (Primary Key)
- `import_job_id`: UUID NOT NULL REFERENCES import_jobs(id) ON DELETE CASCADE
- `raw_payload`: JSONB NOT NULL
- `error_message`: TEXT NOT NULL
- `error_type`: import_failure_type NOT NULL DEFAULT 'DATABASE_CONSTRAINT'
- `row_number`: INT NULL
- `created_at`: TIMESTAMPTZ NOT NULL DEFAULT NOW()
- `updated_at`: TIMESTAMPTZ NOT NULL DEFAULT NOW()

**Indexes:**
- `idx_import_job_failures_job_id` (import_job_id)

### Import Job Staging
- `id`: UUID (Primary Key) - Default: gen_random_uuid()
- `job_id`: UUID NOT NULL REFERENCES import_jobs(id) ON DELETE CASCADE
- `tenant_id`: UUID NOT NULL
- `school_id`: UUID NOT NULL
- `row_number`: INT NOT NULL
- `raw_data`: JSONB NOT NULL
- `status`: import_staging_status NOT NULL DEFAULT 'pending'
- `processed_at`: TIMESTAMPTZ NULL
- `created_at`: TIMESTAMPTZ NOT NULL DEFAULT now()
- `updated_at`: TIMESTAMPTZ NOT NULL DEFAULT NOW()

**Constraints:**
- `uq_import_job_staging_job_id` (job_id)
- `uq_import_job_staging_job_row` UNIQUE (job_id, row_number)
- `uq_import_job_staging_tenant` UNIQUE (tenant_id, id)

### Invitations
- `id`: UUID (Primary Key) - Default: gen_random_uuid()
- `tenant_id`: UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE
- `school_id`: UUID NOT NULL
- `email`: VARCHAR(255) NOT NULL
- `role`: user_role NOT NULL
- `status`: invitation_status NOT NULL DEFAULT 'pending'
- `invited_by`: UUID NULL
- `token`: TEXT NOT NULL
- `token_hash`: TEXT NULL
- `expires_at`: TIMESTAMPTZ NOT NULL
- `accepted_at`: TIMESTAMPTZ NULL
- `full_name`: VARCHAR(255) NOT NULL
- `phone`: VARCHAR(50) NULL
- `registration_number`: VARCHAR(100) NULL
- `stytch_member_id`: VARCHAR(255) NULL
- `import_job_id`: UUID NULL
- `error_message`: TEXT NULL
- `attempt_count`: INT NOT NULL DEFAULT 0
- `created_at`: TIMESTAMPTZ NOT NULL DEFAULT NOW()
- `updated_at`: TIMESTAMPTZ NOT NULL DEFAULT NOW()

**Constraints:**
- `fk_invitations_tenant_school` FOREIGN KEY (tenant_id, school_id) REFERENCES cbc_schools(tenant_id, id) ON DELETE CASCADE
- `fk_invitations_tenant_invited_by` FOREIGN KEY (tenant_id, invited_by) REFERENCES users(tenant_id, id) ON DELETE SET NULL (invited_by)
- `fk_invitations_tenant_import_job` FOREIGN KEY (tenant_id, import_job_id) REFERENCES import_jobs(tenant_id, id) ON DELETE SET NULL (import_job_id)

**Comments:**
- COLUMN invitations.token: 'DEPRECATED — raw invitation token. New code should read token_hash instead. This column will be dropped in a future migration after the app is confirmed fully migrated to hash-based lookups. Do NOT write to this column in new code.'
- COLUMN invitations.token_hash: 'SHA-256 hash of the invitation token (hex-encoded). Backfilled from token column. Use this for token lookups instead of the raw token column.'

**Indexes:**
- `idx_invitations_tenant_id` (tenant_id)
- `idx_invitations_school_id` (school_id)
- `idx_invitations_email` (email)
- `idx_invitations_status` (status)
- `idx_invitations_import_job` (import_job_id)
- `uq_invitations_active_email` UNIQUE (tenant_id, school_id, email) WHERE status NOT IN ('expired', 'revoked')
- `uq_invitations_school_email_pending` UNIQUE (school_id, email) WHERE status = 'pending'
- `idx_invitations_token_hash` UNIQUE (token_hash)

### CBC Parents
- `id`: UUID (Primary Key) - Default: gen_random_uuid()
- `tenant_id`: UUID NOT NULL
- `user_id`: UUID NOT NULL
- `phone_number`: VARCHAR(20) NOT NULL
- `is_active`: BOOLEAN NOT NULL DEFAULT true
- `created_at`: TIMESTAMPTZ NOT NULL DEFAULT NOW()
- `updated_at`: TIMESTAMPTZ NOT NULL DEFAULT NOW()

**Constraints:**
- `uq_cbc_parents_user` UNIQUE (user_id)
- `uq_cbc_parents_tenant` UNIQUE (tenant_id, id)
- `fk_cbc_parents_tenant_user` FOREIGN KEY (tenant_id, user_id) REFERENCES users(tenant_id, id) ON DELETE CASCADE

**Indexes:**
- `idx_cbc_parents_phone` (phone_number)
- `idx_cbc_parents_tenant` (tenant_id)

**Triggers:**
- `trg_cbc_parents_updated_at` BEFORE UPDATE ON cbc_parents FOR EACH ROW EXECUTE FUNCTION fn_set_updated_at()

### CBC Students
- `id`: UUID (Primary Key) - Default: gen_random_uuid()
- `tenant_id`: UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE
- `school_id`: UUID NOT NULL
- `full_name`: VARCHAR(255) NOT NULL
- `gender`: gender_type NOT NULL
- `date_of_birth`: DATE NULL
- `upi_number`: VARCHAR(20) NULL
- `knec_assessment_number`: VARCHAR(15) NULL
- `admission_number`: VARCHAR(20) NULL
- `learning_pathway`: cbc_learning_pathway NOT NULL DEFAULT 'Age_Based'
- `staging_row_id`: UUID NULL
- `is_active`: BOOLEAN NOT NULL DEFAULT true
- `created_at`: TIMESTAMPTZ NOT NULL DEFAULT NOW()
- `updated_at`: TIMESTAMPTZ NOT NULL DEFAULT NOW()

**Constraints:**
- `uq_cbc_students_tenant` UNIQUE (tenant_id, id)
- `fk_cbc_students_tenant_school` FOREIGN KEY (tenant_id, school_id) REFERENCES cbc_schools(tenant_id, id) ON DELETE CASCADE
- `fk_cbc_students_tenant_staging_row` FOREIGN KEY (staging_row_id) REFERENCES import_job_staging(id) ON DELETE SET NULL
- `chk_cbc_student_gender` CHECK (gender IN ('M', 'F'))

**Comments:**
- COLUMN cbc_students.gender: 'CBC/NEMIS-compliant gender field. M=Male, F=Female only. KNEC registration and NEMIS records do not support other values.'
- COLUMN cbc_students.upi_number: 'Unique Personal Identifier assigned by NEMIS at school enrollment. Used in all Ministry of Education reporting and NEMIS data submissions.'
- COLUMN cbc_students.knec_assessment_number: 'Permanent CBC identifier assigned by KNEC from Grade 3 onward. Required for KPSEA/KJSEA/KSSEA exam registration. Parents use this number to access learner results at cba.knec.ac.ke/Parent.'
- COLUMN cbc_students.learning_pathway: 'Determines which KNEC assessment framework governs the learner. Age_Based: standard mainstream CBC curriculum (vast majority). Stage_Based: SNE pathway for learners with severe cognitive or multiple disabilities, governed by the CBAF-FL framework.'
- COLUMN cbc_students.school_id: 'Home school for this student. Set at first enrollment and updated on transfer. Use cbc_student_enrollments for full term-by-term history.'

**Indexes:**
- `idx_cbc_students_upi` UNIQUE (upi_number) WHERE upi_number IS NOT NULL
- `idx_cbc_students_knec_assessment_number` UNIQUE (knec_assessment_number) WHERE knec_assessment_number IS NOT NULL
- `idx_cbc_students_school_staging_row` UNIQUE (school_id, staging_row_id) WHERE staging_row_id IS NOT NULL
- `idx_cbc_students_tenant_id` (tenant_id)
- `idx_cbc_students_school_id` (school_id)

**Triggers:**
- `trg_cbc_students_updated_at` BEFORE UPDATE ON cbc_students FOR EACH ROW EXECUTE FUNCTION fn_set_updated_at()

### CBC Student Parents Junction
- `tenant_id`: UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE
- `student_id`: UUID NOT NULL
- `parent_id`: UUID NOT NULL
- `relationship`: parent_relationship_type NULL
- `is_primary`: BOOLEAN NOT NULL DEFAULT true
- `created_at`: TIMESTAMPTZ NOT NULL DEFAULT NOW()
- `updated_at`: TIMESTAMPTZ NOT NULL DEFAULT NOW()

**Constraints:**
- `pk_cbc_student_parents` PRIMARY KEY (tenant_id, student_id, parent_id)
- `fk_cbc_student_parents_tenant_student` FOREIGN KEY (tenant_id, student_id) REFERENCES cbc_students(tenant_id, id) ON DELETE CASCADE
- `fk_cbc_student_parents_tenant_parent` FOREIGN KEY (tenant_id, parent_id) REFERENCES cbc_parents(tenant_id, id) ON DELETE CASCADE

**Comments:**
- COLUMN cbc_student_parents.relationship: 'Parent/guardian relationship to the student. Enum migrated from free-text in 000003_fix. Values: FATHER, MOTHER, GUARDIAN, OTHER.'

**Indexes:**
- `idx_junction_parent` (parent_id)
- `idx_junction_tenant_student` (tenant_id, student_id)
- `idx_one_primary_parent_per_student` UNIQUE (tenant_id, student_id) WHERE is_primary = true

### CBC Student Enrollments
- `id`: UUID (Primary Key) - Default: gen_random_uuid()
- `tenant_id`: UUID NOT NULL
- `student_id`: UUID NOT NULL
- `school_id`: UUID NOT NULL
- `academic_term_id`: UUID NOT NULL
- `academic_year_id`: UUID NOT NULL
- `class_id`: UUID NULL
- `status`: cbc_enrollment_status NOT NULL DEFAULT 'ACTIVE'
- `created_at`: TIMESTAMPTZ NOT NULL DEFAULT NOW()
- `updated_at`: TIMESTAMPTZ NOT NULL DEFAULT NOW()

**Constraints:**
- `fk_enrollments_tenant_student` FOREIGN KEY (tenant_id, student_id) REFERENCES cbc_students(tenant_id, id) ON DELETE CASCADE
- `fk_enrollments_tenant_school` FOREIGN KEY (tenant_id, school_id) REFERENCES cbc_schools(tenant_id, id) ON DELETE CASCADE
- `fk_enrollments_tenant_school_term` FOREIGN KEY (tenant_id, school_id, academic_term_id) REFERENCES academic_terms(tenant_id, school_id, id) ON DELETE CASCADE
- `fk_enrollments_tenant_class` FOREIGN KEY (tenant_id, class_id) REFERENCES cbc_classes(tenant_id, id) ON DELETE SET NULL (class_id)
- `unique_student_term_enrollment` UNIQUE (student_id, school_id, academic_term_id)
- `fk_enrollments_tenant_academic_year` FOREIGN KEY (tenant_id, academic_year_id) REFERENCES academic_years(tenant_id, id) ON DELETE CASCADE

**Indexes:**
- `idx_cbc_enrollments_tenant_id` (tenant_id)
- `idx_cbc_enrollments_student_id` (student_id)
- `idx_cbc_enrollments_school_id` (school_id)
- `idx_cbc_enrollments_term_id` (academic_term_id)
- `idx_cbc_enrollments_class_id` (class_id)
- `idx_cbc_enrollments_academic_year_id` (academic_year_id)

**Triggers:**
- `trg_cbc_student_enrollments_updated_at` BEFORE UPDATE ON cbc_student_enrollments FOR EACH ROW EXECUTE FUNCTION fn_set_updated_at()

### Medical Incidents
- `id`: UUID (Primary Key) - Default: gen_random_uuid()
- `tenant_id`: UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE
- `student_id`: UUID NOT NULL
- `incident_timestamp`: TIMESTAMPTZ NOT NULL
- `symptoms`: TEXT NOT NULL
- `action_taken`: TEXT NOT NULL
- `logged_by`: UUID NOT NULL
- `created_at`: TIMESTAMPTZ NOT NULL DEFAULT NOW()
- `updated_at`: TIMESTAMPTZ NOT NULL DEFAULT NOW()

**Constraints:**
- `fk_medical_incidents_tenant_student` FOREIGN KEY (tenant_id, student_id) REFERENCES cbc_students(tenant_id, id) ON DELETE CASCADE
- `fk_medical_incidents_tenant_logged_by` FOREIGN KEY (tenant_id, logged_by) REFERENCES users(tenant_id, id) ON DELETE RESTRICT

**Indexes:**
- `idx_medical_incidents_tenant_id` (tenant_id)
- `idx_medical_incidents_student_id` (student_id)

### Student Health Profiles
- `id`: UUID (Primary Key) - Default: gen_random_uuid()
- `tenant_id`: UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE
- `student_id`: UUID UNIQUE NOT NULL
- `blood_group`: VARCHAR(5)
- `allergies`: TEXT[]
- `chronic_conditions`: TEXT[]
- `emergency_instructions`: TEXT
- `logged_by`: UUID NOT NULL
- `created_at`: TIMESTAMPTZ NOT NULL DEFAULT NOW()
- `updated_at`: TIMESTAMPTZ NOT NULL DEFAULT NOW()

**Constraints:**
- `fk_student_health_profiles_tenant_student` FOREIGN KEY (tenant_id, student_id) REFERENCES cbc_students(tenant_id, id) ON DELETE CASCADE
- `fk_student_health_profiles_tenant_logged_by` FOREIGN KEY (tenant_id, logged_by) REFERENCES users(tenant_id, id) ON DELETE RESTRICT

**Indexes:**
- `idx_student_health_profiles_tenant_id` (tenant_id)

### Fee Categories
- `id`: UUID (Primary Key) - Default: gen_random_uuid()
- `tenant_id`: UUID NOT NULL
- `school_id`: UUID NOT NULL
- `name`: VARCHAR(150) NOT NULL
- `is_mandatory`: BOOLEAN NOT NULL DEFAULT true
- `updated_at`: TIMESTAMPTZ NOT NULL DEFAULT NOW()

**Constraints:**
- `fk_fee_categories_tenant_school` FOREIGN KEY (tenant_id, school_id) REFERENCES cbc_schools(tenant_id, id) ON DELETE CASCADE
- `uq_fee_categories_tenant_school_name` UNIQUE (tenant_id, school_id, name)
- `uq_fee_categories_tenant` UNIQUE (tenant_id, id)

**Indexes:**
- `idx_fee_categories_tenant` (tenant_id)
- `idx_fee_categories_school_id` (school_id)

### Fee Templates
- `id`: UUID (Primary Key) - Default: gen_random_uuid()
- `tenant_id`: UUID NOT NULL
- `school_id`: UUID NOT NULL
- `academic_term_id`: UUID NOT NULL
- `grade_level`: cbc_grade_level NOT NULL
- `fee_category_id`: UUID NOT NULL
- `amount`: NUMERIC(12,2) NOT NULL CHECK (amount >= 0)
- `updated_at`: TIMESTAMPTZ NOT NULL DEFAULT NOW()

**Constraints:**
- `fk_fee_templates_tenant_school` FOREIGN KEY (tenant_id, school_id) REFERENCES cbc_schools(tenant_id, id) ON DELETE CASCADE
- `fk_fee_templates_tenant_term` FOREIGN KEY (tenant_id, school_id, academic_term_id) REFERENCES academic_terms(tenant_id, school_id, id) ON DELETE CASCADE
- `fk_fee_templates_tenant_fee_category` FOREIGN KEY (tenant_id, fee_category_id) REFERENCES fee_categories(tenant_id, id) ON DELETE CASCADE
- `unique_fee_template_rule` UNIQUE (academic_term_id, grade_level, fee_category_id)

**Indexes:**
- `idx_fee_templates_tenant` (tenant_id)
- `idx_fee_templates_school_term` (school_id, academic_term_id)
- `idx_fee_templates_grade_level` (grade_level)

### Invoices
- `id`: UUID (Primary Key) - Default: gen_random_uuid()
- `tenant_id`: UUID NOT NULL
- `student_id`: UUID NOT NULL
- `school_id`: UUID NOT NULL
- `academic_term_id`: UUID NOT NULL
- `parent_id`: UUID NULL
- `invoice_label`: VARCHAR(255) NULL
- `payment_status`: invoice_payment_status NOT NULL DEFAULT 'UNPAID'
- `amount_due`: NUMERIC(12,2) NOT NULL DEFAULT 0 CHECK (amount_due >= 0)
- `amount_paid`: NUMERIC(12,2) NOT NULL DEFAULT 0 CHECK (amount_paid >= 0)
- `created_at`: TIMESTAMPTZ NOT NULL DEFAULT NOW()
- `updated_at`: TIMESTAMPTZ NOT NULL DEFAULT NOW()

**Constraints:**
- `uq_invoices_tenant` UNIQUE (tenant_id, id)
- `fk_invoices_tenant_student` FOREIGN KEY (tenant_id, student_id) REFERENCES cbc_students(tenant_id, id) ON DELETE CASCADE
- `fk_invoices_tenant_school` FOREIGN KEY (tenant_id, school_id) REFERENCES cbc_schools(tenant_id, id) ON DELETE CASCADE
- `fk_invoices_tenant_term` FOREIGN KEY (tenant_id, school_id, academic_term_id) REFERENCES academic_terms(tenant_id, school_id, id) ON DELETE CASCADE
- `fk_invoices_tenant_parent` FOREIGN KEY (tenant_id, parent_id) REFERENCES cbc_parents(tenant_id, id) ON DELETE SET NULL (parent_id)
- `unique_invoice_per_student_term` UNIQUE (student_id, academic_term_id)

**Comments:**
- COLUMN invoices.payment_status: 'Denormalised for fast lookups. Kept in sync by trg_sync_invoice_payment_status trigger on payments. WAIVED is set only by application logic — the trigger never overwrites a WAIVED status.'
- COLUMN invoices.amount_due: 'Sum of all invoice_items.amount for this invoice. Set by the application when the invoice is finalised. Not updated automatically.'
- COLUMN invoices.amount_paid: 'Running total of confirmed payments. Updated automatically by trg_sync_invoice_payment_status on every insert/delete on payments.'

**Indexes:**
- `idx_invoices_tenant` (tenant_id)
- `idx_invoices_student_term` (student_id, academic_term_id)
- `idx_invoices_parent` (parent_id)
- `idx_invoices_payment_status` (tenant_id, payment_status)

**Triggers:**
- `trg_invoices_updated_at` BEFORE UPDATE ON invoices FOR EACH ROW EXECUTE FUNCTION fn_set_updated_at()

### Invoice Items
- `id`: UUID (Primary Key) - Default: gen_random_uuid()
- `tenant_id`: UUID NOT NULL
- `invoice_id`: UUID NOT NULL
- `fee_category_id`: UUID NOT NULL
- `description`: VARCHAR(255) NULL
- `amount`: NUMERIC(12,2) NOT NULL CHECK (amount >= 0)
- `created_at`: TIMESTAMPTZ NOT NULL DEFAULT NOW()
- `updated_at`: TIMESTAMPTZ NOT NULL DEFAULT NOW()

**Constraints:**
- `fk_invoice_items_tenant_invoice` FOREIGN KEY (tenant_id, invoice_id) REFERENCES invoices(tenant_id, id) ON DELETE CASCADE
- `fk_invoice_items_tenant_fee_category` FOREIGN KEY (tenant_id, fee_category_id) REFERENCES fee_categories(tenant_id, id) ON DELETE CASCADE

**Indexes:**
- `idx_invoice_items_tenant` (tenant_id)
- `idx_invoice_items_invoice_id` (invoice_id)
- `idx_invoice_items_fee_category` (fee_category_id)

**Triggers:**
- `trg_invoice_items_updated_at` BEFORE UPDATE ON invoice_items FOR EACH ROW EXECUTE FUNCTION fn_set_updated_at()

### Payments
- `id`: UUID (Primary Key) - Default: gen_random_uuid()
- `tenant_id`: UUID NOT NULL
- `invoice_id`: UUID NOT NULL
- `amount`: NUMERIC(12,2) NOT NULL CHECK (amount > 0)
- `parent_id`: UUID NULL
- `payment_method`: payment_method_type NULL
- `reference_code`: VARCHAR(100) NULL
- `recorded_by`: UUID NOT NULL
- `created_at`: TIMESTAMPTZ NOT NULL DEFAULT NOW()
- `updated_at`: TIMESTAMPTZ NOT NULL DEFAULT NOW()

**Constraints:**
- `fk_payments_tenant_invoice` FOREIGN KEY (tenant_id, invoice_id) REFERENCES invoices(tenant_id, id) ON DELETE CASCADE
- `fk_payments_tenant_parent` FOREIGN KEY (tenant_id, parent_id) REFERENCES cbc_parents(tenant_id, id) ON DELETE SET NULL (parent_id)
- `fk_payments_tenant_recorded_by` FOREIGN KEY (tenant_id, recorded_by) REFERENCES users(tenant_id, id) ON DELETE RESTRICT

**Comments:**
- COLUMN payments.payment_method: 'Payment method type enum. Covers the four real Kenyan payment channels (MPESA, CASH, BANK_TRANSFER, CHEQUE) plus OTHER. Original free-text column migrated to enum in 000003_fix.'

**Indexes:**
- `idx_payments_tenant` (tenant_id)
- `idx_payments_invoice_id` (invoice_id)
- `idx_payments_parent` (parent_id)
- `idx_payments_reference_code` UNIQUE (reference_code) WHERE reference_code IS NOT NULL

**Triggers:**
- `trg_payments_updated_at` BEFORE UPDATE ON payments FOR EACH ROW EXECUTE FUNCTION fn_set_updated_at()
- `trg_sync_invoice_payment_status_insert` AFTER INSERT ON payments (statement-level, fn_sync_invoice_payment_status_insert)
- `trg_sync_invoice_payment_status_delete` AFTER DELETE ON payments (statement-level, fn_sync_invoice_payment_status_delete)
- `trg_sync_invoice_payment_status_update` AFTER UPDATE ON payments (statement-level, fn_sync_invoice_payment_status_update)

### CBC Learning Areas
- `id`: UUID (Primary Key) - Default: gen_random_uuid()
- `tenant_id`: UUID NOT NULL
- `school_id`: UUID NOT NULL
- `name`: VARCHAR(150) NOT NULL
- `code`: VARCHAR(50) NOT NULL
- `education_level`: cbc_education_level NOT NULL
- `grade_level`: cbc_grade_level NOT NULL

**Constraints:**
- `fk_cbc_learning_areas_tenant_school` FOREIGN KEY (tenant_id, school_id) REFERENCES cbc_schools(tenant_id, id) ON DELETE CASCADE
- `uq_cbc_learning_areas_school_code_grade` UNIQUE (tenant_id, school_id, code, grade_level)
- `uq_cbc_learning_areas_tenant` UNIQUE (tenant_id, id)

**Comments:**
- COLUMN cbc_learning_areas.education_level: 'The CBC tier this learning area belongs to, per KICD curriculum structure. Determines applicable KNEC assessment instruments and portal upload eligibility.'
- COLUMN cbc_learning_areas.code: 'Short KICD-defined code for this learning area, e.g. MATH, ENG, KISW, INT_SCI, PRE_TECH, SOC_STD. Unique within a school per grade level.'
- COLUMN cbc_learning_areas.grade_level: 'The specific CBC grade this learning area instance is taught in. Each grade has its own set of learning areas per KNEC/KICD curriculum. Combined with code to uniquely identify a learning area within a school.'

**Indexes:**
- `idx_cbc_learning_areas_tenant` (tenant_id)
- `idx_cbc_learning_areas_school_id` (school_id)
- `idx_cbc_learning_areas_education_level` (education_level)
- `idx_cbc_learning_areas_grade_level` (grade_level)

### CBC Strands
- `id`: UUID (Primary Key) - Default: gen_random_uuid()
- `tenant_id`: UUID NOT NULL
- `learning_area_id`: UUID NOT NULL
- `name`: VARCHAR(255) NOT NULL
- `updated_at`: TIMESTAMPTZ NOT NULL DEFAULT NOW()

**Constraints:**
- `uq_cbc_strands_tenant` UNIQUE (tenant_id, id)
- `fk_cbc_strands_tenant_learning_area` FOREIGN KEY (tenant_id, learning_area_id) REFERENCES cbc_learning_areas(tenant_id, id) ON DELETE CASCADE

**Indexes:**
- `idx_cbc_strands_learning_area_id` (learning_area_id)
- `idx_cbc_strands_tenant` (tenant_id)

**Triggers:**
- `trg_cbc_strands_updated_at` BEFORE UPDATE ON cbc_strands FOR EACH ROW EXECUTE FUNCTION fn_set_updated_at()

### CBC Sub-Strands
- `id`: UUID (Primary Key) - Default: gen_random_uuid()
- `tenant_id`: UUID NOT NULL
- `strand_id`: UUID NOT NULL
- `name`: VARCHAR(255) NOT NULL
- `updated_at`: TIMESTAMPTZ NOT NULL DEFAULT NOW()

**Constraints:**
- `uq_cbc_sub_strands_tenant` UNIQUE (tenant_id, id)
- `fk_cbc_sub_strands_tenant_strand` FOREIGN KEY (tenant_id, strand_id) REFERENCES cbc_strands(tenant_id, id) ON DELETE CASCADE

**Indexes:**
- `idx_cbc_sub_strands_strand_id` (strand_id)
- `idx_cbc_sub_strands_tenant` (tenant_id)

**Triggers:**
- `trg_cbc_sub_strands_updated_at` BEFORE UPDATE ON cbc_sub_strands FOR EACH ROW EXECUTE FUNCTION fn_set_updated_at()

### Performance Indicators
- `id`: UUID (Primary Key) - Default: gen_random_uuid()
- `tenant_id`: UUID NOT NULL
- `sub_strand_id`: UUID NOT NULL
- `description`: TEXT NOT NULL
- `sequence_order`: SMALLINT NOT NULL DEFAULT 1
- `updated_at`: TIMESTAMPTZ NOT NULL DEFAULT NOW()

**Constraints:**
- `fk_performance_indicators_tenant_sub_strand` FOREIGN KEY (tenant_id, sub_strand_id) REFERENCES cbc_sub_strands(tenant_id, id) ON DELETE CASCADE
- `uq_performance_indicators_tenant` UNIQUE (tenant_id, id)

**Comments:**
- TABLE performance_indicators: 'Atomic CBC learning outcomes within a sub-strand, as defined in KICD curriculum designs. Leaf nodes of the hierarchy: Learning Area → Strand → Sub-Strand → Performance Indicator.'

**Indexes:**
- `idx_performance_indicators_sub_strand` (sub_strand_id, sequence_order)
- `idx_performance_indicators_tenant` (tenant_id)

**Triggers:**
- `trg_performance_indicators_updated_at` BEFORE UPDATE ON performance_indicators FOR EACH ROW EXECUTE FUNCTION fn_set_updated_at()

### CBC Class Teachers
- `id`: UUID (Primary Key) - Default: gen_random_uuid()
- `tenant_id`: UUID NOT NULL
- `class_id`: UUID NOT NULL
- `user_id`: UUID NOT NULL
- `learning_area_id`: UUID NULL
- `teacher_role`: teacher_role NOT NULL DEFAULT 'SUBJECT_TEACHER'
- `created_at`: TIMESTAMPTZ NOT NULL DEFAULT NOW()
- `updated_at`: TIMESTAMPTZ NOT NULL DEFAULT NOW()

**Constraints:**
- `fk_cbc_class_teachers_tenant_user` FOREIGN KEY (tenant_id, user_id) REFERENCES users(tenant_id, id) ON DELETE CASCADE
- `fk_cbc_class_teachers_tenant_class` FOREIGN KEY (tenant_id, class_id) REFERENCES cbc_classes(tenant_id, id) ON DELETE CASCADE
- `fk_cbc_class_teachers_tenant_learning_area` FOREIGN KEY (tenant_id, learning_area_id) REFERENCES cbc_learning_areas(tenant_id, id) ON DELETE SET NULL (learning_area_id)
- `chk_cct_primary_no_area` CHECK (teacher_role != 'PRIMARY_CLASS_TEACHER' OR learning_area_id IS NULL)
- `chk_cct_subject_area_required` CHECK (teacher_role != 'SUBJECT_TEACHER' OR learning_area_id IS NOT NULL)
- `unique_cbc_class_teacher` UNIQUE (class_id, user_id, learning_area_id)

**Indexes:**
- `idx_cbc_class_teachers_tenant` (tenant_id)
- `idx_cbc_class_teachers_class_id` (class_id)
- `idx_cbc_class_teachers_user_id` (user_id)
- `idx_cbc_class_teachers_role` (teacher_role)
- `idx_cbc_one_primary_per_class` UNIQUE (class_id) WHERE teacher_role = 'PRIMARY_CLASS_TEACHER'

**Triggers:**
- `trg_cbc_class_teachers_updated_at` BEFORE UPDATE ON cbc_class_teachers FOR EACH ROW EXECUTE FUNCTION fn_set_updated_at()

### Timetable Structures
- `id`: UUID (Primary Key) - Default: gen_random_uuid()
- `tenant_id`: UUID NOT NULL
- `school_id`: UUID NOT NULL
- `academic_year_id`: UUID NOT NULL
- `day_of_week`: INT NOT NULL CHECK (day_of_week BETWEEN 1 AND 7)
- `period_name`: VARCHAR(50) NOT NULL
- `start_time`: TIME NOT NULL
- `end_time`: TIME NOT NULL
- `is_break`: BOOLEAN NOT NULL DEFAULT FALSE
- `created_at`: TIMESTAMPTZ NOT NULL DEFAULT NOW()
- `updated_at`: TIMESTAMPTZ NOT NULL DEFAULT NOW()

**Constraints:**
- `chk_timetable_structure_times` CHECK (end_time > start_time)
- `excl_timetable_structure_overlap` EXCLUDE USING gist (school_id WITH =, academic_year_id WITH =, day_of_week WITH =, fn_timerange(day_of_week, start_time, end_time) WITH &&)
- `fk_timetable_structure_tenant_school` FOREIGN KEY (tenant_id, school_id) REFERENCES cbc_schools(tenant_id, id) ON DELETE CASCADE
- `fk_timetable_structure_academic_year` FOREIGN KEY (tenant_id, academic_year_id) REFERENCES academic_years(tenant_id, id) ON DELETE CASCADE
- `uq_timetable_structures_tenant` UNIQUE (tenant_id, id)

**Comments:**
- TABLE timetable_structures: 'Structural day template (Grid Definition Layer). Defines the partitioned time blocks (lessons, breaks, assemblies) that make up a standard school day per academic year. The GiST exclusion constraint guarantees non-overlapping blocks per school per academic year per day. Decoupled from cbc_timetable_slots — allocations reference structure_id instead of carrying raw time ranges.'
- COLUMN timetable_structures.day_of_week: '1=Monday … 7=Sunday. Most schools use Mon-Fri (1-5); weekends are allowed for special sessions.'
- COLUMN timetable_structures.period_name: 'Human-readable name for this time period, e.g. "Lesson 1", "Morning Break", "Recess", "Assembly". Free-text — not an enum, to support school-specific naming.'
- COLUMN timetable_structures.is_break: 'Flags recess, lunch, or other non-instructional blocks. UI can use this to disable assignment cells and render break rows in a distinct style.'

**Indexes:**
- `idx_timetable_structure_tenant` (tenant_id)
- `idx_timetable_structure_school_day` (school_id, day_of_week)
- `idx_timetable_structure_academic_year` (academic_year_id)

**Triggers:**
- `trg_timetable_structures_updated_at` BEFORE UPDATE ON timetable_structures FOR EACH ROW EXECUTE FUNCTION fn_set_updated_at()

### CBC Timetable Slots
- `id`: UUID (Primary Key) - Default: gen_random_uuid()
- `tenant_id`: UUID NOT NULL
- `school_id`: UUID NOT NULL
- `academic_year_id`: UUID NOT NULL
- `structure_id`: UUID NOT NULL
- `class_id`: UUID NOT NULL
- `learning_area_id`: UUID NOT NULL
- `teacher_id`: UUID NOT NULL
- `room_identifier`: VARCHAR(50) NULL
- `created_at`: TIMESTAMPTZ NOT NULL DEFAULT NOW()
- `updated_at`: TIMESTAMPTZ NOT NULL DEFAULT NOW()

**Constraints:**
- `unique_class_slot` UNIQUE (academic_year_id, structure_id, class_id) - one assignment per class per structure block
- `unique_teacher_slot` UNIQUE (academic_year_id, structure_id, teacher_id) - prevents teacher double-booking
- `unique_room_slot` UNIQUE (academic_year_id, structure_id, room_identifier) - prevents room double-booking
- `fk_cbc_timetable_slots_tenant_school` FOREIGN KEY (tenant_id, school_id) REFERENCES cbc_schools(tenant_id, id) ON DELETE CASCADE
- `fk_cbc_timetable_slots_tenant_class` FOREIGN KEY (tenant_id, class_id) REFERENCES cbc_classes(tenant_id, id) ON DELETE CASCADE
- `fk_cbc_timetable_slots_tenant_teacher` FOREIGN KEY (tenant_id, teacher_id) REFERENCES users(tenant_id, id) ON DELETE CASCADE
- `fk_cbc_timetable_slots_tenant_learning_area` FOREIGN KEY (tenant_id, learning_area_id) REFERENCES cbc_learning_areas(tenant_id, id) ON DELETE CASCADE
- `fk_cbc_timetable_slots_tenant_structure` FOREIGN KEY (tenant_id, structure_id) REFERENCES timetable_structures(tenant_id, id) ON DELETE CASCADE
- `fk_cbc_timetable_slots_academic_year` FOREIGN KEY (tenant_id, academic_year_id) REFERENCES academic_years(tenant_id, id) ON DELETE CASCADE

**Comments:**
- TABLE cbc_timetable_slots: 'Grid Allocation Layer — lightweight relational mapping table using fast B-Tree composite unique constraints. The grid definition (time ranges) lives in timetable_structures; this table only stores assignments of class → teacher → learning_area → room per structure block.'

**Indexes:**
- `idx_cbc_timetable_slots_structure` (structure_id)
- `idx_cbc_timetable_slots_class` (class_id)
- `idx_cbc_timetable_slots_teacher` (teacher_id)
- `idx_cbc_timetable_slots_academic_year` (academic_year_id)
- `idx_cbc_timetable_slots_tenant` (tenant_id)
- `idx_cbc_timetable_slots_school` (school_id)
- `uq_cbc_timetable_slots_tenant` UNIQUE (tenant_id, id)

**Triggers:**
- `trg_cbc_timetable_slots_updated_at` BEFORE UPDATE ON cbc_timetable_slots FOR EACH ROW EXECUTE FUNCTION fn_set_updated_at()

### Assessment Weight Configs
- `id`: UUID (Primary Key) - Default: gen_random_uuid()
- `grade_level`: cbc_grade_level NOT NULL
- `assessment_type_code`: VARCHAR(50) NOT NULL
- `target_exam`: VARCHAR(20) NOT NULL
- `weight_percent`: NUMERIC(5,2) NOT NULL CHECK (weight_percent > 0 AND weight_percent <= 100)
- `effective_from`: INTEGER NOT NULL
- `notes`: TEXT NULL
- `created_at`: TIMESTAMPTZ NOT NULL DEFAULT NOW()

**Constraints:**
- `uq_assessment_weight_config` UNIQUE (grade_level, assessment_type_code, target_exam, effective_from)

**Comments:**
- TABLE assessment_weight_configs: 'Official KNEC assessment weighting formulas per grade level. These are nationally mandated and do not vary per school. Schema changes would be required if per-school overrides are ever needed.'
- COLUMN assessment_weight_configs.assessment_type_code: 'KNEC assessment type identifier, e.g. KNEC_SBA_Project, National_KPSEA, National_KJSEA.'
- COLUMN assessment_weight_configs.target_exam: 'The target national exam this weight contributes to: KPSEA, KJSEA, or KSSEA.'
- COLUMN assessment_weight_configs.weight_percent: 'Percentage contribution of this assessment component towards the target exam placement.'
- COLUMN assessment_weight_configs.effective_from: 'Academic year from which this weighting formula is effective.'

**Notes:**
- Seeded (000002) with the official KNEC formulas: KPSEA = 60% SBA (G4 20% + G5 20% + G6 20%) + 40% KPSEA written (G6); KJSEA = 20% KPSEA carry-forward (G6) + 20% SBA (G7 10% + G8 10%) + 60% KJSEA written (G9).

### School Member Counts
- `tenant_id`: UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE
- `school_id`: UUID NOT NULL
- `admins`: INTEGER NOT NULL DEFAULT 0
- `teachers`: INTEGER NOT NULL DEFAULT 0
- `nurses`: INTEGER NOT NULL DEFAULT 0
- `finance`: INTEGER NOT NULL DEFAULT 0
- `parents`: INTEGER NOT NULL DEFAULT 0
- `students`: INTEGER NOT NULL DEFAULT 0
- `updated_at`: TIMESTAMPTZ NOT NULL DEFAULT NOW()

**Constraints:**
- `pk_school_member_counts` PRIMARY KEY (tenant_id, school_id)
- `fk_school_member_counts_tenant_school` FOREIGN KEY (tenant_id, school_id) REFERENCES cbc_schools(tenant_id, id) ON DELETE CASCADE

**Comments:**
- Denormalised per-school counts. Staff counts kept in sync by `trg_memberships_counts_{insert,delete,update}` statement triggers on memberships; student counts by `trg_cbc_students_counts_{insert,delete,update}` statement triggers on cbc_students.

**Indexes:**
- Primary key (tenant_id, school_id) doubles as the lookup index

### Member Active School
- `user_id`: UUID NOT NULL
- `tenant_id`: UUID NOT NULL
- `school_id`: UUID NOT NULL
- `switched_at`: TIMESTAMPTZ NOT NULL DEFAULT NOW()
- `updated_at`: TIMESTAMPTZ NOT NULL DEFAULT NOW()

**Constraints:**
- PRIMARY KEY (user_id)
- `fk_mas_tenant_user` FOREIGN KEY (tenant_id, user_id) REFERENCES users(tenant_id, id) ON DELETE CASCADE
- `fk_mas_tenant_school` FOREIGN KEY (tenant_id, school_id) REFERENCES cbc_schools(tenant_id, id) ON DELETE CASCADE
- `fk_mas_membership` FOREIGN KEY (user_id, school_id) REFERENCES memberships(user_id, school_id) ON DELETE CASCADE

**Comments:**
- TABLE member_active_school: 'Tracks the currently active school context for each user within a tenant. One row per user. Upsert on school switch. The chosen school_id is constrained to schools the user is an active member of via fk_mas_membership.'

**Indexes:**
- `idx_mas_tenant_id` (tenant_id)

**Triggers:**
- `trg_member_active_school_updated_at` BEFORE UPDATE ON member_active_school FOR EACH ROW EXECUTE FUNCTION fn_set_updated_at()

### CBC Attendance Sessions
- `id`: UUID (Primary Key) - Default: gen_random_uuid()
- `tenant_id`: UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE
- `school_id`: UUID NOT NULL
- `timetable_slot_id`: UUID NOT NULL
- `date`: DATE NOT NULL
- `status`: VARCHAR(20) NOT NULL DEFAULT 'SUBMITTED'
- `skip_reason`: TEXT NULL
- `created_at`: TIMESTAMPTZ NOT NULL DEFAULT NOW()
- `updated_at`: TIMESTAMPTZ NOT NULL DEFAULT NOW()

**Constraints:**
- `chk_cbc_attendance_session_status` CHECK (status IN ('SUBMITTED', 'SKIPPED'))
- `uq_cbc_attendance_sessions_slot_date` UNIQUE (school_id, timetable_slot_id, date)
- `fk_cbc_attendance_sessions_tenant_slot` FOREIGN KEY (tenant_id, timetable_slot_id) REFERENCES cbc_timetable_slots(tenant_id, id) ON DELETE CASCADE
- `fk_cbc_attendance_sessions_tenant_school` FOREIGN KEY (tenant_id, school_id) REFERENCES cbc_schools(tenant_id, id) ON DELETE CASCADE

**Comments:**
- TABLE cbc_attendance_sessions: 'Tracks actual lesson execution instances per timetable slot and date. Teachers flag sessions as SKIPPED when a class did not hold (teacher absence, school assembly, sports day, etc.). Skipped sessions exclude their attendance records from terminal percentage calculations so students are not penalised for cancelled lessons.'
- COLUMN cbc_attendance_sessions.status: 'SUBMITTED = lesson held as scheduled (default). SKIPPED = lesson did not hold. Only SKIPPED sessions affect terminal attendance calculations by reducing the expected denominator.'
- COLUMN cbc_attendance_sessions.skip_reason: 'Teacher-provided reason when status is SKIPPED. Examples: School Assembly, Public Holiday, Teacher Absence, Sports/Field Event.'

**Indexes:**
- `idx_cbc_attendance_sessions_slot_date` (timetable_slot_id, date)
- `idx_cbc_attendance_sessions_tenant` (tenant_id)
- `idx_cbc_attendance_sessions_school` (school_id)
- `idx_cbc_attendance_sessions_status` (status)
- `uq_cbc_attendance_sessions_tenant` UNIQUE (tenant_id, id)

**Triggers:**
- `trg_cbc_attendance_sessions_updated_at` BEFORE UPDATE ON cbc_attendance_sessions FOR EACH ROW EXECUTE FUNCTION fn_set_updated_at()

### Attendance Records
- `id`: UUID (Primary Key) - Default: gen_random_uuid()
- `tenant_id`: UUID NOT NULL
- `school_id`: UUID NOT NULL
- `student_id`: UUID NOT NULL
- `timetable_slot_id`: UUID NOT NULL
- `academic_term_id`: UUID NOT NULL
- `date`: DATE NOT NULL
- `status`: attendance_status NOT NULL
- `marked_by`: UUID NOT NULL
- `marked_at`: TIMESTAMPTZ NOT NULL DEFAULT NOW()
- `note`: TEXT NULL
- `attendance_session_id`: UUID NULL
- `created_at`: TIMESTAMPTZ NOT NULL DEFAULT NOW()
- `updated_at`: TIMESTAMPTZ NOT NULL DEFAULT NOW()

**Constraints:**
- `uq_attendance_student_slot_date` UNIQUE (student_id, timetable_slot_id, date)
- `fk_attendance_tenant_student` FOREIGN KEY (tenant_id, student_id) REFERENCES cbc_students(tenant_id, id) ON DELETE CASCADE
- `fk_attendance_timetable_slot` FOREIGN KEY (tenant_id, timetable_slot_id) REFERENCES cbc_timetable_slots(tenant_id, id) ON DELETE CASCADE
- `fk_attendance_tenant_session` FOREIGN KEY (tenant_id, attendance_session_id) REFERENCES cbc_attendance_sessions(tenant_id, id) ON DELETE SET NULL (attendance_session_id)
- `fk_attendance_tenant_term` FOREIGN KEY (tenant_id, school_id, academic_term_id) REFERENCES academic_terms(tenant_id, school_id, id) ON DELETE CASCADE
- `fk_attendance_marked_by` FOREIGN KEY (tenant_id, marked_by) REFERENCES users(tenant_id, id) ON DELETE RESTRICT

**Comments:**
- TABLE attendance_records: 'Per-student, per-timetable-slot, per-date attendance records. The unique constraint (student_id, timetable_slot_id, date) prevents duplicate marks. Only created for slots where timetable_structures.is_break = false.'
- COLUMN attendance_records.attendance_session_id: 'Optional reference to the cbc_attendance_sessions row. Populated when session is marked as SKIPPED to link existing records. NULL for normal (non-skipped) attendance marks.'

**Indexes:**
- `idx_attendance_slot_date` (timetable_slot_id, date)
- `idx_attendance_student_term` (student_id, academic_term_id)
- `idx_attendance_tenant` (tenant_id)
- `idx_attendance_school` (school_id)
- `idx_attendance_records_session` (attendance_session_id)

**Triggers:**
- `trg_attendance_records_updated_at` BEFORE UPDATE ON attendance_records FOR EACH ROW EXECUTE FUNCTION fn_set_updated_at()
- `trg_attendance_check_non_break_slot` BEFORE INSERT OR UPDATE ON attendance_records FOR EACH ROW EXECUTE FUNCTION fn_check_non_break_slot()

### Behavior Categories
- `id`: UUID (Primary Key) - Default: gen_random_uuid()
- `tenant_id`: UUID NOT NULL
- `school_id`: UUID NOT NULL
- `name`: VARCHAR(100) NOT NULL
- `default_severity`: behavior_severity NULL
- `is_active`: BOOLEAN NOT NULL DEFAULT true
- `category_type`: behavior_category_type NOT NULL DEFAULT 'DISCIPLINARY'
- `created_at`: TIMESTAMPTZ NOT NULL DEFAULT NOW()
- `updated_at`: TIMESTAMPTZ NOT NULL DEFAULT NOW()

**Constraints:**
- `fk_behavior_categories_tenant_school` FOREIGN KEY (tenant_id, school_id) REFERENCES cbc_schools(tenant_id, id) ON DELETE CASCADE
- `uq_behavior_category_name` UNIQUE (tenant_id, school_id, name)
- `uq_behavior_categories_tenant` UNIQUE (tenant_id, id)

**Comments:**
- TABLE behavior_categories: 'School-configurable behavior/incident categories. Admins manage these per school rather than a fixed platform-wide enum. Categories are soft-deleted (is_active = false) to preserve historical behavior_notes.'
- COLUMN behavior_categories.category_type: 'Classification of the behavior category: COMMENDATION (positive/laudable behaviour), DISCIPLINARY (negative behaviour / infraction), or OTHER. Used by student_behavior_term_summaries to compute commendations_count and disciplinary_count. Defaults to DISCIPLINARY for existing categories.'

**Indexes:**
- `idx_behavior_categories_tenant` (tenant_id)
- `idx_behavior_categories_school` (school_id)

**Triggers:**
- `trg_behavior_categories_updated_at` BEFORE UPDATE ON behavior_categories FOR EACH ROW EXECUTE FUNCTION fn_set_updated_at()

### Behavior Notes
- `id`: UUID (Primary Key) - Default: gen_random_uuid()
- `tenant_id`: UUID NOT NULL
- `school_id`: UUID NOT NULL
- `student_id`: UUID NOT NULL
- `timetable_slot_id`: UUID NOT NULL
- `date`: DATE NOT NULL
- `category_id`: UUID NOT NULL
- `description`: TEXT NOT NULL
- `is_urgent`: BOOLEAN NOT NULL DEFAULT false
- `status`: behavior_note_status NOT NULL DEFAULT 'PENDING_REVIEW'
- `authored_by_id`: UUID NOT NULL
- `reviewed_by_id`: UUID NULL
- `reviewed_at`: TIMESTAMPTZ NULL
- `created_at`: TIMESTAMPTZ NOT NULL DEFAULT NOW()
- `updated_at`: TIMESTAMPTZ NOT NULL DEFAULT NOW()

**Constraints:**
- `fk_behavior_notes_tenant_student` FOREIGN KEY (tenant_id, student_id) REFERENCES cbc_students(tenant_id, id) ON DELETE CASCADE
- `fk_behavior_notes_timetable_slot` FOREIGN KEY (tenant_id, timetable_slot_id) REFERENCES cbc_timetable_slots(tenant_id, id) ON DELETE CASCADE
- `fk_behavior_notes_tenant_category` FOREIGN KEY (tenant_id, category_id) REFERENCES behavior_categories(tenant_id, id) ON DELETE RESTRICT
- `fk_behavior_notes_authored_by` FOREIGN KEY (tenant_id, authored_by_id) REFERENCES users(tenant_id, id) ON DELETE RESTRICT
- `fk_behavior_notes_reviewed_by` FOREIGN KEY (tenant_id, reviewed_by_id) REFERENCES users(tenant_id, id) ON DELETE SET NULL (reviewed_by_id)

**Comments:**
- TABLE behavior_notes: 'Sparse incident/behavior records logged by teachers. Each note is associated with a specific student, timetable slot, and date. Notes go through admin approval (PENDING_REVIEW → APPROVED/REJECTED) before being included in term reports or reaching parents. Urgent notes bypass term-end batching for immediate parent contact.'
- COLUMN behavior_notes.is_urgent: 'When true and approved, triggers immediate parent notification instead of waiting for term-end compilation.'

**Indexes:**
- `idx_behavior_notes_student` (student_id)
- `idx_behavior_notes_status` (status)
- `idx_behavior_notes_urgent` (is_urgent) WHERE is_urgent = true
- `idx_behavior_notes_slot_date` (timetable_slot_id, date)
- `idx_behavior_notes_tenant` (tenant_id)
- `idx_behavior_notes_school` (school_id)

**Triggers:**
- `trg_behavior_notes_updated_at` BEFORE UPDATE ON behavior_notes FOR EACH ROW EXECUTE FUNCTION fn_set_updated_at()
- `trg_behavior_notes_refresh_term_summary` AFTER INSERT OR UPDATE ON behavior_notes FOR EACH ROW EXECUTE FUNCTION fn_refresh_student_behavior_term_summary_for_note()

### Attendance Term Summaries
- `id`: UUID (Primary Key) - Default: gen_random_uuid()
- `tenant_id`: UUID NOT NULL
- `school_id`: UUID NOT NULL
- `student_id`: UUID NOT NULL
- `academic_term_id`: UUID NOT NULL
- `learning_area_id`: UUID NOT NULL
- `periods_total`: INT NOT NULL
- `periods_present`: INT NOT NULL
- `periods_absent`: INT NOT NULL
- `periods_late`: INT NOT NULL
- `periods_excused`: INT NOT NULL
- `attendance_percentage`: NUMERIC(5,2) NOT NULL
- `last_refreshed_at`: TIMESTAMPTZ NOT NULL DEFAULT NOW()
- `academic_year_id`: UUID NOT NULL
- `created_at`: TIMESTAMPTZ NOT NULL DEFAULT NOW()
- `updated_at`: TIMESTAMPTZ NOT NULL DEFAULT NOW()

**Constraints:**
- `fk_summaries_tenant_student` FOREIGN KEY (tenant_id, student_id) REFERENCES cbc_students(tenant_id, id) ON DELETE CASCADE
- `fk_summaries_tenant_term` FOREIGN KEY (tenant_id, school_id, academic_term_id) REFERENCES academic_terms(tenant_id, school_id, id) ON DELETE CASCADE
- `fk_summaries_learning_area` FOREIGN KEY (tenant_id, learning_area_id) REFERENCES cbc_learning_areas(tenant_id, id) ON DELETE CASCADE
- `uq_summary_student_term_area` UNIQUE (student_id, academic_term_id, learning_area_id)
- `fk_summaries_tenant_academic_year` FOREIGN KEY (tenant_id, academic_year_id) REFERENCES academic_years(tenant_id, id) ON DELETE CASCADE

**Comments:**
- TABLE attendance_term_summaries: 'Materialised rollup of attendance records per student per term per learning area. Populated by a background task (nightly or on-demand when an admin generates a term report). Not authoritative — attendance_records is the source of truth for all attendance calculations.'
- COLUMN attendance_term_summaries.attendance_percentage: 'Calculated as (periods_present / periods_total) * 100, stored as a decimal with two fractional digits (e.g. 92.50).'

**Indexes:**
- `idx_att_summaries_student_term` (student_id, academic_term_id)
- `idx_att_summaries_tenant` (tenant_id)
- `idx_att_summaries_school` (school_id)
- `idx_att_summaries_academic_year` (academic_year_id)

**Triggers:**
- `trg_attendance_term_summaries_updated_at` BEFORE UPDATE ON attendance_term_summaries FOR EACH ROW EXECUTE FUNCTION fn_set_updated_at()

### Grading Scale Profiles
- `id`: UUID (Primary Key) - Default: gen_random_uuid()
- `tenant_id`: UUID NOT NULL
- `school_id`: UUID NOT NULL
- `name`: VARCHAR(255) NOT NULL
- `is_active`: BOOLEAN NOT NULL DEFAULT true
- `created_at`: TIMESTAMPTZ NOT NULL DEFAULT NOW()
- `updated_at`: TIMESTAMPTZ NOT NULL DEFAULT NOW()

**Constraints:**
- `fk_grading_scale_profiles_tenant_school` FOREIGN KEY (tenant_id, school_id) REFERENCES cbc_schools(tenant_id, id) ON DELETE CASCADE
- `uq_grading_scale_profiles_tenant` UNIQUE (tenant_id, id)

**Comments:**
- TABLE grading_scale_profiles: 'Directory of CBC grading scale profiles. Profiles define the translation from numeric percentages to CBC rubric levels (EE, ME, AE, BE). Once created, profile name and settings are read-only. To change a scale, create a new profile and mark the old one is_active = false.'

**Indexes:**
- `idx_grading_scale_profiles_tenant` (tenant_id)
- `idx_grading_scale_profiles_school` (school_id)

**Triggers:**
- `trg_grading_scale_profiles_updated_at` BEFORE UPDATE ON grading_scale_profiles FOR EACH ROW EXECUTE FUNCTION fn_set_updated_at()

### Grading Scale Ranges
- `id`: UUID (Primary Key) - Default: gen_random_uuid()
- `profile_id`: UUID NOT NULL
- `tenant_id`: UUID NOT NULL
- `performance_level`: cbc_performance_level NOT NULL
- `min_percentage`: NUMERIC(5,2) NOT NULL CHECK (min_percentage >= 0 AND min_percentage <= 100)
- `max_percentage`: NUMERIC(5,2) NOT NULL CHECK (max_percentage >= 0 AND max_percentage <= 100)
- `default_percentage_mapping`: NUMERIC(5,2) NULL
- `created_at`: TIMESTAMPTZ NOT NULL DEFAULT NOW()
- `updated_at`: TIMESTAMPTZ NOT NULL DEFAULT NOW()

**Constraints:**
- `chk_range_bounds` CHECK (max_percentage > min_percentage)
- `uq_profile_level` UNIQUE (profile_id, performance_level)
- `excl_profile_range_no_overlap` EXCLUDE USING gist (profile_id WITH =, numrange(min_percentage, max_percentage, '[]') WITH &&)
- `fk_grading_scale_ranges_tenant_profile` FOREIGN KEY (tenant_id, profile_id) REFERENCES grading_scale_profiles(tenant_id, id) ON DELETE CASCADE

**Comments:**
- TABLE grading_scale_ranges: 'Range definitions within a grading scale profile. The EXCLUDE constraint using numrange guarantees no overlapping percentage bands within the same profile. Rows are write-once — UPDATE and DELETE are blocked at the application layer once the profile is actively referenced by sessions.'
- COLUMN grading_scale_ranges.default_percentage_mapping: 'Optional midpoint value used as the default when converting a percentage to a performance level. If NULL, the system uses the midpoint of the range. Example: for range 80-100 → EE, default could be 90.'
- COLUMN grading_scale_ranges.tenant_id: 'Denormalised from grading_scale_profiles for RLS enforcement. Must match the tenant_id of the referenced profile.'

**Indexes:**
- `idx_grading_scale_ranges_tenant` (tenant_id)

**Triggers:**
- `trg_grading_scale_ranges_updated_at` BEFORE UPDATE ON grading_scale_ranges FOR EACH ROW EXECUTE FUNCTION fn_set_updated_at()
- `trg_grading_scale_ranges_immutable` BEFORE UPDATE OR DELETE ON grading_scale_ranges FOR EACH ROW EXECUTE FUNCTION fn_block_grading_scale_range_modification()

### Assessment Sessions
- `id`: UUID (Primary Key) - Default: gen_random_uuid()
- `tenant_id`: UUID NOT NULL
- `school_id`: UUID NOT NULL
- `class_id`: UUID NOT NULL
- `learning_area_id`: UUID NOT NULL
- `academic_term_id`: UUID NOT NULL
- `academic_year_id`: UUID NOT NULL
- `name`: VARCHAR(255) NOT NULL
- `evaluation_method`: assessment_evaluation_method NOT NULL
- `max_points`: NUMERIC(10,2) NULL
- `grading_scale_profile_id`: UUID NULL
- `status`: assessment_session_status NOT NULL DEFAULT 'DRAFT'
- `rejection_comment`: TEXT NULL
- `submitted_by`: UUID NULL
- `approved_by`: UUID NULL
- `scheduled_date`: DATE NULL
- `created_at`: TIMESTAMPTZ NOT NULL DEFAULT NOW()
- `updated_at`: TIMESTAMPTZ NOT NULL DEFAULT NOW()
- `created_by`: UUID NOT NULL

**Constraints:**
- `fk_assessment_sessions_tenant_school` FOREIGN KEY (tenant_id, school_id) REFERENCES cbc_schools(tenant_id, id) ON DELETE CASCADE
- `fk_assessment_sessions_tenant_class` FOREIGN KEY (tenant_id, class_id) REFERENCES cbc_classes(tenant_id, id) ON DELETE CASCADE
- `fk_assessment_sessions_tenant_term` FOREIGN KEY (tenant_id, school_id, academic_term_id) REFERENCES academic_terms(tenant_id, school_id, id) ON DELETE CASCADE
- `fk_assessment_sessions_learning_area` FOREIGN KEY (tenant_id, learning_area_id) REFERENCES cbc_learning_areas(tenant_id, id) ON DELETE CASCADE
- `fk_assessment_sessions_tenant_scale_profile` FOREIGN KEY (tenant_id, grading_scale_profile_id) REFERENCES grading_scale_profiles(tenant_id, id) ON DELETE SET NULL (grading_scale_profile_id)
- `fk_assessment_sessions_tenant_submitted_by` FOREIGN KEY (tenant_id, submitted_by) REFERENCES users(tenant_id, id) ON DELETE SET NULL (submitted_by)
- `fk_assessment_sessions_tenant_approved_by` FOREIGN KEY (tenant_id, approved_by) REFERENCES users(tenant_id, id) ON DELETE SET NULL (approved_by)
- `fk_assessment_sessions_tenant_created_by` FOREIGN KEY (tenant_id, created_by) REFERENCES users(tenant_id, id)
- `uq_assessment_sessions_tenant` UNIQUE (tenant_id, id)
- `chk_quantitative_has_points` CHECK (evaluation_method != 'QUANTITATIVE' OR max_points IS NOT NULL)
- `chk_quantitative_has_scale` CHECK (evaluation_method != 'QUANTITATIVE' OR grading_scale_profile_id IS NOT NULL)
- `chk_rubric_no_points` CHECK (evaluation_method != 'RUBRIC' OR max_points IS NULL)
- `chk_rubric_no_scale` CHECK (evaluation_method != 'RUBRIC' OR grading_scale_profile_id IS NULL)

**Comments:**
- TABLE assessment_sessions: 'Tracks CBC assessment sessions through their lifecycle: DRAFT (teacher creating/grading) → PENDING_APPROVAL (submitted to admin) → PUBLISHED (approved, visible to parents). Rejection returns to DRAFT. Supports two evaluation methods: QUANTITATIVE (total marks converted via grading scale) and RUBRIC (direct indicator-level grading).'
- COLUMN assessment_sessions.max_points: 'Total possible marks for QUANTITATIVE sessions. NULL for RUBRIC sessions. Cannot be updated once any student score rows exist.'
- COLUMN assessment_sessions.rejection_comment: 'Admin feedback when rejecting a session. Cleared on re-submission.'

**Indexes:**
- `idx_assessment_sessions_tenant` (tenant_id)
- `idx_assessment_sessions_school` (school_id)
- `idx_assessment_sessions_class` (class_id)
- `idx_assessment_sessions_term` (academic_term_id)
- `idx_assessment_sessions_status` (status)
- `idx_assessment_sessions_learning_area` (learning_area_id)

**Triggers:**
- `trg_assessment_sessions_updated_at` BEFORE UPDATE ON assessment_sessions FOR EACH ROW EXECUTE FUNCTION fn_set_updated_at()
- `trg_assessment_sessions_max_points_immutable` BEFORE UPDATE ON assessment_sessions WHEN (OLD.max_points IS DISTINCT FROM NEW.max_points) EXECUTE FUNCTION fn_block_assessment_max_points_update()
- `trg_assessment_sessions_refresh_summary` AFTER UPDATE OF status ON assessment_sessions FOR EACH ROW EXECUTE FUNCTION fn_assessment_sessions_after_publish()

### Student Assessment Scores
- `id`: UUID (Primary Key) - Default: gen_random_uuid()
- `tenant_id`: UUID NOT NULL
- `session_id`: UUID NOT NULL
- `student_id`: UUID NOT NULL
- `raw_score`: NUMERIC(10,2) NULL
- `calculated_percentage`: NUMERIC(5,2) NULL
- `final_performance_level`: cbc_performance_level NULL
- `enrollment_status`: VARCHAR(20) NOT NULL DEFAULT 'ACTIVE'
- `created_at`: TIMESTAMPTZ NOT NULL DEFAULT NOW()
- `updated_at`: TIMESTAMPTZ NOT NULL DEFAULT NOW()

**Constraints:**
- `fk_scores_tenant_session` FOREIGN KEY (tenant_id, session_id) REFERENCES assessment_sessions(tenant_id, id) ON DELETE CASCADE
- `fk_scores_tenant_student` FOREIGN KEY (tenant_id, student_id) REFERENCES cbc_students(tenant_id, id) ON DELETE CASCADE
- `uq_score_session_student` UNIQUE (session_id, student_id)
- `chk_score_range` CHECK (raw_score IS NULL OR (raw_score >= 0 AND max_points_check(session_id, raw_score)))

**Comments:**
- TABLE student_assessment_scores: 'Stores student scores for QUANTITATIVE assessment sessions. The final_performance_level is written (snapshotted) at the moment of admin approval — immune to later scale profile changes. NULL for RUBRIC sessions (those use student_assessment_outcome_grades).'
- COLUMN student_assessment_scores.enrollment_status: 'Denormalised enrollment status at time of grading. Used to enforce the No-Grade-Ghosting constraint: scores cannot be entered for students marked ABSENT or EXEMPT. Values: ACTIVE, SUSPENDED, TRANSFERRED, ABSENT, EXEMPT.'

**Indexes:**
- `idx_student_scores_session` (session_id)
- `idx_student_scores_student` (student_id)
- `idx_student_scores_tenant` (tenant_id)

**Triggers:**
- `trg_student_assessment_scores_updated_at` BEFORE UPDATE ON student_assessment_scores FOR EACH ROW EXECUTE FUNCTION fn_set_updated_at()

### Student Assessment Outcome Grades
- `id`: UUID (Primary Key) - Default: gen_random_uuid()
- `tenant_id`: UUID NOT NULL
- `session_id`: UUID NOT NULL
- `student_id`: UUID NOT NULL
- `performance_indicator_id`: UUID NOT NULL
- `awarded_level`: cbc_performance_level NOT NULL
- `created_at`: TIMESTAMPTZ NOT NULL DEFAULT NOW()
- `updated_at`: TIMESTAMPTZ NOT NULL DEFAULT NOW()

**Constraints:**
- `fk_outcome_tenant_session` FOREIGN KEY (tenant_id, session_id) REFERENCES assessment_sessions(tenant_id, id) ON DELETE CASCADE
- `fk_outcome_tenant_student` FOREIGN KEY (tenant_id, student_id) REFERENCES cbc_students(tenant_id, id) ON DELETE CASCADE
- `fk_outcome_performance_indicator` FOREIGN KEY (tenant_id, performance_indicator_id) REFERENCES performance_indicators(tenant_id, id) ON DELETE CASCADE
- `uq_outcome_session_student_indicator` UNIQUE (session_id, student_id, performance_indicator_id)

**Comments:**
- TABLE student_assessment_outcome_grades: 'Stores rubric-level grades for RUBRIC assessment sessions. Each row maps a student to a specific KICD performance indicator with the awarded CBC level (EE, ME, AE, BE). No raw scores or percentages are stored — the teacher assigns the performance level directly.'

**Indexes:**
- `idx_outcome_grades_session` (session_id)
- `idx_outcome_grades_student` (student_id)
- `idx_outcome_grades_indicator` (performance_indicator_id)
- `idx_outcome_grades_tenant` (tenant_id)

**Triggers:**
- `trg_student_assessment_outcome_grades_updated_at` BEFORE UPDATE ON student_assessment_outcome_grades FOR EACH ROW EXECUTE FUNCTION fn_set_updated_at()

### Class Daily Attendance Summaries
- `id`: UUID (Primary Key) - Default: gen_random_uuid()
- `tenant_id`: UUID NOT NULL
- `school_id`: UUID NOT NULL
- `class_id`: UUID NOT NULL
- `academic_term_id`: UUID NOT NULL
- `date`: DATE NOT NULL
- `total_enrolled`: INT NOT NULL
- `present_count`: INT NOT NULL
- `absent_count`: INT NOT NULL
- `late_count`: INT NOT NULL
- `excused_count`: INT NOT NULL
- `daily_attendance_rate`: NUMERIC(5,2) NOT NULL
- `last_refreshed_at`: TIMESTAMPTZ NOT NULL DEFAULT NOW()
- `created_at`: TIMESTAMPTZ NOT NULL DEFAULT NOW()
- `updated_at`: TIMESTAMPTZ NOT NULL DEFAULT NOW()

**Constraints:**
- `uq_class_daily_attendance` UNIQUE (class_id, date)
- `fk_class_daily_tenant_class` FOREIGN KEY (tenant_id, class_id) REFERENCES cbc_classes(tenant_id, id) ON DELETE CASCADE
- `fk_class_daily_tenant_term` FOREIGN KEY (tenant_id, school_id, academic_term_id) REFERENCES academic_terms(tenant_id, school_id, id) ON DELETE CASCADE

**Comments:**
- TABLE class_daily_attendance_summaries: 'Materialised rollup of attendance records per class per date. "Total enrolled" is derived from distinct students who have attendance_records rows that day, not from cbc_student_enrollments.status, because enrollment status has no effective date within a term — a student suspended on day 50 would otherwise vanish from every earlier day too. This is a documented workaround, not a perfect fix.'
- COLUMN class_daily_attendance_summaries.daily_attendance_rate: 'Calculated as (present_count / (present_count + absent_count + late_count + excused_count)) * 100, stored as a decimal with two fractional digits (e.g. 94.60).'

**Indexes:**
- `idx_class_daily_tenant` (tenant_id)
- `idx_class_daily_school` (school_id)
- `idx_class_daily_class_date` (class_id, date)
- `idx_class_daily_academic_term` (academic_term_id)

**Triggers:**
- `trg_class_daily_attendance_summaries_updated_at` BEFORE UPDATE ON class_daily_attendance_summaries FOR EACH ROW EXECUTE FUNCTION fn_set_updated_at()

### Student Term Subject Summaries
- `id`: UUID (Primary Key) - Default: gen_random_uuid()
- `tenant_id`: UUID NOT NULL
- `school_id`: UUID NOT NULL
- `student_id`: UUID NOT NULL
- `academic_term_id`: UUID NOT NULL
- `learning_area_id`: UUID NOT NULL
- `average_percentage`: NUMERIC(5,2)
- `mapped_performance_level`: VARCHAR(5)
- `quantitative_assessment_count`: INT NOT NULL DEFAULT 0
- `rubric_assessment_count`: INT NOT NULL DEFAULT 0
- `indicators_assessed_count`: INT NOT NULL DEFAULT 0
- `has_quantitative_data`: BOOLEAN NOT NULL DEFAULT false
- `has_rubric_data`: BOOLEAN NOT NULL DEFAULT false
- `teacher_remark`: TEXT
- `last_refreshed_at`: TIMESTAMPTZ NOT NULL DEFAULT NOW()
- `created_at`: TIMESTAMPTZ NOT NULL DEFAULT NOW()
- `updated_at`: TIMESTAMPTZ NOT NULL DEFAULT NOW()

**Constraints:**
- `uq_student_term_subject` UNIQUE (student_id, academic_term_id, learning_area_id)
- `fk_summaries_tenant_student` FOREIGN KEY (tenant_id, student_id) REFERENCES cbc_students(tenant_id, id) ON DELETE CASCADE
- `fk_summaries_tenant_term` FOREIGN KEY (tenant_id, school_id, academic_term_id) REFERENCES academic_terms(tenant_id, school_id, id) ON DELETE CASCADE
- `fk_summaries_tenant_school` FOREIGN KEY (tenant_id, school_id) REFERENCES cbc_schools(tenant_id, id) ON DELETE CASCADE
- `fk_summaries_learning_area` FOREIGN KEY (tenant_id, learning_area_id) REFERENCES cbc_learning_areas(tenant_id, id) ON DELETE CASCADE

**Comments:**
- TABLE student_term_subject_summaries: 'Materialised blended rollup of assessment results per student, term, and learning area. Populated via fn_refresh_term_subject_summary_for_session() when an assessment session transitions to PUBLISHED. Quantitative scores contribute their calculated_percentage directly; rubric outcome grades are converted via default_percentage_mapping. The has_quantitative_data / has_rubric_data flags let reports distinguish blended averages from single-source averages.'
- COLUMN student_term_subject_summaries.average_percentage: 'Weighted average across all PUBLISHED assessment scores for this student+term+learning_area. NULL when neither quantitative nor rubric data exists.'
- COLUMN student_term_subject_summaries.mapped_performance_level: 'The CBC performance level (EE/ME/AE/BE) corresponding to average_percentage, determined by the grading scale profile used in the most recent QUANTITATIVE session for this term+learning_area.'
- COLUMN student_term_subject_summaries.teacher_remark: 'Optional free-text remark entered by the subject teacher during term-end compilation. Not populated automatically — set via API by the teacher.'

**Indexes:**
- `idx_term_subject_summaries_tenant` (tenant_id)
- `idx_term_subject_summaries_school` (school_id)
- `idx_term_subject_summaries_student_term` (student_id, academic_term_id)
- `idx_term_subject_summaries_term_area` (academic_term_id, learning_area_id)

**Triggers:**
- `trg_student_term_subject_summaries_updated_at` BEFORE UPDATE ON student_term_subject_summaries FOR EACH ROW EXECUTE FUNCTION fn_set_updated_at()

### Student Term Overall Summaries
- `id`: UUID (Primary Key) - Default: gen_random_uuid()
- `tenant_id`: UUID NOT NULL
- `school_id`: UUID NOT NULL
- `student_id`: UUID NOT NULL
- `academic_term_id`: UUID NOT NULL
- `subjects_assessed_count`: INT NOT NULL DEFAULT 0
- `overall_mean_percentage`: NUMERIC(5,2)
- `overall_performance_level`: VARCHAR(5)
- `exceeding_count`: INT NOT NULL DEFAULT 0
- `meeting_count`: INT NOT NULL DEFAULT 0
- `approaching_count`: INT NOT NULL DEFAULT 0
- `below_count`: INT NOT NULL DEFAULT 0
- `is_weighted_exam_score`: BOOLEAN NOT NULL DEFAULT false
- `headteacher_remark`: TEXT
- `last_refreshed_at`: TIMESTAMPTZ NOT NULL DEFAULT NOW()
- `created_at`: TIMESTAMPTZ NOT NULL DEFAULT NOW()
- `updated_at`: TIMESTAMPTZ NOT NULL DEFAULT NOW()

**Constraints:**
- `uq_student_term_overall` UNIQUE (student_id, academic_term_id)
- `fk_overall_tenant_student` FOREIGN KEY (tenant_id, student_id) REFERENCES cbc_students(tenant_id, id) ON DELETE CASCADE
- `fk_overall_tenant_term` FOREIGN KEY (tenant_id, school_id, academic_term_id) REFERENCES academic_terms(tenant_id, school_id, id) ON DELETE CASCADE
- `fk_overall_tenant_school` FOREIGN KEY (tenant_id, school_id) REFERENCES cbc_schools(tenant_id, id) ON DELETE CASCADE

**Comments:**
- TABLE student_term_overall_summaries: 'Second-level term rollup per student. Aggregates across all learning-area summaries to produce an overall mean, performance level, and per-level counts. The is_weighted_exam_score flag tells consumers whether a KNEC national-exam weighting formula was applied. Populated on-demand via fn_compute_term_overall_summaries_for_term() or nightly batch.'
- COLUMN student_term_overall_summaries.overall_mean_percentage: 'Mean of all per-subject average_percentage values. For non-final terms this is a straight average; for final exam terms (G6/G9/G12) it is a weighted blend using assessment_weight_configs. NULL when no subject data exists.'
- COLUMN student_term_overall_summaries.is_weighted_exam_score: 'TRUE when a KNEC weighting formula (from assessment_weight_configs) was applied instead of a plain average.'
- COLUMN student_term_overall_summaries.headteacher_remark: 'Optional free-text remark entered by the headteacher during term-end compilation. Not populated automatically — set via API.'

**Indexes:**
- `idx_overall_summaries_tenant` (tenant_id)
- `idx_overall_summaries_school` (school_id)
- `idx_overall_summaries_student_term` (student_id, academic_term_id)
- `idx_overall_summaries_term` (academic_term_id)

**Triggers:**
- `trg_student_term_overall_summaries_updated_at` BEFORE UPDATE ON student_term_overall_summaries FOR EACH ROW EXECUTE FUNCTION fn_set_updated_at()

### Student Cohort Position Summaries
- `id`: UUID (Primary Key) - Default: gen_random_uuid()
- `tenant_id`: UUID NOT NULL
- `school_id`: UUID NOT NULL
- `student_id`: UUID NOT NULL
- `class_id`: UUID NOT NULL
- `academic_term_id`: UUID NOT NULL
- `student_score`: NUMERIC(5,2)
- `class_rank`: INT
- `class_headcount`: INT
- `class_average`: NUMERIC(5,2)
- `class_percentile`: NUMERIC(5,2)
- `grade_rank`: INT
- `grade_headcount`: INT
- `grade_average`: NUMERIC(5,2)
- `grade_percentile`: NUMERIC(5,2)
- `variance_from_grade_mean`: NUMERIC(6,2)
- `last_refreshed_at`: TIMESTAMPTZ NOT NULL DEFAULT NOW()
- `created_at`: TIMESTAMPTZ NOT NULL DEFAULT NOW()
- `updated_at`: TIMESTAMPTZ NOT NULL DEFAULT NOW()

**Constraints:**
- `uq_cohort_position_student_class_term` UNIQUE (student_id, class_id, academic_term_id)
- `fk_cohort_position_tenant_student` FOREIGN KEY (tenant_id, student_id) REFERENCES cbc_students(tenant_id, id) ON DELETE CASCADE
- `fk_cohort_position_tenant_class` FOREIGN KEY (tenant_id, class_id) REFERENCES cbc_classes(tenant_id, id) ON DELETE CASCADE
- `fk_cohort_position_tenant_term` FOREIGN KEY (tenant_id, school_id, academic_term_id) REFERENCES academic_terms(tenant_id, school_id, id) ON DELETE CASCADE
- `fk_cohort_position_tenant_school` FOREIGN KEY (tenant_id, school_id) REFERENCES cbc_schools(tenant_id, id) ON DELETE CASCADE

**Comments:**
- TABLE student_cohort_position_summaries: 'Periodic batch-computed class and grade rankings per student per term. NEVER updated incrementally — the batch function fn_compute_cohort_positions_for_term() must be called on a schedule or on-demand via the refresh API. Data sources: student_term_overall_summaries.overall_mean_percentage and cbc_student_enrollments (status = ACTIVE).'
- COLUMN student_cohort_position_summaries.class_rank: 'Rank of the student within their class, ordered by student_score descending. 1 = highest score. NULL when student has no score.'
- COLUMN student_cohort_position_summaries.class_percentile: 'Percentile within the class, computed as (class_headcount - class_rank) / class_headcount * 100. A student ranked 4th out of 32 has percentile = (32-4)/32*100 = 87.50.'
- COLUMN student_cohort_position_summaries.variance_from_grade_mean: 'Difference between the student''s score and the grade average. Positive = above average, Negative = below average.'

**Indexes:**
- `idx_cohort_position_tenant` (tenant_id)
- `idx_cohort_position_school` (school_id)
- `idx_cohort_position_student_term` (student_id, academic_term_id)
- `idx_cohort_position_term` (academic_term_id)
- `idx_cohort_position_class_term` (class_id, academic_term_id)

**Triggers:**
- `trg_student_cohort_position_summaries_updated_at` BEFORE UPDATE ON student_cohort_position_summaries FOR EACH ROW EXECUTE FUNCTION fn_set_updated_at()

### Student Subject Strand Summaries
- `id`: UUID (Primary Key) - Default: gen_random_uuid()
- `tenant_id`: UUID NOT NULL
- `school_id`: UUID NOT NULL
- `student_id`: UUID NOT NULL
- `academic_term_id`: UUID NOT NULL
- `learning_area_id`: UUID NOT NULL
- `strand_id`: UUID NOT NULL
- `sub_strand_id`: UUID NOT NULL
- `mastery_percentage`: NUMERIC(5,2)
- `exceeding_count`: INT NOT NULL DEFAULT 0
- `meeting_count`: INT NOT NULL DEFAULT 0
- `approaching_count`: INT NOT NULL DEFAULT 0
- `below_count`: INT NOT NULL DEFAULT 0
- `mapped_performance_level`: VARCHAR(5)
- `requires_remediation`: BOOLEAN NOT NULL DEFAULT false
- `has_data`: BOOLEAN NOT NULL DEFAULT false
- `last_refreshed_at`: TIMESTAMPTZ NOT NULL DEFAULT NOW()
- `created_at`: TIMESTAMPTZ NOT NULL DEFAULT NOW()
- `updated_at`: TIMESTAMPTZ NOT NULL DEFAULT NOW()

**Constraints:**
- `uq_student_term_sub_strand` UNIQUE (student_id, academic_term_id, sub_strand_id)
- `fk_strand_summaries_tenant_student` FOREIGN KEY (tenant_id, student_id) REFERENCES cbc_students(tenant_id, id) ON DELETE CASCADE
- `fk_strand_summaries_tenant_term` FOREIGN KEY (tenant_id, school_id, academic_term_id) REFERENCES academic_terms(tenant_id, school_id, id) ON DELETE CASCADE
- `fk_strand_summaries_tenant_school` FOREIGN KEY (tenant_id, school_id) REFERENCES cbc_schools(tenant_id, id) ON DELETE CASCADE
- `fk_strand_summaries_learning_area` FOREIGN KEY (tenant_id, learning_area_id) REFERENCES cbc_learning_areas(tenant_id, id) ON DELETE CASCADE
- `fk_strand_summaries_strand` FOREIGN KEY (tenant_id, strand_id) REFERENCES cbc_strands(tenant_id, id) ON DELETE CASCADE
- `fk_strand_summaries_sub_strand` FOREIGN KEY (tenant_id, sub_strand_id) REFERENCES cbc_sub_strands(tenant_id, id) ON DELETE CASCADE

**Comments:**
- TABLE student_subject_strand_summaries: 'Rubric-only sub-strand-level summary per student and term. Counts performance indicators awarded at each CBC level (EE/ME/AE/BE) and computes mastery as the percentage at ME or above. Only populated for RUBRIC sessions — for quantitative subjects has_data stays false.'
- COLUMN student_subject_strand_summaries.mastery_percentage: 'Percentage of performance indicators for this sub-strand that were awarded Meeting Expectations or above: (exceeding_count + meeting_count) / total_indicators * 100. NULL when no data exists.'
- COLUMN student_subject_strand_summaries.requires_remediation: 'TRUE when any indicator was awarded Below Expectations or when mastery_percentage is below 50%. Suggests the student needs targeted intervention on this sub-strand.'
- COLUMN student_subject_strand_summaries.has_data: 'TRUE when at least one rubric outcome grade exists for this student+term+sub_strand. Consumers should check this flag before displaying mastery metrics to avoid misleading 0% displays.'

**Indexes:**
- `idx_strand_summaries_tenant` (tenant_id)
- `idx_strand_summaries_school` (school_id)
- `idx_strand_summaries_student_term` (student_id, academic_term_id)
- `idx_strand_summaries_term_sub_strand` (academic_term_id, sub_strand_id)

**Triggers:**
- `trg_student_subject_strand_summaries_updated_at` BEFORE UPDATE ON student_subject_strand_summaries FOR EACH ROW EXECUTE FUNCTION fn_set_updated_at()

### Student Performance Projections
- `id`: UUID (Primary Key) - Default: gen_random_uuid()
- `tenant_id`: UUID NOT NULL
- `school_id`: UUID NOT NULL
- `student_id`: UUID NOT NULL
- `academic_term_id`: UUID NOT NULL
- `learning_area_id`: UUID NULL (NULL = overall cross-subject projection)
- `momentum_score`: NUMERIC(6,2)
- `projected_score`: NUMERIC(5,2)
- `projected_performance_level`: VARCHAR(5)
- `target_gap_points`: NUMERIC(6,2)
- `risk_level`: VARCHAR(10) NOT NULL DEFAULT 'Unknown'
- `confidence_percentage`: NUMERIC(5,2)
- `last_refreshed_at`: TIMESTAMPTZ NOT NULL DEFAULT NOW()
- `created_at`: TIMESTAMPTZ NOT NULL DEFAULT NOW()
- `updated_at`: TIMESTAMPTZ NOT NULL DEFAULT NOW()

**Constraints:**
- `uq_student_term_learning_area_proj` UNIQUE (student_id, academic_term_id, learning_area_id)
- `fk_projections_tenant_student` FOREIGN KEY (tenant_id, student_id) REFERENCES cbc_students(tenant_id, id) ON DELETE CASCADE
- `fk_projections_tenant_term` FOREIGN KEY (tenant_id, school_id, academic_term_id) REFERENCES academic_terms(tenant_id, school_id, id) ON DELETE CASCADE
- `fk_projections_tenant_school` FOREIGN KEY (tenant_id, school_id) REFERENCES cbc_schools(tenant_id, id) ON DELETE CASCADE
- `fk_projections_learning_area` FOREIGN KEY (tenant_id, learning_area_id) REFERENCES cbc_learning_areas(tenant_id, id) ON DELETE CASCADE

**Comments:**
- TABLE student_performance_projections: 'Periodic batch-computed performance projections per student per term per learning area. Reads the last 2-3 terms of assessment data to compute a momentum trend line and project the next term''s score. NEVER updated incrementally — call fn_compute_performance_projections_for_term() periodically (once per term close).'
- COLUMN student_performance_projections.momentum_score: 'Linear regression slope over the last 2-3 terms of assessment data. Positive = improving trend, Negative = declining trend. NULL when fewer than 2 terms of history exist.'
- COLUMN student_performance_projections.projected_score: 'Predicted score for the next term, calculated as the last term''s score plus the momentum_score. NULL when insufficient history exists.'
- COLUMN student_performance_projections.risk_level: 'Risk classification: Low (confident projection, close to or above ME threshold), Medium (moderate gap or uncertainty), High (significant gap or very low confidence). Defaults to Unknown initially.'
- COLUMN student_performance_projections.confidence_percentage: 'Confidence in the projection based on the number of historical terms available. Capped low (< 30%) when fewer than 2 terms exist.'

**Indexes:**
- `idx_projections_tenant` (tenant_id)
- `idx_projections_school` (school_id)
- `idx_projections_student_term` (student_id, academic_term_id)
- `idx_projections_term` (academic_term_id)
- `idx_projections_learning_area` (learning_area_id)

**Triggers:**
- `trg_student_performance_projections_updated_at` BEFORE UPDATE ON student_performance_projections FOR EACH ROW EXECUTE FUNCTION fn_set_updated_at()

### Student Behavior Term Summaries
- `id`: UUID (Primary Key) - Default: gen_random_uuid()
- `tenant_id`: UUID NOT NULL
- `school_id`: UUID NOT NULL
- `student_id`: UUID NOT NULL
- `academic_term_id`: UUID NOT NULL
- `total_incidents`: INT NOT NULL DEFAULT 0
- `urgent_count`: INT NOT NULL DEFAULT 0
- `commendations_count`: INT NOT NULL DEFAULT 0
- `disciplinary_count`: INT NOT NULL DEFAULT 0
- `pending_review_count`: INT NOT NULL DEFAULT 0
- `resolved_count`: INT NOT NULL DEFAULT 0
- `primary_category_id`: UUID
- `last_refreshed_at`: TIMESTAMPTZ NOT NULL DEFAULT NOW()
- `created_at`: TIMESTAMPTZ NOT NULL DEFAULT NOW()
- `updated_at`: TIMESTAMPTZ NOT NULL DEFAULT NOW()

**Constraints:**
- `uq_student_behavior_term` UNIQUE (student_id, academic_term_id)
- `fk_behavior_summaries_tenant_student` FOREIGN KEY (tenant_id, student_id) REFERENCES cbc_students(tenant_id, id) ON DELETE CASCADE
- `fk_behavior_summaries_tenant_term` FOREIGN KEY (tenant_id, school_id, academic_term_id) REFERENCES academic_terms(tenant_id, school_id, id) ON DELETE CASCADE
- `fk_behavior_summaries_tenant_school` FOREIGN KEY (tenant_id, school_id) REFERENCES cbc_schools(tenant_id, id) ON DELETE CASCADE
- `fk_behavior_summaries_tenant_category` FOREIGN KEY (tenant_id, primary_category_id) REFERENCES behavior_categories(tenant_id, id) ON DELETE SET NULL (primary_category_id)

**Comments:**
- TABLE student_behavior_term_summaries: 'Incremental materialised summary of behavior notes per student per term. Only counts APPROVED and INCLUDED_IN_REPORT notes in the main totals (total_incidents, urgent_count, commendations_count, disciplinary_count, primary_category_id). pending_review_count counts all PENDING_REVIEW notes for admin visibility. Refreshed via trigger on behavior_notes insert/update.'
- COLUMN student_behavior_term_summaries.total_incidents: 'Total count of APPROVED + INCLUDED_IN_REPORT behavior notes for this student+term. Excludes PENDING_REVIEW and REJECTED.'
- COLUMN student_behavior_term_summaries.commendations_count: 'Count of APPROVED + INCLUDED_IN_REPORT notes whose category has category_type = COMMENDATION.'
- COLUMN student_behavior_term_summaries.disciplinary_count: 'Count of APPROVED + INCLUDED_IN_REPORT notes whose category has category_type = DISCIPLINARY.'
- COLUMN student_behavior_term_summaries.pending_review_count: 'Count of PENDING_REVIEW notes for this student+term. Provides admin visibility into backlog.'
- COLUMN student_behavior_term_summaries.primary_category_id: 'The behavior category with the highest count among APPROVED + INCLUDED_IN_REPORT notes for this student+term. Ties are resolved by the most recent note''s category. NULL when no applicable notes exist.'

**Indexes:**
- `idx_behavior_summaries_tenant` (tenant_id)
- `idx_behavior_summaries_school` (school_id)
- `idx_behavior_summaries_student_term` (student_id, academic_term_id)
- `idx_behavior_summaries_term` (academic_term_id)

**Triggers:**
- `trg_student_behavior_term_summaries_updated_at` BEFORE UPDATE ON student_behavior_term_summaries FOR EACH ROW EXECUTE FUNCTION fn_set_updated_at()

### Teacher Subject Performance Summaries
- `id`: UUID (Primary Key) - Default: gen_random_uuid()
- `tenant_id`: UUID NOT NULL
- `school_id`: UUID NOT NULL
- `user_id`: UUID NOT NULL
- `learning_area_id`: UUID NOT NULL
- `class_id`: UUID NOT NULL
- `academic_term_id`: UUID NOT NULL
- `subject_mean_score`: NUMERIC(5,2)
- `cohort_mastery_rate`: NUMERIC(5,2)
- `student_growth_rate`: NUMERIC(6,2)
- `assessment_timeliness_index`: NUMERIC(5,2)
- `strand_coverage_rate`: NUMERIC(5,2)
- `last_refreshed_at`: TIMESTAMPTZ NOT NULL DEFAULT NOW()
- `created_at`: TIMESTAMPTZ NOT NULL DEFAULT NOW()
- `updated_at`: TIMESTAMPTZ NOT NULL DEFAULT NOW()

**Constraints:**
- `uq_teacher_subject_class_term` UNIQUE (user_id, learning_area_id, class_id, academic_term_id)
- `fk_teacher_perf_summaries_tenant_school` FOREIGN KEY (tenant_id, school_id) REFERENCES cbc_schools(tenant_id, id) ON DELETE CASCADE
- `fk_teacher_perf_summaries_tenant_user` FOREIGN KEY (tenant_id, user_id) REFERENCES users(tenant_id, id) ON DELETE CASCADE
- `fk_teacher_perf_summaries_learning_area` FOREIGN KEY (tenant_id, learning_area_id) REFERENCES cbc_learning_areas(tenant_id, id) ON DELETE CASCADE
- `fk_teacher_perf_summaries_class` FOREIGN KEY (tenant_id, class_id) REFERENCES cbc_classes(tenant_id, id) ON DELETE CASCADE
- `fk_teacher_perf_summaries_term` FOREIGN KEY (tenant_id, school_id, academic_term_id) REFERENCES academic_terms(tenant_id, school_id, id) ON DELETE CASCADE

**Comments:**
- TABLE teacher_subject_performance_summaries: 'Periodic batch-computed teacher effectiveness summary per subject per class per term. Teacher attribution uses the current cbc_class_teachers SUBJECT_TEACHER row — no historical tracking, so mid-term reassignments are folded in. Approximation — flag in UI.'
- COLUMN teacher_subject_performance_summaries.subject_mean_score: 'Average of student_term_subject_summaries.average_percentage across all students enrolled in this class+learning_area+term. NULL when no assessment data exists.'
- COLUMN teacher_subject_performance_summaries.cohort_mastery_rate: 'Percentage of enrolled students whose mapped_performance_level is ME or EE in this class+learning_area+term. NULL when no data exists.'
- COLUMN teacher_subject_performance_summaries.student_growth_rate: 'Average percentage-point change (current term vs prior term) for students who were enrolled in both terms in this learning area. NULL for Term 1 or when insufficient matched students exist.'
- COLUMN teacher_subject_performance_summaries.assessment_timeliness_index: 'Percentage of PUBLISHED assessment sessions for this class+learning_area+term that were published on or before their scheduled_date.'
- COLUMN teacher_subject_performance_summaries.strand_coverage_rate: 'Percentage of cbc_strands for this learning_area that have at least one PUBLISHED RUBRIC assessment session in this term.'

**Indexes:**
- `idx_teacher_perf_summaries_tenant` (tenant_id)
- `idx_teacher_perf_summaries_school` (school_id)
- `idx_teacher_perf_summaries_user` (user_id)
- `idx_teacher_perf_summaries_term` (academic_term_id)
- `idx_teacher_perf_summaries_class_term` (class_id, academic_term_id)

**Triggers:**
- `trg_teacher_subject_performance_summaries_updated_at` BEFORE UPDATE ON teacher_subject_performance_summaries FOR EACH ROW EXECUTE FUNCTION fn_set_updated_at()

### Teacher Delivery Summaries
- `id`: UUID (Primary Key) - Default: gen_random_uuid()
- `tenant_id`: UUID NOT NULL
- `school_id`: UUID NOT NULL
- `user_id`: UUID NOT NULL
- `academic_term_id`: UUID NOT NULL
- `total_assigned_slots`: INT NOT NULL DEFAULT 0
- `marked_slots`: INT NOT NULL DEFAULT 0
- `missed_slots`: INT NOT NULL DEFAULT 0
- `sessions_created`: INT NOT NULL DEFAULT 0
- `sessions_approved`: INT NOT NULL DEFAULT 0
- `on_time_submission_rate`: NUMERIC(5,2) NULL
- `last_refreshed_at`: TIMESTAMPTZ NOT NULL DEFAULT NOW()
- `created_at`: TIMESTAMPTZ NOT NULL DEFAULT NOW()
- `updated_at`: TIMESTAMPTZ NOT NULL DEFAULT NOW()

**Constraints:**
- `uq_teacher_delivery_term` UNIQUE (user_id, academic_term_id)
- `fk_teacher_delivery_tenant_school` FOREIGN KEY (tenant_id, school_id) REFERENCES cbc_schools(tenant_id, id) ON DELETE CASCADE
- `fk_teacher_delivery_tenant_user` FOREIGN KEY (tenant_id, user_id) REFERENCES users(tenant_id, id) ON DELETE CASCADE
- `fk_teacher_delivery_term` FOREIGN KEY (tenant_id, school_id, academic_term_id) REFERENCES academic_terms(tenant_id, school_id, id) ON DELETE CASCADE

**Comments:**
- TABLE teacher_delivery_summaries: 'Incrementally updated summary of teacher lesson delivery metrics per term. Grain: (user_id, academic_term_id). Updated via triggers on attendance_records INSERT and cbc_attendance_sessions status changes.'
- COLUMN teacher_delivery_summaries.total_assigned_slots: 'Total number of timetable slot occurrences assigned to this teacher during the term (slot × weeks where the slot day_of_week falls within the term date range).'
- COLUMN teacher_delivery_summaries.marked_slots: 'Number of assigned slot occurrences where attendance_records exist (attendance was taken).'
- COLUMN teacher_delivery_summaries.missed_slots: 'Number of assigned slot occurrences where the lesson was marked SKIPPED (lesson did not take place).'
- COLUMN teacher_delivery_summaries.sessions_created: 'Number of cbc_attendance_sessions records associated with this teacher''s slots in the term (any session status).'
- COLUMN teacher_delivery_summaries.sessions_approved: 'Number of sessions where status = SUBMITTED (attendance was formally recorded and approved).'
- COLUMN teacher_delivery_summaries.on_time_submission_rate: 'Percentage of assigned slots that were either marked or skipped: (marked_slots + missed_slots) / total_assigned_slots * 100.'

**Indexes:**
- `idx_teacher_delivery_tenant` (tenant_id)
- `idx_teacher_delivery_school` (school_id)
- `idx_teacher_delivery_user` (user_id)
- `idx_teacher_delivery_term` (academic_term_id)

**Triggers:**
- `trg_teacher_delivery_summaries_updated_at` BEFORE UPDATE ON teacher_delivery_summaries FOR EACH ROW EXECUTE FUNCTION fn_set_updated_at()

### Teacher Workload Summaries
- `id`: UUID (Primary Key) - Default: gen_random_uuid()
- `tenant_id`: UUID NOT NULL
- `school_id`: UUID NOT NULL
- `user_id`: UUID NOT NULL
- `academic_year_id`: UUID NOT NULL
- `total_assigned_periods`: INT NOT NULL DEFAULT 0
- `unique_subjects`: INT NOT NULL DEFAULT 0
- `classes_taught`: INT NOT NULL DEFAULT 0
- `utilization_percentage`: NUMERIC(5,2) NULL
- `is_overcapacity`: BOOLEAN NOT NULL DEFAULT FALSE
- `last_refreshed_at`: TIMESTAMPTZ NOT NULL DEFAULT NOW()
- `created_at`: TIMESTAMPTZ NOT NULL DEFAULT NOW()
- `updated_at`: TIMESTAMPTZ NOT NULL DEFAULT NOW()

**Constraints:**
- `uq_teacher_workload_year` UNIQUE (user_id, academic_year_id)
- `fk_teacher_workload_tenant_school` FOREIGN KEY (tenant_id, school_id) REFERENCES cbc_schools(tenant_id, id) ON DELETE CASCADE
- `fk_teacher_workload_tenant_user` FOREIGN KEY (tenant_id, user_id) REFERENCES users(tenant_id, id) ON DELETE CASCADE
- `fk_teacher_workload_year` FOREIGN KEY (tenant_id, academic_year_id) REFERENCES academic_years(tenant_id, id) ON DELETE CASCADE

**Comments:**
- TABLE teacher_workload_summaries: 'Batch-computed summary of teacher workload metrics per academic year. Grain: (user_id, academic_year_id). Recomputes on-demand — reassignments via timetable slots or cbc_class_teachers are infrequent.'
- COLUMN teacher_workload_summaries.total_assigned_periods: 'Number of weekly timetable slots assigned to this teacher. Represents the per-week instructional load (e.g. 24 periods/week).'
- COLUMN teacher_workload_summaries.unique_subjects: 'Count of distinct learning areas (subjects) assigned to this teacher.'
- COLUMN teacher_workload_summaries.classes_taught: 'Count of distinct classes this teacher has timetable assignments for.'
- COLUMN teacher_workload_summaries.utilization_percentage: 'Percentage of the school''s total weekly instructional periods that this teacher covers: total_assigned_periods / total_school_periods * 100. NULL when no timetable structures exist for the school.'
- COLUMN teacher_workload_summaries.is_overcapacity: 'TRUE when the teacher''s assigned periods exceed the school''s average teacher capacity per week (currently flagged when utilization exceeds 100% of total school periods / active teachers).'

**Indexes:**
- `idx_teacher_workload_tenant` (tenant_id)
- `idx_teacher_workload_school` (school_id)
- `idx_teacher_workload_user` (user_id)
- `idx_teacher_workload_year` (academic_year_id)

**Triggers:**
- `trg_teacher_workload_summaries_updated_at` BEFORE UPDATE ON teacher_workload_summaries FOR EACH ROW EXECUTE FUNCTION fn_set_updated_at()

### Class Learning Area Term Summaries
- `id`: UUID (Primary Key) - Default: gen_random_uuid()
- `tenant_id`: UUID NOT NULL
- `school_id`: UUID NOT NULL
- `class_id`: UUID NOT NULL
- `learning_area_id`: UUID NOT NULL
- `academic_term_id`: UUID NOT NULL
- `academic_year_id`: UUID NOT NULL
- `students_included`: INT NOT NULL DEFAULT 0
- `periods_total`: INT NOT NULL DEFAULT 0
- `periods_present`: INT NOT NULL DEFAULT 0
- `periods_absent`: INT NOT NULL DEFAULT 0
- `periods_late`: INT NOT NULL DEFAULT 0
- `periods_excused`: INT NOT NULL DEFAULT 0
- `attendance_percentage`: NUMERIC(5,2) NOT NULL DEFAULT 0.00
- `last_refreshed_at`: TIMESTAMPTZ NOT NULL DEFAULT NOW()
- `created_at`: TIMESTAMPTZ NOT NULL DEFAULT NOW()
- `updated_at`: TIMESTAMPTZ NOT NULL DEFAULT NOW()

**Constraints:**
- `uq_class_learning_area_term` UNIQUE (class_id, learning_area_id, academic_term_id)
- `fk_class_la_term_tenant_school` FOREIGN KEY (tenant_id, school_id) REFERENCES cbc_schools(tenant_id, id) ON DELETE CASCADE
- `fk_class_la_term_tenant_class` FOREIGN KEY (tenant_id, class_id) REFERENCES cbc_classes(tenant_id, id) ON DELETE CASCADE
- `fk_class_la_term_learning_area` FOREIGN KEY (tenant_id, learning_area_id) REFERENCES cbc_learning_areas(tenant_id, id) ON DELETE CASCADE
- `fk_class_la_term_tenant_term` FOREIGN KEY (tenant_id, school_id, academic_term_id) REFERENCES academic_terms(tenant_id, school_id, id) ON DELETE CASCADE

**Comments:**
- TABLE class_learning_area_term_summaries: 'Class-grain rollup of attendance_term_summaries per (class, learning_area, academic_term). Refreshed exclusively by the Asynq task attendance:refresh_class_learning_area_term_summary (enqueued from inside handleAttendanceTermRefresh so it runs after the student-grain rollup is current). No cascading DB triggers fire on attendance_term_summaries writes — this keeps the hot attendance marking path predictable.'
- COLUMN class_learning_area_term_summaries.students_included: 'Count of distinct students whose attendance_term_summaries rows contributed to this (class, learning_area, term) aggregate. May be less than the total enrolled count for the class.'
- COLUMN class_learning_area_term_summaries.attendance_percentage: 'Calculated as (periods_present / periods_total) * 100, stored as a decimal with two fractional digits (e.g. 92.50).'

**Indexes:**
- `idx_class_la_term_tenant` (tenant_id)
- `idx_class_la_term_school` (school_id)
- `idx_class_la_term_class_term` (class_id, academic_term_id)
- `idx_class_la_term_area_term` (learning_area_id, academic_term_id)
- `idx_class_la_term_year` (academic_year_id)

**Triggers:**
- `trg_class_learning_area_term_summaries_updated_at` BEFORE UPDATE ON class_learning_area_term_summaries FOR EACH ROW EXECUTE FUNCTION fn_set_updated_at()

### Class Term Attendance Summaries
- `id`: UUID (Primary Key) - Default: gen_random_uuid()
- `tenant_id`: UUID NOT NULL
- `school_id`: UUID NOT NULL
- `class_id`: UUID NOT NULL
- `academic_term_id`: UUID NOT NULL
- `academic_year_id`: UUID NOT NULL
- `days_in_term`: INT NOT NULL DEFAULT 0
- `total_enrolled_avg`: NUMERIC(6,2) NULL
- `present_count`: INT NOT NULL DEFAULT 0
- `absent_count`: INT NOT NULL DEFAULT 0
- `late_count`: INT NOT NULL DEFAULT 0
- `excused_count`: INT NOT NULL DEFAULT 0
- `term_attendance_rate`: NUMERIC(5,2) NOT NULL DEFAULT 0.00
- `last_refreshed_at`: TIMESTAMPTZ NOT NULL DEFAULT NOW()
- `created_at`: TIMESTAMPTZ NOT NULL DEFAULT NOW()
- `updated_at`: TIMESTAMPTZ NOT NULL DEFAULT NOW()

**Constraints:**
- `uq_class_term_attendance` UNIQUE (class_id, academic_term_id)
- `fk_class_term_att_tenant_school` FOREIGN KEY (tenant_id, school_id) REFERENCES cbc_schools(tenant_id, id) ON DELETE CASCADE
- `fk_class_term_att_tenant_class` FOREIGN KEY (tenant_id, class_id) REFERENCES cbc_classes(tenant_id, id) ON DELETE CASCADE
- `fk_class_term_att_tenant_term` FOREIGN KEY (tenant_id, school_id, academic_term_id) REFERENCES academic_terms(tenant_id, school_id, id) ON DELETE CASCADE

**Comments:**
- TABLE class_term_attendance_summaries: 'Term-grain rollup of class_daily_attendance_summaries per (class, academic_term). Refreshed exclusively by the Asynq task attendance:refresh_class_term_summary (enqueued from inside handleClassDailyRefresh so it runs after the daily-grain rollup is current). No cascading DB triggers fire on class_daily_attendance_summaries writes.'
- COLUMN class_term_attendance_summaries.days_in_term: 'Count of class_daily_attendance_summaries rows rolled up for this class/term — i.e. school days with recorded attendance, NOT calendar days or total days in the term date range.'
- COLUMN class_term_attendance_summaries.total_enrolled_avg: 'Average of class_daily_attendance_summaries.total_enrolled across the term. Inherits the documented workaround from the daily table (total_enrolled derived from distinct students with attendance_records that day).'
- COLUMN class_term_attendance_summaries.term_attendance_rate: 'Calculated as (present_count / (present_count + absent_count + late_count + excused_count)) * 100, matching the formula used in class_daily_attendance_summaries.daily_attendance_rate.'

**Indexes:**
- `idx_class_term_att_tenant` (tenant_id)
- `idx_class_term_att_school_term` (school_id, academic_term_id)
- `idx_class_term_att_class` (class_id)
- `idx_class_term_att_term` (academic_term_id)
- `idx_class_term_att_year` (academic_year_id)

**Triggers:**
- `trg_class_term_attendance_summaries_updated_at` BEFORE UPDATE ON class_term_attendance_summaries FOR EACH ROW EXECUTE FUNCTION fn_set_updated_at()

### Member Counts (global aggregate — migration 000063_member_counts)
- `id`: UUID (Primary Key) - Default: gen_random_uuid()
- `students`: INTEGER NOT NULL DEFAULT 0
- `admins`: INTEGER NOT NULL DEFAULT 0
- `nurses`: INTEGER NOT NULL DEFAULT 0
- `teachers`: INTEGER NOT NULL DEFAULT 0
- `parents`: INTEGER NOT NULL DEFAULT 0
- `finance`: INTEGER NOT NULL DEFAULT 0
- `updated_at`: TIMESTAMPTZ NOT NULL DEFAULT now()
- `created_at`: TIMESTAMPTZ NOT NULL DEFAULT now()

**Comments:**
- Single-row global aggregate (000063 deletes all rows and inserts one zeroed row at migration time). Kept in sync by `trg_student_count` (AFTER INSERT/UPDATE/DELETE on cbc_students) and `trg_membership_count` (AFTER INSERT/UPDATE/DELETE on memberships). `SYSTEM_ADMIN` memberships are not counted.

## Relationships Summary

### One-to-Many Relationships
- Tenants → Users (via tenant_id)
- Tenants → Sessions
- Tenants → Schools (cbc_schools)
- Tenants → Students (cbc_students)
- Tenants → Parents (cbc_parents)
- Tenants → Member Active School
- Tenants → School Member Counts
- Schools → Users (memberships)
- Schools → Students
- Schools → CBC Classes
- Schools → CBC Streams
- Schools → Academic Years
- Schools → Academic Terms
- Schools → CBC Learning Areas
- Schools → Behavior Categories
- Schools → Fee Categories / Fee Templates / Invoices
- Schools → Grading Scale Profiles
- Schools → Timetable Structures
- Academic Years → Academic Terms
- Academic Years → Timetable Structures
- Academic Terms → Enrollments, Fee Templates, Invoices, Assessment Sessions, Attendance Records (composite via (tenant_id, school_id, academic_term_id))
- Learning Areas → Strands → Sub-Strands → Performance Indicators
- Classes → Class Teachers, Timetable Slots, Attendance Sessions, Enrollments
- Students → Enrollments
- Students → Medical Incidents
- Students → Health Profiles
- Students → Parent Relationships (junction)
- Students → Attendance Records
- Students → Behavior Notes
- Students → Import Job Staging (via staging_row_id)
- Students → Assessment Scores / Outcome Grades
- Students → Term/Strand/Overall/Cohort/Projection/Behavior summaries
- Import Jobs → Import Job Chunks
- Import Jobs → Import Job Failures
- Import Jobs → Import Job Staging
- Import Jobs → Invitations (import_job_id)
- Users → Invitations (invited_by)
- Users → Sessions
- Users → Medical Incidents (logged_by)
- Users → Student Health Profiles (logged_by)
- Users → Payments (recorded_by)
- Users → Behavior Notes (authored_by / reviewed_by)
- Users → Assessment Sessions (created_by / submitted_by / approved_by)
- Users → Member Active School
- Users → Class Teachers, Timetable Slots (teacher_id)
- Users → Teacher Performance/Delivery/Workload summaries
- Invoices → Invoice Items
- Invoices → Payments
- Grading Scale Profiles → Grading Scale Ranges
- Assessment Sessions → Student Assessment Scores
- Assessment Sessions → Student Assessment Outcome Grades

### Many-to-Many Relationships
- Students ↔ Parents (via cbc_student_parents junction table)
- Classes ↔ Teachers (via cbc_class_teachers)
- Classes ↔ Learning Areas ↔ Teachers (via cbc_timetable_slots)
- Students ↔ Learning Areas (via assessment sessions and attendance)

### Composite Keys (for tenant scoping)
- Tenants (id)
- Users (tenant_id, id)
- Schools (tenant_id, id)
- Students (tenant_id, id)
- Memberships (tenant_id, user_id, school_id)
- Invitations (tenant_id, school_id, email) - unique pending
- Academic Years (tenant_id, school_id, id)
- Academic Terms (tenant_id, school_id, id)
- CBC Classes (tenant_id, id)
- CBC Streams (tenant_id, id)
- CBC Learning Areas (tenant_id, id)
- CBC Strands (tenant_id, id)
- CBC Sub-Strands (tenant_id, id)
- Performance Indicators (tenant_id, id)
- Timetable Structures (tenant_id, id)
- CBC Timetable Slots (tenant_id, id)
- CBC Attendance Sessions (tenant_id, id)
- Fee Categories (tenant_id, id)
- Invoices (tenant_id, id)
- Import Jobs (tenant_id, id)
- Import Job Staging (tenant_id, id)
- Grading Scale Profiles (tenant_id, id)
- Assessment Sessions (tenant_id, id)
- Memberships (user_id, school_id) - unique per user/school

## Relationships with ON DELETE Actions

- **CASCADE**:
  - Tenants → Users, Schools, Memberships, Sessions, School Member Counts
  - Schools → Students, CBC Classes, CBC Streams, Academic Years, Academic Terms, Learning Areas, Memberships, Timetable Structures, Behavior Categories, Fee Categories, Grading Scale Profiles, Import Jobs, Invitations, Member Active School, Attendance Sessions
  - Academic Years → Academic Terms, Timetable Structures
  - Academic Terms → Enrollments, Fee Templates, Invoices, Attendance Records, Assessment Sessions, Attendance Term Summaries (composite FKs)
  - Students → Medical Incidents, Health Profiles, Enrollments, Attendance Records, Behavior Notes, Assessment Scores, Outcome Grades, all student-grain summaries
  - Learning Areas → Strands, Class Teachers (SET NULL on learning_area_id instead)
  - Strands → Sub-Strands
  - Sub-Strands → Performance Indicators
  - Classes → Class Teachers, Timetable Slots, Attendance Sessions, Enrollments (class_id SET NULL)
  - Invoices → Invoice Items, Payments
  - Grading Scale Profiles → Grading Scale Ranges
  - Assessment Sessions → Student Assessment Scores, Student Assessment Outcome Grades
  - Import Jobs → Import Job Chunks, Staging, Failures
  - Timetable Structures → Timetable Slots
  - Member Active School → cascades from users, schools, and memberships

- **SET NULL**:
  - Invitations → Import Jobs (import_job_id)
  - Enrollments → Classes (class_id)
  - Invoices → Parents (parent_id)
  - Payments → Parents (parent_id)
  - Class Teachers → Learning Areas (learning_area_id)
  - Attendance Records → Attendance Sessions (attendance_session_id)
  - Behavior Notes → Reviewers (reviewed_by_id)
  - Assessment Sessions → Scale Profile / Submitted By / Approved By
  - Import Jobs → Created By
  - Students → Import Job Staging (staging_row_id)
  - Behavior Term Summaries → Primary Category (primary_category_id)

- **RESTRICT**:
  - Medical Incidents → Users (logged_by)
  - Health Profiles → Users (logged_by)
  - Payments → Users (recorded_by)
  - Behavior Notes → Categories (category_id) and Authors (authored_by_id)
  - Attendance Records → Users (marked_by)
  - CBC Streams → referenced by CBC Classes (fk_cbc_classes_stream)

## Key Constraints and Indexes

### Unique Constraints
- Tenants (slug), (stytch_org_id)
- Users (tenant_id, LOWER(email)), (tenant_id, tsc_number), (tenant_id, knec_panel_assessor_id)
- Sessions (token_hash)
- Academic Years (tenant_id, school_id, id), EXCL constraint on start/end dates, one current year per school
- Academic Terms (tenant_id, school_id, id), EXCL constraint on start/end dates, one current term per school, unique term_number per year
- CBC Classes (school_id, academic_year_id, grade_level, stream_id), (tenant_id, id)
- CBC Streams (tenant_id, school_id, name)
- Memberships (user_id, school_id)
- Invitations (tenant_id, school_id, email) - active status only, (school_id, email) - pending only
- Import Jobs (tenant_id, idempotency_key), (school_id) - one active at a time
- Import Job Chunks (job_id, chunk_index)
- Import Job Staging (job_id, row_number)
- Students (upi_number), (knec_assessment_number), (school_id, staging_row_id)
- Student Parents (tenant_id, student_id, parent_id) - primary key
- One primary parent per student (is_primary = true)
- Learning Areas (tenant_id, school_id, code, grade_level)
- Fee Categories (tenant_id, school_id, name)
- Fee Templates (academic_term_id, grade_level, fee_category_id)
- Invoices (student_id, academic_term_id)
- Payments (reference_code) WHERE NOT NULL
- Timetable Structures: GiST exclusion per (school_id, academic_year_id, day_of_week, fn_timerange)
- Timetable Slots: (academic_year_id, structure_id, class_id), (academic_year_id, structure_id, teacher_id), (academic_year_id, structure_id, room_identifier)
- Class Teachers (class_id, user_id, learning_area_id); one PRIMARY_CLASS_TEACHER per class
- Behavior Categories (tenant_id, school_id, name)
- Attendance Records (student_id, timetable_slot_id, date)
- Attendance Sessions (school_id, timetable_slot_id, date)
- Assessment Weight Configs (grade_level, assessment_type_code, target_exam, effective_from)
- Grading Scale Ranges: (profile_id, performance_level) + GiST numrange no-overlap
- Assessment Sessions (session_id, student_id) via scores; outcome grades (session_id, student_id, performance_indicator_id)
- School Member Counts PRIMARY KEY (tenant_id, school_id)
- Member Active School PRIMARY KEY (user_id)
- Class Daily Attendance Summaries (class_id, date)
- Student Term Subject Summaries (student_id, academic_term_id, learning_area_id)
- Student Term Overall Summaries (student_id, academic_term_id)
- Student Cohort Position Summaries (student_id, class_id, academic_term_id)
- Student Subject Strand Summaries (student_id, academic_term_id, sub_strand_id)
- Student Performance Projections (student_id, academic_term_id, learning_area_id)
- Student Behavior Term Summaries (student_id, academic_term_id)
- Teacher Subject Performance Summaries (user_id, learning_area_id, class_id, academic_term_id)
- Teacher Delivery Summaries (user_id, academic_term_id)
- Teacher Workload Summaries (user_id, academic_year_id)
- Class Learning Area Term Summaries (class_id, learning_area_id, academic_term_id)
- Class Term Attendance Summaries (class_id, academic_term_id)

### Foreign Keys
- Users → Tenants (ON DELETE CASCADE)
- Sessions → Users, Tenants (COMPOSITE)
- Schools → Tenants (ON DELETE CASCADE)
- Academic Years → Tenants, Schools, Users (created_by/updated_by, COMPOSITE)
- Academic Terms → Academic Years (COMPOSITE), Schools, Users
- Classes → Academic Years (COMPOSITE), Schools, Streams
- Memberships → Tenants, Users, Schools
- Import Jobs → Tenants, Schools, Users (created_by)
- Import Job Chunks → Import Jobs
- Import Job Staging → Import Jobs
- Invitations → Tenants, Schools, Users (invited_by), Import Jobs
- Students → Tenants, Schools, Import Job Staging
- Student Parents → Tenants, Students, Parents
- Enrollments → Students, Schools, Academic Terms, Classes, Academic Years
- Medical Incidents → Tenants, Students, Users (logged_by)
- Health Profiles → Tenants, Students, Users (logged_by)
- Fee Categories → Schools
- Fee Templates → Schools, Academic Terms, Fee Categories
- Invoices → Students, Schools, Academic Terms, Parents
- Invoice Items → Invoices, Fee Categories
- Payments → Invoices, Parents, Users (recorded_by)
- Learning Areas → Schools; Strands → Learning Areas; Sub-Strands → Strands; Performance Indicators → Sub-Strands
- Class Teachers → Users, Classes, Learning Areas
- Timetable Structures → Schools, Academic Years
- Timetable Slots → Schools, Classes, Users (teacher), Learning Areas, Structures, Academic Years
- Attendance Sessions → Timetable Slots, Schools
- Attendance Records → Students, Timetable Slots, Attendance Sessions, Academic Terms, Users (marked_by)
- Behavior Categories → Schools
- Behavior Notes → Students, Timetable Slots, Behavior Categories, Users (authored_by/reviewed_by)
- Attendance Term Summaries → Students, Academic Terms, Learning Areas, Academic Years
- Grading Scale Profiles → Schools; Grading Scale Ranges → Profiles
- Assessment Sessions → Schools, Classes, Academic Terms, Learning Areas, Grading Scale Profiles, Users (submitted/approved/created)
- Assessment Scores → Assessment Sessions, Students
- Outcome Grades → Assessment Sessions, Students, Performance Indicators
- All materialised summaries → Students, Classes, Schools, Academic Terms/Years, Learning Areas (tenant-scoped composite FKs)
- Member Active School → Users, Schools, Memberships
- School Member Counts → Schools

### Check Constraints
- `academic_years`: start_date < end_date
- `academic_terms`: term_number between 1-3, start_date < end_date
- `cbc_students`: gender IN ('M', 'F')
- `fee_templates`: amount >= 0
- `import_jobs`: role required for staff/parent invites
- `invitation_status`: default 'pending'
- `cbc_enrollment_status`: default 'ACTIVE'
- `cbc_class_teachers`: PRIMARY_CLASS_TEACHER must have no learning_area; SUBJECT_TEACHER must have one
- `timetable_structures`: end_time > start_time; day_of_week between 1 and 7
- `cbc_attendance_sessions`: status IN ('SUBMITTED', 'SKIPPED')
- `invoices`: amount_due >= 0, amount_paid >= 0
- `invoice_items`: amount >= 0
- `payments`: amount > 0
- `assessment_weight_configs`: weight_percent > 0 AND <= 100
- `grading_scale_ranges`: min/max percentage 0-100; max > min
- `assessment_sessions`: QUANTITATIVE requires max_points + scale profile; RUBRIC forbids both
- `student_assessment_scores`: raw_score >= 0 AND <= session max_points (when non-null)

### Exclusion Constraints (GiST)
- `EXCL_academic_years_no_overlap`: Prevents overlapping date ranges for academic years per school
- `EXCL_academic_terms_no_overlap`: Prevents overlapping date ranges for academic terms per school
- `excl_timetable_structure_overlap`: Prevents overlapping time blocks per school per academic year per day (uses fn_timerange)
- `excl_profile_range_no_overlap`: Prevents overlapping percentage bands within a grading scale profile (uses numrange)

## Row-Level Security (RLS)

Nearly all tenant-scoped data tables have RLS enabled with a `tenant_isolation_policy` (FOR ALL, USING tenant_id = fn_current_tenant_id(), WITH CHECK same). The application must run `SET LOCAL app.current_tenant_id = '<uuid>'` at the start of each request; without it, all RLS-protected queries return zero rows (safe by default). `fn_resolve_session()` and `fn_pending_invitation_by_email()` are SECURITY DEFINER so session resolution and invite acceptance can run before the tenant is known.

RLS-enabled tables include: users, sessions lookups (via SECURITY DEFINER), tenants-related, cbc_schools, cbc_streams, cbc_classes, cbc_class_teachers, cbc_learning_areas, cbc_strands, cbc_sub_strands, performance_indicators, academic_years, academic_terms, memberships, member_active_school, import_jobs, import_job_staging, invitations, cbc_parents, cbc_students, cbc_student_parents, cbc_student_enrollments, medical_incidents, student_health_profiles, fee_categories, fee_templates, invoices, invoice_items, payments, timetable_structures, cbc_timetable_slots, cbc_attendance_sessions, attendance_records, behavior_categories, behavior_notes, attendance_term_summaries, grading_scale_profiles, grading_scale_ranges, assessment_sessions, student_assessment_scores, student_assessment_outcome_grades, school_member_counts, and all materialised summary tables.

## Notes
1. All tables use UUID primary keys for scalability and security
2. Strong tenant isolation throughout the schema (composite FKs + RLS)
3. Extensive use of composite keys for tenant scoping
4. Comprehensive audit columns (created_at, updated_at) on most tables
5. GiST exclusion constraints prevent overlapping date ranges for academic years/terms, timetable blocks, and grading scale bands
6. Composite foreign keys enforce tenant-scoped referential integrity
7. ON DELETE actions are carefully chosen to preserve historical data where needed
8. Enums are used extensively for data validation and consistency
9. JSONB fields are used for flexible metadata storage (import jobs metadata, staging raw_data)
10. Triggers automatically manage updated_at timestamps across the schema
11. Custom functions support complex business logic (term validation, time range calculations, invoice payment status sync, school member count sync)
12. Careful attention to indexes for query performance
13. Materialised summary/rollup tables (attendance, assessment, behavior, teacher metrics) are refreshed via batch functions, Asynq background jobs, or publish triggers — they are caches, not sources of truth
14. `member_counts` (000063) is a legacy single-row global aggregate; `school_member_counts` is the per-school replacement
15. Assessment sessions support two grading modes (QUANTITATIVE with max_points + grading scale, and RUBRIC with per-indicator outcome grades), with write-once protections on max_points and grading scale ranges
16. KNEC weighting formulas (assessment_weight_configs) are nationally mandated, seeded with official KPSEA/KJSEA figures, and applied for final exam terms (G6/G9/G12)