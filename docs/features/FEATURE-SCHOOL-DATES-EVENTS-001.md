---
id: FEATURE-SCHOOL-DATES-EVENTS-001
title: School Off Dates and Events
status: draft
depends_on: []
touches: []
---

# Feature

## Goal
Enable schools to manage school-wide off dates and per-day events for select students/teachers so attendance eligibility, day close, and assessment date validation can be computed correctly, and students can be pre-marked excused for events.

## Out of scope
Timetable, per-period attendance, SMS notifications, country holiday inheritance.

## Domain references
docs/domain/glossary.md
docs/domain/model.md

## Acceptance criteria
- AC-1: `school_off_dates` supports create/update/delete per school with fields `school_id, date, off_type, note`. Presence of row means date is off.
- AC-2: `off_type` is one of `holiday, closure, inset`. Validation rejects dates outside any term for the school.
- AC-3: API `GET /schools/:id/off-dates?from=&to=` returns sparse list of off dates in range, dates are `yyyy-MM-dd`.
- AC-4: `school_events` supports create/update/delete per school with fields `id, school_id, date, start_time, end_time, title, event_type, reason, affects_attendance boolean`.
- AC-5: `school_event_students` and `school_event_teachers` support many-to-many linking. Linking a student to two events on same date with overlapping times is rejected.
- AC-6: `GET /schools/:id/calendar?from=&to=` returns per date `is_off` and `has_event` derived from `school_off_dates` and `school_events` using generate_series, no pre-population.
- AC-7: Attendance service treats a date as non-attendance day if it is in `school_off_dates` for the school.
- AC-8: When `affects_attendance=true`, attendance roll-call defaults student status to `excused` with reason from event if student is linked to event on that date.
- AC-9: All timestamps stored UTC, dates displayed and computed in Africa/Nairobi.
- AC-10: Day close uses Africa/Nairobi end of day, UTC in DB.

## API changes
Pointer to docs/api/openapi.yaml diff: TODO

## Backend notes
- Tables:
  - `school_off_dates(school_id uuid FK schools, date date, off_type text, note text, created_at timestamptz, updated_at timestamptz, unique(school_id,date))`
  - `school_events(id uuid, school_id uuid, date date, start_time time, end_time time, title text, event_type text, reason text, affects_attendance boolean default true, created_at timestamptz, updated_at timestamptz)`
  - `school_event_students(event_id uuid, student_id uuid, unique(event_id,student_id))`
  - `school_event_teachers(event_id uuid, teacher_id uuid, unique(event_id,teacher_id))`
- Conflict check for overlapping events per student on same date: query existing events for student on date where time ranges overlap.
- `is_school_day` for attendance will later be driven by timetable existence; `school_off_dates` is a hard block.
- No `has_event` column stored, derived.

## Frontend notes
- Calendar month view loads `GET /schools/:id/calendar?from=&to=` and paints holiday/off indicator and event indicator per date.
- Off dates CRUD UI in school admin settings.
- Event creation UI with date picker, start/end time, title, event_type, reason, affects_attendance toggle, multi-select students/teachers.
- Conflict warning shown if selected student already has overlapping event.

## Verification
- `make compliance` passes.
- Unit tests for off date validation within term.
- Unit tests for event overlap rejection per student.
- Integration test for `GET /schools/:id/calendar` returning correct `is_off`/`has_event` flags.
- Manual test: create holiday, verify attendance blocked; create event with student, verify roll-call defaults excused.

## Open questions
- Should `assessment_dates` table be part of this feature or separate?
- Multi-day event editing UX: per-day rows vs create multiple days.
