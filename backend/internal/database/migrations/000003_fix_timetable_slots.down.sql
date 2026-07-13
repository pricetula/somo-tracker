-- Migration: 000003_fix_timetable_slots (down)

DROP TRIGGER IF EXISTS trg_cbc_timetable_slots_updated_at ON cbc_timetable_slots;

ALTER TABLE cbc_timetable_slots
    DROP CONSTRAINT IF EXISTS fk_cbc_timetable_slots_tenant_school,
    DROP CONSTRAINT IF EXISTS fk_cbc_timetable_slots_tenant_class,
    DROP CONSTRAINT IF EXISTS fk_cbc_timetable_slots_tenant_teacher,
    DROP CONSTRAINT IF EXISTS fk_cbc_timetable_slots_academic_year;

DROP INDEX IF EXISTS idx_cbc_timetable_slots_tenant;
DROP INDEX IF EXISTS idx_cbc_timetable_slots_school;

ALTER TABLE cbc_timetable_slots
    DROP COLUMN IF EXISTS tenant_id,
    DROP COLUMN IF EXISTS school_id,
    DROP COLUMN IF EXISTS updated_at;

