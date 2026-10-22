# Gap: Timetable Conflict Detection Missing

**Area:** backend
**Severity:** High
**Status:** In Progress
**Owner:** Platform team
**Created:** 2026-10-15
**Closed:** (pending — error-contract validation + `/review`)

## Description
No validation prevents teacher double-booking, room double-booking, or class overlapping slots when creating `class_timetable_slots`.

## Impact
Invalid schedules can be created leading to operational issues.

## Proposed Solution
Add validation in `SetupClassTimetableSlot` service:
- Teacher membership cannot have two slots same day/time
- Room cannot be double-booked same day/time
- Class room cannot have overlapping slots

## Acceptance Criteria
- [x] API returns error with code `timetable_conflict` on double booking
- [x] Error details indicate conflicting resource
- [x] `errors.conflicts[]` carries structured conflict objects (`type`, `resource_id`, `resource_name`, `existing_slot_id`)

## Links
- Feature: `docs/features/FEATURE-timetable-conflict-detection.md`
- Service: `backend/internal/services/timetable_service.go` → `SetupClassTimetableSlot`
- Frontend: `frontend/src/features/timetable/components/assign-slot-form.tsx`
