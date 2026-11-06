---
id: FEATURE-TIMETABLE-001
title: Timetable Management
status: draft
depends_on: [FEATURE-SCHOOL-DATES-EVENTS-001]
touches: []
---

# Feature

## Goal
Enable schools to define a master time slot template and assign lessons to classrooms per term, with teacher and classroom clash prevention, to drive attendance eligibility and assessment date validation.

## Out of scope
Per-period attendance, room booking, multiple teachers per subject offering, timetable variants per stream in v1.

## Domain references
docs/domain/glossary.md
docs/domain/model.md
docs/features/FEATURE-SCHOOL-DATES-EVENTS-001.md

## Acceptance criteria
- AC-1: `timetable_templates` supports CRUD per school with fields `school_id, timeslot_number, start_time, end_time, is_lesson boolean, name`. Unique `(school_id, timeslot_number)`.
- AC-2: `timetable_assignments` supports CRUD per school/term/classroom with fields `school_id, academic_year_id, term_id, classroom_id, day_of_week int 0-6, timeslot_number, subject_offering_id, room_id nullable`. Assignment only allowed if template.is_lesson=true.
- AC-3: Classroom clash prevention: unique `(school_id, academic_year_id, term_id, classroom_id, day_of_week, timeslot_number)`.
- AC-4: Teacher clash prevention: creating/updating assignment rejects if teacher from `subject_offering.teacher_id` already has an assignment in same school/term/day/timeslot in another classroom.
- AC-5: `GET /schools/:id/timetable-templates` returns ordered slots.
- AC-6: `GET /schools/:id/classrooms/:id/timetable?academic_year_id=&term_id=` returns assignments grouped by day and timeslot.
- AC-7: Attendance eligibility for a classroom on a date is true if date is within term, not in `school_off_dates`, and classroom has at least one assignment for that weekday.
- AC-8: Assessment date validation: assessment date must be within term and on a weekday where subject_offering has at least one assignment in that term.
- AC-9: All times stored as time without tz, interpreted in Africa/Nairobi. DB stores UTC for timestamps.

## API changes
Pointer to docs/api/openapi.yaml diff: TODO

## Backend notes
Tables:
- `timetable_templates(school_id uuid FK, timeslot_number int, start_time time, end_time time, is_lesson boolean, name text, created_at timestamptz, updated_at timestamptz, unique(school_id, timeslot_number))`
- `timetable_assignments(id uuidv7 PK, school_id uuid, academic_year_id uuid FK, term_id uuid FK, classroom_id uuid FK, day_of_week int CHECK 0-6, timeslot_number int, subject_offering_id uuid FK, room_id uuid nullable, created_at timestamptz, updated_at timestamptz, unique(school_id, academic_year_id, term_id, classroom_id, day_of_week, timeslot_number))`
- Service validation:
  * Template is_lesson check before assignment.
  * Teacher clash query: exists assignment where school_id, academic_year_id, term_id, day_of_week, timeslot_number match and subject_offering.teacher_id = current teacher and classroom_id != current.
- RLS via school_id.

## Frontend notes
- Admin UI for timetable template management per school.
- Classroom timetable grid view by week, timeslot rows, day columns.
- Assignment creation via drag-drop or form with subject offering picker filtered by classroom/academic year.
- Clash warnings inline on save.

## Verification
- `make compliance` passes.
- Unit tests for template uniqueness and is_lesson guard.
- Unit tests for classroom and teacher clash prevention.
- Integration test for timetable endpoint returning correct grid.
- Manual test: create overlapping teacher assignment, verify rejection.

## Open questions
- Should timetable template be versioned per academic year or shared school-wide?
- Room support required for v1?
