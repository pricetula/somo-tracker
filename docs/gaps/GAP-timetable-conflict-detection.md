# Gap: Timetable Conflict Detection Missing

**Area:** backend
**Severity:** High
**Status:** Open
**Created:** 2026-10-15

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
- [ ] API returns error with code `timetable_conflict` on double booking
- [ ] Error details indicate conflicting resource

## Links
- Feature: `docs/features/FEATURE-timetable-conflict-detection.md`
- Service: `backend/internal/services/timetable_service.go` → `SetupClassTimetableSlot`
- Frontend: `frontend/src/features/timetable/components/assign-slot-form.tsx`
