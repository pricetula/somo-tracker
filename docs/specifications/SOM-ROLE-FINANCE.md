---
id: SOM-ROLE-FINANCE
title: Role — Finance (FINANCE)
status: draft
author: agent
date: 2026-11-11
updated: 2026-11-11
fr_ids: [BE-020, BE-021, BE-016, BE-017]
nf_ids: [BE-031, BE-032]
api_contract: no
migrations: no
generated_files: []
---

## 1. Summary

The Finance (FINANCE) role manages school finances: fee structures, invoicing, payments, and waivers. Can read students but cannot modify demographics or academics.

## 2. Role definition

| Attribute | Value |
|-----------|-------|
| Role key | `finance` |
| DB enum | `FINANCE` |
| Scope | Single school |
| Can manage other tenants | No |

## 3. Permissions (from SOM-AUTHZ-001 matrix)

| Permission | Granted | Notes |
|------------|---------|-------|
| `auth:read` | Yes | |
| `auth:logout` | Yes | |
| `school:read` | Yes | Own school only |
| `school:write` | No | |
| `user:read` | No | |
| `user:write` | No | |
| `student:read` | Yes | View student records for fee context |
| `student:write` | No | |
| `academic:read` | No | |
| `academic:write` | No | |
| `enrollment:read` | Yes | View enrollment for fee allocation |
| `enrollment:write` | No | |
| `fee:read` | Yes | All fee records in school |
| `fee:write` | Yes | Create/update fee records, waivers |
| `health:read` | No | |
| `health:write` | No | |
| `behavior:read` | No | |
| `behavior:write` | No | |
| `audit:read` | No | |
| `system:manage` | No | |

## 4. Screens and capabilities

| Screen | Path | Capability |
|--------|------|------------|
| Fee catalogue | `/fees` | Manage fee types and amounts |
| Invoicing | `/fees/invoices` | Generate and send invoices |
| Payments | `/fees/payments` | Record payments |
| Waivers | `/fees/waivers` | Approve/revoke fee waivers (audit trail) |
| Student roster | `/students` | View students for fee allocation |

## 5. Business rules

- Scope limited to assigned school
- Fee waivers require audit trail (old/new value, proposer, approver per BE-029)
- Cannot modify student demographics or academic records
- Can read student/enrollment data only for fee allocation context

## 6. Links

- [SOM-AUTHZ-001](./SOM-AUTHZ-001.md) — permission matrix
- [BE-029](../requirements/backend.md) — audit for fee waivers
- [BE-020](../requirements/backend.md) — permission-based authz
