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
    CREATE TYPE teacher_role AS ENUM (
        'PRIMARY_CLASS_TEACHER',
        'SUBJECT_TEACHER',
        'SUBSTITUTE_TEACHER'
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

-- IMPROVE: import_job_status enum replaces unconstrained TEXT on import_jobs.status
DO $$ BEGIN
    CREATE TYPE import_job_status AS ENUM (
        'pending',
        'processing',
        'completed',
        'failed',
        'cancelled'
    );
EXCEPTION WHEN duplicate_object THEN NULL;
END $$;

-- completed_with_errors is used by the application to distinguish successful
-- imports that had partial failures (some records succeeded, some failed)
-- from clean completed imports (all records succeeded).
ALTER TYPE import_job_status ADD VALUE IF NOT EXISTS 'completed_with_errors';

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
    id                     UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    email                  VARCHAR(255) NOT NULL,
    tenant_id              UUID         NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    full_name              VARCHAR(255) NOT NULL DEFAULT '',
    is_active              BOOLEAN      NOT NULL DEFAULT TRUE,
    external_auth_id       VARCHAR(255) UNIQUE,
    tsc_number             VARCHAR(15)  NULL,
    knec_panel_assessor_id VARCHAR(20)  NULL,
    created_at             TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at             TIMESTAMPTZ  NOT NULL DEFAULT NOW(),

    CONSTRAINT uq_users_tenant UNIQUE (tenant_id, id)
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_users_email ON users (email);
CREATE INDEX        IF NOT EXISTS idx_users_tenant ON users (tenant_id);
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
    id                      UUID            PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id               UUID            NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    name                    VARCHAR(255)    NOT NULL,
    knec_school_code        VARCHAR(15)     NULL,
    nemis_institution_code  VARCHAR(20)     NULL,
    county                  VARCHAR(50)     NOT NULL,
    sub_county              VARCHAR(50)     NOT NULL,
    ward                    VARCHAR(50)     NULL,
    school_type             cbc_school_type NOT NULL,
    is_active               BOOLEAN         NOT NULL DEFAULT true,
    created_at              TIMESTAMPTZ     NOT NULL DEFAULT NOW(),
    updated_at              TIMESTAMPTZ     NOT NULL DEFAULT NOW(),

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
-- LAYER 3 — ACADEMIC CALENDAR
-- ============================================================================

-- ---------------------------------------------------------------------------
-- ACADEMIC YEARS
-- IMPROVE: added created_at / updated_at and audit columns (version, deleted_at, created_by, updated_by)
-- ---------------------------------------------------------------------------

CREATE TABLE IF NOT EXISTS academic_years (
    id         UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id  UUID        NOT NULL,
    school_id  UUID        NOT NULL,
    name       VARCHAR(50) NOT NULL,
    start_date DATE        NOT NULL,
    end_date   DATE        NOT NULL,
    is_current BOOLEAN     NOT NULL DEFAULT false,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    version    INTEGER     NOT NULL DEFAULT 1,
    deleted_at TIMESTAMPTZ,
    created_by UUID        NOT NULL REFERENCES users(id),
    updated_by UUID        NOT NULL REFERENCES users(id),

    CONSTRAINT chk_year_dates CHECK (start_date < end_date),
    CONSTRAINT uq_academic_years_tenant UNIQUE (tenant_id, id),
    CONSTRAINT fk_academic_years_tenant_school
        FOREIGN KEY (tenant_id, school_id)
        REFERENCES cbc_schools(tenant_id, id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_academic_years_tenant_id ON academic_years (tenant_id);
CREATE INDEX IF NOT EXISTS idx_academic_years_school_id ON academic_years (school_id);

CREATE UNIQUE INDEX IF NOT EXISTS idx_one_current_year_per_school
    ON academic_years (school_id) WHERE is_current = TRUE AND deleted_at IS NULL;

DROP TRIGGER IF EXISTS trg_academic_years_updated_at ON academic_years;
CREATE TRIGGER trg_academic_years_updated_at
    BEFORE UPDATE ON academic_years
    FOR EACH ROW EXECUTE FUNCTION fn_set_updated_at();

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
    version          INTEGER      NOT NULL DEFAULT 1,
    deleted_at       TIMESTAMPTZ,
    created_by       UUID         NOT NULL REFERENCES users(id),
    updated_by       UUID         NOT NULL REFERENCES users(id),

    CONSTRAINT chk_term_dates   CHECK (start_date < end_date),
    CONSTRAINT chk_term_number  CHECK (term_number BETWEEN 1 AND 3),
    CONSTRAINT uq_academic_terms_tenant        UNIQUE (tenant_id, id),
    CONSTRAINT uq_academic_terms_tenant_school UNIQUE (tenant_id, school_id, id),
    CONSTRAINT fk_academic_terms_tenant_school
        FOREIGN KEY (tenant_id, school_id)
        REFERENCES cbc_schools(tenant_id, id) ON DELETE CASCADE,
    CONSTRAINT fk_academic_terms_tenant_year
        FOREIGN KEY (tenant_id, academic_year_id)
        REFERENCES academic_years(tenant_id, id) ON DELETE RESTRICT
);

CREATE INDEX IF NOT EXISTS idx_academic_terms_tenant_id ON academic_terms (tenant_id);
-- BUG FIX: was incorrectly targeting academic_years; fixed to academic_terms
CREATE INDEX IF NOT EXISTS idx_academic_terms_school_id ON academic_terms (school_id);
CREATE INDEX IF NOT EXISTS idx_academic_terms_year_id   ON academic_terms (academic_year_id);

CREATE UNIQUE INDEX IF NOT EXISTS idx_one_current_term_per_year
    ON academic_terms (academic_year_id) WHERE is_current = TRUE AND deleted_at IS NULL;

CREATE UNIQUE INDEX IF NOT EXISTS idx_unique_term_number_per_year
    ON academic_terms (academic_year_id, term_number)
    WHERE deleted_at IS NULL;

COMMENT ON COLUMN academic_terms.term_number IS
    'Kenya CBC operates a 3-term academic year. term_number enforces this:
     1 = Term 1, 2 = Term 2, 3 = Term 3.';

COMMENT ON COLUMN academic_terms.is_final IS
    'Marks the last term of the academic year before a national KNEC exam cycle
     (KPSEA at end of G6, KJSEA at end of G9, KSSEA at end of G12). The
     application uses this flag to lock SBA submissions and trigger KNEC sync
     workflows. Set to TRUE only on Term 3 of an exam year.';

-- ============================================================================
-- LAYER 2 — CORE CBC ACTORS (continued)
-- ============================================================================

-- ---------------------------------------------------------------------------
-- CBC STREAMS
-- ---------------------------------------------------------------------------

CREATE TABLE IF NOT EXISTS cbc_streams (
    id         UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id  UUID         NOT NULL,
    school_id  UUID         NOT NULL,
    name       VARCHAR(100) NOT NULL,
    created_at TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ  NOT NULL DEFAULT NOW(),

    CONSTRAINT fk_cbc_streams_school
        FOREIGN KEY (school_id) REFERENCES cbc_schools(id) ON DELETE NO ACTION,

    CONSTRAINT uq_cbc_streams_tenant_school_name
        UNIQUE (tenant_id, school_id, name)
);

CREATE INDEX IF NOT EXISTS idx_cbc_streams_school_id ON cbc_streams (school_id);
CREATE INDEX IF NOT EXISTS idx_cbc_streams_tenant_id ON cbc_streams (tenant_id);

DROP TRIGGER IF EXISTS trg_cbc_streams_updated_at ON cbc_streams;
CREATE TRIGGER trg_cbc_streams_updated_at
    BEFORE UPDATE ON cbc_streams
    FOR EACH ROW EXECUTE FUNCTION fn_set_updated_at();

COMMENT ON TABLE cbc_streams IS
    'Named streams within a school (e.g. "Blue", "Red", "Green"). A stream is
     the second axis of class identity alongside grade_level. Streams themselves
     cannot be deleted while any cbc_classes row references them (ON DELETE
     RESTRICT via fk_cbc_classes_stream). Streams with no class references may
     be deleted via the API.';

COMMENT ON CONSTRAINT fk_cbc_streams_school ON cbc_streams IS
    'school_id alone carries the tenancy relationship via cbc_schools. tenant_id
     is stored denormalised for fast multi-tenant filtering without joins.
     ON DELETE NO ACTION — streams are never cascade-deleted. Deletion of a
     school must be handled explicitly upstream before streams can be removed.';

-- ---------------------------------------------------------------------------
-- CBC CLASSES
-- IMPROVE: added created_at / updated_at (were absent despite being on every
--          other major entity) and corresponding updated_at trigger
-- ---------------------------------------------------------------------------

CREATE TABLE IF NOT EXISTS cbc_classes (
    id               UUID            PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id        UUID            NOT NULL,
    school_id        UUID            NOT NULL,
    academic_year_id UUID            NOT NULL,
    grade_level      cbc_grade_level NOT NULL,
    stream_id        UUID            NOT NULL,
    is_active        BOOLEAN         NOT NULL DEFAULT true,
    created_at       TIMESTAMPTZ     NOT NULL DEFAULT NOW(),
    updated_at       TIMESTAMPTZ     NOT NULL DEFAULT NOW(),

    CONSTRAINT uq_cbc_classes_tier_stream
        UNIQUE (school_id, academic_year_id, grade_level, stream_id),
    CONSTRAINT fk_cbc_classes_tenant_school
        FOREIGN KEY (tenant_id, school_id)
        REFERENCES cbc_schools(tenant_id, id) ON DELETE CASCADE,
    CONSTRAINT fk_cbc_classes_tenant_academic_year
        FOREIGN KEY (tenant_id, academic_year_id)
        REFERENCES academic_years(tenant_id, id) ON DELETE CASCADE,
    CONSTRAINT fk_cbc_classes_stream
        FOREIGN KEY (stream_id) REFERENCES cbc_streams(id) ON DELETE RESTRICT,

    -- IMPROVE: composite FK for tenant scoping (tenant_id, id) to allow other
    -- tables to reference this pair directly
    CONSTRAINT uq_cbc_classes_tenant UNIQUE (tenant_id, id)
);

CREATE INDEX IF NOT EXISTS idx_cbc_classes_tenant_id        ON cbc_classes (tenant_id);
CREATE INDEX IF NOT EXISTS idx_cbc_classes_school_id        ON cbc_classes (school_id);
CREATE INDEX IF NOT EXISTS idx_cbc_classes_academic_year_id ON cbc_classes (academic_year_id);
CREATE INDEX IF NOT EXISTS idx_cbc_classes_grade_level      ON cbc_classes (grade_level);
CREATE INDEX IF NOT EXISTS idx_cbc_classes_school_year_grade_stream
    ON cbc_classes (school_id, academic_year_id, grade_level, stream_id);

DROP TRIGGER IF EXISTS trg_cbc_classes_updated_at ON cbc_classes;
CREATE TRIGGER trg_cbc_classes_updated_at
    BEFORE UPDATE ON cbc_classes
    FOR EACH ROW EXECUTE FUNCTION fn_set_updated_at();

COMMENT ON COLUMN cbc_classes.grade_level IS
    'Official KNEC grade designation. Determines which assessment instruments,
     SBA projects, and KNEC portal upload windows apply to the class. Values
     match KNEC CBA portal grade codes: PP1–PP2 (Pre-Primary), G1–G12.';

-- ---------------------------------------------------------------------------
-- MEMBERSHIPS
-- IMPROVE: added updated_at (role changes / is_active toggling had no timestamp)
-- ---------------------------------------------------------------------------

CREATE TABLE IF NOT EXISTS memberships (
    id         UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id  UUID        NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    user_id    UUID        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    school_id  UUID        NOT NULL,
    role       user_role   NOT NULL,
    is_active  BOOLEAN     NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT fk_memberships_tenant_school
        FOREIGN KEY (tenant_id, school_id)
        REFERENCES cbc_schools(tenant_id, id) ON DELETE CASCADE,
    CONSTRAINT unique_user_school_membership UNIQUE (user_id, school_id)
);

CREATE INDEX IF NOT EXISTS idx_memberships_tenant_id ON memberships (tenant_id);
CREATE INDEX IF NOT EXISTS idx_memberships_user_id   ON memberships (user_id);
CREATE INDEX IF NOT EXISTS idx_memberships_school_id ON memberships (school_id);

DROP TRIGGER IF EXISTS trg_memberships_updated_at ON memberships;
CREATE TRIGGER trg_memberships_updated_at
    BEFORE UPDATE ON memberships
    FOR EACH ROW EXECUTE FUNCTION fn_set_updated_at();

-- ---------------------------------------------------------------------------
-- IMPORT JOBS — Bulk Staff Invitation async ingestion
-- IMPROVE: status column changed from unconstrained TEXT to import_job_status enum
-- ---------------------------------------------------------------------------

CREATE TABLE IF NOT EXISTS import_jobs (
    id                   UUID              PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id            UUID              NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    school_id            UUID              NOT NULL,
    role                 user_role         NOT NULL,
    created_by           UUID              REFERENCES users(id) ON DELETE SET NULL,
    status               import_job_status NOT NULL DEFAULT 'pending',
    total_records        INT               NOT NULL DEFAULT 0,
    processed_records    INT               NOT NULL DEFAULT 0,
    success_count        INT               NOT NULL DEFAULT 0,
    failed_count         INT               NOT NULL DEFAULT 0,
    parent_import_job_id UUID              NULL,
    created_at           TIMESTAMPTZ       NOT NULL DEFAULT NOW(),
    started_at           TIMESTAMPTZ       NULL,
    completed_at         TIMESTAMPTZ       NULL,

    CONSTRAINT fk_import_jobs_tenant_school
        FOREIGN KEY (tenant_id, school_id)
        REFERENCES cbc_schools(tenant_id, id) ON DELETE CASCADE,
    CONSTRAINT fk_import_jobs_parent
        FOREIGN KEY (parent_import_job_id)
        REFERENCES import_jobs(id) DEFERRABLE INITIALLY DEFERRED
);

CREATE INDEX IF NOT EXISTS idx_import_jobs_tenant_id  ON import_jobs (tenant_id);
CREATE INDEX IF NOT EXISTS idx_import_jobs_school_id  ON import_jobs (school_id);
CREATE INDEX IF NOT EXISTS idx_import_jobs_created_by ON import_jobs (created_by);
CREATE INDEX IF NOT EXISTS idx_import_jobs_status     ON import_jobs (status);

CREATE TABLE IF NOT EXISTS import_job_failures (
    id            BIGSERIAL   PRIMARY KEY,
    import_job_id UUID        NOT NULL REFERENCES import_jobs(id) ON DELETE CASCADE,
    raw_payload   JSONB       NOT NULL,
    error_message TEXT        NOT NULL,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_import_job_failures_job_id ON import_job_failures (import_job_id);

-- ---------------------------------------------------------------------------
-- IMPORT JOB STAGING — Student bulk import staging rows
-- ---------------------------------------------------------------------------

CREATE TABLE IF NOT EXISTS import_job_staging (
    id         UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    job_id     UUID        NOT NULL REFERENCES import_jobs(id) ON DELETE CASCADE,
    tenant_id  UUID        NOT NULL,
    school_id  UUID        NOT NULL,
    row_number INT         NOT NULL,
    raw_data   JSONB       NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
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

    CONSTRAINT fk_invitations_tenant_school
        FOREIGN KEY (tenant_id, school_id)
        REFERENCES cbc_schools(tenant_id, id) ON DELETE CASCADE,
    CONSTRAINT fk_invitations_import_job
        FOREIGN KEY (import_job_id)
        REFERENCES import_jobs(id) ON DELETE SET NULL
);

CREATE INDEX IF NOT EXISTS idx_invitations_tenant_id  ON invitations (tenant_id);
CREATE INDEX IF NOT EXISTS idx_invitations_school_id  ON invitations (school_id);
CREATE INDEX IF NOT EXISTS idx_invitations_email      ON invitations (email);
CREATE INDEX IF NOT EXISTS idx_invitations_status     ON invitations (status);
CREATE INDEX IF NOT EXISTS idx_invitations_import_job ON invitations (import_job_id);

CREATE UNIQUE INDEX IF NOT EXISTS uq_invitations_active_email
    ON invitations (tenant_id, school_id, email)
    WHERE status NOT IN ('expired', 'revoked');

-- ---------------------------------------------------------------------------
-- CBC PARENTS
-- ---------------------------------------------------------------------------

CREATE TABLE IF NOT EXISTS cbc_parents (
    id           UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id    UUID         NOT NULL,
    user_id      UUID         NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    phone_number VARCHAR(20)  NOT NULL, -- Crucial for M-Pesa & SMS notifications
    is_active    BOOLEAN      NOT NULL DEFAULT true,
    created_at   TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at   TIMESTAMPTZ  NOT NULL DEFAULT NOW(),

    CONSTRAINT uq_cbc_parents_user UNIQUE (user_id),
    CONSTRAINT fk_cbc_parents_tenant_user
        FOREIGN KEY (tenant_id, user_id)
        REFERENCES users(tenant_id, id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_cbc_parents_phone    ON cbc_parents (phone_number);
CREATE INDEX IF NOT EXISTS idx_cbc_parents_tenant   ON cbc_parents (tenant_id);

DROP TRIGGER IF EXISTS trg_cbc_parents_updated_at ON cbc_parents;
CREATE TRIGGER trg_cbc_parents_updated_at
    BEFORE UPDATE ON cbc_parents
    FOR EACH ROW EXECUTE FUNCTION fn_set_updated_at();

COMMENT ON TABLE cbc_parents IS
    'Profile extension table for users acting as parents or guardians. Links
     directly to the platform users table to leverage Stytch B2B auth loops.';

-- ---------------------------------------------------------------------------
-- CBC STUDENTS
-- ---------------------------------------------------------------------------

CREATE TABLE IF NOT EXISTS cbc_students (
    id                     UUID                 PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id              UUID                 NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    school_id              UUID                 NOT NULL REFERENCES cbc_schools(id) ON DELETE RESTRICT,
    full_name              VARCHAR(255)         NOT NULL,
    gender                 gender_type          NOT NULL,
    date_of_birth          DATE                 NULL,
    upi_number             VARCHAR(20)          NULL,
    knec_assessment_number VARCHAR(15)          NULL,
    learning_pathway       cbc_learning_pathway NOT NULL DEFAULT 'Age_Based',
    is_active              BOOLEAN              NOT NULL DEFAULT true,
    created_at             TIMESTAMPTZ          NOT NULL DEFAULT NOW(),
    updated_at             TIMESTAMPTZ          NOT NULL DEFAULT NOW(),

    CONSTRAINT uq_cbc_students_tenant UNIQUE (tenant_id, id),
    CONSTRAINT chk_cbc_student_gender CHECK (gender IN ('M', 'F'))
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_cbc_students_upi
    ON cbc_students (upi_number) WHERE upi_number IS NOT NULL;
CREATE UNIQUE INDEX IF NOT EXISTS idx_cbc_students_knec_assessment_number
    ON cbc_students (knec_assessment_number) WHERE knec_assessment_number IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_cbc_students_tenant_id ON cbc_students (tenant_id);
CREATE INDEX IF NOT EXISTS idx_cbc_students_school_id ON cbc_students (school_id);

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
-- CBC STUDENT PARENTS JUNCTION
-- ---------------------------------------------------------------------------

CREATE TABLE IF NOT EXISTS cbc_student_parents (
    student_id   UUID        NOT NULL REFERENCES cbc_students(id) ON DELETE CASCADE,
    parent_id    UUID        NOT NULL REFERENCES cbc_parents(id)  ON DELETE CASCADE,
    relationship VARCHAR(50) NULL, -- 'Father', 'Mother', 'Guardian'
    is_primary   BOOLEAN     NOT NULL DEFAULT true,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    PRIMARY KEY (student_id, parent_id)
);

CREATE INDEX IF NOT EXISTS idx_junction_parent ON cbc_student_parents (parent_id);

-- ---------------------------------------------------------------------------
-- CBC STUDENT ENROLLMENTS
-- IMPROVE: added updated_at so status transitions (ACTIVE→SUSPENDED→TRANSFERRED)
--          are timestamped at the row level
-- ---------------------------------------------------------------------------

CREATE TABLE IF NOT EXISTS cbc_student_enrollments (
    id               UUID                  PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id        UUID                  NOT NULL,
    student_id       UUID                  NOT NULL,
    school_id        UUID                  NOT NULL,
    academic_term_id UUID                  NOT NULL,
    class_id         UUID                  NULL,
    status           cbc_enrollment_status NOT NULL DEFAULT 'ACTIVE',
    created_at       TIMESTAMPTZ           NOT NULL DEFAULT NOW(),
    updated_at       TIMESTAMPTZ           NOT NULL DEFAULT NOW(),

    CONSTRAINT fk_enrollments_tenant_student
        FOREIGN KEY (tenant_id, student_id)
        REFERENCES cbc_students(tenant_id, id) ON DELETE CASCADE,
    CONSTRAINT fk_enrollments_tenant_school
        FOREIGN KEY (tenant_id, school_id)
        REFERENCES cbc_schools(tenant_id, id) ON DELETE CASCADE,
    CONSTRAINT fk_enrollments_tenant_school_term
        FOREIGN KEY (tenant_id, school_id, academic_term_id)
        REFERENCES academic_terms(tenant_id, school_id, id) ON DELETE CASCADE,
    -- Data-detachment intent: when class_id is set to NULL (mid-term removal),
    -- cbc_attendance_logs rows are preserved. The FK uses ON DELETE SET NULL
    -- so that student attendance history is never cascaded away.
    -- NOTE: class_id going NULL leaves tenant_id set; the composite FK is then
    -- skipped by Postgres (any NULL in the key = no FK check). The simple
    -- school→class cascade on cbc_classes handles the referential side.
    CONSTRAINT fk_enrollments_tenant_class
        FOREIGN KEY (tenant_id, class_id)
        REFERENCES cbc_classes(tenant_id, id) ON DELETE SET NULL,
    CONSTRAINT unique_student_term_enrollment UNIQUE (student_id, academic_term_id)
);

CREATE INDEX IF NOT EXISTS idx_cbc_enrollments_tenant_id  ON cbc_student_enrollments (tenant_id);
CREATE INDEX IF NOT EXISTS idx_cbc_enrollments_student_id ON cbc_student_enrollments (student_id);
CREATE INDEX IF NOT EXISTS idx_cbc_enrollments_school_id  ON cbc_student_enrollments (school_id);
CREATE INDEX IF NOT EXISTS idx_cbc_enrollments_term_id    ON cbc_student_enrollments (academic_term_id);
CREATE INDEX IF NOT EXISTS idx_cbc_enrollments_class_id   ON cbc_student_enrollments (class_id);

DROP TRIGGER IF EXISTS trg_cbc_student_enrollments_updated_at ON cbc_student_enrollments;
CREATE TRIGGER trg_cbc_student_enrollments_updated_at
    BEFORE UPDATE ON cbc_student_enrollments
    FOR EACH ROW EXECUTE FUNCTION fn_set_updated_at();

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
    id                     UUID    PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id              UUID    NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    student_id             UUID    UNIQUE NOT NULL REFERENCES cbc_students(id) ON DELETE CASCADE,
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

    CONSTRAINT fk_fee_categories_tenant_school
        FOREIGN KEY (tenant_id, school_id)
        REFERENCES cbc_schools(tenant_id, id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_fee_categories_tenant    ON fee_categories (tenant_id);
CREATE INDEX IF NOT EXISTS idx_fee_categories_school_id ON fee_categories (school_id);

-- ---------------------------------------------------------------------------
-- FEE TEMPLATES
-- ---------------------------------------------------------------------------

CREATE TABLE IF NOT EXISTS fee_templates (
    id               UUID             PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id        UUID             NOT NULL,
    school_id        UUID             NOT NULL,
    academic_term_id UUID             NOT NULL,
    grade_level      cbc_grade_level  NOT NULL,
    fee_category_id  UUID             NOT NULL REFERENCES fee_categories(id) ON DELETE CASCADE,
    amount           NUMERIC(12,2)    NOT NULL CHECK (amount >= 0),

    CONSTRAINT fk_fee_templates_tenant_school
        FOREIGN KEY (tenant_id, school_id)
        REFERENCES cbc_schools(tenant_id, id) ON DELETE CASCADE,
    CONSTRAINT fk_fee_templates_tenant_term
        FOREIGN KEY (tenant_id, school_id, academic_term_id)
        REFERENCES academic_terms(tenant_id, school_id, id) ON DELETE CASCADE,
    CONSTRAINT unique_fee_template_rule
        UNIQUE (academic_term_id, grade_level, fee_category_id)
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
    CONSTRAINT fk_invoices_tenant_student
        FOREIGN KEY (tenant_id, student_id)
        REFERENCES cbc_students(tenant_id, id) ON DELETE CASCADE,
    CONSTRAINT fk_invoices_tenant_school
        FOREIGN KEY (tenant_id, school_id)
        REFERENCES cbc_schools(tenant_id, id) ON DELETE CASCADE,
    CONSTRAINT fk_invoices_tenant_term
        FOREIGN KEY (tenant_id, school_id, academic_term_id)
        REFERENCES academic_terms(tenant_id, school_id, id) ON DELETE CASCADE,
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

    CONSTRAINT fk_invoice_items_tenant_invoice
        FOREIGN KEY (tenant_id, invoice_id)
        REFERENCES invoices(tenant_id, id) ON DELETE CASCADE
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

    CONSTRAINT fk_payments_tenant_invoice
        FOREIGN KEY (tenant_id, invoice_id)
        REFERENCES invoices(tenant_id, id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_payments_tenant     ON payments (tenant_id);
CREATE INDEX IF NOT EXISTS idx_payments_invoice_id ON payments (invoice_id);
CREATE INDEX IF NOT EXISTS idx_payments_parent     ON payments (parent_id);
-- IMPROVE: M-Pesa reconciliation lookups by reference_code; partial keeps index small
CREATE UNIQUE INDEX IF NOT EXISTS idx_payments_reference_code
    ON payments (reference_code) WHERE reference_code IS NOT NULL;

-- ============================================================
-- TRIGGER: Sync invoice payment_status and amount_paid
-- BUG FIX: Split into 3 separate functions so each trigger only accesses the
--          transition table(s) available to it. The original single function
--          referenced both inserted_rows and deleted_rows regardless of the
--          trigger event, which would fail at runtime for INSERT and DELETE.
-- ============================================================

CREATE OR REPLACE FUNCTION fn_sync_invoice_payment_status_insert()
RETURNS TRIGGER AS $$
BEGIN
    UPDATE invoices i
    SET
        amount_paid    = COALESCE(p.total_paid, 0),
        payment_status = CASE
            WHEN i.payment_status = 'WAIVED'               THEN 'WAIVED'
            WHEN COALESCE(p.total_paid, 0) = 0             THEN 'UNPAID'
            WHEN COALESCE(p.total_paid, 0) >= i.amount_due THEN 'PAID'
            ELSE 'PARTIAL'
        END
    FROM (SELECT DISTINCT invoice_id FROM inserted_rows) ai
    LEFT JOIN (
        SELECT invoice_id, SUM(amount) AS total_paid
        FROM payments
        GROUP BY invoice_id
    ) p ON p.invoice_id = ai.invoice_id
    WHERE i.id = ai.invoice_id;

    RETURN NULL;
END;
$$ LANGUAGE plpgsql;

CREATE OR REPLACE FUNCTION fn_sync_invoice_payment_status_delete()
RETURNS TRIGGER AS $$
BEGIN
    UPDATE invoices i
    SET
        amount_paid    = COALESCE(p.total_paid, 0),
        payment_status = CASE
            WHEN i.payment_status = 'WAIVED'               THEN 'WAIVED'
            WHEN COALESCE(p.total_paid, 0) = 0             THEN 'UNPAID'
            WHEN COALESCE(p.total_paid, 0) >= i.amount_due THEN 'PAID'
            ELSE 'PARTIAL'
        END
    FROM (SELECT DISTINCT invoice_id FROM deleted_rows) ai
    LEFT JOIN (
        SELECT invoice_id, SUM(amount) AS total_paid
        FROM payments
        GROUP BY invoice_id
    ) p ON p.invoice_id = ai.invoice_id
    WHERE i.id = ai.invoice_id;

    RETURN NULL;
END;
$$ LANGUAGE plpgsql;

CREATE OR REPLACE FUNCTION fn_sync_invoice_payment_status_update()
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
            WHEN i.payment_status = 'WAIVED'               THEN 'WAIVED'
            WHEN COALESCE(p.total_paid, 0) = 0             THEN 'UNPAID'
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
    EXECUTE FUNCTION fn_sync_invoice_payment_status_insert();

-- Fires on DELETE
CREATE TRIGGER trg_sync_invoice_payment_status_delete
    AFTER DELETE ON payments
    REFERENCING OLD TABLE AS deleted_rows
    FOR EACH STATEMENT
    EXECUTE FUNCTION fn_sync_invoice_payment_status_delete();

-- Fires on UPDATE
CREATE TRIGGER trg_sync_invoice_payment_status_update
    AFTER UPDATE ON payments
    REFERENCING NEW TABLE AS inserted_rows OLD TABLE AS deleted_rows
    FOR EACH STATEMENT
    EXECUTE FUNCTION fn_sync_invoice_payment_status_update();

-- ============================================================================
-- LAYER 5 — CBC CURRICULUM STRUCTURE
-- BUG FIX: Moved from after Layer 6 to here. cbc_class_teachers,
--          cbc_attendance_periods, and cbc_timetable_slots all FK-reference
--          cbc_learning_areas; they must be created after it.
-- ============================================================================

CREATE TABLE IF NOT EXISTS cbc_learning_areas (
    id              UUID                NOT NULL DEFAULT gen_random_uuid(),
    tenant_id       UUID                NOT NULL,
    school_id       UUID                NOT NULL,
    name            VARCHAR(150)        NOT NULL,
    code            VARCHAR(50)         NOT NULL,
    education_level cbc_education_level NOT NULL,

    PRIMARY KEY (id),
    CONSTRAINT fk_cbc_learning_areas_tenant_school
        FOREIGN KEY (tenant_id, school_id)
        REFERENCES cbc_schools(tenant_id, id) ON DELETE CASCADE,
    CONSTRAINT uq_cbc_learning_area_school_code UNIQUE (tenant_id, school_id, code),
    -- IMPROVE: expose (tenant_id, id) pair so downstream tables can composite-FK
    CONSTRAINT uq_cbc_learning_areas_tenant UNIQUE (tenant_id, id)
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
-- PERFORMANCE INDICATORS
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
    id               UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id        UUID         NOT NULL,
    class_id         UUID         NOT NULL,
    user_id          UUID         NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    learning_area_id UUID         NULL REFERENCES cbc_learning_areas(id) ON DELETE SET NULL,
    teacher_role     teacher_role NOT NULL DEFAULT 'SUBJECT_TEACHER',
    created_at       TIMESTAMPTZ  NOT NULL DEFAULT NOW(),

    CONSTRAINT fk_cbc_class_teachers_tenant_class
        FOREIGN KEY (tenant_id, class_id)
        REFERENCES cbc_classes(tenant_id, id) ON DELETE CASCADE,
    CONSTRAINT chk_cct_primary_no_area CHECK (
        teacher_role != 'PRIMARY_CLASS_TEACHER' OR learning_area_id IS NULL
    ),
    CONSTRAINT chk_cct_subject_area_required CHECK (
        teacher_role != 'SUBJECT_TEACHER' OR learning_area_id IS NOT NULL
    ),
    CONSTRAINT unique_cbc_class_teacher UNIQUE (class_id, user_id, learning_area_id)
);

CREATE INDEX IF NOT EXISTS idx_cbc_class_teachers_tenant   ON cbc_class_teachers (tenant_id);
CREATE INDEX IF NOT EXISTS idx_cbc_class_teachers_class_id ON cbc_class_teachers (class_id);
CREATE INDEX IF NOT EXISTS idx_cbc_class_teachers_user_id  ON cbc_class_teachers (user_id);
CREATE INDEX IF NOT EXISTS idx_cbc_class_teachers_role     ON cbc_class_teachers (teacher_role);

-- Only one PRIMARY_CLASS_TEACHER per class
CREATE UNIQUE INDEX IF NOT EXISTS idx_cbc_one_primary_per_class
    ON cbc_class_teachers (class_id)
    WHERE teacher_role = 'PRIMARY_CLASS_TEACHER';

-- ---------------------------------------------------------------------------
-- CBC ATTENDANCE PERIODS
-- ---------------------------------------------------------------------------

CREATE TABLE IF NOT EXISTS cbc_attendance_periods (
    id                   UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id            UUID         NOT NULL,
    school_id            UUID         NOT NULL,
    academic_term_id     UUID         NOT NULL,
    class_id             UUID         NOT NULL,
    cbc_learning_area_id UUID         NOT NULL REFERENCES cbc_learning_areas(id) ON DELETE CASCADE,
    date_recorded        DATE         NOT NULL,
    recorded_by          UUID         NOT NULL REFERENCES users(id),
    authorized_by_role   teacher_role NULL,
    created_at           TIMESTAMPTZ  NOT NULL DEFAULT NOW(),

    CONSTRAINT uq_cbc_attendance_periods_tenant UNIQUE (tenant_id, id),
    CONSTRAINT fk_cbc_att_periods_tenant_school
        FOREIGN KEY (tenant_id, school_id)
        REFERENCES cbc_schools(tenant_id, id) ON DELETE CASCADE,
    CONSTRAINT fk_cbc_att_periods_tenant_term
        FOREIGN KEY (tenant_id, school_id, academic_term_id)
        REFERENCES academic_terms(tenant_id, school_id, id) ON DELETE CASCADE,
    CONSTRAINT fk_cbc_att_periods_tenant_class
        FOREIGN KEY (tenant_id, class_id)
        REFERENCES cbc_classes(tenant_id, id) ON DELETE CASCADE
);

CREATE INDEX        IF NOT EXISTS idx_cbc_att_periods_tenant     ON cbc_attendance_periods (tenant_id);
CREATE INDEX        IF NOT EXISTS idx_cbc_att_periods_class_date ON cbc_attendance_periods (class_id, date_recorded);
CREATE UNIQUE INDEX IF NOT EXISTS idx_cbc_unique_attendance_period
    ON cbc_attendance_periods (class_id, date_recorded, cbc_learning_area_id);

COMMENT ON COLUMN cbc_attendance_periods.authorized_by_role IS
    'The teacher_role that authorised this attendance period record. Populated
     when a period recorded by a SUBSTITUTE_TEACHER is counter-signed by a
     PRIMARY_CLASS_TEACHER. NULL means the recording teacher is also the
     authorising teacher (normal case). Used for audit and KNEC compliance
     reporting where substitute attendance requires authorisation.';

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

    CONSTRAINT fk_cbc_att_logs_tenant_period
        FOREIGN KEY (tenant_id, cbc_attendance_period_id)
        REFERENCES cbc_attendance_periods(tenant_id, id) ON DELETE CASCADE,
    CONSTRAINT fk_cbc_att_logs_tenant_student
        FOREIGN KEY (tenant_id, student_id)
        REFERENCES cbc_students(tenant_id, id) ON DELETE CASCADE,
    CONSTRAINT unique_cbc_student_attendance_period UNIQUE (cbc_attendance_period_id, student_id)
);

CREATE INDEX IF NOT EXISTS idx_cbc_att_logs_tenant  ON cbc_attendance_logs (tenant_id);
CREATE INDEX IF NOT EXISTS idx_cbc_att_logs_period  ON cbc_attendance_logs (cbc_attendance_period_id);
CREATE INDEX IF NOT EXISTS idx_cbc_att_logs_student ON cbc_attendance_logs (student_id);

-- ---------------------------------------------------------------------------
-- CBC TIMETABLE SLOTS
-- IMPROVE: added CHECK (end_time > start_time) — GiST exclusion prevents
--          overlaps but nothing previously blocked end <= start on the row itself
-- ---------------------------------------------------------------------------

CREATE TABLE IF NOT EXISTS cbc_timetable_slots (
    id                   UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id            UUID        NOT NULL,
    school_id            UUID        NOT NULL,
    academic_year_id     UUID        NOT NULL,
    academic_term_id     UUID        NOT NULL,
    class_id             UUID        NOT NULL,
    teacher_id           UUID        NOT NULL,
    cbc_learning_area_id UUID        NULL REFERENCES cbc_learning_areas(id) ON DELETE SET NULL,
    room_identifier      VARCHAR(50) NULL,
    day_of_week          INT         NOT NULL CHECK (day_of_week BETWEEN 1 AND 7),
    start_time           TIME        NOT NULL,
    end_time             TIME        NOT NULL,

    CONSTRAINT chk_timetable_times CHECK (end_time > start_time),
    CONSTRAINT fk_cbc_timetable_tenant_school
        FOREIGN KEY (tenant_id, school_id)
        REFERENCES cbc_schools(tenant_id, id) ON DELETE CASCADE,
    CONSTRAINT fk_cbc_timetable_tenant_year
        FOREIGN KEY (tenant_id, academic_year_id)
        REFERENCES academic_years(tenant_id, id) ON DELETE CASCADE,
    CONSTRAINT fk_cbc_timetable_tenant_term
        FOREIGN KEY (tenant_id, school_id, academic_term_id)
        REFERENCES academic_terms(tenant_id, school_id, id) ON DELETE CASCADE,
    CONSTRAINT fk_cbc_timetable_tenant_class
        FOREIGN KEY (tenant_id, class_id)
        REFERENCES cbc_classes(tenant_id, id) ON DELETE CASCADE,
    CONSTRAINT fk_cbc_timetable_tenant_teacher
        FOREIGN KEY (tenant_id, teacher_id)
        REFERENCES users(tenant_id, id) ON DELETE CASCADE
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

-- ---------------------------------------------------------------------------
-- AUTO-REGISTER / AUTO-CLEAN SUBJECT TEACHER TRIGGER
-- IMPROVE: Extended to also clean up stale SUBJECT_TEACHER registrations when
--          a timetable slot's teacher_id or cbc_learning_area_id changes. The
--          original only inserted, leaving ghost assignments on UPDATE.
-- ---------------------------------------------------------------------------

CREATE OR REPLACE FUNCTION fn_auto_register_subject_teacher()
RETURNS TRIGGER AS $$
BEGIN
    -- On UPDATE: remove the old SUBJECT_TEACHER registration if teacher or
    -- learning area has changed, to avoid ghost assignments.
    IF TG_OP = 'UPDATE' THEN
        IF (OLD.teacher_id IS DISTINCT FROM NEW.teacher_id OR
            OLD.cbc_learning_area_id IS DISTINCT FROM NEW.cbc_learning_area_id) AND
            OLD.cbc_learning_area_id IS NOT NULL
        THEN
            -- Only remove if this slot was the sole reason for the assignment
            -- (i.e. no other active slot ties this teacher+class+area together).
            DELETE FROM cbc_class_teachers
            WHERE tenant_id        = OLD.tenant_id
              AND class_id         = OLD.class_id
              AND user_id          = OLD.teacher_id
              AND learning_area_id = OLD.cbc_learning_area_id
              AND teacher_role     = 'SUBJECT_TEACHER'
              AND NOT EXISTS (
                  SELECT 1 FROM cbc_timetable_slots
                  WHERE tenant_id            = OLD.tenant_id
                    AND class_id             = OLD.class_id
                    AND teacher_id           = OLD.teacher_id
                    AND cbc_learning_area_id = OLD.cbc_learning_area_id
                    AND id                  != OLD.id   -- exclude the row being updated
              );
        END IF;
    END IF;

    -- Insert new SUBJECT_TEACHER registration when a learning area is set
    IF NEW.cbc_learning_area_id IS NOT NULL THEN
        IF NOT EXISTS (
            SELECT 1 FROM cbc_class_teachers
            WHERE tenant_id = NEW.tenant_id
              AND class_id  = NEW.class_id
              AND user_id   = NEW.teacher_id
        ) THEN
            INSERT INTO cbc_class_teachers (tenant_id, class_id, user_id, learning_area_id, teacher_role)
            VALUES (NEW.tenant_id, NEW.class_id, NEW.teacher_id, NEW.cbc_learning_area_id, 'SUBJECT_TEACHER');
        END IF;
    END IF;

    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS trg_auto_register_subject_teacher ON cbc_timetable_slots;
CREATE TRIGGER trg_auto_register_subject_teacher
    AFTER INSERT OR UPDATE OF teacher_id, cbc_learning_area_id ON cbc_timetable_slots
    FOR EACH ROW
    EXECUTE FUNCTION fn_auto_register_subject_teacher();

-- ============================================================================
-- LAYER 7 — CBC ASSESSMENT ARCHITECTURE
-- ============================================================================

-- ---------------------------------------------------------------------------
-- ASSESSMENT WEIGHT CONFIGS
-- ---------------------------------------------------------------------------

CREATE TABLE IF NOT EXISTS assessment_weight_configs (
    id                   UUID                NOT NULL DEFAULT gen_random_uuid(),
    grade_level          cbc_grade_level     NOT NULL,
    assessment_type_code cbc_assessment_type NOT NULL,
    target_exam          knec_target_exam    NOT NULL,
    weight_percent       NUMERIC(5,2)        NOT NULL,
    effective_from       SMALLINT            NOT NULL,
    notes                TEXT                NULL,

    PRIMARY KEY (id),
    CONSTRAINT chk_awc_weight_percent    CHECK (weight_percent BETWEEN 0.00 AND 100.00),
    CONSTRAINT chk_awc_effective_from    CHECK (effective_from >= 2017),
    CONSTRAINT uq_awc_grade_type_exam_effective
        UNIQUE (grade_level, assessment_type_code, target_exam, effective_from)
);

CREATE INDEX IF NOT EXISTS idx_awc_grade_exam ON assessment_weight_configs (grade_level, target_exam);

COMMENT ON TABLE assessment_weight_configs IS
    'Official KNEC weighting formula per grade per assessment type. Seeded with
     the published KNEC formula. KPSEA: 60% SBA (G4+G5) + 40% KPSEA written (G6).
     KJSEA: 20% SBA (G7+G8) + 20% KPSEA result + 60% KJSEA written (G9).
     This table is intentionally global (no tenant_id): KNEC weights are
     nationally mandated and do not vary per school. Schema changes would be
     required if per-school overrides are ever needed.';

-- ---------------------------------------------------------------------------
-- ASSESSMENT BLUEPRINTS
-- IMPROVE: added unique constraint to prevent duplicate blueprints for the
--          same school/grade/term combination
-- ---------------------------------------------------------------------------

CREATE TABLE IF NOT EXISTS assessment_blueprints (
    id            UUID                NOT NULL DEFAULT gen_random_uuid(),
    tenant_id     UUID                NOT NULL,
    school_id     UUID                NOT NULL,
    title         VARCHAR(255)        NOT NULL,
    type          cbc_assessment_type NOT NULL,
    grade_level   cbc_grade_level     NOT NULL,
    academic_year SMALLINT            NOT NULL,
    term          SMALLINT            NOT NULL,

    PRIMARY KEY (id),
    CONSTRAINT fk_blueprints_tenant_school
        FOREIGN KEY (tenant_id, school_id)
        REFERENCES cbc_schools(tenant_id, id) ON DELETE CASCADE,
    CONSTRAINT chk_blueprint_term          CHECK (term BETWEEN 1 AND 3),
    CONSTRAINT chk_blueprint_academic_year CHECK (academic_year >= 2017),
    CONSTRAINT uq_blueprint_per_school_grade_term
        UNIQUE (school_id, title, type, grade_level, academic_year, term)
);

CREATE INDEX IF NOT EXISTS idx_blueprints_tenant     ON assessment_blueprints (tenant_id);
CREATE INDEX IF NOT EXISTS idx_blueprints_school     ON assessment_blueprints (school_id);
CREATE INDEX IF NOT EXISTS idx_blueprints_grade_year ON assessment_blueprints (grade_level, academic_year, type);

-- ---------------------------------------------------------------------------
-- ASSESSMENT BLUEPRINT INDICATORS
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
-- ASSESSMENT SESSIONS
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

    CONSTRAINT fk_asessions_tenant_class
        FOREIGN KEY (tenant_id, class_id)
        REFERENCES cbc_classes(tenant_id, id) ON DELETE RESTRICT
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
-- LEARNER RUBRIC RESULTS
-- IMPROVE: added CHECK (raw_score >= 0) — NUMERIC(5,2) previously allowed
--          negative marks; added index on (session_id, student_id) for the
--          most common access pattern (all results for a student in a session)
-- ---------------------------------------------------------------------------

CREATE TABLE IF NOT EXISTS learner_rubric_results (
    id                        UUID           PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id                 UUID           NOT NULL,
    session_id                UUID           NOT NULL REFERENCES assessment_sessions(id) ON DELETE CASCADE,
    student_id                UUID           NOT NULL,
    indicator_id              UUID           NOT NULL REFERENCES performance_indicators(id) ON DELETE RESTRICT,
    score_type                lrr_score_type NOT NULL,
    raw_score                 NUMERIC(5,2)   NULL CHECK (raw_score >= 0),
    rubric_level              cbc_rubric_level NOT NULL,
    teacher_observation_notes TEXT           NULL,

    CONSTRAINT fk_lrr_tenant_student
        FOREIGN KEY (tenant_id, student_id)
        REFERENCES cbc_students(tenant_id, id) ON DELETE CASCADE,
    CONSTRAINT unique_lrr_per_student_indicator UNIQUE (session_id, student_id, indicator_id)
);

CREATE INDEX IF NOT EXISTS idx_lrr_tenant            ON learner_rubric_results (tenant_id);
CREATE INDEX IF NOT EXISTS idx_lrr_session           ON learner_rubric_results (session_id);
-- IMPROVE: fast fetch of all results for a student within a session
CREATE INDEX IF NOT EXISTS idx_lrr_session_student   ON learner_rubric_results (session_id, student_id);
CREATE INDEX IF NOT EXISTS idx_lrr_student_indicator ON learner_rubric_results (student_id, indicator_id);
CREATE INDEX IF NOT EXISTS idx_lrr_indicator         ON learner_rubric_results (indicator_id);

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
-- LEARNER PORTFOLIOS
-- ---------------------------------------------------------------------------

CREATE TABLE IF NOT EXISTS learner_portfolios (
    id               UUID                    PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id        UUID                    NOT NULL,
    student_id       UUID                    NOT NULL,
    sub_strand_id    UUID                    NOT NULL REFERENCES cbc_sub_strands(id) ON DELETE RESTRICT,
    evidence_type    portfolio_evidence_type NOT NULL,
    storage_pointer  TEXT                    NOT NULL,
    linked_result_id UUID                    NULL REFERENCES learner_rubric_results(id) ON DELETE SET NULL,
    date_collected   DATE                    NULL,
    created_at       TIMESTAMPTZ             NOT NULL DEFAULT NOW(),

    CONSTRAINT fk_portfolios_tenant_student
        FOREIGN KEY (tenant_id, student_id)
        REFERENCES cbc_students(tenant_id, id) ON DELETE CASCADE
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
-- CBC TERM COMPETENCY SUMMARIES
-- IMPROVE: added index on (student_id, learning_area_id) for fetching all
--          terms in a student's subject history
-- ---------------------------------------------------------------------------

CREATE TABLE IF NOT EXISTS cbc_term_competency_summaries (
    id               UUID                             PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id        UUID                             NOT NULL,
    student_id       UUID                             NOT NULL,
    learning_area_id UUID                             NOT NULL REFERENCES cbc_learning_areas(id) ON DELETE RESTRICT,
    class_id         UUID                             NOT NULL,
    academic_year    SMALLINT                         NOT NULL,
    term             SMALLINT                         NOT NULL,
    calculated_level cbc_rubric_level_with_sub_levels NOT NULL,
    override_level   cbc_rubric_level_with_sub_levels NULL,
    final_level      cbc_rubric_level                 NOT NULL,
    knec_sync_status knec_sync_status                 NOT NULL DEFAULT 'Pending',
    knec_synced_at   TIMESTAMPTZ                      NULL,

    CONSTRAINT fk_summaries_tenant_student
        FOREIGN KEY (tenant_id, student_id)
        REFERENCES cbc_students(tenant_id, id) ON DELETE CASCADE,
    CONSTRAINT fk_summaries_tenant_class
        FOREIGN KEY (tenant_id, class_id)
        REFERENCES cbc_classes(tenant_id, id) ON DELETE RESTRICT,
    CONSTRAINT chk_summary_term         CHECK (term BETWEEN 1 AND 3),
    CONSTRAINT chk_summary_academic_year CHECK (academic_year >= 2017),
    CONSTRAINT unique_summary_per_student_area_term
        UNIQUE (student_id, learning_area_id, academic_year, term)
);

CREATE INDEX IF NOT EXISTS idx_summaries_tenant       ON cbc_term_competency_summaries (tenant_id);
CREATE INDEX IF NOT EXISTS idx_summaries_sync_batch   ON cbc_term_competency_summaries (academic_year, term, knec_sync_status);
CREATE INDEX IF NOT EXISTS idx_summaries_student_year ON cbc_term_competency_summaries (student_id, academic_year);
CREATE INDEX IF NOT EXISTS idx_summaries_class        ON cbc_term_competency_summaries (class_id);
-- IMPROVE: fast lookup of all terms for a student's subject history
CREATE INDEX IF NOT EXISTS idx_summaries_student_area ON cbc_term_competency_summaries (student_id, learning_area_id);

COMMENT ON TABLE cbc_term_competency_summaries IS
    'Definitive per-term competency record per learner per learning area.
     final_level is the KNEC portal submission value — must always be one of
     EE/ME/AE/BE. Sub-levels (EE1 etc.) are only valid for the internal
     calculated_level and override_level fields. knec_synced_at is NULL until
     the first successful upload to cba.knec.ac.ke.';

-- ---------------------------------------------------------------------------
-- SCHOOL MEMBER COUNTS
-- IMPROVE: added IF NOT EXISTS (was the only CREATE TABLE in the file missing it)
-- ---------------------------------------------------------------------------

CREATE TABLE IF NOT EXISTS school_member_counts (
    school_id  UUID      PRIMARY KEY REFERENCES cbc_schools(id) ON DELETE CASCADE,
    admins     INT       NOT NULL DEFAULT 0,
    teachers   INT       NOT NULL DEFAULT 0,
    nurses     INT       NOT NULL DEFAULT 0,
    finance    INT       NOT NULL DEFAULT 0,
    parents    INT       NOT NULL DEFAULT 0,
    students   INT       NOT NULL DEFAULT 0,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_school_member_counts ON school_member_counts (school_id);

-- ============================================================
-- TRIGGER: Sync school staff/parent counts from memberships
-- ============================================================

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
       AND m.is_active  = true
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
       AND m.is_active  = true
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
       AND m.is_active  = true
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
       AND st.is_active  = true
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
       AND st.is_active  = true
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
       AND st.is_active  = true
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

CREATE TABLE IF NOT EXISTS member_active_school (
    user_id     UUID        NOT NULL,
    tenant_id   UUID        NOT NULL,
    school_id   UUID        NOT NULL,
    switched_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    PRIMARY KEY (user_id),

    CONSTRAINT fk_mas_user
        FOREIGN KEY (user_id)              REFERENCES users(id)                       ON DELETE CASCADE,
    CONSTRAINT fk_mas_tenant_user
        FOREIGN KEY (tenant_id, user_id)   REFERENCES users(tenant_id, id)            ON DELETE CASCADE,
    CONSTRAINT fk_mas_tenant_school
        FOREIGN KEY (tenant_id, school_id) REFERENCES cbc_schools(tenant_id, id)      ON DELETE CASCADE,
    CONSTRAINT fk_mas_membership
        FOREIGN KEY (user_id, school_id)   REFERENCES memberships(user_id, school_id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_mas_tenant_id ON member_active_school (tenant_id);

COMMENT ON TABLE member_active_school IS
    'Tracks the currently active school context for each user within a tenant.
     One row per user. Upsert on school switch. The chosen school_id is
     constrained to schools the user is an active member of via fk_mas_membership.';

-- ============================================================================
-- END OF MIGRATION
-- ============================================================================

COMMIT;