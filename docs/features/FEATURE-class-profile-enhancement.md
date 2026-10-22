# Feature: Class Profile Enhancement

**Status:** In Progress
**Owner:** Product / Platform
**Created:** 2026-10-15
**Last Updated:** 2026-10-15
**Related Issues:** #
**Affected Layers:** backend | frontend | docs

## Summary
Enhance class detail page with comprehensive profile, enrolled students, teacher assignments, and timetable.

## Current State
**Backend:** Class API exists, no enriched profile endpoint.
**Frontend:** Class detail page exists but basic.

## Problem Statement
No holistic view of class.

## Solution Overview
Add enriched `GET /api/classes/:id` with students, teachers, timetable. Enrollment counts sourced from `class_enrollment_stats` summary table per ADR-2026-10-15-stats-architecture.

### Acceptance Criteria
- [x] Class profile shows overview, students, teachers, timetable
- [ ] Edit capability

## References
- Gap: `docs/gaps/GAP-class-profile-enhancement.md`
