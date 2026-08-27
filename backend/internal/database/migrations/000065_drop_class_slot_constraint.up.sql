-- Migration: 000065_drop_class_slot_constraint
-- SomoTracker — drop unique_class_slot constraint from timetable_allocations
-- Rationale: a class can have multiple learning areas (subjects) in the same block.
-- Only teacher double-booking (unique_teacher_slot) and room double-booking (unique_room_slot)
-- remain as hard DB-level constraints.
-- Apply after: 000035_timetable_allocations (table exists with constraint)
-- Rollback: see 000065_drop_class_slot_constraint.down.sql

ALTER TABLE timetable_allocations
    DROP CONSTRAINT IF EXISTS unique_class_slot;
