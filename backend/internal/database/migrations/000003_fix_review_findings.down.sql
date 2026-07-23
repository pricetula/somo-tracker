-- Migration: 000003_fix_review_findings (rollback)
-- SomoTracker — Kenya CBC/CBE academic platform
-- Reverses every change from the up migration in strict reverse order.
-- Restores the exact prior state as of 000001_initial_schema.

-- ============================================================================
-- ITEM 9 — Restore global, case-sensitive unique email index
-- ============================================================================

DROP INDEX IF EXISTS idx_users_tenant_email;
CREATE UNIQUE INDEX IF NOT EXISTS idx_users_email ON users (email);

-- ============================================================================
-- ITEM 8 — Drop hashed-token columns (sessions, invitations)
-- ============================================================================

DROP INDEX IF EXISTS idx_sessions_token_hash;
ALTER TABLE IF EXISTS sessions DROP COLUMN IF EXISTS token_hash;

DROP INDEX IF EXISTS idx_invitations_token_hash;
ALTER TABLE IF EXISTS invitations DROP COLUMN IF EXISTS token_hash;

-- Restore original comments (remove deprecation notes)
COMMENT ON COLUMN sessions.token IS NULL;
COMMENT ON COLUMN invitations.token IS NULL;

-- Restore original stytch_session_token comment
COMMENT ON COLUMN sessions.stytch_session_token IS NULL;

-- ============================================================================
-- ITEM 7 — Revert free-text columns back from enums to VARCHAR
-- ============================================================================

-- ---------------------------------------------------------------
-- 7b: cbc_student_parents.relationship — revert from enum to VARCHAR(50)
-- ---------------------------------------------------------------

ALTER TABLE IF EXISTS cbc_student_parents
    ADD COLUMN IF NOT EXISTS relationship_old VARCHAR(50);

UPDATE cbc_student_parents
SET relationship_old = CASE
    WHEN relationship = 'FATHER'   THEN 'Father'
    WHEN relationship = 'MOTHER'   THEN 'Mother'
    WHEN relationship = 'GUARDIAN' THEN 'Guardian'
    ELSE NULL
END;

ALTER TABLE IF EXISTS cbc_student_parents
    DROP COLUMN IF EXISTS relationship;

ALTER TABLE IF EXISTS cbc_student_parents
    RENAME COLUMN relationship_old TO relationship;

COMMENT ON COLUMN cbc_student_parents.relationship IS NULL;

-- Drop the enum (only if no other columns reference it)
DROP TYPE IF EXISTS parent_relationship_type;

-- ---------------------------------------------------------------
-- 7a: payments.payment_method — revert from enum to VARCHAR(50)
-- ---------------------------------------------------------------

ALTER TABLE IF EXISTS payments
    ADD COLUMN IF NOT EXISTS payment_method_old VARCHAR(50);

UPDATE payments
SET payment_method_old = CASE
    WHEN payment_method = 'MPESA'         THEN 'MPESA'
    WHEN payment_method = 'CASH'          THEN 'CASH'
    WHEN payment_method = 'BANK_TRANSFER' THEN 'BANK_TRANSFER'
    WHEN payment_method = 'CHEQUE'        THEN 'CHEQUE'
    ELSE NULL
END;

ALTER TABLE IF EXISTS payments
    DROP COLUMN IF EXISTS payment_method;

ALTER TABLE IF EXISTS payments
    RENAME COLUMN payment_method_old TO payment_method;

COMMENT ON COLUMN payments.payment_method IS NULL;

-- Drop the enum (only if no other columns reference it)
DROP TYPE IF EXISTS payment_method_type;

-- ============================================================================
-- ITEM 6 — Drop immutability triggers and their functions
-- ============================================================================

DROP TRIGGER IF EXISTS trg_grading_scale_ranges_immutable ON grading_scale_ranges;
DROP FUNCTION IF EXISTS fn_block_grading_scale_range_modification();

DROP TRIGGER IF EXISTS trg_assessment_sessions_max_points_immutable ON assessment_sessions;
DROP FUNCTION IF EXISTS fn_block_assessment_max_points_update();

-- ============================================================================
-- ITEM 5 — Drop "one primary parent per student" unique index
-- ============================================================================

DROP INDEX IF EXISTS idx_one_primary_parent_per_student;

-- ============================================================================
-- ITEM 4 — Remove tenant scoping from curriculum leaf tables
-- ============================================================================

-- ---------------------------------------------------------------
-- 4c: performance_indicators
-- ---------------------------------------------------------------

DROP POLICY IF EXISTS tenant_isolation_policy ON performance_indicators;
ALTER TABLE IF EXISTS performance_indicators DISABLE ROW LEVEL SECURITY;
ALTER TABLE IF EXISTS performance_indicators NO FORCE ROW LEVEL SECURITY;

ALTER TABLE IF EXISTS performance_indicators
    DROP CONSTRAINT IF EXISTS fk_performance_indicators_tenant_sub_strand;

DROP INDEX IF EXISTS idx_performance_indicators_tenant;
ALTER TABLE IF EXISTS performance_indicators DROP COLUMN IF EXISTS tenant_id;

-- ---------------------------------------------------------------
-- 4b: cbc_sub_strands
-- ---------------------------------------------------------------

DROP POLICY IF EXISTS tenant_isolation_policy ON cbc_sub_strands;
ALTER TABLE IF EXISTS cbc_sub_strands DISABLE ROW LEVEL SECURITY;
ALTER TABLE IF EXISTS cbc_sub_strands NO FORCE ROW LEVEL SECURITY;

ALTER TABLE IF EXISTS cbc_sub_strands
    DROP CONSTRAINT IF EXISTS fk_cbc_sub_strands_tenant_strand;

ALTER TABLE IF EXISTS cbc_sub_strands
    DROP CONSTRAINT IF EXISTS uq_cbc_sub_strands_tenant;

DROP INDEX IF EXISTS idx_cbc_sub_strands_tenant;
ALTER TABLE IF EXISTS cbc_sub_strands DROP COLUMN IF EXISTS tenant_id;

-- ---------------------------------------------------------------
-- 4a: cbc_strands
-- ---------------------------------------------------------------

DROP POLICY IF EXISTS tenant_isolation_policy ON cbc_strands;
ALTER TABLE IF EXISTS cbc_strands DISABLE ROW LEVEL SECURITY;
ALTER TABLE IF EXISTS cbc_strands NO FORCE ROW LEVEL SECURITY;

ALTER TABLE IF EXISTS cbc_strands
    DROP CONSTRAINT IF EXISTS fk_cbc_strands_tenant_learning_area;

ALTER TABLE IF EXISTS cbc_strands
    DROP CONSTRAINT IF EXISTS uq_cbc_strands_tenant;

DROP INDEX IF EXISTS idx_cbc_strands_tenant;
ALTER TABLE IF EXISTS cbc_strands DROP COLUMN IF EXISTS tenant_id;

-- ============================================================================
-- ITEM 3 — Restore single-column FKs on health/student tables
-- ============================================================================

-- ---------------------------------------------------------------
-- 3c: student_health_profiles — drop composite FK, restore single-column FK
-- ---------------------------------------------------------------

ALTER TABLE IF EXISTS student_health_profiles
    DROP CONSTRAINT IF EXISTS fk_student_health_profiles_tenant_student;

DO $$
BEGIN
    ALTER TABLE IF EXISTS student_health_profiles
        ADD CONSTRAINT student_health_profiles_student_id_fkey
        FOREIGN KEY (student_id)
        REFERENCES cbc_students(id)
        ON DELETE CASCADE;
EXCEPTION WHEN duplicate_object THEN NULL;
END $$;

-- ---------------------------------------------------------------
-- 3b: medical_incidents — drop composite FK, restore single-column FK
-- ---------------------------------------------------------------

ALTER TABLE IF EXISTS medical_incidents
    DROP CONSTRAINT IF EXISTS fk_medical_incidents_tenant_student;

DO $$
BEGIN
    ALTER TABLE IF EXISTS medical_incidents
        ADD CONSTRAINT medical_incidents_student_id_fkey
        FOREIGN KEY (student_id)
        REFERENCES cbc_students(id)
        ON DELETE CASCADE;
EXCEPTION WHEN duplicate_object THEN NULL;
END $$;

-- ---------------------------------------------------------------
-- 3a: cbc_students — drop composite FK, restore single-column FK
-- ---------------------------------------------------------------

ALTER TABLE IF EXISTS cbc_students
    DROP CONSTRAINT IF EXISTS fk_cbc_students_tenant_school;

DO $$
BEGIN
    ALTER TABLE IF EXISTS cbc_students
        ADD CONSTRAINT cbc_students_school_id_fkey
        FOREIGN KEY (school_id)
        REFERENCES cbc_schools(id)
        ON DELETE CASCADE;
EXCEPTION WHEN duplicate_object THEN NULL;
END $$;

-- ============================================================================
-- ITEM 2 — Remove tenant scoping from grading_scale_ranges
-- ============================================================================

DROP POLICY IF EXISTS tenant_isolation_policy ON grading_scale_ranges;

ALTER TABLE IF EXISTS grading_scale_ranges
    DROP CONSTRAINT IF EXISTS fk_grading_scale_ranges_tenant_profile;

DROP INDEX IF EXISTS idx_grading_scale_ranges_tenant;
ALTER TABLE IF EXISTS grading_scale_ranges DROP COLUMN IF EXISTS tenant_id;

-- ============================================================================
-- ITEM 1 — Restore old (broken) chk_score_range constraint
-- ============================================================================

ALTER TABLE IF EXISTS student_assessment_scores
    DROP CONSTRAINT IF EXISTS chk_score_range;

-- Restore the original OR-based constraint (intentionally broken — this is the rollback)
ALTER TABLE IF EXISTS student_assessment_scores
    ADD CONSTRAINT chk_score_range CHECK (
        raw_score IS NULL OR max_points_check(session_id, raw_score) OR raw_score >= 0
    );

COMMENT ON CONSTRAINT chk_score_range ON student_assessment_scores IS NULL;

-- ============================================================================
-- END OF MIGRATION
-- ============================================================================
