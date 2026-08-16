-- Migration: 000051_student_cohort_position_summaries
-- SomoTracker — Kenya CBC/CBE academic platform (CBC-only)
-- Split from the squashed 000001_initial_schema on 2026-08-16. DDL unchanged.
-- Purpose: student_cohort_position_summaries

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
