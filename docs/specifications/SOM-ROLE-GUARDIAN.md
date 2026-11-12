---
id: SOM-ROLE-GUARDIAN
title: Role — Guardian (GUARDIAN)
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

The Guardian (GUARDIAN) is a parent or legal guardian linked to one or more students. Read-only access to their own children's records: health, behavior, and enrollment status. Cannot modify any records.

## 2. Role definition

| Attribute | Value |
|-----------|-------|
| Role key | `guardian` |
| DB enum | `GUARDIAN` |
| Scope | Linked students only (object-level) |
| Can manage other tenants | No |

## 3. Permissions (from SOM-AUTHZ-001 matrix)

| Permission | Granted | Notes |
|------------|---------|-------|
| `auth:read` | Yes | |
| `auth:logout` | Yes | |
| `school:read` | No | |
| `school:write` | No | |
| `user:read` | No | |
| `user:write` | No | |
| `student:read` | Yes | Only linked students |
| `student:write` | No | |
| `academic:read` | No | |
| `academic:write` | No | |
| `enrollment:read` | Yes | Only linked students |
| `enrollment:write` | No | |
| `fee:read` | No | |
| `fee:write` | No | |
| `health:read` | Yes | Only linked students |
| `health:write` | No | |
| `behavior:read` | Yes | Only linked students |
| `behavior:write` | No | |
| `audit:read` | No | |
| `system:manage` | No | |

## 4. Screens and capabilities

| Screen | Path | Capability |
|--------|------|------------|
| Dashboard | `/guardian` | Overview of linked children |
| Child profile | `/guardian/children/{id}` | View child's details |
| Health records | `/guardian/children/{id}/health` | View child's health events |
| Behavior records | `/guardian/children/{id}/behavior` | View child's behavior incidents |
| Enrollment | `/guardian/children/{id}/enrollment` | View child's class/term enrollment |

## 5. Business rules

- Can only access students linked via `guardian_student_links` where `guardian_id` = guardian's membership
- Single-record access (`GET /api/v1/students/{id}`) returns 404 if not linked (identical to nonexistent)
- List access (`GET /api/v1/students`) auto-filters to linked students via dedicated sqlc query
- Cannot modify any records (read-only)
- Emergency contact flag managed by school staff, not guardian

## 6. Links

- [SOM-AUTHZ-001](./SOM-AUTHZ-001.md) — permission matrix
- [SOM-STUDENTS-001](./SOM-STUDENTS-001.md) — students domain (guardian links)
- [BE-020](../requirements/backend.md) — permission-based authz
