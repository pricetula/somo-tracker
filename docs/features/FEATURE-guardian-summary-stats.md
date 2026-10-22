# Feature: Guardian Summary Stats

**Status:** Proposed
**Owner:** Product / Platform
**Created:** 2026-10-15
**Last Updated:** 2026-10-15
**Related Issues:** #
**Affected Layers:** backend | frontend | docs

## Summary
Add summary KPIs for guardians to support triage, e.g., total guardians, unlinked guardians, guardians with multiple students.

## Current State
**Backend:** `GET /api/guardians/summary` may exist but unclear scope. No stats for unlinked or multi-student guardians.
**Frontend:** Guardian list page exists but no summary cards.

## Problem Statement
Admins lack quick overview of guardian health.

## Solution Overview
Add summary endpoint and UI cards. Stats sourced from `guardian_link_stats` summary table updated by background workers per ADR-2026-10-15-stats-architecture.

### Acceptance Criteria
- [ ] `GET /api/guardians/summary` returns total, unlinked, multi-student counts
- [ ] Dashboard / Guardian list shows summary cards

## References
- Feature: `docs/features/FEATURE-guardian-profile-enhancement.md`
