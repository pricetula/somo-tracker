# Feature: Student Schema Enhancement

**Status:** Done
**Owner:** Platform Team
**Created:** 2026-10-15
**Last Updated:** 2026-10-15
**Related Issues:** #
**Affected Layers:** backend | docs

## Summary
Add structured profile fields to the `students` table to support a complete student profile UI and reliable querying/reporting.

## Current State
**Backend:** `students` table has `full_name, date_of_birth, gender, admission_number, metadata JSONB`. No structured columns for contact/address/emergency info.
**Frontend:** `StudentDetail` component design expects address, phone, email, nationality, emergency contact fields.
**Docs:** Gap documented in `docs/gaps/GAP-student-schema-missing-fields.md`

## Problem Statement
Personal data is stored as unstructured `metadata JSONB`. Queries for address/phone/email are unreliable, reporting is impossible, and frontend cannot render consistent profile fields.

## Solution Overview
Migrate `students` table to add structured columns and update service DTOs, RLS policies, and API responses.

### User Stories
- As a Data Officer, I want structured contact fields so I can search and report on student demographics.
- As an Admin, I want to edit address/phone/email inline in student profile.

### Acceptance Criteria
- [x] Migration adds columns: `address`, `phone`, `email`, `nationality`, `emergency_contact_name`, `emergency_contact_phone`, `special_needs`
- [x] Columns are nullable with appropriate constraints
- [x] RLS policies updated to cover new columns
- [x] `GET /students/:id` returns new fields
- [x] `PATCH /students/:id` accepts and validates new fields
- [x] Backfill script migrates existing `metadata` values where present
- [x] API error contract compliant

## Scope

### Backend
- DB changes / migrations:
  - Add columns to `students` table
  - Create index on `email` and `phone` for search
- Service changes:
  - Update `sqlc` queries and DTOs
  - Update `StudentsService.GetStudentByID` and `UpdateStudent`
- RLS / multi-tenancy: Ensure policies include new columns

### Frontend
- No UI changes in this feature — enables upcoming Enhanced Student Profile feature

### Non-Goals
- Document/photo uploads
- Normalizing contacts to separate table — deferred

## Design Notes
Keep fields nullable for backward compatibility. Consider future migration to `student_contacts` normalized table.

## Open Gaps / Risks
- Data quality of existing `metadata` backfill
- Email uniqueness per school — not enforced in v1

## Test Plan
- Migration integration test
- API contract tests for new fields
- RLS tests

## Rollout
- Migration first, then service update

## References
- Gap: `docs/gaps/GAP-student-schema-missing-fields.md`
- Feature: `docs/features/FEATURE-enhanced-student-profile.md`
- Schema: `docs/reference/backend_schema.md`
