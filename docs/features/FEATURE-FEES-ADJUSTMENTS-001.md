---
id: FEATURE-FEES-ADJUSTMENTS-001
title: Fee Adjustments – Discounts, Waivers and Corrections
status: draft
depends_on: [FEATURE-FEES-INVOICING-001]
touches: [backend, frontend]
---

# Feature

## Goal
Apply adjustments to specific invoice lines — discounts, waivers, bursary contributions, corrections, or additional charges. Adjustments are traceable, require approval where configured, and modify the outstanding balance without altering the original invoice snapshot.

## Out of scope
* Percentage-based discounts (v1 uses fixed amounts only)
* Multi-stage approval workflows (v1: single approver field)
* Third-party sponsor/bursary tracking (v1: adjustment reason text only)
* Automatic recurring adjustments

## Domain references
* docs/domain/model.md
* docs/domain/glossary.md
* docs/architecture/schema.md
* docs/domain/rules/RULE-001-permissions.md

## Acceptance criteria
- AC-1: Finance Officer can create an adjustment on a student_invoice_item with type: discount, waiver, correction, additional_charge.
- AC-2: Adjustment has description (reason), amount (KES minor units, positive), adjustment_date, status (pending, approved, rejected).
- AC-3: Discount, waiver, correction reduce amount owed; additional_charge increases amount owed.
- AC-4: Only approved adjustments affect invoice balance calculation.
- AC-5: Adjustment cannot reduce an invoice line's net amount below zero unless type = correction with explicit override.
- AC-6: Adjustment references creator and optional approver/approval_timestamp.
- AC-7: Original invoice and invoice items remain unchanged — adjustments are separate records.
- AC-8: All adjustment operations audit logged with actor, timestamp, school context.
- AC-9: API and UI enforce role checks: Finance Officer creates adjustments; School Admin approves (if required); Guardian read-only own children; Teacher no access.

## API changes
Pointer to docs/api/openapi.yaml diff: TODO

## Backend notes
Table: fee_adjustments
Fields: id, school_id, student_invoice_id, student_invoice_item_id, adjustment_type (discount, waiver, correction, additional_charge), description, amount, adjustment_date, status (pending, approved, rejected), created_by, approved_by, approved_at, created_at, updated_at
RLS via school_id + tenant.
Adjustment application: on approval, recalculate invoice balance = sum(items) - sum(approved discounts/waivers/corrections) + sum(approved additional_charges).
Validation: net line amount = item.total_amount - sum(approved reductions) + sum(approved increases) ≥ 0 (unless correction with override).

## Frontend notes
* Adjustment create form: select invoice → select line → type, amount, description, date
* Adjustment list per invoice with status badges
* Approve/Reject actions for pending adjustments (School Admin)
* Show net line amount preview after adjustment
* Use pure Shadcn components, no custom colors

## Verification
* AC-3 → unit test: balance calculation applies adjustment types correctly
* AC-5 → integration test: reduction below zero rejected (unless correction override)
* AC-7 → unit test: original invoice_item.total_amount unchanged after adjustment
* AC-8 → audit log entries verified for create, approve, reject
* make compliance passes

## Open questions
* Should corrections be split into "reduce" and "increase" sub-types for clarity?
* Configurable approval threshold (e.g., auto-approve < KSh 1,000)?
* Support adjustment on invoice header (not line) for global discounts?