-- Migration: 000034_cbc_timetable_slots
-- SomoTracker — Kenya CBC/CBE academic platform (CBC-only)
-- Split from the squashed 000001_initial_schema on 2026-08-16. DDL unchanged.
-- Purpose: cbc_timetable_slots

CREATE TABLE IF NOT EXISTS cbc_timetable_slots (
    id                UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id         UUID        NOT NULL,
    school_id         UUID        NOT NULL,
    academic_year_id  UUID        NOT NULL,
    structure_id      UUID        NOT NULL,
    class_id          UUID        NOT NULL,
    learning_area_id  UUID        NOT NULL,
    teacher_id        UUID        NOT NULL,
    room_identifier   VARCHAR(50) NULL,
    created_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    -- CONSTRAINT 1: A class can only have ONE assignment per specific structure block
    CONSTRAINT unique_class_slot
        UNIQUE (academic_year_id, structure_id, class_id),

    -- CONSTRAINT 2: A teacher cannot be double-booked during the same structure block
    CONSTRAINT unique_teacher_slot
        UNIQUE (academic_year_id, structure_id, teacher_id),

    -- CONSTRAINT 3: A room cannot be double-booked during the same structure block
    CONSTRAINT unique_room_slot
        UNIQUE (academic_year_id, structure_id, room_identifier),

    CONSTRAINT fk_cbc_timetable_slots_tenant_school
        FOREIGN KEY (tenant_id, school_id)
        REFERENCES cbc_schools(tenant_id, id) ON DELETE CASCADE,
    CONSTRAINT fk_cbc_timetable_slots_tenant_class
        FOREIGN KEY (tenant_id, class_id)
        REFERENCES cbc_classes(tenant_id, id) ON DELETE CASCADE,
    CONSTRAINT fk_cbc_timetable_slots_tenant_teacher
        FOREIGN KEY (tenant_id, teacher_id)
        REFERENCES users(tenant_id, id) ON DELETE CASCADE,
    CONSTRAINT fk_cbc_timetable_slots_tenant_learning_area
        FOREIGN KEY (tenant_id, learning_area_id)
        REFERENCES cbc_learning_areas(tenant_id, id) ON DELETE CASCADE,
    CONSTRAINT fk_cbc_timetable_slots_tenant_structure
        FOREIGN KEY (tenant_id, structure_id)
        REFERENCES timetable_structures(tenant_id, id) ON DELETE CASCADE,
    CONSTRAINT fk_cbc_timetable_slots_academic_year
        FOREIGN KEY (tenant_id, academic_year_id)
        REFERENCES academic_years(tenant_id, id) ON DELETE CASCADE
);

COMMENT ON TABLE cbc_timetable_slots IS
    'Grid Allocation Layer — lightweight relational mapping table using fast
     B-Tree composite unique constraints. The grid definition (time ranges)
     lives in timetable_structures; this table only stores assignments of
     class → teacher → learning_area → room per structure block.';

DROP TRIGGER IF EXISTS trg_cbc_timetable_slots_updated_at ON cbc_timetable_slots;
CREATE TRIGGER trg_cbc_timetable_slots_updated_at
    BEFORE UPDATE ON cbc_timetable_slots
    FOR EACH ROW EXECUTE FUNCTION fn_set_updated_at();

CREATE INDEX IF NOT EXISTS idx_cbc_timetable_slots_structure
    ON cbc_timetable_slots (structure_id);
CREATE INDEX IF NOT EXISTS idx_cbc_timetable_slots_class
    ON cbc_timetable_slots (class_id);
CREATE INDEX IF NOT EXISTS idx_cbc_timetable_slots_teacher
    ON cbc_timetable_slots (teacher_id);
CREATE INDEX IF NOT EXISTS idx_cbc_timetable_slots_academic_year
    ON cbc_timetable_slots (academic_year_id);
CREATE INDEX IF NOT EXISTS idx_cbc_timetable_slots_tenant
    ON cbc_timetable_slots (tenant_id);
CREATE INDEX IF NOT EXISTS idx_cbc_timetable_slots_school
    ON cbc_timetable_slots (school_id);

-- Composite key so cbc_attendance_sessions / attendance_records / behavior_notes
-- can reference (tenant_id, timetable_slot_id)
CREATE UNIQUE INDEX IF NOT EXISTS uq_cbc_timetable_slots_tenant
    ON cbc_timetable_slots (tenant_id, id);
