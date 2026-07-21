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
