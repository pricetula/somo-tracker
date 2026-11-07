---
id: FEATURE-FEES-BALANCES-001
title: Balance Calculations and Student Fee Statements
status: draft
depends_on: [FEATURE-FEES-INVOICING-001, FEATURE-FEES-PAYMENTS-001, FEATURE-FEES-ADJUSTMENTS-001, FEATURE-FEES-REFUNDS-001]
touches: [backend, frontend]
---

# Feature

## Goal
Provide consistent, derived balance calculations for invoices, students, and payments. Generate read-only student fee statements for guardians showing invoices, items, adjustments, payments, refunds, outstanding balances, and unallocated credit.

## Out of scope
* Stored/denormalized balance fields (all balances computed from source records)
* Real-time balance WebSocket updates
* Guardian-facing payment portal

## Domain references
* docs/domain/model.md
* docs/domain/glossary.md
* docs/architecture/schema.md
* docs/domain/rules/RULE-001-permissions.md

## Acceptance criteria
- AC-1: Invoice balance = sum(invoice_items.total_amount) + sum(approved additional_charge adjustments) - sum(approved discount/waiver/correction adjustments) - sum(payment_allocation_items.amount) for that invoice.
- AC-2: Invoice status derived: draft | issued | partially_paid (balance > 0 and < total) | paid (balance = 0) | void.
- AC-3: Invoice overdue = issued AND due_date < today AND balance > 0.
- AC-4: Student balance = sum of outstanding balances across all non-void invoices for that student.
- AC-5: Unallocated payment credit = sum(completed payments) - sum(active payment_allocations.amount) - sum(completed refunds against those payments).
- AC-6: Payment allocation status derived: unallocated | partially_allocated | fully_allocated.
- AC-7: Guardian can view fee statement for linked children showing: invoices (with items), adjustments, payment allocations (with line breakdown), refunds, outstanding per invoice, total outstanding, unallocated credit.
- AC-8: Statement is read-only, paginated, filterable by date range and status.
- AC-9: All calculations use KES minor units (bigint), displayed as formatted currency.
- AC-10: API and UI enforce role checks: Finance Officer/School Admin full access; Guardian read-only own children; Teacher no access.

## API changes
Pointer to docs/api/openapi.yaml diff: TODO

## Backend notes
No new tables — all calculations derived via SQL views or application-layer queries.
Key queries:
- Invoice balance: aggregate invoice_items + adjustments + payment_allocation_items
- Student balance: sum over invoices WHERE status != 'void'
- Unallocated credit: payments.completed - allocations.active - refunds.completed
- Statement: JOIN invoices → items → adjustments + allocations → allocation_items + payments + refunds
Performance: materialized view or indexed view for student balances if needed.
RLS ensures school-scoped results.

## Frontend notes
* Student Fee Statement page (guardian portal): child selector, date range, status filter
* Statement sections: Invoices table, Payments table, Summary cards (Total Outstanding, Unallocated Credit)
* Invoice row expandable to show items, adjustments, allocations per line
* Currency formatting: KES major units with separators (e.g., KSh 25,000)
* Print/export button (PDF in future)
* Use pure Shadcn components, no custom colors

## Verification
* AC-1 → unit tests: balance calculation with various adjustment/payment combinations
* AC-3 → unit test: overdue derivation logic
* AC-4 → integration test: student balance sums across multiple invoices
* AC-5 → integration test: unallocated credit tracks payment → allocation → refund lifecycle
* AC-7 → integration test: guardian API returns only linked children's statements
* make compliance passes

## Open questions
* Should statement show fee_item-level totals (tuition paid vs transport paid)?
* Cache balance calculations or compute on demand? (v1: compute on demand with query optimization)
* Include voided invoices in statement with strikethrough?