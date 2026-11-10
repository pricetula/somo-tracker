---
id: FEATURE-FEES-OPENING-BALANCES-001
title: Opening Balances – Migration and Arrears Import
status: draft
depends_on: [FEATURE-FEES-INVOICING-001, FEATURE-FEES-PAYMENTS-001]
touches: [backend, frontend]
---

# Feature

## Goal
Import pre-existing student fee balances (arrears from previous terms or legacy systems) as opening-balance invoices. These invoices behave like standard invoices for payment allocation and balance calculation but are distinguishable in reports and do not require a fee structure or enrollment.

## Out of scope
* Automated migration tooling (v1: manual CSV import or UI entry)
* Historical payment/allocation reconstruction (opening balance = net outstanding only)
* Per-fee-item breakdown of opening balances (v1: single line per opening invoice)

## Domain references
* docs/domain/model.md
* docs/domain/glossary.md
* docs/architecture/schema.md
* docs/domain/rules/RULE-001-permissions.md

## Acceptance criteria
- AC-1: Finance Officer can create an opening-balance invoice with student, description, amount (KES minor units), issue_date, due_date.
- AC-2: Opening-balance invoice has invoice_type = 'opening_balance', no fee_structure_id, no enrollment_id required.
- AC-3: Opening-balance invoice contains a single invoice_item with fee_item_id = NULL, description = provided description, quantity = 1, unit_amount = amount.
- AC-4: Opening-balance invoices participate in balance calculations, payment allocation, and statements identically to standard invoices.
- AC-5: Opening-balance invoices are excluded from the unique-enrollment-per-structure constraint.
- AC-6: Opening-balance invoices are filterable and identifiable in all reports (Outstanding, Class, Opening Balance Report).
- AC-7: Payments can be allocated to opening-balance invoices via standard allocation flow.
- AC-8: All operations audit logged with actor, timestamp, school context.
- AC-9: API and UI enforce role checks: Finance Officer creates/allocates; School Admin read-only; Guardian read-only own children; Teacher no access.

## API changes
Pointer to docs/api/openapi.yaml diff: TODO

## Backend notes
Uses existing student_invoices and student_invoice_items tables with invoice_type = 'opening_balance'.
No new tables.
Unique constraint on (school_id, enrollment_id, fee_structure_id) has WHERE invoice_type = 'standard' — opening balances excluded.
CSV import endpoint (optional v1 enhancement): accept student_id/reference, description, amount, issue_date, due_date; create opening invoices in bulk transaction.

## Frontend notes
* Create Opening Balance form: student selector, description, amount, issue date, due date
* Bulk Import modal: CSV upload with columns (student_adm_number, description, amount, issue_date, due_date), preview, confirm
* Opening Balance list with filter (standard vs opening_balance type)
* Distinct visual badge "Opening Balance" on invoice rows and statements
* Use pure Shadcn components, no custom colors

## Verification
* AC-2/3 → unit test: opening invoice created with correct type and single NULL fee_item line
* AC-4 → integration test: payment allocation to opening invoice reduces student balance
* AC-5 → integration test: multiple opening invoices allowed per student
* AC-6 → integration test: Opening Balance Report shows only opening_balance invoices
* AC-8 → audit log entries verified for create and bulk import
* make compliance passes

## Open questions
* Should opening balance support multiple lines (e.g., separate tuition and transport arrears)?
* Validate student exists and belongs to school on CSV import?
* Auto-generate description from template (e.g., "Arrears as of {date}")?