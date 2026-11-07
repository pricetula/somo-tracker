---
id: FEATURE-FEES-REFUNDS-001
title: Refunds – Money Returned to Payers
status: draft
depends_on: [FEATURE-FEES-PAYMENTS-001]
touches: [backend, frontend]
---

# Feature

## Goal
Record refunds issued by the school to payers (overpayments, student withdrawal, erroneous payments). Refunds track the money returned and link to the original payment where applicable. Refunds do not automatically reduce invoice balances — they affect available credit.

## Out of scope
* Refund to payment gateway (M-Pesa reversal API)
* Automated refund eligibility calculation
* Multi-stage refund approval workflows (v1: single approver)

## Domain references
* docs/domain/model.md
* docs/domain/glossary.md
* docs/architecture/schema.md
* docs/domain/rules/RULE-001-permissions.md

## Acceptance criteria
- AC-1: Finance Officer can create a refund with amount (KES minor units), refund_date, refund_method, optional reference, reason, optional payment_id (original payment), optional student_id.
- AC-2: Refund number unique per school, generated via concurrency-safe sequence.
- AC-3: Refund status: pending → approved → completed | cancelled.
- AC-4: Refund amount cannot exceed refundable amount on referenced payment (completed payment amount - completed refunds against it).
- AC-5: Completed refunds do not modify invoice balances; they reduce unallocated payment credit.
- AC-6: Refund currency matches school currency (from original payment).
- AC-7: All refund operations transactional with row locking to prevent concurrent over-refunding.
- AC-8: All operations audit logged with actor, timestamp, school context.
- AC-9: API and UI enforce role checks: Finance Officer creates/completes refunds; School Admin approves; Guardian read-only own children; Teacher no access.

## API changes
Pointer to docs/api/openapi.yaml diff: TODO

## Backend notes
Table: refunds
Fields: id, school_id, payment_id (nullable), student_id (nullable), refund_number, amount, currency_code, refund_date, refund_method, reference, reason, status (pending, approved, completed, cancelled), created_by, approved_by, approved_at, completed_at, created_at, updated_at
Unique constraint: (school_id, refund_number)
RLS via school_id + tenant.
Refundable amount calculation: SELECT SUM(amount) FROM payments WHERE id = payment_id AND status = 'completed' MINUS SELECT SUM(amount) FROM refunds WHERE payment_id = payment_id AND status = 'completed'.
Concurrency: SELECT FOR UPDATE on payment when creating/completing refund.

## Frontend notes
* Create Refund form: search payment (show refundable amount), or enter student + amount manually, reason, method, date
* Refund list with status filters
* Approve/Complete/Cancel actions with confirmation
* Refund detail view showing original payment, refundable amount calculation
* Use pure Shadcn components, no custom colors

## Verification
* AC-4 → integration test: refund exceeding refundable amount rejected
* AC-5 → unit test: completed refund reduces unallocated credit, not invoice balance
* AC-7 → integration test: concurrent refund creation on same payment prevented
* AC-8 → audit log entries verified for create, approve, complete, cancel
* make compliance passes

## Open questions
* Should refund_method be enum (cash, bank_transfer, mobile_money, cheque, offset)?
* Support partial refund of a payment (yes, up to refundable amount)?
* Auto-create refund when payment reversed? (Separate operation — reversal releases allocations, refund returns money)