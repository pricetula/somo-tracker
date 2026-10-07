-- Migration: 000015_add_class_timetable_slots_unique
-- Purpose: Prevent duplicate timetable assignments per class/term/day/slot.

ALTER TABLE class_timetable_slots
ADD CONSTRAINT class_timetable_slots_unique_assignment
UNIQUE (school_id, class_room_id, academic_term_id, day_of_week, time_slot_id);
