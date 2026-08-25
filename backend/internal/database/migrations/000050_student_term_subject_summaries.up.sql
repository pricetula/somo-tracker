-- Migration: 000049_student_term_subject_summaries
-- SomoTracker — Kenya CBC/CBE academic platform (CBC-only)
-- Split from the squashed 000001_initial_schema on 2026-08-16. DDL unchanged.
-- Purpose: student_term_subject_summaries

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
        FOREIGN KEY (tenant_id, learning_area_id)
        REFERENCES cbc_learning_areas(tenant_id, id) ON DELETE CASCADE
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
