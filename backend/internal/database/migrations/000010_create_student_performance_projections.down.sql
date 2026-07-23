-- Migration: 000010_create_student_performance_projections — DOWN
-- Reverts: table, function, RLS

-- Drop the batch computation function
DROP FUNCTION IF EXISTS fn_compute_performance_projections_for_term(UUID);

-- Drop RLS policy
DROP POLICY IF EXISTS tenant_isolation_policy ON student_performance_projections;

-- Drop triggers
DROP TRIGGER IF EXISTS trg_student_performance_projections_updated_at
    ON student_performance_projections;

-- Drop indexes
DROP INDEX IF EXISTS idx_projections_learning_area;
DROP INDEX IF EXISTS idx_projections_term;
DROP INDEX IF EXISTS idx_projections_student_term;
DROP INDEX IF EXISTS idx_projections_school;
DROP INDEX IF EXISTS idx_projections_tenant;

-- Drop table
DROP TABLE IF EXISTS student_performance_projections;
