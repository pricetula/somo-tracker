-- Migration: 000044_grading_scale_ranges (rollback)
-- SomoTracker — rollback for 000044_grading_scale_ranges.

DROP FUNCTION IF EXISTS fn_block_grading_scale_range_modification CASCADE;

DROP TABLE IF EXISTS grading_scale_ranges CASCADE;
