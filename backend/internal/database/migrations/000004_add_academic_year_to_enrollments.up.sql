-- Migration: 000004_add_academic_year_to_enrollments
-- Adds a direct academic_year_id relationship to cbc_student_enrollments
-- for efficient year-scoped queries and future assessment session support.

-- ============================================================================
-- STEP 1: Add nullable column
-- ============================================================================

ALTER TABLE cbc_student_enrollments
    ADD COLUMN IF NOT EXISTS academic_year_id UUID;

-- ============================================================================
-- STEP 2: Backfill existing rows from the academic_year_id on academic_terms
-- ============================================================================

UPDATE cbc_student_enrollments e
SET academic_year_id = t.academic_year_id
FROM academic_terms t
WHERE e.academic_term_id = t.id
  AND e.academic_year_id IS NULL;

-- ============================================================================
-- STEP 3: Add foreign key and NOT NULL constraint
-- ============================================================================

-- Add FK referencing academic_years via the composite (tenant_id, id) pair
ALTER TABLE cbc_student_enrollments
    ADD CONSTRAINT fk_enrollments_tenant_academic_year
    FOREIGN KEY (tenant_id, academic_year_id)
    REFERENCES academic_years(tenant_id, id)
    ON DELETE CASCADE;

-- Now enforce NOT NULL (safe after backfill)
ALTER TABLE cbc_student_enrollments
    ALTER COLUMN academic_year_id SET NOT NULL;

-- ============================================================================
-- STEP 4: Create index for year-scoped queries
-- ============================================================================

CREATE INDEX IF NOT EXISTS idx_cbc_enrollments_academic_year_id
    ON cbc_student_enrollments (academic_year_id);
