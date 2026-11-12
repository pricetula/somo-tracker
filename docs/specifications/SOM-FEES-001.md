---
id: SOM-FEES-001
title: Fees Domain - Fee Structures, Invoices, Payments, Waivers
status: draft
author: agent
date: 2026-11-11
updated: 2026-11-11
fr_ids: [BE-006, BE-007, BE-009, BE-010, BE-012, BE-013, BE-014, BE-015, BE-016, BE-020, BE-021, BE-025, BE-029, BE-036, FE-007, FE-008, FE-009, FE-011, FE-014, FE-019, FE-020, FE-024, FE-026, FE-027]
nf_ids: [BE-031, BE-032]
api_contract: yes
migrations: yes (new fees tables required - see section 8)
generated_files: docs/api/openapi.yaml, backend/docs/swagger.*, frontend/src/features/fees/api/generated/types.ts, backend/internal/fees/store/gen/
---

## 1. Summary

Fees domain manages fee structures, invoicing, payments, and waivers. Money is stored as integer minor units (KES). Fee waivers require an audit trail (old/new value, proposer, approver per BE-029).

## 2. Requirements

| FR/NFR ID | Rationale |
|-----------|-----------|
| BE-006 | Handlers are thin |
| BE-007 | Handlers return named DTO structs |
| BE-009 | SQL only in store/queries/*.sql via sqlc |
| BE-010 | Stores map sqlc rows to domain types |
| BE-012 | One transaction per service call via TxManager.Run |
| BE-013 | Money is bigint minor units (KES), never floats |
| BE-014 | Keyset pagination |
| BE-015 | Invariants in DB constraints; atomic operations |
| BE-016 | Tenant-scoped queries; cross-tenant identical 404 |
| BE-020 | Routes register with authz.Require(perm) |
| BE-021 | Services obtain Scope from authz.ScopeFor |
| BE-025 | snake_case JSON, amountMinor suffix for money |
| BE-029 | Audit: create/update/delete on fee records in same transaction; fee waivers record old/new value, proposer, approver |
| BE-036 | Per-endpoint tenant-isolation + per-role tests |
| FE-007 | Frontend never sends school/tenant ID |
| FE-008 | Generated API types only |
| FE-009 / FE-019 | useInfiniteQuery + keyset pagination |
| FE-010 | ApiError from {code,message,errors?,request_id} |
| FE-011 | Active school from useActiveSchool() |
| FE-014 | Add/edit/view via intercepting + parallel routes in Dialog/Sheet |
| FE-019 | Tables use shared DataTable |
| FE-020 | Forms use react-hook-form + zodResolver |
| FE-024 | Money integer minor units, formatted via formatKes() |
| FE-026 | FR/NFR IDs referenced; Given/When/Then AC |
| FE-027 | Role-gated screens have per-role tests; modal routes have soft-nav + hard-load tests |
| BE-031 / BE-032 | Logging and Sentry rules |

## 3. Domain Model Affected

### New tables (migration required)

fee_structures: Fee types and amounts per school/class level
fee_invoices: Invoices issued to students
fee_payments: Payments recorded against invoices
fee_waivers: Waiver requests with approval workflow

### Existing tables referenced

students (migration 04), enrollments (migration 05), schools (migration 03)

### New domain

internal/fees/{handler,service,store}/ owning domain.

## 4. API Contract

### New endpoints

| Method | Path | Permission | Description |
|--------|------|------------|-------------|
| GET | /api/v1/fee-structures | fee:read | List fee structures |
| POST | /api/v1/fee-structures | fee:write | Create fee structure |
| GET | /api/v1/fee-invoices | fee:read | List invoices |
| POST | /api/v1/fee-invoices | fee:write | Create invoice |
| GET | /api/v1/fee-payments | fee:read | List payments |
| POST | /api/v1/fee-payments | fee:write | Record payment |
| GET | /api/v1/fee-waivers | fee:read | List waivers |
| POST | /api/v1/fee-waivers | fee:write | Create waiver request |
| PUT | /api/v1/fee-waivers/{id}/approve | fee:write | Approve/reject waiver |

### Request/Response DTOs

FeeStructureDTO:
{ id, school_id, class_level_id, name, amount_minor, currency, created_at, updated_at }

FeeInvoiceDTO:
{ id, student_id, amount_minor, amount_paid_minor, status, due_date, created_at, updated_at }

FeeWaiverDTO:
{ id, fee_invoice_id, old_amount_minor, new_amount_minor, proposer_id, approver_id, status, created_at, updated_at }

### Errors

- 404 NOT_FOUND for missing OR cross-tenant record
- 403 FORBIDDEN if role lacks permission
- 409 CONFLICT if waiver already approved/rejected
- 422 VALIDATION_FAILED for invalid input

## 5. Workflow (Given/When/Then)

### Create fee waiver

Given an authenticated finance officer with fee:write
When POST /api/v1/fee-waivers with invoice_id, new_amount_minor
Then returns 201 with waiver in pending status; audit record written in same transaction

### Approve waiver

Given an authenticated finance officer with fee:write
When PUT /api/v1/fee-waivers/{id}/approve with approver decision
Then returns updated waiver; approver_id and status updated; audit record with old/new value

### Cross-tenant isolation

Given a fee invoice in tenant A
When a user from tenant B requests GET /api/v1/fee-invoices/{id}
Then returns 404 NOT_FOUND (identical to nonexistent, BE-016)

## 6. Implementation Plan

### Backend

1. New migration for fees tables (backend/migrations/00000000000007_fees_domain.sql)
2. internal/fees/store/queries/*.sql sqlc queries
3. internal/fees/store/store.go store methods
4. internal/fees/service/service.go business logic, waiver approval workflow, audit
5. internal/fees/handler/handler.go thin handlers with authz.Require(perm)
6. internal/app/app.go register handlers under protectedAPI

### Frontend

7. frontend/src/features/fees/ feature module

### Generated files

- swagger, frontend types, sqlc gen

## 7. Tests Required

- Handler: tenant isolation + per-role tests for every endpoint
- Service: waiver approval workflow, atomic transactions, audit trail
- Store: real PostgreSQL (testcontainers), RLS policies
- Contract tests against generated spec
- Frontend: per-role tests, modal soft-nav + hard-load tests

## 8. Resolved decisions

| Question | Decision |
|----------|----------|
| Money format | Integer minor units (KES), amount_minor suffix, never floats |
| Waiver audit | old/new value, proposer, approver recorded (BE-029) |
| Delete semantics | Soft delete via deleted_at (BE-041) |
| New migration | Required for fees tables |
| OpenAPI | Regenerate after endpoint changes |

## 9. Links

- SOM-AUTHZ-001 permission matrix
- SOM-ROLE-FINANCE finance capabilities
- BE-020 permission-based authz
- BE-029 audit for fee records
