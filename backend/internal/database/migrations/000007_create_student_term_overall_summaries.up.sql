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
