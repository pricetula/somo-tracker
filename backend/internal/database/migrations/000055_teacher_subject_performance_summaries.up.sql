-- Migration: 000055_teacher_subject_performance_summaries
-- SomoTracker — Kenya CBC/CBE academic platform (CBC-only)
-- Split from the squashed 000001_initial_schema on 2026-08-16. DDL unchanged.
-- Purpose: teacher_subject_performance_summaries

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
        FOREIGN KEY (tenant_id, learning_area_id)
        REFERENCES cbc_learning_areas(tenant_id, id) ON DELETE CASCADE,
    CONSTRAINT fk_teacher_perf_summaries_class
        FOREIGN KEY (tenant_id, class_id)
        REFERENCES cbc_classes(tenant_id, id) ON DELETE CASCADE,
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
