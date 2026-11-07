---
id: FEATURE-FEES-INVOICING-001
title: Student Invoicing – Invoice Generation and Invoice Items
status: draft
depends_on: [FEATURE-FEES-CATALOGUE-001, FEATURE-FEES-ASSIGNMENTS-001]
touches: [backend, frontend]
---

# Feature

## Goal
Generate per-student invoices from fee structures and student fee assignments. Support standard invoices (from enrollment + fee structure) and opening-balance invoices (for migrated arrears). Invoices are immutable once issued — amounts and descriptions are snapshotted.

## Out of scope
* Payment recording and allocation (see FEATURE-FEES-PAYMENTS-001)
* Discounts, waivers, adjustments (see FEATURE-FEES-ADJUSTMENTS-001)
* Refunds
* Automated reminder emails/SMS

## Domain references
* docs/domain/model.md
* docs/domain/glossary.md
* docs/architecture/schema.md
* docs/domain/rules/RULE-001-permissions.md

## Acceptance criteria
- AC-1: Finance Officer can generate a standard invoice for a student's enrollment using the applicable active fee structure.
- AC-2: Standard invoice includes all mandatory fee_structure_items plus optional items where student has active student_fee_assignment within effective dates.
- AC-3: Invoice items snapshot fee_item name, description, quantity, unit_amount (KES minor units), total_amount at generation time.
- AC-4: Invoice number is unique per school, human-readable, generated via concurrency-safe sequence.
- AC-5: One standard invoice per enrollment per billing period (enforced by unique constraint).
- AC-6: Finance Officer can create opening-balance invoices without fee_structure or enrollment, with custom description and amount.
- AC-7: Invoice status flow: draft → issued → partially_paid → paid (auto) | void (manual with reason).
- AC-8: Issued invoices are immutable — amounts, items, descriptions cannot be changed.
- AC-9: Voiding an invoice with allocations releases allocations back to unallocated payment credit.
- AC-10: All invoice operations audit logged with actor, timestamp, school context.
- AC-11: API and UI enforce role checks: Finance Officer generates/issues/voids invoices; School Admin read-only; Guardian read-only own children; Teacher no access.

## API changes
Pointer to docs/api/openapi.yaml diff: TODO

## Backend notes
Tables:
* student_invoices – header with type (standard, opening_balance), enrollment_id, fee_structure_id, invoice_number, issue_date, due_date, currency_code, total_amount, status, voided_at, voided_by, void_reason
* student_invoice_items – lines with fee_item_id (nullable for opening-balance), description snapshot, quantity, unit_amount, total_amount
Unique constraints:
- (school_id, invoice_number)
- (school_id, enrollment_id, fee_structure_id) WHERE invoice_type = 'standard'
RLS via school_id + tenant.
Money as bigint minor units KES. Currency_code snapshot from school's country.
Invoice generation transaction: read fee_structure + assignments → create invoice + items atomically.

## Frontend notes
* Generate Invoice modal: select student/enrollment, preview items (mandatory + assigned optional), confirm
* Opening Balance Invoice form: student, description, amount, issue date
* Invoice list with filters (student, status, date range, type)
* Invoice detail view: items, status, allocations, adjustments, balance
* Void action with required reason field
* Use pure Shadcn components, no custom colors

## Verification
* AC-2 → integration test: generated invoice items match fee_structure + active assignments
* AC-3 → unit test: invoice item fields are snapshots, not live references
* AC-5 → integration test: duplicate generation for same enrollment+structure rejected
* AC-9 → integration test: void invoice with allocations releases allocation amounts
* AC-10 → audit log entries verified for generate, issue, void
* make compliance passes

## Open questions
* Should draft invoices be editable before issuing? (Spec says issued = immutable, implies draft editable)
* Auto-calculate due_date based on school policy (e.g., +14 days from issue_date)?
* Support multiple billing cycles per term (mid-term invoices)?