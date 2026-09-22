-- Migration: 000017_add_student_bulk_import_support
-- Purpose: Extend bulk_jobs to support STUDENT_IMPORT, add student_gender_counts for reporting.
-- Dependencies:
--   - 000012_bulk_job_ingestion (bulk_jobs table)
--   - 000004_create_sis_schema (schools, students tables)

-- ============================================================================
-- Section 1: Extend bulk_jobs job_type check
-- ============================================================================

ALTER TABLE bulk_jobs DROP CONSTRAINT IF EXISTS bulk_jobs_job_type_check;
ALTER TABLE bulk_jobs ADD CONSTRAINT bulk_jobs_job_type_check
    CHECK (job_type IN ('ADMIN_INVITATION','STUDENT_IMPORT'));

COMMENT ON COLUMN bulk_jobs.job_type IS 'Bulk operation category. Extendable via CHECK constraint or lookup table migration. Supported: ADMIN_INVITATION, STUDENT_IMPORT.';

-- ============================================================================
-- Section 2: student_gender_counts table
-- ============================================================================

CREATE TABLE student_gender_counts (
    school_id   UUID        NOT NULL PRIMARY KEY REFERENCES schools(id) ON DELETE CASCADE,
    male_count  INTEGER     NOT NULL DEFAULT 0,
    female_count INTEGER    NOT NULL DEFAULT 0,
    other_count INTEGER     NOT NULL DEFAULT 0,
    total_count INTEGER     NOT NULL DEFAULT 0,
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

COMMENT ON TABLE student_gender_counts IS 'Denormalized gender counts per school for fast reporting. Updated after bulk student imports and student CRUD operations.';
COMMENT ON COLUMN student_gender_counts.school_id IS 'FK to schools(id). One row per school.';
COMMENT ON COLUMN student_gender_counts.male_count IS 'Number of students with gender normalized to M.';
COMMENT ON COLUMN student_gender_counts.female_count IS 'Number of students with gender normalized to F.';
COMMENT ON COLUMN student_gender_counts.other_count IS 'Number of students with gender normalized to OTHER.';
COMMENT ON COLUMN student_gender_counts.total_count IS 'Sum of male + female + other.';
COMMENT ON COLUMN student_gender_counts.updated_at IS 'Last recompute timestamp.';

-- updated_at trigger
CREATE TRIGGER student_gender_counts_updated_at_trg
    BEFORE UPDATE ON student_gender_counts
    FOR EACH ROW
    EXECUTE FUNCTION set_updated_at();

-- Index for quick lookup by school (primary key covers this)
