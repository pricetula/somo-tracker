# Feature: Timetable Class Assignment UI Polish

**Status:** Proposed
**Owner:** Product / Platform
**Created:** 2026-10-15
**Last Updated:** 2026-10-15
**Related Issues:** #
**Affected Layers:** frontend | backend | docs

## Summary
Polish the class timetable assignment workflow to provide validation, conflict feedback, bulk actions, and clear UX for assigning subject/teacher/room to class slots.

## Current State
**Backend:** `POST /api/timetable/setup` and `SetupClassTimetableSlot` service exist. No conflict detection currently.
**Frontend:** `AssignSlotForm` exists in `frontend/src/features/timetable/components/assign-slot-form.tsx` with subject/teacher/room comboboxes. Weekly matrix `TimetableGrid` provides Assign/Edit buttons. No bulk assignment, no conflict error handling beyond generic toast.
**Docs:** Gap documented in `docs/gaps/GAP-timetable-class-assignment-ui.md`

## Problem Statement
Assignment UI works technically but lacks validation feedback, bulk actions, and clear success/error states. Admins can create invalid timetables due to missing conflict detection feedback.

## Solution Overview
Enhance existing assignment modal and matrix with validation, conflict handling, and bulk actions.

### User Stories
- As Admin, I want to assign a subject/teacher/room to a slot with immediate validation feedback
- As Admin, I want to see conflicts with clear messages so I can fix them quickly
- As Admin, I want bulk assign capability for recurring patterns

### Acceptance Criteria
- [ ] Admin can open class timetable editor and select empty slot
- [ ] Slot selection opens assignment form pre-filled with day/time/class context
- [ ] Form validates required fields client-side with zod
- [ ] Server 409 `timetable_conflict` errors surface inline with resource details
- [ ] Successful assignment refreshes matrix grid optimistically
- [ ] Success toast shown on save
- [ ] Bulk assign mode allows selecting multiple slots and applying same assignment pattern
- [ ] No dumb back buttons

## Scope

### Backend
- Leverages existing `POST /api/timetable/setup`
- Conflict detection feature `FEATURE-timetable-conflict-detection.md` must be implemented first to provide error feedback

### Frontend
- Enhance `AssignSlotForm` error handling for 409 conflicts
- Add inline error messages per field and summary conflict banner
- Add bulk assign toolbar in weekly matrix
- Improve loading states and success feedback

### Non-Goals
- Template creation UI
- Substitution assignment UI

## Design Notes
Keep existing shadcn components. Use `toast` from Sonner for success.

## Open Gaps / Risks
- Conflict detection must be implemented first
- Bulk actions may require new API endpoint for efficiency

## Test Plan
- E2E assign slot happy path
- E2E conflict error display
- Bulk assign E2E

## Rollout
- Frontend changes after backend conflict detection

## References
- Gap: `docs/gaps/GAP-timetable-class-assignment-ui.md`
- Feature: `docs/features/FEATURE-timetable-conflict-detection.md`
- Frontend: `frontend/src/features/timetable/components/assign-slot-form.tsx`
- Frontend: `frontend/src/features/timetable/components/timetable-grid.tsx`
