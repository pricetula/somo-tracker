-- Migration: 000033_timetable_blocks
-- SomoTracker — Kenya CBC/CBE academic platform (CBC-only)
-- Split from the squashed 000001_initial_schema on 2026-08-16. DDL unchanged.
-- Purpose: timetable_blocks

CREATE TABLE IF NOT EXISTS timetable_blocks (
    id               UUID             PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id        UUID             NOT NULL,
    school_id        UUID             NOT NULL,
    academic_year_id UUID             NOT NULL,
    day_of_week      INT              NOT NULL CHECK (day_of_week BETWEEN 1 AND 7),
    period_name      VARCHAR(50)      NOT NULL,
    start_time       TIME             NOT NULL,
    end_time         TIME             NOT NULL,
    is_break         BOOLEAN          NOT NULL DEFAULT FALSE,
    created_at       TIMESTAMPTZ      NOT NULL DEFAULT NOW(),
    updated_at       TIMESTAMPTZ      NOT NULL DEFAULT NOW(),

    CONSTRAINT chk_timetable_block_times CHECK (end_time > start_time),
    CONSTRAINT excl_timetable_block_overlap
        EXCLUDE USING gist (
            school_id WITH =,
            academic_year_id WITH =,
            day_of_week WITH =,
            fn_timerange(day_of_week, start_time, end_time) WITH &&
        ),
    CONSTRAINT fk_timetable_block_tenant_school
        FOREIGN KEY (tenant_id, school_id)
        REFERENCES cbc_schools(tenant_id, id) ON DELETE CASCADE,
    CONSTRAINT fk_timetable_block_academic_year
        FOREIGN KEY (tenant_id, academic_year_id)
        REFERENCES academic_years(tenant_id, id) ON DELETE CASCADE,
    -- Composite key so timetable_allocations can reference (tenant_id, block_id)
    CONSTRAINT uq_timetable_blocks_tenant UNIQUE (tenant_id, id)
);

CREATE INDEX IF NOT EXISTS idx_timetable_block_tenant
    ON timetable_blocks (tenant_id);
CREATE INDEX IF NOT EXISTS idx_timetable_block_school_day
    ON timetable_blocks (school_id, day_of_week);
CREATE INDEX IF NOT EXISTS idx_timetable_block_academic_year
    ON timetable_blocks (academic_year_id);

DROP TRIGGER IF EXISTS trg_timetable_blocks_updated_at ON timetable_blocks;
CREATE TRIGGER trg_timetable_blocks_updated_at
    BEFORE UPDATE ON timetable_blocks
    FOR EACH ROW EXECUTE FUNCTION fn_set_updated_at();

COMMENT ON TABLE timetable_blocks IS
    'Structural day template (Grid Definition Layer). Defines the partitioned time
     blocks (lessons, breaks, assemblies) that make up a standard school day per
     academic year. The GiST exclusion constraint guarantees non-overlapping blocks
     per school per academic year per day. Decoupled from timetable_allocations —
     allocations reference block_id instead of carrying raw time ranges.';

COMMENT ON COLUMN timetable_blocks.day_of_week IS
    '1=Monday, 2=Tuesday, 3=Wednesday, 4=Thursday, 5=Friday, 6=Saturday, 7=Sunday.
     Most schools use Mon-Fri (1-5); weekends are allowed for special sessions.';

COMMENT ON COLUMN timetable_blocks.period_name IS
    'Human-readable name for this time period, e.g. "Lesson 1", "Morning Break",
     "Recess", "Assembly". Free-text — not an enum, to support school-specific naming.';

COMMENT ON COLUMN timetable_blocks.is_break IS
    'Flags recess, lunch, or other non-instructional blocks. UI can use this to
     disable assignment cells and render break rows in a distinct style.';
