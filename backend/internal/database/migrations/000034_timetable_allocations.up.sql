-- Migration: 000034_timetable_allocations
-- SomoTracker — Kenya CBC/CBE academic platform (CBC-only)
-- Split from the squashed 000001_initial_schema on 2026-08-16. DDL unchanged.
-- Purpose: timetable_allocations

CREATE TABLE IF NOT EXISTS timetable_allocations (
    id                UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id         UUID        NOT NULL,
    school_id         UUID        NOT NULL,
    block_id      UUID        NOT NULL,
    class_id          UUID        NOT NULL,
    learning_area_id  UUID        NOT NULL,
    teacher_id        UUID        NOT NULL,
    room_identifier   VARCHAR(50) NULL,
    created_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    -- CONSTRAINT 1: A class can only have ONE assignment per specific structure block
    CONSTRAINT unique_class_slot
        UNIQUE (tenant_id, block_id, class_id),

    -- CONSTRAINT 2: A teacher cannot be double-booked during the same structure block
    CONSTRAINT unique_teacher_slot
        UNIQUE (tenant_id, block_id, teacher_id),

    -- CONSTRAINT 3: A room cannot be double-booked during the same structure block
    CONSTRAINT unique_room_slot
        UNIQUE (tenant_id, block_id, room_identifier),

    CONSTRAINT fk_timetable_allocations_tenant_school
        FOREIGN KEY (tenant_id, school_id)
        REFERENCES cbc_schools(tenant_id, id) ON DELETE CASCADE,
    CONSTRAINT fk_timetable_allocations_tenant_class
        FOREIGN KEY (tenant_id, class_id)
        REFERENCES cbc_classes(tenant_id, id) ON DELETE CASCADE,
    CONSTRAINT fk_timetable_allocations_tenant_teacher
        FOREIGN KEY (tenant_id, teacher_id)
        REFERENCES users(tenant_id, id) ON DELETE CASCADE,
    CONSTRAINT fk_timetable_allocations_tenant_learning_area
        FOREIGN KEY (tenant_id, learning_area_id)
        REFERENCES cbc_learning_areas(tenant_id, id) ON DELETE CASCADE,
    CONSTRAINT fk_timetable_allocations_tenant_block
        FOREIGN KEY (tenant_id, block_id)
        REFERENCES timetable_blocks(tenant_id, id) ON DELETE CASCADE
);

COMMENT ON TABLE timetable_allocations IS
    'Grid Allocation Layer — lightweight relational mapping table using fast
     B-Tree composite unique constraints. The grid definition (time ranges)
     lives in timetable_blocks; this table only stores assignments of
     class → teacher → learning_area → room per structure block.
     Academic year/term context is inherited via block → track → timetable_tracks.';

DROP TRIGGER IF EXISTS trg_timetable_allocations_updated_at ON timetable_allocations;
CREATE TRIGGER trg_timetable_allocations_updated_at
    BEFORE UPDATE ON timetable_allocations
    FOR EACH ROW EXECUTE FUNCTION fn_set_updated_at();

CREATE INDEX IF NOT EXISTS idx_timetable_allocations_block
    ON timetable_allocations (block_id);
CREATE INDEX IF NOT EXISTS idx_timetable_allocations_class
    ON timetable_allocations (class_id);
CREATE INDEX IF NOT EXISTS idx_timetable_allocations_teacher
    ON timetable_allocations (teacher_id);
CREATE INDEX IF NOT EXISTS idx_timetable_allocations_tenant
    ON timetable_allocations (tenant_id);
CREATE INDEX IF NOT EXISTS idx_timetable_allocations_school
    ON timetable_allocations (school_id);

-- Composite key so cbc_attendance_sessions / attendance_records / behavior_notes
-- can reference (tenant_id, timetable_allocation_id)
CREATE UNIQUE INDEX IF NOT EXISTS uq_timetable_allocations_tenant
    ON timetable_allocations (tenant_id, id);
