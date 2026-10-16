# Feature: Timetable Conflict Detection

**Status:** In Progress
**Owner:** Platform Team
**Created:** 2026-10-15
**Last Updated:** 2026-10-15
**Related Issues:** #
**Affected Layers:** backend | frontend | docs

## Summary
Prevent invalid class timetable assignments by detecting teacher double-booking, room double-booking, and class overlapping slots at creation/update time, returning a typed error with details.

## Rationale
**Why now:** Without server-side validation, schools can publish timetables with teachers in two places at once and rooms over-booked, leading to operational failure on day one of term.
**Business outcome:** Data integrity for the core scheduling workflow, reduced manual corrections, increased trust in Somotracker vs spreadsheets.
**Success metrics:**
- Zero double-booked teacher/room/class slots created in production
- Conflict detection latency < 100ms p95
- 409 error handling coverage 100% in assign flow

## Current State
**Backend:** `SetupClassTimetableSlot` in `backend/internal/services/timetable_service.go` parses IDs, resolves academic period, and creates `class_timetable_slot` in a transaction. No conflict checks exist for teacher, room, or class. Validation only checks `day_of_week` range 1-7.
**Frontend:** `frontend/src/features/timetable/components/assign-slot-form.tsx` exists with Subject/Teacher/Room comboboxes and `useAssignSlot` mutation. Error handling only maps `ApiError` 400 errors to form fields; no handling for 409 conflict errors. Weekly matrix UI supports assign/edit.
**Docs:** Gap documented in `docs/gaps/GAP-timetable-conflict-detection.md`.

## Problem Statement
* Context: `SetupClassTimetableSlot` allows assigning a subject/teacher/room to a class for a specific day and time slot.
* Pain: No validation exists today. Admins can assign the same teacher to two different classes at the same time, double-book a room, or assign overlapping slots to the same class.
* Business impact: Operational chaos on school days, loss of trust in system, manual corrections.

## Solution Overview
Add server-side validation in the Timetable service before persisting `class_timetable_slots`. Validation runs in the same transaction as insert/update, with RLS enforced. On conflict, return canonical error `timetable_conflict` with structured details.

### User Stories
- As a School Admin, I want the system to block teacher double-booking so that schedules are feasible.
- As a School Admin, I want the system to block room double-booking so that rooms are not over-allocated.
- As a School Admin, I want clear conflict messages so I can fix the assignment quickly.
- As a System, I want conflicts detected at write time to maintain data integrity.

### Acceptance Criteria
- [ ] `POST /api/timetable/setup` validates teacher availability per day + time + academic term
- [ ] `POST /api/timetable/setup` validates room availability per day + time + academic term
- [ ] `POST /api/timetable/setup` validates class does not already have a slot for same day + time
- [ ] On conflict, API returns 409 with body:
  ```json
  {
    "code": "timetable_conflict",
    "message": "Timetable conflict detected",
    "errors": {
      "conflicts": [
        {
          "type": "teacher_double_book",
          "resource_id": "uuid",
          "resource_name": "Teacher Name",
          "existing_slot_id": "uuid"
        }
      ]
    }
  }
  ```
- [ ] Validation respects current academic term resolution; never uses term_id from client
- [ ] Validation is tenant-isolated via RLS

## Scope

### Backend
- API changes:
  - No new endpoint. Enhance existing `POST /api/timetable/setup` handler `Timetable.SetupClassTimetableSlot`
  - `PATCH /api/timetable/templates/:id` and template slot updates should also trigger validation if they affect class assignments
- Service changes:
  - Add `ValidateTimetableSlot` in `internal/services/timetable_service.go`
  - Checks:
    1. Teacher: `SELECT id FROM class_timetable_slots WHERE school_id = ? AND academic_term_id = ? AND day_of_week = ? AND time_slot_id = ? AND teacher_id = ? AND id != ?`
    2. Room: same query with `room_id`
    3. Class: same query with `class_id`
- DB changes / migrations:
  - No schema change required. Ensure indexes exist:
    - `class_timetable_slots(school_id, academic_term_id, day_of_week, time_slot_id, teacher_id)`
    - `class_timetable_slots(school_id, academic_term_id, day_of_week, time_slot_id, room_id)`
    - `class_timetable_slots(school_id, academic_term_id, day_of_week, time_slot_id, class_id)`
- Error contract compliance: see `backend/internal/api/errors.go`. Code `timetable_conflict` is snake_case.
- RLS / multi-tenancy: All queries filtered by `school_id` from `app.current_tenant_id`. No client-supplied school/term IDs.

### Frontend
- Pages / flows:
  - Timetable assignment modal `frontend/src/features/timetable/components/assign-slot-form.tsx`
  - Weekly matrix grid
- Behavior:
  - On 409 `timetable_conflict`, surface error messages inline next to form fields
  - Disable Save when validation fails, show toast with `getErrorMessage`
  - No dumb back buttons
- No UI redesign required, only error handling.

### Public / Marketing
- No changes

### Non-Goals
- Automatic rescheduling or suggestions
- Bulk conflict resolution UI
- Historical conflict audit log v1
- Conflict detection for substitutions – out of scope for v1

## Design Notes
- Academic term resolution uses `internal/services.AcademicPeriodResolver`. Writes during `break` status target next term automatically.
- Conflict detection runs before insert/update inside same transaction to avoid race conditions.
- Consider using `SELECT ... FOR UPDATE` on conflicting rows for high contention schools.

## Open Gaps / Risks
- Performance: per-write checks with 3 queries. Mitigate with indexes.
- Race condition between concurrent assignments — use transaction isolation and unique partial indexes as defense in depth.
- Teacher vs subject vs class teacher mapping may need clarification.

## Test Plan
- Unit tests: service validation for each conflict type
- Integration tests: create conflicting slots and assert 409 with correct `code` and `errors.conflicts`
- RLS tests: ensure tenant cannot see other school conflicts
- Frontend E2E: assign conflicting teacher, assert error banner appears, assignment not persisted

### Test Matrix
| ID | Scope | Given | When | Then |
|----|-------|-------|------|------|
| T-01 | Teacher double-book | Teacher A assigned Mon 08:00-09:00 Class 1 | POST /timetable/setup assigns Teacher A Mon 08:00-09:00 Class 2 | 409, code `timetable_conflict`, conflicts[0].type = `teacher_double_book` |
| T-02 | Room double-book | Room 101 assigned Mon 08:00-09:00 Class 1 | POST assigns Room 101 Mon 08:00-09:00 Class 2 | 409, conflicts[0].type = `room_double_book` |
| T-03 | Class overlap | Class 1 has Mon 08:00-09:00 slot | POST assigns another slot Mon 08:00-09:00 to same Class 1 | 409, conflicts[0].type = `class_overlap` |
| T-04 | Happy path | No conflicts exist | POST valid slot | 201/200, slot created |
| T-05 | Term resolution | School in break status | POST without term_id | Term resolved to next term, validation uses resolved term |
| T-06 | RLS isolation | Tenant A has slot, Tenant B tries same teacher/time via different school | POST from Tenant B | No conflict reported, tenant isolation enforced |
| T-07 | Frontend error UX | 409 returned from API | Assign form receives conflict | Inline error shown, Save disabled, toast displayed |


## Rollout
- Feature flag: `timetable_conflict_detection` default on
- Backend first, frontend error handling second
- Monitor error rate for `timetable_conflict` in logs

## Implementation Notes
**2026-10-15 – In Progress**
- Docs enhanced with Rationale, Test Matrix and Implementation Notes template to pilot document-led workflow
- Backend validation implemented in `internal/services/timetable_service.go` SetupClassTimetableSlot: teacher, room, class conflict checks inside transaction
- Handler `api/timetable_handler.go` maps `timetable_conflict:*` errors to 409 with code `timetable_conflict`
- E2E Playwright tests added in `frontend/tests/e2e/timetable-conflict.spec.ts` covering teacher/room/class conflicts and happy path
- Frontend error handling for 409 `timetable_conflict` pending in assign form
- Feature flag `timetable_conflict_detection` remains default on

## References
- Gap: `docs/gaps/GAP-timetable-conflict-detection.md`
- Gap: `docs/gaps/GAP-timetable-class-assignment-ui.md`
- Router: `backend/internal/api/router.go` → `protected.Post("/timetable/setup", ...)`
- Service: `backend/internal/services/timetable_service.go`
- Frontend assign form: `frontend/src/features/timetable/components/assign-slot-form.tsx`
