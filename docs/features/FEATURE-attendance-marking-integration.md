# Feature: Attendance Marking Integration

**Status:** In Progress
**Owner:** Product / Platform
**Created:** 2026-10-15
**Last Updated:** 2026-10-15
**Related Issues:** #
**Affected Layers:** backend | frontend | docs

## Summary
Integrate attendance marking UI with timetable slots for context-aware attendance taking.

## Current State
**Backend:** `POST /api/attendance` exists, `GET /api/attendance/sessions` exists. No integration with timetable slots.
**Frontend:** Attendance page exists but marks attendance without slot context.

## Problem Statement
Attendance marking is disconnected from schedule, leading to errors.

## Solution Overview
Enhance attendance marking to show class timetable slots, allow marking per slot.

### Acceptance Criteria
- [ ] Attendance marking shows timetable slot context
- [ ] Mark attendance per student per slot
- [ ] Validation

## References
