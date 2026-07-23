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

CREATE TABLE IF NOT EXISTS student_subject_strand_summaries (
    id                      UUID          PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id               UUID          NOT NULL,
    school_id               UUID          NOT NULL,
    student_id              UUID          NOT NULL,
    academic_term_id        UUID          NOT NULL,
    learning_area_id        UUID          NOT NULL,
    strand_id               UUID          NOT NULL,
    sub_strand_id           UUID          NOT NULL,
    mastery_percentage      NUMERIC(5,2),
    exceeding_count         INT           NOT NULL DEFAULT 0,
    meeting_count           INT           NOT NULL DEFAULT 0,
    approaching_count       INT           NOT NULL DEFAULT 0,
    below_count             INT           NOT NULL DEFAULT 0,
    mapped_performance_level VARCHAR(5),
    requires_remediation    BOOLEAN       NOT NULL DEFAULT false,
    has_data                BOOLEAN       NOT NULL DEFAULT false,
    last_refreshed_at       TIMESTAMPTZ   NOT NULL DEFAULT NOW(),
    created_at              TIMESTAMPTZ   NOT NULL DEFAULT NOW(),
    updated_at              TIMESTAMPTZ   NOT NULL DEFAULT NOW(),

    CONSTRAINT uq_student_term_sub_strand UNIQUE (student_id, academic_term_id, sub_strand_id),
    CONSTRAINT fk_strand_summaries_tenant_student
        FOREIGN KEY (tenant_id, student_id)
        REFERENCES cbc_students(tenant_id, id) ON DELETE CASCADE,
    CONSTRAINT fk_strand_summaries_tenant_term
        FOREIGN KEY (tenant_id, school_id, academic_term_id)
        REFERENCES academic_terms(tenant_id, school_id, id) ON DELETE CASCADE,
    CONSTRAINT fk_strand_summaries_tenant_school
        FOREIGN KEY (tenant_id, school_id)
        REFERENCES cbc_schools(tenant_id, id) ON DELETE CASCADE,
    CONSTRAINT fk_strand_summaries_learning_area
        FOREIGN KEY (learning_area_id)
        REFERENCES cbc_learning_areas(id) ON DELETE CASCADE,
    CONSTRAINT fk_strand_summaries_strand
        FOREIGN KEY (strand_id)
        REFERENCES cbc_strands(id) ON DELETE CASCADE,
    CONSTRAINT fk_strand_summaries_sub_strand
        FOREIGN KEY (sub_strand_id)
        REFERENCES cbc_sub_strands(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_strand_summaries_tenant
    ON student_subject_strand_summaries (tenant_id);
CREATE INDEX IF NOT EXISTS idx_strand_summaries_school
    ON student_subject_strand_summaries (school_id);
CREATE INDEX IF NOT EXISTS idx_strand_summaries_student_term
    ON student_subject_strand_summaries (student_id, academic_term_id);
CREATE INDEX IF NOT EXISTS idx_strand_summaries_term_sub_strand
    ON student_subject_strand_summaries (academic_term_id, sub_strand_id);

DROP TRIGGER IF EXISTS trg_student_subject_strand_summaries_updated_at
    ON student_subject_strand_summaries;
CREATE TRIGGER trg_student_subject_strand_summaries_updated_at
    BEFORE UPDATE ON student_subject_strand_summaries
    FOR EACH ROW EXECUTE FUNCTION fn_set_updated_at();

COMMENT ON TABLE student_subject_strand_summaries IS
    'Rubric-only sub-strand-level summary per student and term. Counts
     performance indicators awarded at each CBC level (EE/ME/AE/BE) and
     computes mastery as the percentage at ME or above. Only populated for
     RUBRIC sessions — for quantitative subjects has_data stays false.';

COMMENT ON COLUMN student_subject_strand_summaries.mastery_percentage IS
    'Percentage of performance indicators for this sub-strand that were
     awarded Meeting Expectations or above:
     (exceeding_count + meeting_count) / (total_indicators) * 100.
     NULL when no data exists.';

COMMENT ON COLUMN student_subject_strand_summaries.mapped_performance_level IS
    'The CBC performance level (EE/ME/AE/BE) corresponding to
     mastery_percentage, determined by the school''s active grading
     scale profile. NULL when no profile can be resolved.';

COMMENT ON COLUMN student_subject_strand_summaries.requires_remediation IS
    'TRUE when any indicator was awarded Below Expectations or when
     mastery_percentage is below 50%. Suggests the student needs
     targeted intervention on this sub-strand.';

COMMENT ON COLUMN student_subject_strand_summaries.has_data IS
    'TRUE when at least one rubric outcome grade exists for this
     student+term+sub_strand. For subjects assessed purely quantitatively,
     this stays false — consumers should check this flag before displaying
     mastery metrics to avoid misleading 0% displays.';

-- ============================================================================
-- FUNCTION: fn_refresh_subject_strand_summary_for_session(target_session_id UUID)
--
-- Recomputes student_subject_strand_summaries for all students in the given
-- session, scoped to the session's academic_term_id and sub-strands that
-- were assessed.
--
-- This function ONLY processes sessions with evaluation_method = 'RUBRIC'.
-- For QUANTITATIVE sessions, it is a no-op.
--
-- The algorithm:
--   1. Resolves the session's metadata (tenant_id, school_id,
--      academic_term_id, learning_area_id).
--   2. If the session is not RUBRIC, returns immediately (no-op).
--   3. For each student in the session, groups outcome grades by
--      sub_strand_id (via performance_indicator_id → cbc_sub_strands).
--   4. Counts indicators at each level (EE, ME, AE, BE).
--   5. Computes mastery_percentage = (EE_count + ME_count) / total * 100.
--   6. Maps mastery_percentage to a CBC level using the school's active
--      grading scale profile.
--   7. Sets requires_remediation = (below_count > 0 OR mastery < 50%).
--   8. Sets has_data = true when any outcome grade exists.
--   9. Upserts into student_subject_strand_summaries.
-- ============================================================================

CREATE OR REPLACE FUNCTION fn_refresh_subject_strand_summary_for_session(target_session_id UUID)
RETURNS void
LANGUAGE plpgsql
AS $$
DECLARE
    v_tenant_id          UUID;
    v_school_id          UUID;
    v_academic_term_id   UUID;
    v_learning_area_id   UUID;
    v_evaluation_method  TEXT;
    v_scale_profile_id   UUID;
BEGIN
    -- Resolve session metadata
    SELECT tenant_id, school_id, academic_term_id, learning_area_id,
           evaluation_method::TEXT
    INTO v_tenant_id, v_school_id, v_academic_term_id, v_learning_area_id,
         v_evaluation_method
    FROM assessment_sessions
    WHERE id = target_session_id;

    IF NOT FOUND THEN
        RETURN;
    END IF;

    -- This table is rubric-only — skip quantitative sessions
    IF v_evaluation_method != 'RUBRIC' THEN
        RETURN;
    END IF;

    -- Find the school's active grading scale profile for level mapping
    SELECT id INTO v_scale_profile_id
    FROM grading_scale_profiles
    WHERE school_id = v_school_id AND is_active = true
    ORDER BY created_at DESC
    LIMIT 1;

    -- Upsert summary for each student+sub_strand in this session
    INSERT INTO student_subject_strand_summaries (
        tenant_id, school_id, student_id, academic_term_id,
        learning_area_id, strand_id, sub_strand_id,
        mastery_percentage,
        exceeding_count, meeting_count, approaching_count, below_count,
        mapped_performance_level,
        requires_remediation, has_data,
        last_refreshed_at
    )
    WITH affected_students AS (
        SELECT DISTINCT student_id
        FROM student_assessment_outcome_grades
        WHERE session_id = target_session_id
    ),
    indicator_hierarchy AS (
        -- Resolve the strand and sub-strand for each performance indicator
        SELECT
            pi.id AS indicator_id,
            ss.id AS sub_strand_id,
            ss.name AS sub_strand_name,
            s.id AS strand_id,
            s.name AS strand_name
        FROM performance_indicators pi
        JOIN cbc_sub_strands ss ON ss.id = pi.sub_strand_id
        JOIN cbc_strands s ON s.id = ss.strand_id
    ),
    all_outcome_grades_for_term AS (
        -- All outcome grades for these students in this term and learning area,
        -- from ALL PUBLISHED rubric sessions (not just the target session)
        SELECT
            sog.student_id,
            sog.performance_indicator_id,
            sog.awarded_level::TEXT AS awarded_level,
            ih.sub_strand_id,
            ih.strand_id
        FROM student_assessment_outcome_grades sog
        JOIN assessment_sessions s ON s.id = sog.session_id
        JOIN indicator_hierarchy ih ON ih.indicator_id = sog.performance_indicator_id
        WHERE s.academic_term_id = v_academic_term_id
          AND s.learning_area_id = v_learning_area_id
          AND s.status = 'PUBLISHED'
          AND s.evaluation_method = 'RUBRIC'::assessment_evaluation_method
          AND sog.student_id IN (SELECT student_id FROM affected_students)
    ),
    level_counts AS (
        SELECT
            student_id,
            sub_strand_id,
            strand_id,
            COUNT(*) FILTER (WHERE awarded_level = 'EE') AS ee_count,
            COUNT(*) FILTER (WHERE awarded_level = 'ME') AS me_count,
            COUNT(*) FILTER (WHERE awarded_level = 'AE') AS ae_count,
            COUNT(*) FILTER (WHERE awarded_level = 'BE') AS be_count,
            COUNT(*) AS total_count
        FROM all_outcome_grades_for_term
        GROUP BY student_id, sub_strand_id, strand_id
    )
    SELECT
        v_tenant_id,
        v_school_id,
        lc.student_id,
        v_academic_term_id,
        v_learning_area_id,
        lc.strand_id,
        lc.sub_strand_id,
        -- mastery_percentage: (EE + ME) / total * 100
        CASE
            WHEN lc.total_count > 0
            THEN ROUND((lc.ee_count + lc.me_count)::NUMERIC / lc.total_count * 100, 2)
            ELSE NULL
        END AS mastery_percentage,
        lc.ee_count,
        lc.me_count,
        lc.ae_count,
        lc.be_count,
        -- mapped_performance_level: map mastery_percentage via scale profile
        CASE
            WHEN lc.total_count > 0
             AND v_scale_profile_id IS NOT NULL
            THEN (
                SELECT r.performance_level::TEXT
                FROM grading_scale_ranges r
                WHERE r.profile_id = v_scale_profile_id
                  AND ROUND((lc.ee_count + lc.me_count)::NUMERIC / lc.total_count * 100, 2) >= r.min_percentage
                  AND ROUND((lc.ee_count + lc.me_count)::NUMERIC / lc.total_count * 100, 2) <= r.max_percentage
                LIMIT 1
            )
            ELSE NULL
        END AS mapped_performance_level,
        -- requires_remediation: BE > 0 OR mastery < 50%
        CASE
            WHEN lc.total_count > 0
            THEN (lc.be_count > 0)
               OR (ROUND((lc.ee_count + lc.me_count)::NUMERIC / lc.total_count * 100, 2) < 50)
            ELSE false
        END AS requires_remediation,
        -- has_data: true if any outcome grades exist
        lc.total_count > 0 AS has_data,
        NOW()
    FROM level_counts lc
    WHERE lc.total_count > 0

    ON CONFLICT (student_id, academic_term_id, sub_strand_id)
    DO UPDATE SET
        mastery_percentage        = EXCLUDED.mastery_percentage,
        exceeding_count           = EXCLUDED.exceeding_count,
        meeting_count             = EXCLUDED.meeting_count,
        approaching_count         = EXCLUDED.approaching_count,
        below_count               = EXCLUDED.below_count,
        mapped_performance_level  = EXCLUDED.mapped_performance_level,
        requires_remediation      = EXCLUDED.requires_remediation,
        has_data                  = EXCLUDED.has_data,
        last_refreshed_at         = NOW(),
        updated_at                = NOW();

    -- Clean up orphaned rows where the student no longer has any outcome
    -- grades in this term for this sub-strand (e.g. session was unpublished)
    DELETE FROM student_subject_strand_summaries
    WHERE academic_term_id = v_academic_term_id
      AND learning_area_id = v_learning_area_id
      AND student_id IN (SELECT student_id FROM affected_students)
      AND has_data = false;
END;
$$;

COMMENT ON FUNCTION fn_refresh_subject_strand_summary_for_session IS
    'Refreshes student_subject_strand_summaries for all students in the given
     rubric session. Groups outcome grades by sub-strand, counts level
     distributions, computes mastery percentage, and determines remediation
     need. No-op for QUANTITATIVE sessions.';

-- ============================================================================
-- EXTEND THE EXISTING AFTER-PUBLISH TRIGGER
--
-- The existing trigger fn_assessment_sessions_after_publish() is extended
-- to also call fn_refresh_subject_strand_summary_for_session() when a
-- rubric session is published.
-- ============================================================================

CREATE OR REPLACE FUNCTION fn_assessment_sessions_after_publish()
RETURNS TRIGGER
LANGUAGE plpgsql
AS $$
BEGIN
    IF NEW.status = 'PUBLISHED' AND (OLD.status IS DISTINCT FROM 'PUBLISHED') THEN
        -- Refresh term-level subject summaries (existing)
        PERFORM fn_refresh_term_subject_summary_for_session(NEW.id);

        -- Refresh sub-strand-level summaries (new, rubric-only)
        PERFORM fn_refresh_subject_strand_summary_for_session(NEW.id);
    END IF;
    RETURN NEW;
END;
$$;

-- ============================================================================
-- RLS POLICY
-- ============================================================================

ALTER TABLE IF EXISTS student_subject_strand_summaries ENABLE ROW LEVEL SECURITY;

DO $$ BEGIN
    DROP POLICY IF EXISTS tenant_isolation_policy ON student_subject_strand_summaries;
    CREATE POLICY tenant_isolation_policy ON student_subject_strand_summaries
        FOR ALL
        USING (tenant_id = fn_current_tenant_id())
        WITH CHECK (tenant_id = fn_current_tenant_id());
END $$;

COMMENT ON TABLE student_subject_strand_summaries IS
    'Rubric-only sub-strand-level summary per student and term. RLS-enabled
     — tenant-scoped.';
