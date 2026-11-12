---
id: SOM-ROLE-TEACHER
title: Role — Teacher (TEACHER)
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

The Teacher (TEACHER) manages classroom instruction: reads student records, records attendance, grades, health, and behavior. Cannot modify student demographics or academic structure.

## 2. Role definition

| Attribute | Value |
|-----------|-------|
| Role key | `teacher` |
| DB enum | `TEACHER` |
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
| `student:read` | Yes | All students in school |
| `student:write` | No | Cannot modify demographics |
| `academic:read` | Yes | View subjects, topics |
| `academic:write` | No | |
| `enrollment:read` | Yes | View enrollments |
| `enrollment:write` | No | |
| `fee:read` | No | |
| `fee:write` | No | |
| `health:read` | Yes | View health records |
| `health:write` | No | Headteacher records health |
| `behavior:read` | Yes | View behavior records |
| `behavior:write` | No | Headteacher records behavior |
| `audit:read` | No | |
| `system:manage` | No | |

## 4. Screens and capabilities

| Screen | Path | Capability |
|--------|------|------------|
| Class roster | `/students` | View students in own classes/school |
| Student profile | `/students/{id}` | View student details |
| Attendance | `/attendance` | Record daily attendance (future spec) |
| Grades/assessment | `/assessments` | Record grades (future spec) |
| Timetable | `/timetable` | View class schedule |

## 5. Business rules

- Scope limited to assigned school
- Read-only on student demographics (no create/update/delete)
- Cannot manage fees, health, or behavior (headteacher/finance owns those)
- Future specs (attendance, assessments) will add teacher write capabilities

## 6. Links

- [SOM-AUTHZ-001](./SOM-AUTHZ-001.md) — permission matrix
- [BE-020](../requirements/backend.md) — permission-based authz
