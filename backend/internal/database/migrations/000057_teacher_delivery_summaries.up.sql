-- Migration: 000056_teacher_delivery_summaries
-- SomoTracker — Kenya CBC/CBE academic platform (CBC-only)
-- Split from the squashed 000001_initial_schema on 2026-08-16. DDL unchanged.
-- Purpose: teacher_delivery_summaries

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
     the term. Computed as the count of (timetable_allocations × weeks) where
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
--   3. Count attendance_records per (timetable_allocation_id, date) for this teacher's slots.
--   4. Count SKIPPED sessions per (timetable_allocation_id, date) for this teacher's slots.
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
            ts.id AS timetable_allocation_id,
            tstr.day_of_week,
            ts.class_id,
            ts.learning_area_id
        FROM timetable_allocations ts
        JOIN timetable_blocks tstr ON tstr.id = ts.block_id
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
            ts.timetable_allocation_id,
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
            COUNT(DISTINCT (ar.timetable_allocation_id, ar.date))::INT AS marked_count
        FROM attendance_records ar
        JOIN timetable_allocations ts ON ts.id = ar.timetable_allocation_id
        WHERE ar.tenant_id = v_tenant_id
          AND ar.academic_term_id = target_term_id
          AND ts.teacher_id IS NOT NULL
        GROUP BY ar.tenant_id, ts.teacher_id
    ),
    -- The 'marked' CTE already uses ar.timetable_allocation_id which is correct,
    -- Count missed slots: sessions with status = SKIPPED
    missed_cte AS (
        SELECT
            s.tenant_id,
            ts.teacher_id AS user_id,
            COUNT(*)::INT AS missed_count
        FROM cbc_attendance_sessions s
        JOIN timetable_allocations ts ON ts.id = s.timetable_allocation_id
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
        JOIN timetable_allocations ts ON ts.id = s.timetable_allocation_id
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
        JOIN timetable_allocations ts ON ts.id = s.timetable_allocation_id
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
            COUNT(DISTINCT (so.timetable_allocation_id, so.occurrence_date))::INT AS total_assigned,
            COALESCE(m.marked_count, 0) AS marked,
            COALESCE(mi.missed_count, 0) AS missed,
            COALESCE(sc.sessions_count, 0) AS sessions_created_count,
            COALESCE(sa.approved_count, 0) AS sessions_approved_count
        FROM slot_occurrences so
        LEFT JOIN marked m ON m.tenant_id = v_tenant_id AND m.user_id = so.user_id
        LEFT JOIN missed_cte mi ON mi.tenant_id = v_tenant_id AND mi.user_id = so.user_id
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
          FROM timetable_allocations ts
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
