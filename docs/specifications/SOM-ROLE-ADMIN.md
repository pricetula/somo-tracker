---
id: SOM-ROLE-ADMIN
title: Role — School Admin (ADMIN)
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

The School Admin (ADMIN) manages a single school's operations: students, staff, academics, and fees within their assigned school. Cannot manage other schools or tenant-level settings.

## 2. Role definition

| Attribute | Value |
|-----------|-------|
| Role key | `admin` |
| DB enum | `ADMIN` |
| Scope | Single school |
| Can manage other tenants | No |

## 3. Permissions (from SOM-AUTHZ-001 matrix)

| Permission | Granted | Notes |
|------------|---------|-------|
| `auth:read` | Yes | |
| `auth:logout` | Yes | |
| `school:read` | Yes | Own school only |
| `school:write` | No | Cannot create/edit schools |
| `user:read` | Yes | Users in own school |
| `user:write` | Yes | Invite/edit users in own school |
| `student:read` | Yes | All students in school |
| `student:write` | Yes | Full student CRUD |
| `academic:read` | Yes | |
| `academic:write` | Yes | |
| `enrollment:read` | Yes | |
| `enrollment:write` | Yes | |
| `fee:read` | Yes | |
| `fee:write` | Yes | |
| `health:read` | Yes | |
| `health:write` | Yes | |
| `behavior:read` | Yes | |
| `behavior:write` | Yes | |
| `audit:read` | No | Owner/platformoperator only |
| `system:manage` | No | |

## 4. Screens and capabilities

| Screen | Path | Capability |
|--------|------|------------|
| Student management | `/students` | Full CRUD in own school |
| Staff management | `/staff` | Invite/edit users in own school |
| Class management | `/classes` | Manage classrooms, streams |
| Academic management | `/academics` | Manage subjects, topics for school |
| Fee management | `/fees` | Configure and collect fees |
| Health records | `/health` | View/record student health |
| Behavior records | `/behavior` | Record student behavior |
| Enrollment | `/enrollments` | Enroll/transfer/withdraw students |

## 5. Business rules

- Scope limited to assigned school (RLS enforced via `school_id`)
- Cannot create or deactivate schools
- Cannot manage users outside own school
- Cannot view audit logs
- Can assign teacher/guardian/finance roles to users in own school

## 6. Links

- [SOM-AUTHZ-001](./SOM-AUTHZ-001.md) — permission matrix
- [BE-020](../requirements/backend.md) — permission-based authz
