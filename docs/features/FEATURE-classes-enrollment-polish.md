# Feature: Classes Enrollment Polish

**Status:** Proposed
**Owner:** Product / Platform
**Created:** 2026-10-15
**Last Updated:** 2026-10-15
**Related Issues:** #
**Affected Layers:** backend | frontend | docs

## Summary
Polish class enrollment workflow with bulk actions, validation, and clear feedback.

## Current State
**Backend:** Enrollment API exists via `POST /classes/:id/enrollments`.
**Frontend:** Enrollment UI exists but limited.

## Problem Statement
Enrollment management is cumbersome.

## Solution Overview
Enhance UI with bulk assign, validation. Enrollment counts displayed from `class_enrollment_stats` summary table per ADR-2026-10-15-stats-architecture.

### Acceptance Criteria
- [ ] Bulk assign students to class
- [ ] Validation for duplicate enrollments
- [ ] Success feedback

## References
- Gap: `docs/gaps/GAP-classes-enrollment-polish.md`
