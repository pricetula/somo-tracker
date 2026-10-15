# Feature: Teacher Profile Enhancement

**Status:** Proposed
**Owner:** Product / Platform
**Created:** 2026-10-15
**Last Updated:** 2026-10-15
**Related Issues:** #
**Affected Layers:** backend | frontend | docs

## Summary
Enhance teacher detail page with comprehensive profile, classes taught, timetable summary, and stats.

## Current State
**Backend:** Teacher list/summary exists. No enriched profile endpoint with classes/timetable.
**Frontend:** Teacher detail page exists but basic.

## Problem Statement
No holistic view of teacher workload.

## Solution Overview
Add enriched `GET /api/teachers/:id` with classes, timetable summary.

### Acceptance Criteria
- [ ] Teacher profile shows personal info, classes taught, subjects
- [ ] Timetable summary per term
- [ ] Edit capability

## References
- Gap: `docs/gaps/GAP-teacher-profile-enhancement.md`
