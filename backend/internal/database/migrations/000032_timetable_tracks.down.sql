-- ============================================================================
-- Downgrade Migration: Remove Timetable Tracks & Revert Time Blocks
-- Description: Rolls back the introduction of timetable tracks, restores
--              school-wide global overlap checking on time blocks, and drops tracks.
-- ============================================================================

-- 1. Drop timetable_tracks and its dependencies
-- ============================================================================

-- Drop trigger
DROP TRIGGER IF EXISTS trg_timetable_tracks_updated_at ON timetable_tracks;

-- Drop RLS policy
DROP POLICY IF EXISTS tenant_isolation_policy ON timetable_tracks;

-- Drop indexes
DROP INDEX IF EXISTS idx_timetable_tracks_tenant;
DROP INDEX IF EXISTS idx_timetable_tracks_school_year;

-- Drop the table itself
DROP TABLE IF EXISTS timetable_tracks;
