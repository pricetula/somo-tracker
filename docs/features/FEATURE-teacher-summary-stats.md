# Feature: Teacher Summary Stats

**Status:** Proposed
**Owner:** Product / Platform
**Created:** 2026-10-15
**Last Updated:** 2026-10-15
**Related Issues:** #
**Affected Layers:** backend | frontend | docs

## Summary
Add summary KPIs for teachers to support triage, e.g., total teachers, unassigned teachers, teachers with high workload.

## Current State
**Backend:** `GET /api/teachers/summary` may exist but limited scope.
**Frontend:** Teacher list has no summary cards.

## Problem Statement
Admins lack quick overview of teacher health.

## Solution Overview
Add summary endpoint and UI cards. Stats sourced from `teacher_workload_stats` summary table updated by background workers per ADR-2026-10-15-stats-architecture.

### Acceptance Criteria
- [ ] `GET /api/teachers/summary` returns total, unassigned, workload stats
- [ ] Dashboard / Teacher list shows summary cards

## References
- Feature: `docs/features/FEATURE-teacher-profile-enhancement.md`
