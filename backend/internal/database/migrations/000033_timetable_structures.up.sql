-- Migration: 000033_timetable_structures
-- SomoTracker — Kenya CBC/CBE academic platform (CBC-only)
-- Split from the squashed 000001_initial_schema on 2026-08-16. DDL unchanged.
-- Purpose: timetable_structures

CREATE TABLE IF NOT EXISTS timetable_structures (
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

    CONSTRAINT chk_timetable_structure_times CHECK (end_time > start_time),
    CONSTRAINT excl_timetable_structure_overlap
        EXCLUDE USING gist (
            school_id WITH =,
            academic_year_id WITH =,
            day_of_week WITH =,
            fn_timerange(day_of_week, start_time, end_time) WITH &&
        ),
    CONSTRAINT fk_timetable_structure_tenant_school
        FOREIGN KEY (tenant_id, school_id)
        REFERENCES cbc_schools(tenant_id, id) ON DELETE CASCADE,
    CONSTRAINT fk_timetable_structure_academic_year
        FOREIGN KEY (tenant_id, academic_year_id)
        REFERENCES academic_years(tenant_id, id) ON DELETE CASCADE,
    -- Composite key so cbc_timetable_slots can reference (tenant_id, structure_id)
    CONSTRAINT uq_timetable_structures_tenant UNIQUE (tenant_id, id)
);

CREATE INDEX IF NOT EXISTS idx_timetable_structure_tenant
    ON timetable_structures (tenant_id);
CREATE INDEX IF NOT EXISTS idx_timetable_structure_school_day
    ON timetable_structures (school_id, day_of_week);
CREATE INDEX IF NOT EXISTS idx_timetable_structure_academic_year
    ON timetable_structures (academic_year_id);

DROP TRIGGER IF EXISTS trg_timetable_structures_updated_at ON timetable_structures;
CREATE TRIGGER trg_timetable_structures_updated_at
    BEFORE UPDATE ON timetable_structures
    FOR EACH ROW EXECUTE FUNCTION fn_set_updated_at();

COMMENT ON TABLE timetable_structures IS
    'Structural day template (Grid Definition Layer). Defines the partitioned time
     blocks (lessons, breaks, assemblies) that make up a standard school day per
     academic year. The GiST exclusion constraint guarantees non-overlapping blocks
     per school per academic year per day. Decoupled from cbc_timetable_slots —
     allocations reference structure_id instead of carrying raw time ranges.';

COMMENT ON COLUMN timetable_structures.day_of_week IS
    '1=Monday, 2=Tuesday, 3=Wednesday, 4=Thursday, 5=Friday, 6=Saturday, 7=Sunday.
     Most schools use Mon-Fri (1-5); weekends are allowed for special sessions.';

COMMENT ON COLUMN timetable_structures.period_name IS
    'Human-readable name for this time period, e.g. "Lesson 1", "Morning Break",
     "Recess", "Assembly". Free-text — not an enum, to support school-specific naming.';

COMMENT ON COLUMN timetable_structures.is_break IS
    'Flags recess, lunch, or other non-instructional blocks. UI can use this to
     disable assignment cells and render break rows in a distinct style.';
