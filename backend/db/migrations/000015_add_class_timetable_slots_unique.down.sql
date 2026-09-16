-- Migration: 000015_add_class_timetable_slots_unique
-- Purpose: Revert unique constraint.

ALTER TABLE class_timetable_slots
DROP CONSTRAINT IF EXISTS class_timetable_slots_unique_assignment;
