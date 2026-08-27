-- Migration: 000065 (rollback)
-- Restores unique_class_slot on timetable_allocations.
-- Note: if any duplicate (tenant_id, block_id, class_id) rows exist, ADD CONSTRAINT will fail.
-- Clean them up before running the rollback.

ALTER TABLE timetable_allocations
    ADD CONSTRAINT unique_class_slot
    UNIQUE (tenant_id, block_id, class_id);
