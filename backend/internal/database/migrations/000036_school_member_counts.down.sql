-- Migration: 000036_school_member_counts (rollback)
-- SomoTracker — rollback for 000036_school_member_counts.

DROP FUNCTION IF EXISTS fn_sync_school_staff_counts_insert CASCADE;
DROP FUNCTION IF EXISTS fn_sync_school_staff_counts_delete CASCADE;
DROP FUNCTION IF EXISTS fn_sync_school_staff_counts_update CASCADE;
DROP FUNCTION IF EXISTS fn_sync_school_student_counts_insert CASCADE;
DROP FUNCTION IF EXISTS fn_sync_school_student_counts_delete CASCADE;
DROP FUNCTION IF EXISTS fn_sync_school_student_counts_update CASCADE;

DROP TABLE IF EXISTS school_member_counts CASCADE;
