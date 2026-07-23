-- Migration: 000005_extend_summaries_and_daily
-- 1. Adds academic_year_id and created_at to attendance_term_summaries
-- 2. Creates the class_daily_attendance_summaries materialised table

-- ============================================================================
-- PART 1: Extend attendance_term_summaries
-- ============================================================================

-- Step 1a: Add academic_year_id (nullable initially for backfill)
ALTER TABLE attendance_term_summaries
    ADD COLUMN IF NOT EXISTS academic_year_id UUID;

-- Step 1b: Add created_at
ALTER TABLE attendance_term_summaries
    ADD COLUMN IF NOT EXISTS created_at TIMESTAMPTZ NOT NULL DEFAULT NOW();

-- Step 1c: Backfill academic_year_id from academic_terms
UPDATE attendance_term_summaries s
SET academic_year_id = t.academic_year_id
FROM academic_terms t
WHERE s.academic_term_id = t.id
  AND s.academic_year_id IS NULL;

-- Step 1d: Add foreign key and NOT NULL constraint
ALTER TABLE attendance_term_summaries
    ADD CONSTRAINT fk_summaries_tenant_academic_year
    FOREIGN KEY (tenant_id, academic_year_id)
    REFERENCES academic_years(tenant_id, id)
    ON DELETE CASCADE;

ALTER TABLE attendance_term_summaries
    ALTER COLUMN academic_year_id SET NOT NULL;

-- Step 1e: Index for year-scoped queries
CREATE INDEX IF NOT EXISTS idx_att_summaries_academic_year
    ON attendance_term_summaries (academic_year_id);

-- ============================================================================
-- PART 2: Create class_daily_attendance_summaries
-- ============================================================================

CREATE TABLE IF NOT EXISTS class_daily_attendance_summaries (
    id                    UUID          PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id             UUID          NOT NULL,
    school_id             UUID          NOT NULL,
    class_id              UUID          NOT NULL,
    academic_term_id      UUID          NOT NULL,
    date                  DATE          NOT NULL,
    total_enrolled        INT           NOT NULL,
    present_count         INT           NOT NULL,
    absent_count          INT           NOT NULL,
    late_count            INT           NOT NULL,
    excused_count         INT           NOT NULL,
    daily_attendance_rate NUMERIC(5,2)  NOT NULL,
    last_refreshed_at     TIMESTAMPTZ   NOT NULL DEFAULT NOW(),
    created_at            TIMESTAMPTZ   NOT NULL DEFAULT NOW(),
    updated_at            TIMESTAMPTZ   NOT NULL DEFAULT NOW(),

    CONSTRAINT uq_class_daily_attendance UNIQUE (class_id, date),
    CONSTRAINT fk_class_daily_tenant_class
        FOREIGN KEY (tenant_id, class_id)
        REFERENCES cbc_classes(tenant_id, id) ON DELETE CASCADE,
    CONSTRAINT fk_class_daily_tenant_term
        FOREIGN KEY (tenant_id, school_id, academic_term_id)
        REFERENCES academic_terms(tenant_id, school_id, id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_class_daily_tenant
    ON class_daily_attendance_summaries (tenant_id);
CREATE INDEX IF NOT EXISTS idx_class_daily_school
    ON class_daily_attendance_summaries (school_id);
CREATE INDEX IF NOT EXISTS idx_class_daily_class_date
    ON class_daily_attendance_summaries (class_id, date);
CREATE INDEX IF NOT EXISTS idx_class_daily_academic_term
    ON class_daily_attendance_summaries (academic_term_id);

DROP TRIGGER IF EXISTS trg_class_daily_attendance_summaries_updated_at
    ON class_daily_attendance_summaries;
CREATE TRIGGER trg_class_daily_attendance_summaries_updated_at
    BEFORE UPDATE ON class_daily_attendance_summaries
    FOR EACH ROW EXECUTE FUNCTION fn_set_updated_at();

COMMENT ON TABLE class_daily_attendance_summaries IS
    'Materialised rollup of attendance records per class per date. Populated
     by incremental background tasks triggered when all attendance for a
     class-date is marked (or on a timeout). "Total enrolled" is derived from
     distinct students who have attendance_records rows that day, not from
     cbc_student_enrollments.status, because enrollment status has no effective
     date within a term — a student suspended on day 50 would otherwise vanish
     from every earlier day too. This is a documented workaround, not a perfect fix.';

COMMENT ON COLUMN class_daily_attendance_summaries.daily_attendance_rate IS
    'Calculated as (present_count / (present_count + absent_count + late_count + excused_count)) * 100,
     stored as a decimal with two fractional digits (e.g. 94.60).';
