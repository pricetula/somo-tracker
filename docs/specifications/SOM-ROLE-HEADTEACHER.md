---
id: SOM-ROLE-HEADTEACHER
title: Role — Headteacher (HEADTEACHER)
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

The Headteacher (HEADTEACHER) is the academic leader of a school. Manages students, teachers, academics, and enrollment. Cannot manage fee records or system configuration.

## 2. Role definition

| Attribute | Value |
|-----------|-------|
| Role key | `headteacher` |
| DB enum | `HEADTEACHER` |
| Scope | Single school |
| Can manage other tenants | No |

## 3. Permissions (from SOM-AUTHZ-001 matrix)

| Permission | Granted | Notes |
|------------|---------|-------|
| `auth:read` | Yes | |
| `auth:logout` | Yes | |
| `school:read` | Yes | Own school only |
| `school:write` | No | |
| `user:read` | Yes | Staff in own school |
| `user:write` | No | Cannot invite/edit users |
| `student:read` | Yes | All students in school |
| `student:write` | Yes | Full student CRUD |
| `academic:read` | Yes | |
| `academic:write` | Yes | Manage subjects, topics |
| `enrollment:read` | Yes | |
| `enrollment:write` | Yes | Enroll/transfer/withdraw |
| `fee:read` | Yes | View fees only |
| `fee:write` | No | Finance manages fees |
| `health:read` | Yes | |
| `health:write` | Yes | Record health events |
| `behavior:read` | Yes | |
| `behavior:write` | Yes | Record behavior incidents |
| `audit:read` | No | |
| `system:manage` | No | |

## 4. Screens and capabilities

| Screen | Path | Capability |
|--------|------|------------|
| Student management | `/students` | Full CRUD in own school |
| Academic management | `/academics` | Manage subjects, topics, class levels |
| Enrollment | `/enrollments` | Enroll, transfer, graduate, withdraw students |
| Health records | `/health` | Record student health events |
| Behavior records | `/behavior` | Record behavior incidents |
| Timetable | `/timetable` | View class schedules |

## 5. Business rules

- Scope limited to assigned school
- Cannot manage user accounts (delegated to Admin/Owner)
- Cannot manage fees (delegated to Finance)
- Can record health and behavior events for any student in school

## 6. Links

- [SOM-AUTHZ-001](./SOM-AUTHZ-001.md) — permission matrix
- [BE-020](../requirements/backend.md) — permission-based authz
