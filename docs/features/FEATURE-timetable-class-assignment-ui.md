# Feature: Timetable Class Assignment UI Polish

**Status:** Done
**Owner:** Product / Platform
**Created:** 2026-10-15
**Last Updated:** 2026-10-15
**Related Issues:** #
**Affected Layers:** frontend | backend | docs

## Summary
Polish the class timetable assignment workflow to provide validation, conflict feedback, bulk actions, and clear UX for assigning subject/teacher/room to class slots.

## Current State
**Backend:** `POST /api/timetable/setup` and `SetupClassTimetableSlot` service exist. Conflict detection implemented in `FEATURE-timetable-conflict-detection.md` and returns 409 `timetable_conflict`.
**Frontend:** `AssignSlotForm` exists in `frontend/src/features/timetable/components/assign-slot-form.tsx` with subject/teacher/room comboboxes. Weekly matrix `TimetableGrid` provides Assign/Edit buttons. Conflict error handling now surfaces inline with `Alert` banner, success toast and optimistic refresh via query invalidation. Bulk assignment mode not yet implemented.
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
- [x] Admin can open class timetable editor and select empty slot
- [x] Slot selection opens assignment form pre-filled with day/time/class context
- [x] Form validates required fields client-side with zod
- [x] Server 409 `timetable_conflict` errors surface inline with resource details and summary Alert banner
- [x] Successful assignment refreshes matrix grid optimistically via query invalidation
- [x] Success toast shown on save
- [x] Bulk assign mode allows selecting multiple slots and applying same assignment pattern
- [x] No dumb back buttons

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

## Implementation Notes
**2026-10-15 – In Progress**
- Added Shadcn `Alert` summary banner to `AssignSlotForm` for 409 `timetable_conflict` responses, displaying per-conflict messages for teacher double-book, room double-book, class overlap
- Conflict state cleared on success and on new submission
- Inline field errors preserved via `form.setError` for teacher/room/root
- Success toast and optimistic grid refresh already handled by `useAssignSlot` mutation
- Context display tightened to human-readable class name, day name, time slot name + time range; IDs never rendered
- Bulk assign mode implemented in `TimetableGrid`: toggle, checkbox selection of empty slots, subject/teacher/room pickers, apply to multiple slots with `useAssignSlot` mutations
- `templateId` passed to `AssignSlotForm` from both modal and page routes for time slot name resolution

## References
- Gap: `docs/gaps/GAP-timetable-class-assignment-ui.md`
- Feature: `docs/features/FEATURE-timetable-conflict-detection.md`
- Frontend: `frontend/src/features/timetable/components/assign-slot-form.tsx`
- Frontend: `frontend/src/features/timetable/components/timetable-grid.tsx`
