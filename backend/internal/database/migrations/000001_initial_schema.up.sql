-- Migration: 000001_initial_schema
-- SomoTracker — Kenya CBC/CBE academic platform (CBC-only, v5)
-- Drops all generic education system abstractions.
-- Rebuilds as a purpose-built, single-system CBC schema.

BEGIN;

-- ============================================================================
-- EXTENSIONS
-- ============================================================================

CREATE EXTENSION IF NOT EXISTS btree_gist;

-- ============================================================================
-- FUNCTIONS
-- ============================================================================

CREATE OR REPLACE FUNCTION fn_set_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Maps day_of_week (1=Mon…7=Sun) onto base week 2024-01-01 so
-- GiST exclusion constraints only conflict within the same day.
CREATE OR REPLACE FUNCTION fn_timerange(day_of_week INT, start_time TIME, end_time TIME)
RETURNS tsrange AS $$
    SELECT tsrange(
        ('2024-01-01'::DATE + (day_of_week - 1)) + start_time,
        ('2024-01-01'::DATE + (day_of_week - 1)) + end_time,
        '[)'
    );
$$ LANGUAGE sql IMMUTABLE;

-- ============================================================================
-- ENUMS
-- ============================================================================

DO $$ BEGIN
    CREATE TYPE user_role AS ENUM ('SYSTEM_ADMIN', 'SCHOOL_ADMIN', 'TEACHER', 'NURSE', 'FINANCE', 'PARENT');
EXCEPTION WHEN duplicate_object THEN NULL;
END $$;



DO $$ BEGIN
    CREATE TYPE attendance_status AS ENUM ('PRESENT', 'ABSENT', 'LATE', 'EXCUSED');
EXCEPTION WHEN duplicate_object THEN NULL;
END $$;

DO $$ BEGIN
    CREATE TYPE invitation_status AS ENUM ('pending', 'accepted', 'expired', 'revoked', 'invite_failed');
EXCEPTION WHEN duplicate_object THEN NULL;
END $$;

DO $$ BEGIN
    CREATE TYPE gender_type AS ENUM ('M', 'F');
EXCEPTION WHEN duplicate_object THEN NULL;
END $$;

DO $$ BEGIN
    CREATE TYPE cbc_enrollment_status AS ENUM (
        'ACTIVE',            -- Currently enrolled and attending
        'SUSPENDED',         -- Temporarily removed from active learning
        'TRANSFERRED',       -- Moved to another school; record retained
        'COMPLETED_CYCLE'    -- Successfully completed a CBC education cycle
    );
EXCEPTION WHEN duplicate_object THEN NULL;
END $$;

DO $$ BEGIN
    CREATE TYPE cbc_grade_level AS ENUM (
        'PP1','PP2',
        'G1','G2','G3',
        'G4','G5','G6',
        'G7','G8','G9',
        'G10','G11','G12'
    );
EXCEPTION WHEN duplicate_object THEN NULL;
END $$;

DO $$ BEGIN
    CREATE TYPE cbc_education_level AS ENUM (
        'Early_Years',
        'Upper_Primary',
        'Junior_Secondary',
        'Senior_School'
    );
EXCEPTION WHEN duplicate_object THEN NULL;
END $$;

DO $$ BEGIN
    CREATE TYPE cbc_school_type AS ENUM (
        'Public',
        'Private',
        'Special_Needs_School'
    );
EXCEPTION WHEN duplicate_object THEN NULL;
END $$;

DO $$ BEGIN
    CREATE TYPE cbc_learning_pathway AS ENUM (
        'Age_Based',
        'Stage_Based'
    );
EXCEPTION WHEN duplicate_object THEN NULL;
END $$;

DO $$ BEGIN
    CREATE TYPE cbc_assessment_type AS ENUM (
        'Formative_Classroom',
        'KNEC_Written_Assessment',
        'KNEC_SBA_Project',
        'National_KPSEA',
        'National_KJSEA',
        'National_KSSEA'
    );
EXCEPTION WHEN duplicate_object THEN NULL;
END $$;

DO $$ BEGIN
    CREATE TYPE knec_target_exam AS ENUM (
        'KPSEA',
        'KJSEA',
        'KSSEA',
        'None'
    );
EXCEPTION WHEN duplicate_object THEN NULL;
END $$;

DO $$ BEGIN
    CREATE TYPE cbc_rubric_level AS ENUM (
        'EE','ME','AE','BE'
    );
EXCEPTION WHEN duplicate_object THEN NULL;
END $$;

DO $$ BEGIN
    CREATE TYPE cbc_rubric_level_with_sub_levels AS ENUM (
        'EE','ME','AE','BE',
        'EE1','EE2',
        'ME1','ME2',
        'AE1','AE2',
        'BE1','BE2'
    );
EXCEPTION WHEN duplicate_object THEN NULL;
END $$;

DO $$ BEGIN
    CREATE TYPE lrr_score_type AS ENUM (
        'Numeric_Raw',
        'Rubric_Direct'
    );
EXCEPTION WHEN duplicate_object THEN NULL;
END $$;

DO $$ BEGIN
    CREATE TYPE portfolio_evidence_type AS ENUM (
        'Physical_File_Reference',
        'Digital_Artifact_URL',
        'Video_Recording',
        'Audio_Log',
        'Observation_Checklist'
    );
EXCEPTION WHEN duplicate_object THEN NULL;
END $$;

DO $$ BEGIN
    CREATE TYPE knec_sync_status AS ENUM (
        'Pending',
        'Synced',
        'Failed'
    );
EXCEPTION WHEN duplicate_object THEN NULL;
END $$;

DO $$ BEGIN
    CREATE TYPE invoice_payment_status AS ENUM (
        'UNPAID',    -- No payments recorded yet
        'PARTIAL',   -- Some payment made, balance remains
        'PAID',      -- Fully settled
        'WAIVED'     -- Debt forgiven by finance admin
    );
EXCEPTION WHEN duplicate_object THEN NULL;
END $$;

-- ============================================================================
-- LAYER 1 — PLATFORM INFRASTRUCTURE
-- ============================================================================

-- ---------------------------------------------------------------------------
-- TENANTS
-- ---------------------------------------------------------------------------

CREATE TABLE IF NOT EXISTS tenants (
    id            UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    name          VARCHAR(255) NOT NULL,
    slug          VARCHAR(255) NOT NULL UNIQUE,
    stytch_org_id VARCHAR(255) NOT NULL UNIQUE,
    created_at    TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_tenants_slug          ON tenants (slug);
CREATE INDEX IF NOT EXISTS idx_tenants_stytch_org_id ON tenants (stytch_org_id);

-- ---------------------------------------------------------------------------
-- USERS
-- ---------------------------------------------------------------------------

CREATE TABLE IF NOT EXISTS users (
    id                  UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    email               VARCHAR(255) NOT NULL,
    tenant_id           UUID         NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    full_name           VARCHAR(255) NOT NULL DEFAULT '',
    is_active           BOOLEAN      NOT NULL DEFAULT TRUE,
    external_auth_id    VARCHAR(255) UNIQUE,
    tsc_number          VARCHAR(15)  NULL,
    knec_panel_assessor_id VARCHAR(20) NULL,
    created_at          TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ  NOT NULL DEFAULT NOW(),

    CONSTRAINT uq_users_tenant UNIQUE (tenant_id, id)
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_users_email    ON users (email);
CREATE INDEX        IF NOT EXISTS idx_users_tenant   ON users (tenant_id);
CREATE UNIQUE INDEX IF NOT EXISTS idx_users_tsc_number
    ON users (tsc_number) WHERE tsc_number IS NOT NULL;
CREATE UNIQUE INDEX IF NOT EXISTS idx_users_knec_panel_assessor_id
    ON users (knec_panel_assessor_id) WHERE knec_panel_assessor_id IS NOT NULL;

DROP TRIGGER IF EXISTS trg_users_updated_at ON users;
CREATE TRIGGER trg_users_updated_at
    BEFORE UPDATE ON users
    FOR EACH ROW EXECUTE FUNCTION fn_set_updated_at();

COMMENT ON COLUMN users.tsc_number IS
    'Teachers Service Commission registration number. Populated only for users
     with the TEACHER role. Required for TSC portal access and official deployment.';

COMMENT ON COLUMN users.knec_panel_assessor_id IS
    'Assigned ONLY to teachers formally appointed to KNEC national exam panels
     (KPSEA, KJSEA, KSSEA invigilation or marking). NOT required for classroom
     SBA delivery — all SBA uploads use the school knec_school_code, not teacher IDs.';

-- ---------------------------------------------------------------------------
-- SESSIONS
-- ---------------------------------------------------------------------------

CREATE TABLE IF NOT EXISTS sessions (
    id                   UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    token                VARCHAR(128) NOT NULL UNIQUE,
    user_id              UUID         NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    tenant_id            UUID         NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    stytch_member_id     VARCHAR(255) NOT NULL,
    stytch_org_id        VARCHAR(255) NOT NULL,
    stytch_session_token VARCHAR(512) NOT NULL DEFAULT '',
    device_fingerprint   VARCHAR(128) NOT NULL DEFAULT '',
    expires_at           TIMESTAMPTZ  NOT NULL,
    created_at           TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_sessions_token                ON sessions (token);
CREATE INDEX IF NOT EXISTS idx_sessions_user_id              ON sessions (user_id);
CREATE INDEX IF NOT EXISTS idx_sessions_tenant_id            ON sessions (tenant_id);
CREATE INDEX IF NOT EXISTS idx_sessions_stytch_session_token ON sessions (stytch_session_token);

-- ============================================================================
-- LAYER 2 — CORE CBC ACTORS
-- ============================================================================

-- ---------------------------------------------------------------------------
-- CBC SCHOOLS
-- ---------------------------------------------------------------------------

CREATE TABLE IF NOT EXISTS cbc_schools (
    id                      UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id               UUID         NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    name                    VARCHAR(255) NOT NULL,
    knec_school_code        VARCHAR(15)  NULL,
    nemis_institution_code  VARCHAR(20)  NULL,
    county                  VARCHAR(50)  NOT NULL,
    sub_county              VARCHAR(50)  NOT NULL,
    ward                    VARCHAR(50)  NULL,
    school_type             cbc_school_type  NOT NULL,
    is_active               BOOLEAN      NOT NULL DEFAULT true,
    created_at              TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at              TIMESTAMPTZ  NOT NULL DEFAULT NOW(),

    CONSTRAINT uq_cbc_schools_tenant UNIQUE (tenant_id, id)
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_cbc_schools_knec_code
    ON cbc_schools (knec_school_code) WHERE knec_school_code IS NOT NULL;
CREATE UNIQUE INDEX IF NOT EXISTS idx_cbc_schools_nemis_code
    ON cbc_schools (nemis_institution_code) WHERE nemis_institution_code IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_cbc_schools_tenant_id ON cbc_schools (tenant_id);

DROP TRIGGER IF EXISTS trg_cbc_schools_updated_at ON cbc_schools;
CREATE TRIGGER trg_cbc_schools_updated_at
    BEFORE UPDATE ON cbc_schools
    FOR EACH ROW EXECUTE FUNCTION fn_set_updated_at();

COMMENT ON COLUMN cbc_schools.knec_school_code IS
    'Official KNEC center code (8–10 digit numeric string). Used as the school
     login username on the CBA portal at cba.knec.ac.ke. Required before any
     SBA score uploads can be submitted to KNEC.';

COMMENT ON COLUMN cbc_schools.nemis_institution_code IS
    'National Education Management Information System institution code.
     Assigned by the Ministry of Education. Used for MoE reporting and
     NEMIS data synchronisation.';

-- ============================================================================
-- LAYER 3 — ACADEMIC CALENDAR (placed here for FK ordering)
-- ============================================================================

-- ---------------------------------------------------------------------------
-- ACADEMIC YEARS
-- ---------------------------------------------------------------------------

CREATE TABLE IF NOT EXISTS academic_years (
    id         UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id  UUID        NOT NULL,
    school_id  UUID        NOT NULL,
    name       VARCHAR(50) NOT NULL,
    start_date DATE        NOT NULL,
    end_date   DATE        NOT NULL,
    is_current BOOLEAN     NOT NULL DEFAULT false,

    CONSTRAINT chk_academic_year_dates CHECK (end_date > start_date),
    CONSTRAINT uq_academic_years_tenant UNIQUE (tenant_id, id),
    CONSTRAINT fk_academic_years_tenant_school FOREIGN KEY (tenant_id, school_id) REFERENCES cbc_schools(tenant_id, id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_academic_years_tenant_id ON academic_years (tenant_id);
CREATE INDEX IF NOT EXISTS idx_academic_years_school_id ON academic_years (school_id);

CREATE UNIQUE INDEX IF NOT EXISTS idx_one_current_year_per_school
    ON academic_years (school_id) WHERE is_current = true;

-- ---------------------------------------------------------------------------
-- ACADEMIC TERMS
-- ---------------------------------------------------------------------------

CREATE TABLE IF NOT EXISTS academic_terms (
    id               UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id        UUID         NOT NULL,
    school_id        UUID         NOT NULL,
    academic_year_id UUID         NOT NULL,
    name             VARCHAR(100) NOT NULL,
    term_number      SMALLINT     NOT NULL,
    start_date       DATE         NOT NULL,
    end_date         DATE         NOT NULL,
    is_current       BOOLEAN      NOT NULL DEFAULT false,
    is_final         BOOLEAN      NOT NULL DEFAULT false,

    CONSTRAINT chk_academic_term_dates CHECK (end_date > start_date),
    CONSTRAINT chk_academic_term_number CHECK (term_number BETWEEN 1 AND 3),
    CONSTRAINT uq_academic_terms_tenant UNIQUE (tenant_id, id),
    CONSTRAINT uq_academic_terms_tenant_school UNIQUE (tenant_id, school_id, id),
    CONSTRAINT uq_academic_year_term_number UNIQUE (academic_year_id, term_number), 
    CONSTRAINT fk_academic_terms_tenant_school FOREIGN KEY (tenant_id, school_id) REFERENCES cbc_schools(tenant_id, id) ON DELETE CASCADE,
    CONSTRAINT fk_academic_terms_tenant_year FOREIGN KEY (tenant_id, academic_year_id) REFERENCES academic_years(tenant_id, id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_academic_terms_tenant_id ON academic_terms (tenant_id);
CREATE INDEX IF NOT EXISTS idx_academic_terms_school_id ON academic_years (school_id);
CREATE INDEX IF NOT EXISTS idx_academic_terms_year_id   ON academic_terms (academic_year_id);

CREATE UNIQUE INDEX IF NOT EXISTS idx_one_current_term_per_year
    ON academic_terms (academic_year_id) WHERE is_current = true;

COMMENT ON COLUMN academic_terms.term_number IS
    'Kenya CBC operates a 3-term academic year. term_number enforces this:
     1 = Term 1, 2 = Term 2, 3 = Term 3.';

-- ============================================================================
-- LAYER 2 — CORE CBC ACTORS (continued)
-- ============================================================================

-- ---------------------------------------------------------------------------
-- CBC CLASSES (replaces generic classes)
-- ---------------------------------------------------------------------------

CREATE TABLE IF NOT EXISTS cbc_classes (
    id               UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id        UUID         NOT NULL,
    school_id        UUID         NOT NULL,
    academic_year_id UUID         NOT NULL,
    name             VARCHAR(100) NOT NULL,
    grade_level      cbc_grade_level   NOT NULL,
    stream           VARCHAR(100) NOT NULL DEFAULT '',
    is_active        BOOLEAN      NOT NULL DEFAULT true,

    CONSTRAINT uq_cbc_classes_tenant UNIQUE (tenant_id, id),
    CONSTRAINT fk_cbc_classes_tenant_school FOREIGN KEY (tenant_id, school_id) REFERENCES cbc_schools(tenant_id, id) ON DELETE CASCADE,
    CONSTRAINT fk_cbc_classes_tenant_academic_year FOREIGN KEY (tenant_id, academic_year_id) REFERENCES academic_years(tenant_id, id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_cbc_classes_tenant_id        ON cbc_classes (tenant_id);
CREATE INDEX IF NOT EXISTS idx_cbc_classes_school_id        ON cbc_classes (school_id);
CREATE INDEX IF NOT EXISTS idx_cbc_classes_academic_year_id ON cbc_classes (academic_year_id);
CREATE INDEX IF NOT EXISTS idx_cbc_classes_grade_level      ON cbc_classes (grade_level);
CREATE INDEX IF NOT EXISTS idx_cbc_classes_stream           ON cbc_classes (stream);

COMMENT ON COLUMN cbc_classes.grade_level IS
    'Official KNEC grade designation. Determines which assessment instruments,
     SBA projects, and KNEC portal upload windows apply to the class. Values
     match KNEC CBA portal grade codes: PP1–PP2 (Pre-Primary), G1–G12.';

-- ---------------------------------------------------------------------------
-- MEMBERSHIPS
-- ---------------------------------------------------------------------------

CREATE TABLE IF NOT EXISTS memberships (
    id         UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id  UUID        NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    user_id    UUID        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    school_id  UUID        NOT NULL,
    role       user_role   NOT NULL,
    is_active  BOOLEAN     NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT fk_memberships_tenant_school FOREIGN KEY (tenant_id, school_id) REFERENCES cbc_schools(tenant_id, id) ON DELETE CASCADE,
    CONSTRAINT unique_user_school_membership UNIQUE (user_id, school_id)
);

CREATE INDEX IF NOT EXISTS idx_memberships_tenant_id ON memberships (tenant_id);
CREATE INDEX IF NOT EXISTS idx_memberships_user_id   ON memberships (user_id);
CREATE INDEX IF NOT EXISTS idx_memberships_school_id ON memberships (school_id);

-- ---------------------------------------------------------------------------
-- IMPORT JOBS — Bulk Staff Invitation async ingestion
-- ---------------------------------------------------------------------------

CREATE TABLE IF NOT EXISTS import_jobs (
    id                   UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id            UUID        NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    school_id            UUID        NOT NULL,
    role                 user_role   NOT NULL,
    created_by           UUID        REFERENCES users(id) ON DELETE SET NULL,
    status               TEXT        NOT NULL DEFAULT 'pending',
    total_records        INT         NOT NULL DEFAULT 0,
    processed_records    INT         NOT NULL DEFAULT 0,
    success_count        INT         NOT NULL DEFAULT 0,
    failed_count         INT         NOT NULL DEFAULT 0,
    parent_import_job_id UUID        NULL,
    created_at           TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    started_at           TIMESTAMPTZ NULL,
    completed_at         TIMESTAMPTZ NULL,

    CONSTRAINT fk_import_jobs_tenant_school FOREIGN KEY (tenant_id, school_id) REFERENCES cbc_schools(tenant_id, id) ON DELETE CASCADE,
    CONSTRAINT fk_import_jobs_parent FOREIGN KEY (parent_import_job_id) REFERENCES import_jobs(id) DEFERRABLE INITIALLY DEFERRED
);

CREATE INDEX IF NOT EXISTS idx_import_jobs_tenant_id ON import_jobs (tenant_id);
CREATE INDEX IF NOT EXISTS idx_import_jobs_school_id ON import_jobs (school_id);
CREATE INDEX IF NOT EXISTS idx_import_jobs_created_by ON import_jobs (created_by);
CREATE INDEX IF NOT EXISTS idx_import_jobs_status ON import_jobs (status);

CREATE TABLE IF NOT EXISTS import_job_failures (
    id             BIGSERIAL   PRIMARY KEY,
    import_job_id  UUID        NOT NULL REFERENCES import_jobs(id) ON DELETE CASCADE,
    raw_payload    JSONB       NOT NULL,
    error_message  TEXT        NOT NULL,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_import_job_failures_job_id ON import_job_failures (import_job_id);

-- ---------------------------------------------------------------------------
-- IMPORT JOB STAGING — Student bulk import staging rows
-- ---------------------------------------------------------------------------

CREATE TABLE IF NOT EXISTS import_job_staging (
    id         UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    job_id     UUID         NOT NULL REFERENCES import_jobs(id) ON DELETE CASCADE,
    tenant_id  UUID         NOT NULL,
    school_id  UUID         NOT NULL,
    row_number INT          NOT NULL,
    raw_data   JSONB        NOT NULL,
    created_at TIMESTAMPTZ  NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_import_job_staging_job_id ON import_job_staging (job_id);

-- ---------------------------------------------------------------------------
-- INVITATIONS
-- ---------------------------------------------------------------------------

CREATE TABLE IF NOT EXISTS invitations (
    id                  UUID              PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id           UUID              NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    school_id           UUID              NOT NULL,
    email               VARCHAR(255)      NOT NULL,
    role                user_role         NOT NULL,
    status              invitation_status NOT NULL DEFAULT 'pending',
    invited_by          UUID              REFERENCES users(id) ON DELETE SET NULL,
    token               TEXT              NOT NULL,
    expires_at          TIMESTAMPTZ       NOT NULL,
    accepted_at         TIMESTAMPTZ       NULL,
    full_name           VARCHAR(255)      NOT NULL,
    phone               VARCHAR(50)       NULL,
    registration_number VARCHAR(100)      NULL,
    stytch_member_id    VARCHAR(255)      NULL,
    import_job_id       UUID              NULL,
    error_message       TEXT              NULL,
    attempt_count       INT               NOT NULL DEFAULT 0,
    created_at          TIMESTAMPTZ       NOT NULL DEFAULT NOW(),

    CONSTRAINT fk_invitations_tenant_school FOREIGN KEY (tenant_id, school_id) REFERENCES cbc_schools(tenant_id, id) ON DELETE CASCADE,
    CONSTRAINT fk_invitations_import_job FOREIGN KEY (import_job_id) REFERENCES import_jobs(id) ON DELETE SET NULL
);

CREATE INDEX IF NOT EXISTS idx_invitations_tenant_id ON invitations (tenant_id);
CREATE INDEX IF NOT EXISTS idx_invitations_school_id ON invitations (school_id);
CREATE INDEX IF NOT EXISTS idx_invitations_email     ON invitations (email);
CREATE INDEX IF NOT EXISTS idx_invitations_status    ON invitations (status);
CREATE INDEX IF NOT EXISTS idx_invitations_import_job ON invitations (import_job_id);

CREATE UNIQUE INDEX IF NOT EXISTS uq_invitations_active_email
  ON invitations (tenant_id, school_id, email)
  WHERE status NOT IN ('expired', 'revoked');

-- ---------------------------------------------------------------------------
-- CBC PARENTS (Profile Extension linking to core platform Users)
-- ---------------------------------------------------------------------------

CREATE TABLE IF NOT EXISTS cbc_parents (
    id             UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id      UUID         NOT NULL,
    user_id        UUID         NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    phone_number   VARCHAR(20)  NOT NULL, -- Crucial for M-Pesa & SMS notifications
    is_active      BOOLEAN      NOT NULL DEFAULT true,
    created_at     TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at     TIMESTAMPTZ  NOT NULL DEFAULT NOW(),

    CONSTRAINT uq_cbc_parents_user UNIQUE (user_id),
    CONSTRAINT fk_cbc_parents_tenant_user FOREIGN KEY (tenant_id, user_id) REFERENCES users(tenant_id, id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_cbc_parents_phone ON cbc_parents (phone_number);
CREATE INDEX IF NOT EXISTS idx_cbc_parents_tenant ON cbc_parents (tenant_id);

DROP TRIGGER IF EXISTS trg_cbc_parents_updated_at ON cbc_parents;
CREATE TRIGGER trg_cbc_parents_updated_at
    BEFORE UPDATE ON cbc_parents
    FOR EACH ROW EXECUTE FUNCTION fn_set_updated_at();

COMMENT ON TABLE cbc_parents IS
    'Profile extension table for users acting as parents or guardians. Links
     directly to the platform users table to leverage Stytch B2B auth loops.';

-- ---------------------------------------------------------------------------
-- CBC STUDENTS (replaces generic students)
-- ---------------------------------------------------------------------------

CREATE TABLE IF NOT EXISTS cbc_students (
    id                      UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id               UUID         NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    school_id               UUID         NOT NULL REFERENCES cbc_schools(id) ON DELETE RESTRICT,
    full_name               VARCHAR(255) NOT NULL,
    gender                  gender_type      NOT NULL,
    date_of_birth           DATE         NULL,
    upi_number              VARCHAR(20)  NULL,
    knec_assessment_number  VARCHAR(15)  NULL,
    learning_pathway        cbc_learning_pathway  NOT NULL DEFAULT 'Age_Based',
    is_active               BOOLEAN      NOT NULL DEFAULT true,
    created_at              TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at              TIMESTAMPTZ  NOT NULL DEFAULT NOW(),

    CONSTRAINT uq_cbc_students_tenant UNIQUE (tenant_id, id),
    CONSTRAINT chk_cbc_student_gender CHECK (gender IN ('M', 'F'))
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_cbc_students_upi
    ON cbc_students (upi_number) WHERE upi_number IS NOT NULL;
CREATE UNIQUE INDEX IF NOT EXISTS idx_cbc_students_knec_assessment_number
    ON cbc_students (knec_assessment_number) WHERE knec_assessment_number IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_cbc_students_tenant_id ON cbc_students (tenant_id);
CREATE INDEX IF NOT EXISTS idx_cbc_students_school_id  ON cbc_students (school_id);

DROP TRIGGER IF EXISTS trg_cbc_students_updated_at ON cbc_students;
CREATE TRIGGER trg_cbc_students_updated_at
    BEFORE UPDATE ON cbc_students
    FOR EACH ROW EXECUTE FUNCTION fn_set_updated_at();

COMMENT ON COLUMN cbc_students.gender IS
    'CBC/NEMIS-compliant gender field. M=Male, F=Female only. KNEC registration
     and NEMIS records do not support other values.';

COMMENT ON COLUMN cbc_students.upi_number IS
    'Unique Personal Identifier assigned by NEMIS at school enrollment. Used in
     all Ministry of Education reporting and NEMIS data submissions.';

COMMENT ON COLUMN cbc_students.knec_assessment_number IS
    'Permanent CBC identifier assigned by KNEC from Grade 3 onward. Required for
     KPSEA/KJSEA/KSSEA exam registration. Parents use this number to access
     learner results at cba.knec.ac.ke/Parent.';

COMMENT ON COLUMN cbc_students.learning_pathway IS
    'Determines which KNEC assessment framework governs the learner.
     Age_Based: standard mainstream CBC curriculum (vast majority).
     Stage_Based: SNE pathway for learners with severe cognitive or multiple
     disabilities, governed by the CBAF-FL framework.';

COMMENT ON COLUMN cbc_students.school_id IS
    'Home school for this student. Set at first enrollment and updated on transfer.
     Use cbc_student_enrollments for full term-by-term history.';

-- ---------------------------------------------------------------------------
-- CBC STUDENT PARENTS JUNCTION (Many-to-Many Relationship Mapping)
-- ---------------------------------------------------------------------------

CREATE TABLE IF NOT EXISTS cbc_student_parents (
    student_id    UUID        NOT NULL REFERENCES cbc_students(id) ON DELETE CASCADE,
    parent_id     UUID        NOT NULL REFERENCES cbc_parents(id) ON DELETE CASCADE,
    relationship  VARCHAR(50) NULL, -- 'Father', 'Mother', 'Guardian'
    is_primary    BOOLEAN     NOT NULL DEFAULT true,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    PRIMARY KEY (student_id, parent_id)
);

CREATE INDEX IF NOT EXISTS idx_junction_parent ON cbc_student_parents (parent_id);

-- ---------------------------------------------------------------------------
-- CBC STUDENT ENROLLMENTS (replaces generic student_enrollments)
-- ---------------------------------------------------------------------------

CREATE TABLE IF NOT EXISTS cbc_student_enrollments (
    id               UUID                   PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id        UUID                   NOT NULL,
    student_id       UUID                   NOT NULL,
    school_id        UUID                   NOT NULL,
    academic_term_id UUID                   NOT NULL,
    class_id         UUID                   NULL,
    status           cbc_enrollment_status  NOT NULL DEFAULT 'ACTIVE',
    created_at       TIMESTAMPTZ            NOT NULL DEFAULT NOW(),

    CONSTRAINT fk_enrollments_tenant_student FOREIGN KEY (tenant_id, student_id) REFERENCES cbc_students(tenant_id, id) ON DELETE CASCADE,
    CONSTRAINT fk_enrollments_tenant_school FOREIGN KEY (tenant_id, school_id) REFERENCES cbc_schools(tenant_id, id) ON DELETE CASCADE,
    CONSTRAINT fk_enrollments_tenant_school_term FOREIGN KEY (tenant_id, school_id, academic_term_id) REFERENCES academic_terms(tenant_id, school_id, id) ON DELETE CASCADE,
    CONSTRAINT fk_enrollments_tenant_class FOREIGN KEY (tenant_id, class_id) REFERENCES cbc_classes(tenant_id, id) ON DELETE SET NULL,
    CONSTRAINT unique_student_term_enrollment UNIQUE (student_id, academic_term_id)
);

CREATE INDEX IF NOT EXISTS idx_cbc_enrollments_tenant_id  ON cbc_student_enrollments (tenant_id);
CREATE INDEX IF NOT EXISTS idx_cbc_enrollments_student_id ON cbc_student_enrollments (student_id);
CREATE INDEX IF NOT EXISTS idx_cbc_enrollments_school_id  ON cbc_student_enrollments (school_id);
CREATE INDEX IF NOT EXISTS idx_cbc_enrollments_term_id    ON cbc_student_enrollments (academic_term_id);
CREATE INDEX IF NOT EXISTS idx_cbc_enrollments_class_id   ON cbc_student_enrollments (class_id);

-- ============================================================================
-- LAYER 4 — HEALTH & FINANCIALS
-- ============================================================================

-- ---------------------------------------------------------------------------
-- MEDICAL INCIDENTS
-- ---------------------------------------------------------------------------

CREATE TABLE IF NOT EXISTS medical_incidents (
    id                 UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id          UUID        NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    student_id         UUID        NOT NULL REFERENCES cbc_students(id) ON DELETE CASCADE,
    incident_timestamp TIMESTAMPTZ NOT NULL,
    symptoms           TEXT        NOT NULL,
    action_taken       TEXT        NOT NULL,
    logged_by          UUID        NOT NULL REFERENCES users(id)
);

CREATE INDEX IF NOT EXISTS idx_medical_incidents_tenant_id  ON medical_incidents (tenant_id);
CREATE INDEX IF NOT EXISTS idx_medical_incidents_student_id ON medical_incidents (student_id);

-- ---------------------------------------------------------------------------
-- STUDENT HEALTH PROFILES
-- ---------------------------------------------------------------------------

CREATE TABLE IF NOT EXISTS student_health_profiles (
    id                     UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id              UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    student_id             UUID UNIQUE NOT NULL REFERENCES cbc_students(id) ON DELETE CASCADE,
    blood_group            VARCHAR(5),
    allergies              TEXT[],
    chronic_conditions     TEXT[],
    emergency_instructions TEXT
);

CREATE INDEX IF NOT EXISTS idx_student_health_profiles_tenant_id ON student_health_profiles (tenant_id);

-- ---------------------------------------------------------------------------
-- FEE CATEGORIES
-- ---------------------------------------------------------------------------

CREATE TABLE IF NOT EXISTS fee_categories (
    id           UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id    UUID         NOT NULL,
    school_id    UUID         NOT NULL,
    name         VARCHAR(150) NOT NULL,
    is_mandatory BOOLEAN      NOT NULL DEFAULT true,

    CONSTRAINT fk_fee_categories_tenant_school FOREIGN KEY (tenant_id, school_id) REFERENCES cbc_schools(tenant_id, id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_fee_categories_tenant    ON fee_categories (tenant_id);
CREATE INDEX IF NOT EXISTS idx_fee_categories_school_id ON fee_categories (school_id);

CREATE TABLE IF NOT EXISTS fee_templates (
    id                UUID          PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id         UUID          NOT NULL,
    school_id         UUID          NOT NULL,
    academic_term_id  UUID          NOT NULL,
    grade_level       cbc_grade_level    NOT NULL,
    fee_category_id   UUID          NOT NULL REFERENCES fee_categories(id) ON DELETE CASCADE,
    amount            NUMERIC(12,2) NOT NULL CHECK (amount >= 0),

    CONSTRAINT fk_fee_templates_tenant_school FOREIGN KEY (tenant_id, school_id) REFERENCES cbc_schools(tenant_id, id) ON DELETE CASCADE,
    CONSTRAINT fk_fee_templates_tenant_term FOREIGN KEY (tenant_id, school_id, academic_term_id) REFERENCES academic_terms(tenant_id, school_id, id) ON DELETE CASCADE,
    CONSTRAINT unique_fee_template_rule UNIQUE (academic_term_id, grade_level, fee_category_id)
);

CREATE INDEX IF NOT EXISTS idx_fee_templates_tenant      ON fee_templates (tenant_id);
CREATE INDEX IF NOT EXISTS idx_fee_templates_school_term ON fee_templates (school_id, academic_term_id);
CREATE INDEX IF NOT EXISTS idx_fee_templates_grade_level ON fee_templates (grade_level);

-- ---------------------------------------------------------------------------
-- INVOICES
-- ---------------------------------------------------------------------------

CREATE TABLE IF NOT EXISTS invoices (
    id               UUID                   PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id        UUID                   NOT NULL,
    student_id       UUID                   NOT NULL,
    school_id        UUID                   NOT NULL,
    academic_term_id UUID                   NOT NULL,
    parent_id        UUID                   NULL REFERENCES cbc_parents(id) ON DELETE SET NULL,
    invoice_label    VARCHAR(255)           NULL,
    payment_status   invoice_payment_status NOT NULL DEFAULT 'UNPAID',
    amount_due       NUMERIC(12,2)          NOT NULL DEFAULT 0 CHECK (amount_due >= 0),
    amount_paid      NUMERIC(12,2)          NOT NULL DEFAULT 0 CHECK (amount_paid >= 0),
    created_at       TIMESTAMPTZ            NOT NULL DEFAULT NOW(),

    CONSTRAINT uq_invoices_tenant UNIQUE (tenant_id, id),
    CONSTRAINT fk_invoices_tenant_student FOREIGN KEY (tenant_id, student_id) REFERENCES cbc_students(tenant_id, id) ON DELETE CASCADE,
    CONSTRAINT fk_invoices_tenant_school FOREIGN KEY (tenant_id, school_id) REFERENCES cbc_schools(tenant_id, id) ON DELETE CASCADE,
    CONSTRAINT fk_invoices_tenant_term FOREIGN KEY (tenant_id, school_id, academic_term_id) REFERENCES academic_terms(tenant_id, school_id, id) ON DELETE CASCADE,
    CONSTRAINT unique_invoice_per_student_term UNIQUE (student_id, academic_term_id)
);

CREATE INDEX IF NOT EXISTS idx_invoices_tenant         ON invoices (tenant_id);
CREATE INDEX IF NOT EXISTS idx_invoices_student_term   ON invoices (student_id, academic_term_id);
CREATE INDEX IF NOT EXISTS idx_invoices_parent         ON invoices (parent_id);
CREATE INDEX IF NOT EXISTS idx_invoices_payment_status ON invoices (tenant_id, payment_status);

COMMENT ON COLUMN invoices.payment_status IS
    'Denormalised for fast lookups. Kept in sync by trg_sync_invoice_payment_status
     trigger on payments. WAIVED is set only by application logic — the trigger
     never overwrites a WAIVED status.';
COMMENT ON COLUMN invoices.amount_due IS
    'Sum of all invoice_items.amount for this invoice. Set by the application
     when the invoice is finalised. Not updated automatically.';
COMMENT ON COLUMN invoices.amount_paid IS
    'Running total of confirmed payments. Updated automatically by
     trg_sync_invoice_payment_status on every insert/delete on payments.';

-- ---------------------------------------------------------------------------
-- INVOICE ITEMS
-- ---------------------------------------------------------------------------

CREATE TABLE IF NOT EXISTS invoice_items (
    id              UUID          PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id       UUID          NOT NULL,
    invoice_id      UUID          NOT NULL,
    fee_category_id UUID          NOT NULL REFERENCES fee_categories(id) ON DELETE CASCADE,
    description     VARCHAR(255)  NULL,
    amount          NUMERIC(12,2) NOT NULL CHECK (amount >= 0),
    created_at      TIMESTAMPTZ   NOT NULL DEFAULT NOW(),

    CONSTRAINT fk_invoice_items_tenant_invoice FOREIGN KEY (tenant_id, invoice_id) REFERENCES invoices(tenant_id, id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_invoice_items_tenant       ON invoice_items (tenant_id);
CREATE INDEX IF NOT EXISTS idx_invoice_items_invoice_id   ON invoice_items (invoice_id);
CREATE INDEX IF NOT EXISTS idx_invoice_items_fee_category ON invoice_items (fee_category_id);

-- ---------------------------------------------------------------------------
-- PAYMENTS
-- ---------------------------------------------------------------------------

CREATE TABLE IF NOT EXISTS payments (
    id             UUID          PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id      UUID          NOT NULL,
    invoice_id     UUID          NOT NULL,
    amount         NUMERIC(12,2) NOT NULL CHECK (amount > 0),
    parent_id      UUID          NULL REFERENCES cbc_parents(id) ON DELETE SET NULL,
    payment_method VARCHAR(50)   NULL,
    reference_code VARCHAR(100)  NULL,
    recorded_by    UUID          NOT NULL REFERENCES users(id),
    created_at     TIMESTAMPTZ   NOT NULL DEFAULT NOW(),

    CONSTRAINT fk_payments_tenant_invoice FOREIGN KEY (tenant_id, invoice_id) REFERENCES invoices(tenant_id, id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_payments_tenant     ON payments (tenant_id);
CREATE INDEX IF NOT EXISTS idx_payments_invoice_id ON payments (invoice_id);
CREATE INDEX IF NOT EXISTS idx_payments_parent     ON payments (parent_id);

-- ============================================================
-- TRIGGER: Sync invoice payment_status and amount_paid
-- ============================================================

CREATE OR REPLACE FUNCTION fn_sync_invoice_payment_status()
RETURNS TRIGGER AS $$
BEGIN
    WITH affected_invoices AS (
        SELECT DISTINCT invoice_id FROM inserted_rows
        UNION
        SELECT DISTINCT invoice_id FROM deleted_rows
    )
    UPDATE invoices i
    SET
        amount_paid    = COALESCE(p.total_paid, 0),
        payment_status = CASE
            WHEN i.payment_status = 'WAIVED'              THEN 'WAIVED'
            WHEN COALESCE(p.total_paid, 0) = 0            THEN 'UNPAID'
            WHEN COALESCE(p.total_paid, 0) >= i.amount_due THEN 'PAID'
            ELSE 'PARTIAL'
        END
    FROM affected_invoices ai
    LEFT JOIN (
        SELECT invoice_id, SUM(amount) AS total_paid
        FROM payments
        GROUP BY invoice_id
    ) p ON p.invoice_id = ai.invoice_id
    WHERE i.id = ai.invoice_id;

    RETURN NULL;
END;
$$ LANGUAGE plpgsql;

-- Fires on INSERT
CREATE TRIGGER trg_sync_invoice_payment_status_insert
    AFTER INSERT ON payments
    REFERENCING NEW TABLE AS inserted_rows
    FOR EACH STATEMENT
    EXECUTE FUNCTION fn_sync_invoice_payment_status();

-- Fires on DELETE
CREATE TRIGGER trg_sync_invoice_payment_status_delete
    AFTER DELETE ON payments
    REFERENCING OLD TABLE AS deleted_rows
    FOR EACH STATEMENT
    EXECUTE FUNCTION fn_sync_invoice_payment_status();

-- Fires on UPDATE
CREATE TRIGGER trg_sync_invoice_payment_status_update
    AFTER UPDATE ON payments
    REFERENCING NEW TABLE AS inserted_rows OLD TABLE AS deleted_rows
    FOR EACH STATEMENT
    EXECUTE FUNCTION fn_sync_invoice_payment_status();

-- ============================================================================
-- LAYER 5 — CBC CURRICULUM STRUCTURE
-- ============================================================================

CREATE TABLE IF NOT EXISTS cbc_learning_areas (
    id               UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id        UUID         NOT NULL,
    school_id        UUID         NOT NULL,
    name             VARCHAR(150) NOT NULL,
    code             VARCHAR(50)  NOT NULL,
    education_level  cbc_education_level  NOT NULL,

    CONSTRAINT fk_cbc_learning_areas_tenant_school FOREIGN KEY (tenant_id, school_id) REFERENCES cbc_schools(tenant_id, id) ON DELETE CASCADE,
    CONSTRAINT uq_cbc_learning_area_school_code UNIQUE (tenant_id, school_id, code)
);

CREATE INDEX IF NOT EXISTS idx_cbc_learning_areas_tenant          ON cbc_learning_areas (tenant_id);
CREATE INDEX IF NOT EXISTS idx_cbc_learning_areas_school_id       ON cbc_learning_areas (school_id);
CREATE INDEX IF NOT EXISTS idx_cbc_learning_areas_education_level ON cbc_learning_areas (education_level);

COMMENT ON COLUMN cbc_learning_areas.education_level IS
    'The CBC tier this learning area belongs to, per KICD curriculum structure.
     Determines applicable KNEC assessment instruments and portal upload eligibility.';

COMMENT ON COLUMN cbc_learning_areas.code IS
    'Short KICD-defined code for this learning area, e.g. MATH, ENG, KISW,
     INT_SCI, PRE_TECH, SOC_STD. Unique within a school''s curriculum.';

-- ---------------------------------------------------------------------------
-- CBC STRANDS
-- ---------------------------------------------------------------------------

CREATE TABLE IF NOT EXISTS cbc_strands (
    id               UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    learning_area_id UUID         NOT NULL REFERENCES cbc_learning_areas(id) ON DELETE CASCADE,
    name             VARCHAR(255) NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_cbc_strands_learning_area_id ON cbc_strands (learning_area_id);

-- ---------------------------------------------------------------------------
-- CBC SUB-STRANDS
-- ---------------------------------------------------------------------------

CREATE TABLE IF NOT EXISTS cbc_sub_strands (
    id        UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    strand_id UUID         NOT NULL REFERENCES cbc_strands(id) ON DELETE CASCADE,
    name      VARCHAR(255) NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_cbc_sub_strands_strand_id ON cbc_sub_strands (strand_id);

-- ---------------------------------------------------------------------------
-- PERFORMANCE INDICATORS  (NEW — leaf node of the curriculum hierarchy)
-- ---------------------------------------------------------------------------

CREATE TABLE IF NOT EXISTS performance_indicators (
    id             UUID     PRIMARY KEY DEFAULT gen_random_uuid(),
    sub_strand_id  UUID     NOT NULL REFERENCES cbc_sub_strands(id) ON DELETE CASCADE,
    description    TEXT     NOT NULL,
    sequence_order SMALLINT NOT NULL DEFAULT 1
);

CREATE INDEX IF NOT EXISTS idx_performance_indicators_sub_strand
    ON performance_indicators (sub_strand_id, sequence_order);

COMMENT ON TABLE performance_indicators IS
    'Atomic CBC learning outcomes within a sub-strand, as defined in KICD
     curriculum designs. Leaf nodes of the hierarchy:
     Learning Area → Strand → Sub-Strand → Performance Indicator.';

-- ============================================================================
-- LAYER 6 — TEACHER ASSIGNMENTS, ATTENDANCE, TIMETABLE
-- ============================================================================

-- ---------------------------------------------------------------------------
-- CBC CLASS TEACHERS
-- ---------------------------------------------------------------------------

CREATE TABLE IF NOT EXISTS cbc_class_teachers (
    id               UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id        UUID        NOT NULL,
    class_id         UUID        NOT NULL,
    user_id          UUID        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    learning_area_id UUID        NOT NULL REFERENCES cbc_learning_areas(id) ON DELETE CASCADE,
    is_primary       BOOLEAN     NOT NULL DEFAULT false,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT fk_cbc_class_teachers_tenant_class FOREIGN KEY (tenant_id, class_id) REFERENCES cbc_classes(tenant_id, id) ON DELETE CASCADE,
    CONSTRAINT unique_cbc_class_teacher UNIQUE (class_id, user_id, learning_area_id)
);

CREATE INDEX IF NOT EXISTS idx_cbc_class_teachers_tenant   ON cbc_class_teachers (tenant_id);
CREATE INDEX IF NOT EXISTS idx_cbc_class_teachers_class_id ON cbc_class_teachers (class_id);
CREATE INDEX IF NOT EXISTS idx_cbc_class_teachers_user_id  ON cbc_class_teachers (user_id);
CREATE INDEX IF NOT EXISTS idx_cbc_class_teachers_area_id  ON cbc_class_teachers (learning_area_id);

CREATE UNIQUE INDEX IF NOT EXISTS idx_cbc_one_primary_per_area
    ON cbc_class_teachers (class_id, learning_area_id) WHERE is_primary = true;

-- ---------------------------------------------------------------------------
-- CBC ATTENDANCE PERIODS
-- ---------------------------------------------------------------------------

CREATE TABLE IF NOT EXISTS cbc_attendance_periods (
    id                   UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id            UUID        NOT NULL,
    school_id            UUID        NOT NULL,
    academic_term_id     UUID        NOT NULL,
    class_id             UUID        NOT NULL,
    cbc_learning_area_id UUID        NOT NULL REFERENCES cbc_learning_areas(id) ON DELETE CASCADE,
    date_recorded        DATE        NOT NULL,
    recorded_by          UUID        NOT NULL REFERENCES users(id),
    created_at           TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT uq_cbc_attendance_periods_tenant UNIQUE (tenant_id, id),
    CONSTRAINT fk_cbc_att_periods_tenant_school FOREIGN KEY (tenant_id, school_id) REFERENCES cbc_schools(tenant_id, id) ON DELETE CASCADE,
    CONSTRAINT fk_cbc_att_periods_tenant_term FOREIGN KEY (tenant_id, school_id, academic_term_id) REFERENCES academic_terms(tenant_id, school_id, id) ON DELETE CASCADE,
    CONSTRAINT fk_cbc_att_periods_tenant_class FOREIGN KEY (tenant_id, class_id) REFERENCES cbc_classes(tenant_id, id) ON DELETE CASCADE
);

CREATE INDEX        IF NOT EXISTS idx_cbc_att_periods_tenant     ON cbc_attendance_periods (tenant_id);
CREATE INDEX        IF NOT EXISTS idx_cbc_att_periods_class_date ON cbc_attendance_periods (class_id, date_recorded);
CREATE UNIQUE INDEX IF NOT EXISTS idx_cbc_unique_attendance_period
    ON cbc_attendance_periods (class_id, date_recorded, cbc_learning_area_id);

-- ---------------------------------------------------------------------------
-- CBC ATTENDANCE LOGS
-- ---------------------------------------------------------------------------

CREATE TABLE IF NOT EXISTS cbc_attendance_logs (
    id                       UUID              PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id                UUID              NOT NULL,
    cbc_attendance_period_id UUID              NOT NULL,
    student_id               UUID              NOT NULL,
    status                   attendance_status NOT NULL,
    remarks                  VARCHAR(255)      NULL,
    recorded_by              UUID              NOT NULL REFERENCES users(id),

    CONSTRAINT fk_cbc_att_logs_tenant_period FOREIGN KEY (tenant_id, cbc_attendance_period_id) REFERENCES cbc_attendance_periods(tenant_id, id) ON DELETE CASCADE,
    CONSTRAINT fk_cbc_att_logs_tenant_student FOREIGN KEY (tenant_id, student_id) REFERENCES cbc_students(tenant_id, id) ON DELETE CASCADE,
    CONSTRAINT unique_cbc_student_attendance_period UNIQUE (cbc_attendance_period_id, student_id)
);

CREATE INDEX IF NOT EXISTS idx_cbc_att_logs_tenant  ON cbc_attendance_logs (tenant_id);
CREATE INDEX IF NOT EXISTS idx_cbc_att_logs_period  ON cbc_attendance_logs (cbc_attendance_period_id);
CREATE INDEX IF NOT EXISTS idx_cbc_att_logs_student ON cbc_attendance_logs (student_id);

-- ---------------------------------------------------------------------------
-- CBC TIMETABLE SLOTS (GiST exclusion constraints kept verbatim)
-- ---------------------------------------------------------------------------

CREATE TABLE IF NOT EXISTS cbc_timetable_slots (
    id                   UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id            UUID        NOT NULL,
    school_id            UUID        NOT NULL,
    academic_year_id     UUID        NOT NULL,
    class_id             UUID        NOT NULL,
    teacher_id           UUID        NOT NULL,
    cbc_learning_area_id UUID        NULL REFERENCES cbc_learning_areas(id) ON DELETE SET NULL,
    room_identifier      VARCHAR(50) NULL,
    day_of_week          INT         NOT NULL CHECK (day_of_week BETWEEN 1 AND 7),
    start_time           TIME        NOT NULL,
    end_time             TIME        NOT NULL,

    CONSTRAINT fk_cbc_timetable_tenant_school FOREIGN KEY (tenant_id, school_id) REFERENCES cbc_schools(tenant_id, id) ON DELETE CASCADE,
    CONSTRAINT fk_cbc_timetable_tenant_year FOREIGN KEY (tenant_id, academic_year_id) REFERENCES academic_years(tenant_id, id) ON DELETE CASCADE,
    CONSTRAINT fk_cbc_timetable_tenant_class FOREIGN KEY (tenant_id, class_id) REFERENCES cbc_classes(tenant_id, id) ON DELETE CASCADE,
    CONSTRAINT fk_cbc_timetable_tenant_teacher FOREIGN KEY (tenant_id, teacher_id) REFERENCES users(tenant_id, id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_cbc_timetable_tenant      ON cbc_timetable_slots (tenant_id);
CREATE INDEX IF NOT EXISTS idx_cbc_timetable_school_year ON cbc_timetable_slots (school_id, academic_year_id);
CREATE INDEX IF NOT EXISTS idx_cbc_timetable_class       ON cbc_timetable_slots (class_id);
CREATE INDEX IF NOT EXISTS idx_cbc_timetable_teacher     ON cbc_timetable_slots (teacher_id);

DO $$ BEGIN
    ALTER TABLE cbc_timetable_slots ADD CONSTRAINT excl_cbc_timetable_teacher
        EXCLUDE USING gist (
            teacher_id       WITH =,
            academic_year_id WITH =,
            fn_timerange(day_of_week, start_time, end_time) WITH &&
        );
EXCEPTION WHEN duplicate_object THEN NULL;
END $$;

DO $$ BEGIN
    ALTER TABLE cbc_timetable_slots ADD CONSTRAINT excl_cbc_timetable_room
        EXCLUDE USING gist (
            room_identifier  WITH =,
            academic_year_id WITH =,
            fn_timerange(day_of_week, start_time, end_time) WITH &&
        );
EXCEPTION WHEN duplicate_object THEN NULL;
END $$;

-- ============================================================================
-- LAYER 7 — CBC ASSESSMENT ARCHITECTURE
-- ============================================================================

-- ---------------------------------------------------------------------------
-- ASSESSMENT WEIGHT CONFIGS  (NEW — KNEC-defined contribution weights)
-- ---------------------------------------------------------------------------

CREATE TABLE IF NOT EXISTS assessment_weight_configs (
    id                   UUID          PRIMARY KEY DEFAULT gen_random_uuid(),
    grade_level          cbc_grade_level    NOT NULL,
    assessment_type_code cbc_assessment_type NOT NULL,
    target_exam          knec_target_exam    NOT NULL,
    weight_percent       NUMERIC(5,2)  NOT NULL,
    effective_from       SMALLINT      NOT NULL,
    notes                TEXT          NULL,

    CONSTRAINT chk_awc_weight_percent CHECK (weight_percent BETWEEN 0.00 AND 100.00),
    CONSTRAINT chk_awc_effective_from CHECK (effective_from >= 2017),
    CONSTRAINT uq_awc_grade_type_exam_effective UNIQUE (grade_level, assessment_type_code, target_exam, effective_from)
);

CREATE INDEX IF NOT EXISTS idx_awc_grade_exam ON assessment_weight_configs (grade_level, target_exam);

COMMENT ON TABLE assessment_weight_configs IS
    'Official KNEC weighting formula per grade per assessment type. Seeded with
     the published KNEC formula. KPSEA: 60% SBA (G4+G5) + 40% KPSEA written (G6).
     KJSEA: 20% SBA (G7+G8) + 20% KPSEA result + 60% KJSEA written (G9).';

-- ---------------------------------------------------------------------------
-- ASSESSMENT BLUEPRINTS  (NEW — replaces defunct assessment_types lookup table)
-- ---------------------------------------------------------------------------

CREATE TABLE IF NOT EXISTS assessment_blueprints (
    id               UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id        UUID         NOT NULL,
    school_id        UUID         NOT NULL,
    title            VARCHAR(255) NOT NULL,
    type             cbc_assessment_type NOT NULL,
    grade_level      cbc_grade_level   NOT NULL,
    academic_year    SMALLINT     NOT NULL,
    term             SMALLINT     NOT NULL,

    CONSTRAINT fk_blueprints_tenant_school FOREIGN KEY (tenant_id, school_id) REFERENCES cbc_schools(tenant_id, id) ON DELETE CASCADE,
    CONSTRAINT chk_blueprint_term CHECK (term BETWEEN 1 AND 3),
    CONSTRAINT chk_blueprint_academic_year CHECK (academic_year >= 2017)
);

CREATE INDEX IF NOT EXISTS idx_blueprints_tenant      ON assessment_blueprints (tenant_id);
CREATE INDEX IF NOT EXISTS idx_blueprints_school      ON assessment_blueprints (school_id);
CREATE INDEX IF NOT EXISTS idx_blueprints_grade_year  ON assessment_blueprints (grade_level, academic_year, type);

-- ---------------------------------------------------------------------------
-- ASSESSMENT BLUEPRINT INDICATORS  (NEW — junction table)
-- ---------------------------------------------------------------------------

CREATE TABLE IF NOT EXISTS assessment_blueprint_indicators (
    blueprint_id UUID NOT NULL REFERENCES assessment_blueprints(id) ON DELETE CASCADE,
    indicator_id UUID NOT NULL REFERENCES performance_indicators(id) ON DELETE CASCADE,

    PRIMARY KEY (blueprint_id, indicator_id)
);

CREATE INDEX IF NOT EXISTS idx_blueprint_indicators_indicator
    ON assessment_blueprint_indicators (indicator_id);

-- ============================================================================
-- LAYER 8 — CBC ASSESSMENT EXECUTION & RESULTS
-- ============================================================================

-- ---------------------------------------------------------------------------
-- ASSESSMENT SESSIONS  (NEW — records one execution of a blueprint)
-- ---------------------------------------------------------------------------

CREATE TABLE IF NOT EXISTS assessment_sessions (
    id                    UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id             UUID        NOT NULL,
    blueprint_id          UUID        NOT NULL REFERENCES assessment_blueprints(id) ON DELETE RESTRICT,
    class_id              UUID        NOT NULL,
    assessed_by_user_id   UUID        NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    date_administered     DATE        NOT NULL,   -- NO DEFAULT. Must be entered explicitly.
    knec_upload_reference VARCHAR(50) NULL,
    created_at            TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT fk_asessions_tenant_class FOREIGN KEY (tenant_id, class_id) REFERENCES cbc_classes(tenant_id, id) ON DELETE RESTRICT
);

CREATE INDEX IF NOT EXISTS idx_asessions_tenant     ON assessment_sessions (tenant_id);
CREATE INDEX IF NOT EXISTS idx_asessions_blueprint  ON assessment_sessions (blueprint_id);
CREATE INDEX IF NOT EXISTS idx_asessions_class      ON assessment_sessions (class_id);
CREATE INDEX IF NOT EXISTS idx_asessions_teacher    ON assessment_sessions (assessed_by_user_id);
CREATE INDEX IF NOT EXISTS idx_asessions_class_date ON assessment_sessions (class_id, date_administered);

COMMENT ON COLUMN assessment_sessions.date_administered IS
    'The calendar date on which this assessment was administered. DATE type
     (not TIMESTAMPTZ) because CBC records reference dates, not timestamps.
     No DEFAULT: must be set explicitly. Retroactive entry is common in CBC
     as teachers often batch-enter assessments at end of week.';

COMMENT ON COLUMN assessment_sessions.knec_upload_reference IS
    'Reference token returned by cba.knec.ac.ke after a successful SBA score
     upload. NULL for Formative_Classroom type sessions, which are never
     uploaded to KNEC.';

-- ---------------------------------------------------------------------------
-- LEARNER RUBRIC RESULTS  (NEW — atomic CBC assessment record)
-- ---------------------------------------------------------------------------

CREATE TABLE IF NOT EXISTS learner_rubric_results (
    id                        UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id                 UUID         NOT NULL,
    session_id                UUID         NOT NULL REFERENCES assessment_sessions(id) ON DELETE CASCADE,
    student_id                UUID         NOT NULL,
    indicator_id              UUID         NOT NULL REFERENCES performance_indicators(id) ON DELETE RESTRICT,
    score_type                lrr_score_type  NOT NULL,
    raw_score                 NUMERIC(5,2) NULL,
    rubric_level              cbc_rubric_level   NOT NULL,
    teacher_observation_notes TEXT         NULL,

    CONSTRAINT fk_lrr_tenant_student FOREIGN KEY (tenant_id, student_id) REFERENCES cbc_students(tenant_id, id) ON DELETE CASCADE,
    CONSTRAINT unique_lrr_per_student_indicator UNIQUE (session_id, student_id, indicator_id)
);

CREATE INDEX IF NOT EXISTS idx_lrr_tenant                ON learner_rubric_results (tenant_id);
CREATE INDEX IF NOT EXISTS idx_lrr_session               ON learner_rubric_results (session_id);
CREATE INDEX IF NOT EXISTS idx_lrr_student_indicator     ON learner_rubric_results (student_id, indicator_id);
CREATE INDEX IF NOT EXISTS idx_lrr_indicator             ON learner_rubric_results (indicator_id);

COMMENT ON COLUMN learner_rubric_results.rubric_level IS
    'Official KNEC 4-level rubric outcome. EE/ME/AE/BE only. No sub-levels
     (EE1, ME2 etc.) are permitted here. Sub-levels may exist in internal
     school tooling but are not valid in KNEC portal submissions.';

COMMENT ON COLUMN learner_rubric_results.raw_score IS
    'Pre-conversion numeric mark. Only populated when score_type = Numeric_Raw.
     Represents the raw score before it is mapped to a rubric level. NEVER
     summed or averaged across indicators — doing so would constitute a CBC
     compliance violation.';

-- ---------------------------------------------------------------------------
-- LEARNER PORTFOLIOS  (NEW — evidence artifacts)
-- ---------------------------------------------------------------------------

CREATE TABLE IF NOT EXISTS learner_portfolios (
    id               UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id        UUID        NOT NULL,
    student_id       UUID        NOT NULL,
    sub_strand_id    UUID        NOT NULL REFERENCES cbc_sub_strands(id) ON DELETE RESTRICT,
    evidence_type    portfolio_evidence_type NOT NULL,
    storage_pointer  TEXT        NOT NULL,
    linked_result_id UUID        NULL REFERENCES learner_rubric_results(id) ON DELETE SET NULL,
    date_collected   DATE        NULL,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT fk_portfolios_tenant_student FOREIGN KEY (tenant_id, student_id) REFERENCES cbc_students(tenant_id, id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_portfolios_tenant     ON learner_portfolios (tenant_id);
CREATE INDEX IF NOT EXISTS idx_portfolios_student    ON learner_portfolios (student_id);
CREATE INDEX IF NOT EXISTS idx_portfolios_sub_strand ON learner_portfolios (sub_strand_id);
CREATE INDEX IF NOT EXISTS idx_portfolios_result     ON learner_portfolios (linked_result_id);

COMMENT ON COLUMN learner_portfolios.storage_pointer IS
    'For Digital_Artifact_URL and Video_Recording: full URL to stored file.
     For Physical_File_Reference: descriptive location string
     (e.g. "Portfolio Binder 2B, page 14, Teacher: J. Mwangi").';

-- ============================================================================
-- LAYER 9 — CBC AGGREGATION & REPORTING
-- ============================================================================

-- ---------------------------------------------------------------------------
-- CBC TERM COMPETENCY SUMMARIES  (NEW — definitive per-term competency record)
-- ---------------------------------------------------------------------------

CREATE TABLE IF NOT EXISTS cbc_term_competency_summaries (
    id               UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id        UUID         NOT NULL,
    student_id       UUID         NOT NULL,
    learning_area_id UUID         NOT NULL REFERENCES cbc_learning_areas(id) ON DELETE RESTRICT,
    class_id         UUID         NOT NULL,
    academic_year    SMALLINT     NOT NULL,
    term             SMALLINT     NOT NULL,
    calculated_level cbc_rubric_level_with_sub_levels NOT NULL,
    override_level   cbc_rubric_level_with_sub_levels NULL,
    final_level      cbc_rubric_level NOT NULL,
    knec_sync_status knec_sync_status NOT NULL DEFAULT 'Pending',
    knec_synced_at   TIMESTAMPTZ  NULL,

    CONSTRAINT fk_summaries_tenant_student FOREIGN KEY (tenant_id, student_id) REFERENCES cbc_students(tenant_id, id) ON DELETE CASCADE,
    CONSTRAINT fk_summaries_tenant_class FOREIGN KEY (tenant_id, class_id) REFERENCES cbc_classes(tenant_id, id) ON DELETE RESTRICT,
    CONSTRAINT chk_summary_term CHECK (term BETWEEN 1 AND 3),
    CONSTRAINT chk_summary_academic_year CHECK (academic_year >= 2017),
    CONSTRAINT unique_summary_per_student_area_term UNIQUE (student_id, learning_area_id, academic_year, term)
);

CREATE INDEX IF NOT EXISTS idx_summaries_tenant          ON cbc_term_competency_summaries (tenant_id);
CREATE INDEX IF NOT EXISTS idx_summaries_sync_batch      ON cbc_term_competency_summaries (academic_year, term, knec_sync_status);
CREATE INDEX IF NOT EXISTS idx_summaries_student_year    ON cbc_term_competency_summaries (student_id, academic_year);
CREATE INDEX IF NOT EXISTS idx_summaries_class           ON cbc_term_competency_summaries (class_id);

COMMENT ON TABLE cbc_term_competency_summaries IS
    'Definitive per-term competency record per learner per learning area.
     final_level is the KNEC portal submission value — must always be one of
     EE/ME/AE/BE. Sub-levels (EE1 etc.) are only valid for the internal
     calculated_level and override_level fields. knec_synced_at is NULL until
     the first successful upload to cba.knec.ac.ke.';

CREATE TABLE school_member_counts (
    school_id UUID PRIMARY KEY REFERENCES cbc_schools(id) ON DELETE CASCADE,
    admins INT NOT NULL DEFAULT 0,
    teachers INT NOT NULL DEFAULT 0,
    nurses INT NOT NULL DEFAULT 0,
    finance INT NOT NULL DEFAULT 0,
    parents INT NOT NULL DEFAULT 0,
    students INT NOT NULL DEFAULT 0,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_school_member_counts ON school_member_counts (school_id);

-- ============================================================
-- TRIGGER: Sync school staff/parent counts from memberships
-- ============================================================

-- Separate functions so each trigger only references the transition tables it has access to.

CREATE OR REPLACE FUNCTION fn_sync_school_staff_counts_insert()
RETURNS TRIGGER AS $$
BEGIN
    INSERT INTO school_member_counts (school_id, admins, teachers, nurses, finance, parents, updated_at)
    SELECT
        s.school_id,
        COUNT(*) FILTER (WHERE m.role = 'SCHOOL_ADMIN') AS admins,
        COUNT(*) FILTER (WHERE m.role = 'TEACHER')      AS teachers,
        COUNT(*) FILTER (WHERE m.role = 'NURSE')        AS nurses,
        COUNT(*) FILTER (WHERE m.role = 'FINANCE')      AS finance,
        COUNT(*) FILTER (WHERE m.role = 'PARENT')       AS parents,
        NOW()
    FROM (SELECT DISTINCT school_id FROM inserted_rows) s
    LEFT JOIN memberships m
        ON m.school_id = s.school_id
        AND m.is_active = true
    GROUP BY s.school_id
    ON CONFLICT (school_id) DO UPDATE SET
        admins     = EXCLUDED.admins,
        teachers   = EXCLUDED.teachers,
        nurses     = EXCLUDED.nurses,
        finance    = EXCLUDED.finance,
        parents    = EXCLUDED.parents,
        updated_at = NOW();

    RETURN NULL;
END;
$$ LANGUAGE plpgsql;

CREATE OR REPLACE FUNCTION fn_sync_school_staff_counts_delete()
RETURNS TRIGGER AS $$
BEGIN
    INSERT INTO school_member_counts (school_id, admins, teachers, nurses, finance, parents, updated_at)
    SELECT
        s.school_id,
        COUNT(*) FILTER (WHERE m.role = 'SCHOOL_ADMIN') AS admins,
        COUNT(*) FILTER (WHERE m.role = 'TEACHER')      AS teachers,
        COUNT(*) FILTER (WHERE m.role = 'NURSE')        AS nurses,
        COUNT(*) FILTER (WHERE m.role = 'FINANCE')      AS finance,
        COUNT(*) FILTER (WHERE m.role = 'PARENT')       AS parents,
        NOW()
    FROM (SELECT DISTINCT school_id FROM deleted_rows) s
    LEFT JOIN memberships m
        ON m.school_id = s.school_id
        AND m.is_active = true
    GROUP BY s.school_id
    ON CONFLICT (school_id) DO UPDATE SET
        admins     = EXCLUDED.admins,
        teachers   = EXCLUDED.teachers,
        nurses     = EXCLUDED.nurses,
        finance    = EXCLUDED.finance,
        parents    = EXCLUDED.parents,
        updated_at = NOW();

    RETURN NULL;
END;
$$ LANGUAGE plpgsql;

CREATE OR REPLACE FUNCTION fn_sync_school_staff_counts_update()
RETURNS TRIGGER AS $$
BEGIN
    WITH affected_schools AS (
        SELECT DISTINCT school_id FROM inserted_rows
        UNION
        SELECT DISTINCT school_id FROM deleted_rows
    )
    INSERT INTO school_member_counts (school_id, admins, teachers, nurses, finance, parents, updated_at)
    SELECT
        s.school_id,
        COUNT(*) FILTER (WHERE m.role = 'SCHOOL_ADMIN') AS admins,
        COUNT(*) FILTER (WHERE m.role = 'TEACHER')      AS teachers,
        COUNT(*) FILTER (WHERE m.role = 'NURSE')        AS nurses,
        COUNT(*) FILTER (WHERE m.role = 'FINANCE')      AS finance,
        COUNT(*) FILTER (WHERE m.role = 'PARENT')       AS parents,
        NOW()
    FROM affected_schools s
    LEFT JOIN memberships m
        ON m.school_id = s.school_id
        AND m.is_active = true
    GROUP BY s.school_id
    ON CONFLICT (school_id) DO UPDATE SET
        admins     = EXCLUDED.admins,
        teachers   = EXCLUDED.teachers,
        nurses     = EXCLUDED.nurses,
        finance    = EXCLUDED.finance,
        parents    = EXCLUDED.parents,
        updated_at = NOW();

    RETURN NULL;
END;
$$ LANGUAGE plpgsql;

-- Fires on INSERT
CREATE TRIGGER trg_memberships_counts_insert
    AFTER INSERT ON memberships
    REFERENCING NEW TABLE AS inserted_rows
    FOR EACH STATEMENT
    EXECUTE FUNCTION fn_sync_school_staff_counts_insert();

-- Fires on DELETE
CREATE TRIGGER trg_memberships_counts_delete
    AFTER DELETE ON memberships
    REFERENCING OLD TABLE AS deleted_rows
    FOR EACH STATEMENT
    EXECUTE FUNCTION fn_sync_school_staff_counts_delete();

-- Fires on UPDATE
CREATE TRIGGER trg_memberships_counts_update
    AFTER UPDATE ON memberships
    REFERENCING NEW TABLE AS inserted_rows OLD TABLE AS deleted_rows
    FOR EACH STATEMENT
    EXECUTE FUNCTION fn_sync_school_staff_counts_update();


-- ============================================================
-- TRIGGER: Sync school student counts from cbc_students
-- ============================================================

-- Separate functions so each trigger only references the transition tables it has access to.

CREATE OR REPLACE FUNCTION fn_sync_school_student_counts_insert()
RETURNS TRIGGER AS $$
BEGIN
    INSERT INTO school_member_counts (school_id, students, updated_at)
    SELECT
        s.school_id,
        COUNT(st.id) AS students,
        NOW()
    FROM (SELECT DISTINCT school_id FROM inserted_rows) s
    LEFT JOIN cbc_students st
        ON st.school_id = s.school_id
        AND st.is_active = true
    GROUP BY s.school_id
    ON CONFLICT (school_id) DO UPDATE SET
        students   = EXCLUDED.students,
        updated_at = NOW();

    RETURN NULL;
END;
$$ LANGUAGE plpgsql;

CREATE OR REPLACE FUNCTION fn_sync_school_student_counts_delete()
RETURNS TRIGGER AS $$
BEGIN
    INSERT INTO school_member_counts (school_id, students, updated_at)
    SELECT
        s.school_id,
        COUNT(st.id) AS students,
        NOW()
    FROM (SELECT DISTINCT school_id FROM deleted_rows) s
    LEFT JOIN cbc_students st
        ON st.school_id = s.school_id
        AND st.is_active = true
    GROUP BY s.school_id
    ON CONFLICT (school_id) DO UPDATE SET
        students   = EXCLUDED.students,
        updated_at = NOW();

    RETURN NULL;
END;
$$ LANGUAGE plpgsql;

CREATE OR REPLACE FUNCTION fn_sync_school_student_counts_update()
RETURNS TRIGGER AS $$
BEGIN
    WITH affected_schools AS (
        SELECT DISTINCT school_id FROM inserted_rows
        UNION
        SELECT DISTINCT school_id FROM deleted_rows
    )
    INSERT INTO school_member_counts (school_id, students, updated_at)
    SELECT
        s.school_id,
        COUNT(st.id) AS students,
        NOW()
    FROM affected_schools s
    LEFT JOIN cbc_students st
        ON st.school_id = s.school_id
        AND st.is_active = true
    GROUP BY s.school_id
    ON CONFLICT (school_id) DO UPDATE SET
        students   = EXCLUDED.students,
        updated_at = NOW();

    RETURN NULL;
END;
$$ LANGUAGE plpgsql;

-- Fires on INSERT
CREATE TRIGGER trg_cbc_students_counts_insert
    AFTER INSERT ON cbc_students
    REFERENCING NEW TABLE AS inserted_rows
    FOR EACH STATEMENT
    EXECUTE FUNCTION fn_sync_school_student_counts_insert();

-- Fires on DELETE
CREATE TRIGGER trg_cbc_students_counts_delete
    AFTER DELETE ON cbc_students
    REFERENCING OLD TABLE AS deleted_rows
    FOR EACH STATEMENT
    EXECUTE FUNCTION fn_sync_school_student_counts_delete();
-- Fires on UPDATE
CREATE TRIGGER trg_cbc_students_counts_update
    AFTER UPDATE ON cbc_students
    REFERENCING NEW TABLE AS inserted_rows OLD TABLE AS deleted_rows
    FOR EACH STATEMENT
    EXECUTE FUNCTION fn_sync_school_student_counts_update();


-- ============================================================================
-- LAYER 10 — USER ACTIVE SCHOOL CONTEXT
-- ============================================================================

-- ---------------------------------------------------------------------------
-- MEMBER ACTIVE SCHOOL
-- ---------------------------------------------------------------------------
-- Tracks which school is the current working context for a user.
-- A user belonging to multiple schools in one tenant can switch context freely.
-- Application upsert pattern:
--   INSERT INTO member_active_school (user_id, tenant_id, school_id, switched_at)
--   VALUES ($1, $2, $3, NOW())
--   ON CONFLICT (user_id) DO UPDATE
--     SET school_id   = EXCLUDED.school_id,
--         tenant_id   = EXCLUDED.tenant_id,
--         switched_at = NOW();

CREATE TABLE IF NOT EXISTS member_active_school (
    user_id     UUID        NOT NULL,
    tenant_id   UUID        NOT NULL,
    school_id   UUID        NOT NULL,
    switched_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    PRIMARY KEY (user_id),

    CONSTRAINT fk_mas_user            FOREIGN KEY (user_id)              REFERENCES users(id)                     ON DELETE CASCADE,
    CONSTRAINT fk_mas_tenant_user     FOREIGN KEY (tenant_id, user_id)   REFERENCES users(tenant_id, id)          ON DELETE CASCADE,
    CONSTRAINT fk_mas_tenant_school   FOREIGN KEY (tenant_id, school_id) REFERENCES cbc_schools(tenant_id, id)    ON DELETE CASCADE,
    CONSTRAINT fk_mas_membership      FOREIGN KEY (user_id, school_id)   REFERENCES memberships(user_id, school_id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_mas_tenant_id ON member_active_school (tenant_id);

COMMENT ON TABLE member_active_school IS
    'Tracks the currently active school context for each user within a tenant.
     One row per user. Upsert on school switch. The chosen school_id is
     constrained to schools the user is an active member of via fk_mas_membership.';

-- ============================================================================
-- ACADEMIC CALENDAR ENHANCEMENTS (2026-06-26)
--   - Optimistic locking, soft delete, audit trail
--   - Updated partial unique indexes for soft-delete awareness
--   - FK changed to RESTRICT (CASCADE incompatible with soft delete)
-- ============================================================================

-- ---------------------------------------------------------------------------
-- ACADEMIC YEARS — Add versioning, soft delete, audit columns
-- ---------------------------------------------------------------------------

ALTER TABLE academic_years
  ADD COLUMN IF NOT EXISTS version    INTEGER              NOT NULL DEFAULT 1,
  ADD COLUMN IF NOT EXISTS deleted_at TIMESTAMPTZ,
  ADD COLUMN IF NOT EXISTS created_by UUID                 NOT NULL REFERENCES users(id),
  ADD COLUMN IF NOT EXISTS updated_by UUID                 NOT NULL REFERENCES users(id);

-- Drop the old partial unique index and replace with one that excludes
-- soft-deleted rows. The old index used WHERE is_current = true;
DROP INDEX IF EXISTS idx_one_current_year_per_school;
CREATE UNIQUE INDEX IF NOT EXISTS idx_one_current_year_per_school
  ON academic_years (school_id)
  WHERE is_current = TRUE AND deleted_at IS NULL;

-- Rename / normalise the check constraint (was end_date > start_date)
ALTER TABLE academic_years
  DROP CONSTRAINT IF EXISTS chk_academic_year_dates,
  ADD CONSTRAINT chk_year_dates CHECK (start_date < end_date);

-- ---------------------------------------------------------------------------
-- ACADEMIC TERMS — Add versioning, soft delete, audit columns
-- ---------------------------------------------------------------------------

ALTER TABLE academic_terms
  ADD COLUMN IF NOT EXISTS version    INTEGER              NOT NULL DEFAULT 1,
  ADD COLUMN IF NOT EXISTS deleted_at TIMESTAMPTZ,
  ADD COLUMN IF NOT EXISTS created_by UUID                 NOT NULL REFERENCES users(id),
  ADD COLUMN IF NOT EXISTS updated_by UUID                 NOT NULL REFERENCES users(id);

-- Drop old partial unique index on current term, replace with soft-delete-aware
DROP INDEX IF EXISTS idx_one_current_term_per_year;
CREATE UNIQUE INDEX IF NOT EXISTS idx_one_current_term_per_year
  ON academic_terms (academic_year_id)
  WHERE is_current = TRUE AND deleted_at IS NULL;

-- Drop old full unique constraint on (academic_year_id, term_number) and
-- replace with a partial unique index that excludes soft-deleted rows.
ALTER TABLE academic_terms
  DROP CONSTRAINT IF EXISTS uq_academic_year_term_number;
CREATE UNIQUE INDEX IF NOT EXISTS idx_unique_term_number_per_year
  ON academic_terms (academic_year_id, term_number)
  WHERE deleted_at IS NULL;

-- Normalise check constraint names
ALTER TABLE academic_terms
  DROP CONSTRAINT IF EXISTS chk_academic_term_dates,
  ADD CONSTRAINT chk_term_dates CHECK (start_date < end_date);

ALTER TABLE academic_terms
  DROP CONSTRAINT IF EXISTS chk_academic_term_number;

-- chk_term_number already named correctly in the original CREATE TABLE, but
-- in case the previous DROP removed it, re-add:
ALTER TABLE academic_terms
  ADD CONSTRAINT IF NOT EXISTS chk_term_number CHECK (term_number BETWEEN 1 AND 3);

-- Change FK from CASCADE to RESTRICT (soft-delete replaces cascade deletion)
ALTER TABLE academic_terms
  DROP CONSTRAINT IF EXISTS fk_academic_terms_tenant_year,
  ADD CONSTRAINT fk_academic_terms_tenant_year
    FOREIGN KEY (tenant_id, academic_year_id)
    REFERENCES academic_years(tenant_id, id)
    ON DELETE RESTRICT;

-- ============================================================================
-- END OF MIGRATION
-- ============================================================================

COMMIT;
