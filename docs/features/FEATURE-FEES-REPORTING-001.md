---
id: FEATURE-FEES-REPORTING-001
title: Fee Reports – Collection, Outstanding, Fee Item and Class Reports
status: draft
depends_on: [FEATURE-FEES-BALANCES-001]
touches: [backend, frontend]
---

# Feature

## Goal
Provide operational fee reports for Finance Officers and School Admins: outstanding fees, payment collections, fee-item collection breakdown, class-level summaries, and opening balance tracking.

## Out of scope
* Advanced analytics dashboards with charts
* Scheduled/automated report delivery
* Custom report builder
* Export to Excel/PDF (v1: on-screen only, CSV export for collections)

## Domain references
* docs/domain/model.md
* docs/domain/glossary.md
* docs/architecture/schema.md
* docs/domain/rules/RULE-001-permissions.md

## Acceptance criteria
- AC-1: Outstanding Fees Report — list students with outstanding balance > 0, showing student, class, invoice, outstanding amount, due date, overdue amount, days overdue. Filterable by class, academic year, term, date range.
- AC-2: Payment Collection Report — list completed payments with date, payer, student(s), amount, payment method, reference. Filterable by date range, payment method, class. CSV export.
- AC-3: Fee Item Collection Report — for each fee_item (tuition, transport, meals, etc.): total charged (sum of invoice_items), total collected (sum of payment_allocation_items), total outstanding. Filterable by academic year, term, class.
- AC-4: Class Fees Report — per class: total invoiced, total collected, total outstanding, student count billed, student count with outstanding. Filterable by academic year, term.
- AC-5: Opening Balance Report — list opening_balance invoices with original amount, amount paid, remaining balance. Filterable by student, class.
- AC-6: All reports respect school-scoped RLS — Finance Officer/School Admin see only their school's data.
- AC-7: Reports paginated, server-side, with reasonable default page size.
- AC-8: API and UI enforce role checks: Finance Officer/School Admin full access; Guardian/Teacher no access.

## API changes
Pointer to docs/api/openapi.yaml diff: TODO

## Backend notes
No new tables — reports use derived queries from existing tables.
Key data sources:
- Outstanding: student_invoices + balances (from FEATURE-FEES-BALANCES-001)
- Collections: payments (completed) + payment_allocations + student info
- Fee Item Collection: student_invoice_items (joined to fee_items) + payment_allocation_items
- Class Report: enrollments + invoices + balances grouped by class
- Opening Balance: student_invoices WHERE invoice_type = 'opening_balance'
Performance: consider materialized views for fee_item_collection and class_report if data volume grows.
All queries scoped by school_id via RLS.

## Frontend notes
* Reports landing page with tabs/cards for each report type
* Each report: filter bar (date range, class, year, term, status), results table, pagination
* Outstanding report: highlight overdue rows, sort by days overdue
* Collections report: CSV download button
* Fee Item report: expandable rows showing breakdown by class/term
* Class report: summary cards at top (totals across all classes)
* Use pure Shadcn components, no custom colors

## Verification
* AC-1 → integration test: outstanding report matches manual balance calculation
* AC-3 → integration test: fee item collection totals = sum of allocation_items per fee_item
* AC-4 → integration test: class report aggregates match individual student balances
* AC-5 → integration test: opening balance report tracks payments against opening invoices
* AC-6 → RLS test: cross-school data not visible
* make compliance passes

## Open questions
* Add "receipts" report (list of all payment numbers with dates)?
* Support "fee statement batch" for printing multiple student statements?
* Date range filters: by payment_date, invoice.issue_date, or allocation created_at?