---
id: SOM-ROLE-OWNER
title: Role — Tenant Owner (OWNER)
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

The Tenant Owner (OWNER) is the top-level role within a tenant (school organization). Owns the tenant, manages schools, users, and billing. Can act across all schools in the tenant but cannot manage other tenants.

## 2. Role definition

| Attribute | Value |
|-----------|-------|
| Role key | `owner` |
| DB enum | `OWNER` |
| Scope | Tenant-level (all schools in tenant) |
| Can manage other tenants | No |

## 3. Permissions (from SOM-AUTHZ-001 matrix)

| Permission | Granted | Notes |
|------------|---------|-------|
| `auth:read` | Yes | Read own session/profile |
| `auth:logout` | Yes | Log out / revoke session |
| `school:read` | Yes | Read any school in tenant |
| `school:write` | Yes | Create/update any school in tenant |
| `user:read` | Yes | Read any user in tenant |
| `user:write` | Yes | Create/update any user in tenant |
| `student:read` | Yes | Read any student in any school |
| `student:write` | Yes | Create/update any student |
| `academic:read` | Yes | Read academic data |
| `academic:write` | Yes | Create/update academic data |
| `enrollment:read` | Yes | Read enrollments |
| `enrollment:write` | Yes | Create/update enrollments |
| `fee:read` | Yes | Read fee records |
| `fee:write` | Yes | Create/update fee records |
| `health:read` | Yes | Read health records |
| `health:write` | Yes | Create/update health records |
| `behavior:read` | Yes | Read behavior records |
| `behavior:write` | Yes | Create/update behavior records |
| `audit:read` | Yes | Read audit logs |
| `system:manage` | Yes | System configuration within tenant |

## 4. Screens and capabilities

| Screen | Path | Capability |
|--------|------|------------|
| School management | `/schools` | Create, edit, deactivate schools in tenant |
| User management | `/users` | Invite, edit, deactivate users; assign school memberships |
| Student management | `/students` | Full CRUD across all schools |
| Academic management | `/academics` | Manage subjects, topics, class levels |
| Fee management | `/fees` | Configure fee structures, view all fees |
| Health records | `/health` | View all health records |
| Behavior records | `/behavior` | View all behavior records |
| Audit log | `/audit` | View tenant-wide audit trail |
| Tenant settings | `/settings` | Tenant configuration, Stytch org link |

## 5. Business rules

- Can manage all schools within their tenant only
- Cannot access data from other tenants (RLS enforced, identical 404 cross-tenant)
- Can assign any role to any user within their tenant's schools
- Can create and deactivate schools
- Cannot manage platform-level configuration (`system:manage` is granted but platformoperator owns cross-tenant ops)

## 6. Links

- [SOM-AUTHZ-001](./SOM-AUTHZ-001.md) — permission matrix
- [BE-020](../requirements/backend.md) — permission-based authz
- [BE-016](../requirements/backend.md) — tenant-scoped queries
