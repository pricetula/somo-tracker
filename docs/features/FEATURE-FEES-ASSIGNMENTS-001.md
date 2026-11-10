---
id: FEATURE-FEES-ASSIGNMENTS-001
title: Student Fee Assignments – Optional Fees Per Student
status: draft
depends_on: [FEATURE-FEES-CATALOGUE-001]
touches: [backend, frontend]
---

# Feature

## Goal
Allow schools to assign optional fee structure items (transport, meals, boarding, activities) to individual students with effective date ranges and optional amount overrides. Assignments determine which optional fees appear on a student's invoice.

## Out of scope
* Mandatory fee assignment (automatic via fee structure)
* Invoice generation
* Prorated daily/monthly charges (v1 uses effective date range only)
* Complex approval workflows for assignments

## Domain references
* docs/domain/model.md
* docs/domain/glossary.md
* docs/architecture/schema.md
* docs/domain/rules/RULE-001-permissions.md

## Acceptance criteria
- AC-1: Finance Officer can assign an optional fee structure item to a student for a specific enrollment with effective_from and optional effective_until dates.
- AC-2: Assignment references fee_structure_item which must have is_optional = true.
- AC-3: Assignment can specify amount_override (KES minor units) replacing the structure's standard amount.
- AC-4: Overlapping active assignments for the same student and fee_structure_item are prevented.
- AC-5: Assignment status (active/inactive) controls whether the fee applies during invoice generation.
- AC-6: Assignments are scoped to school, student, and enrollment — cross-school assignments prevented by RLS.
- AC-7: All assignment changes are audit logged with actor, timestamp, and school context.
- AC-8: API and UI enforce role checks: Finance Officer creates/edits assignments; School Admin read-only; Guardian/Teacher no access.

## API changes
Pointer to docs/api/openapi.yaml diff: TODO

## Backend notes
Table: student_fee_assignments
Fields: id, school_id, student_id, enrollment_id, fee_structure_item_id, amount_override, effective_from, effective_until, status, created_by, created_at, updated_at
Unique constraint to prevent overlapping active assignments: (school_id, student_id, fee_structure_item_id) with status = 'active' and date range overlap check.
RLS via school_id + tenant.
Foreign keys to students, enrollments, fee_structure_items (which links to fee_structures).

## Frontend notes
* Assignment list per student/enrollment with inline create/edit
* Date range picker for effective_from / effective_until
* Amount override input (optional, shows structure default as placeholder)
* Status toggle (active/inactive)
* Validation: fee_structure_item must be optional; no overlapping active ranges
* Use pure Shadcn components, no custom colors

## Verification
* AC-2 → unit test rejects assignment to mandatory fee_structure_item
* AC-4 → integration test prevents overlapping active assignments
* AC-5 → integration test: invoice generation includes assigned optional items within effective dates only
* AC-7 → audit log entries verified for assignment CRUD
* make compliance passes

## Open questions
* Should assignment support multiple non-overlapping ranges for same fee_structure_item (e.g., transport Term 1 only, then again Term 3)?
* Bulk assignment UI for class-level optional fees (e.g., all Grade 5 students get meals)?