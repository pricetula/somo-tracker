---
id: SOM-ROLE-PLATFORMOPERATOR
title: Role — Platform Operator (PLATFORMOPERATOR)
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

The Platform Operator (PLATFORMOPERATOR) is a cross-tenant system administrator. Manages platform-level configuration, tenant onboarding, and can access any school in any tenant. Not assigned to a single school.

## 2. Role definition

| Attribute | Value |
|-----------|-------|
| Role key | `platformoperator` |
| DB enum | `PLATFORMOPERATOR` |
| Scope | All tenants and schools |
| Can manage other tenants | Yes (platform-level) |

## 3. Permissions (from SOM-AUTHZ-001 matrix)

| Permission | Granted | Notes |
|------------|---------|-------|
| `auth:read` | Yes | |
| `auth:logout` | Yes | |
| `school:read` | Yes | Any school, any tenant |
| `school:write` | Yes | Any school, any tenant |
| `user:read` | Yes | Any user, any tenant |
| `user:write` | Yes | Any user, any tenant |
| `student:read` | Yes | Any student |
| `student:write` | Yes | Any student |
| `academic:read` | Yes | Any academic data |
| `academic:write` | Yes | Any academic data |
| `enrollment:read` | Yes | Any enrollment |
| `enrollment:write` | Yes | Any enrollment |
| `fee:read` | Yes | Any fee record |
| `fee:write` | Yes | Any fee record |
| `health:read` | Yes | Any health record |
| `health:write` | Yes | Any health record |
| `behavior:read` | Yes | Any behavior record |
| `behavior:write` | Yes | Any behavior record |
| `audit:read` | Yes | Any tenant's audit log |
| `system:manage` | Yes | Platform configuration |

## 4. Screens and capabilities

| Screen | Path | Capability |
|--------|------|------------|
| Tenant management | `/platform/tenants` | Create, suspend, activate tenants |
| School management | `/platform/schools` | Manage schools across tenants |
| User management | `/platform/users` | Manage users across tenants |
| Onboarding | `/platform/onboarding` | Monitor tenant provisioning |
| Audit log | `/platform/audit` | View cross-tenant audit trail |
| System settings | `/platform/settings` | Platform configuration |

## 5. Business rules

- Not bound to a single school; operates across all tenants
- RLS still applies but platformoperator can access any tenant (admin override)
- Platform operator sessions are subject to the same idle/absolute expiry (BE-019)
- All platformoperator actions are audited (BE-029)

## 6. Links

- [SOM-AUTHZ-001](./SOM-AUTHZ-001.md) — permission matrix
- [BE-020](../requirements/backend.md) — permission-based authz
- [BE-019](../requirements/backend.md) — session expiry
