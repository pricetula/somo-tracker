# Feature: Teacher Class/Subject Linking

**Status:** Proposed
**Owner:** Product / Platform
**Created:** 2026-10-15
**Last Updated:** 2026-10-15
**Related Issues:** #
**Affected Layers:** backend | frontend | docs

## Summary
Enable linking teachers to classes and subjects with validation and UI management.

## Current State
**Backend:** Teacher assignments happen via timetable slots, no explicit linking API.
**Frontend:** No UI for managing teacher-class-subject links.

## Problem Statement
No centralized management of teacher assignments.

## Solution Overview
Add API endpoints and UI to manage links.

### Acceptance Criteria
- [ ] `POST /api/teachers/:id/classes` links teacher to class/subject
- [ ] `DELETE` unlink
- [ ] UI in teacher profile

## References
- Gap: `docs/gaps/GAP-teacher-class-subject-linking.md`
