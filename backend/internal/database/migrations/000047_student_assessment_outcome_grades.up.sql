-- Migration: 000047_student_assessment_outcome_grades
-- SomoTracker — Kenya CBC/CBE academic platform (CBC-only)
-- Split from the squashed 000001_initial_schema on 2026-08-16. DDL unchanged.
-- Purpose: student_assessment_outcome_grades

CREATE TABLE IF NOT EXISTS student_assessment_outcome_grades (
    id                      UUID                  PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id               UUID                  NOT NULL,
    session_id              UUID                  NOT NULL,
    student_id              UUID                  NOT NULL,
    performance_indicator_id UUID                 NOT NULL,
    awarded_level           cbc_performance_level NOT NULL,
    created_at              TIMESTAMPTZ           NOT NULL DEFAULT NOW(),
    updated_at              TIMESTAMPTZ           NOT NULL DEFAULT NOW(),

    CONSTRAINT fk_outcome_tenant_session
        FOREIGN KEY (tenant_id, session_id)
        REFERENCES assessment_sessions(tenant_id, id) ON DELETE CASCADE,
    CONSTRAINT fk_outcome_tenant_student
        FOREIGN KEY (tenant_id, student_id)
        REFERENCES cbc_students(tenant_id, id) ON DELETE CASCADE,
    CONSTRAINT fk_outcome_performance_indicator
        FOREIGN KEY (tenant_id, performance_indicator_id)
        REFERENCES performance_indicators(tenant_id, id) ON DELETE CASCADE,
    CONSTRAINT uq_outcome_session_student_indicator
        UNIQUE (session_id, student_id, performance_indicator_id)
);

CREATE INDEX IF NOT EXISTS idx_outcome_grades_session
    ON student_assessment_outcome_grades (session_id);
CREATE INDEX IF NOT EXISTS idx_outcome_grades_student
    ON student_assessment_outcome_grades (student_id);
CREATE INDEX IF NOT EXISTS idx_outcome_grades_indicator
    ON student_assessment_outcome_grades (performance_indicator_id);
CREATE INDEX IF NOT EXISTS idx_outcome_grades_tenant
    ON student_assessment_outcome_grades (tenant_id);

DROP TRIGGER IF EXISTS trg_student_assessment_outcome_grades_updated_at ON student_assessment_outcome_grades;
CREATE TRIGGER trg_student_assessment_outcome_grades_updated_at
    BEFORE UPDATE ON student_assessment_outcome_grades
    FOR EACH ROW EXECUTE FUNCTION fn_set_updated_at();

COMMENT ON TABLE student_assessment_outcome_grades IS
    'Stores rubric-level grades for RUBRIC assessment sessions. Each row
     maps a student to a specific KICD performance indicator with the
     awarded CBC level (EE, ME, AE, BE). No raw scores or percentages
     are stored — the teacher assigns the performance level directly.';
