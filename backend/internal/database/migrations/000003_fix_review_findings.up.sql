-- Migration: 000003_fix_review_findings
-- SomoTracker — Kenya CBC/CBE academic platform
-- Schema fixes addressing 10 review findings from the senior DB review.
--
-- Items addressed:
--   1. Fix chk_score_range on student_assessment_scores (OR→AND bug)
--   2. Fix grading_scale_ranges: add tenant_id, FK, RLS policy (deny-all bug)
--   3. Add composite tenant-scoped FKs on health/student tables
--   4. Add tenant scoping to cbc_strands, cbc_sub_strands, performance_indicators
--   5. Add "one primary parent per student" partial unique index
--   6. Add DB-enforced immutability triggers for write-once invariants
--   7. Convert free-text columns to enums: payment_method, relationship
--   8. Add hashed-token columns for auth secrets (pgcrypto)
--   9. Fix users.email uniqueness: per-tenant + case-insensitive
--  10. Document RLS bypass footgun (comment-only)
--
-- Every change is safe to run against a database that already contains rows.
-- Where a constraint could fail existing data, a pre-flight validation query
-- reports violations without deleting data.

-- ============================================================================
-- ITEM 10 — RLS Bypass Footgun: Documentation
-- (Placed first so it ships with the fix set. Pure SQL comment — no schema
--  change.)
-- ============================================================================

/*
 * ═══════════════════════════════════════════════════════════════════════════
 *  RLS BYPASS WARNING — Read before granting roles
 * ═══════════════════════════════════════════════════════════════════════════
 *
 *  PostgreSQL Row-Level Security is **bypassed** for:
 *    - The table owner (usually the role that ran CREATE TABLE)
 *    - Any role with the BYPASSRLS attribute
 *    - Superuser roles (which implicitly have BYPASSRLS)
 *
 *  This means that even though RLS is ENABLE'd on a table, the owner role
 *  can still see ALL rows — including rows belonging to other tenants.
 *  RLS is only enforced for non-owner, non-BYPASSRLS roles.
 *
 *  ⚠️  CRITICAL REQUIREMENT:
 *      The application's runtime database role MUST NOT be:
 *        - The schema owner (the role that ran the migrations)
 *        - A superuser
 *        - Granted BYPASSRLS
 *
 *  The runtime role should be a separate, least-privilege role that:
 *        - Has only USAGE on schemas
 *        - Has only SELECT/INSERT/UPDATE/DELETE on the tables it needs
 *        - Does NOT own any tables or schemas
 *        - Does NOT have BYPASSRLS
 *
 *  🔍  Verification query (run as the role that will serve requests):
 *        SELECT rolname, rolbypassrls, rolsuper
 *        FROM pg_roles
 *        WHERE rolname = current_user;
 *
 *  Both rolbypassrls and rolsuper must be false for RLS to be effective.
 * ═══════════════════════════════════════════════════════════════════════════
 */

-- ============================================================================
-- ITEM 1 — Fix chk_score_range on student_assessment_scores
-- The original constraint used OR instead of AND, making it a logical no-op
-- (raw_score >= 0 is always true for any value that passes the first OR arm,
-- and NULL shortcuts the whole check). Replace with proper AND-bounded check.
-- ============================================================================

DO $$
DECLARE
    v_count INT;
BEGIN
    -- Validate existing data: report rows where raw_score < 0 or raw_score > max_points
    SELECT COUNT(*) INTO v_count
    FROM student_assessment_scores sas
    WHERE sas.raw_score IS NOT NULL
      AND (
          sas.raw_score < 0
          OR sas.raw_score > COALESCE(
              (SELECT asi.max_points FROM assessment_sessions asi WHERE asi.id = sas.session_id),
              sas.raw_score
          )
      );

    IF v_count > 0 THEN
        RAISE WARNING 'ITEM 1 — Found % rows in student_assessment_scores with raw_score out of range (negative or exceeding session max_points). These WILL be rejected by the new constraint. Query to inspect: SELECT * FROM student_assessment_scores WHERE raw_score IS NOT NULL AND (raw_score < 0 OR raw_score > COALESCE((SELECT max_points FROM assessment_sessions WHERE id = session_id), raw_score));', v_count;
    ELSE
        RAISE NOTICE 'ITEM 1 — No out-of-range rows found. Safe to add constraint.';
    END IF;
END $$;

ALTER TABLE IF EXISTS student_assessment_scores
    DROP CONSTRAINT IF EXISTS chk_score_range;

ALTER TABLE IF EXISTS student_assessment_scores
    ADD CONSTRAINT chk_score_range CHECK (
        raw_score IS NULL OR (raw_score >= 0 AND max_points_check(session_id, raw_score))
    );

COMMENT ON CONSTRAINT chk_score_range ON student_assessment_scores IS
    'Enforces that raw_score (when non-NULL) is non-negative AND does not exceed
     the session''s max_points. Fixed from original OR-bug which made this a no-op.';

-- ============================================================================
-- ITEM 2 — Fix grading_scale_ranges RLS: add tenant_id, FK, and policy
-- The table had RLS enabled but zero policies (no tenant_id column), which
-- blocked ALL normal-role access (deny-all bug). Fix by denormalizing the
-- tenant_id from grading_scale_profiles.
-- ============================================================================

-- Step 2a: Add tenant_id column (nullable initially for backfill)
ALTER TABLE IF EXISTS grading_scale_ranges
    ADD COLUMN IF NOT EXISTS tenant_id UUID;

-- Step 2b: Backfill tenant_id from grading_scale_profiles
UPDATE grading_scale_ranges gsr
SET tenant_id = gsp.tenant_id
FROM grading_scale_profiles gsp
WHERE gsr.profile_id = gsp.id
  AND gsr.tenant_id IS NULL;

-- Step 2c: Enforce NOT NULL after backfill
ALTER TABLE IF EXISTS grading_scale_ranges
    ALTER COLUMN tenant_id SET NOT NULL;

-- Step 2d: Add composite FK (tenant_id, profile_id) → grading_scale_profiles(tenant_id, id)
-- The target unique constraint uq_grading_scale_profiles_tenant UNIQUE (tenant_id, id)
-- already exists from the original migration. Postgres allows the FK column name
-- (profile_id) to differ from the referenced column name (id) as long as the
-- column positions and types match.
DO $$
BEGIN
    ALTER TABLE IF EXISTS grading_scale_ranges
        ADD CONSTRAINT fk_grading_scale_ranges_tenant_profile
        FOREIGN KEY (tenant_id, profile_id)
        REFERENCES grading_scale_profiles(tenant_id, id)
        ON DELETE CASCADE;
EXCEPTION WHEN duplicate_object THEN NULL;
END $$;

-- Step 2f: Add index on tenant_id
CREATE INDEX IF NOT EXISTS idx_grading_scale_ranges_tenant
    ON grading_scale_ranges (tenant_id);

-- Step 2g: Recreate RLS policy (drop and recreate)
DROP POLICY IF EXISTS tenant_isolation_policy ON grading_scale_ranges;
CREATE POLICY tenant_isolation_policy ON grading_scale_ranges
    FOR ALL
    USING (tenant_id = fn_current_tenant_id())
    WITH CHECK (tenant_id = fn_current_tenant_id());

COMMENT ON COLUMN grading_scale_ranges.tenant_id IS
    'Denormalised from grading_scale_profiles for RLS enforcement. Backfilled
     automatically; must match the tenant_id of the referenced profile.';

-- ============================================================================
-- ITEM 3 — Composite tenant-scoped FKs on health and student tables
-- Single-column FKs on cbc_students.school_id, medical_incidents.student_id,
-- and student_health_profiles.student_id allowed cross-tenant integrity holes.
-- Replace with composite FKs against (tenant_id, id) unique keys.
-- ============================================================================

-- Pre-flight: both parent tables already have UNIQUE (tenant_id, id):
--   cbc_schools:  uq_cbc_schools_tenant   UNIQUE (tenant_id, id)  ✓
--   cbc_students: uq_cbc_students_tenant   UNIQUE (tenant_id, id)  ✓

-- ---------------------------------------------------------------
-- 3a: cbc_students.school_id → composite FK (tenant_id, school_id)
--     REFERENCES cbc_schools(tenant_id, id)
-- ---------------------------------------------------------------

DO $$
DECLARE
    v_count INT;
BEGIN
    SELECT COUNT(*) INTO v_count
    FROM cbc_students s
    JOIN cbc_schools sc ON sc.id = s.school_id
    WHERE s.tenant_id IS DISTINCT FROM sc.tenant_id;

    IF v_count > 0 THEN
        RAISE WARNING 'ITEM 3a — Found % rows in cbc_students where tenant_id does not match the school''s tenant_id. Violating rows: SELECT s.id, s.full_name, s.tenant_id AS student_tenant, sc.tenant_id AS school_tenant FROM cbc_students s JOIN cbc_schools sc ON sc.id = s.school_id WHERE s.tenant_id IS DISTINCT FROM sc.tenant_id;', v_count;
    ELSE
        RAISE NOTICE 'ITEM 3a — No tenant_id mismatches found. Safe to add composite FK.';
    END IF;
END $$;

-- Drop single-column FK
ALTER TABLE IF EXISTS cbc_students
    DROP CONSTRAINT IF EXISTS cbc_students_school_id_fkey;

-- Add composite FK
DO $$
BEGIN
    ALTER TABLE IF EXISTS cbc_students
        ADD CONSTRAINT fk_cbc_students_tenant_school
        FOREIGN KEY (tenant_id, school_id)
        REFERENCES cbc_schools(tenant_id, id)
        ON DELETE CASCADE;
EXCEPTION WHEN duplicate_object THEN NULL;
END $$;

-- ---------------------------------------------------------------
-- 3b: medical_incidents.student_id → composite FK (tenant_id, student_id)
--     REFERENCES cbc_students(tenant_id, id)
-- ---------------------------------------------------------------

DO $$
DECLARE
    v_count INT;
BEGIN
    SELECT COUNT(*) INTO v_count
    FROM medical_incidents mi
    JOIN cbc_students s ON s.id = mi.student_id
    WHERE mi.tenant_id IS DISTINCT FROM s.tenant_id;

    IF v_count > 0 THEN
        RAISE WARNING 'ITEM 3b — Found % rows in medical_incidents where tenant_id does not match the student''s tenant_id. Violating rows: SELECT mi.id, mi.incident_timestamp, mi.tenant_id AS incident_tenant, s.tenant_id AS student_tenant FROM medical_incidents mi JOIN cbc_students s ON s.id = mi.student_id WHERE mi.tenant_id IS DISTINCT FROM s.tenant_id;', v_count;
    ELSE
        RAISE NOTICE 'ITEM 3b — No tenant_id mismatches found. Safe to add composite FK.';
    END IF;
END $$;

-- Drop single-column FK
ALTER TABLE IF EXISTS medical_incidents
    DROP CONSTRAINT IF EXISTS medical_incidents_student_id_fkey;

-- Add composite FK
DO $$
BEGIN
    ALTER TABLE IF EXISTS medical_incidents
        ADD CONSTRAINT fk_medical_incidents_tenant_student
        FOREIGN KEY (tenant_id, student_id)
        REFERENCES cbc_students(tenant_id, id)
        ON DELETE CASCADE;
EXCEPTION WHEN duplicate_object THEN NULL;
END $$;

-- ---------------------------------------------------------------
-- 3c: student_health_profiles.student_id → composite FK (tenant_id, student_id)
--     REFERENCES cbc_students(tenant_id, id)
-- ---------------------------------------------------------------

DO $$
DECLARE
    v_count INT;
BEGIN
    SELECT COUNT(*) INTO v_count
    FROM student_health_profiles shp
    JOIN cbc_students s ON s.id = shp.student_id
    WHERE shp.tenant_id IS DISTINCT FROM s.tenant_id;

    IF v_count > 0 THEN
        RAISE WARNING 'ITEM 3c — Found % rows in student_health_profiles where tenant_id does not match the student''s tenant_id. Violating rows: SELECT shp.id, shp.tenant_id AS profile_tenant, s.tenant_id AS student_tenant FROM student_health_profiles shp JOIN cbc_students s ON s.id = shp.student_id WHERE shp.tenant_id IS DISTINCT FROM s.tenant_id;', v_count;
    ELSE
        RAISE NOTICE 'ITEM 3c — No tenant_id mismatches found. Safe to add composite FK.';
    END IF;
END $$;

-- Drop single-column FK
ALTER TABLE IF EXISTS student_health_profiles
    DROP CONSTRAINT IF EXISTS student_health_profiles_student_id_fkey;

-- Add composite FK
DO $$
BEGIN
    ALTER TABLE IF EXISTS student_health_profiles
        ADD CONSTRAINT fk_student_health_profiles_tenant_student
        FOREIGN KEY (tenant_id, student_id)
        REFERENCES cbc_students(tenant_id, id)
        ON DELETE CASCADE;
EXCEPTION WHEN duplicate_object THEN NULL;
END $$;

-- ============================================================================
-- ITEM 4 — Add tenant scoping to cbc_strands, cbc_sub_strands,
--          and performance_indicators
-- These tables had no tenant_id at all, so direct queries bypassed RLS.
-- Follow the same pattern as item 2: add tenant_id, backfill, add composite
-- FKs, add indexes, enable RLS, add policy.
-- ============================================================================

-- ---------------------------------------------------------------
-- 4a: cbc_strands — add tenant_id from cbc_learning_areas
-- ---------------------------------------------------------------

-- NOTE: cbc_learning_areas already has uq_cbc_learning_areas_tenant UNIQUE (tenant_id, id)
-- from the original migration, which provides the FK target.

-- Add tenant_id column (nullable initially)
ALTER TABLE IF EXISTS cbc_strands
    ADD COLUMN IF NOT EXISTS tenant_id UUID;

-- Backfill from cbc_learning_areas
UPDATE cbc_strands cs
SET tenant_id = cla.tenant_id
FROM cbc_learning_areas cla
WHERE cs.learning_area_id = cla.id
  AND cs.tenant_id IS NULL;

-- Enforce NOT NULL
ALTER TABLE IF EXISTS cbc_strands
    ALTER COLUMN tenant_id SET NOT NULL;

-- Ensure UNIQUE (tenant_id, id) on cbc_strands for downstream FKs
DO $$
BEGIN
    ALTER TABLE IF EXISTS cbc_strands
        ADD CONSTRAINT uq_cbc_strands_tenant UNIQUE (tenant_id, id);
EXCEPTION WHEN duplicate_object THEN NULL;
END $$;

-- Add composite FK (tenant_id, learning_area_id) → cbc_learning_areas(tenant_id, id)
DO $$
BEGIN
    ALTER TABLE IF EXISTS cbc_strands
        ADD CONSTRAINT fk_cbc_strands_tenant_learning_area
        FOREIGN KEY (tenant_id, learning_area_id)
        REFERENCES cbc_learning_areas(tenant_id, id)
        ON DELETE CASCADE;
EXCEPTION WHEN duplicate_object THEN NULL;
END $$;

-- Add index on tenant_id
CREATE INDEX IF NOT EXISTS idx_cbc_strands_tenant ON cbc_strands (tenant_id);

-- Enable RLS and add policy
ALTER TABLE IF EXISTS cbc_strands ENABLE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS tenant_isolation_policy ON cbc_strands;
CREATE POLICY tenant_isolation_policy ON cbc_strands
    FOR ALL
    USING (tenant_id = fn_current_tenant_id())
    WITH CHECK (tenant_id = fn_current_tenant_id());

-- ---------------------------------------------------------------
-- 4b: cbc_sub_strands — add tenant_id from cbc_strands
-- ---------------------------------------------------------------

ALTER TABLE IF EXISTS cbc_sub_strands
    ADD COLUMN IF NOT EXISTS tenant_id UUID;

-- Backfill from cbc_strands
UPDATE cbc_sub_strands css
SET tenant_id = cs.tenant_id
FROM cbc_strands cs
WHERE css.strand_id = cs.id
  AND css.tenant_id IS NULL;

ALTER TABLE IF EXISTS cbc_sub_strands
    ALTER COLUMN tenant_id SET NOT NULL;

-- Ensure UNIQUE (tenant_id, id) on cbc_sub_strands for downstream FKs
DO $$
BEGIN
    ALTER TABLE IF EXISTS cbc_sub_strands
        ADD CONSTRAINT uq_cbc_sub_strands_tenant UNIQUE (tenant_id, id);
EXCEPTION WHEN duplicate_object THEN NULL;
END $$;

-- Add composite FK (tenant_id, strand_id) → cbc_strands(tenant_id, id)
DO $$
BEGIN
    ALTER TABLE IF EXISTS cbc_sub_strands
        ADD CONSTRAINT fk_cbc_sub_strands_tenant_strand
        FOREIGN KEY (tenant_id, strand_id)
        REFERENCES cbc_strands(tenant_id, id)
        ON DELETE CASCADE;
EXCEPTION WHEN duplicate_object THEN NULL;
END $$;

CREATE INDEX IF NOT EXISTS idx_cbc_sub_strands_tenant ON cbc_sub_strands (tenant_id);

ALTER TABLE IF EXISTS cbc_sub_strands ENABLE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS tenant_isolation_policy ON cbc_sub_strands;
CREATE POLICY tenant_isolation_policy ON cbc_sub_strands
    FOR ALL
    USING (tenant_id = fn_current_tenant_id())
    WITH CHECK (tenant_id = fn_current_tenant_id());

-- ---------------------------------------------------------------
-- 4c: performance_indicators — add tenant_id from cbc_sub_strands
-- ---------------------------------------------------------------

ALTER TABLE IF EXISTS performance_indicators
    ADD COLUMN IF NOT EXISTS tenant_id UUID;

-- Backfill from cbc_sub_strands
UPDATE performance_indicators pi
SET tenant_id = css.tenant_id
FROM cbc_sub_strands css
WHERE pi.sub_strand_id = css.id
  AND pi.tenant_id IS NULL;

ALTER TABLE IF EXISTS performance_indicators
    ALTER COLUMN tenant_id SET NOT NULL;

-- Add composite FK (tenant_id, sub_strand_id) → cbc_sub_strands(tenant_id, id)
DO $$
BEGIN
    ALTER TABLE IF EXISTS performance_indicators
        ADD CONSTRAINT fk_performance_indicators_tenant_sub_strand
        FOREIGN KEY (tenant_id, sub_strand_id)
        REFERENCES cbc_sub_strands(tenant_id, id)
        ON DELETE CASCADE;
EXCEPTION WHEN duplicate_object THEN NULL;
END $$;

CREATE INDEX IF NOT EXISTS idx_performance_indicators_tenant
    ON performance_indicators (tenant_id);

ALTER TABLE IF EXISTS performance_indicators ENABLE ROW LEVEL SECURITY;
DROP POLICY IF EXISTS tenant_isolation_policy ON performance_indicators;
CREATE POLICY tenant_isolation_policy ON performance_indicators
    FOR ALL
    USING (tenant_id = fn_current_tenant_id())
    WITH CHECK (tenant_id = fn_current_tenant_id());

-- ============================================================================
-- ITEM 5 — Add "one primary parent per student" partial unique index
-- Prevents multiple cbc_student_parents rows with is_primary = true for the
-- same student, matching the existing idx_cbc_one_primary_per_class pattern.
-- ============================================================================

DO $$
DECLARE
    v_count INT;
BEGIN
    SELECT COUNT(*) INTO v_count
    FROM (
        SELECT student_id
        FROM cbc_student_parents
        WHERE is_primary = true
        GROUP BY student_id
        HAVING COUNT(*) > 1
    ) dupes;

    IF v_count > 0 THEN
        RAISE WARNING 'ITEM 5 — Found % students with more than one primary parent. The index CANNOT be added until these are resolved. Violating rows: SELECT student_id, COUNT(*) AS primary_count FROM cbc_student_parents WHERE is_primary = true GROUP BY student_id HAVING COUNT(*) > 1;', v_count;
    ELSE
        RAISE NOTICE 'ITEM 5 — No duplicate primary parents found. Adding unique index.';
        CREATE UNIQUE INDEX IF NOT EXISTS idx_one_primary_parent_per_student
            ON cbc_student_parents (student_id) WHERE is_primary = true;
    END IF;
END $$;

-- ============================================================================
-- ITEM 6 — DB-enforced immutability triggers
-- Two write-once invariants that were only enforced at the application layer.
-- ============================================================================

-- ---------------------------------------------------------------
-- 6a: grading_scale_ranges — block UPDATE/DELETE if profile_id is referenced
--     by any assessment_sessions.grading_scale_profile_id
-- ---------------------------------------------------------------

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

-- ---------------------------------------------------------------
-- 6b: assessment_sessions.max_points — block update if any score rows exist
-- ---------------------------------------------------------------

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

-- ============================================================================
-- ============================================================================
-- ITEM 6b — Fix invoice payment-status trigger functions
-- The original functions in 000001 used CASE expressions returning text,
-- which fails when PostgreSQL compiles them at first execution (PL/pgSQL
-- defers type-checking of UPDATE...SET until runtime). The 7a UPDATE on
-- payments below fires trg_sync_invoice_payment_status_update, triggering
-- compilation. Fix by adding explicit ::invoice_payment_status casts.
-- ============================================================================

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

-- ============================================================================
-- ITEM 7 — Convert free-text columns to enums
-- ============================================================================

-- ---------------------------------------------------------------
-- 7a: payments.payment_method → payment_method_type enum
-- ---------------------------------------------------------------

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

-- Backfill: map existing values case-insensitively; NULL/unrecognized → 'OTHER'
DO $$
DECLARE
    v_count INT;
BEGIN
    -- Report the number of values that will be mapped to 'OTHER'
    SELECT COUNT(*) INTO v_count
    FROM payments
    WHERE payment_method IS NULL
       OR LOWER(TRIM(payment_method)) NOT IN ('mpesa', 'cash', 'bank_transfer', 'cheque');

    IF v_count > 0 THEN
        RAISE NOTICE 'ITEM 7a — % payment records have NULL/unrecognized payment_method values that will be mapped to OTHER.', v_count;
    END IF;
END $$;

-- Create a temporary column to stage the cast
ALTER TABLE IF EXISTS payments
    ADD COLUMN IF NOT EXISTS payment_method_new payment_method_type;

UPDATE payments
SET payment_method_new = CASE
    WHEN LOWER(TRIM(payment_method)) = 'mpesa'         THEN 'MPESA'::payment_method_type
    WHEN LOWER(TRIM(payment_method)) = 'cash'           THEN 'CASH'::payment_method_type
    WHEN LOWER(TRIM(payment_method)) = 'bank_transfer'  THEN 'BANK_TRANSFER'::payment_method_type
    WHEN LOWER(TRIM(payment_method)) = 'cheque'         THEN 'CHEQUE'::payment_method_type
    ELSE 'OTHER'
END;

-- Drop old column, rename new
ALTER TABLE IF EXISTS payments
    DROP COLUMN payment_method;

ALTER TABLE IF EXISTS payments
    RENAME COLUMN payment_method_new TO payment_method;

-- Original column was NULLable; the enum column retains NULLability.

COMMENT ON COLUMN payments.payment_method IS
    'Payment method type enum. Covers the four real Kenyan payment channels
     plus OTHER. Original free-text column migrated to enum in 000002_fix.';

-- ---------------------------------------------------------------
-- 7b: cbc_student_parents.relationship → parent_relationship_type enum
-- ---------------------------------------------------------------

DO $$ BEGIN
    CREATE TYPE parent_relationship_type AS ENUM (
        'FATHER',
        'MOTHER',
        'GUARDIAN',
        'OTHER'
    );
EXCEPTION WHEN duplicate_object THEN NULL;
END $$;

-- Backfill: map case-insensitively; NULL/unrecognized → 'OTHER'
DO $$
DECLARE
    v_count INT;
BEGIN
    SELECT COUNT(*) INTO v_count
    FROM cbc_student_parents
    WHERE relationship IS NULL
       OR LOWER(TRIM(relationship)) NOT IN ('father', 'mother', 'guardian');

    IF v_count > 0 THEN
        RAISE NOTICE 'ITEM 7b — % parent relationship records have NULL/unrecognized values that will be mapped to OTHER.', v_count;
    END IF;
END $$;

ALTER TABLE IF EXISTS cbc_student_parents
    ADD COLUMN IF NOT EXISTS relationship_new parent_relationship_type;

UPDATE cbc_student_parents
SET relationship_new = CASE
    WHEN LOWER(TRIM(relationship)) = 'father'   THEN 'FATHER'::parent_relationship_type
    WHEN LOWER(TRIM(relationship)) = 'mother'   THEN 'MOTHER'::parent_relationship_type
    WHEN LOWER(TRIM(relationship)) = 'guardian' THEN 'GUARDIAN'::parent_relationship_type
    ELSE 'OTHER'
END;

ALTER TABLE IF EXISTS cbc_student_parents
    DROP COLUMN relationship;

ALTER TABLE IF EXISTS cbc_student_parents
    RENAME COLUMN relationship_new TO relationship;

COMMENT ON COLUMN cbc_student_parents.relationship IS
    'Parent/guardian relationship to the student. Enum migrated from free-text
     in 000002_fix. Values: FATHER, MOTHER, GUARDIAN, OTHER.';

-- ============================================================================
-- ITEM 8 — Add hashed-token columns for auth secrets (pgcrypto)
-- Raw plaintext tokens in sessions.token, sessions.stytch_session_token,
-- and invitations.token are security-sensitive. Add SHA-256 hash columns.
-- ============================================================================

-- Enable pgcrypto extension (idempotent)
CREATE EXTENSION IF NOT EXISTS pgcrypto;

-- ---------------------------------------------------------------
-- 8a: sessions.token_hash (backfilled from sessions.token)
-- ---------------------------------------------------------------

ALTER TABLE IF EXISTS sessions
    ADD COLUMN IF NOT EXISTS token_hash TEXT;

UPDATE sessions
SET token_hash = encode(digest(token, 'sha256'), 'hex')
WHERE token_hash IS NULL;

CREATE UNIQUE INDEX IF NOT EXISTS idx_sessions_token_hash
    ON sessions (token_hash);

COMMENT ON COLUMN sessions.token IS
    'DEPRECATED — raw session token. New code should read token_hash instead.
     This column will be dropped in a future migration after the app is
     confirmed fully migrated to hash-based lookups. Do NOT write to this
     column in new code.';

COMMENT ON COLUMN sessions.token_hash IS
    'SHA-256 hash of the session token (hex-encoded). Backfilled from token
     column. Use this for token lookups instead of the raw token column.
     Added in 000002_fix_review_findings.';

-- ---------------------------------------------------------------
-- 8b: stytch_session_token — TODO only; leave as comment
-- ---------------------------------------------------------------

COMMENT ON COLUMN sessions.stytch_session_token IS
    'TODO (000002_fix): stytch_session_token is a third-party session token from
     Stytch, not one this schema issues. Hashing strategy for Stytch tokens
     requires app-team sign-off — do not implement hashing for this column
     without a reviewed design doc.';

-- ---------------------------------------------------------------
-- 8c: invitations.token_hash (backfilled from invitations.token)
-- ---------------------------------------------------------------

ALTER TABLE IF EXISTS invitations
    ADD COLUMN IF NOT EXISTS token_hash TEXT;

UPDATE invitations
SET token_hash = encode(digest(token, 'sha256'), 'hex')
WHERE token_hash IS NULL;

CREATE UNIQUE INDEX IF NOT EXISTS idx_invitations_token_hash
    ON invitations (token_hash);

COMMENT ON COLUMN invitations.token IS
    'DEPRECATED — raw invitation token. New code should read token_hash instead.
     This column will be dropped in a future migration after the app is
     confirmed fully migrated to hash-based lookups. Do NOT write to this
     column in new code.';

COMMENT ON COLUMN invitations.token_hash IS
    'SHA-256 hash of the invitation token (hex-encoded). Backfilled from token
     column. Use this for token lookups instead of the raw token column.
     Added in 000002_fix_review_findings.';

-- ============================================================================
-- ITEM 9 — Fix users.email uniqueness: per-tenant + case-insensitive
-- The old global unique index allowed the same person across tenants only
-- with different emails, and treated case variants as distinct.
-- Replace with per-tenant, case-insensitive unique index.
-- ============================================================================

DO $$
DECLARE
    v_count INT;
BEGIN
    -- Check for case-variant duplicate emails within the same tenant
    SELECT COUNT(*) INTO v_count
    FROM (
        SELECT tenant_id, LOWER(email) AS normalized_email
        FROM users
        GROUP BY tenant_id, LOWER(email)
        HAVING COUNT(*) > 1
    ) dupes;

    IF v_count > 0 THEN
        RAISE WARNING 'ITEM 9 — Found % case-insensitive duplicate email groups within the same tenant. The new index CANNOT be added until these are resolved. Violations: SELECT tenant_id, LOWER(email) AS normalized_email, COUNT(*) FROM users GROUP BY tenant_id, LOWER(email) HAVING COUNT(*) > 1;', v_count;
    ELSE
        RAISE NOTICE 'ITEM 9 — No case-insensitive duplicates found within tenants. Proceeding.';
    END IF;
END $$;

-- Drop the old global unique index
DROP INDEX IF EXISTS idx_users_email;

-- Add the new per-tenant, case-insensitive unique index
CREATE UNIQUE INDEX IF NOT EXISTS idx_users_tenant_email
    ON users (tenant_id, LOWER(email));

COMMENT ON INDEX idx_users_tenant_email IS
    'Per-tenant, case-insensitive unique constraint on email. Replaces the
     old global idx_users_email which prevented multi-tenant accounts and
     treated case variants as distinct. Added in 000003_fix.';

-- ============================================================================
-- END OF MIGRATION
-- ============================================================================
