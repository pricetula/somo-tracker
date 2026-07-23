-- Migration: 000008_create_student_cohort_position_summaries (down)
-- Reverses the changes from the up migration.

DROP FUNCTION IF EXISTS fn_compute_cohort_positions_for_term(UUID);

DROP TRIGGER IF EXISTS trg_student_cohort_position_summaries_updated_at
    ON student_cohort_position_summaries;

DROP POLICY IF EXISTS tenant_isolation_policy ON student_cohort_position_summaries;

ALTER TABLE IF EXISTS student_cohort_position_summaries DISABLE ROW LEVEL SECURITY;

DROP TABLE IF EXISTS student_cohort_position_summaries;
