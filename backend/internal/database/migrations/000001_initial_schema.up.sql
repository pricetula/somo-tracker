-- Migration: 000001_initial_schema
-- SomoTracker — Kenya CBC/CBE academic platform (CBC-only, v5)
-- Drops all generic education system abstractions.
-- Rebuilds as a purpose-built, single-system CBC schema.

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
        'cancelled',
        'completed_with_errors',
        'cancelling'
    );
EXCEPTION WHEN duplicate_object THEN NULL;
END $$;

DO $$ BEGIN
    CREATE TYPE import_job_type AS ENUM ('STAFF_INVITE', 'STUDENT_IMPORT', 'PARENT_INVITE');
EXCEPTION WHEN duplicate_object THEN NULL;
END $$;

DO $$ BEGIN
    CREATE TYPE import_staging_status AS ENUM ('pending', 'succeeded', 'failed');
EXCEPTION WHEN duplicate_object THEN NULL;
END $$;

DO $$ BEGIN
    CREATE TYPE import_chunk_status AS ENUM ('pending', 'processing', 'completed', 'cancelled');
EXCEPTION WHEN duplicate_object THEN NULL;
END $$;

DO $$ BEGIN
    CREATE TYPE block_type AS ENUM ('Lesson', 'Break', 'Assembly', 'ExtraCurricular');
EXCEPTION WHEN duplicate_object THEN NULL;
END $$;

DO $$ BEGIN
    CREATE TYPE import_failure_type AS ENUM (
        'SCHEMA_VALIDATION',
        'DATABASE_CONSTRAINT',
        'BUSINESS_RULE_VIOLATION',
        'DUPLICATE_EMAIL',
        'INVALID_EMAIL_FORMAT',
        'STYTCH_API_ERROR',
        'INVITATION_INSERT_FAILED'
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
-- IMPROVE: added created_at / updated_at and audit columns (version, created_by, updated_by)
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
    ON academic_years (school_id) WHERE is_current = TRUE;

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
    created_at       TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at       TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
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
        REFERENCES academic_years(tenant_id, id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_academic_terms_tenant_id ON academic_terms (tenant_id);
-- BUG FIX: was incorrectly targeting academic_years; fixed to academic_terms
CREATE INDEX IF NOT EXISTS idx_academic_terms_school_id ON academic_terms (school_id);
CREATE INDEX IF NOT EXISTS idx_academic_terms_year_id   ON academic_terms (academic_year_id);

CREATE UNIQUE INDEX IF NOT EXISTS idx_one_current_term_per_year
    ON academic_terms (academic_year_id) WHERE is_current = TRUE;

DROP TRIGGER IF EXISTS trg_academic_terms_updated_at ON academic_terms;
CREATE TRIGGER trg_academic_terms_updated_at
    BEFORE UPDATE ON academic_terms
    FOR EACH ROW EXECUTE FUNCTION fn_set_updated_at();

CREATE UNIQUE INDEX IF NOT EXISTS idx_unique_term_number_per_year
    ON academic_terms (academic_year_id, term_number);

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
    color      VARCHAR(50)  NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ  NOT NULL DEFAULT NOW(),

    CONSTRAINT fk_cbc_streams_school
        FOREIGN KEY (tenant_id, school_id) REFERENCES cbc_schools(tenant_id, id) ON DELETE CASCADE,

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
    'Composite FK (tenant_id, school_id) enforces tenant-scoped referential
     integrity. ON DELETE CASCADE — streams are removed when their school is
     deleted.';

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
    CONSTRAINT fk_memberships_tenant_user
        FOREIGN KEY (tenant_id, user_id)
        REFERENCES users(tenant_id, id),
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
    job_type             import_job_type   NOT NULL,
    role                 user_role         NULL,
    created_by           UUID              REFERENCES users(id) ON DELETE SET NULL,
    status               import_job_status NOT NULL DEFAULT 'pending',
    total_records        INT               NOT NULL DEFAULT 0,
    processed_records    INT               NOT NULL DEFAULT 0,
    success_count        INT               NOT NULL DEFAULT 0,
    failed_count         INT               NOT NULL DEFAULT 0,
    idempotency_key      TEXT              NULL,
    payload_hash         TEXT              NULL,
    total_chunks         INT               NOT NULL DEFAULT 0,
    processed_chunks     INT               NOT NULL DEFAULT 0,
    metadata             JSONB             NOT NULL DEFAULT '{}'::jsonb,
    created_at           TIMESTAMPTZ       NOT NULL DEFAULT NOW(),
    started_at           TIMESTAMPTZ       NULL,
    completed_at         TIMESTAMPTZ       NULL,
    last_progress_at    TIMESTAMPTZ       NULL,

    CONSTRAINT fk_import_jobs_tenant_school
        FOREIGN KEY (tenant_id, school_id)
        REFERENCES cbc_schools(tenant_id, id) ON DELETE CASCADE,
    CONSTRAINT chk_import_jobs_role_required_for_staff
        CHECK (
            (job_type IN ('STAFF_INVITE', 'PARENT_INVITE') AND role IS NOT NULL)
            OR (job_type NOT IN ('STAFF_INVITE', 'PARENT_INVITE'))
        )
);

CREATE INDEX IF NOT EXISTS idx_import_jobs_tenant_id  ON import_jobs (tenant_id);
CREATE INDEX IF NOT EXISTS idx_import_jobs_school_id  ON import_jobs (school_id);
CREATE INDEX IF NOT EXISTS idx_import_jobs_created_by ON import_jobs (created_by);
CREATE INDEX IF NOT EXISTS idx_import_jobs_status     ON import_jobs (status);
CREATE UNIQUE INDEX IF NOT EXISTS uq_import_jobs_tenant_idempotency
    ON import_jobs (tenant_id, idempotency_key)
    WHERE idempotency_key IS NOT NULL;

-- At most one active (processing or cancelling) import job per school at a time.
-- A new submission while one is active is rejected with import_already_in_progress.
CREATE UNIQUE INDEX IF NOT EXISTS uq_import_jobs_one_active_per_school
    ON import_jobs (school_id)
    WHERE status IN ('processing'::import_job_status, 'cancelling'::import_job_status);

-- ---------------------------------------------------------------------------
-- IMPORT JOB CHUNKS — Track chunk claim/redelivery safety
-- ---------------------------------------------------------------------------

CREATE TABLE IF NOT EXISTS import_job_chunks (
    id            UUID                PRIMARY KEY DEFAULT gen_random_uuid(),
    job_id        UUID                NOT NULL REFERENCES import_jobs(id) ON DELETE CASCADE,
    chunk_index   INT                 NOT NULL,
    status        import_chunk_status NOT NULL DEFAULT 'pending',
    row_start     INT                 NOT NULL DEFAULT 0,
    row_end       INT                 NOT NULL DEFAULT 0,
    claimed_at    TIMESTAMPTZ         NULL,
    completed_at  TIMESTAMPTZ         NULL,
    created_at    TIMESTAMPTZ         NOT NULL DEFAULT NOW(),

    CONSTRAINT uq_import_job_chunks_job_chunk UNIQUE (job_id, chunk_index)
);

CREATE INDEX IF NOT EXISTS idx_import_job_chunks_job_id ON import_job_chunks (job_id);
CREATE INDEX IF NOT EXISTS idx_import_job_chunks_status ON import_job_chunks (job_id, status);

-- ---------------------------------------------------------------------------
-- IMPORT JOB FAILURES
-- ---------------------------------------------------------------------------

CREATE TABLE IF NOT EXISTS import_job_failures (
    id            BIGSERIAL            PRIMARY KEY,
    import_job_id UUID                 NOT NULL REFERENCES import_jobs(id) ON DELETE CASCADE,
    raw_payload   JSONB                NOT NULL,
    error_message TEXT                 NOT NULL,
    error_type    import_failure_type  NOT NULL DEFAULT 'DATABASE_CONSTRAINT',
    row_number    INT                  NULL,
    created_at    TIMESTAMPTZ          NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_import_job_failures_job_id ON import_job_failures (import_job_id);

-- ---------------------------------------------------------------------------
-- IMPORT JOB STAGING — Student bulk import staging rows
-- ---------------------------------------------------------------------------

CREATE TABLE IF NOT EXISTS import_job_staging (
    id            UUID                  PRIMARY KEY DEFAULT gen_random_uuid(),
    job_id        UUID                  NOT NULL REFERENCES import_jobs(id) ON DELETE CASCADE,
    tenant_id     UUID                  NOT NULL,
    school_id     UUID                  NOT NULL,
    row_number    INT                   NOT NULL,
    raw_data      JSONB                 NOT NULL,
    status        import_staging_status NOT NULL DEFAULT 'pending',
    processed_at  TIMESTAMPTZ           NULL,
    created_at    TIMESTAMPTZ           NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_import_job_staging_job_id ON import_job_staging (job_id);
CREATE UNIQUE INDEX IF NOT EXISTS uq_import_job_staging_job_row
    ON import_job_staging (job_id, row_number);

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

-- Prevents race conditions where two concurrent chunks try to invite
-- the same email for the same school.
CREATE UNIQUE INDEX IF NOT EXISTS uq_invitations_school_email_pending
    ON invitations (school_id, email)
    WHERE status = 'pending';

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
    school_id              UUID                 NOT NULL REFERENCES cbc_schools(id) ON DELETE CASCADE,
    full_name              VARCHAR(255)         NOT NULL,
    gender                 gender_type          NOT NULL,
    date_of_birth          DATE                 NULL,
    upi_number             VARCHAR(20)          NULL,
    knec_assessment_number VARCHAR(15)          NULL,
    admission_number       VARCHAR(20)          NULL,
    learning_pathway       cbc_learning_pathway NOT NULL DEFAULT 'Age_Based',
    staging_row_id         UUID                 NULL REFERENCES import_job_staging(id) ON DELETE SET NULL,
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
CREATE UNIQUE INDEX IF NOT EXISTS idx_cbc_students_school_staging_row
    ON cbc_students (school_id, staging_row_id) WHERE staging_row_id IS NOT NULL;
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
    -- student attendance records (if any) are preserved via ON DELETE SET NULL
    -- so that history is never cascaded away.
    -- NOTE: class_id going NULL leaves tenant_id set; the composite FK is then
    -- skipped by Postgres (any NULL in the key = no FK check). The simple
    -- school→class cascade on cbc_classes handles the referential side.
    CONSTRAINT fk_enrollments_tenant_class
        FOREIGN KEY (tenant_id, class_id)
        REFERENCES cbc_classes(tenant_id, id) ON DELETE SET NULL,
    CONSTRAINT unique_student_term_enrollment UNIQUE (student_id, school_id, academic_term_id)
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
-- BUG FIX: Moved from after Layer 6 to here. cbc_class_teachers and
--          cbc_timetable_slots both FK-reference
--          cbc_learning_areas; they must be created after it.
-- ============================================================================

CREATE TABLE IF NOT EXISTS cbc_learning_areas (
    id              UUID                NOT NULL DEFAULT gen_random_uuid(),
    tenant_id       UUID                NOT NULL,
    school_id       UUID                NOT NULL,
    name            VARCHAR(150)        NOT NULL,
    code            VARCHAR(50)         NOT NULL,
    education_level cbc_education_level NOT NULL,
    grade_level     cbc_grade_level     NOT NULL,

    PRIMARY KEY (id),
    CONSTRAINT fk_cbc_learning_areas_tenant_school
        FOREIGN KEY (tenant_id, school_id)
        REFERENCES cbc_schools(tenant_id, id) ON DELETE CASCADE,
    CONSTRAINT uq_cbc_learning_areas_school_code_grade
        UNIQUE (tenant_id, school_id, code, grade_level),
    -- IMPROVE: expose (tenant_id, id) pair so downstream tables can composite-FK
    CONSTRAINT uq_cbc_learning_areas_tenant UNIQUE (tenant_id, id)
);

CREATE INDEX IF NOT EXISTS idx_cbc_learning_areas_tenant          ON cbc_learning_areas (tenant_id);
CREATE INDEX IF NOT EXISTS idx_cbc_learning_areas_school_id       ON cbc_learning_areas (school_id);
CREATE INDEX IF NOT EXISTS idx_cbc_learning_areas_education_level ON cbc_learning_areas (education_level);
CREATE INDEX IF NOT EXISTS idx_cbc_learning_areas_grade_level     ON cbc_learning_areas (grade_level);

COMMENT ON COLUMN cbc_learning_areas.education_level IS
    'The CBC tier this learning area belongs to, per KICD curriculum structure.
     Determines applicable KNEC assessment instruments and portal upload eligibility.';

COMMENT ON COLUMN cbc_learning_areas.code IS
    'Short KICD-defined code for this learning area, e.g. MATH, ENG, KISW,
     INT_SCI, PRE_TECH, SOC_STD. Unique within a school per grade level.';

COMMENT ON COLUMN cbc_learning_areas.grade_level IS
    'The specific CBC grade this learning area instance is taught in.
     Each grade has its own set of learning areas per KNEC/KICD curriculum.
     Combined with code to uniquely identify a learning area within a school.';

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
-- LAYER 6 — TEACHER ASSIGNMENTS & TIMETABLE
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

    CONSTRAINT fk_cbc_class_teachers_tenant_user
        FOREIGN KEY (tenant_id, user_id)
        REFERENCES users(tenant_id, id) ON DELETE CASCADE,
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
-- TIMETABLE STRUCTURES (Grid Definition Layer — Master Templates)
-- Holds time ranges and rules. Decoupled from allocation slots.
-- ---------------------------------------------------------------------------

CREATE TABLE IF NOT EXISTS timetable_structures (
    id               UUID             PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id        UUID             NOT NULL,
    school_id        UUID             NOT NULL,
    academic_year_id UUID             NOT NULL,
    day_of_week      INT              NOT NULL CHECK (day_of_week BETWEEN 1 AND 7),
    period_name      VARCHAR(50)      NOT NULL,
    start_time       TIME             NOT NULL,
    end_time         TIME             NOT NULL,
    is_break         BOOLEAN          NOT NULL DEFAULT FALSE,
    created_at       TIMESTAMPTZ      NOT NULL DEFAULT NOW(),
    updated_at       TIMESTAMPTZ      NOT NULL DEFAULT NOW(),

    CONSTRAINT chk_timetable_structure_times CHECK (end_time > start_time),
    CONSTRAINT excl_timetable_structure_overlap
        EXCLUDE USING gist (
            school_id WITH =,
            academic_year_id WITH =,
            day_of_week WITH =,
            fn_timerange(day_of_week, start_time, end_time) WITH &&
        ),
    CONSTRAINT fk_timetable_structure_tenant_school
        FOREIGN KEY (tenant_id, school_id)
        REFERENCES cbc_schools(tenant_id, id) ON DELETE CASCADE,
    CONSTRAINT fk_timetable_structure_academic_year
        FOREIGN KEY (tenant_id, academic_year_id)
        REFERENCES academic_years(tenant_id, id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_timetable_structure_tenant
    ON timetable_structures (tenant_id);
CREATE INDEX IF NOT EXISTS idx_timetable_structure_school_day
    ON timetable_structures (school_id, day_of_week);
CREATE INDEX IF NOT EXISTS idx_timetable_structure_academic_year
    ON timetable_structures (academic_year_id);

DROP TRIGGER IF EXISTS trg_timetable_structures_updated_at ON timetable_structures;
CREATE TRIGGER trg_timetable_structures_updated_at
    BEFORE UPDATE ON timetable_structures
    FOR EACH ROW EXECUTE FUNCTION fn_set_updated_at();

COMMENT ON TABLE timetable_structures IS
    'Structural day template (Grid Definition Layer). Defines the partitioned time
     blocks (lessons, breaks, assemblies) that make up a standard school day per
     academic year. The GiST exclusion constraint guarantees non-overlapping blocks
     per school per academic year per day. Decoupled from cbc_timetable_slots —
     allocations reference structure_id instead of carrying raw time ranges.';

COMMENT ON COLUMN timetable_structures.day_of_week IS
    '1=Monday, 2=Tuesday, 3=Wednesday, 4=Thursday, 5=Friday, 6=Saturday, 7=Sunday.
     Most schools use Mon-Fri (1-5); weekends are allowed for special sessions.';

COMMENT ON COLUMN timetable_structures.period_name IS
    'Human-readable name for this time period, e.g. "Lesson 1", "Morning Break",
     "Recess", "Assembly". Free-text — not an enum, to support school-specific naming.';

COMMENT ON COLUMN timetable_structures.is_break IS
    'Flags recess, lunch, or other non-instructional blocks. UI can use this to
     disable assignment cells and render break rows in a distinct style.';

-- ---------------------------------------------------------------------------
-- CBC TIMETABLE SLOTS (Grid Allocation Layer — Lightweight Assignments)
-- A lightweight relational mapping table using fast B-Tree composite unique
-- constraints instead of GiST exclusion constraints. The grid definition
-- (time ranges) lives in timetable_structures; this table only stores
-- assignments of class → teacher → learning_area → room per structure block.
-- ---------------------------------------------------------------------------

CREATE TABLE IF NOT EXISTS cbc_timetable_slots (
    id                UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id         UUID        NOT NULL,
    school_id         UUID        NOT NULL,
    academic_year_id  UUID        NOT NULL,
    structure_id      UUID        NOT NULL REFERENCES timetable_structures(id) ON DELETE CASCADE,
    class_id          UUID        NOT NULL,
    learning_area_id  UUID        NOT NULL REFERENCES cbc_learning_areas(id) ON DELETE CASCADE,
    teacher_id        UUID        NOT NULL,
    room_identifier   VARCHAR(50) NULL,
    created_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    -- CONSTRAINT 1: A class can only have ONE assignment per specific structure block
    CONSTRAINT unique_class_slot
        UNIQUE (academic_year_id, structure_id, class_id),

    -- CONSTRAINT 2: A teacher cannot be double-booked during the same structure block
    CONSTRAINT unique_teacher_slot
        UNIQUE (academic_year_id, structure_id, teacher_id),

    -- CONSTRAINT 3: A room cannot be double-booked during the same structure block
    CONSTRAINT unique_room_slot
        UNIQUE (academic_year_id, structure_id, room_identifier),

    CONSTRAINT fk_cbc_timetable_slots_tenant_school
        FOREIGN KEY (tenant_id, school_id)
        REFERENCES cbc_schools(tenant_id, id) ON DELETE CASCADE,
    CONSTRAINT fk_cbc_timetable_slots_tenant_class
        FOREIGN KEY (tenant_id, class_id)
        REFERENCES cbc_classes(tenant_id, id) ON DELETE CASCADE,
    CONSTRAINT fk_cbc_timetable_slots_tenant_teacher
        FOREIGN KEY (tenant_id, teacher_id)
        REFERENCES users(tenant_id, id) ON DELETE CASCADE,
    CONSTRAINT fk_cbc_timetable_slots_academic_year
        FOREIGN KEY (tenant_id, academic_year_id)
        REFERENCES academic_years(tenant_id, id) ON DELETE CASCADE
);

COMMENT ON TABLE cbc_timetable_slots IS
    'Grid Allocation Layer — lightweight relational mapping table using fast
     B-Tree composite unique constraints. The grid definition (time ranges)
     lives in timetable_structures; this table only stores assignments of
     class → teacher → learning_area → room per structure block.';

DROP TRIGGER IF EXISTS trg_cbc_timetable_slots_updated_at ON cbc_timetable_slots;
CREATE TRIGGER trg_cbc_timetable_slots_updated_at
    BEFORE UPDATE ON cbc_timetable_slots
    FOR EACH ROW EXECUTE FUNCTION fn_set_updated_at();

CREATE INDEX IF NOT EXISTS idx_cbc_timetable_slots_structure
    ON cbc_timetable_slots (structure_id);
CREATE INDEX IF NOT EXISTS idx_cbc_timetable_slots_class
    ON cbc_timetable_slots (class_id);
CREATE INDEX IF NOT EXISTS idx_cbc_timetable_slots_teacher
    ON cbc_timetable_slots (teacher_id);
CREATE INDEX IF NOT EXISTS idx_cbc_timetable_slots_academic_year
    ON cbc_timetable_slots (academic_year_id);
CREATE INDEX IF NOT EXISTS idx_cbc_timetable_slots_tenant
    ON cbc_timetable_slots (tenant_id);
CREATE INDEX IF NOT EXISTS idx_cbc_timetable_slots_school
    ON cbc_timetable_slots (school_id);



-- ============================================================================
-- LAYER 7 — CBC ASSESSMENT ARCHITECTURE
-- ============================================================================

-- ---------------------------------------------------------------------------
     nationally mandated and do not vary per school. Schema changes would be
     required if per-school overrides are ever needed.';

-- ---------------------------------------------------------------------------
-- ---------------------------------------------------------------------------

-- ---------------------------------------------------------------------------

-- ============================================================================
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
-- LAYER 11 — ATTENDANCE & BEHAVIOR
-- ============================================================================

-- ---------------------------------------------------------------------------
-- ATTENDANCE STATUS ENUM
-- ---------------------------------------------------------------------------

DO $$ BEGIN
    CREATE TYPE attendance_status AS ENUM ('PRESENT', 'ABSENT', 'LATE', 'EXCUSED');
EXCEPTION WHEN duplicate_object THEN NULL;
END $$;

DO $$ BEGIN
    CREATE TYPE behavior_note_status AS ENUM ('PENDING_REVIEW', 'APPROVED', 'REJECTED', 'INCLUDED_IN_REPORT');
EXCEPTION WHEN duplicate_object THEN NULL;
END $$;

DO $$ BEGIN
    CREATE TYPE behavior_severity AS ENUM ('MINOR', 'NEEDS_FOLLOW_UP');
EXCEPTION WHEN duplicate_object THEN NULL;
END $$;

-- ---------------------------------------------------------------------------
-- ATTENDANCE RECORDS
-- One row per student, per timetable slot occurrence, per date.
-- Uniqueness: one record per (student_id, timetable_slot_id, date).
-- ---------------------------------------------------------------------------

CREATE TABLE IF NOT EXISTS attendance_records (
    id                UUID              PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id         UUID              NOT NULL,
    school_id         UUID              NOT NULL,
    student_id        UUID              NOT NULL,
    timetable_slot_id UUID              NOT NULL,
    academic_term_id  UUID              NOT NULL,
    date              DATE              NOT NULL,
    status            attendance_status NOT NULL,
    marked_by         UUID              NOT NULL,
    marked_at         TIMESTAMPTZ       NOT NULL DEFAULT NOW(),
    note              TEXT              NULL,
    created_at        TIMESTAMPTZ       NOT NULL DEFAULT NOW(),
    updated_at        TIMESTAMPTZ       NOT NULL DEFAULT NOW(),

    CONSTRAINT uq_attendance_student_slot_date
        UNIQUE (student_id, timetable_slot_id, date),
    CONSTRAINT fk_attendance_tenant_student
        FOREIGN KEY (tenant_id, student_id)
        REFERENCES cbc_students(tenant_id, id) ON DELETE CASCADE,
    CONSTRAINT fk_attendance_timetable_slot
        FOREIGN KEY (timetable_slot_id)
        REFERENCES cbc_timetable_slots(id) ON DELETE CASCADE,
    CONSTRAINT fk_attendance_tenant_term
        FOREIGN KEY (tenant_id, school_id, academic_term_id)
        REFERENCES academic_terms(tenant_id, school_id, id) ON DELETE CASCADE,
    CONSTRAINT fk_attendance_marked_by
        FOREIGN KEY (tenant_id, marked_by)
        REFERENCES users(tenant_id, id) ON DELETE RESTRICT
);

CREATE INDEX IF NOT EXISTS idx_attendance_slot_date
    ON attendance_records (timetable_slot_id, date);
CREATE INDEX IF NOT EXISTS idx_attendance_student_term
    ON attendance_records (student_id, academic_term_id);
CREATE INDEX IF NOT EXISTS idx_attendance_tenant
    ON attendance_records (tenant_id);
CREATE INDEX IF NOT EXISTS idx_attendance_school
    ON attendance_records (school_id);

COMMENT ON TABLE attendance_records IS
    'Per-student, per-timetable-slot, per-date attendance records. The unique
     constraint (student_id, timetable_slot_id, date) prevents duplicate marks.
     Only created for slots where timetable_structures.is_break = false.';

COMMENT ON COLUMN attendance_records.note IS
    'Optional short free text (e.g. "left early, picked up by parent").';

-- ============================================================================
-- updated_at TRIGGER + NON-BREAK CONSTRAINT (squashed from 000003)
-- The updated_at column is already in the CREATE TABLE above.
-- ============================================================================

DROP TRIGGER IF EXISTS trg_attendance_records_updated_at ON attendance_records;
CREATE TRIGGER trg_attendance_records_updated_at
    BEFORE UPDATE ON attendance_records
    FOR EACH ROW EXECUTE FUNCTION fn_set_updated_at();

COMMENT ON COLUMN attendance_records.updated_at IS
    'Tracks when the record was last modified (status or note update).
     Populated automatically by the trg_attendance_records_updated_at trigger.';

-- ---------------------------------------------------------------------------
-- Non-break slot enforcement: attendance can only be marked for instructional
-- periods, not breaks, recess, or assemblies.
-- ---------------------------------------------------------------------------

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

DROP TRIGGER IF EXISTS trg_attendance_check_non_break_slot ON attendance_records;
CREATE TRIGGER trg_attendance_check_non_break_slot
    BEFORE INSERT OR UPDATE ON attendance_records
    FOR EACH ROW
    EXECUTE FUNCTION fn_check_non_break_slot();

COMMENT ON TRIGGER trg_attendance_check_non_break_slot ON attendance_records IS
    'Enforces that attendance records can only reference timetable slots
     whose corresponding timetable_structures row has is_break = false.
     Prevents system or application bugs from creating attendance marks
     for break/assembly/recess periods.';

-- ---------------------------------------------------------------------------
-- BEHAVIOR CATEGORIES
-- School-managed reference list — admins define categories per school.
-- Soft-delete via is_active = false to preserve historical notes.
-- ---------------------------------------------------------------------------

CREATE TABLE IF NOT EXISTS behavior_categories (
    id               UUID                PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id        UUID                NOT NULL,
    school_id        UUID                NOT NULL,
    name             VARCHAR(100)        NOT NULL,
    default_severity behavior_severity   NULL,
    is_active        BOOLEAN             NOT NULL DEFAULT true,
    created_at       TIMESTAMPTZ         NOT NULL DEFAULT NOW(),

    CONSTRAINT fk_behavior_categories_tenant_school
        FOREIGN KEY (tenant_id, school_id)
        REFERENCES cbc_schools(tenant_id, id) ON DELETE CASCADE,
    CONSTRAINT uq_behavior_category_name
        UNIQUE (tenant_id, school_id, name)
);

CREATE INDEX IF NOT EXISTS idx_behavior_categories_tenant
    ON behavior_categories (tenant_id);
CREATE INDEX IF NOT EXISTS idx_behavior_categories_school
    ON behavior_categories (school_id);

COMMENT ON TABLE behavior_categories IS
    'School-configurable behavior/incident categories. Admins manage these
     per school rather than a fixed platform-wide enum. Categories are soft-
     deleted (is_active = false) to preserve historical behavior_notes.';

-- ---------------------------------------------------------------------------
-- BEHAVIOR NOTES
-- Sparse — only exists when a teacher logs an incident.
-- Goes through admin approval before being included in term reports.
-- ---------------------------------------------------------------------------

CREATE TABLE IF NOT EXISTS behavior_notes (
    id                UUID                 PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id         UUID                 NOT NULL,
    school_id         UUID                 NOT NULL,
    student_id        UUID                 NOT NULL,
    timetable_slot_id UUID                 NOT NULL,
    date              DATE                 NOT NULL,
    category_id       UUID                 NOT NULL,
    description       TEXT                 NOT NULL,
    is_urgent         BOOLEAN              NOT NULL DEFAULT false,
    status            behavior_note_status NOT NULL DEFAULT 'PENDING_REVIEW',
    authored_by_id    UUID                 NOT NULL,
    reviewed_by_id    UUID                 NULL,
    reviewed_at       TIMESTAMPTZ          NULL,
    created_at        TIMESTAMPTZ          NOT NULL DEFAULT NOW(),

    CONSTRAINT fk_behavior_notes_tenant_student
        FOREIGN KEY (tenant_id, student_id)
        REFERENCES cbc_students(tenant_id, id) ON DELETE CASCADE,
    CONSTRAINT fk_behavior_notes_timetable_slot
        FOREIGN KEY (timetable_slot_id)
        REFERENCES cbc_timetable_slots(id) ON DELETE CASCADE,
    CONSTRAINT fk_behavior_notes_category
        FOREIGN KEY (category_id)
        REFERENCES behavior_categories(id) ON DELETE RESTRICT,
    CONSTRAINT fk_behavior_notes_authored_by
        FOREIGN KEY (tenant_id, authored_by_id)
        REFERENCES users(tenant_id, id) ON DELETE RESTRICT,
    CONSTRAINT fk_behavior_notes_reviewed_by
        FOREIGN KEY (tenant_id, reviewed_by_id)
        REFERENCES users(tenant_id, id) ON DELETE SET NULL
);

CREATE INDEX IF NOT EXISTS idx_behavior_notes_student
    ON behavior_notes (student_id);
CREATE INDEX IF NOT EXISTS idx_behavior_notes_status
    ON behavior_notes (status);
CREATE INDEX IF NOT EXISTS idx_behavior_notes_urgent
    ON behavior_notes (is_urgent) WHERE is_urgent = true;
CREATE INDEX IF NOT EXISTS idx_behavior_notes_slot_date
    ON behavior_notes (timetable_slot_id, date);
CREATE INDEX IF NOT EXISTS idx_behavior_notes_tenant
    ON behavior_notes (tenant_id);
CREATE INDEX IF NOT EXISTS idx_behavior_notes_school
    ON behavior_notes (school_id);

COMMENT ON TABLE behavior_notes IS
    'Sparse incident/behavior records logged by teachers. Each note is
     associated with a specific student, timetable slot, and date. Notes
     go through admin approval (PENDING_REVIEW → APPROVED/REJECTED) before
     being included in term reports or reaching parents. Urgent notes bypass
     term-end batching for immediate parent contact.';

COMMENT ON COLUMN behavior_notes.is_urgent IS
    'When true and approved, triggers immediate parent notification instead of
     waiting for term-end compilation.';

-- ---------------------------------------------------------------------------
-- ATTENDANCE TERM SUMMARIES
-- Materialised rollup per student per term per learning area.
-- Populated by a background task (nightly or on-demand).
-- Source of truth remains attendance_records; this is a speed cache.
-- ---------------------------------------------------------------------------

CREATE TABLE IF NOT EXISTS attendance_term_summaries (
    id                   UUID          PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id            UUID          NOT NULL,
    school_id            UUID          NOT NULL,
    student_id           UUID          NOT NULL,
    academic_term_id     UUID          NOT NULL,
    learning_area_id     UUID          NOT NULL,
    periods_total        INT           NOT NULL,
    periods_present      INT           NOT NULL,
    periods_absent       INT           NOT NULL,
    periods_late         INT           NOT NULL,
    periods_excused      INT           NOT NULL,
    attendance_percentage NUMERIC(5,2) NOT NULL,
    last_refreshed_at    TIMESTAMPTZ   NOT NULL DEFAULT NOW(),

    CONSTRAINT fk_summaries_tenant_student
        FOREIGN KEY (tenant_id, student_id)
        REFERENCES cbc_students(tenant_id, id) ON DELETE CASCADE,
    CONSTRAINT fk_summaries_tenant_term
        FOREIGN KEY (tenant_id, school_id, academic_term_id)
        REFERENCES academic_terms(tenant_id, school_id, id) ON DELETE CASCADE,
    CONSTRAINT fk_summaries_learning_area
        FOREIGN KEY (learning_area_id)
        REFERENCES cbc_learning_areas(id) ON DELETE CASCADE,
    CONSTRAINT uq_summary_student_term_area
        UNIQUE (student_id, academic_term_id, learning_area_id)
);

CREATE INDEX IF NOT EXISTS idx_att_summaries_student_term
    ON attendance_term_summaries (student_id, academic_term_id);
CREATE INDEX IF NOT EXISTS idx_att_summaries_tenant
    ON attendance_term_summaries (tenant_id);
CREATE INDEX IF NOT EXISTS idx_att_summaries_school
    ON attendance_term_summaries (school_id);

COMMENT ON TABLE attendance_term_summaries IS
    'Materialised rollup of attendance records per student per term per learning
     area. Populated by a background task (nightly or on-demand when an admin
     generates a term report). Not authoritative — attendance_records is the

COMMENT ON COLUMN attendance_term_summaries.attendance_percentage IS
    'Calculated as (periods_present / periods_total) * 100, stored as a
     decimal with two fractional digits (e.g. 92.50).';

-- ---------------------------------------------------------------------------
-- ============================================================================
-- updated_at COLUMNS & TRIGGERS (missing from initial CREATE TABLE definitions)
--
-- These ALTER TABLE statements add updated_at tracking to tables that were
-- originally created without it. For fresh installs, the column already exists
-- in the CREATE TABLE above, so ADD COLUMN IF NOT EXISTS is a no-op.
-- For existing databases upgraded from an earlier schema version, this block
-- adds the column and trigger idempotently.
--
-- Tables that already have updated_at inline: users, cbc_schools,
-- academic_years, academic_terms, cbc_streams, cbc_classes, memberships,
-- cbc_parents, cbc_students, cbc_student_enrollments, timetable_structures,
-- cbc_timetable_slots, attendance_records, school_member_counts.
-- ============================================================================

-- ---------------------------------------------------------------------------
-- PLATFORM INFRASTRUCTURE
-- ---------------------------------------------------------------------------

ALTER TABLE invitations ADD COLUMN IF NOT EXISTS updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW();
DROP TRIGGER IF EXISTS trg_invitations_updated_at ON invitations;
CREATE TRIGGER trg_invitations_updated_at
    BEFORE UPDATE ON invitations
    FOR EACH ROW EXECUTE FUNCTION fn_set_updated_at();
COMMENT ON COLUMN invitations.updated_at IS
    'Tracks status transitions (pending, accepted, expired, revoked).';

ALTER TABLE import_jobs ADD COLUMN IF NOT EXISTS updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW();
DROP TRIGGER IF EXISTS trg_import_jobs_updated_at ON import_jobs;
CREATE TRIGGER trg_import_jobs_updated_at
    BEFORE UPDATE ON import_jobs
    FOR EACH ROW EXECUTE FUNCTION fn_set_updated_at();
COMMENT ON COLUMN import_jobs.updated_at IS
    'Tracks import lifecycle: pending, processing, completed, failed.';

ALTER TABLE import_job_chunks ADD COLUMN IF NOT EXISTS updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW();
DROP TRIGGER IF EXISTS trg_import_job_chunks_updated_at ON import_job_chunks;
CREATE TRIGGER trg_import_job_chunks_updated_at
    BEFORE UPDATE ON import_job_chunks
    FOR EACH ROW EXECUTE FUNCTION fn_set_updated_at();
COMMENT ON COLUMN import_job_chunks.updated_at IS
    'Tracks chunk processing: pending, processing, completed, cancelled.';

ALTER TABLE import_job_staging ADD COLUMN IF NOT EXISTS updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW();
DROP TRIGGER IF EXISTS trg_import_job_staging_updated_at ON import_job_staging;
CREATE TRIGGER trg_import_job_staging_updated_at
    BEFORE UPDATE ON import_job_staging
    FOR EACH ROW EXECUTE FUNCTION fn_set_updated_at();
COMMENT ON COLUMN import_job_staging.updated_at IS
    'Tracks staging row processing: pending, succeeded, failed.';

ALTER TABLE import_job_failures ADD COLUMN IF NOT EXISTS updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW();
DROP TRIGGER IF EXISTS trg_import_job_failures_updated_at ON import_job_failures;
CREATE TRIGGER trg_import_job_failures_updated_at
    BEFORE UPDATE ON import_job_failures
    FOR EACH ROW EXECUTE FUNCTION fn_set_updated_at();
COMMENT ON COLUMN import_job_failures.updated_at IS
    'Tracks when failure details were last modified.';

-- ---------------------------------------------------------------------------
-- HEALTH & FINANCIALS
-- ---------------------------------------------------------------------------

ALTER TABLE medical_incidents ADD COLUMN IF NOT EXISTS updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW();
DROP TRIGGER IF EXISTS trg_medical_incidents_updated_at ON medical_incidents;
CREATE TRIGGER trg_medical_incidents_updated_at
    BEFORE UPDATE ON medical_incidents
    FOR EACH ROW EXECUTE FUNCTION fn_set_updated_at();
COMMENT ON COLUMN medical_incidents.updated_at IS
    'Tracks medical record corrections and follow-ups.';

ALTER TABLE student_health_profiles ADD COLUMN IF NOT EXISTS updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW();
DROP TRIGGER IF EXISTS trg_student_health_profiles_updated_at ON student_health_profiles;
CREATE TRIGGER trg_student_health_profiles_updated_at
    BEFORE UPDATE ON student_health_profiles
    FOR EACH ROW EXECUTE FUNCTION fn_set_updated_at();
COMMENT ON COLUMN student_health_profiles.updated_at IS
    'Tracks health profile updates (allergies, conditions, instructions).';

ALTER TABLE fee_categories ADD COLUMN IF NOT EXISTS updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW();
DROP TRIGGER IF EXISTS trg_fee_categories_updated_at ON fee_categories;
CREATE TRIGGER trg_fee_categories_updated_at
    BEFORE UPDATE ON fee_categories
    FOR EACH ROW EXECUTE FUNCTION fn_set_updated_at();
COMMENT ON COLUMN fee_categories.updated_at IS
    'Tracks fee category metadata changes.';

ALTER TABLE fee_templates ADD COLUMN IF NOT EXISTS updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW();
DROP TRIGGER IF EXISTS trg_fee_templates_updated_at ON fee_templates;
CREATE TRIGGER trg_fee_templates_updated_at
    BEFORE UPDATE ON fee_templates
    FOR EACH ROW EXECUTE FUNCTION fn_set_updated_at();
COMMENT ON COLUMN fee_templates.updated_at IS
    'Tracks fee amount and configuration changes per term.';

ALTER TABLE invoices ADD COLUMN IF NOT EXISTS updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW();
DROP TRIGGER IF EXISTS trg_invoices_updated_at ON invoices;
CREATE TRIGGER trg_invoices_updated_at
    BEFORE UPDATE ON invoices
    FOR EACH ROW EXECUTE FUNCTION fn_set_updated_at();
COMMENT ON COLUMN invoices.updated_at IS
    'Tracks invoice modifications and payment status sync.';

ALTER TABLE invoice_items ADD COLUMN IF NOT EXISTS updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW();
DROP TRIGGER IF EXISTS trg_invoice_items_updated_at ON invoice_items;
CREATE TRIGGER trg_invoice_items_updated_at
    BEFORE UPDATE ON invoice_items
    FOR EACH ROW EXECUTE FUNCTION fn_set_updated_at();
COMMENT ON COLUMN invoice_items.updated_at IS
    'Tracks invoice line-item corrections.';

ALTER TABLE payments ADD COLUMN IF NOT EXISTS updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW();
DROP TRIGGER IF EXISTS trg_payments_updated_at ON payments;
CREATE TRIGGER trg_payments_updated_at
    BEFORE UPDATE ON payments
    FOR EACH ROW EXECUTE FUNCTION fn_set_updated_at();
COMMENT ON COLUMN payments.updated_at IS
    'Tracks payment record corrections and reconciliations.';

-- ---------------------------------------------------------------------------
-- CBC CURRICULUM STRUCTURE
-- ---------------------------------------------------------------------------

ALTER TABLE cbc_strands ADD COLUMN IF NOT EXISTS updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW();
DROP TRIGGER IF EXISTS trg_cbc_strands_updated_at ON cbc_strands;
CREATE TRIGGER trg_cbc_strands_updated_at
    BEFORE UPDATE ON cbc_strands
    FOR EACH ROW EXECUTE FUNCTION fn_set_updated_at();
COMMENT ON COLUMN cbc_strands.updated_at IS
    'Tracks curriculum strand revisions.';

ALTER TABLE cbc_sub_strands ADD COLUMN IF NOT EXISTS updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW();
DROP TRIGGER IF EXISTS trg_cbc_sub_strands_updated_at ON cbc_sub_strands;
CREATE TRIGGER trg_cbc_sub_strands_updated_at
    BEFORE UPDATE ON cbc_sub_strands
    FOR EACH ROW EXECUTE FUNCTION fn_set_updated_at();
COMMENT ON COLUMN cbc_sub_strands.updated_at IS
    'Tracks curriculum sub-strand revisions.';

ALTER TABLE performance_indicators ADD COLUMN IF NOT EXISTS updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW();
DROP TRIGGER IF EXISTS trg_performance_indicators_updated_at ON performance_indicators;
CREATE TRIGGER trg_performance_indicators_updated_at
    BEFORE UPDATE ON performance_indicators
    FOR EACH ROW EXECUTE FUNCTION fn_set_updated_at();
COMMENT ON COLUMN performance_indicators.updated_at IS
    'Tracks performance indicator revisions and re-sequencing.';

-- TEACHER ASSIGNMENTS & CBC ACTOR JUNCTIONS
-- ---------------------------------------------------------------------------

ALTER TABLE cbc_class_teachers ADD COLUMN IF NOT EXISTS updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW();
DROP TRIGGER IF EXISTS trg_cbc_class_teachers_updated_at ON cbc_class_teachers;
CREATE TRIGGER trg_cbc_class_teachers_updated_at
    BEFORE UPDATE ON cbc_class_teachers
    FOR EACH ROW EXECUTE FUNCTION fn_set_updated_at();
COMMENT ON COLUMN cbc_class_teachers.updated_at IS
    'Tracks teacher assignment changes mid-term.';

ALTER TABLE cbc_student_parents ADD COLUMN IF NOT EXISTS updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW();
DROP TRIGGER IF EXISTS trg_cbc_student_parents_updated_at ON cbc_student_parents;
CREATE TRIGGER trg_cbc_student_parents_updated_at
    BEFORE UPDATE ON cbc_student_parents
    FOR EACH ROW EXECUTE FUNCTION fn_set_updated_at();
COMMENT ON COLUMN cbc_student_parents.updated_at IS
    'Tracks parent relationship and primary-contact changes.';

-- ---------------------------------------------------------------------------
-- ATTENDANCE & BEHAVIOR
-- ---------------------------------------------------------------------------

ALTER TABLE behavior_categories ADD COLUMN IF NOT EXISTS updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW();
DROP TRIGGER IF EXISTS trg_behavior_categories_updated_at ON behavior_categories;
CREATE TRIGGER trg_behavior_categories_updated_at
    BEFORE UPDATE ON behavior_categories
    FOR EACH ROW EXECUTE FUNCTION fn_set_updated_at();
COMMENT ON COLUMN behavior_categories.updated_at IS
    'Tracks category name changes and soft-delete toggles.';

ALTER TABLE behavior_notes ADD COLUMN IF NOT EXISTS updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW();
DROP TRIGGER IF EXISTS trg_behavior_notes_updated_at ON behavior_notes;
CREATE TRIGGER trg_behavior_notes_updated_at
    BEFORE UPDATE ON behavior_notes
    FOR EACH ROW EXECUTE FUNCTION fn_set_updated_at();
COMMENT ON COLUMN behavior_notes.updated_at IS
    'Tracks approval workflow: PENDING_REVIEW, APPROVED, REJECTED.';

ALTER TABLE attendance_term_summaries ADD COLUMN IF NOT EXISTS updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW();
DROP TRIGGER IF EXISTS trg_attendance_term_summaries_updated_at ON attendance_term_summaries;
CREATE TRIGGER trg_attendance_term_summaries_updated_at
    BEFORE UPDATE ON attendance_term_summaries
    FOR EACH ROW EXECUTE FUNCTION fn_set_updated_at();
COMMENT ON COLUMN attendance_term_summaries.updated_at IS
    'Tracks materialised summary refresh cycles.';

    FOR EACH ROW EXECUTE FUNCTION fn_set_updated_at();
    'Tracks report compilation and publication.';

-- ---------------------------------------------------------------------------
-- USER CONTEXT
-- ---------------------------------------------------------------------------

ALTER TABLE member_active_school ADD COLUMN IF NOT EXISTS updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW();
DROP TRIGGER IF EXISTS trg_member_active_school_updated_at ON member_active_school;
CREATE TRIGGER trg_member_active_school_updated_at
    BEFORE UPDATE ON member_active_school
    FOR EACH ROW EXECUTE FUNCTION fn_set_updated_at();
COMMENT ON COLUMN member_active_school.updated_at IS
    'Tracks active school context switches.';

-- ============================================================================
-- ROW-LEVEL SECURITY
--
-- Every table trusting the app layer to filter by tenant_id is one missed
-- WHERE clause away from a cross-tenant data leak. RLS provides a second
-- line of defence. The application must set app.current_tenant_id at the
-- start of each request via SET LOCAL app.current_tenant_id = '<uuid>';
-- before running any queries. The function fn_current_tenant_id() reads
-- that parameter; RLS policies use it to silently filter rows.
--
-- IMPORTANT: The session user must have the TENANT_ISOLATION attribute
-- set (ALTER ROLE ... SET app.current_tenant_id TO ...) or must call
-- SET LOCAL before each transaction. Without this, ALL RLS-protected
-- queries return zero rows.
-- ============================================================================

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

COMMENT ON FUNCTION fn_current_tenant_id() IS
    'Returns the tenant_id set via app.current_tenant_id for the current
     session. Returns NULL if not set (which causes RLS policies to filter
     out ALL rows — safe by default). The application must SET LOCAL
     app.current_tenant_id before each request.';

-- ============================================================================
-- RLS: ENABLE & POLICIES
-- ============================================================================

-- Apply RLS to all tenant-scoped data tables. Tables that hold student
-- health records, financial data, and assessment results are the highest
-- priority under Kenya''s Data Protection Act (2019).

ALTER TABLE IF EXISTS academic_terms                    ENABLE ROW LEVEL SECURITY;
ALTER TABLE IF EXISTS academic_years                    ENABLE ROW LEVEL SECURITY;
ALTER TABLE IF EXISTS cbc_classes                       ENABLE ROW LEVEL SECURITY;
ALTER TABLE IF EXISTS cbc_class_teachers                ENABLE ROW LEVEL SECURITY;
ALTER TABLE IF EXISTS cbc_learning_areas                ENABLE ROW LEVEL SECURITY;
ALTER TABLE IF EXISTS cbc_parents                       ENABLE ROW LEVEL SECURITY;
ALTER TABLE IF EXISTS cbc_student_enrollments           ENABLE ROW LEVEL SECURITY;
ALTER TABLE IF EXISTS cbc_student_parents               ENABLE ROW LEVEL SECURITY;
ALTER TABLE IF EXISTS cbc_students                      ENABLE ROW LEVEL SECURITY;
ALTER TABLE IF EXISTS cbc_schools                       ENABLE ROW LEVEL SECURITY;
ALTER TABLE IF EXISTS cbc_streams                       ENABLE ROW LEVEL SECURITY;
ALTER TABLE IF EXISTS cbc_timetable_slots               ENABLE ROW LEVEL SECURITY;
ALTER TABLE IF EXISTS fee_categories                    ENABLE ROW LEVEL SECURITY;
ALTER TABLE IF EXISTS fee_templates                     ENABLE ROW LEVEL SECURITY;
ALTER TABLE IF EXISTS import_jobs                       ENABLE ROW LEVEL SECURITY;
ALTER TABLE IF EXISTS import_job_staging                ENABLE ROW LEVEL SECURITY;
ALTER TABLE IF EXISTS invitations                       ENABLE ROW LEVEL SECURITY;
ALTER TABLE IF EXISTS invoices                          ENABLE ROW LEVEL SECURITY;
ALTER TABLE IF EXISTS invoice_items                     ENABLE ROW LEVEL SECURITY;
ALTER TABLE IF EXISTS medical_incidents                 ENABLE ROW LEVEL SECURITY;
ALTER TABLE IF EXISTS member_active_school              ENABLE ROW LEVEL SECURITY;
ALTER TABLE IF EXISTS memberships                       ENABLE ROW LEVEL SECURITY;
ALTER TABLE IF EXISTS payments                          ENABLE ROW LEVEL SECURITY;
ALTER TABLE IF EXISTS users                             ENABLE ROW LEVEL SECURITY;
ALTER TABLE IF EXISTS school_member_counts              ENABLE ROW LEVEL SECURITY;
ALTER TABLE IF EXISTS student_health_profiles           ENABLE ROW LEVEL SECURITY;
ALTER TABLE IF EXISTS timetable_structures              ENABLE ROW LEVEL SECURITY;
ALTER TABLE IF EXISTS attendance_records                  ENABLE ROW LEVEL SECURITY;
ALTER TABLE IF EXISTS behavior_categories                 ENABLE ROW LEVEL SECURITY;
ALTER TABLE IF EXISTS behavior_notes                      ENABLE ROW LEVEL SECURITY;
ALTER TABLE IF EXISTS attendance_term_summaries            ENABLE ROW LEVEL SECURITY;

-- Tenant-scoped policy function (reusable): returns a WHERE clause that
-- filters to the current tenant. Tables without tenant_id get no policy;
-- the default-deny behaviour means every query returns zero rows.
CREATE OR REPLACE FUNCTION fn_rls_tenant_policy()
RETURNS TEXT
LANGUAGE plpgsql
STABLE
AS $$
BEGIN
    RETURN format('tenant_id = %L::UUID', fn_current_tenant_id());
END;
$$;

DO $$ DECLARE
    tbl TEXT;
    policy_text TEXT;
    current_id UUID;
BEGIN
    FOR tbl IN
        SELECT unnest(ARRAY[
            'academic_terms',
            'academic_years',
            'cbc_classes',
            'cbc_class_teachers',
            'cbc_learning_areas',
            'cbc_parents',
            'cbc_student_enrollments',
            'cbc_student_parents',
            'cbc_students',
            'cbc_schools',
            'cbc_streams',
            'cbc_timetable_slots',
            'fee_categories',
            'fee_templates',
            'import_jobs',
            'import_job_staging',
            'invitations',
            'invoices',
            'invoice_items',
            'medical_incidents',
            'member_active_school',
            'memberships',
            'payments',
            'school_member_counts',
            'users',
            'student_health_profiles',
            'timetable_structures',
            'attendance_records',
            'behavior_categories',
            'behavior_notes',
            'attendance_term_summaries',
        ])
    LOOP
        EXECUTE format(
            'DROP POLICY IF EXISTS tenant_isolation_policy ON %I',
            tbl
        );
        -- Only create tenant-scoped policy if the table has a tenant_id column.
        -- Junction tables (e.g. cbc_student_parents) and denormalised count
        -- tables (e.g. school_member_counts) may not have one.
        IF EXISTS (
            SELECT 1 FROM information_schema.columns
            WHERE table_name = tbl AND column_name = 'tenant_id'
        ) THEN
            EXECUTE format(
                'CREATE POLICY tenant_isolation_policy ON %I '
                'FOR ALL '
                'USING (tenant_id = fn_current_tenant_id()) '
                'WITH CHECK (tenant_id = fn_current_tenant_id())',
                tbl
            );
        END IF;

    END LOOP;
END $$;

COMMENT ON TABLE cbc_student_enrollments IS
    'Per-term enrollment records. UNIQUE (student_id, school_id, academic_term_id)
     allows same-term transfers (old school sets status=TRANSFERRED, new school
     inserts its own row). RLS enforces tenant isolation at the database level.';

-- ============================================================================
-- END OF MIGRATION
-- ============================================================================