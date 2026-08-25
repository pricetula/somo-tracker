-- Migration: 000053_student_performance_projections
-- SomoTracker — Kenya CBC/CBE academic platform (CBC-only)
-- Split from the squashed 000001_initial_schema on 2026-08-16. DDL unchanged.
-- Purpose: student_performance_projections

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
        FOREIGN KEY (tenant_id, learning_area_id)
        REFERENCES cbc_learning_areas(tenant_id, id) ON DELETE CASCADE
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
