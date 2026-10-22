# Feature: Guardian Linking

**Status:** Proposed
**Owner:** Product / Platform
**Created:** 2026-10-15
**Last Updated:** 2026-10-15
**Related Issues:** #
**Affected Layers:** backend | frontend | docs

## Summary
Enable linking and unlinking guardians to students with relationship type, and display guardian contacts in student profile and listings.

## Current State
**Backend:** `guardian_student_links` table likely exists but no documented API for linking flow exposed in router. No validation or UI integration.
**Frontend:** No UI for linking guardians. Student detail page has Guardians tab scaffolded but empty.
**Docs:** Gap not previously documented.

## Problem Statement
Admins cannot associate guardians with students, so contact info is missing from profiles and listings.

## Solution Overview
Add API endpoints for linking/unlinking and UI flow in student profile. Support multiple guardians per student with relationship type and primary contact flag.

### User Stories
- As Admin, I want to link multiple guardians to a student so all contacts are recorded
- As Admin, I want to set a primary guardian for communications
- As Admin, I want to see relationship type to understand context

### Acceptance Criteria
- [ ] `POST /api/students/:id/guardians` links guardian with relationship_type and is_primary flag
- [ ] `PATCH /api/students/:id/guardians/:guardianId` updates relationship_type / is_primary
- [ ] `DELETE /api/students/:id/guardians/:guardianId` unlinks
- [ ] Only one primary guardian per student enforced
- [ ] Validation prevents linking guardian from different school
- [ ] RLS ensures tenant isolation
- [ ] Student profile Guardians tab shows list with name, phone, email, relationship, primary badge, link/unlink actions
- [ ] Student list shows guardian_linked badge and count
- [ ] Error contract compliant: 400 for invalid relationship_type, 409 for duplicate link

## Scope
Backend: new API endpoints, service validation
Frontend: UI in student detail Guardians tab

## References
- Gap: `docs/gaps/GAP-guardian-linking-ui.md`
