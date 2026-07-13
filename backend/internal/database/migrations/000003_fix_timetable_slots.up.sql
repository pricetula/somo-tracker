-- Migration: 000003_fix_timetable_slots
-- SomoTracker — adds missing columns, foreign keys, and audit fields to
-- cbc_timetable_slots for proper tenant isolation and data integrity.
--
-- This migration is designed to be safe whether applied against the old
-- 000001 schema (without tenant_id/school_id/updated_at) or the updated
-- 000001 schema (which includes them). Uses IF NOT EXISTS / DROP THEN ADD
-- patterns to be idempotent.

-- ============================================================================
-- 1. Add missing columns (idempotent — IF NOT EXISTS)
-- ============================================================================

ALTER TABLE cbc_timetable_slots
    ADD COLUMN IF NOT EXISTS tenant_id UUID,
    ADD COLUMN IF NOT EXISTS school_id UUID,
    ADD COLUMN IF NOT EXISTS updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW();

-- ============================================================================
-- 2. Backfill tenant_id and school_id only if they were just added (NULL rows)
-- ============================================================================

UPDATE cbc_timetable_slots sl
SET
    tenant_id = ts.tenant_id,
    school_id = ts.school_id
FROM timetable_structures ts
WHERE ts.id = sl.structure_id
  AND sl.tenant_id IS NULL;

-- ============================================================================
-- 3. Make columns NOT NULL (safe to re-run — ALTER COLUMN SET NOT NULL on an
--    already-NOT-NULL column is a no-op)
-- ============================================================================

ALTER TABLE cbc_timetable_slots
    ALTER COLUMN tenant_id SET NOT NULL,
    ALTER COLUMN school_id SET NOT NULL;

-- ============================================================================
-- 4. Add foreign key constraints (safe — drop first if exists)
-- ============================================================================

ALTER TABLE cbc_timetable_slots
    DROP CONSTRAINT IF EXISTS fk_cbc_timetable_slots_tenant_school,
    DROP CONSTRAINT IF EXISTS fk_cbc_timetable_slots_tenant_class,
    DROP CONSTRAINT IF EXISTS fk_cbc_timetable_slots_tenant_teacher,
    DROP CONSTRAINT IF EXISTS fk_cbc_timetable_slots_academic_year;

ALTER TABLE cbc_timetable_slots
    ADD CONSTRAINT fk_cbc_timetable_slots_tenant_school
        FOREIGN KEY (tenant_id, school_id)
        REFERENCES cbc_schools(tenant_id, id) ON DELETE CASCADE,
    ADD CONSTRAINT fk_cbc_timetable_slots_tenant_class
        FOREIGN KEY (tenant_id, class_id)
        REFERENCES cbc_classes(tenant_id, id) ON DELETE CASCADE,
    ADD CONSTRAINT fk_cbc_timetable_slots_tenant_teacher
        FOREIGN KEY (tenant_id, teacher_id)
        REFERENCES users(tenant_id, id) ON DELETE SET NULL,
    ADD CONSTRAINT fk_cbc_timetable_slots_academic_year
        FOREIGN KEY (academic_year_id)
        REFERENCES academic_years(id) ON DELETE CASCADE;

-- ============================================================================
-- 5. Add updated_at trigger (safe — DROP THEN CREATE)
-- ============================================================================

DROP TRIGGER IF EXISTS trg_cbc_timetable_slots_updated_at ON cbc_timetable_slots;
CREATE TRIGGER trg_cbc_timetable_slots_updated_at
    BEFORE UPDATE ON cbc_timetable_slots
    FOR EACH ROW EXECUTE FUNCTION fn_set_updated_at();

-- ============================================================================
-- 6. Add indexes (safe — IF NOT EXISTS)
-- ============================================================================

CREATE INDEX IF NOT EXISTS idx_cbc_timetable_slots_tenant ON cbc_timetable_slots (tenant_id);
CREATE INDEX IF NOT EXISTS idx_cbc_timetable_slots_school ON cbc_timetable_slots (school_id);

