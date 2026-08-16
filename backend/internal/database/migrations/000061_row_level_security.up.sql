-- Migration: 000061_row_level_security
-- SomoTracker — Kenya CBC/CBE academic platform (CBC-only)
-- Split from the squashed 000001_initial_schema on 2026-08-16. DDL unchanged.
-- Purpose: Row-Level Security enable + policies + SECURITY DEFINER helpers

ALTER TABLE IF EXISTS student_term_subject_summaries ENABLE ROW LEVEL SECURITY;

DO $$ BEGIN
    DROP POLICY IF EXISTS tenant_isolation_policy ON student_term_subject_summaries;
    CREATE POLICY tenant_isolation_policy ON student_term_subject_summaries
        FOR ALL
        USING (tenant_id = fn_current_tenant_id())
        WITH CHECK (tenant_id = fn_current_tenant_id());
END $$;




ALTER TABLE IF EXISTS student_term_overall_summaries ENABLE ROW LEVEL SECURITY;

DO $$ BEGIN
    DROP POLICY IF EXISTS tenant_isolation_policy ON student_term_overall_summaries;
    CREATE POLICY tenant_isolation_policy ON student_term_overall_summaries
        FOR ALL
        USING (tenant_id = fn_current_tenant_id())
        WITH CHECK (tenant_id = fn_current_tenant_id());
END $$;




ALTER TABLE IF EXISTS student_cohort_position_summaries ENABLE ROW LEVEL SECURITY;

DO $$ BEGIN
    DROP POLICY IF EXISTS tenant_isolation_policy ON student_cohort_position_summaries;
    CREATE POLICY tenant_isolation_policy ON student_cohort_position_summaries
        FOR ALL
        USING (tenant_id = fn_current_tenant_id())
        WITH CHECK (tenant_id = fn_current_tenant_id());
END $$;




ALTER TABLE IF EXISTS student_subject_strand_summaries ENABLE ROW LEVEL SECURITY;

DO $$ BEGIN
    DROP POLICY IF EXISTS tenant_isolation_policy ON student_subject_strand_summaries;
    CREATE POLICY tenant_isolation_policy ON student_subject_strand_summaries
        FOR ALL
        USING (tenant_id = fn_current_tenant_id())
        WITH CHECK (tenant_id = fn_current_tenant_id());
END $$;




ALTER TABLE IF EXISTS student_performance_projections ENABLE ROW LEVEL SECURITY;

DO $$ BEGIN
    DROP POLICY IF EXISTS tenant_isolation_policy ON student_performance_projections;
    CREATE POLICY tenant_isolation_policy ON student_performance_projections
        FOR ALL
        USING (tenant_id = fn_current_tenant_id())
        WITH CHECK (tenant_id = fn_current_tenant_id());
END $$;




ALTER TABLE IF EXISTS student_behavior_term_summaries ENABLE ROW LEVEL SECURITY;

DO $$ BEGIN
    DROP POLICY IF EXISTS tenant_isolation_policy ON student_behavior_term_summaries;
    CREATE POLICY tenant_isolation_policy ON student_behavior_term_summaries
        FOR ALL
        USING (tenant_id = fn_current_tenant_id())
        WITH CHECK (tenant_id = fn_current_tenant_id());
END $$;




ALTER TABLE IF EXISTS teacher_subject_performance_summaries ENABLE ROW LEVEL SECURITY;

DO $$ BEGIN
    DROP POLICY IF EXISTS tenant_isolation_policy ON teacher_subject_performance_summaries;
    CREATE POLICY tenant_isolation_policy ON teacher_subject_performance_summaries
        FOR ALL
        USING (tenant_id = fn_current_tenant_id())
        WITH CHECK (tenant_id = fn_current_tenant_id());
END $$;




ALTER TABLE IF EXISTS teacher_delivery_summaries ENABLE ROW LEVEL SECURITY;

DO $$ BEGIN
    DROP POLICY IF EXISTS tenant_isolation_policy ON teacher_delivery_summaries;
    CREATE POLICY tenant_isolation_policy ON teacher_delivery_summaries
        FOR ALL
        USING (tenant_id = fn_current_tenant_id())
        WITH CHECK (tenant_id = fn_current_tenant_id());
END $$;




ALTER TABLE IF EXISTS teacher_workload_summaries ENABLE ROW LEVEL SECURITY;

DO $$ BEGIN
    DROP POLICY IF EXISTS tenant_isolation_policy ON teacher_workload_summaries;
    CREATE POLICY tenant_isolation_policy ON teacher_workload_summaries
        FOR ALL
        USING (tenant_id = fn_current_tenant_id())
        WITH CHECK (tenant_id = fn_current_tenant_id());
END $$;


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

-- ============================================================================
-- SESSION RESOLUTION BYPASS (SECURITY DEFINER)
--
-- Session lookup happens BEFORE the tenant is known (the tenant is read FROM
-- the session row), so it cannot run under tenant-scoped RLS. This function
-- runs with the privileges of its owner (the migration role) and bypasses RLS
-- for the single, narrow purpose of resolving a session from its unguessable
-- token_hash. It is read-only and keyed by the SHA-256 token hash — brute
-- forcing it is infeasible.
-- ============================================================================

CREATE OR REPLACE FUNCTION fn_resolve_session(p_token_hash TEXT)
RETURNS TABLE (
    user_id            UUID,
    tenant_id          UUID,
    device_fingerprint VARCHAR(128),
    role               TEXT,
    school_id          TEXT,
    schools            TEXT[]
)
LANGUAGE sql
STABLE
SECURITY DEFINER
SET search_path = public, pg_temp
AS $$
    SELECT
        s.user_id,
        s.tenant_id,
        s.device_fingerprint,
        (
            SELECT m.role::text FROM memberships m
              WHERE m.user_id = s.user_id AND m.tenant_id = s.tenant_id AND m.is_active = true
              ORDER BY
                CASE m.role
                  WHEN 'SYSTEM_ADMIN' THEN 1
                  WHEN 'SCHOOL_ADMIN' THEN 2
                  WHEN 'TEACHER' THEN 3
                  WHEN 'NURSE' THEN 4
                  WHEN 'FINANCE' THEN 5
                END
              LIMIT 1
        ) AS role,
        COALESCE(
            (SELECT mas.school_id::text FROM member_active_school mas
              WHERE mas.user_id = s.user_id AND mas.tenant_id = s.tenant_id),
            (SELECT m2.school_id::text FROM memberships m2
              WHERE m2.user_id = s.user_id AND m2.tenant_id = s.tenant_id AND m2.is_active = true
              ORDER BY
                CASE m2.role
                  WHEN 'SYSTEM_ADMIN' THEN 1
                  WHEN 'SCHOOL_ADMIN' THEN 2
                  WHEN 'TEACHER' THEN 3
                  WHEN 'NURSE' THEN 4
                  WHEN 'FINANCE' THEN 5
                END
              LIMIT 1)
        ) AS school_id,
        COALESCE(ARRAY(
            SELECT m3.school_id::text FROM memberships m3
              WHERE m3.user_id = s.user_id AND m3.tenant_id = s.tenant_id AND m3.is_active = true
        ), ARRAY[]::text[]) AS schools
    FROM sessions s
    WHERE s.token_hash = p_token_hash AND s.expires_at > NOW()
$$;

COMMENT ON FUNCTION fn_resolve_session(TEXT) IS
    'SECURITY DEFINER — bypasses RLS to resolve a session by its unguessable
     token_hash before the tenant is known. Read-only; returns the session''s
     tenant_id, role, and school context in one round trip.';

-- Invite acceptance also runs pre-session (no tenant context yet), so the
-- pending-invitation lookup must bypass RLS the same way.
CREATE OR REPLACE FUNCTION fn_pending_invitation_by_email(p_email TEXT)
RETURNS TABLE (
    id                  UUID,
    tenant_id           UUID,
    school_id           UUID,
    role                TEXT,
    email               VARCHAR(255),
    full_name           VARCHAR(255),
    status              TEXT,
    stytch_member_id    VARCHAR(255),
    registration_number VARCHAR(100),
    expires_at          TIMESTAMPTZ
)
LANGUAGE sql
STABLE
SECURITY DEFINER
SET search_path = public, pg_temp
AS $$
    SELECT id, tenant_id, school_id, role::text, email,
           COALESCE(full_name, '') AS full_name, status::text,
           COALESCE(stytch_member_id, '') AS stytch_member_id,
           COALESCE(registration_number, '') AS registration_number, expires_at
    FROM invitations
    WHERE LOWER(email) = LOWER(p_email)
      AND status = 'pending'
      AND expires_at > NOW()
    ORDER BY created_at DESC
    LIMIT 1
$$;

COMMENT ON FUNCTION fn_pending_invitation_by_email(TEXT) IS
    'SECURITY DEFINER — bypasses RLS to look up a pending invitation by email
     before the tenant is known (invite acceptance flow).';

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

-- ============================================================================
-- RLS: Explicit tenant-scoped policies for junction / denormalised tables.
-- cbc_student_parents and school_member_counts now carry tenant_id, so the
-- dynamic loop above also covers them. These explicit statements guarantee
-- coverage even if the loop's information_schema lookup were skipped.
-- ============================================================================

ALTER TABLE IF EXISTS cbc_student_parents ENABLE ROW LEVEL SECURITY;
DO $$ BEGIN
    DROP POLICY IF EXISTS tenant_isolation_policy ON cbc_student_parents;
    CREATE POLICY tenant_isolation_policy ON cbc_student_parents
        FOR ALL
        USING (tenant_id = fn_current_tenant_id())
        WITH CHECK (tenant_id = fn_current_tenant_id());
END $$;

ALTER TABLE IF EXISTS school_member_counts ENABLE ROW LEVEL SECURITY;
DO $$ BEGIN
    DROP POLICY IF EXISTS tenant_isolation_policy ON school_member_counts;
    CREATE POLICY tenant_isolation_policy ON school_member_counts
        FOR ALL
        USING (tenant_id = fn_current_tenant_id())
        WITH CHECK (tenant_id = fn_current_tenant_id());
END $$;

COMMENT ON TABLE cbc_student_enrollments IS
    'Per-term enrollment records. UNIQUE (student_id, school_id, academic_term_id)
     allows same-term transfers (old school sets status=TRANSFERRED, new school
     inserts its own row). RLS enforces tenant isolation at the database level.';

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
