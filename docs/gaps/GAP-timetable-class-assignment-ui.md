# Gap: Timetable Class Assignment UI Partial

**Area:** frontend
**Severity:** Medium
**Status:** Done
**Created:** 2026-10-15

## Description
Backend supports `SetupClassTimetableSlot` to assign subject/teacher/room to a class for a day/time slot. Frontend has `AssignSlotForm` and `TimetableGrid` with Assign/Edit buttons, but workflow may lack validation, bulk assignment, and conflict feedback.

## Current Behavior
Weekly matrix allows Assign via modal/page, assignment persists. However conflict detection and UX polish may be missing.

## Expected Behavior
Assignment UI works end-to-end with conflict detection, bulk actions, and clear success feedback.

## Impact
Without conflict detection, invalid timetables can be created. Workflow needs polish for scale.

## Proposed Solution
Extend weekly matrix UI to support assignment mode. Add assign slot form with validation and conflict detection.

## Acceptance Criteria
- [x] Admin can open class timetable editor
- [x] Slot selection opens assignment form
- [x] Form validates required fields
- [x] Assignments persist via API
- [x] Changes reflected in matrix

## Links
- Analysis: `docs/timetable-gap-analysis.md`
- Feature: `docs/features/FEATURE-timetable-class-assignment-ui.md`
