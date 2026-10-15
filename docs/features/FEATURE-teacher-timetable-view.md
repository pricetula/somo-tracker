# Feature: Teacher Timetable View

**Status:** Proposed
**Owner:** Product / Platform
**Created:** 2026-10-15
**Last Updated:** 2026-10-15
**Related Issues:** #
**Affected Layers:** backend | frontend | docs

## Summary
Provide teachers with a personal weekly timetable view showing classes they are assigned to, with subject, class, room, and time.

## Current State
**Backend:** No endpoint for teacher schedule. `class_timetable_slots` contains `teacher_membership_id` but no service aggregates per teacher.
**Frontend:** No teacher timetable page. Teacher profile exists but no schedule view.
**Docs:** Gap identified in `docs/planning/timetable-gap-analysis.md`.

## Problem Statement
Teachers cannot see their weekly schedule in-app, leading to manual checks.

## Solution Overview
Add `GET /api/timetable/teacher/:membershipId/schedule` returning weekly grid for current term.

### Acceptance Criteria
- [ ] API returns weekly slots for teacher
- [ ] Frontend page renders weekly grid
- [ ] Shows subject, class, room, time

## Scope
Backend: new service method, API endpoint
Frontend: new page/view using existing grid component

## References
- `docs/planning/timetable-gap-analysis.md`
