# Gap: Student Schema Missing Structured Fields

**Area:** backend
**Severity:** High
**Status:** Open
**Owner:** 
**Created:** 2026-10-15

## Description
`students` table stores personal data only as `full_name`, `date_of_birth`, `gender`, `admission_number`, and unstructured `metadata JSONB`. No structured fields for address, phone, email, nationality, emergency contact, special needs.

## Impact
Cannot reliably query or report on contact info. Frontend cannot render consistent profile. Data quality risk.

## Proposed Solution
Add structured columns to `students` table for core profile fields + create normalized tables for contacts and notes as per `docs/student-detail-gap-analysis.md`. Document/photo uploads deferred.

## Acceptance Criteria
- [ ] Migration adds address, phone, email, nationality, emergency_contact_name, emergency_contact_phone
- [ ] RLS policies updated
- [ ] API GET /students/:id returns new fields
- [ ] Backfill script for existing metadata

## Links
- Analysis: `docs/student-detail-gap-analysis.md`
- Feature: `docs/features/FEATURE-enhanced-student-profile.md`
- Feature: `docs/features/FEATURE-student-schema-enhancement.md`
