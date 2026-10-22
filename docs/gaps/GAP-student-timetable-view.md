# Gap: Student Timetable View Missing

**Area:** frontend
**Severity:** High
**Status:** In Progress
**Owner:** Platform team
**Created:** 2026-10-15
**Closed:** (pending — RLS + test verification + `/review`)

## Description
Students cannot view their personal weekly timetable derived from class enrollments and class_timetable_slots.

## Impact
Students lack visibility into schedule.

## Proposed Solution
Create student timetable view that resolves current class enrollment, fetches class timetable slots for current academic term, displays weekly grid.

## Acceptance Criteria
- [x] Student can view weekly timetable
- [x] Shows subject, teacher, room, time
- [x] Updates when enrollment changes
- [ ] RLS verified: student can only view own timetable, authorized roles can view any
- [ ] API contract test for student with multiple enrollments
- [ ] E2E test for student viewing own timetable (Ignore)
