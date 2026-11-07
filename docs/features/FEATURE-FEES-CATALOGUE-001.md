---
id: FEATURE-FEES-CATALOGUE-001
title: Fee Catalogue – Fee Items, Structures and Structure Items
status: draft
depends_on: []
touches: [backend, frontend]
---

# Feature

## Goal
Enable schools to define reusable fee items (tuition, transport, meals, etc.) and assemble them into versioned fee structures per class, academic year and term. Fee structures define standard amounts and optional flags for each fee item.

## Out of scope
* Student-specific fee assignments (see FEATURE-FEES-ASSIGNMENTS-001)
* Invoice generation (see FEATURE-FEES-INVOICING-001)
* Payment recording
* Discounts, waivers, adjustments
* Refunds

## Domain references
* docs/domain/model.md
* docs/domain/glossary.md
* docs/architecture/schema.md
* docs/domain/rules/RULE-001-permissions.md

## Acceptance criteria
- AC-1: School Admin can create, edit, archive fee items per school with fields: name, description, is_active.
- AC-2: School Admin can create fee structures for a specific class, academic year, and term with status (draft, active, archived).
- AC-3: Only one active fee structure per school, academic year, term, and class.
- AC-4: School Admin can add fee structure items linking fee items to a fee structure with amount (KES minor units) and is_optional flag.
- AC-5: Mandatory fee structure items are automatically included when generating invoices; optional items require student fee assignment.
- AC-6: Fee item names and structure item amounts are snapshotted on invoice generation — changes to catalogue do not affect issued invoices.
- AC-7: Inactive fee items cannot be added to new fee structures.
- AC-8: All catalogue changes are audit logged with actor, timestamp, and school context.
- AC-9: API and UI enforce role checks: School Admin manages catalogue; Finance Officer read-only; Guardian/Teacher no access.

## API changes
Pointer to docs/api/openapi.yaml diff: TODO

## Backend notes
Tables:
* fee_items – school-scoped reusable fee definitions
* fee_structures – per class/year/term fee structure header
* fee_structure_items – line items linking fee_items to fee_structures with amount and optional flag
Relations to schools, classes, academic_years, academic_terms.
Money stored as bigint minor units KES. RLS via school_id + tenant.
Unique constraint: one active fee_structure per (school_id, academic_year_id, academic_term_id, class_id).

## Frontend notes
* Fee Items table with CRUD (name, description, active toggle)
* Fee Structures table with CRUD (class, year, term, status)
* Fee Structure Items inline editor per structure (fee item selector, amount input, optional checkbox)
* Use pure Shadcn components, no custom colors

## Verification
* AC-1 → backend unit tests for fee_items CRUD + RLS
* AC-3 → integration test unique active structure constraint
* AC-6 → unit test invoice generation snapshots fee item name and amount
* AC-8 → audit log entries verified for all catalogue mutations
* make compliance passes

## Open questions
* Should fee_structure_items support quantity > 1 (e.g., multiple uniform sets)?
* Archive vs soft-delete for fee_items referenced by historical invoices?