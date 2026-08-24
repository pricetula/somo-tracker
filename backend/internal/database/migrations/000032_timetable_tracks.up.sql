-- ============================================================================
-- Migration: Add Timetable Tracks & Refactor Time Blocks
-- Description: Introduces parallel scheduling tracks (e.g., Lower Primary,
--              Upper Primary, JSS) and scopes timetable blocks per track.
-- ============================================================================

-- 1. Create the timetable_tracks table
CREATE TABLE timetable_tracks (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    school_id UUID NOT NULL,
    academic_year_id UUID NOT NULL,
    name VARCHAR(100) NOT NULL, -- e.g., "Lower Primary Track", "JSS Track"
    description TEXT NULL,
    is_default BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    -- Constraints & Foreign Keys
    CONSTRAINT fk_timetable_tracks_tenant_school
        FOREIGN KEY (tenant_id, school_id)
        REFERENCES cbc_schools(tenant_id, id) ON DELETE CASCADE,

    CONSTRAINT fk_timetable_tracks_academic_year
        FOREIGN KEY (tenant_id, academic_year_id)
        REFERENCES academic_years(tenant_id, id) ON DELETE CASCADE,

    CONSTRAINT uq_timetable_tracks_tenant UNIQUE (tenant_id, id),

    CONSTRAINT uq_timetable_tracks_name_per_year
        UNIQUE (tenant_id, school_id, academic_year_id, name)
);

-- Comments for documentation
COMMENT ON TABLE timetable_tracks IS 'Defines parallel bell-schedule tracks within a school and academic year (e.g., Lower Primary vs. JSS), allowing different sections to run independent time structures concurrently.';

-- 2. Indexes for timetable_tracks
CREATE INDEX idx_timetable_tracks_tenant ON timetable_tracks(tenant_id);
CREATE INDEX idx_timetable_tracks_school_year ON timetable_tracks(school_id, academic_year_id);

-- 3. Row-Level Security (RLS) for timetable_tracks
ALTER TABLE timetable_tracks ENABLE ROW LEVEL SECURITY;

CREATE POLICY tenant_isolation_policy ON timetable_tracks
    FOR ALL
    USING (tenant_id = fn_current_tenant_id())
    WITH CHECK (tenant_id = fn_current_tenant_id());

-- 4. Updated_at Trigger
CREATE TRIGGER trg_timetable_tracks_updated_at
    BEFORE UPDATE ON timetable_tracks
    FOR EACH ROW
    EXECUTE FUNCTION fn_set_updated_at();
