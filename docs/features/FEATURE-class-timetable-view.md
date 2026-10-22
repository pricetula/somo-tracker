# Feature: Class Timetable View

**Status:** Proposed
**Owner:** Product / Platform
**Created:** 2026-10-15
**Last Updated:** 2026-10-15
**Related Issues:** #
**Affected Layers:** backend | frontend | docs

## Summary
Provide a read-only weekly timetable view for a class, showing subjects, teachers, rooms per slot.

## Current State
**Backend:** `GetClassTimetableSlotsByTemplate` exists but UI for class view not exposed to students/teachers.
**Frontend:** No dedicated class timetable view page.
**Docs:** Gap identified in `docs/planning/timetable-gap-analysis.md`.

## Problem Statement
Students and teachers cannot see class schedule without navigating through admin assignment UI.

## Solution Overview
Expose class timetable grid in read-only mode.

### Acceptance Criteria
- [ ] `GET /api/classes/:id/timetable` returns weekly grid for current term
- [ ] Frontend page renders grid
- [ ] Read-only for students/teachers

## Scope
Backend: new endpoint or reuse existing service
Frontend: new page/view

## References
- `docs/planning/timetable-gap-analysis.md`
