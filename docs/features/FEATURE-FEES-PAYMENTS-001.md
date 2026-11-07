---
id: FEATURE-FEES-PAYMENTS-001
title: Payments – Recording, Allocation and Allocation Items
status: draft
depends_on: [FEATURE-FEES-INVOICING-001]
touches: [backend, frontend]
---

# Feature

## Goal
Record payments received by the school and allocate them to specific student invoices and invoice lines. Support one payment covering multiple students/invoices, partial allocations, and strict line-level tracking for fee-item reporting. Payments are immutable once completed.

## Out of scope
* Payment gateway integration (M-Pesa, bank APIs)
* Automated payment reconciliation from bank statements
* Refunds (see FEATURE-FEES-REFUNDS-001)
* Discounts/waivers (see FEATURE-FEES-ADJUSTMENTS-001)

## Domain references
* docs/domain/model.md
* docs/domain/glossary.md
* docs/architecture/schema.md
* docs/domain/rules/RULE-001-permissions.md

## Acceptance criteria
- AC-1: Finance Officer can record a payment with payer_name, amount (KES minor units), payment_date, payment_method, optional reference, optional idempotency_key.
- AC-2: Payment number (receipt number) is unique per school, generated via concurrency-safe sequence.
- AC-3: Payment status: pending → completed (manual or auto) | reversed.
- AC-4: Only completed payments can be allocated and reduce outstanding balances.
- AC-5: Finance Officer can allocate a completed payment to one or more student_invoices with amounts.
- AC-6: Allocation creates payment_allocation_items distributing the allocation amount across invoice lines (oldest outstanding line first for auto-allocation; manual override supported).
- AC-7: Sum of allocation_item amounts equals parent allocation amount; sum of allocations ≤ completed payment amount.
- AC-8: Allocation cannot exceed invoice's outstanding balance per line.
- AC-9: Reversing a payment releases all its active allocations back to unallocated credit.
- AC-10: All payment/allocation operations are transactional with row locking to prevent concurrent over-allocation.
- AC-11: Idempotency key prevents duplicate payment creation from external callbacks.
- AC-12: All operations audit logged with actor, timestamp, school context.
- AC-13: API and UI enforce role checks: Finance Officer records/allocates/reverses; School Admin read-only; Guardian read-only own children's payments; Teacher no access.

## API changes
Pointer to docs/api/openapi.yaml diff: TODO

## Backend notes
Tables:
* payments – header with school_id, payment_number, payer_name, payer_contact, amount, currency_code, payment_date, payment_method, reference, idempotency_key, status, reversed_at, reversed_by, reversal_reason
* payment_allocations – payment_id, student_invoice_id, amount
* payment_allocation_items – payment_allocation_id, student_invoice_item_id, amount
Unique constraints:
- (school_id, payment_number)
- (school_id, reference) WHERE reference IS NOT NULL (per payment_method scope)
RLS via school_id + tenant.
Money as bigint minor units KES.
Allocation logic: transaction with SELECT FOR UPDATE on payment and invoices; validate sums; create allocation + items atomically.
Auto-allocation order: student_invoice.issue_date ASC, student_invoice_item.id ASC.

## Frontend notes
* Record Payment form: payer details, amount, date, method, reference, idempotency key (optional)
* Allocation UI: search invoices by student, show outstanding balances, add allocation lines, auto-fill from oldest outstanding
* Allocation Items breakdown: show per invoice line allocation (read-only for auto, editable for manual)
* Payment detail view: allocations, allocation items, status, reversal action
* Reversal confirmation modal with required reason
* Use pure Shadcn components, no custom colors

## Verification
* AC-2 → integration test: concurrent payment creation generates unique numbers
* AC-4 → unit test: pending payment allocation rejected
* AC-6 → integration test: auto-allocation follows oldest-line-first rule
* AC-7/8 → integration test: allocation sums validated, cannot exceed outstanding
* AC-9 → integration test: payment reversal releases allocations
* AC-11 → integration test: duplicate idempotency_key returns existing payment
* AC-12 → audit log entries verified for payment, allocation, reversal
* make compliance passes

## Open questions
* Should payment_method be an enum or free text? (Enum: cash, bank_transfer, mobile_money, cheque, card, other)
* Support partial payment reversal (refund) vs full reversal only?
* Allow allocation to opening-balance invoices? (Yes, same as standard invoices)