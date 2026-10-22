# Feature: Guardian Profile Enhancement

**Status:** Proposed
**Owner:** Product / Platform
**Created:** 2026-10-15
**Last Updated:** 2026-10-15
**Related Issues:** #
**Affected Layers:** backend | frontend | docs

## Summary
Enhance guardian detail page with comprehensive profile, linked students, contact history, and communication preferences.

## Current State
**Backend:** `GET /api/guardians` list exists, `GET /api/guardians/:id` likely exists but not verified. No enriched profile endpoint with linked students.
**Frontend:** Guardian detail page exists at `/guardians/[id]` but may be placeholder. No student linking UI.

## Problem Statement
Admins cannot view full guardian context in one place, making communication and relationship management difficult.

## Solution Overview
Build enriched guardian profile with tabs for Overview, Linked Students, Contact History, Notes.

### Acceptance Criteria
- [ ] `GET /api/guardians/:id` returns enriched profile with linked students
- [ ] Guardian detail page shows personal info, contact info, linked students with relationship type
- [ ] Link/unlink students from guardian profile
- [ ] Shows communication preferences

## Scope
Backend: enhance service to return linked students
Frontend: enhance guardian detail UI

## References
- Feature: `docs/features/FEATURE-guardian-linking.md`
