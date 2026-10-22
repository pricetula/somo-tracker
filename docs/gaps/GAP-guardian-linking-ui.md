# Gap: Guardian Linking UI Missing

**Area:** frontend | backend
**Severity:** High
**Status:** Done
**Owner:** Platform team
**Created:** 2026-10-15
**Closed:** 2026-10-15

## Description
No UI exists to link/unlink guardians to students, view relationship type, or manage contacts in student profile. Backend may have linking tables but no documented API for linking flow.

## Impact
Student profiles cannot show guardian contacts, admins cannot manage guardian relationships, listing badges for guardian-linked status cannot be accurate.

## Proposed Solution
Document UI flow for linking/unlinking guardians, view relationship type and contacts in student detail page. Add API endpoints for linking.

## Acceptance Criteria
- [x] Admin can link/unlink guardians to students
- [x] Relationship type selectable
- [x] Guardian contacts visible in student profile
- [x] Listing shows guardian badge

## Links
- Roadmap: `docs/planning/student-module-roadmap.md`
- Feature: `docs/features/FEATURE-enhanced-student-profile.md`
