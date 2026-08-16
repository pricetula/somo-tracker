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

## Relationships Summary

### One-to-Many Relationships
- Tenants → Users (via tenant_id)
- Tenants → Users (sessions)
- Tenants → Schools
- Tenants → Students
- Tenants → Parents
- Schools → Users (memberships)
- Schools → Students
- Schools → CBC Classes
- Schools → CBC Streams
- Schools → Academic Years
- Schools → Academic Terms
- Academic Terms → Academic Years
- Students → Enrollments
- Students → Medical Incidents
- Students → Health Profiles
- Students → Parent Relationships
- Students → Import Job Staging (via staging_row_id)
- Import Jobs → Import Job Chunks
- Import Jobs → Import Job Failures
- Users → Invitations (invited_by)
- Users → Sessions
- Users → Medical Incidents (logged_by)
- Users → Student Health Profiles (logged_by)

### Many-to-Many Relationships
- Students ↔ Parents (via cbc_student_parents junction table)
- Schools ↔ Students (via enrollments)
- Schools ↔ Classes (via academic_year_id)
- Classes ↔ Students (via enrollments)

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
- Memberships (user_id, school_id) - unique per user/school

## Relationships with ON DELETE Actions

- **CASCADE**: 
  - Tenants → Users, Schools, Memberships
  - Schools → Students, CBC Classes, CBC Streams, Academic Terms
  - Academic Years → Academic Terms
  - Academic Terms → Academic Years (reference)
  - Students → Medical Incidents, Health Profiles
  - Import Jobs → Import Job Chunks, Staging, Failures

- **SET NULL**:
  - Sessions → Users (token_hash)
  - Invitations → Import Jobs
  - Enrollments → Classes (class_id)

- **RESTRICT**:
  - Medical Incidents → Users (logged_by)
  - Health Profiles → Users (logged_by)

## Key Constraints and Indexes

### Unique Constraints
- Tenants (slug), (stytch_org_id)
- Users (tenant_id, LOWER(email)), (tenant_id, tsc_number), (tenant_id, knec_panel_assessor_id)
- Sessions (token_hash)
- Academic Years (tenant_id, school_id, id), EXCL constraint on start/end dates
- Academic Terms (tenant_id, school_id, id), EXCL constraint on start/end dates
- CBC Classes (school_id, academic_year_id, grade_level, stream_id), (tenant_id, id)
- Memberships (user_id, school_id)
- Invitations (tenant_id, school_id, email) - pending status only
- Invitations (tenant_id, idempotency_key) - unique when not null
- Import Jobs (tenant_id, idempotency_key), (school_id) - one active at a time
- Import Job Chunks (job_id, chunk_index)
- Import Job Staging (job_id, row_number)
- Students (upi_number), (knec_assessment_number), (school_id, staging_row_id)
- Student Parents (tenant_id, student_id, parent_id) - primary key
- One primary parent per student (is_primary = true)

### Foreign Keys
- Users → Tenants (ON DELETE CASCADE)
- Sessions → Users, Tenants (COMPOSITE)
- Schools → Tenants (ON DELETE CASCADE)
- Academic Years → Tenants, Schools (COMPOSITE)
- Academic Terms → Academic Years (COMPOSITE)
- Classes → Academic Years (COMPOSITE)
- Classes → Schools (COMPOSITE)
- Memberships → Tenants, Users, Schools
- Import Jobs → Tenants, Schools, Import Jobs (self-reference)
- Import Job Chunks → Import Jobs
- Import Job Staging → Import Jobs, Schools, Tenants
- Invitations → Tenants, Schools, Import Jobs
- Students → Tenants, Schools
- Student Parents → Tenants, Students, Parents
- Enrollments → Students, Schools, Academic Terms, Classes
- Medical Incidents → Tenants, Students
- Health Profiles → Tenants, Students

### Check Constraints
- `users`: is_active boolean default true
- `academic_years`: start_date < end_date
- `academic_terms`: term_number between 1-3, start_date < end_date
- `cbc_students`: gender IN ('M', 'F')
- `fee_templates`: amount >= 0
- `import_jobs`: role required for staff/parent invites
- `invitation_status`: default 'pending'
- `cbc_enrollment_status`: default 'ACTIVE'

### Exclusion Constraints (GiST)
- `EXCL_academic_years_no_overlap`: Prevents overlapping date ranges for academic years per school
- `EXCL_academic_terms_no_overlap`: Prevents overlapping date ranges for academic terms per school

## Notes
1. All tables use UUID primary keys for scalability and security
2. Strong tenant isolation throughout the schema
3. Extensive use of composite keys for tenant scoping
4. Comprehensive audit columns (created_at, updated_at) on most tables
5. GiST exclusion constraints prevent overlapping date ranges for academic years/terms
6. Composite foreign keys enforce tenant-scoped referential integrity
7. ON DELETE actions are carefully chosen to preserve historical data where needed
8. Enums are used extensively for data validation and consistency
9. JSONB fields are used for flexible metadata storage
10. Triggers automatically manage updated_at timestamps across the schema
11. Custom functions support complex business logic (term validation, time range calculations)
12. Careful attention to indexes for query performance