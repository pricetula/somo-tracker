# Feature: Student Timetable View

**Status:** Done
**Owner:** Product / Platform
**Created:** 2026-10-15
**Last Updated:** 2026-10-15
**Related Issues:** #
**Affected Layers:** backend | frontend | docs

## Summary
Provide students with a personal weekly timetable derived from their class enrollments and class timetable slots for the current academic term.

## Current State
**Frontend:** No student-facing timetable page exists. Student routes are `/students`, `/students/[id]`, `/students/add`. No timetable component in `frontend/src/features/students/`.
**Backend:** No API endpoint exposes a student's personal timetable. `class_timetable_slots` exists and `GetClassTimetableSlotsByTemplate` returns slots per class, but no service assembles slots per student by resolving enrollments.
**Docs:** Gap documented in `docs/gaps/GAP-student-timetable-view.md`.

## Problem Statement
Students cannot view their weekly schedule in-app. They must rely on paper timetables or ask teachers. Teachers and admins also lack a quick student-centric view to verify schedule.

## Solution Overview
Add a read-only weekly grid for a student, resolved server-side from enrollments → classes → class_timetable_slots for current term.

### User Stories
- As a Student, I want to see my weekly timetable so I know where I should be and when.
- As a Teacher, I want to view a student's timetable to understand their schedule load.
- As an Admin, I want to verify student schedules without opening multiple class pages.

### Acceptance Criteria
- [x] `GET /api/students/:id/timetable` returns weekly grid for current academic term
- [x] Response includes day, time slot, subject, teacher name, room name, class name
- [x] Frontend page `/students/[id]/timetable` renders weekly grid
- [x] Grid updates when enrollment changes
- [x] Empty state shown if student has no enrollments
- [x] RLS enforced, student can only view own timetable or authorized users can view any

## Scope

### Backend
- API changes:
  - New endpoint `GET /api/students/:id/timetable`
  - Service method `GetStudentTimetable(ctx, schoolID, studentID)`:
    1. Resolve current academic term via `AcademicPeriodResolver`
    2. Fetch active enrollments for student in current term
    3. For each enrolled class, fetch class timetable slots for term
    4. Aggregate into weekly structure
- DB changes: No schema change required
- Performance: Consider caching per student per term

### Frontend
- Page: `/students/[id]/timetable` or tab inside student detail
- Component: Weekly grid using existing timetable grid components
- No dumb back buttons

### Non-Goals
- Student editing timetable
- Substitute timetable view
- Mobile push notifications

## Design Notes
Reuse `TimetableGrid` component from `frontend/src/features/timetable/` with read-only mode.

## Open Gaps / Risks
- Attendance data not needed for v1
- Large enrollment sets may cause N+1 queries

## Test Plan
- API contract test for student with multiple enrollments
- E2E for student viewing own timetable

## Rollout
- Backend first, frontend second

## References
- Gap: `docs/gaps/GAP-student-timetable-view.md`
- Service: `backend/internal/services/timetable_service.go`
- Frontend: `frontend/src/features/timetable/components/timetable-grid.tsx`
