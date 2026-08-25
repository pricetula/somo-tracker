-- Migration: 000046_student_assessment_scores
-- SomoTracker — Kenya CBC/CBE academic platform (CBC-only)
-- Split from the squashed 000001_initial_schema on 2026-08-16. DDL unchanged.
-- Purpose: student_assessment_scores

CREATE OR REPLACE FUNCTION max_points_check(session_id UUID, raw_score NUMERIC)
RETURNS BOOLEAN
LANGUAGE sql
STABLE
AS $$
    SELECT raw_score <= COALESCE((SELECT max_points FROM assessment_sessions WHERE id = session_id), raw_score);
$$;

CREATE TABLE IF NOT EXISTS student_assessment_scores (
    id                     UUID                  PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id              UUID                  NOT NULL,
    session_id             UUID                  NOT NULL,
    student_id             UUID                  NOT NULL,
    raw_score              NUMERIC(10,2)         NULL,
    calculated_percentage  NUMERIC(5,2)          NULL,
    final_performance_level cbc_performance_level NULL,
    enrollment_status      VARCHAR(20)           NOT NULL DEFAULT 'ACTIVE',
    created_at             TIMESTAMPTZ           NOT NULL DEFAULT NOW(),
    updated_at             TIMESTAMPTZ           NOT NULL DEFAULT NOW(),

    CONSTRAINT fk_scores_tenant_session
        FOREIGN KEY (tenant_id, session_id)
        REFERENCES assessment_sessions(tenant_id, id) ON DELETE CASCADE,
    CONSTRAINT fk_scores_tenant_student
        FOREIGN KEY (tenant_id, student_id)
        REFERENCES cbc_students(tenant_id, id) ON DELETE CASCADE,
    CONSTRAINT uq_score_session_student UNIQUE (session_id, student_id),
    CONSTRAINT chk_score_range CHECK (
        raw_score IS NULL OR (raw_score >= 0 AND max_points_check(session_id, raw_score))
    )
);

CREATE INDEX IF NOT EXISTS idx_student_scores_session
    ON student_assessment_scores (session_id);
CREATE INDEX IF NOT EXISTS idx_student_scores_student
    ON student_assessment_scores (student_id);
CREATE INDEX IF NOT EXISTS idx_student_scores_tenant
    ON student_assessment_scores (tenant_id);

DROP TRIGGER IF EXISTS trg_student_assessment_scores_updated_at ON student_assessment_scores;
CREATE TRIGGER trg_student_assessment_scores_updated_at
    BEFORE UPDATE ON student_assessment_scores
    FOR EACH ROW EXECUTE FUNCTION fn_set_updated_at();

COMMENT ON TABLE student_assessment_scores IS
    'Stores student scores for QUANTITATIVE assessment sessions. The
     final_performance_level is written (snapshotted) at the moment of
     admin approval — immune to later scale profile changes. NULL for
     RUBRIC sessions (those use student_assessment_outcome_grades).';

COMMENT ON COLUMN student_assessment_scores.enrollment_status IS
    'Denormalised enrollment status at time of grading. Used to enforce
     the No-Grade-Ghosting constraint: scores cannot be entered for
     students marked ABSENT or EXEMPT. Values: ACTIVE, SUSPENDED,
     TRANSFERRED, ABSENT, EXEMPT.';

COMMENT ON CONSTRAINT chk_score_range ON student_assessment_scores IS
    'Enforces that raw_score (when non-NULL) is non-negative AND does not exceed
     the session''s max_points. Fixed from original OR-bug which made this a no-op.';
