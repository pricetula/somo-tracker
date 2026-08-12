-- Migration: 000001_initial_schema
-- SomoTracker — Kenya CBC/CBE academic platform (CBC-only, v5)
-- Drops all generic education system abstractions.
-- Rebuilds as a purpose-built, single-system CBC schema.
--
-- NOTE: This file is the SQUASHED fresh-install schema. It folds in the
-- net effect of the former migrations 000003–000017 (review fixes, tenant
-- scoping, summary/rollup tables) directly into the object definitions —
-- no CREATE-then-ALTER sequences, no backfills. The seed data (000002)
-- remains a separate migration.

-- ============================================================================
-- EXTENSIONS
-- ============================================================================

CREATE EXTENSION IF NOT EXISTS btree_gist;
CREATE EXTENSION IF NOT EXISTS pgcrypto;

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

DO $$ BEGIN
    CREATE TYPE payment_method_type AS ENUM (
        'MPESA',
        'CASH',
        'BANK_TRANSFER',
        'CHEQUE',
        'OTHER'
    );
EXCEPTION WHEN duplicate_object THEN NULL;
END $$;

DO $$ BEGIN
    CREATE TYPE parent_relationship_type AS ENUM (
        'FATHER',
        'MOTHER',
        'GUARDIAN',
        'OTHER'
    );
EXCEPTION WHEN duplicate_object THEN NULL;
END $$;

DO $$ BEGIN
    CREATE TYPE behavior_category_type AS ENUM ('COMMENDATION', 'DISCIPLINARY', 'OTHER');
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

CREATE UNIQUE INDEX IF NOT EXISTS idx_users_tenant_email
    ON users (tenant_id, LOWER(email));

COMMENT ON INDEX idx_users_tenant_email IS
    'Per-tenant, case-insensitive unique constraint on email. Replaces the
     old global idx_users_email which prevented multi-tenant accounts and
     treated case variants as distinct. Added in 000003_fix.';
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
    token                VARCHAR(128) NULL,
    token_hash           TEXT         NULL,
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

CREATE UNIQUE INDEX IF NOT EXISTS idx_sessions_token_hash ON sessions (token_hash);

COMMENT ON COLUMN sessions.token IS
    'DEPRECATED and will be removed in a future migration. This column is now nullable.
All lookups should use token_hash instead. New sessions will insert NULL here.';

COMMENT ON COLUMN sessions.token_hash IS
    'SHA-256 hash of the session token (hex-encoded). Use this for token
     lookups instead of the raw token column.';

COMMENT ON COLUMN sessions.stytch_session_token IS
    'TODO: stytch_session_token is a third-party session token from
     Stytch, not one this schema issues. Hashing strategy for Stytch tokens
     requires app-team sign-off — do not implement hashing for this column
     without a reviewed design doc.';

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
    updated_at          TIMESTAMPTZ       NOT NULL DEFAULT NOW(),

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
    updated_at    TIMESTAMPTZ         NOT NULL DEFAULT NOW(),

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
    created_at    TIMESTAMPTZ          NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ          NOT NULL DEFAULT NOW()
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
    created_at    TIMESTAMPTZ           NOT NULL DEFAULT now(),
    updated_at    TIMESTAMPTZ           NOT NULL DEFAULT NOW()
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
    token_hash          TEXT              NULL,
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
    updated_at          TIMESTAMPTZ       NOT NULL DEFAULT NOW(),

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

CREATE UNIQUE INDEX IF NOT EXISTS idx_invitations_token_hash ON invitations (token_hash);

COMMENT ON COLUMN invitations.token IS
    'DEPRECATED — raw invitation token. New code should read token_hash instead.
     This column will be dropped in a future migration after the app is
     confirmed fully migrated to hash-based lookups. Do NOT write to this
     column in new code.';

COMMENT ON COLUMN invitations.token_hash IS
    'SHA-256 hash of the invitation token (hex-encoded). Backfilled from token
     column. Use this for token lookups instead of the raw token column.';

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
    school_id              UUID                 NOT NULL,
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
    CONSTRAINT fk_cbc_students_tenant_school
        FOREIGN KEY (tenant_id, school_id)
        REFERENCES cbc_schools(tenant_id, id) ON DELETE CASCADE,
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
    relationship parent_relationship_type NULL, -- FATHER, MOTHER, GUARDIAN, OTHER
    is_primary   BOOLEAN     NOT NULL DEFAULT true,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    PRIMARY KEY (student_id, parent_id)
);

CREATE INDEX IF NOT EXISTS idx_junction_parent ON cbc_student_parents (parent_id);

-- One primary parent per student (000003 item 5)
CREATE UNIQUE INDEX IF NOT EXISTS idx_one_primary_parent_per_student
    ON cbc_student_parents (student_id) WHERE is_primary = true;

COMMENT ON COLUMN cbc_student_parents.relationship IS
    'Parent/guardian relationship to the student. Enum migrated from free-text
     in 000003_fix. Values: FATHER, MOTHER, GUARDIAN, OTHER.';

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
    academic_year_id UUID                  NOT NULL,
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
    CONSTRAINT unique_student_term_enrollment UNIQUE (student_id, school_id, academic_term_id),
    CONSTRAINT fk_enrollments_tenant_academic_year
        FOREIGN KEY (tenant_id, academic_year_id)
        REFERENCES academic_years(tenant_id, id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_cbc_enrollments_tenant_id  ON cbc_student_enrollments (tenant_id);
CREATE INDEX IF NOT EXISTS idx_cbc_enrollments_student_id ON cbc_student_enrollments (student_id);
CREATE INDEX IF NOT EXISTS idx_cbc_enrollments_school_id  ON cbc_student_enrollments (school_id);
CREATE INDEX IF NOT EXISTS idx_cbc_enrollments_term_id    ON cbc_student_enrollments (academic_term_id);
CREATE INDEX IF NOT EXISTS idx_cbc_enrollments_class_id   ON cbc_student_enrollments (class_id);
CREATE INDEX IF NOT EXISTS idx_cbc_enrollments_academic_year_id
    ON cbc_student_enrollments (academic_year_id);

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
    student_id         UUID        NOT NULL,
    incident_timestamp TIMESTAMPTZ NOT NULL,
    symptoms           TEXT        NOT NULL,
    action_taken       TEXT        NOT NULL,
    logged_by          UUID        NOT NULL REFERENCES users(id),
    created_at         TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at         TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT fk_medical_incidents_tenant_student
        FOREIGN KEY (tenant_id, student_id)
        REFERENCES cbc_students(tenant_id, id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_medical_incidents_tenant_id  ON medical_incidents (tenant_id);
CREATE INDEX IF NOT EXISTS idx_medical_incidents_student_id ON medical_incidents (student_id);

-- ---------------------------------------------------------------------------
-- STUDENT HEALTH PROFILES
-- ---------------------------------------------------------------------------

CREATE TABLE IF NOT EXISTS student_health_profiles (
    id                     UUID    PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id              UUID    NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    student_id             UUID    UNIQUE NOT NULL,
    blood_group            VARCHAR(5),
    allergies              TEXT[],
    chronic_conditions     TEXT[],
    emergency_instructions TEXT,
    created_at             TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at             TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT fk_student_health_profiles_tenant_student
        FOREIGN KEY (tenant_id, student_id)
        REFERENCES cbc_students(tenant_id, id) ON DELETE CASCADE
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
    updated_at   TIMESTAMPTZ   NOT NULL DEFAULT NOW(),

    CONSTRAINT fk_fee_categories_tenant_school
        FOREIGN KEY (tenant_id, school_id)
        REFERENCES cbc_schools(tenant_id, id) ON DELETE CASCADE,
    CONSTRAINT uq_fee_categories_tenant_school_name
        UNIQUE (tenant_id, school_id, name)
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
    updated_at       TIMESTAMPTZ      NOT NULL DEFAULT NOW(),

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
    updated_at       TIMESTAMPTZ            NOT NULL DEFAULT NOW(),

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
    updated_at      TIMESTAMPTZ   NOT NULL DEFAULT NOW(),

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
    payment_method payment_method_type NULL,
    reference_code VARCHAR(100)  NULL,
    recorded_by    UUID          NOT NULL REFERENCES users(id),
    created_at     TIMESTAMPTZ   NOT NULL DEFAULT NOW(),
    updated_at     TIMESTAMPTZ   NOT NULL DEFAULT NOW(),

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

COMMENT ON COLUMN payments.payment_method IS
    'Payment method type enum. Covers the four real Kenyan payment channels
     plus OTHER. Original free-text column migrated to enum in 000003_fix.';

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
        payment_status = (CASE
            WHEN i.payment_status = 'WAIVED'               THEN 'WAIVED'::invoice_payment_status
            WHEN COALESCE(p.total_paid, 0) = 0             THEN 'UNPAID'::invoice_payment_status
            WHEN COALESCE(p.total_paid, 0) >= i.amount_due THEN 'PAID'::invoice_payment_status
            ELSE 'PARTIAL'::invoice_payment_status
        END)
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
        payment_status = (CASE
            WHEN i.payment_status = 'WAIVED'               THEN 'WAIVED'::invoice_payment_status
            WHEN COALESCE(p.total_paid, 0) = 0             THEN 'UNPAID'::invoice_payment_status
            WHEN COALESCE(p.total_paid, 0) >= i.amount_due THEN 'PAID'::invoice_payment_status
            ELSE 'PARTIAL'::invoice_payment_status
        END)
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
        payment_status = (CASE
            WHEN i.payment_status = 'WAIVED'               THEN 'WAIVED'::invoice_payment_status
            WHEN COALESCE(p.total_paid, 0) = 0             THEN 'UNPAID'::invoice_payment_status
            WHEN COALESCE(p.total_paid, 0) >= i.amount_due THEN 'PAID'::invoice_payment_status
            ELSE 'PARTIAL'::invoice_payment_status
        END)
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
    tenant_id        UUID         NOT NULL,
    learning_area_id UUID         NOT NULL REFERENCES cbc_learning_areas(id) ON DELETE CASCADE,
    name             VARCHAR(255) NOT NULL,
    updated_at       TIMESTAMPTZ  NOT NULL DEFAULT NOW(),

    CONSTRAINT uq_cbc_strands_tenant UNIQUE (tenant_id, id),
    CONSTRAINT fk_cbc_strands_tenant_learning_area
        FOREIGN KEY (tenant_id, learning_area_id)
        REFERENCES cbc_learning_areas(tenant_id, id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_cbc_strands_learning_area_id ON cbc_strands (learning_area_id);
CREATE INDEX IF NOT EXISTS idx_cbc_strands_tenant ON cbc_strands (tenant_id);

-- ---------------------------------------------------------------------------
-- CBC SUB-STRANDS
-- ---------------------------------------------------------------------------

CREATE TABLE IF NOT EXISTS cbc_sub_strands (
    id        UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID         NOT NULL,
    strand_id UUID         NOT NULL REFERENCES cbc_strands(id) ON DELETE CASCADE,
    name      VARCHAR(255) NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT uq_cbc_sub_strands_tenant UNIQUE (tenant_id, id),
    CONSTRAINT fk_cbc_sub_strands_tenant_strand
        FOREIGN KEY (tenant_id, strand_id)
        REFERENCES cbc_strands(tenant_id, id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_cbc_sub_strands_strand_id ON cbc_sub_strands (strand_id);
CREATE INDEX IF NOT EXISTS idx_cbc_sub_strands_tenant ON cbc_sub_strands (tenant_id);

-- ---------------------------------------------------------------------------
-- PERFORMANCE INDICATORS
-- ---------------------------------------------------------------------------

CREATE TABLE IF NOT EXISTS performance_indicators (
    id             UUID     PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id      UUID     NOT NULL,
    sub_strand_id  UUID     NOT NULL REFERENCES cbc_sub_strands(id) ON DELETE CASCADE,
    description    TEXT     NOT NULL,
    sequence_order SMALLINT NOT NULL DEFAULT 1,
    updated_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT fk_performance_indicators_tenant_sub_strand
        FOREIGN KEY (tenant_id, sub_strand_id)
        REFERENCES cbc_sub_strands(tenant_id, id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_performance_indicators_sub_strand
    ON performance_indicators (sub_strand_id, sequence_order);
CREATE INDEX IF NOT EXISTS idx_performance_indicators_tenant
    ON performance_indicators (tenant_id);

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
    updated_at       TIMESTAMPTZ  NOT NULL DEFAULT NOW(),

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
-- ASSESSMENT WEIGHT CONFIGS — official KNEC weighting formulas
-- KPSEA: 60% SBA (G4+G5) + 40% KPSEA written (G6)
-- KJSEA: 20% SBA (G7+G8) + 20% KPSEA result + 60% KJSEA written (G9)
-- ---------------------------------------------------------------------------

CREATE TABLE IF NOT EXISTS assessment_weight_configs (
    id                   UUID           PRIMARY KEY DEFAULT gen_random_uuid(),
    grade_level          cbc_grade_level NOT NULL,
    assessment_type_code VARCHAR(50)    NOT NULL,
    target_exam          VARCHAR(20)    NOT NULL,
    weight_percent       NUMERIC(5,2)   NOT NULL CHECK (weight_percent > 0 AND weight_percent <= 100),
    effective_from       INTEGER        NOT NULL,
    notes                TEXT           NULL,
    created_at           TIMESTAMPTZ    NOT NULL DEFAULT NOW(),

    CONSTRAINT uq_assessment_weight_config
        UNIQUE (grade_level, assessment_type_code, target_exam, effective_from)
);

COMMENT ON TABLE assessment_weight_configs IS
    'Official KNEC assessment weighting formulas per grade level. These are
     nationally mandated and do not vary per school. Schema changes would be
     required if per-school overrides are ever needed.';

COMMENT ON COLUMN assessment_weight_configs.assessment_type_code IS
    'KNEC assessment type identifier, e.g. KNEC_SBA_Project, National_KPSEA, National_KJSEA.';
COMMENT ON COLUMN assessment_weight_configs.target_exam IS
    'The target national exam this weight contributes to: KPSEA, KJSEA, or KSSEA.';
COMMENT ON COLUMN assessment_weight_configs.weight_percent IS
    'Percentage contribution of this assessment component towards the target exam placement.';
COMMENT ON COLUMN assessment_weight_configs.effective_from IS
    'Academic year from which this weighting formula is effective.';

-- ============================================================================
-- SCHOOL MEMBER COUNTS — materialised denormalised counts
-- ============================================================================

CREATE TABLE IF NOT EXISTS school_member_counts (
    school_id  UUID         PRIMARY KEY,
    admins     INTEGER      NOT NULL DEFAULT 0,
    teachers   INTEGER      NOT NULL DEFAULT 0,
    nurses     INTEGER      NOT NULL DEFAULT 0,
    finance    INTEGER      NOT NULL DEFAULT 0,
    parents    INTEGER      NOT NULL DEFAULT 0,
    students   INTEGER      NOT NULL DEFAULT 0,
    updated_at TIMESTAMPTZ  NOT NULL DEFAULT NOW(),

    CONSTRAINT fk_school_member_counts_school
        FOREIGN KEY (school_id) REFERENCES cbc_schools(id) ON DELETE CASCADE
);

-- ============================================================================
-- TRIGGER: Sync school staff counts from memberships
-- ============================================================================

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
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),

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
-- CBC ATTENDANCE SESSIONS
-- Tracks actual lesson execution instances so teachers can flag dates
-- that did not hold (teacher absence, assembly, sports day, etc.).
-- Skipped sessions are excluded from terminal attendance denominator
-- calculations to avoid penalising students.
-- ---------------------------------------------------------------------------

CREATE TABLE IF NOT EXISTS cbc_attendance_sessions (
    id                UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id         UUID         NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    school_id         UUID         NOT NULL,
    timetable_slot_id UUID         NOT NULL REFERENCES cbc_timetable_slots(id) ON DELETE CASCADE,
    date              DATE         NOT NULL,
    status            VARCHAR(20)  NOT NULL DEFAULT 'SUBMITTED',
    skip_reason       TEXT         NULL,
    created_at        TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at        TIMESTAMPTZ  NOT NULL DEFAULT NOW(),

    CONSTRAINT chk_cbc_attendance_session_status
        CHECK (status IN ('SUBMITTED', 'SKIPPED')),
    CONSTRAINT uq_cbc_attendance_sessions_slot_date
        UNIQUE (school_id, timetable_slot_id, date),
    CONSTRAINT fk_cbc_attendance_sessions_tenant_school
        FOREIGN KEY (tenant_id, school_id)
        REFERENCES cbc_schools(tenant_id, id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_cbc_attendance_sessions_slot_date
    ON cbc_attendance_sessions (timetable_slot_id, date);
CREATE INDEX IF NOT EXISTS idx_cbc_attendance_sessions_tenant
    ON cbc_attendance_sessions (tenant_id);
CREATE INDEX IF NOT EXISTS idx_cbc_attendance_sessions_school
    ON cbc_attendance_sessions (school_id);
CREATE INDEX IF NOT EXISTS idx_cbc_attendance_sessions_status
    ON cbc_attendance_sessions (status);

COMMENT ON TABLE cbc_attendance_sessions IS
    'Tracks actual lesson execution instances per timetable slot and date.
     Teachers flag sessions as SKIPPED when a class did not hold (teacher
     absence, school assembly, sports day, etc.). Skipped sessions exclude
     their attendance records from terminal percentage calculations so
     students are not penalised for cancelled lessons.';

COMMENT ON COLUMN cbc_attendance_sessions.status IS
    'SUBMITTED = lesson held as scheduled (default). SKIPPED = lesson did
     not hold. Only SKIPPED sessions affect terminal attendance calculations
     by reducing the expected denominator.';

COMMENT ON COLUMN cbc_attendance_sessions.skip_reason IS
    'Teacher-provided reason when status is SKIPPED. Examples: School
     Assembly, Public Holiday, Teacher Absence, Sports/Field Event.';

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
    attendance_session_id UUID NULL REFERENCES cbc_attendance_sessions(id) ON DELETE SET NULL,
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
    category_type    behavior_category_type NOT NULL DEFAULT 'DISCIPLINARY',
    created_at       TIMESTAMPTZ         NOT NULL DEFAULT NOW(),
    updated_at       TIMESTAMPTZ         NOT NULL DEFAULT NOW(),

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

COMMENT ON COLUMN behavior_categories.category_type IS
    'Classification of the behavior category: COMMENDATION (positive/laudable
     behaviour), DISCIPLINARY (negative behaviour / infraction), or OTHER.
     Used by student_behavior_term_summaries to compute commendations_count
     and disciplinary_count. Defaults to DISCIPLINARY for existing categories.';

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
    updated_at        TIMESTAMPTZ          NOT NULL DEFAULT NOW(),

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
    academic_year_id     UUID          NOT NULL,
    created_at           TIMESTAMPTZ   NOT NULL DEFAULT NOW(),
    updated_at           TIMESTAMPTZ   NOT NULL DEFAULT NOW(),

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
        UNIQUE (student_id, academic_term_id, learning_area_id),
    CONSTRAINT fk_summaries_tenant_academic_year
        FOREIGN KEY (tenant_id, academic_year_id)
        REFERENCES academic_years(tenant_id, id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_att_summaries_student_term
    ON attendance_term_summaries (student_id, academic_term_id);
CREATE INDEX IF NOT EXISTS idx_att_summaries_tenant
    ON attendance_term_summaries (tenant_id);
CREATE INDEX IF NOT EXISTS idx_att_summaries_school
    ON attendance_term_summaries (school_id);
CREATE INDEX IF NOT EXISTS idx_att_summaries_academic_year
    ON attendance_term_summaries (academic_year_id);

COMMENT ON TABLE attendance_term_summaries IS
    'Materialised rollup of attendance records per student per term per learning
     area. Populated by a background task (nightly or on-demand when an admin
     generates a term report). Not authoritative — attendance_records is the
     source of truth for all attendance calculations.';

COMMENT ON COLUMN attendance_term_summaries.attendance_percentage IS
    'Calculated as (periods_present / periods_total) * 100, stored as a
     decimal with two fractional digits (e.g. 92.50).';





CREATE INDEX IF NOT EXISTS idx_attendance_records_session
    ON attendance_records (attendance_session_id);

COMMENT ON COLUMN attendance_records.attendance_session_id IS
    'Optional reference to the cbc_attendance_sessions row. Populated when
     session is marked as SKIPPED to link existing records. NULL for normal
     (non-skipped) attendance marks.';

-- ============================================================================
-- updated_at TRIGGERS
--
-- All updated_at columns are declared inline in their CREATE TABLE above.
-- These blocks only create the BEFORE UPDATE triggers that stamp the column.
-- ============================================================================

-- ---------------------------------------------------------------------------
-- PLATFORM INFRASTRUCTURE
-- ---------------------------------------------------------------------------

DROP TRIGGER IF EXISTS trg_invitations_updated_at ON invitations;
CREATE TRIGGER trg_invitations_updated_at
    BEFORE UPDATE ON invitations
    FOR EACH ROW EXECUTE FUNCTION fn_set_updated_at();
COMMENT ON COLUMN invitations.updated_at IS
    'Tracks status transitions (pending, accepted, expired, revoked).';

DROP TRIGGER IF EXISTS trg_import_jobs_updated_at ON import_jobs;
CREATE TRIGGER trg_import_jobs_updated_at
    BEFORE UPDATE ON import_jobs
    FOR EACH ROW EXECUTE FUNCTION fn_set_updated_at();
COMMENT ON COLUMN import_jobs.updated_at IS
    'Tracks import lifecycle: pending, processing, completed, failed.';

DROP TRIGGER IF EXISTS trg_import_job_chunks_updated_at ON import_job_chunks;
CREATE TRIGGER trg_import_job_chunks_updated_at
    BEFORE UPDATE ON import_job_chunks
    FOR EACH ROW EXECUTE FUNCTION fn_set_updated_at();
COMMENT ON COLUMN import_job_chunks.updated_at IS
    'Tracks chunk processing: pending, processing, completed, cancelled.';

DROP TRIGGER IF EXISTS trg_import_job_staging_updated_at ON import_job_staging;
CREATE TRIGGER trg_import_job_staging_updated_at
    BEFORE UPDATE ON import_job_staging
    FOR EACH ROW EXECUTE FUNCTION fn_set_updated_at();
COMMENT ON COLUMN import_job_staging.updated_at IS
    'Tracks staging row processing: pending, succeeded, failed.';

DROP TRIGGER IF EXISTS trg_import_job_failures_updated_at ON import_job_failures;
CREATE TRIGGER trg_import_job_failures_updated_at
    BEFORE UPDATE ON import_job_failures
    FOR EACH ROW EXECUTE FUNCTION fn_set_updated_at();
COMMENT ON COLUMN import_job_failures.updated_at IS
    'Tracks when failure details were last modified.';

-- ---------------------------------------------------------------------------
-- HEALTH & FINANCIALS
-- ---------------------------------------------------------------------------

DROP TRIGGER IF EXISTS trg_medical_incidents_updated_at ON medical_incidents;
CREATE TRIGGER trg_medical_incidents_updated_at
    BEFORE UPDATE ON medical_incidents
    FOR EACH ROW EXECUTE FUNCTION fn_set_updated_at();
COMMENT ON COLUMN medical_incidents.updated_at IS
    'Tracks medical record corrections and follow-ups.';

DROP TRIGGER IF EXISTS trg_student_health_profiles_updated_at ON student_health_profiles;
CREATE TRIGGER trg_student_health_profiles_updated_at
    BEFORE UPDATE ON student_health_profiles
    FOR EACH ROW EXECUTE FUNCTION fn_set_updated_at();
COMMENT ON COLUMN student_health_profiles.updated_at IS
    'Tracks health profile updates (allergies, conditions, instructions).';

DROP TRIGGER IF EXISTS trg_fee_categories_updated_at ON fee_categories;
CREATE TRIGGER trg_fee_categories_updated_at
    BEFORE UPDATE ON fee_categories
    FOR EACH ROW EXECUTE FUNCTION fn_set_updated_at();
COMMENT ON COLUMN fee_categories.updated_at IS
    'Tracks fee category metadata changes.';

DROP TRIGGER IF EXISTS trg_fee_templates_updated_at ON fee_templates;
CREATE TRIGGER trg_fee_templates_updated_at
    BEFORE UPDATE ON fee_templates
    FOR EACH ROW EXECUTE FUNCTION fn_set_updated_at();
COMMENT ON COLUMN fee_templates.updated_at IS
    'Tracks fee amount and configuration changes per term.';

DROP TRIGGER IF EXISTS trg_invoices_updated_at ON invoices;
CREATE TRIGGER trg_invoices_updated_at
    BEFORE UPDATE ON invoices
    FOR EACH ROW EXECUTE FUNCTION fn_set_updated_at();
COMMENT ON COLUMN invoices.updated_at IS
    'Tracks invoice modifications and payment status sync.';

DROP TRIGGER IF EXISTS trg_invoice_items_updated_at ON invoice_items;
CREATE TRIGGER trg_invoice_items_updated_at
    BEFORE UPDATE ON invoice_items
    FOR EACH ROW EXECUTE FUNCTION fn_set_updated_at();
COMMENT ON COLUMN invoice_items.updated_at IS
    'Tracks invoice line-item corrections.';

DROP TRIGGER IF EXISTS trg_payments_updated_at ON payments;
CREATE TRIGGER trg_payments_updated_at
    BEFORE UPDATE ON payments
    FOR EACH ROW EXECUTE FUNCTION fn_set_updated_at();
COMMENT ON COLUMN payments.updated_at IS
    'Tracks payment record corrections and reconciliations.';

-- ---------------------------------------------------------------------------
-- CBC CURRICULUM STRUCTURE
-- ---------------------------------------------------------------------------

DROP TRIGGER IF EXISTS trg_cbc_strands_updated_at ON cbc_strands;
CREATE TRIGGER trg_cbc_strands_updated_at
    BEFORE UPDATE ON cbc_strands
    FOR EACH ROW EXECUTE FUNCTION fn_set_updated_at();
COMMENT ON COLUMN cbc_strands.updated_at IS
    'Tracks curriculum strand revisions.';

DROP TRIGGER IF EXISTS trg_cbc_sub_strands_updated_at ON cbc_sub_strands;
CREATE TRIGGER trg_cbc_sub_strands_updated_at
    BEFORE UPDATE ON cbc_sub_strands
    FOR EACH ROW EXECUTE FUNCTION fn_set_updated_at();
COMMENT ON COLUMN cbc_sub_strands.updated_at IS
    'Tracks curriculum sub-strand revisions.';

DROP TRIGGER IF EXISTS trg_performance_indicators_updated_at ON performance_indicators;
CREATE TRIGGER trg_performance_indicators_updated_at
    BEFORE UPDATE ON performance_indicators
    FOR EACH ROW EXECUTE FUNCTION fn_set_updated_at();
COMMENT ON COLUMN performance_indicators.updated_at IS
    'Tracks performance indicator revisions and re-sequencing.';

-- TEACHER ASSIGNMENTS & CBC ACTOR JUNCTIONS
-- ---------------------------------------------------------------------------

DROP TRIGGER IF EXISTS trg_cbc_class_teachers_updated_at ON cbc_class_teachers;
CREATE TRIGGER trg_cbc_class_teachers_updated_at
    BEFORE UPDATE ON cbc_class_teachers
    FOR EACH ROW EXECUTE FUNCTION fn_set_updated_at();
COMMENT ON COLUMN cbc_class_teachers.updated_at IS
    'Tracks teacher assignment changes mid-term.';

DROP TRIGGER IF EXISTS trg_cbc_student_parents_updated_at ON cbc_student_parents;
CREATE TRIGGER trg_cbc_student_parents_updated_at
    BEFORE UPDATE ON cbc_student_parents
    FOR EACH ROW EXECUTE FUNCTION fn_set_updated_at();
COMMENT ON COLUMN cbc_student_parents.updated_at IS
    'Tracks parent relationship and primary-contact changes.';

-- ---------------------------------------------------------------------------
-- ATTENDANCE & BEHAVIOR
-- ---------------------------------------------------------------------------

DROP TRIGGER IF EXISTS trg_behavior_categories_updated_at ON behavior_categories;
CREATE TRIGGER trg_behavior_categories_updated_at
    BEFORE UPDATE ON behavior_categories
    FOR EACH ROW EXECUTE FUNCTION fn_set_updated_at();
COMMENT ON COLUMN behavior_categories.updated_at IS
    'Tracks category name changes and soft-delete toggles.';

DROP TRIGGER IF EXISTS trg_behavior_notes_updated_at ON behavior_notes;
CREATE TRIGGER trg_behavior_notes_updated_at
    BEFORE UPDATE ON behavior_notes
    FOR EACH ROW EXECUTE FUNCTION fn_set_updated_at();
COMMENT ON COLUMN behavior_notes.updated_at IS
    'Tracks approval workflow: PENDING_REVIEW, APPROVED, REJECTED.';

DROP TRIGGER IF EXISTS trg_attendance_term_summaries_updated_at ON attendance_term_summaries;
CREATE TRIGGER trg_attendance_term_summaries_updated_at
    BEFORE UPDATE ON attendance_term_summaries
    FOR EACH ROW EXECUTE FUNCTION fn_set_updated_at();
COMMENT ON COLUMN attendance_term_summaries.updated_at IS
    'Tracks materialised summary refresh cycles.';

DROP TRIGGER IF EXISTS trg_cbc_attendance_sessions_updated_at ON cbc_attendance_sessions;
CREATE TRIGGER trg_cbc_attendance_sessions_updated_at
    BEFORE UPDATE ON cbc_attendance_sessions
    FOR EACH ROW EXECUTE FUNCTION fn_set_updated_at();
COMMENT ON COLUMN cbc_attendance_sessions.updated_at IS
    'Tracks session status changes and skip reason updates.';

-- ---------------------------------------------------------------------------
-- USER CONTEXT
-- ---------------------------------------------------------------------------

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
ALTER TABLE IF EXISTS cbc_attendance_sessions            ENABLE ROW LEVEL SECURITY;
ALTER TABLE IF EXISTS cbc_strands                        ENABLE ROW LEVEL SECURITY;
ALTER TABLE IF EXISTS cbc_sub_strands                    ENABLE ROW LEVEL SECURITY;
ALTER TABLE IF EXISTS performance_indicators             ENABLE ROW LEVEL SECURITY;

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
            'cbc_attendance_sessions',
            'cbc_strands',
            'cbc_sub_strands',
            'performance_indicators'
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
-- LAYER 12 — ASSESSMENT & GRADING ENGINE
-- ============================================================================

-- ---------------------------------------------------------------------------
-- ASSESSMENT ENUMS
-- ---------------------------------------------------------------------------

DO $$ BEGIN
    CREATE TYPE cbc_performance_level AS ENUM ('EE', 'ME', 'AE', 'BE');
EXCEPTION WHEN duplicate_object THEN NULL;
END $$;

COMMENT ON TYPE cbc_performance_level IS
    'CBC rubric performance levels: EE = Exceeding Expectation, ME = Meeting
     Expectation, AE = Approaching Expectation, BE = Below Expectation.';

DO $$ BEGIN
    CREATE TYPE assessment_session_status AS ENUM ('DRAFT', 'PENDING_APPROVAL', 'PUBLISHED');
EXCEPTION WHEN duplicate_object THEN NULL;
END $$;

COMMENT ON TYPE assessment_session_status IS
    'Lifecycle state for an assessment session: DRAFT (teacher editing),
     PENDING_APPROVAL (submitted for admin review), PUBLISHED (finalised
     and visible to parents).';

DO $$ BEGIN
    CREATE TYPE assessment_evaluation_method AS ENUM ('QUANTITATIVE', 'RUBRIC');
EXCEPTION WHEN duplicate_object THEN NULL;
END $$;

COMMENT ON TYPE assessment_evaluation_method IS
    'QUANTITATIVE: total-marks grading (raw_score + percentage → performance level).
     RUBRIC: indicator-level grading (direct CBC level per performance_indicator).';

-- ---------------------------------------------------------------------------
-- GRADING SCALE PROFILES (The Directory)
-- Immutable once created. Administrators toggle is_active to deprecate.
-- ---------------------------------------------------------------------------

CREATE TABLE IF NOT EXISTS grading_scale_profiles (
    id         UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id  UUID         NOT NULL,
    school_id  UUID         NOT NULL,
    name       VARCHAR(255) NOT NULL,
    is_active  BOOLEAN      NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ  NOT NULL DEFAULT NOW(),

    CONSTRAINT fk_grading_scale_profiles_tenant_school
        FOREIGN KEY (tenant_id, school_id)
        REFERENCES cbc_schools(tenant_id, id) ON DELETE CASCADE,
    CONSTRAINT uq_grading_scale_profiles_tenant UNIQUE (tenant_id, id)
);

CREATE INDEX IF NOT EXISTS idx_grading_scale_profiles_tenant
    ON grading_scale_profiles (tenant_id);
CREATE INDEX IF NOT EXISTS idx_grading_scale_profiles_school
    ON grading_scale_profiles (school_id);

DROP TRIGGER IF EXISTS trg_grading_scale_profiles_updated_at ON grading_scale_profiles;
CREATE TRIGGER trg_grading_scale_profiles_updated_at
    BEFORE UPDATE ON grading_scale_profiles
    FOR EACH ROW EXECUTE FUNCTION fn_set_updated_at();

COMMENT ON TABLE grading_scale_profiles IS
    'Directory of CBC grading scale profiles. Profiles define the translation
     from numeric percentages to CBC rubric levels (EE, ME, AE, BE). Once
     created, profile name and settings are read-only. To change a scale,
     create a new profile and mark the old one is_active = false.';

-- ---------------------------------------------------------------------------
-- GRADING SCALE RANGES (The Rules)
-- Write-once rows within each profile. PostgreSQL EXCLUDE constraint using
-- numrange prevents overlapping percentage bands within the same profile.
-- ---------------------------------------------------------------------------

CREATE TABLE IF NOT EXISTS grading_scale_ranges (
    id                        UUID                  PRIMARY KEY DEFAULT gen_random_uuid(),
    profile_id                UUID                  NOT NULL REFERENCES grading_scale_profiles(id) ON DELETE CASCADE,
    tenant_id                 UUID                  NOT NULL,
    performance_level         cbc_performance_level NOT NULL,
    min_percentage            NUMERIC(5,2)          NOT NULL CHECK (min_percentage >= 0 AND min_percentage <= 100),
    max_percentage            NUMERIC(5,2)          NOT NULL CHECK (max_percentage >= 0 AND max_percentage <= 100),
    default_percentage_mapping NUMERIC(5,2)          NULL,
    created_at                TIMESTAMPTZ           NOT NULL DEFAULT NOW(),
    updated_at                TIMESTAMPTZ           NOT NULL DEFAULT NOW(),

    CONSTRAINT chk_range_bounds CHECK (max_percentage > min_percentage),
    CONSTRAINT uq_profile_level UNIQUE (profile_id, performance_level),
    CONSTRAINT excl_profile_range_no_overlap
        EXCLUDE USING gist (
            profile_id WITH =,
            numrange(min_percentage, max_percentage, '[]') WITH &&
        ),
    CONSTRAINT fk_grading_scale_ranges_tenant_profile
        FOREIGN KEY (tenant_id, profile_id)
        REFERENCES grading_scale_profiles(tenant_id, id) ON DELETE CASCADE
);

DROP TRIGGER IF EXISTS trg_grading_scale_ranges_updated_at ON grading_scale_ranges;
CREATE TRIGGER trg_grading_scale_ranges_updated_at
    BEFORE UPDATE ON grading_scale_ranges
    FOR EACH ROW EXECUTE FUNCTION fn_set_updated_at();

CREATE INDEX IF NOT EXISTS idx_grading_scale_ranges_tenant
    ON grading_scale_ranges (tenant_id);

-- Write-once enforcement (000003 item 6a): block UPDATE/DELETE of ranges whose
-- profile is referenced by any assessment_sessions row.
CREATE OR REPLACE FUNCTION fn_block_grading_scale_range_modification()
RETURNS TRIGGER AS $$
BEGIN
    IF EXISTS (
        SELECT 1 FROM assessment_sessions
        WHERE grading_scale_profile_id = OLD.profile_id
        LIMIT 1
    ) THEN
        RAISE EXCEPTION 'Cannot modify or delete grading scale range (profile_id: %) — the grading profile is actively referenced by assessment sessions', OLD.profile_id
            USING ERRCODE = 'P0002';  -- assigned application-level error code
    END IF;
    IF TG_OP = 'DELETE' THEN
        RETURN OLD;
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS trg_grading_scale_ranges_immutable ON grading_scale_ranges;
CREATE TRIGGER trg_grading_scale_ranges_immutable
    BEFORE UPDATE OR DELETE ON grading_scale_ranges
    FOR EACH ROW
    EXECUTE FUNCTION fn_block_grading_scale_range_modification();

COMMENT ON TRIGGER trg_grading_scale_ranges_immutable ON grading_scale_ranges IS
    'Enforces write-once semantics: prevents UPDATE or DELETE of grading scale
     ranges whose profile is referenced by any assessment_sessions row. Throws
     error code P0002 which the application can catch specifically.';

COMMENT ON TABLE grading_scale_ranges IS
    'Range definitions within a grading scale profile. The EXCLUDE constraint
     using numrange guarantees no overlapping percentage bands within the same
     profile. Rows are write-once — UPDATE and DELETE are blocked at the
     application layer once the profile is actively referenced by sessions.';

COMMENT ON COLUMN grading_scale_ranges.default_percentage_mapping IS
    'Optional midpoint value used as the default when converting a percentage
     to a performance level. If NULL, the system uses the midpoint of the range.
     Example: for range 80-100 → EE, default could be 90.';

COMMENT ON COLUMN grading_scale_ranges.tenant_id IS
    'Denormalised from grading_scale_profiles for RLS enforcement. Must match
     the tenant_id of the referenced profile.';

-- ---------------------------------------------------------------------------
-- ASSESSMENT SESSIONS
-- Tracks the lifecycle: DRAFT → PENDING_APPROVAL → PUBLISHED
-- ---------------------------------------------------------------------------

CREATE TABLE IF NOT EXISTS assessment_sessions (
    id                      UUID                        PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id               UUID                        NOT NULL,
    school_id               UUID                        NOT NULL,
    class_id                UUID                        NOT NULL,
    learning_area_id        UUID                        NOT NULL,
    academic_term_id        UUID                        NOT NULL,
    academic_year_id        UUID                        NOT NULL,
    name                    VARCHAR(255)                NOT NULL,
    evaluation_method       assessment_evaluation_method NOT NULL,
    max_points              NUMERIC(10,2)               NULL,
    grading_scale_profile_id UUID                       NULL REFERENCES grading_scale_profiles(id) ON DELETE SET NULL,
    status                  assessment_session_status   NOT NULL DEFAULT 'DRAFT',
    rejection_comment       TEXT                        NULL,
    submitted_by            UUID                        NULL REFERENCES users(id) ON DELETE SET NULL,
    approved_by             UUID                        NULL REFERENCES users(id) ON DELETE SET NULL,
    scheduled_date          DATE                        NULL,
    created_at              TIMESTAMPTZ                 NOT NULL DEFAULT NOW(),
    updated_at              TIMESTAMPTZ                 NOT NULL DEFAULT NOW(),
    created_by              UUID                        NOT NULL REFERENCES users(id),

    CONSTRAINT fk_assessment_sessions_tenant_school
        FOREIGN KEY (tenant_id, school_id)
        REFERENCES cbc_schools(tenant_id, id) ON DELETE CASCADE,
    CONSTRAINT fk_assessment_sessions_tenant_class
        FOREIGN KEY (tenant_id, class_id)
        REFERENCES cbc_classes(tenant_id, id) ON DELETE CASCADE,
    CONSTRAINT fk_assessment_sessions_tenant_term
        FOREIGN KEY (tenant_id, school_id, academic_term_id)
        REFERENCES academic_terms(tenant_id, school_id, id) ON DELETE CASCADE,
    CONSTRAINT fk_assessment_sessions_learning_area
        FOREIGN KEY (learning_area_id)
        REFERENCES cbc_learning_areas(id) ON DELETE CASCADE,
    CONSTRAINT uq_assessment_sessions_tenant UNIQUE (tenant_id, id),
    CONSTRAINT chk_quantitative_has_points CHECK (
        evaluation_method != 'QUANTITATIVE' OR max_points IS NOT NULL
    ),
    CONSTRAINT chk_quantitative_has_scale CHECK (
        evaluation_method != 'QUANTITATIVE' OR grading_scale_profile_id IS NOT NULL
    ),
    CONSTRAINT chk_rubric_no_points CHECK (
        evaluation_method != 'RUBRIC' OR max_points IS NULL
    ),
    CONSTRAINT chk_rubric_no_scale CHECK (
        evaluation_method != 'RUBRIC' OR grading_scale_profile_id IS NULL
    )
);

CREATE INDEX IF NOT EXISTS idx_assessment_sessions_tenant
    ON assessment_sessions (tenant_id);
CREATE INDEX IF NOT EXISTS idx_assessment_sessions_school
    ON assessment_sessions (school_id);
CREATE INDEX IF NOT EXISTS idx_assessment_sessions_class
    ON assessment_sessions (class_id);
CREATE INDEX IF NOT EXISTS idx_assessment_sessions_term
    ON assessment_sessions (academic_term_id);
CREATE INDEX IF NOT EXISTS idx_assessment_sessions_status
    ON assessment_sessions (status);
CREATE INDEX IF NOT EXISTS idx_assessment_sessions_learning_area
    ON assessment_sessions (learning_area_id);

DROP TRIGGER IF EXISTS trg_assessment_sessions_updated_at ON assessment_sessions;
CREATE TRIGGER trg_assessment_sessions_updated_at
    BEFORE UPDATE ON assessment_sessions
    FOR EACH ROW EXECUTE FUNCTION fn_set_updated_at();

COMMENT ON TABLE assessment_sessions IS
    'Tracks CBC assessment sessions through their lifecycle:
     DRAFT (teacher creating/grading) → PENDING_APPROVAL (submitted to admin)
     → PUBLISHED (approved, visible to parents). Rejection returns to DRAFT.
     Supports two evaluation methods: QUANTITATIVE (total marks converted via
     grading scale) and RUBRIC (direct indicator-level grading).';

COMMENT ON COLUMN assessment_sessions.max_points IS
    'Total possible marks for QUANTITATIVE sessions. NULL for RUBRIC sessions.
     Cannot be updated once any student score rows exist.';

COMMENT ON COLUMN assessment_sessions.rejection_comment IS
    'Admin feedback when rejecting a session. Cleared on re-submission.';

-- max_points write-once enforcement (000003 item 6b)
CREATE OR REPLACE FUNCTION fn_block_assessment_max_points_update()
RETURNS TRIGGER AS $$
BEGIN
    IF NEW.max_points IS DISTINCT FROM OLD.max_points THEN
        IF EXISTS (
            SELECT 1 FROM student_assessment_scores
            WHERE session_id = OLD.id
            LIMIT 1
        ) THEN
            RAISE EXCEPTION 'Cannot update max_points for session (id: %) — student assessment scores already exist for this session', OLD.id
                USING ERRCODE = 'P0002';  -- assigned application-level error code
        END IF;
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS trg_assessment_sessions_max_points_immutable ON assessment_sessions;
CREATE TRIGGER trg_assessment_sessions_max_points_immutable
    BEFORE UPDATE ON assessment_sessions
    FOR EACH ROW
    WHEN (OLD.max_points IS DISTINCT FROM NEW.max_points)
    EXECUTE FUNCTION fn_block_assessment_max_points_update();

COMMENT ON TRIGGER trg_assessment_sessions_max_points_immutable ON assessment_sessions IS
    'Enforces that max_points cannot be changed after any student assessment
     score rows reference this session. Throws error code P0002 which the
     application can catch specifically.';

-- ---------------------------------------------------------------------------
-- STUDENT ASSESSMENT SCORES
-- Stores raw scores for QUANTITATIVE sessions. Snapshots final performance
-- level at approval time for historical immutability.
-- ---------------------------------------------------------------------------

-- Create a helper function to validate raw_score <= max_points for the session
CREATE OR REPLACE FUNCTION max_points_check(session_id UUID, raw_score NUMERIC)
RETURNS BOOLEAN
LANGUAGE sql
STABLE
AS $$
    SELECT raw_score <= COALESCE((SELECT max_points FROM assessment_sessions WHERE id = session_id), raw_score);
$$;

CREATE TABLE IF NOT EXISTS student_assessment_scores (
    id                     UUID                  PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id              UUID                  NOT NULL,
    session_id             UUID                  NOT NULL,
    student_id             UUID                  NOT NULL,
    raw_score              NUMERIC(10,2)         NULL,
    calculated_percentage  NUMERIC(5,2)          NULL,
    final_performance_level cbc_performance_level NULL,
    enrollment_status      VARCHAR(20)           NOT NULL DEFAULT 'ACTIVE',
    created_at             TIMESTAMPTZ           NOT NULL DEFAULT NOW(),
    updated_at             TIMESTAMPTZ           NOT NULL DEFAULT NOW(),

    CONSTRAINT fk_scores_tenant_session
        FOREIGN KEY (tenant_id, session_id)
        REFERENCES assessment_sessions(tenant_id, id) ON DELETE CASCADE,
    CONSTRAINT fk_scores_tenant_student
        FOREIGN KEY (tenant_id, student_id)
        REFERENCES cbc_students(tenant_id, id) ON DELETE CASCADE,
    CONSTRAINT uq_score_session_student UNIQUE (session_id, student_id),
    CONSTRAINT chk_score_range CHECK (
        raw_score IS NULL OR (raw_score >= 0 AND max_points_check(session_id, raw_score))
    )
);

CREATE INDEX IF NOT EXISTS idx_student_scores_session
    ON student_assessment_scores (session_id);
CREATE INDEX IF NOT EXISTS idx_student_scores_student
    ON student_assessment_scores (student_id);
CREATE INDEX IF NOT EXISTS idx_student_scores_tenant
    ON student_assessment_scores (tenant_id);

DROP TRIGGER IF EXISTS trg_student_assessment_scores_updated_at ON student_assessment_scores;
CREATE TRIGGER trg_student_assessment_scores_updated_at
    BEFORE UPDATE ON student_assessment_scores
    FOR EACH ROW EXECUTE FUNCTION fn_set_updated_at();

COMMENT ON TABLE student_assessment_scores IS
    'Stores student scores for QUANTITATIVE assessment sessions. The
     final_performance_level is written (snapshotted) at the moment of
     admin approval — immune to later scale profile changes. NULL for
     RUBRIC sessions (those use student_assessment_outcome_grades).';

COMMENT ON COLUMN student_assessment_scores.enrollment_status IS
    'Denormalised enrollment status at time of grading. Used to enforce
     the No-Grade-Ghosting constraint: scores cannot be entered for
     students marked ABSENT or EXEMPT. Values: ACTIVE, SUSPENDED,
     TRANSFERRED, ABSENT, EXEMPT.';

COMMENT ON CONSTRAINT chk_score_range ON student_assessment_scores IS
    'Enforces that raw_score (when non-NULL) is non-negative AND does not exceed
     the session''s max_points. Fixed from original OR-bug which made this a no-op.';

-- ---------------------------------------------------------------------------
-- STUDENT ASSESSMENT OUTCOME GRADES
-- Stores rubric-level grades for RUBRIC sessions, linking student to
-- specific KICD performance indicators with the awarded CBC level.
-- ---------------------------------------------------------------------------

CREATE TABLE IF NOT EXISTS student_assessment_outcome_grades (
    id                      UUID                  PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id               UUID                  NOT NULL,
    session_id              UUID                  NOT NULL,
    student_id              UUID                  NOT NULL,
    performance_indicator_id UUID                 NOT NULL,
    awarded_level           cbc_performance_level NOT NULL,
    created_at              TIMESTAMPTZ           NOT NULL DEFAULT NOW(),
    updated_at              TIMESTAMPTZ           NOT NULL DEFAULT NOW(),

    CONSTRAINT fk_outcome_tenant_session
        FOREIGN KEY (tenant_id, session_id)
        REFERENCES assessment_sessions(tenant_id, id) ON DELETE CASCADE,
    CONSTRAINT fk_outcome_tenant_student
        FOREIGN KEY (tenant_id, student_id)
        REFERENCES cbc_students(tenant_id, id) ON DELETE CASCADE,
    CONSTRAINT fk_outcome_performance_indicator
        FOREIGN KEY (performance_indicator_id)
        REFERENCES performance_indicators(id) ON DELETE CASCADE,
    CONSTRAINT uq_outcome_session_student_indicator
        UNIQUE (session_id, student_id, performance_indicator_id)
);

CREATE INDEX IF NOT EXISTS idx_outcome_grades_session
    ON student_assessment_outcome_grades (session_id);
CREATE INDEX IF NOT EXISTS idx_outcome_grades_student
    ON student_assessment_outcome_grades (student_id);
CREATE INDEX IF NOT EXISTS idx_outcome_grades_indicator
    ON student_assessment_outcome_grades (performance_indicator_id);
CREATE INDEX IF NOT EXISTS idx_outcome_grades_tenant
    ON student_assessment_outcome_grades (tenant_id);

DROP TRIGGER IF EXISTS trg_student_assessment_outcome_grades_updated_at ON student_assessment_outcome_grades;
CREATE TRIGGER trg_student_assessment_outcome_grades_updated_at
    BEFORE UPDATE ON student_assessment_outcome_grades
    FOR EACH ROW EXECUTE FUNCTION fn_set_updated_at();

COMMENT ON TABLE student_assessment_outcome_grades IS
    'Stores rubric-level grades for RUBRIC assessment sessions. Each row
     maps a student to a specific KICD performance indicator with the
     awarded CBC level (EE, ME, AE, BE). No raw scores or percentages
     are stored — the teacher assigns the performance level directly.';

-- ============================================================================
-- RLS POLICIES — New Assessment & Grading Tables
-- ============================================================================

ALTER TABLE IF EXISTS grading_scale_profiles              ENABLE ROW LEVEL SECURITY;
ALTER TABLE IF EXISTS grading_scale_ranges                ENABLE ROW LEVEL SECURITY;
ALTER TABLE IF EXISTS assessment_sessions                 ENABLE ROW LEVEL SECURITY;
ALTER TABLE IF EXISTS student_assessment_scores           ENABLE ROW LEVEL SECURITY;
ALTER TABLE IF EXISTS student_assessment_outcome_grades   ENABLE ROW LEVEL SECURITY;

-- Extend the RLS policy loop to include new assessment tables
DO $$ DECLARE
    tbl TEXT;
BEGIN
    FOR tbl IN
        SELECT unnest(ARRAY[
            'grading_scale_profiles',
            'grading_scale_ranges',
            'assessment_sessions',
            'student_assessment_scores',
            'student_assessment_outcome_grades'
        ])
    LOOP
        EXECUTE format(
            'DROP POLICY IF EXISTS tenant_isolation_policy ON %I',
            tbl
        );
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

-- ============================================================================
-- MATERIALISED SUMMARY & ROLLUP TABLES
-- ----------------------------------------------------------------------------
-- Squashed from migrations 000005–000016. These tables and their refresh
-- functions/triggers are created directly in their final state.
-- ============================================================================

-- ============================================================================
-- PART 2: Create class_daily_attendance_summaries
-- ============================================================================

CREATE TABLE IF NOT EXISTS class_daily_attendance_summaries (
    id                    UUID          PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id             UUID          NOT NULL,
    school_id             UUID          NOT NULL,
    class_id              UUID          NOT NULL,
    academic_term_id      UUID          NOT NULL,
    date                  DATE          NOT NULL,
    total_enrolled        INT           NOT NULL,
    present_count         INT           NOT NULL,
    absent_count          INT           NOT NULL,
    late_count            INT           NOT NULL,
    excused_count         INT           NOT NULL,
    daily_attendance_rate NUMERIC(5,2)  NOT NULL,
    last_refreshed_at     TIMESTAMPTZ   NOT NULL DEFAULT NOW(),
    created_at            TIMESTAMPTZ   NOT NULL DEFAULT NOW(),
    updated_at            TIMESTAMPTZ   NOT NULL DEFAULT NOW(),

    CONSTRAINT uq_class_daily_attendance UNIQUE (class_id, date),
    CONSTRAINT fk_class_daily_tenant_class
        FOREIGN KEY (tenant_id, class_id)
        REFERENCES cbc_classes(tenant_id, id) ON DELETE CASCADE,
    CONSTRAINT fk_class_daily_tenant_term
        FOREIGN KEY (tenant_id, school_id, academic_term_id)
        REFERENCES academic_terms(tenant_id, school_id, id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_class_daily_tenant
    ON class_daily_attendance_summaries (tenant_id);
CREATE INDEX IF NOT EXISTS idx_class_daily_school
    ON class_daily_attendance_summaries (school_id);
CREATE INDEX IF NOT EXISTS idx_class_daily_class_date
    ON class_daily_attendance_summaries (class_id, date);
CREATE INDEX IF NOT EXISTS idx_class_daily_academic_term
    ON class_daily_attendance_summaries (academic_term_id);

DROP TRIGGER IF EXISTS trg_class_daily_attendance_summaries_updated_at
    ON class_daily_attendance_summaries;
CREATE TRIGGER trg_class_daily_attendance_summaries_updated_at
    BEFORE UPDATE ON class_daily_attendance_summaries
    FOR EACH ROW EXECUTE FUNCTION fn_set_updated_at();

COMMENT ON TABLE class_daily_attendance_summaries IS
    'Materialised rollup of attendance records per class per date. Populated
     by incremental background tasks triggered when all attendance for a
     class-date is marked (or on a timeout). "Total enrolled" is derived from
     distinct students who have attendance_records rows that day, not from
     cbc_student_enrollments.status, because enrollment status has no effective
     date within a term — a student suspended on day 50 would otherwise vanish
     from every earlier day too. This is a documented workaround, not a perfect fix.';

COMMENT ON COLUMN class_daily_attendance_summaries.daily_attendance_rate IS
    'Calculated as (present_count / (present_count + absent_count + late_count + excused_count)) * 100,
     stored as a decimal with two fractional digits (e.g. 94.60).';

-- Migration: 000006_create_student_term_subject_summaries
-- Creates the student_term_subject_summaries materialised table and the
-- PostgreSQL function + triggers that keep it in sync when assessment
-- sessions are published.
--
-- Grain: (student_id, academic_term_id, learning_area_id)
--
-- This table is a blended rollup of quantitative scores and rubric outcome
-- grades across all PUBLISHED assessment sessions for a given student,
-- term, and learning area.
--
-- Quantitative scores contribute their calculated_percentage directly.
-- Rubric outcome grades are converted to a percentage using the
-- grading_scale_ranges.default_percentage_mapping for the awarded level.
-- Both sources are then averaged together into average_percentage.
--
-- The has_quantitative_data and has_rubric_data flags let the UI render
-- the result honestly — a blended average from rubric-only data implies
-- false precision that these flags help the report avoid.

-- ============================================================================
-- TABLE: student_term_subject_summaries
-- ============================================================================

CREATE TABLE IF NOT EXISTS student_term_subject_summaries (
    id                             UUID          PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id                      UUID          NOT NULL,
    school_id                      UUID          NOT NULL,
    student_id                     UUID          NOT NULL,
    academic_term_id               UUID          NOT NULL,
    learning_area_id               UUID          NOT NULL,
    average_percentage             NUMERIC(5,2),
    mapped_performance_level       VARCHAR(5),
    quantitative_assessment_count  INT           NOT NULL DEFAULT 0,
    rubric_assessment_count        INT           NOT NULL DEFAULT 0,
    indicators_assessed_count      INT           NOT NULL DEFAULT 0,
    has_quantitative_data          BOOLEAN       NOT NULL DEFAULT false,
    has_rubric_data                BOOLEAN       NOT NULL DEFAULT false,
    teacher_remark                 TEXT,
    last_refreshed_at              TIMESTAMPTZ   NOT NULL DEFAULT NOW(),
    created_at                     TIMESTAMPTZ   NOT NULL DEFAULT NOW(),
    updated_at                     TIMESTAMPTZ   NOT NULL DEFAULT NOW(),

    CONSTRAINT uq_student_term_subject UNIQUE (student_id, academic_term_id, learning_area_id),
    CONSTRAINT fk_summaries_tenant_student
        FOREIGN KEY (tenant_id, student_id)
        REFERENCES cbc_students(tenant_id, id) ON DELETE CASCADE,
    CONSTRAINT fk_summaries_tenant_term
        FOREIGN KEY (tenant_id, school_id, academic_term_id)
        REFERENCES academic_terms(tenant_id, school_id, id) ON DELETE CASCADE,
    CONSTRAINT fk_summaries_tenant_school
        FOREIGN KEY (tenant_id, school_id)
        REFERENCES cbc_schools(tenant_id, id) ON DELETE CASCADE,
    CONSTRAINT fk_summaries_learning_area
        FOREIGN KEY (learning_area_id)
        REFERENCES cbc_learning_areas(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_term_subject_summaries_tenant
    ON student_term_subject_summaries (tenant_id);
CREATE INDEX IF NOT EXISTS idx_term_subject_summaries_school
    ON student_term_subject_summaries (school_id);
CREATE INDEX IF NOT EXISTS idx_term_subject_summaries_student_term
    ON student_term_subject_summaries (student_id, academic_term_id);
CREATE INDEX IF NOT EXISTS idx_term_subject_summaries_term_area
    ON student_term_subject_summaries (academic_term_id, learning_area_id);

DROP TRIGGER IF EXISTS trg_student_term_subject_summaries_updated_at
    ON student_term_subject_summaries;
CREATE TRIGGER trg_student_term_subject_summaries_updated_at
    BEFORE UPDATE ON student_term_subject_summaries
    FOR EACH ROW EXECUTE FUNCTION fn_set_updated_at();

COMMENT ON TABLE student_term_subject_summaries IS
    'Materialised blended rollup of assessment results per student, term,
     and learning area. Populated via fn_refresh_term_subject_summary()
     when an assessment session transitions to PUBLISHED. Quantitative
     scores contribute their calculated_percentage directly; rubric
     outcome grades are converted via default_percentage_mapping. The
     has_quantitative_data / has_rubric_data flags let reports distinguish
     blended-averages from single-source averages.';

COMMENT ON COLUMN student_term_subject_summaries.average_percentage IS
    'Weighted average across all PUBLISHED assessment scores for this
     student+term+learning_area. Rubric outcomes are mapped to a percentage
     via grading_scale_ranges.default_percentage_mapping for the awarded
     level, then blended with quantitative percentages. NULL when neither
     quantitative nor rubric data exists.';

COMMENT ON COLUMN student_term_subject_summaries.mapped_performance_level IS
    'The CBC performance level (EE/ME/AE/BE) corresponding to
     average_percentage, determined by the grading scale profile used in
     the most recent QUANTITATIVE session for this term+learning_area.
     NULL when no scale profile can be resolved.';

COMMENT ON COLUMN student_term_subject_summaries.teacher_remark IS
    'Optional free-text remark entered by the subject teacher during term-end
     compilation. Not populated automatically — set via API by the teacher.';

-- ============================================================================
-- FUNCTION: fn_refresh_term_subject_summary_for_session(session_id UUID)
--
-- Recomputes student_term_subject_summaries for all students in the given
-- session, scoped to the session's academic_term_id and learning_area_id.
--
-- The algorithm:
--   1. Gathers the session's tenant_id, school_id, academic_term_id,
--      learning_area_id, and grading_scale_profile_id (if any).
--   2. For each student who has scores or grades in this session, scans ALL
--      PUBLISHED sessions for the same term+learning_area.
--   3. Quantitative scores contribute calculated_percentage directly.
--   4. Rubric outcome grades are converted via default_percentage_mapping
--      of the grading_scale_ranges matching the awarded_level. The first
--      matching range from any active profile belonging to the school is
--      used (MIN default_percentage_mapping for determinism).
--   5. Blends all resolved percentages into a single average.
--   6. Maps the average to a performance level using the grading scale
--      profile from the most recent QUANTITATIVE session.
--   7. Upserts into student_term_subject_summaries.
-- ============================================================================

CREATE OR REPLACE FUNCTION fn_refresh_term_subject_summary_for_session(target_session_id UUID)
RETURNS void
LANGUAGE plpgsql
AS $$
DECLARE
    v_tenant_id          UUID;
    v_school_id          UUID;
    v_academic_term_id   UUID;
    v_learning_area_id   UUID;
    v_scale_profile_id   UUID;
BEGIN
    -- Resolve session metadata
    SELECT tenant_id, school_id, academic_term_id, learning_area_id, grading_scale_profile_id
    INTO v_tenant_id, v_school_id, v_academic_term_id, v_learning_area_id, v_scale_profile_id
    FROM assessment_sessions
    WHERE id = target_session_id;

    IF NOT FOUND THEN
        RETURN;
    END IF;

    -- If this session has no grading_scale_profile_id, try to find one
    -- from the most recent QUANTITATIVE PUBLISHED session for the same
    -- term and learning area (for mapping the blended average).
    IF v_scale_profile_id IS NULL THEN
        SELECT grading_scale_profile_id
        INTO v_scale_profile_id
        FROM assessment_sessions
        WHERE academic_term_id = v_academic_term_id
          AND learning_area_id = v_learning_area_id
          AND status = 'PUBLISHED'
          AND evaluation_method = 'QUANTITATIVE'
          AND grading_scale_profile_id IS NOT NULL
        ORDER BY updated_at DESC
        LIMIT 1;
    END IF;

    -- Upsert summary for each student who has data in this session
    INSERT INTO student_term_subject_summaries (
        tenant_id, school_id, student_id, academic_term_id, learning_area_id,
        average_percentage, mapped_performance_level,
        quantitative_assessment_count, rubric_assessment_count,
        indicators_assessed_count,
        has_quantitative_data, has_rubric_data,
        last_refreshed_at
    )
    WITH affected_students AS (
        SELECT student_id FROM student_assessment_scores WHERE session_id = target_session_id
        UNION
        SELECT student_id FROM student_assessment_outcome_grades WHERE session_id = target_session_id
    ),
    all_published_scores AS (
        -- Quantitative scores from all PUBLISHED sessions for this term+area
        SELECT
            sas.student_id,
            sas.calculated_percentage AS resolved_pct,
            'QUANTITATIVE'::TEXT      AS source_type,
            sas.session_id            AS src_session_id,
            NULL::TEXT                AS indicator_id
        FROM student_assessment_scores sas
        JOIN assessment_sessions s ON s.id = sas.session_id
        WHERE s.academic_term_id = v_academic_term_id
          AND s.learning_area_id = v_learning_area_id
          AND s.status = 'PUBLISHED'
          AND sas.calculated_percentage IS NOT NULL

        UNION ALL

        -- Rubric outcome grades from all PUBLISHED sessions for this term+area
        -- Convert awarded_level → percentage via default_percentage_mapping
        SELECT
            sog.student_id,
            r.default_percentage_mapping AS resolved_pct,
            'RUBRIC'::TEXT               AS source_type,
            sog.session_id               AS src_session_id,
            sog.performance_indicator_id::TEXT AS indicator_id
        FROM student_assessment_outcome_grades sog
        JOIN assessment_sessions s ON s.id = sog.session_id
        LEFT JOIN grading_scale_ranges r
            ON r.performance_level = sog.awarded_level
            AND r.default_percentage_mapping IS NOT NULL
        WHERE s.academic_term_id = v_academic_term_id
          AND s.learning_area_id = v_learning_area_id
          AND s.status = 'PUBLISHED'
    ),
    filtered AS (
        -- Only keep rows with a resolved percentage
        SELECT * FROM all_published_scores WHERE resolved_pct IS NOT NULL
    ),
    aggregations AS (
        SELECT
            student_id,
            ROUND(AVG(resolved_pct)::numeric, 2) AS average_percentage,
            COUNT(DISTINCT CASE WHEN source_type = 'QUANTITATIVE' THEN src_session_id END) AS quantitative_assessment_count,
            COUNT(DISTINCT CASE WHEN source_type = 'RUBRIC' THEN src_session_id END) AS rubric_assessment_count,
            COUNT(DISTINCT CASE WHEN source_type = 'RUBRIC' THEN indicator_id END) AS indicators_assessed_count,
            BOOL_OR(source_type = 'QUANTITATIVE') AS has_quantitative_data,
            BOOL_OR(source_type = 'RUBRIC') AS has_rubric_data
        FROM filtered
        GROUP BY student_id
    )
    SELECT
        v_tenant_id,
        v_school_id,
        a.student_id,
        v_academic_term_id,
        v_learning_area_id,
        a.average_percentage,
        CASE
            WHEN a.average_percentage IS NOT NULL AND v_scale_profile_id IS NOT NULL THEN (
                SELECT r.performance_level::TEXT
                FROM grading_scale_ranges r
                WHERE r.profile_id = v_scale_profile_id
                  AND a.average_percentage >= r.min_percentage
                  AND a.average_percentage <= r.max_percentage
                LIMIT 1
            )
            ELSE NULL
        END AS mapped_performance_level,
        a.quantitative_assessment_count,
        a.rubric_assessment_count,
        a.indicators_assessed_count,
        a.has_quantitative_data,
        a.has_rubric_data,
        NOW()
    FROM aggregations a
    JOIN affected_students aff ON aff.student_id = a.student_id
    ON CONFLICT (student_id, academic_term_id, learning_area_id)
    DO UPDATE SET
        average_percentage            = EXCLUDED.average_percentage,
        mapped_performance_level      = EXCLUDED.mapped_performance_level,
        quantitative_assessment_count = EXCLUDED.quantitative_assessment_count,
        rubric_assessment_count       = EXCLUDED.rubric_assessment_count,
        indicators_assessed_count     = EXCLUDED.indicators_assessed_count,
        has_quantitative_data         = EXCLUDED.has_quantitative_data,
        has_rubric_data               = EXCLUDED.has_rubric_data,
        last_refreshed_at             = NOW(),
        updated_at                    = NOW();

    -- Also clean up any orphaned summary rows for these students where the
    -- student is no longer in any PUBLISHED session for this term+area
    -- (e.g. if scores were deleted and the session re-published). We only
    -- remove rows where both counts are zero, preserving teacher_remark.
    DELETE FROM student_term_subject_summaries
    WHERE academic_term_id = v_academic_term_id
      AND learning_area_id = v_learning_area_id
      AND student_id IN (
          SELECT student_id FROM student_assessment_scores WHERE session_id = target_session_id
          UNION
          SELECT student_id FROM student_assessment_outcome_grades WHERE session_id = target_session_id
      )
      AND quantitative_assessment_count = 0
      AND rubric_assessment_count = 0
      AND teacher_remark IS NULL;
END;
$$;

COMMENT ON FUNCTION fn_refresh_term_subject_summary_for_session IS
    'Refreshes student_term_subject_summaries for all students in the given
     session. Called automatically when an assessment session transitions
     to PUBLISHED, or manually via the /refresh API endpoint.';

-- ============================================================================
-- RLS POLICY
-- ============================================================================

ALTER TABLE IF EXISTS student_term_subject_summaries ENABLE ROW LEVEL SECURITY;

DO $$ BEGIN
    DROP POLICY IF EXISTS tenant_isolation_policy ON student_term_subject_summaries;
    CREATE POLICY tenant_isolation_policy ON student_term_subject_summaries
        FOR ALL
        USING (tenant_id = fn_current_tenant_id())
        WITH CHECK (tenant_id = fn_current_tenant_id());
END $$;

COMMENT ON TABLE student_term_subject_summaries IS
    'Materialised blended rollup of assessment results per student, term,
     and learning area. RLS-enabled — tenant-scoped.';

-- Migration: 000007_create_student_term_overall_summaries
-- Creates the student_term_overall_summaries materialised table and the
-- PostgreSQL function that computes the rolling term-level aggregate from
-- student_term_subject_summaries.
--
-- Grain: (student_id, academic_term_id)
--
-- This table is a second-level rollup. For each student+term it counts how
-- many learning areas have data, computes the overall mean percentage, maps
-- it to a CBC performance level (EE/ME/AE/BE), breaks out per-level counts,
-- and — crucially — flags whether a KPSEA/KJSEA/KSSEA weighting formula was
-- used (is_weighted_exam_score). This lets parent-facing report cards and
-- routine progress checks display different math transparently.
--
-- Weighting logic:
--   When academic_terms.is_final = true AND the grade level is a national
--   exam year (G6 → KPSEA, G9 → KJSEA, G12 → KSSEA), the function pulls the
--   matching formula from assessment_weight_configs and computes a weighted
--   blend instead of a plain average across subjects.

-- ============================================================================
-- TABLE: student_term_overall_summaries
-- ============================================================================

CREATE TABLE IF NOT EXISTS student_term_overall_summaries (
    id                       UUID          PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id                UUID          NOT NULL,
    school_id                UUID          NOT NULL,
    student_id               UUID          NOT NULL,
    academic_term_id         UUID          NOT NULL,
    subjects_assessed_count  INT           NOT NULL DEFAULT 0,
    overall_mean_percentage  NUMERIC(5,2),
    overall_performance_level VARCHAR(5),
    exceeding_count          INT           NOT NULL DEFAULT 0,
    meeting_count            INT           NOT NULL DEFAULT 0,
    approaching_count        INT           NOT NULL DEFAULT 0,
    below_count              INT           NOT NULL DEFAULT 0,
    is_weighted_exam_score   BOOLEAN       NOT NULL DEFAULT false,
    headteacher_remark       TEXT,
    last_refreshed_at        TIMESTAMPTZ   NOT NULL DEFAULT NOW(),
    created_at               TIMESTAMPTZ   NOT NULL DEFAULT NOW(),
    updated_at               TIMESTAMPTZ   NOT NULL DEFAULT NOW(),

    CONSTRAINT uq_student_term_overall UNIQUE (student_id, academic_term_id),
    CONSTRAINT fk_overall_tenant_student
        FOREIGN KEY (tenant_id, student_id)
        REFERENCES cbc_students(tenant_id, id) ON DELETE CASCADE,
    CONSTRAINT fk_overall_tenant_term
        FOREIGN KEY (tenant_id, school_id, academic_term_id)
        REFERENCES academic_terms(tenant_id, school_id, id) ON DELETE CASCADE,
    CONSTRAINT fk_overall_tenant_school
        FOREIGN KEY (tenant_id, school_id)
        REFERENCES cbc_schools(tenant_id, id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_overall_summaries_tenant
    ON student_term_overall_summaries (tenant_id);
CREATE INDEX IF NOT EXISTS idx_overall_summaries_school
    ON student_term_overall_summaries (school_id);
CREATE INDEX IF NOT EXISTS idx_overall_summaries_student_term
    ON student_term_overall_summaries (student_id, academic_term_id);
CREATE INDEX IF NOT EXISTS idx_overall_summaries_term
    ON student_term_overall_summaries (academic_term_id);

DROP TRIGGER IF EXISTS trg_student_term_overall_summaries_updated_at
    ON student_term_overall_summaries;
CREATE TRIGGER trg_student_term_overall_summaries_updated_at
    BEFORE UPDATE ON student_term_overall_summaries
    FOR EACH ROW EXECUTE FUNCTION fn_set_updated_at();

COMMENT ON TABLE student_term_overall_summaries IS
    'Second-level term rollup per student. Aggregates across all learning-area
     summaries to produce an overall mean, performance level, and per-level
     counts. The is_weighted_exam_score flag tells consumers whether a KNEC
     national-exam weighting formula was applied. Populated on-demand via
     fn_compute_term_overall_summaries_for_term() or nightly batch.';

COMMENT ON COLUMN student_term_overall_summaries.subjects_assessed_count IS
    'Number of learning areas that have at least one published assessment
     score/grade in this term (i.e. rows in student_term_subject_summaries).';

COMMENT ON COLUMN student_term_overall_summaries.overall_mean_percentage IS
    'Mean of all per-subject average_percentage values. For non-final terms
     this is a straight average; for final exam terms (G6/G9/G12) it is a
     weighted blend using assessment_weight_configs. NULL when no subject
     data exists.';

COMMENT ON COLUMN student_term_overall_summaries.overall_performance_level IS
    'The CBC performance level (EE/ME/AE/BE) corresponding to
     overall_mean_percentage, determined by the grading scale profile in use
     for the student''s grade level. NULL when not resolvable.';

COMMENT ON COLUMN student_term_overall_summaries.is_weighted_exam_score IS
    'TRUE when a KNEC weighting formula (from assessment_weight_configs) was
     applied instead of a plain average. This prevents silent math changes
     between parent-facing report cards and routine progress checks.';

COMMENT ON COLUMN student_term_overall_summaries.headteacher_remark IS
    'Optional free-text remark entered by the headteacher during term-end
     compilation. Not populated automatically — set via API.';

-- ============================================================================
-- FUNCTION: fn_compute_term_overall_summaries_for_term(target_term_id UUID)
--
-- Computes (or recomputes) student_term_overall_summaries for ALL students
-- enrolled in the given academic term.
--
-- Algorithm:
--   1. Resolves the term's tenant_id, school_id, grade_level from the
--      class associated with each student's enrollment.
--   2. Checks academic_terms.is_final. If true AND grade_level is an exam
--      year (G6/G9/G12), looks up assessment_weight_configs for the
--      matching target_exam (KPSEA/KJSEA/KSSEA) and effective_from year.
--   3. For each student enrolled in the term, reads all subject summaries
--      from student_term_subject_summaries:
--      a. Non-weighted: overall_mean_percentage = plain average of all
--         average_percentage values.
--      b. Weighted:   overall_mean_percentage = weighted average using
--         config weights. Each subject's average_percentage is multiplied
--         by the matching config weight, summed, and divided by total weight.
--   4. Maps overall_mean_percentage to a CBC level using the school's
--      active grading scale profile.
--   5. Counts how many subjects fall into each level (EE/ME/AE/BE).
--   6. Upserts into student_term_overall_summaries.
--   7. Cleans up rows for students no longer enrolled in the term.
-- ============================================================================

CREATE OR REPLACE FUNCTION fn_compute_term_overall_summaries_for_term(target_term_id UUID)
RETURNS void
LANGUAGE plpgsql
AS $$
DECLARE
    v_tenant_id         UUID;
    v_school_id         UUID;
    v_is_final          BOOLEAN;
    v_grade_level       TEXT;
    v_target_exam       TEXT;
    v_effective_from    INT;
    v_weight_total      NUMERIC;
    v_scale_profile_id  UUID;
BEGIN
    -- Resolve term metadata
    SELECT tenant_id, school_id, is_final
    INTO v_tenant_id, v_school_id, v_is_final
    FROM academic_terms
    WHERE id = target_term_id;

    IF NOT FOUND THEN
        RETURN;
    END IF;

    -- Determine if this is an exam-grade final term
    v_target_exam := CASE
        WHEN v_is_final THEN
            -- Determine target_exam from the grade level of classes in this term
            -- We look at the highest grade level enrolled; for a school this term
            -- may serve multiple grades, but the weight config applies per student.
            NULL
        ELSE NULL
    END;

    -- Get the effective_from year from the academic year
    SELECT ay.name::INT
    INTO v_effective_from
    FROM academic_years ay
    JOIN academic_terms t ON t.academic_year_id = ay.id
    WHERE t.id = target_term_id;

    -- Pre-query weight configs if this is a final term (each student's
    -- grade level is resolved per-row below). We cache them in a temp table
    -- to avoid repeated lookups.
    CREATE TEMP TABLE IF NOT EXISTS tmp_weight_configs (
        grade_level         TEXT,
        assessment_type_code TEXT,
        weight_percent      NUMERIC(5,2)
    ) ON COMMIT DROP;

    -- =====================================================================
    -- Main UPSERT: for every student enrolled in this term
    -- =====================================================================
    INSERT INTO student_term_overall_summaries (
        tenant_id, school_id, student_id, academic_term_id,
        subjects_assessed_count, overall_mean_percentage, overall_performance_level,
        exceeding_count, meeting_count, approaching_count, below_count,
        is_weighted_exam_score, last_refreshed_at
    )
    WITH enrolled_students AS (
        -- All students enrolled in this term with their grade level
        SELECT DISTINCT ON (e.student_id)
            e.student_id,
            e.tenant_id,
            e.school_id,
            c.grade_level::TEXT AS grade_level
        FROM cbc_student_enrollments e
        LEFT JOIN cbc_classes c ON c.id = e.class_id
        WHERE e.academic_term_id = target_term_id
          AND (e.status = 'ACTIVE' OR e.status = 'COMPLETED_CYCLE')
    ),
    subject_summaries AS (
        -- All subject-level summaries for these students in this term
        SELECT
            s.student_id,
            s.average_percentage,
            s.mapped_performance_level
        FROM student_term_subject_summaries s
        WHERE s.academic_term_id = target_term_id
          AND s.average_percentage IS NOT NULL
    ),
    resolved_weight_configs AS (
        -- For exam-grade final terms, pull the matching weight formulas.
        -- Map grade level to target exam: G6→KPSEA, G9→KJSEA, G12→KSSEA.
        SELECT
            es.student_id,
            wc.assessment_type_code,
            wc.weight_percent
        FROM enrolled_students es
        CROSS JOIN LATERAL (
            SELECT wc.assessment_type_code, wc.weight_percent
            FROM assessment_weight_configs wc
            WHERE wc.grade_level::TEXT = es.grade_level
              AND wc.effective_from = v_effective_from
              AND wc.target_exam = CASE
                    WHEN es.grade_level = 'G6' THEN 'KPSEA'
                    WHEN es.grade_level = 'G9' THEN 'KJSEA'
                    WHEN es.grade_level = 'G12' THEN 'KSSEA'
                    ELSE NULL
                  END
        ) wc
        WHERE v_is_final
          AND es.grade_level IN ('G6', 'G9', 'G12')
    ),
    per_student_aggregates AS (
        SELECT
            es.student_id,
            es.tenant_id,
            es.school_id,
            es.grade_level,
            COUNT(ss.average_percentage)::INT AS subjects_assessed_count,
            CASE
                -- Weighted: use config-based weighted average
                WHEN COUNT(rwc.student_id) > 0 THEN (
                    SELECT ROUND(
                        SUM(ss2.average_percentage * rwc2.weight_percent) /
                        NULLIF(SUM(rwc2.weight_percent), 0)::NUMERIC
                    , 2)
                    FROM subject_summaries ss2
                    JOIN resolved_weight_configs rwc2 ON rwc2.student_id = ss2.student_id
                    WHERE ss2.student_id = es.student_id
                )
                -- Non-weighted: plain average across subjects
                ELSE ROUND(AVG(ss.average_percentage), 2)
            END AS overall_mean_percentage,
            BOOL_OR(rwc.student_id IS NOT NULL) AS is_weighted
        FROM enrolled_students es
        LEFT JOIN subject_summaries ss ON ss.student_id = es.student_id
        LEFT JOIN resolved_weight_configs rwc ON rwc.student_id = es.student_id
        GROUP BY es.student_id, es.tenant_id, es.school_id, es.grade_level
    ),
    level_counts AS (
        SELECT
            student_id,
            COUNT(*) FILTER (WHERE mapped_performance_level = 'EE') AS exceeding_count,
            COUNT(*) FILTER (WHERE mapped_performance_level = 'ME') AS meeting_count,
            COUNT(*) FILTER (WHERE mapped_performance_level = 'AE') AS approaching_count,
            COUNT(*) FILTER (WHERE mapped_performance_level = 'BE') AS below_count
        FROM subject_summaries
        GROUP BY student_id
    ),
    scale_profile AS (
        -- Find the first active grading scale profile for this school
        SELECT id FROM grading_scale_profiles
        WHERE school_id = v_school_id AND is_active = true
        ORDER BY created_at DESC
        LIMIT 1
    )
    SELECT
        pa.tenant_id,
        pa.school_id,
        pa.student_id,
        target_term_id,
        pa.subjects_assessed_count,
        pa.overall_mean_percentage,
        CASE
            WHEN pa.overall_mean_percentage IS NOT NULL AND sp.id IS NOT NULL THEN (
                SELECT r.performance_level::TEXT
                FROM grading_scale_ranges r
                WHERE r.profile_id = sp.id
                  AND pa.overall_mean_percentage >= r.min_percentage
                  AND pa.overall_mean_percentage <= r.max_percentage
                LIMIT 1
            )
            ELSE NULL
        END AS overall_performance_level,
        COALESCE(lc.exceeding_count, 0),
        COALESCE(lc.meeting_count, 0),
        COALESCE(lc.approaching_count, 0),
        COALESCE(lc.below_count, 0),
        COALESCE(pa.is_weighted, false),
        NOW()
    FROM per_student_aggregates pa
    LEFT JOIN level_counts lc ON lc.student_id = pa.student_id
    CROSS JOIN scale_profile sp
    WHERE pa.subjects_assessed_count > 0

    ON CONFLICT (student_id, academic_term_id)
    DO UPDATE SET
        subjects_assessed_count     = EXCLUDED.subjects_assessed_count,
        overall_mean_percentage     = EXCLUDED.overall_mean_percentage,
        overall_performance_level   = EXCLUDED.overall_performance_level,
        exceeding_count             = EXCLUDED.exceeding_count,
        meeting_count               = EXCLUDED.meeting_count,
        approaching_count           = EXCLUDED.approaching_count,
        below_count                 = EXCLUDED.below_count,
        is_weighted_exam_score      = EXCLUDED.is_weighted_exam_score,
        last_refreshed_at           = NOW(),
        updated_at                  = NOW();

    -- Clean up orphaned rows (students no longer enrolled)
    DELETE FROM student_term_overall_summaries
    WHERE academic_term_id = target_term_id
      AND student_id NOT IN (
          SELECT student_id FROM cbc_student_enrollments
          WHERE academic_term_id = target_term_id
            AND (status = 'ACTIVE' OR status = 'COMPLETED_CYCLE')
      )
      AND headteacher_remark IS NULL;
END;
$$;

COMMENT ON FUNCTION fn_compute_term_overall_summaries_for_term IS
    'Computes student_term_overall_summaries for all students enrolled in the
     given academic term. Applies KNEC weighting formulas when the term is a
     final exam term (G6→KPSEA, G9→KJSEA, G12→KSSEA). Call on-demand via the
     /refresh API or in a nightly cron job.';

-- ============================================================================
-- FUNCTION: fn_compute_single_student_term_overall_summary
--
-- Convenience wrapper that computes the overall summary for a single
-- student+term pair. Useful for on-demand refresh after a subject summary
-- is updated (e.g. when an assessment is published).
-- ============================================================================

CREATE OR REPLACE FUNCTION fn_compute_single_student_term_overall_summary(
    p_student_id UUID,
    p_term_id    UUID
)
RETURNS void
LANGUAGE plpgsql
AS $$
DECLARE
    v_tenant_id  UUID;
    v_school_id  UUID;
    v_is_final   BOOLEAN;
    v_grade_level TEXT;
    v_target_exam TEXT;
    v_effective_from INT;
BEGIN
    -- Get term info
    SELECT tenant_id, school_id, is_final
    INTO v_tenant_id, v_school_id, v_is_final
    FROM academic_terms WHERE id = p_term_id;

    IF NOT FOUND THEN
        RETURN;
    END IF;

    -- Get student's grade level from current enrollment
    SELECT c.grade_level::TEXT
    INTO v_grade_level
    FROM cbc_student_enrollments e
    JOIN cbc_classes c ON c.id = e.class_id
    WHERE e.student_id = p_student_id
      AND e.academic_term_id = p_term_id
      AND (e.status = 'ACTIVE' OR e.status = 'COMPLETED_CYCLE')
    LIMIT 1;

    -- Get effective year
    SELECT ay.name::INT INTO v_effective_from
    FROM academic_years ay
    JOIN academic_terms t ON t.academic_year_id = ay.id
    WHERE t.id = p_term_id;

    -- Upsert the single-student overall summary
    INSERT INTO student_term_overall_summaries (
        tenant_id, school_id, student_id, academic_term_id,
        subjects_assessed_count, overall_mean_percentage, overall_performance_level,
        exceeding_count, meeting_count, approaching_count, below_count,
        is_weighted_exam_score, last_refreshed_at
    )
    WITH subject_summaries AS (
        SELECT average_percentage, mapped_performance_level
        FROM student_term_subject_summaries
        WHERE student_id = p_student_id
          AND academic_term_id = p_term_id
          AND average_percentage IS NOT NULL
    ),
    weight_configs AS (
        SELECT weight_percent
        FROM assessment_weight_configs
        WHERE grade_level::TEXT = v_grade_level
          AND effective_from = v_effective_from
          AND target_exam = CASE
                WHEN v_grade_level = 'G6' THEN 'KPSEA'
                WHEN v_grade_level = 'G9' THEN 'KJSEA'
                WHEN v_grade_level = 'G12' THEN 'KSSEA'
                ELSE NULL
              END
    ),
    weighted_ok AS (
        SELECT v_is_final
           AND v_grade_level IN ('G6', 'G9', 'G12')
           AND EXISTS (SELECT 1 FROM weight_configs) AS use_weighted
    ),
    agg AS (
        SELECT
            COUNT(*)::INT AS subjects_assessed_count,
            CASE
                WHEN wo.use_weighted THEN (
                    SELECT ROUND(
                        SUM(ss.average_percentage * wc.weight_percent) /
                        NULLIF(SUM(wc.weight_percent), 0)::NUMERIC
                    , 2)
                    FROM subject_summaries ss
                    CROSS JOIN weight_configs wc
                )
                ELSE ROUND(AVG(ss.average_percentage), 2)
            END AS overall_mean_percentage,
            COALESCE(wo.use_weighted, false) AS is_weighted
        FROM subject_summaries ss
        CROSS JOIN weighted_ok wo
    ),
    lc AS (
        SELECT
            COUNT(*) FILTER (WHERE mapped_performance_level = 'EE') AS exceeding_count,
            COUNT(*) FILTER (WHERE mapped_performance_level = 'ME') AS meeting_count,
            COUNT(*) FILTER (WHERE mapped_performance_level = 'AE') AS approaching_count,
            COUNT(*) FILTER (WHERE mapped_performance_level = 'BE') AS below_count
        FROM subject_summaries
    ),
    sp AS (
        SELECT id FROM grading_scale_profiles
        WHERE school_id = v_school_id AND is_active = true
        ORDER BY created_at DESC LIMIT 1
    )
    SELECT
        v_tenant_id, v_school_id, p_student_id, p_term_id,
        agg.subjects_assessed_count,
        agg.overall_mean_percentage,
        CASE
            WHEN agg.overall_mean_percentage IS NOT NULL AND sp.id IS NOT NULL THEN (
                SELECT r.performance_level::TEXT
                FROM grading_scale_ranges r
                WHERE r.profile_id = sp.id
                  AND agg.overall_mean_percentage >= r.min_percentage
                  AND agg.overall_mean_percentage <= r.max_percentage
                LIMIT 1
            )
            ELSE NULL
        END,
        COALESCE(lc.exceeding_count, 0),
        COALESCE(lc.meeting_count, 0),
        COALESCE(lc.approaching_count, 0),
        COALESCE(lc.below_count, 0),
        agg.is_weighted,
        NOW()
    FROM agg, lc, sp
    ON CONFLICT (student_id, academic_term_id)
    DO UPDATE SET
        subjects_assessed_count     = EXCLUDED.subjects_assessed_count,
        overall_mean_percentage     = EXCLUDED.overall_mean_percentage,
        overall_performance_level   = EXCLUDED.overall_performance_level,
        exceeding_count             = EXCLUDED.exceeding_count,
        meeting_count               = EXCLUDED.meeting_count,
        approaching_count           = EXCLUDED.approaching_count,
        below_count                 = EXCLUDED.below_count,
        is_weighted_exam_score      = EXCLUDED.is_weighted_exam_score,
        last_refreshed_at           = NOW(),
        updated_at                  = NOW();
END;
$$;

COMMENT ON FUNCTION fn_compute_single_student_term_overall_summary IS
    'Computes the overall summary for a single student+term. Useful for
     on-demand refresh when subject summaries change.';

-- ============================================================================
-- RLS POLICY
-- ============================================================================

ALTER TABLE IF EXISTS student_term_overall_summaries ENABLE ROW LEVEL SECURITY;

DO $$ BEGIN
    DROP POLICY IF EXISTS tenant_isolation_policy ON student_term_overall_summaries;
    CREATE POLICY tenant_isolation_policy ON student_term_overall_summaries
        FOR ALL
        USING (tenant_id = fn_current_tenant_id())
        WITH CHECK (tenant_id = fn_current_tenant_id());
END $$;

COMMENT ON TABLE student_term_overall_summaries IS
    'Second-level term rollup per student. RLS-enabled — tenant-scoped.';

-- Migration: 000008_create_student_cohort_position_summaries
-- Creates the student_cohort_position_summaries materialised table and the
-- PostgreSQL function that computes class and grade rankings for all students
-- enrolled in a given academic term.
--
-- Grain: (student_id, class_id, academic_term_id)
--
-- This is a periodic batch-only table. It is NEVER updated incrementally;
-- ranking one student requires knowing every other student's score in the
-- same class and grade for that term. Computation is triggered on a schedule
-- (e.g. every 30 minutes during active grading windows) via the batch
-- function fn_compute_cohort_positions_for_term().
--
-- Data sources:
--   - Student's current class:  cbc_student_enrollments (status = ACTIVE)
--   - Student's overall score:  student_term_overall_summaries.overall_mean_percentage
--
-- Computed fields explained:
--   class_rank            = position within the class (1 = highest score)
--   class_headcount       = total number of scored students in the class
--   class_percentile      = (class_headcount - class_rank) / class_headcount * 100
--   grade_rank            = position within the same grade_level across the school
--   grade_headcount       = total number of scored students in the grade
--   grade_percentile      = (grade_headcount - grade_rank) / grade_headcount * 100
--   student_score         = the student's overall_mean_percentage
--   class_average         = mean of all scored students' overall_mean_percentage in the class
--   grade_average         = mean of all scored students' overall_mean_percentage in the grade
--   variance_from_grade_mean = student_score - grade_average

-- ============================================================================
-- TABLE: student_cohort_position_summaries
-- ============================================================================

CREATE TABLE IF NOT EXISTS student_cohort_position_summaries (
    id                      UUID          PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id               UUID          NOT NULL,
    school_id               UUID          NOT NULL,
    student_id              UUID          NOT NULL,
    class_id                UUID          NOT NULL,
    academic_term_id        UUID          NOT NULL,
    student_score           NUMERIC(5,2),           -- overall_mean_percentage
    class_rank              INT,                    -- 1 = highest score in class
    class_headcount         INT,                    -- scored students in class
    class_average           NUMERIC(5,2),           -- mean of class scores
    class_percentile        NUMERIC(5,2),           -- (headcount - rank) / headcount * 100
    grade_rank              INT,                    -- 1 = highest score in grade
    grade_headcount         INT,                    -- scored students in grade
    grade_average           NUMERIC(5,2),           -- mean of grade scores
    grade_percentile        NUMERIC(5,2),           -- (headcount - rank) / headcount * 100
    variance_from_grade_mean NUMERIC(6,2),          -- student_score - grade_average
    last_refreshed_at       TIMESTAMPTZ   NOT NULL DEFAULT NOW(),
    created_at              TIMESTAMPTZ   NOT NULL DEFAULT NOW(),
    updated_at              TIMESTAMPTZ   NOT NULL DEFAULT NOW(),

    CONSTRAINT uq_cohort_position_student_class_term
        UNIQUE (student_id, class_id, academic_term_id),
    CONSTRAINT fk_cohort_position_tenant_student
        FOREIGN KEY (tenant_id, student_id)
        REFERENCES cbc_students(tenant_id, id) ON DELETE CASCADE,
    CONSTRAINT fk_cohort_position_tenant_class
        FOREIGN KEY (tenant_id, class_id)
        REFERENCES cbc_classes(tenant_id, id) ON DELETE CASCADE,
    CONSTRAINT fk_cohort_position_tenant_term
        FOREIGN KEY (tenant_id, school_id, academic_term_id)
        REFERENCES academic_terms(tenant_id, school_id, id) ON DELETE CASCADE,
    CONSTRAINT fk_cohort_position_tenant_school
        FOREIGN KEY (tenant_id, school_id)
        REFERENCES cbc_schools(tenant_id, id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_cohort_position_tenant
    ON student_cohort_position_summaries (tenant_id);
CREATE INDEX IF NOT EXISTS idx_cohort_position_school
    ON student_cohort_position_summaries (school_id);
CREATE INDEX IF NOT EXISTS idx_cohort_position_student_term
    ON student_cohort_position_summaries (student_id, academic_term_id);
CREATE INDEX IF NOT EXISTS idx_cohort_position_term
    ON student_cohort_position_summaries (academic_term_id);
CREATE INDEX IF NOT EXISTS idx_cohort_position_class_term
    ON student_cohort_position_summaries (class_id, academic_term_id);

DROP TRIGGER IF EXISTS trg_student_cohort_position_summaries_updated_at
    ON student_cohort_position_summaries;
CREATE TRIGGER trg_student_cohort_position_summaries_updated_at
    BEFORE UPDATE ON student_cohort_position_summaries
    FOR EACH ROW EXECUTE FUNCTION fn_set_updated_at();

COMMENT ON TABLE student_cohort_position_summaries IS
    'Periodic batch-computed class and grade rankings per student per term.
     NEVER updated incrementally — the batch function must be called on a
     schedule or on-demand via the refresh API.';

COMMENT ON COLUMN student_cohort_position_summaries.student_score IS
    'The student''s overall_mean_percentage from
     student_term_overall_summaries at the time of computation.';

COMMENT ON COLUMN student_cohort_position_summaries.class_rank IS
    'Rank of the student within their class, ordered by student_score
     descending. 1 = highest score. NULL when student has no score.';

COMMENT ON COLUMN student_cohort_position_summaries.class_headcount IS
    'Number of students in the same class who have a non-null
     overall_mean_percentage.';

COMMENT ON COLUMN student_cohort_position_summaries.class_percentile IS
    'Percentile within the class, computed as
     (class_headcount - class_rank) / class_headcount * 100.
     A student ranked 4th out of 32 has percentile = (32-4)/32*100 = 87.50.';

COMMENT ON COLUMN student_cohort_position_summaries.grade_rank IS
    'Rank of the student within the same grade_level across the entire school,
     ordered by student_score descending. 1 = highest score in the grade.';

COMMENT ON COLUMN student_cohort_position_summaries.grade_headcount IS
    'Number of students in the same grade_level across the school who have a
     non-null overall_mean_percentage.';

COMMENT ON COLUMN student_cohort_position_summaries.grade_percentile IS
    'Percentile within the grade, computed as
     (grade_headcount - grade_rank) / grade_headcount * 100.';

COMMENT ON COLUMN student_cohort_position_summaries.class_average IS
    'Mean of student_score across all scored students in the same class.';

COMMENT ON COLUMN student_cohort_position_summaries.grade_average IS
    'Mean of student_score across all scored students in the same grade_level
     across the school.';

COMMENT ON COLUMN student_cohort_position_summaries.variance_from_grade_mean IS
    'Difference between the student''s score and the grade average.
     Positive = above average, Negative = below average.';

-- ============================================================================
-- FUNCTION: fn_compute_cohort_positions_for_term(target_term_id UUID)
--
-- Computes (or recomputes) student_cohort_position_summaries for ALL students
-- enrolled in the given academic term.
--
-- Algorithm:
--   1. Fetch all ACTIVE enrollments for the term, joining to cbc_classes for
--      grade_level and to student_term_overall_summaries for scores.
--   2. Use window functions (RANK() OVER class, RANK() OVER grade) to compute
--      class_rank and grade_rank.
--   3. Compute class_average and grade_average using AVG() OVER.
--   4. Derive percentiles from ranks and headcounts.
--   5. Compute variance_from_grade_mean.
--   6. Upsert all rows into student_cohort_position_summaries.
--   7. Clean up orphaned rows (students no longer enrolled).
-- ============================================================================

CREATE OR REPLACE FUNCTION fn_compute_cohort_positions_for_term(target_term_id UUID)
RETURNS void
LANGUAGE plpgsql
AS $$
DECLARE
    v_tenant_id UUID;
    v_school_id UUID;
BEGIN
    -- Resolve term metadata
    SELECT tenant_id, school_id
    INTO v_tenant_id, v_school_id
    FROM academic_terms
    WHERE id = target_term_id;

    IF NOT FOUND THEN
        RETURN;
    END IF;

    -- =====================================================================
    -- Main UPSERT: compute cohort positions for every enrolled student
    -- =====================================================================
    INSERT INTO student_cohort_position_summaries (
        tenant_id, school_id, student_id, class_id, academic_term_id,
        student_score,
        class_rank, class_headcount, class_average, class_percentile,
        grade_rank, grade_headcount, grade_average, grade_percentile,
        variance_from_grade_mean,
        last_refreshed_at
    )
    WITH scored_enrollments AS (
        -- All ACTIVE enrollments with their overall score, joining through
        -- cbc_classes to get grade_level for the grade-level ranking.
        SELECT
            e.tenant_id,
            e.school_id,
            e.student_id,
            e.class_id,
            c.grade_level::TEXT AS grade_level,
            s.overall_mean_percentage AS student_score
        FROM cbc_student_enrollments e
        JOIN cbc_classes c ON c.id = e.class_id AND c.tenant_id = e.tenant_id
        LEFT JOIN student_term_overall_summaries s
            ON s.student_id = e.student_id
            AND s.academic_term_id = e.academic_term_id
        WHERE e.academic_term_id = target_term_id
          AND e.status = 'ACTIVE'
    ),
    class_stats AS (
        SELECT
            class_id,
            ROUND(AVG(student_score)::NUMERIC, 2) AS class_average,
            COUNT(*) FILTER (WHERE student_score IS NOT NULL) AS class_scored_count
        FROM scored_enrollments
        GROUP BY class_id
    ),
    grade_stats AS (
        SELECT
            grade_level,
            school_id,
            ROUND(AVG(student_score)::NUMERIC, 2) AS grade_average,
            COUNT(*) FILTER (WHERE student_score IS NOT NULL) AS grade_scored_count
        FROM scored_enrollments
        GROUP BY grade_level, school_id
    ),
    ranked AS (
        SELECT
            se.tenant_id,
            se.school_id,
            se.student_id,
            se.class_id,
            se.student_score,
            se.grade_level,
            -- Class-level rank: scored students ranked within their class
            CASE
                WHEN se.student_score IS NOT NULL
                THEN RANK() OVER (
                    PARTITION BY se.class_id
                    ORDER BY se.student_score DESC NULLS LAST
                )::INT
                ELSE NULL
            END AS class_rank,
            -- Grade-level rank: scored students ranked within their grade
            CASE
                WHEN se.student_score IS NOT NULL
                THEN RANK() OVER (
                    PARTITION BY se.grade_level, se.school_id
                    ORDER BY se.student_score DESC NULLS LAST
                )::INT
                ELSE NULL
            END AS grade_rank,
            cs.class_average,
            cs.class_scored_count,
            gs.grade_average,
            gs.grade_scored_count
        FROM scored_enrollments se
        LEFT JOIN class_stats cs ON cs.class_id = se.class_id
        LEFT JOIN grade_stats gs
            ON gs.grade_level = se.grade_level
            AND gs.school_id = se.school_id
    )
    SELECT
        tenant_id,
        school_id,
        student_id,
        class_id,
        target_term_id,
        student_score,
        class_rank,
        class_scored_count,
        class_average,
        -- class_percentile: (headcount - rank) / headcount * 100
        CASE
            WHEN class_rank IS NOT NULL AND class_scored_count > 0
            THEN ROUND(
                (class_scored_count - class_rank)::NUMERIC / class_scored_count * 100,
                2
            )
            ELSE NULL
        END,
        grade_rank,
        grade_scored_count,
        grade_average,
        -- grade_percentile: (headcount - rank) / headcount * 100
        CASE
            WHEN grade_rank IS NOT NULL AND grade_scored_count > 0
            THEN ROUND(
                (grade_scored_count - grade_rank)::NUMERIC / grade_scored_count * 100,
                2
            )
            ELSE NULL
        END,
        -- variance_from_grade_mean: student_score - grade_average
        CASE
            WHEN student_score IS NOT NULL AND grade_average IS NOT NULL
            THEN ROUND(student_score - grade_average, 2)
            ELSE NULL
        END,
        NOW()
    FROM ranked
    -- Only insert rows where the student has a score (no score = no ranking)
    WHERE student_score IS NOT NULL

    ON CONFLICT (student_id, class_id, academic_term_id)
    DO UPDATE SET
        student_score           = EXCLUDED.student_score,
        class_rank              = EXCLUDED.class_rank,
        class_headcount         = EXCLUDED.class_headcount,
        class_average           = EXCLUDED.class_average,
        class_percentile        = EXCLUDED.class_percentile,
        grade_rank              = EXCLUDED.grade_rank,
        grade_headcount         = EXCLUDED.grade_headcount,
        grade_average           = EXCLUDED.grade_average,
        grade_percentile        = EXCLUDED.grade_percentile,
        variance_from_grade_mean = EXCLUDED.variance_from_grade_mean,
        last_refreshed_at       = NOW(),
        updated_at              = NOW();

    -- Clean up orphaned rows (students no longer enrolled with ACTIVE status)
    DELETE FROM student_cohort_position_summaries
    WHERE academic_term_id = target_term_id
      AND student_id NOT IN (
          SELECT student_id FROM cbc_student_enrollments
          WHERE academic_term_id = target_term_id
            AND status = 'ACTIVE'
      );

END;
$$;

COMMENT ON FUNCTION fn_compute_cohort_positions_for_term IS
    'Batch-computes student_cohort_position_summaries for all students enrolled
     in the given academic term. Uses window functions to compute class and
     grade ranks, percentiles, averages, and variance. Must be called on a
     schedule — never incrementally.';

-- ============================================================================
-- RLS POLICY
-- ============================================================================

ALTER TABLE IF EXISTS student_cohort_position_summaries ENABLE ROW LEVEL SECURITY;

DO $$ BEGIN
    DROP POLICY IF EXISTS tenant_isolation_policy ON student_cohort_position_summaries;
    CREATE POLICY tenant_isolation_policy ON student_cohort_position_summaries
        FOR ALL
        USING (tenant_id = fn_current_tenant_id())
        WITH CHECK (tenant_id = fn_current_tenant_id());
END $$;

-- Migration: 000009_create_student_subject_strand_summaries
-- Creates the student_subject_strand_summaries materialised table and the
-- PostgreSQL function + trigger that keep it in sync when rubric assessment
-- sessions are published.
--
-- Grain: (student_id, academic_term_id, sub_strand_id)
--
-- This table is a rubric-only summary at the sub-strand level. It counts how
-- many performance indicators within the sub-strand were awarded each CBC
-- performance level (EE, ME, AE, BE) and computes a mastery_percentage as
-- the percentage of indicators at Meeting Expectations or above.
--
-- For subjects taught purely quantitatively (no rubric sessions), has_data
-- stays false rather than showing a misleading 0%. Consumers should always
-- check has_data before displaying mastery metrics.
--
-- requires_remediation is set to true when any indicator is Below
-- Expectations or when mastery_percentage drops below 50%.

-- ============================================================================
-- TABLE: student_subject_strand_summaries
-- ============================================================================

CREATE TABLE IF NOT EXISTS student_subject_strand_summaries (
    id                      UUID          PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id               UUID          NOT NULL,
    school_id               UUID          NOT NULL,
    student_id              UUID          NOT NULL,
    academic_term_id        UUID          NOT NULL,
    learning_area_id        UUID          NOT NULL,
    strand_id               UUID          NOT NULL,
    sub_strand_id           UUID          NOT NULL,
    mastery_percentage      NUMERIC(5,2),
    exceeding_count         INT           NOT NULL DEFAULT 0,
    meeting_count           INT           NOT NULL DEFAULT 0,
    approaching_count       INT           NOT NULL DEFAULT 0,
    below_count             INT           NOT NULL DEFAULT 0,
    mapped_performance_level VARCHAR(5),
    requires_remediation    BOOLEAN       NOT NULL DEFAULT false,
    has_data                BOOLEAN       NOT NULL DEFAULT false,
    last_refreshed_at       TIMESTAMPTZ   NOT NULL DEFAULT NOW(),
    created_at              TIMESTAMPTZ   NOT NULL DEFAULT NOW(),
    updated_at              TIMESTAMPTZ   NOT NULL DEFAULT NOW(),

    CONSTRAINT uq_student_term_sub_strand UNIQUE (student_id, academic_term_id, sub_strand_id),
    CONSTRAINT fk_strand_summaries_tenant_student
        FOREIGN KEY (tenant_id, student_id)
        REFERENCES cbc_students(tenant_id, id) ON DELETE CASCADE,
    CONSTRAINT fk_strand_summaries_tenant_term
        FOREIGN KEY (tenant_id, school_id, academic_term_id)
        REFERENCES academic_terms(tenant_id, school_id, id) ON DELETE CASCADE,
    CONSTRAINT fk_strand_summaries_tenant_school
        FOREIGN KEY (tenant_id, school_id)
        REFERENCES cbc_schools(tenant_id, id) ON DELETE CASCADE,
    CONSTRAINT fk_strand_summaries_learning_area
        FOREIGN KEY (learning_area_id)
        REFERENCES cbc_learning_areas(id) ON DELETE CASCADE,
    CONSTRAINT fk_strand_summaries_strand
        FOREIGN KEY (strand_id)
        REFERENCES cbc_strands(id) ON DELETE CASCADE,
    CONSTRAINT fk_strand_summaries_sub_strand
        FOREIGN KEY (sub_strand_id)
        REFERENCES cbc_sub_strands(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_strand_summaries_tenant
    ON student_subject_strand_summaries (tenant_id);
CREATE INDEX IF NOT EXISTS idx_strand_summaries_school
    ON student_subject_strand_summaries (school_id);
CREATE INDEX IF NOT EXISTS idx_strand_summaries_student_term
    ON student_subject_strand_summaries (student_id, academic_term_id);
CREATE INDEX IF NOT EXISTS idx_strand_summaries_term_sub_strand
    ON student_subject_strand_summaries (academic_term_id, sub_strand_id);

DROP TRIGGER IF EXISTS trg_student_subject_strand_summaries_updated_at
    ON student_subject_strand_summaries;
CREATE TRIGGER trg_student_subject_strand_summaries_updated_at
    BEFORE UPDATE ON student_subject_strand_summaries
    FOR EACH ROW EXECUTE FUNCTION fn_set_updated_at();

COMMENT ON TABLE student_subject_strand_summaries IS
    'Rubric-only sub-strand-level summary per student and term. Counts
     performance indicators awarded at each CBC level (EE/ME/AE/BE) and
     computes mastery as the percentage at ME or above. Only populated for
     RUBRIC sessions — for quantitative subjects has_data stays false.';

COMMENT ON COLUMN student_subject_strand_summaries.mastery_percentage IS
    'Percentage of performance indicators for this sub-strand that were
     awarded Meeting Expectations or above:
     (exceeding_count + meeting_count) / (total_indicators) * 100.
     NULL when no data exists.';

COMMENT ON COLUMN student_subject_strand_summaries.mapped_performance_level IS
    'The CBC performance level (EE/ME/AE/BE) corresponding to
     mastery_percentage, determined by the school''s active grading
     scale profile. NULL when no profile can be resolved.';

COMMENT ON COLUMN student_subject_strand_summaries.requires_remediation IS
    'TRUE when any indicator was awarded Below Expectations or when
     mastery_percentage is below 50%. Suggests the student needs
     targeted intervention on this sub-strand.';

COMMENT ON COLUMN student_subject_strand_summaries.has_data IS
    'TRUE when at least one rubric outcome grade exists for this
     student+term+sub_strand. For subjects assessed purely quantitatively,
     this stays false — consumers should check this flag before displaying
     mastery metrics to avoid misleading 0% displays.';

-- ============================================================================
-- FUNCTION: fn_refresh_subject_strand_summary_for_session(target_session_id UUID)
--
-- Recomputes student_subject_strand_summaries for all students in the given
-- session, scoped to the session's academic_term_id and sub-strands that
-- were assessed.
--
-- This function ONLY processes sessions with evaluation_method = 'RUBRIC'.
-- For QUANTITATIVE sessions, it is a no-op.
--
-- The algorithm:
--   1. Resolves the session's metadata (tenant_id, school_id,
--      academic_term_id, learning_area_id).
--   2. If the session is not RUBRIC, returns immediately (no-op).
--   3. For each student in the session, groups outcome grades by
--      sub_strand_id (via performance_indicator_id → cbc_sub_strands).
--   4. Counts indicators at each level (EE, ME, AE, BE).
--   5. Computes mastery_percentage = (EE_count + ME_count) / total * 100.
--   6. Maps mastery_percentage to a CBC level using the school's active
--      grading scale profile.
--   7. Sets requires_remediation = (below_count > 0 OR mastery < 50%).
--   8. Sets has_data = true when any outcome grade exists.
--   9. Upserts into student_subject_strand_summaries.
-- ============================================================================

CREATE OR REPLACE FUNCTION fn_refresh_subject_strand_summary_for_session(target_session_id UUID)
RETURNS void
LANGUAGE plpgsql
AS $$
DECLARE
    v_tenant_id          UUID;
    v_school_id          UUID;
    v_academic_term_id   UUID;
    v_learning_area_id   UUID;
    v_evaluation_method  TEXT;
    v_scale_profile_id   UUID;
BEGIN
    -- Resolve session metadata
    SELECT tenant_id, school_id, academic_term_id, learning_area_id,
           evaluation_method::TEXT
    INTO v_tenant_id, v_school_id, v_academic_term_id, v_learning_area_id,
         v_evaluation_method
    FROM assessment_sessions
    WHERE id = target_session_id;

    IF NOT FOUND THEN
        RETURN;
    END IF;

    -- This table is rubric-only — skip quantitative sessions
    IF v_evaluation_method != 'RUBRIC' THEN
        RETURN;
    END IF;

    -- Find the school's active grading scale profile for level mapping
    SELECT id INTO v_scale_profile_id
    FROM grading_scale_profiles
    WHERE school_id = v_school_id AND is_active = true
    ORDER BY created_at DESC
    LIMIT 1;

    -- Upsert summary for each student+sub_strand in this session
    INSERT INTO student_subject_strand_summaries (
        tenant_id, school_id, student_id, academic_term_id,
        learning_area_id, strand_id, sub_strand_id,
        mastery_percentage,
        exceeding_count, meeting_count, approaching_count, below_count,
        mapped_performance_level,
        requires_remediation, has_data,
        last_refreshed_at
    )
    WITH affected_students AS (
        SELECT DISTINCT student_id
        FROM student_assessment_outcome_grades
        WHERE session_id = target_session_id
    ),
    indicator_hierarchy AS (
        -- Resolve the strand and sub-strand for each performance indicator
        SELECT
            pi.id AS indicator_id,
            ss.id AS sub_strand_id,
            ss.name AS sub_strand_name,
            s.id AS strand_id,
            s.name AS strand_name
        FROM performance_indicators pi
        JOIN cbc_sub_strands ss ON ss.id = pi.sub_strand_id
        JOIN cbc_strands s ON s.id = ss.strand_id
    ),
    all_outcome_grades_for_term AS (
        -- All outcome grades for these students in this term and learning area,
        -- from ALL PUBLISHED rubric sessions (not just the target session)
        SELECT
            sog.student_id,
            sog.performance_indicator_id,
            sog.awarded_level::TEXT AS awarded_level,
            ih.sub_strand_id,
            ih.strand_id
        FROM student_assessment_outcome_grades sog
        JOIN assessment_sessions s ON s.id = sog.session_id
        JOIN indicator_hierarchy ih ON ih.indicator_id = sog.performance_indicator_id
        WHERE s.academic_term_id = v_academic_term_id
          AND s.learning_area_id = v_learning_area_id
          AND s.status = 'PUBLISHED'
          AND s.evaluation_method = 'RUBRIC'::assessment_evaluation_method
          AND sog.student_id IN (SELECT student_id FROM affected_students)
    ),
    level_counts AS (
        SELECT
            student_id,
            sub_strand_id,
            strand_id,
            COUNT(*) FILTER (WHERE awarded_level = 'EE') AS ee_count,
            COUNT(*) FILTER (WHERE awarded_level = 'ME') AS me_count,
            COUNT(*) FILTER (WHERE awarded_level = 'AE') AS ae_count,
            COUNT(*) FILTER (WHERE awarded_level = 'BE') AS be_count,
            COUNT(*) AS total_count
        FROM all_outcome_grades_for_term
        GROUP BY student_id, sub_strand_id, strand_id
    )
    SELECT
        v_tenant_id,
        v_school_id,
        lc.student_id,
        v_academic_term_id,
        v_learning_area_id,
        lc.strand_id,
        lc.sub_strand_id,
        -- mastery_percentage: (EE + ME) / total * 100
        CASE
            WHEN lc.total_count > 0
            THEN ROUND((lc.ee_count + lc.me_count)::NUMERIC / lc.total_count * 100, 2)
            ELSE NULL
        END AS mastery_percentage,
        lc.ee_count,
        lc.me_count,
        lc.ae_count,
        lc.be_count,
        -- mapped_performance_level: map mastery_percentage via scale profile
        CASE
            WHEN lc.total_count > 0
             AND v_scale_profile_id IS NOT NULL
            THEN (
                SELECT r.performance_level::TEXT
                FROM grading_scale_ranges r
                WHERE r.profile_id = v_scale_profile_id
                  AND ROUND((lc.ee_count + lc.me_count)::NUMERIC / lc.total_count * 100, 2) >= r.min_percentage
                  AND ROUND((lc.ee_count + lc.me_count)::NUMERIC / lc.total_count * 100, 2) <= r.max_percentage
                LIMIT 1
            )
            ELSE NULL
        END AS mapped_performance_level,
        -- requires_remediation: BE > 0 OR mastery < 50%
        CASE
            WHEN lc.total_count > 0
            THEN (lc.be_count > 0)
               OR (ROUND((lc.ee_count + lc.me_count)::NUMERIC / lc.total_count * 100, 2) < 50)
            ELSE false
        END AS requires_remediation,
        -- has_data: true if any outcome grades exist
        lc.total_count > 0 AS has_data,
        NOW()
    FROM level_counts lc
    WHERE lc.total_count > 0

    ON CONFLICT (student_id, academic_term_id, sub_strand_id)
    DO UPDATE SET
        mastery_percentage        = EXCLUDED.mastery_percentage,
        exceeding_count           = EXCLUDED.exceeding_count,
        meeting_count             = EXCLUDED.meeting_count,
        approaching_count         = EXCLUDED.approaching_count,
        below_count               = EXCLUDED.below_count,
        mapped_performance_level  = EXCLUDED.mapped_performance_level,
        requires_remediation      = EXCLUDED.requires_remediation,
        has_data                  = EXCLUDED.has_data,
        last_refreshed_at         = NOW(),
        updated_at                = NOW();

    -- Clean up orphaned rows where the student no longer has any outcome
    -- grades in this term for this sub-strand (e.g. session was unpublished)
    DELETE FROM student_subject_strand_summaries
    WHERE academic_term_id = v_academic_term_id
      AND learning_area_id = v_learning_area_id
      AND student_id IN (
          SELECT DISTINCT student_id
          FROM student_assessment_outcome_grades
          WHERE session_id = target_session_id
      )
      AND has_data = false;
END;
$$;

COMMENT ON FUNCTION fn_refresh_subject_strand_summary_for_session IS
    'Refreshes student_subject_strand_summaries for all students in the given
     rubric session. Groups outcome grades by sub-strand, counts level
     distributions, computes mastery percentage, and determines remediation
     need. No-op for QUANTITATIVE sessions.';

-- ============================================================================
-- RLS POLICY
-- ============================================================================

ALTER TABLE IF EXISTS student_subject_strand_summaries ENABLE ROW LEVEL SECURITY;

DO $$ BEGIN
    DROP POLICY IF EXISTS tenant_isolation_policy ON student_subject_strand_summaries;
    CREATE POLICY tenant_isolation_policy ON student_subject_strand_summaries
        FOR ALL
        USING (tenant_id = fn_current_tenant_id())
        WITH CHECK (tenant_id = fn_current_tenant_id());
END $$;

COMMENT ON TABLE student_subject_strand_summaries IS
    'Rubric-only sub-strand-level summary per student and term. RLS-enabled
     — tenant-scoped.';

-- Migration: 000010_create_student_performance_projections
-- Creates the student_performance_projections table and the PostgreSQL
-- batch function that computes projection metrics for all students enrolled
-- in a given academic term.
--
-- Grain: (student_id, academic_term_id, learning_area_id)
--
-- learning_area_id may be NULL for an overall (cross-subject) projection.
--
-- This is a PERIODIC BATCH-ONLY table. It is NEVER updated incrementally.
-- A single new score should not reshuffle a trend line. Computation is
-- triggered once per term close via the batch function
-- fn_compute_performance_projections_for_term().
--
-- Data source:
--   Reads the last 2–3 terms of student_term_subject_summaries for the same
--   student and learning area (or student_term_overall_summaries for overall).
--
-- Computation:
--   momentum_score       = linear regression slope over available terms
--   projected_score      = last_term_score + momentum_score (next term's estimate)
--   projected_performance_level = CBC level for projected_score via active scale
--   target_gap_points    = projected_score - ME_threshold_percentage
--   risk_level           = 'Low'|'Medium'|'High' based on gap + confidence
--   confidence_percentage = based on number of data points (capped low when < 2)

-- ============================================================================
-- TABLE: student_performance_projections
-- ============================================================================

CREATE TABLE IF NOT EXISTS student_performance_projections (
    id                         UUID          PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id                  UUID          NOT NULL,
    school_id                  UUID          NOT NULL,
    student_id                 UUID          NOT NULL,
    academic_term_id           UUID          NOT NULL,
    learning_area_id           UUID,                   -- NULL = overall projection
    momentum_score             NUMERIC(6,2),            -- slope per term
    projected_score            NUMERIC(5,2),            -- predicted next-term score
    projected_performance_level VARCHAR(5),
    target_gap_points          NUMERIC(6,2),            -- diff from ME threshold
    risk_level                 VARCHAR(10)   NOT NULL DEFAULT 'Unknown',
    confidence_percentage      NUMERIC(5,2),
    last_refreshed_at          TIMESTAMPTZ   NOT NULL DEFAULT NOW(),
    created_at                 TIMESTAMPTZ   NOT NULL DEFAULT NOW(),
    updated_at                 TIMESTAMPTZ   NOT NULL DEFAULT NOW(),

    CONSTRAINT uq_student_term_learning_area_proj
        UNIQUE (student_id, academic_term_id, learning_area_id),
    CONSTRAINT fk_projections_tenant_student
        FOREIGN KEY (tenant_id, student_id)
        REFERENCES cbc_students(tenant_id, id) ON DELETE CASCADE,
    CONSTRAINT fk_projections_tenant_term
        FOREIGN KEY (tenant_id, school_id, academic_term_id)
        REFERENCES academic_terms(tenant_id, school_id, id) ON DELETE CASCADE,
    CONSTRAINT fk_projections_tenant_school
        FOREIGN KEY (tenant_id, school_id)
        REFERENCES cbc_schools(tenant_id, id) ON DELETE CASCADE,
    CONSTRAINT fk_projections_learning_area
        FOREIGN KEY (learning_area_id)
        REFERENCES cbc_learning_areas(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_projections_tenant
    ON student_performance_projections (tenant_id);
CREATE INDEX IF NOT EXISTS idx_projections_school
    ON student_performance_projections (school_id);
CREATE INDEX IF NOT EXISTS idx_projections_student_term
    ON student_performance_projections (student_id, academic_term_id);
CREATE INDEX IF NOT EXISTS idx_projections_term
    ON student_performance_projections (academic_term_id);
CREATE INDEX IF NOT EXISTS idx_projections_learning_area
    ON student_performance_projections (learning_area_id);

DROP TRIGGER IF EXISTS trg_student_performance_projections_updated_at
    ON student_performance_projections;
CREATE TRIGGER trg_student_performance_projections_updated_at
    BEFORE UPDATE ON student_performance_projections
    FOR EACH ROW EXECUTE FUNCTION fn_set_updated_at();

COMMENT ON TABLE student_performance_projections IS
    'Periodic batch-computed performance projections per student per term per
     learning area. Reads the last 2-3 terms of assessment data to compute a
     momentum trend line and project the next term''s score. NEVER updated
     incrementally — call fn_compute_performance_projections_for_term()
     periodically (once per term close).';

COMMENT ON COLUMN student_performance_projections.momentum_score IS
    'Linear regression slope over the last 2-3 terms of assessment data.
     Positive = improving trend, Negative = declining trend.
     NULL when fewer than 2 terms of history exist.';

COMMENT ON COLUMN student_performance_projections.projected_score IS
    'Predicted score for the next term, calculated as the last term''s score
     plus the momentum_score. NULL when insufficient history exists.';

COMMENT ON COLUMN student_performance_projections.projected_performance_level IS
    'The CBC performance level (EE/ME/AE/BE) corresponding to
     projected_score, determined by the school''s active grading scale
     profile. NULL when not resolvable.';

COMMENT ON COLUMN student_performance_projections.target_gap_points IS
    'Difference between projected_score and the minimum percentage required
     for Meeting Expectations (from the active grading scale profile).
     Negative = student is projected below the ME threshold.';

COMMENT ON COLUMN student_performance_projections.risk_level IS
    'Risk classification: Low (confident projection, close to or above ME
     threshold), Medium (moderate gap or uncertainty), High (significant
     gap or very low confidence). Defaults to Unknown initially.';

COMMENT ON COLUMN student_performance_projections.confidence_percentage IS
    'Confidence in the projection based on the number of historical terms
     available. Capped low (< 30%) when fewer than 2 terms exist, to signal
     that the projection is less trustworthy for new enrollees.';

-- ============================================================================
-- FUNCTION: fn_compute_performance_projections_for_term(target_term_id UUID)
--
-- Computes (or recomputes) student_performance_projections for ALL students
-- enrolled in the given academic term.
--
-- Algorithm:
--   1. Find the target term's tenant_id, school_id, and its term_number
--      within the academic year to identify the "current" term.
--   2. For each student enrolled in the target term, collect the last 2-3
--      terms of subject summary data (including the current term).
--   3. For each student+learning_area:
--      a. If 2+ terms of history exist, compute linear regression slope
--         (momentum_score) = COVAR(x,y) / VAR(x) where x = term_index and
--         y = average_percentage.
--      b. projected_score = last_term_score + momentum_score.
--      c. Map projected_score via active grading scale profile.
--      d. target_gap_points = projected_score - ME_threshold.
--      e. risk_level based on gap and confidence.
--      f. confidence_percentage based on data points.
--   4. If fewer than 2 terms of history exist, write a row with
--      confidence_percentage capped low (15%) and NULL scores, so new
--      enrollees show up but visibly less trustworthy.
--   5. Clean up orphaned rows for students no longer enrolled.
-- ============================================================================

CREATE OR REPLACE FUNCTION fn_compute_performance_projections_for_term(target_term_id UUID)
RETURNS void
LANGUAGE plpgsql
AS $$
DECLARE
    v_tenant_id        UUID;
    v_school_id        UUID;
    v_me_threshold     NUMERIC(5,2);
    v_scale_profile_id UUID;
    v_current_term_num INT;
    v_current_year_id  UUID;
BEGIN
    -- Resolve term metadata
    SELECT tenant_id, school_id, term_number, academic_year_id
    INTO v_tenant_id, v_school_id, v_current_term_num, v_current_year_id
    FROM academic_terms
    WHERE id = target_term_id;

    IF NOT FOUND THEN
        RETURN;
    END IF;

    -- Find the school's active grading scale profile
    SELECT id INTO v_scale_profile_id
    FROM grading_scale_profiles
    WHERE school_id = v_school_id AND is_active = true
    ORDER BY created_at DESC
    LIMIT 1;

    -- Find the ME threshold from the active scale profile
    SELECT r.min_percentage INTO v_me_threshold
    FROM grading_scale_ranges r
    WHERE r.profile_id = v_scale_profile_id
      AND r.performance_level = 'ME'::cbc_performance_level
    LIMIT 1;

    -- If no ME threshold can be resolved, default to 50%
    IF v_me_threshold IS NULL THEN
        v_me_threshold := 50.00;
    END IF;

    -- =====================================================================
    -- Step 1: Build a list of eligible terms (current + up to 2 prior)
    -- =====================================================================
    CREATE TEMP TABLE IF NOT EXISTS tmp_eligible_terms (
        term_id         UUID,
        term_number     INT,
        academic_year_id UUID,
        sequential_idx  INT NOT NULL  -- 0 = earliest, N = current
    ) ON COMMIT DROP;

    INSERT INTO tmp_eligible_terms
    WITH ordered AS (
        SELECT
            id,
            term_number,
            academic_year_id,
            ROW_NUMBER() OVER (ORDER BY academic_year_id ASC, term_number ASC) - 1 AS idx
        FROM academic_terms
        WHERE school_id = v_school_id
          AND tenant_id = v_tenant_id
          AND (
               academic_year_id = v_current_year_id
               OR
               -- Include the previous academic year's terms
               academic_year_id = (
                   SELECT id FROM academic_years
                   WHERE tenant_id = v_tenant_id
                     AND school_id = v_school_id
                     AND name::INT = (
                         SELECT (name::INT - 1) FROM academic_years WHERE id = v_current_year_id
                     )
               )
          )
        ORDER BY academic_year_id ASC, term_number ASC
    )
    SELECT id, term_number, academic_year_id, idx
    FROM ordered
    WHERE idx >= GREATEST(
        (SELECT idx FROM ordered WHERE id = target_term_id) - 2,
        0
    );

    -- =====================================================================
    -- Step 2: Per-student + per-learning-area projections
    -- =====================================================================
    INSERT INTO student_performance_projections (
        tenant_id, school_id, student_id, academic_term_id, learning_area_id,
        momentum_score, projected_score, projected_performance_level,
        target_gap_points, risk_level, confidence_percentage,
        last_refreshed_at
    )
    WITH enrolled_students AS (
        SELECT DISTINCT student_id
        FROM cbc_student_enrollments
        WHERE academic_term_id = target_term_id
          AND (status = 'ACTIVE' OR status = 'COMPLETED_CYCLE')
    ),
    -- Overall projections (learning_area_id = NULL): use overall summaries
    overall_history AS (
        SELECT
            s.student_id,
            NULL::UUID AS learning_area_id,
            et.sequential_idx,
            s.overall_mean_percentage::NUMERIC AS score,
            COUNT(*) OVER (PARTITION BY s.student_id) AS total_terms
        FROM student_term_overall_summaries s
        JOIN tmp_eligible_terms et ON et.term_id = s.academic_term_id
        WHERE s.student_id IN (SELECT student_id FROM enrolled_students)
          AND s.overall_mean_percentage IS NOT NULL
    ),
    -- Per-learning-area projections: use subject summaries
    subject_history AS (
        SELECT
            s.student_id,
            s.learning_area_id,
            et.sequential_idx,
            s.average_percentage::NUMERIC AS score,
            COUNT(*) OVER (PARTITION BY s.student_id, s.learning_area_id) AS total_terms
        FROM student_term_subject_summaries s
        JOIN tmp_eligible_terms et ON et.term_id = s.academic_term_id
        WHERE s.student_id IN (SELECT student_id FROM enrolled_students)
          AND s.average_percentage IS NOT NULL
    ),
    -- Regression base: compute sums for linear regression (y = mx + c)
    -- momentum (m) = (n*sum_xy - sum_x*sum_y) / (n*sum_xx - sum_x*sum_x)
    overall_regression AS (
        SELECT
            student_id,
            learning_area_id,
            COUNT(*) AS n,
            SUM(sequential_idx) AS sum_x,
            SUM(score) AS sum_y,
            SUM(sequential_idx * score) AS sum_xy,
            SUM(sequential_idx * sequential_idx) AS sum_xx
        FROM overall_history
        GROUP BY student_id, learning_area_id
    ),
    -- Last overall score per student (most recent term)
    overall_last_score AS (
        SELECT DISTINCT ON (student_id)
            student_id,
            learning_area_id,
            score AS last_score
        FROM overall_history
        ORDER BY student_id, sequential_idx DESC
    ),
    -- Compute momentum and projected from regression base
    overall_computed AS (
        SELECT
            r.student_id,
            r.learning_area_id,
            CASE
                WHEN r.n >= 2 AND (r.n * r.sum_xx - r.sum_x * r.sum_x) > 0
                THEN ROUND(
                    ((r.n * r.sum_xy - r.sum_x * r.sum_y)::NUMERIC
                     / (r.n * r.sum_xx - r.sum_x * r.sum_x)),
                    2
                )
                ELSE NULL
            END AS momentum_score,
            CASE
                WHEN r.n >= 2 AND (r.n * r.sum_xx - r.sum_x * r.sum_x) > 0
                THEN ROUND(
                    (l.last_score
                     + (r.n * r.sum_xy - r.sum_x * r.sum_y)::NUMERIC
                       / (r.n * r.sum_xx - r.sum_x * r.sum_x)),
                    2
                )
                ELSE NULL
            END AS projected_score,
            r.n AS history_term_count,
            CASE
                WHEN r.n >= 3 THEN 85.00
                WHEN r.n = 2 THEN 60.00
                ELSE 15.00
            END AS confidence_pct
        FROM overall_regression r
        LEFT JOIN overall_last_score l
            ON l.student_id = r.student_id
    ),
    -- Subject-level regression (same formula, per learning_area)
    subject_regression AS (
        SELECT
            student_id,
            learning_area_id,
            COUNT(*) AS n,
            SUM(sequential_idx) AS sum_x,
            SUM(score) AS sum_y,
            SUM(sequential_idx * score) AS sum_xy,
            SUM(sequential_idx * sequential_idx) AS sum_xx
        FROM subject_history
        GROUP BY student_id, learning_area_id
    ),
    subject_last_score AS (
        SELECT DISTINCT ON (student_id, learning_area_id)
            student_id,
            learning_area_id,
            score AS last_score
        FROM subject_history
        ORDER BY student_id, learning_area_id, sequential_idx DESC
    ),
    subject_computed AS (
        SELECT
            r.student_id,
            r.learning_area_id,
            CASE
                WHEN r.n >= 2 AND (r.n * r.sum_xx - r.sum_x * r.sum_x) > 0
                THEN ROUND(
                    ((r.n * r.sum_xy - r.sum_x * r.sum_y)::NUMERIC
                     / (r.n * r.sum_xx - r.sum_x * r.sum_x)),
                    2
                )
                ELSE NULL
            END AS momentum_score,
            CASE
                WHEN r.n >= 2 AND (r.n * r.sum_xx - r.sum_x * r.sum_x) > 0
                THEN ROUND(
                    (l.last_score
                     + (r.n * r.sum_xy - r.sum_x * r.sum_y)::NUMERIC
                       / (r.n * r.sum_xx - r.sum_x * r.sum_x)),
                    2
                )
                ELSE NULL
            END AS projected_score,
            r.n AS history_term_count,
            CASE
                WHEN r.n >= 3 THEN 85.00
                WHEN r.n = 2 THEN 60.00
                ELSE 15.00
            END AS confidence_pct
        FROM subject_regression r
        LEFT JOIN subject_last_score l
            ON l.student_id = r.student_id
            AND l.learning_area_id = r.learning_area_id
    ),
    -- Combine overall and subject projections
    all_projections AS (
        SELECT student_id, learning_area_id, momentum_score, projected_score,
               history_term_count, confidence_pct
        FROM overall_computed
        UNION ALL
        SELECT student_id, learning_area_id, momentum_score, projected_score,
               history_term_count, confidence_pct
        FROM subject_computed
    ),
    -- New enrollees: students with no history at all
    new_enrollees AS (
        SELECT
            es.student_id,
            la_ids.learning_area_id
        FROM enrolled_students es
        CROSS JOIN (
            SELECT NULL::UUID AS learning_area_id
            UNION
            SELECT DISTINCT s.learning_area_id
            FROM student_term_subject_summaries s
            WHERE s.student_id IN (SELECT student_id FROM enrolled_students)
              AND s.academic_term_id = target_term_id
        ) la_ids
        WHERE NOT EXISTS (
            SELECT 1 FROM all_projections ap
            WHERE ap.student_id = es.student_id
              AND (ap.learning_area_id IS NOT DISTINCT FROM la_ids.learning_area_id)
        )
    )
    SELECT
        v_tenant_id,
        v_school_id,
        ap.student_id,
        target_term_id,
        ap.learning_area_id,
        ap.momentum_score,
        -- Clamp projected_score to 0-100 range
        CASE
            WHEN ap.projected_score IS NOT NULL
            THEN GREATEST(0.00, LEAST(100.00, ap.projected_score))
            ELSE NULL
        END AS projected_score,
        -- Map projected score to performance level
        CASE
            WHEN ap.projected_score IS NOT NULL AND v_scale_profile_id IS NOT NULL
            THEN (
                SELECT r.performance_level::TEXT
                FROM grading_scale_ranges r
                WHERE r.profile_id = v_scale_profile_id
                  AND GREATEST(0.00, LEAST(100.00, ap.projected_score)) >= r.min_percentage
                  AND GREATEST(0.00, LEAST(100.00, ap.projected_score)) <= r.max_percentage
                LIMIT 1
            )
            ELSE NULL
        END AS projected_performance_level,
        -- target_gap_points: projected - ME threshold
        CASE
            WHEN ap.projected_score IS NOT NULL
            THEN ROUND((GREATEST(0.00, LEAST(100.00, ap.projected_score)) - v_me_threshold)::NUMERIC, 2)
            ELSE NULL
        END AS target_gap_points,
        -- risk_level
        CASE
            WHEN ap.projected_score IS NULL THEN 'Unknown'
            WHEN ap.confidence_pct < 30 THEN 'High'
            WHEN (GREATEST(0.00, LEAST(100.00, ap.projected_score)) - v_me_threshold) < -15 THEN 'High'
            WHEN (GREATEST(0.00, LEAST(100.00, ap.projected_score)) - v_me_threshold) < -5 THEN 'Medium'
            WHEN ap.confidence_pct < 60 THEN 'Medium'
            ELSE 'Low'
        END AS risk_level,
        ap.confidence_pct AS confidence_percentage,
        NOW()
    FROM all_projections ap

    UNION ALL

    -- New enrollees: write with low confidence and no scores
    SELECT
        v_tenant_id,
        v_school_id,
        ne.student_id,
        target_term_id,
        ne.learning_area_id,
        NULL AS momentum_score,
        NULL AS projected_score,
        NULL AS projected_performance_level,
        NULL AS target_gap_points,
        'High' AS risk_level,
        15.00 AS confidence_percentage,
        NOW()
    FROM new_enrollees ne
    WHERE ne.learning_area_id IS NOT NULL  -- Only per-subject for new enrollees
       OR ne.learning_area_id IS NULL       -- And overall

    ON CONFLICT (student_id, academic_term_id, learning_area_id)
    DO UPDATE SET
        momentum_score              = EXCLUDED.momentum_score,
        projected_score             = EXCLUDED.projected_score,
        projected_performance_level = EXCLUDED.projected_performance_level,
        target_gap_points           = EXCLUDED.target_gap_points,
        risk_level                  = EXCLUDED.risk_level,
        confidence_percentage       = EXCLUDED.confidence_percentage,
        last_refreshed_at           = NOW(),
        updated_at                  = NOW();

    -- Clean up orphaned rows (students no longer enrolled)
    DELETE FROM student_performance_projections
    WHERE academic_term_id = target_term_id
      AND student_id NOT IN (
          SELECT student_id FROM cbc_student_enrollments
          WHERE academic_term_id = target_term_id
            AND (status = 'ACTIVE' OR status = 'COMPLETED_CYCLE')
      );

    DROP TABLE IF EXISTS tmp_eligible_terms;
END;
$$;

COMMENT ON FUNCTION fn_compute_performance_projections_for_term IS
    'Batch-computes student_performance_projections for all students enrolled
     in the given academic term. Uses linear regression over the last 2-3
     terms to compute momentum and project next-term scores. Students with
     fewer than 2 terms of history get low-confidence placeholder rows.
     Must be called on a schedule — never incrementally.';

-- ============================================================================
-- RLS POLICY
-- ============================================================================

ALTER TABLE IF EXISTS student_performance_projections ENABLE ROW LEVEL SECURITY;

DO $$ BEGIN
    DROP POLICY IF EXISTS tenant_isolation_policy ON student_performance_projections;
    CREATE POLICY tenant_isolation_policy ON student_performance_projections
        FOR ALL
        USING (tenant_id = fn_current_tenant_id())
        WITH CHECK (tenant_id = fn_current_tenant_id());
END $$;

COMMENT ON TABLE student_performance_projections IS
    'Periodic batch-computed performance projections. RLS-enabled —
     tenant-scoped.';

-- Migration: 000011_create_student_behavior_term_summaries
-- Creates the student_behavior_term_summaries materialised table, the
-- category_type classification for behavior_categories, and the trigger
-- that keeps the summary in sync when behavior notes are inserted/updated.
--
-- Grain: (student_id, academic_term_id)
--
-- This table is an incremental materialised summary of APPROVED and
-- INCLUDED_IN_REPORT behavior notes per student per term. When a behavior
-- note transitions to APPROVED or INCLUDED_IN_REPORT, the summary is
-- refreshed for that student+term. PENDING_REVIEW and REJECTED notes are
-- excluded from the main counts but included in pending_review_count and
-- resolved_count for admin visibility.
--
-- primary_category_id is the behavior category with the highest count
-- among notes in this term (APPROVED + INCLUDED_IN_REPORT only). Ties
-- are resolved by the most recent note's category_id.

-- ============================================================================
-- ENRICHMENT: Add category_type to behavior_categories
-- Allows the system to distinguish commendations from disciplinary incidents.
-- ============================================================================

DO $$ BEGIN
    CREATE TYPE behavior_category_type AS ENUM ('COMMENDATION', 'DISCIPLINARY', 'OTHER');
EXCEPTION WHEN duplicate_object THEN NULL;
END $$;


COMMENT ON COLUMN behavior_categories.category_type IS
    'Classification of the behavior category: COMMENDATION (positive/laudable
     behaviour), DISCIPLINARY (negative behaviour / infraction), or OTHER.
     Used by student_behavior_term_summaries to compute commendations_count
     and disciplinary_count. Defaults to DISCIPLINARY for existing categories.';

-- ============================================================================
-- TABLE: student_behavior_term_summaries
-- ============================================================================

CREATE TABLE IF NOT EXISTS student_behavior_term_summaries (
    id                    UUID          PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id             UUID          NOT NULL,
    school_id             UUID          NOT NULL,
    student_id            UUID          NOT NULL,
    academic_term_id      UUID          NOT NULL,
    total_incidents       INT           NOT NULL DEFAULT 0,
    urgent_count          INT           NOT NULL DEFAULT 0,
    commendations_count   INT           NOT NULL DEFAULT 0,
    disciplinary_count    INT           NOT NULL DEFAULT 0,
    pending_review_count  INT           NOT NULL DEFAULT 0,
    resolved_count        INT           NOT NULL DEFAULT 0,
    primary_category_id   UUID,                   -- category with highest count (or NULL if none)
    last_refreshed_at     TIMESTAMPTZ   NOT NULL DEFAULT NOW(),
    created_at            TIMESTAMPTZ   NOT NULL DEFAULT NOW(),
    updated_at            TIMESTAMPTZ   NOT NULL DEFAULT NOW(),

    CONSTRAINT uq_student_behavior_term UNIQUE (student_id, academic_term_id),
    CONSTRAINT fk_behavior_summaries_tenant_student
        FOREIGN KEY (tenant_id, student_id)
        REFERENCES cbc_students(tenant_id, id) ON DELETE CASCADE,
    CONSTRAINT fk_behavior_summaries_tenant_term
        FOREIGN KEY (tenant_id, school_id, academic_term_id)
        REFERENCES academic_terms(tenant_id, school_id, id) ON DELETE CASCADE,
    CONSTRAINT fk_behavior_summaries_tenant_school
        FOREIGN KEY (tenant_id, school_id)
        REFERENCES cbc_schools(tenant_id, id) ON DELETE CASCADE,
    CONSTRAINT fk_behavior_summaries_category
        FOREIGN KEY (primary_category_id)
        REFERENCES behavior_categories(id) ON DELETE SET NULL
);

CREATE INDEX IF NOT EXISTS idx_behavior_summaries_tenant
    ON student_behavior_term_summaries (tenant_id);
CREATE INDEX IF NOT EXISTS idx_behavior_summaries_school
    ON student_behavior_term_summaries (school_id);
CREATE INDEX IF NOT EXISTS idx_behavior_summaries_student_term
    ON student_behavior_term_summaries (student_id, academic_term_id);
CREATE INDEX IF NOT EXISTS idx_behavior_summaries_term
    ON student_behavior_term_summaries (academic_term_id);

DROP TRIGGER IF EXISTS trg_student_behavior_term_summaries_updated_at
    ON student_behavior_term_summaries;
CREATE TRIGGER trg_student_behavior_term_summaries_updated_at
    BEFORE UPDATE ON student_behavior_term_summaries
    FOR EACH ROW EXECUTE FUNCTION fn_set_updated_at();

COMMENT ON TABLE student_behavior_term_summaries IS
    'Incremental materialised summary of behavior notes per student per term.
     Only counts APPROVED and INCLUDED_IN_REPORT notes in the main totals
     (total_incidents, urgent_count, commendations_count, disciplinary_count,
     primary_category_id). pending_review_count counts all PENDING_REVIEW
     notes for admin visibility. Refreshed via trigger on behavior_notes
     insert/update.';

COMMENT ON COLUMN student_behavior_term_summaries.total_incidents IS
    'Total count of APPROVED + INCLUDED_IN_REPORT behavior notes for this
     student+term. Excludes PENDING_REVIEW and REJECTED.';

COMMENT ON COLUMN student_behavior_term_summaries.urgent_count IS
    'Count of APPROVED + INCLUDED_IN_REPORT notes where is_urgent = true.';

COMMENT ON COLUMN student_behavior_term_summaries.commendations_count IS
    'Count of APPROVED + INCLUDED_IN_REPORT notes whose category has
     category_type = COMMENDATION.';

COMMENT ON COLUMN student_behavior_term_summaries.disciplinary_count IS
    'Count of APPROVED + INCLUDED_IN_REPORT notes whose category has
     category_type = DISCIPLINARY.';

COMMENT ON COLUMN student_behavior_term_summaries.pending_review_count IS
    'Count of PENDING_REVIEW notes for this student+term (regardless of
     approval status). Provides admin visibility into backlog.';

COMMENT ON COLUMN student_behavior_term_summaries.resolved_count IS
    'Count of notes with status in (APPROVED, INCLUDED_IN_REPORT, REJECTED)
     — any note that has been acted upon. total_incidents + pending_review_count
     + rejected notes = all notes for the term.';

COMMENT ON COLUMN student_behavior_term_summaries.primary_category_id IS
    'The behavior category with the highest count among APPROVED +
     INCLUDED_IN_REPORT notes for this student+term. Ties are resolved
     by the most recent note''s category. NULL when no applicable notes exist.';

-- ============================================================================
-- FUNCTION: fn_refresh_student_behavior_term_summary(target_student_id UUID,
--                                                     target_term_id UUID)
--
-- Recomputes the student_behavior_term_summary row for the given student+term
-- from scratch (idempotent upsert).
-- ============================================================================

CREATE OR REPLACE FUNCTION fn_refresh_student_behavior_term_summary(
    target_student_id UUID,
    target_term_id UUID
)
RETURNS void
LANGUAGE plpgsql
AS $$
DECLARE
    v_tenant_id UUID;
    v_school_id UUID;
BEGIN
    -- Resolve tenant_id and school_id from the student's enrollment in this term
    SELECT enr.tenant_id, enr.school_id
    INTO v_tenant_id, v_school_id
    FROM cbc_student_enrollments enr
    WHERE enr.student_id = target_student_id
      AND enr.academic_term_id = target_term_id
    LIMIT 1;

    -- If the student is not enrolled in this term, nothing to do
    IF NOT FOUND THEN
        RETURN;
    END IF;

    -- Upsert the summary row
    INSERT INTO student_behavior_term_summaries (
        tenant_id, school_id, student_id, academic_term_id,
        total_incidents, urgent_count,
        commendations_count, disciplinary_count,
        pending_review_count, resolved_count,
        primary_category_id, last_refreshed_at
    )
    WITH approved_notes AS (
        -- Only APPROVED and INCLUDED_IN_REPORT notes count toward main totals
        SELECT *
        FROM behavior_notes
        WHERE student_id = target_student_id
          AND tenant_id = v_tenant_id
          AND status IN ('APPROVED', 'INCLUDED_IN_REPORT')
    ),
    all_term_notes AS (
        -- All notes for this student+term (for pending/resolved counts)
        SELECT *
        FROM behavior_notes
        WHERE student_id = target_student_id
          AND tenant_id = v_tenant_id
    ),
    category_counts AS (
        SELECT
            category_id,
            COUNT(*) AS cnt,
            MAX(created_at) AS most_recent
        FROM approved_notes
        GROUP BY category_id
    ),
    ranked_categories AS (
        SELECT category_id
        FROM category_counts
        ORDER BY cnt DESC, most_recent DESC
        LIMIT 1
    )
    SELECT
        v_tenant_id,
        v_school_id,
        target_student_id,
        target_term_id,

        -- total_incidents: count of APPROVED + INCLUDED_IN_REPORT
        (SELECT COUNT(*) FROM approved_notes)::INT,

        -- urgent_count: those flagged urgent among approved
        (SELECT COUNT(*) FROM approved_notes WHERE is_urgent = true)::INT,

        -- commendations_count: approved notes in COMMENDATION categories
        (SELECT COUNT(*)
         FROM approved_notes an
         JOIN behavior_categories bc ON bc.id = an.category_id
         WHERE bc.category_type = 'COMMENDATION')::INT,

        -- disciplinary_count: approved notes in DISCIPLINARY categories
        (SELECT COUNT(*)
         FROM approved_notes an
         JOIN behavior_categories bc ON bc.id = an.category_id
         WHERE bc.category_type = 'DISCIPLINARY')::INT,

        -- pending_review_count: all PENDING_REVIEW notes (not just approved)
        (SELECT COUNT(*)
         FROM all_term_notes
         WHERE status = 'PENDING_REVIEW')::INT,

        -- resolved_count: notes with status APPROVED, INCLUDED_IN_REPORT, or REJECTED
        (SELECT COUNT(*)
         FROM all_term_notes
         WHERE status IN ('APPROVED', 'INCLUDED_IN_REPORT', 'REJECTED'))::INT,

        -- primary_category_id: highest-count category (tie-break by most recent)
        (SELECT category_id FROM ranked_categories),

        NOW()

    ON CONFLICT (student_id, academic_term_id)
    DO UPDATE SET
        tenant_id            = EXCLUDED.tenant_id,
        school_id            = EXCLUDED.school_id,
        total_incidents      = EXCLUDED.total_incidents,
        urgent_count         = EXCLUDED.urgent_count,
        commendations_count  = EXCLUDED.commendations_count,
        disciplinary_count   = EXCLUDED.disciplinary_count,
        pending_review_count = EXCLUDED.pending_review_count,
        resolved_count       = EXCLUDED.resolved_count,
        primary_category_id  = EXCLUDED.primary_category_id,
        last_refreshed_at    = NOW(),
        updated_at           = NOW();
END;
$$;

COMMENT ON FUNCTION fn_refresh_student_behavior_term_summary IS
    'Refreshes student_behavior_term_summaries for a single student+term.
     Idempotent — safe to call on INSERT or UPDATE of any behavior note.';

-- ============================================================================
-- FUNCTION: fn_refresh_student_behavior_term_summary_for_note()
-- Trigger function that resolves the student+term from the affected note
-- and calls fn_refresh_student_behavior_term_summary.
-- ============================================================================

CREATE OR REPLACE FUNCTION fn_refresh_student_behavior_term_summary_for_note()
RETURNS TRIGGER
LANGUAGE plpgsql
AS $$
DECLARE
    v_term_id UUID;
BEGIN
    -- Find the academic term that contains this note's date.
    -- We join the note's timetable_slot → class → enrollments → academic_term.
    SELECT enr.academic_term_id INTO v_term_id
    FROM cbc_student_enrollments enr
    WHERE enr.student_id = COALESCE(NEW.student_id, OLD.student_id)
      AND enr.status = 'ACTIVE'
      -- Use the note's date to find which term it falls in
      AND enr.academic_term_id IN (
          SELECT at.id
          FROM academic_terms at
          WHERE at.tenant_id = COALESCE(NEW.tenant_id, OLD.tenant_id)
            AND at.school_id = COALESCE(NEW.school_id, OLD.school_id)
            AND COALESCE(NEW.date, OLD.date) BETWEEN at.start_date AND at.end_date
      )
    LIMIT 1;

    IF FOUND THEN
        PERFORM fn_refresh_student_behavior_term_summary(
            COALESCE(NEW.student_id, OLD.student_id),
            v_term_id
        );
    END IF;

    RETURN COALESCE(NEW, OLD);
END;
$$;

COMMENT ON FUNCTION fn_refresh_student_behavior_term_summary_for_note IS
    'Trigger function: on INSERT or UPDATE of behavior_notes, resolves
     the student+enrollment term from the note''s date and refreshes the
     student_behavior_term_summary.';

-- ============================================================================
-- TRIGGER: trg_behavior_notes_refresh_term_summary
-- Fires AFTER INSERT OR UPDATE on behavior_notes.
-- Calls fn_refresh_student_behavior_term_summary_for_note for the affected note.
-- ============================================================================

DROP TRIGGER IF EXISTS trg_behavior_notes_refresh_term_summary
    ON behavior_notes;
CREATE TRIGGER trg_behavior_notes_refresh_term_summary
    AFTER INSERT OR UPDATE ON behavior_notes
    FOR EACH ROW
    EXECUTE FUNCTION fn_refresh_student_behavior_term_summary_for_note();

COMMENT ON TRIGGER trg_behavior_notes_refresh_term_summary ON behavior_notes IS
    'After a behavior note is inserted or updated, refresh the
     student_behavior_term_summary for the affected student+term.';

-- ============================================================================
-- RLS POLICY
-- ============================================================================

ALTER TABLE IF EXISTS student_behavior_term_summaries ENABLE ROW LEVEL SECURITY;

DO $$ BEGIN
    DROP POLICY IF EXISTS tenant_isolation_policy ON student_behavior_term_summaries;
    CREATE POLICY tenant_isolation_policy ON student_behavior_term_summaries
        FOR ALL
        USING (tenant_id = fn_current_tenant_id())
        WITH CHECK (tenant_id = fn_current_tenant_id());
END $$;

COMMENT ON TABLE student_behavior_term_summaries IS
    'Incremental materialised summary of behavior notes per student per term.
     RLS-enabled — tenant-scoped.';

-- Migration: 000012_create_teacher_subject_performance_summaries
-- Creates the teacher_subject_performance_summaries table — a periodic batch
-- summary of teacher effectiveness metrics per learning area, class, and term.
--
-- Grain: (user_id, learning_area_id, class_id, academic_term_id)
--
-- Teacher attribution: the teacher is resolved from cbc_class_teachers at
-- computation time via the current SUBJECT_TEACHER row for that
-- class+learning_area. There is no historical assignment tracking, so a
-- mid-term substitute or reassignment gets folded into whoever holds the
-- role at computation time. This is an approximation — flag it in the UI.
--
-- This is a PERIODIC BATCH-ONLY table. It is NOT updated incrementally.
-- Computation is triggered once per term close (or on-demand).

-- ============================================================================
-- TABLE: teacher_subject_performance_summaries
-- ============================================================================

CREATE TABLE IF NOT EXISTS teacher_subject_performance_summaries (
    id                          UUID          PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id                   UUID          NOT NULL,
    school_id                   UUID          NOT NULL,
    user_id                     UUID          NOT NULL,
    learning_area_id            UUID          NOT NULL,
    class_id                    UUID          NOT NULL,
    academic_term_id            UUID          NOT NULL,
    subject_mean_score          NUMERIC(5,2),           -- avg of student avg_percentages
    cohort_mastery_rate         NUMERIC(5,2),           -- % students at ME or EE
    student_growth_rate         NUMERIC(6,2),           -- avg % change vs prior term
    assessment_timeliness_index NUMERIC(5,2),           -- % sessions published on/before scheduled
    strand_coverage_rate        NUMERIC(5,2),           -- % of learning area strands assessed
    last_refreshed_at           TIMESTAMPTZ   NOT NULL DEFAULT NOW(),
    created_at                  TIMESTAMPTZ   NOT NULL DEFAULT NOW(),
    updated_at                  TIMESTAMPTZ   NOT NULL DEFAULT NOW(),

    CONSTRAINT uq_teacher_subject_class_term
        UNIQUE (user_id, learning_area_id, class_id, academic_term_id),
    CONSTRAINT fk_teacher_perf_summaries_tenant_school
        FOREIGN KEY (tenant_id, school_id)
        REFERENCES cbc_schools(tenant_id, id) ON DELETE CASCADE,
    CONSTRAINT fk_teacher_perf_summaries_tenant_user
        FOREIGN KEY (tenant_id, user_id)
        REFERENCES users(tenant_id, id) ON DELETE CASCADE,
    CONSTRAINT fk_teacher_perf_summaries_learning_area
        FOREIGN KEY (learning_area_id)
        REFERENCES cbc_learning_areas(id) ON DELETE CASCADE,
    CONSTRAINT fk_teacher_perf_summaries_class
        FOREIGN KEY (class_id)
        REFERENCES cbc_classes(id) ON DELETE CASCADE,
    CONSTRAINT fk_teacher_perf_summaries_term
        FOREIGN KEY (tenant_id, school_id, academic_term_id)
        REFERENCES academic_terms(tenant_id, school_id, id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_teacher_perf_summaries_tenant
    ON teacher_subject_performance_summaries (tenant_id);
CREATE INDEX IF NOT EXISTS idx_teacher_perf_summaries_school
    ON teacher_subject_performance_summaries (school_id);
CREATE INDEX IF NOT EXISTS idx_teacher_perf_summaries_user
    ON teacher_subject_performance_summaries (user_id);
CREATE INDEX IF NOT EXISTS idx_teacher_perf_summaries_term
    ON teacher_subject_performance_summaries (academic_term_id);
CREATE INDEX IF NOT EXISTS idx_teacher_perf_summaries_class_term
    ON teacher_subject_performance_summaries (class_id, academic_term_id);

DROP TRIGGER IF EXISTS trg_teacher_subject_performance_summaries_updated_at
    ON teacher_subject_performance_summaries;
CREATE TRIGGER trg_teacher_subject_performance_summaries_updated_at
    BEFORE UPDATE ON teacher_subject_performance_summaries
    FOR EACH ROW EXECUTE FUNCTION fn_set_updated_at();

COMMENT ON TABLE teacher_subject_performance_summaries IS
    'Periodic batch-computed teacher effectiveness summary per subject per
     class per term. Teacher attribution uses current cbc_class_teachers
     SUBJECT_TEACHER row — no historical tracking, so mid-term reassignments
     are folded in. Approximation — flag in UI.';

COMMENT ON COLUMN teacher_subject_performance_summaries.subject_mean_score IS
    'Average of student_term_subject_summaries.average_percentage across
     all students enrolled in this class+learning_area+term. NULL when no
     assessment data exists.';

COMMENT ON COLUMN teacher_subject_performance_summaries.cohort_mastery_rate IS
    'Percentage of enrolled students whose mapped_performance_level is ME
     or EE in this class+learning_area+term. NULL when no data exists.';

COMMENT ON COLUMN teacher_subject_performance_summaries.student_growth_rate IS
    'Average percentage-point change (current term vs prior term) for
     students who were enrolled in both terms in this learning area.
     Positive = improvement; Negative = decline. NULL for Term 1 (no prior
     term) or when insufficient matched students exist.';

COMMENT ON COLUMN teacher_subject_performance_summaries.assessment_timeliness_index IS
    'Percentage of PUBLISHED assessment sessions for this class+learning_area
     +term that were published on or before their scheduled_date. A high rate
     indicates timely assessment completion. NULL when no sessions exist.';

COMMENT ON COLUMN teacher_subject_performance_summaries.strand_coverage_rate IS
    'Percentage of cbc_strands for this learning_area that have at least one
     PUBLISHED RUBRIC assessment session in this term. NULL when no strands
     exist for the learning area.';

-- ============================================================================
-- FUNCTION: fn_compute_teacher_subject_performance_summaries(target_term_id UUID)
--
-- Computes (or recomputes) teacher_subject_performance_summaries for ALL
-- SUBJECT_TEACHER assignments in the given academic term.
--
-- Algorithm per (teacher, learning_area, class):
--   1. Resolve the teacher via cbc_class_teachers WHERE teacher_role =
--      'SUBJECT_TEACHER' AND learning_area_id matches.
--   2. subject_mean_score = AVG(stss.average_percentage) for all students
--      in that class+term+learning_area.
--   3. cohort_mastery_rate = percentage of students whose
--      mapped_performance_level IN ('ME','EE').
--   4. student_growth_rate = for students who have data in both current and
--      prior term, AVG(current avg% - prior avg%).
--   5. assessment_timeliness_index = for PUBLISHED sessions with a
--      scheduled_date, percentage published before or on that date.
--   6. strand_coverage_rate = number of strands with >=1 RUBRIC PUBLISHED
--      session / total strands for the learning area.
-- ============================================================================

CREATE OR REPLACE FUNCTION fn_compute_teacher_subject_performance_summaries(target_term_id UUID)
RETURNS void
LANGUAGE plpgsql
AS $$
DECLARE
    v_tenant_id        UUID;
    v_school_id        UUID;
    v_prior_term_id    UUID;
BEGIN
    -- Resolve term metadata
    SELECT tenant_id, school_id
    INTO v_tenant_id, v_school_id
    FROM academic_terms
    WHERE id = target_term_id;

    IF NOT FOUND THEN
        RETURN;
    END IF;

    -- Find the prior term (same academic year, term_number - 1)
    -- or the last term of the previous academic year
    WITH term_info AS (
        SELECT term_number, academic_year_id
        FROM academic_terms
        WHERE id = target_term_id
    )
    SELECT at.id INTO v_prior_term_id
    FROM academic_terms at, term_info ti
    WHERE at.tenant_id = v_tenant_id
      AND at.school_id = v_school_id
      AND (
          (ti.term_number > 1
           AND at.academic_year_id = ti.academic_year_id
           AND at.term_number = ti.term_number - 1)
          OR
          (ti.term_number = 1
           AND at.academic_year_id = (
               SELECT id FROM academic_years
               WHERE tenant_id = v_tenant_id
                 AND school_id = v_school_id
                 AND name::INT = (
                     SELECT (name::INT - 1) FROM academic_years WHERE id = ti.academic_year_id
                 )
               LIMIT 1
           )
           AND at.term_number = 3)
      )
    LIMIT 1;

    -- Compute and upsert
    INSERT INTO teacher_subject_performance_summaries (
        tenant_id, school_id, user_id, learning_area_id, class_id, academic_term_id,
        subject_mean_score, cohort_mastery_rate, student_growth_rate,
        assessment_timeliness_index, strand_coverage_rate, last_refreshed_at
    )
    WITH
    -- All SUBJECT_TEACHER assignments for the school + term
    teacher_assignments AS (
        SELECT
            ct.user_id,
            ct.learning_area_id,
            ct.class_id
        FROM cbc_class_teachers ct
        WHERE ct.tenant_id = v_tenant_id
          AND ct.teacher_role = 'SUBJECT_TEACHER'
          AND ct.learning_area_id IS NOT NULL
    ),
    -- Students enrolled in this term for each class
    class_enrollments AS (
        SELECT
            enr.class_id,
            enr.student_id
        FROM cbc_student_enrollments enr
        WHERE enr.academic_term_id = target_term_id
          AND enr.tenant_id = v_tenant_id
          AND enr.school_id = v_school_id
          AND enr.class_id IS NOT NULL
          AND enr.status = 'ACTIVE'
    ),
    -- Student term subject summaries for this term
    current_summaries AS (
        SELECT
            stss.student_id,
            stss.learning_area_id,
            stss.average_percentage,
            stss.mapped_performance_level
        FROM student_term_subject_summaries stss
        WHERE stss.academic_term_id = target_term_id
          AND stss.tenant_id = v_tenant_id
          AND stss.school_id = v_school_id
          AND stss.average_percentage IS NOT NULL
    ),
    -- Student term subject summaries for the prior term (if exists)
    prior_summaries AS (
        SELECT
            stss.student_id,
            stss.learning_area_id,
            stss.average_percentage
        FROM student_term_subject_summaries stss
        WHERE stss.academic_term_id = v_prior_term_id
          AND stss.tenant_id = v_tenant_id
          AND stss.school_id = v_school_id
          AND stss.average_percentage IS NOT NULL
    ),
    -- Compute per-assignment metrics
    assignment_metrics AS (
        SELECT
            ta.user_id,
            ta.learning_area_id,
            ta.class_id,
            -- subject_mean_score
            ROUND(AVG(cs.average_percentage)::numeric, 2) AS subject_mean_score,
            -- cohort_mastery_rate
            CASE
                WHEN COUNT(cs.*) > 0
                THEN ROUND(
                    (COUNT(*) FILTER (WHERE cs.mapped_performance_level IN ('ME', 'EE'))::numeric
                     / COUNT(*)::numeric * 100),
                    2
                )
                ELSE NULL
            END AS cohort_mastery_rate,
            -- student_growth_rate: for students with data in both terms
            CASE
                WHEN v_prior_term_id IS NOT NULL THEN (
                    SELECT ROUND(AVG(delta)::numeric, 2)
                    FROM (
                        SELECT
                            cs.student_id,
                            cs.average_percentage - ps.average_percentage AS delta
                        FROM current_summaries cs
                        JOIN prior_summaries ps
                            ON ps.student_id = cs.student_id
                            AND ps.learning_area_id = cs.learning_area_id
                        WHERE cs.learning_area_id = ta.learning_area_id
                          AND cs.student_id IN (
                              SELECT ce.student_id FROM class_enrollments ce WHERE ce.class_id = ta.class_id
                          )
                    ) deltas
                    WHERE delta IS NOT NULL
                )
                ELSE NULL
            END AS student_growth_rate,
            -- assessment_timeliness_index
            (
                SELECT
                    CASE
                        WHEN COUNT(*) > 0
                        THEN ROUND(
                            (COUNT(*) FILTER (
                                WHERE s.status = 'PUBLISHED'
                                  AND s.scheduled_date IS NOT NULL
                                  AND s.updated_at::DATE <= s.scheduled_date::DATE
                            ))::numeric / COUNT(*)::numeric * 100,
                            2
                        )
                        ELSE NULL
                    END
                FROM assessment_sessions s
                WHERE s.tenant_id = v_tenant_id
                  AND s.school_id = v_school_id
                  AND s.academic_term_id = target_term_id
                  AND s.learning_area_id = ta.learning_area_id
                  AND s.class_id = ta.class_id
            ) AS assessment_timeliness_index,
            -- strand_coverage_rate: % of learning area strands covered by
            -- published RUBRIC assessments in this term. We determine coverage
            -- by looking at outcome grades' performance_indicators -> sub_strands
            -- -> strands.
            (
                SELECT
                    CASE
                        WHEN total_strands.count > 0
                        THEN ROUND(
                            covered_strands.count::numeric / total_strands.count::numeric * 100,
                            2
                        )
                        ELSE NULL
                    END
                FROM (
                    SELECT COUNT(*) AS count
                    FROM cbc_strands s
                    WHERE s.learning_area_id = ta.learning_area_id
                ) total_strands
                CROSS JOIN (
                    SELECT COUNT(DISTINCT str.id) AS count
                    FROM assessment_sessions asess
                    JOIN student_assessment_outcome_grades sog
                        ON sog.session_id = asess.id
                    JOIN performance_indicators pi
                        ON pi.id = sog.performance_indicator_id
                    JOIN cbc_sub_strands sstr
                        ON sstr.id = pi.sub_strand_id
                    JOIN cbc_strands str
                        ON str.id = sstr.strand_id
                    WHERE asess.tenant_id = v_tenant_id
                      AND asess.school_id = v_school_id
                      AND asess.academic_term_id = target_term_id
                      AND asess.learning_area_id = ta.learning_area_id
                      AND asess.class_id = ta.class_id
                      AND asess.status = 'PUBLISHED'
                      AND asess.evaluation_method = 'RUBRIC'
                ) covered_strands
            ) AS strand_coverage_rate
        FROM teacher_assignments ta
        LEFT JOIN class_enrollments ce ON ce.class_id = ta.class_id
        LEFT JOIN current_summaries cs
            ON cs.student_id = ce.student_id
            AND cs.learning_area_id = ta.learning_area_id
        GROUP BY ta.user_id, ta.learning_area_id, ta.class_id
    )
    SELECT
        v_tenant_id,
        v_school_id,
        am.user_id,
        am.learning_area_id,
        am.class_id,
        target_term_id,
        am.subject_mean_score,
        am.cohort_mastery_rate,
        am.student_growth_rate,
        am.assessment_timeliness_index,
        am.strand_coverage_rate,
        NOW()
    FROM assignment_metrics am
    -- Only insert rows that have at least one non-null data point
    WHERE am.subject_mean_score IS NOT NULL
       OR am.cohort_mastery_rate IS NOT NULL
       OR am.student_growth_rate IS NOT NULL
       OR am.assessment_timeliness_index IS NOT NULL
       OR am.strand_coverage_rate IS NOT NULL

    ON CONFLICT (user_id, learning_area_id, class_id, academic_term_id)
    DO UPDATE SET
        subject_mean_score          = EXCLUDED.subject_mean_score,
        cohort_mastery_rate         = EXCLUDED.cohort_mastery_rate,
        student_growth_rate         = EXCLUDED.student_growth_rate,
        assessment_timeliness_index = EXCLUDED.assessment_timeliness_index,
        strand_coverage_rate        = EXCLUDED.strand_coverage_rate,
        last_refreshed_at           = NOW(),
        updated_at                  = NOW();

    -- Clean up orphaned rows where the teacher is no longer assigned
    DELETE FROM teacher_subject_performance_summaries
    WHERE academic_term_id = target_term_id
      AND tenant_id = v_tenant_id
      AND school_id = v_school_id
      AND (user_id, learning_area_id, class_id) NOT IN (
          SELECT ct.user_id, ct.learning_area_id, ct.class_id
          FROM cbc_class_teachers ct
          WHERE ct.tenant_id = v_tenant_id
            AND ct.teacher_role = 'SUBJECT_TEACHER'
            AND ct.learning_area_id IS NOT NULL
      );
END;
$$;

COMMENT ON FUNCTION fn_compute_teacher_subject_performance_summaries IS
    'Batch-computes teacher_subject_performance_summaries for all
     SUBJECT_TEACHER assignments in the given term. Uses current assessment
     data and prior-term summaries for growth. Must be called on a schedule
     (once per term close). Teacher attribution is based on the current
     cbc_class_teachers row — no historical tracking.';

-- ============================================================================
-- RLS POLICY
-- ============================================================================

ALTER TABLE IF EXISTS teacher_subject_performance_summaries ENABLE ROW LEVEL SECURITY;

DO $$ BEGIN
    DROP POLICY IF EXISTS tenant_isolation_policy ON teacher_subject_performance_summaries;
    CREATE POLICY tenant_isolation_policy ON teacher_subject_performance_summaries
        FOR ALL
        USING (tenant_id = fn_current_tenant_id())
        WITH CHECK (tenant_id = fn_current_tenant_id());
END $$;

COMMENT ON TABLE teacher_subject_performance_summaries IS
    'Periodic batch-computed teacher effectiveness summary per subject per
     class per term. RLS-enabled — tenant-scoped.';

-- Migration: 000013_create_teacher_delivery_summaries
-- Creates the teacher_delivery_summaries table — an incrementally updated
-- summary of teacher lesson delivery metrics per term.
--
-- Grain: (user_id, academic_term_id)
--
-- Incremental task: triggered on attendance_records insert and on
-- cbc_attendance_sessions.status changes to SKIPPED. Slot ownership
-- resolved via cbc_timetable_slots.teacher_id.

-- ============================================================================
-- TABLE: teacher_delivery_summaries
-- ============================================================================

CREATE TABLE IF NOT EXISTS teacher_delivery_summaries (
    id                      UUID          PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id               UUID          NOT NULL,
    school_id               UUID          NOT NULL,
    user_id                 UUID          NOT NULL,
    academic_term_id        UUID          NOT NULL,
    total_assigned_slots    INT           NOT NULL DEFAULT 0,
    marked_slots            INT           NOT NULL DEFAULT 0,
    missed_slots            INT           NOT NULL DEFAULT 0,
    sessions_created        INT           NOT NULL DEFAULT 0,
    sessions_approved       INT           NOT NULL DEFAULT 0,
    on_time_submission_rate NUMERIC(5,2)  NULL,
    last_refreshed_at       TIMESTAMPTZ   NOT NULL DEFAULT NOW(),
    created_at              TIMESTAMPTZ   NOT NULL DEFAULT NOW(),
    updated_at              TIMESTAMPTZ   NOT NULL DEFAULT NOW(),

    CONSTRAINT uq_teacher_delivery_term UNIQUE (user_id, academic_term_id),
    CONSTRAINT fk_teacher_delivery_tenant_school
        FOREIGN KEY (tenant_id, school_id)
        REFERENCES cbc_schools(tenant_id, id) ON DELETE CASCADE,
    CONSTRAINT fk_teacher_delivery_tenant_user
        FOREIGN KEY (tenant_id, user_id)
        REFERENCES users(tenant_id, id) ON DELETE CASCADE,
    CONSTRAINT fk_teacher_delivery_term
        FOREIGN KEY (tenant_id, school_id, academic_term_id)
        REFERENCES academic_terms(tenant_id, school_id, id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_teacher_delivery_tenant
    ON teacher_delivery_summaries (tenant_id);
CREATE INDEX IF NOT EXISTS idx_teacher_delivery_school
    ON teacher_delivery_summaries (school_id);
CREATE INDEX IF NOT EXISTS idx_teacher_delivery_user
    ON teacher_delivery_summaries (user_id);
CREATE INDEX IF NOT EXISTS idx_teacher_delivery_term
    ON teacher_delivery_summaries (academic_term_id);

DROP TRIGGER IF EXISTS trg_teacher_delivery_summaries_updated_at
    ON teacher_delivery_summaries;
CREATE TRIGGER trg_teacher_delivery_summaries_updated_at
    BEFORE UPDATE ON teacher_delivery_summaries
    FOR EACH ROW EXECUTE FUNCTION fn_set_updated_at();

COMMENT ON TABLE teacher_delivery_summaries IS
    'Incrementally updated summary of teacher lesson delivery metrics per term.
     Grain: (user_id, academic_term_id). Updated via triggers on
     attendance_records INSERT and cbc_attendance_sessions status changes.';

COMMENT ON COLUMN teacher_delivery_summaries.total_assigned_slots IS
    'Total number of timetable slot occurrences assigned to this teacher during
     the term. Computed as the count of (cbc_timetable_slots × weeks) where
     the slot day_of_week falls within the term date range.';

COMMENT ON COLUMN teacher_delivery_summaries.marked_slots IS
    'Number of assigned slot occurrences where attendance_records exist
     (attendance was taken).';

COMMENT ON COLUMN teacher_delivery_summaries.missed_slots IS
    'Number of assigned slot occurrences where the lesson was marked SKIPPED
     (lesson did not take place).';

COMMENT ON COLUMN teacher_delivery_summaries.sessions_created IS
    'Number of cbc_attendance_sessions records associated with this teacher''s
     slots in the term (any session status).';

COMMENT ON COLUMN teacher_delivery_summaries.sessions_approved IS
    'Number of sessions where status = SUBMITTED (attendance was formally
     recorded and approved).';

COMMENT ON COLUMN teacher_delivery_summaries.on_time_submission_rate IS
    'Percentage of assigned slots that were either marked or skipped:
     (marked_slots + missed_slots) / total_assigned_slots * 100.';

-- ============================================================================
-- FUNCTION: fn_compute_teacher_delivery_summaries(target_term_id UUID)
--
-- Computes (or recomputes) teacher_delivery_summaries for ALL teachers in
-- the given academic term.
--
-- Algorithm per teacher:
--   1. Resolve all timetable slots assigned to the teacher.
--   2. Count expected occurrences per slot within the term date range
--      (matching day_of_week).
--   3. Count attendance_records per (slot_id, date) for this teacher's slots.
--   4. Count SKIPPED sessions per (slot_id, date) for this teacher's slots.
--   5. Compute on_time_submission_rate.
-- ============================================================================

CREATE OR REPLACE FUNCTION fn_compute_teacher_delivery_summaries(target_term_id UUID)
RETURNS void
LANGUAGE plpgsql
AS $$
DECLARE
    v_tenant_id UUID;
    v_school_id UUID;
    v_term_start DATE;
    v_term_end DATE;
    v_academic_year_id UUID;
BEGIN
    -- Resolve term metadata
    SELECT at.tenant_id, at.school_id, at.start_date, at.end_date, at.academic_year_id
    INTO v_tenant_id, v_school_id, v_term_start, v_term_end, v_academic_year_id
    FROM academic_terms at
    WHERE at.id = target_term_id;

    IF NOT FOUND THEN
        RETURN;
    END IF;

    -- Compute and upsert delivery summaries per teacher
    INSERT INTO teacher_delivery_summaries (
        tenant_id, school_id, user_id, academic_term_id,
        total_assigned_slots, marked_slots, missed_slots,
        sessions_created, sessions_approved,
        on_time_submission_rate, last_refreshed_at
    )
    WITH
    -- All teachers with timetable slots in this academic year + school
    teacher_slots AS (
        SELECT DISTINCT
            ts.teacher_id AS user_id,
            ts.id AS slot_id,
            tstr.day_of_week,
            ts.class_id,
            ts.learning_area_id
        FROM cbc_timetable_slots ts
        JOIN timetable_structures tstr ON tstr.id = ts.structure_id
        WHERE ts.tenant_id = v_tenant_id
          AND ts.school_id = v_school_id
          AND ts.academic_year_id = v_academic_year_id
          AND tstr.is_break = false
    ),
    -- Generate expected slot occurrences within the term (one per week per slot)
    -- by cross-joining with a date series that matches day_of_week
    slot_occurrences AS (
        SELECT
            ts.user_id,
            ts.slot_id,
            d.date::DATE AS occurrence_date
        FROM teacher_slots ts
        CROSS JOIN LATERAL (
            SELECT generate_series(
                v_term_start,
                v_term_end,
                '1 day'::INTERVAL
            )::DATE AS date
        ) d
        WHERE EXTRACT(DOW FROM d.date) = ts.day_of_week
          -- Adjust DOW: PostgreSQL DOW is 0=Sun, 1=Mon...6=Sat
          -- Our day_of_week: 1=Mon...7=Sun
          AND (
              (ts.day_of_week = 7 AND EXTRACT(DOW FROM d.date) = 0)
              OR
              (ts.day_of_week = EXTRACT(DOW FROM d.date))
          )
    ),
    -- Count marked slots: slot+date combinations with attendance records
    marked AS (
        SELECT
            ar.tenant_id,
            ts.teacher_id AS user_id,
            COUNT(DISTINCT (ar.timetable_slot_id, ar.date))::INT AS marked_count
        FROM attendance_records ar
        JOIN cbc_timetable_slots ts ON ts.id = ar.timetable_slot_id
        WHERE ar.tenant_id = v_tenant_id
          AND ar.academic_term_id = target_term_id
          AND ts.teacher_id IS NOT NULL
        GROUP BY ar.tenant_id, ts.teacher_id
    ),
    -- Count missed slots: sessions with status = SKIPPED
    missed AS (
        SELECT
            s.tenant_id,
            ts.teacher_id AS user_id,
            COUNT(*)::INT AS missed_count
        FROM cbc_attendance_sessions s
        JOIN cbc_timetable_slots ts ON ts.id = s.timetable_slot_id
        WHERE s.tenant_id = v_tenant_id
          AND s.date >= v_term_start
          AND s.date <= v_term_end
          AND s.status = 'SKIPPED'
          AND ts.teacher_id IS NOT NULL
        GROUP BY s.tenant_id, ts.teacher_id
    ),
    -- Count sessions created
    sessions_created_cte AS (
        SELECT
            s.tenant_id,
            ts.teacher_id AS user_id,
            COUNT(*)::INT AS sessions_count
        FROM cbc_attendance_sessions s
        JOIN cbc_timetable_slots ts ON ts.id = s.timetable_slot_id
        WHERE s.tenant_id = v_tenant_id
          AND s.date >= v_term_start
          AND s.date <= v_term_end
          AND ts.teacher_id IS NOT NULL
        GROUP BY s.tenant_id, ts.teacher_id
    ),
    -- Count sessions approved (status = SUBMITTED)
    sessions_approved_cte AS (
        SELECT
            s.tenant_id,
            ts.teacher_id AS user_id,
            COUNT(*)::INT AS approved_count
        FROM cbc_attendance_sessions s
        JOIN cbc_timetable_slots ts ON ts.id = s.timetable_slot_id
        WHERE s.tenant_id = v_tenant_id
          AND s.date >= v_term_start
          AND s.date <= v_term_end
          AND s.status = 'SUBMITTED'
          AND ts.teacher_id IS NOT NULL
        GROUP BY s.tenant_id, ts.teacher_id
    ),
    -- Aggregate per teacher
    teacher_aggregates AS (
        SELECT
            so.user_id,
            COUNT(DISTINCT (so.slot_id, so.occurrence_date))::INT AS total_assigned,
            COALESCE(m.marked_count, 0) AS marked,
            COALESCE(mi.missed_count, 0) AS missed,
            COALESCE(sc.sessions_count, 0) AS sessions_created_count,
            COALESCE(sa.approved_count, 0) AS sessions_approved_count
        FROM slot_occurrences so
        LEFT JOIN marked m ON m.tenant_id = v_tenant_id AND m.user_id = so.user_id
        LEFT JOIN missed mi ON mi.tenant_id = v_tenant_id AND mi.user_id = so.user_id
        LEFT JOIN sessions_created_cte sc ON sc.tenant_id = v_tenant_id AND sc.user_id = so.user_id
        LEFT JOIN sessions_approved_cte sa ON sa.tenant_id = v_tenant_id AND sa.user_id = so.user_id
        GROUP BY so.user_id, m.marked_count, mi.missed_count, sc.sessions_count, sa.approved_count
    )
    SELECT
        v_tenant_id,
        v_school_id,
        ta.user_id,
        target_term_id,
        ta.total_assigned,
        ta.marked,
        ta.missed,
        ta.sessions_created_count,
        ta.sessions_approved_count,
        CASE
            WHEN ta.total_assigned > 0
            THEN ROUND(
                ((ta.marked + ta.missed)::NUMERIC / ta.total_assigned::NUMERIC) * 100,
                2
            )
            ELSE NULL
        END,
        NOW()
    FROM teacher_aggregates ta

    ON CONFLICT (user_id, academic_term_id)
    DO UPDATE SET
        total_assigned_slots    = EXCLUDED.total_assigned_slots,
        marked_slots            = EXCLUDED.marked_slots,
        missed_slots            = EXCLUDED.missed_slots,
        sessions_created        = EXCLUDED.sessions_created,
        sessions_approved       = EXCLUDED.sessions_approved,
        on_time_submission_rate = EXCLUDED.on_time_submission_rate,
        last_refreshed_at       = NOW(),
        updated_at              = NOW();

    -- Clean up orphaned rows where the teacher no longer has any slots
    DELETE FROM teacher_delivery_summaries
    WHERE academic_term_id = target_term_id
      AND tenant_id = v_tenant_id
      AND school_id = v_school_id
      AND user_id NOT IN (
          SELECT DISTINCT ts.teacher_id
          FROM cbc_timetable_slots ts
          WHERE ts.tenant_id = v_tenant_id
            AND ts.school_id = v_school_id
            AND ts.academic_year_id = v_academic_year_id
            AND ts.teacher_id IS NOT NULL
      );
END;
$$;

COMMENT ON FUNCTION fn_compute_teacher_delivery_summaries IS
    'Batch-computes teacher_delivery_summaries for all teachers with timetable
     slots in the given term. Uses attendance_records and
     cbc_attendance_sessions to calculate delivery metrics.';

-- ============================================================================
-- RLS POLICY
-- ============================================================================

ALTER TABLE IF EXISTS teacher_delivery_summaries ENABLE ROW LEVEL SECURITY;

DO $$ BEGIN
    DROP POLICY IF EXISTS tenant_isolation_policy ON teacher_delivery_summaries;
    CREATE POLICY tenant_isolation_policy ON teacher_delivery_summaries
        FOR ALL
        USING (tenant_id = fn_current_tenant_id())
        WITH CHECK (tenant_id = fn_current_tenant_id());
END $$;

COMMENT ON TABLE teacher_delivery_summaries IS
    'Incrementally updated summary of teacher lesson delivery metrics per term.
     RLS-enabled — tenant-scoped.';

-- Migration: 000014_create_teacher_workload_summaries
-- Creates the teacher_workload_summaries table — a computed summary of
-- teacher workload metrics per academic year. Reassignments via timetable
-- slots or class-teacher assignments are infrequent, so this table is
-- batch-computed on-demand rather than incrementally triggered.
--
-- Grain: (user_id, academic_year_id)

-- ============================================================================
-- TABLE: teacher_workload_summaries
-- ============================================================================

CREATE TABLE IF NOT EXISTS teacher_workload_summaries (
    id                     UUID          PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id              UUID          NOT NULL,
    school_id              UUID          NOT NULL,
    user_id                UUID          NOT NULL,
    academic_year_id       UUID          NOT NULL,
    total_assigned_periods INT           NOT NULL DEFAULT 0,
    unique_subjects        INT           NOT NULL DEFAULT 0,
    classes_taught         INT           NOT NULL DEFAULT 0,
    utilization_percentage NUMERIC(5,2)  NULL,
    is_overcapacity        BOOLEAN       NOT NULL DEFAULT FALSE,
    last_refreshed_at      TIMESTAMPTZ   NOT NULL DEFAULT NOW(),
    created_at             TIMESTAMPTZ   NOT NULL DEFAULT NOW(),
    updated_at             TIMESTAMPTZ   NOT NULL DEFAULT NOW(),

    CONSTRAINT uq_teacher_workload_year UNIQUE (user_id, academic_year_id),
    CONSTRAINT fk_teacher_workload_tenant_school
        FOREIGN KEY (tenant_id, school_id)
        REFERENCES cbc_schools(tenant_id, id) ON DELETE CASCADE,
    CONSTRAINT fk_teacher_workload_tenant_user
        FOREIGN KEY (tenant_id, user_id)
        REFERENCES users(tenant_id, id) ON DELETE CASCADE,
    CONSTRAINT fk_teacher_workload_year
        FOREIGN KEY (tenant_id, academic_year_id)
        REFERENCES academic_years(tenant_id, id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_teacher_workload_tenant
    ON teacher_workload_summaries (tenant_id);
CREATE INDEX IF NOT EXISTS idx_teacher_workload_school
    ON teacher_workload_summaries (school_id);
CREATE INDEX IF NOT EXISTS idx_teacher_workload_user
    ON teacher_workload_summaries (user_id);
CREATE INDEX IF NOT EXISTS idx_teacher_workload_year
    ON teacher_workload_summaries (academic_year_id);

DROP TRIGGER IF EXISTS trg_teacher_workload_summaries_updated_at
    ON teacher_workload_summaries;
CREATE TRIGGER trg_teacher_workload_summaries_updated_at
    BEFORE UPDATE ON teacher_workload_summaries
    FOR EACH ROW EXECUTE FUNCTION fn_set_updated_at();

COMMENT ON TABLE teacher_workload_summaries IS
    'Batch-computed summary of teacher workload metrics per academic year.
     Grain: (user_id, academic_year_id). Recomputes on-demand — reassignments
     via timetable slots or cbc_class_teachers are infrequent.';

COMMENT ON COLUMN teacher_workload_summaries.total_assigned_periods IS
    'Number of weekly timetable slots assigned to this teacher. Represents the
     per-week instructional load (e.g. 24 periods/week).';

COMMENT ON COLUMN teacher_workload_summaries.unique_subjects IS
    'Count of distinct learning areas (subjects) assigned to this teacher.';

COMMENT ON COLUMN teacher_workload_summaries.classes_taught IS
    'Count of distinct classes this teacher has timetable assignments for.';

COMMENT ON COLUMN teacher_workload_summaries.utilization_percentage IS
    'Percentage of the school''s total weekly instructional periods that this
     teacher covers. Computed as total_assigned_periods / total_school_periods
     * 100. NULL when no timetable structures exist for the school.';

COMMENT ON COLUMN teacher_workload_summaries.is_overcapacity IS
    'TRUE when the teacher''s assigned periods exceed the school''s average
     teacher capacity per week. Currently flagged when utilization exceeds
     100% of a simple heuristic (total school periods / active teachers).';

-- ============================================================================
-- FUNCTION: fn_compute_teacher_workload_summaries(target_year_id UUID)
--
-- Computes (or recomputes) teacher_workload_summaries for ALL teachers in
-- the given academic year.
--
-- Algorithm per teacher:
--   1. Count timetable slots per teacher (weekly period count).
--   2. Count distinct learning_area_ids from those slots.
--   3. Count distinct class_ids from those slots.
--   4. Compute utilization vs school-wide non-break period count.
--   5. Flag overcapacity when utilization > 100.
-- ============================================================================

CREATE OR REPLACE FUNCTION fn_compute_teacher_workload_summaries(target_year_id UUID)
RETURNS void
LANGUAGE plpgsql
AS $$
DECLARE
    v_tenant_id UUID;
    v_school_id UUID;
    v_total_school_periods INT;
    v_active_teacher_count INT;
BEGIN
    -- Resolve year metadata
    SELECT ay.tenant_id, ay.school_id
    INTO v_tenant_id, v_school_id
    FROM academic_years ay
    WHERE ay.id = target_year_id;

    IF NOT FOUND THEN
        RETURN;
    END IF;

    -- Count total non-break periods per week in the school's timetable
    SELECT COUNT(*)::INT INTO v_total_school_periods
    FROM timetable_structures ts
    WHERE ts.tenant_id = v_tenant_id
      AND ts.school_id = v_school_id
      AND ts.academic_year_id = target_year_id
      AND ts.is_break = false;

    -- Count active teachers (users with timetable slots)
    SELECT COUNT(DISTINCT ts.teacher_id)::INT INTO v_active_teacher_count
    FROM cbc_timetable_slots ts
    WHERE ts.tenant_id = v_tenant_id
      AND ts.school_id = v_school_id
      AND ts.academic_year_id = target_year_id
      AND ts.teacher_id IS NOT NULL;

    -- Compute and upsert workload summaries per teacher
    INSERT INTO teacher_workload_summaries (
        tenant_id, school_id, user_id, academic_year_id,
        total_assigned_periods, unique_subjects, classes_taught,
        utilization_percentage, is_overcapacity, last_refreshed_at
    )
    WITH
    teacher_metrics AS (
        SELECT
            ts.teacher_id AS user_id,
            COUNT(*)::INT AS assigned_periods,
            COUNT(DISTINCT ts.learning_area_id)::INT AS subjects_count,
            COUNT(DISTINCT ts.class_id)::INT AS classes_count
        FROM cbc_timetable_slots ts
        JOIN timetable_structures tstr ON tstr.id = ts.structure_id
        WHERE ts.tenant_id = v_tenant_id
          AND ts.school_id = v_school_id
          AND ts.academic_year_id = target_year_id
          AND tstr.is_break = false
        GROUP BY ts.teacher_id
    )
    SELECT
        v_tenant_id,
        v_school_id,
        tm.user_id,
        target_year_id,
        tm.assigned_periods,
        tm.subjects_count,
        tm.classes_count,
        CASE
            WHEN v_total_school_periods > 0
            THEN ROUND(
                (tm.assigned_periods::NUMERIC / v_total_school_periods::NUMERIC) * 100,
                2
            )
            ELSE NULL
        END,
        CASE
            WHEN v_total_school_periods > 0
            THEN tm.assigned_periods > v_total_school_periods
            ELSE FALSE
        END,
        NOW()
    FROM teacher_metrics tm

    ON CONFLICT (user_id, academic_year_id)
    DO UPDATE SET
        total_assigned_periods  = EXCLUDED.total_assigned_periods,
        unique_subjects         = EXCLUDED.unique_subjects,
        classes_taught          = EXCLUDED.classes_taught,
        utilization_percentage  = EXCLUDED.utilization_percentage,
        is_overcapacity         = EXCLUDED.is_overcapacity,
        last_refreshed_at       = NOW(),
        updated_at              = NOW();

    -- Clean up orphaned rows (teacher no longer has slots or is deactivated)
    DELETE FROM teacher_workload_summaries
    WHERE academic_year_id = target_year_id
      AND tenant_id = v_tenant_id
      AND school_id = v_school_id
      AND user_id NOT IN (
          SELECT DISTINCT ts.teacher_id
          FROM cbc_timetable_slots ts
          WHERE ts.tenant_id = v_tenant_id
            AND ts.school_id = v_school_id
            AND ts.academic_year_id = target_year_id
            AND ts.teacher_id IS NOT NULL
      );
END;
$$;

COMMENT ON FUNCTION fn_compute_teacher_workload_summaries IS
    'Batch-computes teacher_workload_summaries for all teachers with timetable
     slots in the given academic year. Uses cbc_timetable_slots and
     timetable_structures for workload metrics.';

-- ============================================================================
-- RLS POLICY
-- ============================================================================

ALTER TABLE IF EXISTS teacher_workload_summaries ENABLE ROW LEVEL SECURITY;

DO $$ BEGIN
    DROP POLICY IF EXISTS tenant_isolation_policy ON teacher_workload_summaries;
    CREATE POLICY tenant_isolation_policy ON teacher_workload_summaries
        FOR ALL
        USING (tenant_id = fn_current_tenant_id())
        WITH CHECK (tenant_id = fn_current_tenant_id());
END $$;

COMMENT ON TABLE teacher_workload_summaries IS
    'Batch-computed summary of teacher workload metrics per academic year.
     RLS-enabled — tenant-scoped.';

-- Migration: 000016_create_class_attendance_rollups
-- Creates two class-grain attendance rollup tables, refreshed exclusively
-- by Asynq background jobs (no cascading DB triggers fire on high-frequency
-- attendance_records writes — the hot write path is untouched):
--
--   1. class_learning_area_term_summaries
--      Per (class_id, learning_area_id, academic_term_id) — rolls up the
--      student-grain attendance_term_summaries into class-grain so admin
--      and teacher reports can answer "which subjects have the worst
--      attendance for this class this term" without aggregating across
--      every individual student row at report-render time.
--
--   2. class_term_attendance_summaries
--      Per (class_id, academic_term_id) — rolls up the day-grain
--      class_daily_attendance_summaries into term-grain so admin reports
--      can answer "what was class X's attendance rate for the whole term"
--      without summing 60-90 daily rows at report-render time.
--
-- Both tables are populated exclusively by Asynq tasks
-- `attendance:refresh_class_learning_area_term_summary` and
-- `attendance:refresh_class_term_summary` (defined in
-- backend/internal/attendance/worker.go). These jobs are enqueued from
-- inside the upstream refresh handlers (handleAttendanceTermRefresh and
-- handleClassDailyRefresh) so the rollup runs *after* its source table is
-- up-to-date, not on an independent timer.

-- ============================================================================
-- TABLE 1: class_learning_area_term_summaries
-- ============================================================================

CREATE TABLE IF NOT EXISTS class_learning_area_term_summaries (
    id                    UUID          PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id             UUID          NOT NULL,
    school_id             UUID          NOT NULL,
    class_id              UUID          NOT NULL,
    learning_area_id      UUID          NOT NULL,
    academic_term_id      UUID          NOT NULL,
    academic_year_id      UUID          NOT NULL,
    students_included     INT           NOT NULL DEFAULT 0,
    periods_total         INT           NOT NULL DEFAULT 0,
    periods_present       INT           NOT NULL DEFAULT 0,
    periods_absent        INT           NOT NULL DEFAULT 0,
    periods_late          INT           NOT NULL DEFAULT 0,
    periods_excused       INT           NOT NULL DEFAULT 0,
    attendance_percentage NUMERIC(5,2)  NOT NULL DEFAULT 0.00,
    last_refreshed_at     TIMESTAMPTZ   NOT NULL DEFAULT NOW(),
    created_at            TIMESTAMPTZ   NOT NULL DEFAULT NOW(),
    updated_at            TIMESTAMPTZ   NOT NULL DEFAULT NOW(),

    CONSTRAINT uq_class_learning_area_term
        UNIQUE (class_id, learning_area_id, academic_term_id),
    CONSTRAINT fk_class_la_term_tenant_school
        FOREIGN KEY (tenant_id, school_id)
        REFERENCES cbc_schools(tenant_id, id) ON DELETE CASCADE,
    CONSTRAINT fk_class_la_term_tenant_class
        FOREIGN KEY (tenant_id, class_id)
        REFERENCES cbc_classes(tenant_id, id) ON DELETE CASCADE,
    CONSTRAINT fk_class_la_term_learning_area
        FOREIGN KEY (learning_area_id)
        REFERENCES cbc_learning_areas(id) ON DELETE CASCADE,
    CONSTRAINT fk_class_la_term_tenant_term
        FOREIGN KEY (tenant_id, school_id, academic_term_id)
        REFERENCES academic_terms(tenant_id, school_id, id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_class_la_term_tenant
    ON class_learning_area_term_summaries (tenant_id);
CREATE INDEX IF NOT EXISTS idx_class_la_term_school
    ON class_learning_area_term_summaries (school_id);
CREATE INDEX IF NOT EXISTS idx_class_la_term_class_term
    ON class_learning_area_term_summaries (class_id, academic_term_id);
CREATE INDEX IF NOT EXISTS idx_class_la_term_area_term
    ON class_learning_area_term_summaries (learning_area_id, academic_term_id);
CREATE INDEX IF NOT EXISTS idx_class_la_term_year
    ON class_learning_area_term_summaries (academic_year_id);

DROP TRIGGER IF EXISTS trg_class_learning_area_term_summaries_updated_at
    ON class_learning_area_term_summaries;
CREATE TRIGGER trg_class_learning_area_term_summaries_updated_at
    BEFORE UPDATE ON class_learning_area_term_summaries
    FOR EACH ROW EXECUTE FUNCTION fn_set_updated_at();

COMMENT ON TABLE class_learning_area_term_summaries IS
    'Class-grain rollup of attendance_term_summaries per (class, learning_area,
     academic_term). Refreshed exclusively by the Asynq task
     attendance:refresh_class_learning_area_term_summary (enqueued from
     inside handleAttendanceTermRefresh so it runs after the student-grain
     rollup is current). No cascading DB triggers fire on
     attendance_term_summaries writes — this keeps the hot attendance
     marking path predictable.';

COMMENT ON COLUMN class_learning_area_term_summaries.students_included IS
    'Count of distinct students whose attendance_term_summaries rows
     contributed to this (class, learning_area, term) aggregate. May be less
     than the total enrolled count for the class because not every enrolled
     student necessarily has an attendance_term_summaries row for every
     subject (e.g. subject not yet taught, no periods in the term).';

COMMENT ON COLUMN class_learning_area_term_summaries.attendance_percentage IS
    'Calculated as (periods_present / periods_total) * 100, stored as a
     decimal with two fractional digits (e.g. 92.50). Matches the formula
     used in attendance_term_summaries.attendance_percentage.';

COMMENT ON COLUMN class_learning_area_term_summaries.last_refreshed_at IS
    'Timestamp of the most recent successful Asynq refresh of this row.
     Report generators must surface this value so consumers can flag
     "data as of X" — it is NOT refreshed automatically by DB triggers.';

-- ============================================================================
-- TABLE 2: class_term_attendance_summaries
-- ============================================================================

CREATE TABLE IF NOT EXISTS class_term_attendance_summaries (
    id                   UUID          PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id            UUID          NOT NULL,
    school_id            UUID          NOT NULL,
    class_id             UUID          NOT NULL,
    academic_term_id     UUID          NOT NULL,
    academic_year_id     UUID          NOT NULL,
    days_in_term         INT           NOT NULL DEFAULT 0,
    total_enrolled_avg   NUMERIC(6,2)  NULL,
    present_count        INT           NOT NULL DEFAULT 0,
    absent_count         INT           NOT NULL DEFAULT 0,
    late_count           INT           NOT NULL DEFAULT 0,
    excused_count        INT           NOT NULL DEFAULT 0,
    term_attendance_rate NUMERIC(5,2)  NOT NULL DEFAULT 0.00,
    last_refreshed_at    TIMESTAMPTZ   NOT NULL DEFAULT NOW(),
    created_at           TIMESTAMPTZ   NOT NULL DEFAULT NOW(),
    updated_at           TIMESTAMPTZ   NOT NULL DEFAULT NOW(),

    CONSTRAINT uq_class_term_attendance UNIQUE (class_id, academic_term_id),
    CONSTRAINT fk_class_term_att_tenant_school
        FOREIGN KEY (tenant_id, school_id)
        REFERENCES cbc_schools(tenant_id, id) ON DELETE CASCADE,
    CONSTRAINT fk_class_term_att_tenant_class
        FOREIGN KEY (tenant_id, class_id)
        REFERENCES cbc_classes(tenant_id, id) ON DELETE CASCADE,
    CONSTRAINT fk_class_term_att_tenant_term
        FOREIGN KEY (tenant_id, school_id, academic_term_id)
        REFERENCES academic_terms(tenant_id, school_id, id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_class_term_att_tenant
    ON class_term_attendance_summaries (tenant_id);
CREATE INDEX IF NOT EXISTS idx_class_term_att_school_term
    ON class_term_attendance_summaries (school_id, academic_term_id);
CREATE INDEX IF NOT EXISTS idx_class_term_att_class
    ON class_term_attendance_summaries (class_id);
CREATE INDEX IF NOT EXISTS idx_class_term_att_term
    ON class_term_attendance_summaries (academic_term_id);
CREATE INDEX IF NOT EXISTS idx_class_term_att_year
    ON class_term_attendance_summaries (academic_year_id);

DROP TRIGGER IF EXISTS trg_class_term_attendance_summaries_updated_at
    ON class_term_attendance_summaries;
CREATE TRIGGER trg_class_term_attendance_summaries_updated_at
    BEFORE UPDATE ON class_term_attendance_summaries
    FOR EACH ROW EXECUTE FUNCTION fn_set_updated_at();

COMMENT ON TABLE class_term_attendance_summaries IS
    'Term-grain rollup of class_daily_attendance_summaries per
     (class, academic_term). Refreshed exclusively by the Asynq task
     attendance:refresh_class_term_summary (enqueued from inside
     handleClassDailyRefresh so it runs after the daily-grain rollup is
     current). No cascading DB triggers fire on
     class_daily_attendance_summaries writes.';

COMMENT ON COLUMN class_term_attendance_summaries.days_in_term IS
    'Count of class_daily_attendance_summaries rows rolled up for this
     class/term — i.e. school days with recorded attendance, NOT calendar
     days or total days in the term date range.';

COMMENT ON COLUMN class_term_attendance_summaries.total_enrolled_avg IS
    'Average of class_daily_attendance_summaries.total_enrolled across the
     term. Enrollment fluctuates day to day; we inherit the documented
     workaround from the daily table (total_enrolled is derived from
     distinct students with attendance_records that day, not from
     cbc_student_enrollments.status, because enrollment status has no
     effective date within a term). This rollup does NOT attempt to fix
     that limitation — it preserves the same per-day enrollment snapshot
     as the source table.';

COMMENT ON COLUMN class_term_attendance_summaries.term_attendance_rate IS
    'Calculated as (present_count / (present_count + absent_count +
     late_count + excused_count)) * 100, matching the formula used in
     class_daily_attendance_summaries.daily_attendance_rate. Stored as a
     decimal with two fractional digits.';

COMMENT ON COLUMN class_term_attendance_summaries.last_refreshed_at IS
    'Timestamp of the most recent successful Asynq refresh of this row.
     Report generators must surface this value so consumers can flag
     "data as of X" — it is NOT refreshed automatically by DB triggers.';

-- ============================================================================
-- RLS POLICIES
-- ============================================================================

ALTER TABLE IF EXISTS class_learning_area_term_summaries ENABLE ROW LEVEL SECURITY;

DO $$ BEGIN
    DROP POLICY IF EXISTS tenant_isolation_policy ON class_learning_area_term_summaries;
    CREATE POLICY tenant_isolation_policy ON class_learning_area_term_summaries
        FOR ALL
        USING (tenant_id = fn_current_tenant_id())
        WITH CHECK (tenant_id = fn_current_tenant_id());
END $$;

COMMENT ON TABLE class_learning_area_term_summaries IS
    'Class-grain rollup of attendance_term_summaries per (class, learning_area,
     academic_term). RLS-enabled — tenant-scoped.';

ALTER TABLE IF EXISTS class_term_attendance_summaries ENABLE ROW LEVEL SECURITY;

DO $$ BEGIN
    DROP POLICY IF EXISTS tenant_isolation_policy ON class_term_attendance_summaries;
    CREATE POLICY tenant_isolation_policy ON class_term_attendance_summaries
        FOR ALL
        USING (tenant_id = fn_current_tenant_id())
        WITH CHECK (tenant_id = fn_current_tenant_id());
END $$;

COMMENT ON TABLE class_term_attendance_summaries IS
    'Term-grain rollup of class_daily_attendance_summaries per
     (class, academic_term). RLS-enabled — tenant-scoped.';

-- ============================================================================
-- AFTER-PUBLISH REFRESH TRIGGER (assessment_sessions)
-- Fires AFTER UPDATE on assessment_sessions when status changes to PUBLISHED.
-- Refreshes both the term-level subject summaries (000006) and the
-- sub-strand-level rubric summaries (000009).
-- ============================================================================

CREATE OR REPLACE FUNCTION fn_assessment_sessions_after_publish()
RETURNS TRIGGER
LANGUAGE plpgsql
AS $$
BEGIN
    IF NEW.status = 'PUBLISHED' AND (OLD.status IS DISTINCT FROM 'PUBLISHED') THEN
        -- Refresh term-level subject summaries (existing)
        PERFORM fn_refresh_term_subject_summary_for_session(NEW.id);

        -- Refresh sub-strand-level summaries (new, rubric-only)
        PERFORM fn_refresh_subject_strand_summary_for_session(NEW.id);
    END IF;
    RETURN NEW;
END;
$$;

DROP TRIGGER IF EXISTS trg_assessment_sessions_refresh_summary
    ON assessment_sessions;
CREATE TRIGGER trg_assessment_sessions_refresh_summary
    AFTER UPDATE OF status ON assessment_sessions
    FOR EACH ROW
    EXECUTE FUNCTION fn_assessment_sessions_after_publish();

COMMENT ON TRIGGER trg_assessment_sessions_refresh_summary ON assessment_sessions IS
    'After an assessment session is published, refresh the term-level subject
     summaries and the rubric sub-strand summaries for all students in that
     session.';

-- ============================================================================
-- END OF MIGRATION
-- ============================================================================