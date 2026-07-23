-- Migration: 000007_create_student_term_overall_summaries (down)
-- Reverses the changes from the up migration.

DROP FUNCTION IF EXISTS fn_compute_single_student_term_overall_summary(UUID, UUID);

DROP FUNCTION IF EXISTS fn_compute_term_overall_summaries_for_term(UUID);

DROP TRIGGER IF EXISTS trg_student_term_overall_summaries_updated_at
    ON student_term_overall_summaries;

DROP POLICY IF EXISTS tenant_isolation_policy ON student_term_overall_summaries;

ALTER TABLE IF EXISTS student_term_overall_summaries DISABLE ROW LEVEL SECURITY;

DROP TABLE IF EXISTS student_term_overall_summaries;
