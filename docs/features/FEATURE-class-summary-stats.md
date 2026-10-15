# Feature: Class Summary Stats

**Status:** Proposed
**Owner:** Product / Platform
**Created:** 2026-10-15
**Last Updated:** 2026-10-15
**Related Issues:** #
**Affected Layers:** backend | frontend | docs

## Summary
Add summary KPIs for classes to support triage.

## Current State
No summary stats for classes.

## Problem Statement
Admins lack quick overview.

## Solution Overview
Add `GET /api/classes/summary`. Stats sourced from `class_enrollment_stats` summary table updated by background workers per ADR-2026-10-15-stats-architecture.

### Acceptance Criteria
- [ ] Summary cards show total classes, unassigned, etc.

## References
