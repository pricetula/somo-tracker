# Timetable Module — Roadmap & Dependency Order

**Created:** 2026-10-15
**Status:** Draft
**Scope:** Timetable assignment, conflict detection, student view

## Context
Timetable features are inter-dependent: conflict detection is a data integrity prerequisite for any assignment UI, and student view consumes validated timetable data.

## Proposed Order

### Phase 1 — Data Integrity First
**Feature:** Timetable Conflict Detection
* File: `docs/features/FEATURE-timetable-conflict-detection.md`
* Why first: Prevents invalid writes. Without it, UI polish just makes invalid data easier to create.
* Backend only: add validation in `SetupClassTimetableSlot`, return `timetable_conflict` error
* No UI changes yet

### Phase 2 — Assignment UX Polish
**Feature:** Timetable Class Assignment UI Polish
* File: `docs/features/FEATURE-timetable-class-assignment-ui.md`
* Depends on: Phase 1 conflict detection
* Frontend enhancements to `AssignSlotForm` and `TimetableGrid`:
  * Inline 409 conflict messages
  * Success feedback, loading states
  * Bulk assign mode
* Minimal backend work

### Phase 3 — Consumer View
**Feature:** Student Timetable View
* File: `docs/features/FEATURE-student-timetable-view.md`
* Depends on: Phase 1 + 2 data quality
* New `GET /api/students/:id/timetable` endpoint
* Read-only weekly grid for students
* No writes, safe to add after integrity is ensured

## Deferrals
* Automated rescheduling suggestions
* Substitution conflict detection — separate feature
* Bulk import of timetable templates — out of scope

## Risks
* If UI polish ships before conflict detection, users will hit generic errors
* Student view performance: N+1 queries for enrollments → classes → slots. Consider materialized view later.

## Recommendation
Ship Phase 1 → Phase 2 → Phase 3 sequentially. Do not parallelize Phase 2 and Phase 1.

## References
* `docs/gaps/GAP-timetable-conflict-detection.md`
* `docs/gaps/GAP-timetable-class-assignment-ui.md`
* `docs/gaps/GAP-student-timetable-view.md`
