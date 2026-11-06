---
id: FEATURE-ATTENDANCE-001
title: Per-Period Attendance
status: draft
depends_on: [FEATURE-SCHOOL-DATES-EVENTS-001, FEATURE-TIMETABLE-001]
touches: []
---

# Feature

## Goal
Enable teachers to mark per-period attendance for their subject offerings, with pre-mark excused from events, day-close enforcement, and audit for edits after close.

## Out of scope
Per-class aggregate attendance, SMS notifications, guardian notifications in v1.

## Domain references
docs/domain/glossary.md
docs/domain/model.md
docs/workflow/notes-calendar-timetable.md
docs/features/FEATURE-SCHOOL-DATES-EVENTS-001.md
docs/features/FEATURE-TIMETABLE-001.md

## Acceptance criteria
- AC-1: Attendance can only be recorded for dates within term and not in `school_off_dates`.
- AC-2: Attendance is per period: mark is tied to `timetable_assignment_id` + `date` + `enrollment_id`.
- AC-3: `GET /teachers/me/timetable?date=` returns teacher's assignments for date with subject, classroom, timeslot, and attendance status pending/partial/complete.
- AC-4: Opening a period returns active enrollments for classroom/term with suggested status from overlapping events.
- AC-5: `POST /attendance/marks/bulk` upserts marks for a `timetable_assignment_id + date`. Default status is Present if not provided.
- AC-6: Day close is end of Africa/Nairobi day. Edits to marks after close require non-empty reason and create audit log entry.
- AC-7: Attendance status enum is Present, Late, Absent, Excused.
- AC-8: Attendance completion is derived: complete when marks exist for all active enrollments for that assignment/date.
- AC-9: All timestamps stored UTC, dates computed in Africa/Nairobi.

## API changes
Pointer to docs/api/openapi.yaml diff: TODO

## Backend notes
Tables:
- `attendance_marks(id uuidv7 PK, timetable_assignment_id uuid FK, enrollment_id uuid FK, date date, status text CHECK, marked_by uuid FK, marked_at timestamptz, reason text nullable, updated_at timestamptz, UNIQUE(timetable_assignment_id, enrollment_id, date))`
- `attendance_audit_log(id uuidv7 PK, attendance_marks_id uuid FK, old_status text, new_status text, changed_by uuid, changed_at timestamptz, reason text)`

Validation:
- Assignment belongs to teacher via subject_offering.teacher_id.
- Date within term and not school_off_dates.
- Day close check on update.

## Frontend notes
- Teacher timetable view shows next subject, attendance status badge.
- Period attendance screen shows student list, default Present, ticks for Late/Absent, pre-filled Excused from events.
- Submit button posts bulk marks.
- After day close, edit requires reason modal.

## Verification
- `make compliance` passes.
- Unit tests for day close edit reason requirement.
- Unit tests for attendance status derivation.
- Integration test for bulk upsert and audit log creation.
- Manual test: mark attendance, close day, attempt edit without reason → reject.

## Open questions
- Should attendance marks be pre-created for all students at period open, or created on submit only?
