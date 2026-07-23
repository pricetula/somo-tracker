-- Migration: 000006_create_student_term_subject_summaries (down)
-- Reverses the changes from the up migration.

DROP TRIGGER IF EXISTS trg_assessment_sessions_refresh_summary
    ON assessment_sessions;

DROP FUNCTION IF EXISTS fn_assessment_sessions_after_publish();

DROP FUNCTION IF EXISTS fn_refresh_term_subject_summary_for_session(UUID);

DROP TRIGGER IF EXISTS trg_student_term_subject_summaries_updated_at
    ON student_term_subject_summaries;

DROP POLICY IF EXISTS tenant_isolation_policy ON student_term_subject_summaries;

ALTER TABLE IF EXISTS student_term_subject_summaries DISABLE ROW LEVEL SECURITY;

DROP TABLE IF EXISTS student_term_subject_summaries;
