-- Migration: 000004_add_academic_year_to_enrollments (down)
-- Reverts the academic_year_id column addition.

-- Drop FK constraint first (implicitly drops the index too? No, index is separate)
ALTER TABLE cbc_student_enrollments
    DROP CONSTRAINT IF EXISTS fk_enrollments_tenant_academic_year;

-- Drop the index
DROP INDEX IF EXISTS idx_cbc_enrollments_academic_year_id;

-- Drop the column
ALTER TABLE cbc_student_enrollments
    DROP COLUMN IF EXISTS academic_year_id;
