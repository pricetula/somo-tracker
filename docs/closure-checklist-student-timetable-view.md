# Closure Checklist – Student Timetable View

Feature: `docs/features/FEATURE-student-timetable-view.md`
Gap: `docs/gaps/GAP-student-timetable-view.md`

## Definition of Done

- [ ] Acceptance Criteria all checked
- [ ] Backend endpoint `GET /api/students/:id/timetable` implemented
- [ ] Service resolves current academic term via `AcademicPeriodResolver`
- [ ] Fetches active enrollments, aggregates class timetable slots
- [ ] Response includes day, time slot, subject, teacher name, room name, class name
- [ ] RLS enforced: student can view own timetable or authorized users
- [ ] Frontend page `/students/[id]/timetable` renders weekly grid read-only
- [ ] Reuses `TimetableGrid` component, no tabs, no dumb back buttons
- [ ] Empty state shown when no enrollments
- [ ] Tests pass
  - [ ] API contract test for student with multiple enrollments
  - [ ] E2E test for student viewing own timetable
  - [ ] RLS test
- [ ] Error contract validated
- [ ] Compliance passed via `/review`
- [ ] Docs synced: Feature status `Done`, Gap status `Done` with Closed date

## Current State
Status reverted to `In Progress` on 2026-10-15 due to unchecked acceptance criteria.

## Next Steps
1. Implement backend service and endpoint if missing
2. Implement frontend page
3. Tick acceptance criteria
4. Run tests and reviews
5. Sync GAP and mark Done
