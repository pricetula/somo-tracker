---
id: SOM-ENROLLMENTS-001
title: Enrollments Domain - Academic Years, Terms, Enrollments
status: draft
author: agent
date: 2026-11-11
updated: 2026-11-11
fr_ids: [BE-006, BE-007, BE-009, BE-010, BE-012, BE-013, BE-014, BE-015, BE-016, BE-020, BE-021, BE-025, BE-036, FE-007, FE-008, FE-009, FE-011, FE-014, FE-019, FE-020, FE-023, FE-026, FE-027]
nf_ids: [BE-031, BE-032]
api_contract: yes
migrations: no (existing academic_years, terms, enrollments, classrooms)
generated_files: docs/api/openapi.yaml, backend/docs/swagger.*, frontend/src/features/enrollments/api/generated/types.ts, backend/internal/enrollments/store/gen/
---

## 1. Summary

Enrollments domain manages academic years, terms, and student enrollments. Enforces "one active enrollment per student per term" at the DB level. Academic years and terms are scoped to a school.

## 2. Requirements

| FR/NFR ID | Rationale |
|-----------|-----------|
| BE-006 | Handlers are thin |
| BE-007 | Handlers return named DTO structs |
| BE-009 | SQL only in store/queries/*.sql via sqlc |
| BE-010 | Stores map sqlc rows to domain types |
| BE-012 | One transaction per service call via TxManager.Run |
| BE-013 | UUIDv7 PKs, timestamptz UTC, school-day dates date Africa/Nairobi |
| BE-014 | Keyset pagination |
| BE-015 | Invariants in DB constraints (unique active enrollment per student per term) |
| BE-016 | Tenant-scoped queries; cross-tenant identical 404 |
| BE-020 | Routes register with authz.Require(perm) |
| BE-021 | Services obtain Scope from authz.ScopeFor |
| BE-025 | snake_case JSON, camelCase query params, PUT full body |
| BE-036 | Per-endpoint tenant-isolation + per-role tests |
| FE-007 | Frontend never sends school/tenant ID |
| FE-008 | Generated API types only |
| FE-009 / FE-019 | useInfiniteQuery + keyset pagination |
| FE-010 | ApiError from {code,message,errors?,request_id} |
| FE-011 | Active school from useActiveSchool() |
| FE-014 | Add/edit/view via intercepting + parallel routes in Dialog/Sheet |
| FE-019 | Tables use shared DataTable |
| FE-020 | Forms use react-hook-form + zodResolver |
| FE-023 | Dates use date-fns + @date-fns/tz Africa/Nairobi; yyyy-MM-dd strings |
| FE-026 | FR/NFR IDs referenced; Given/When/Then AC |
| FE-027 | Role-gated screens have per-role tests; modal routes have soft-nav + hard-load tests |
| BE-031 / BE-032 | Logging and Sentry rules |

## 3. Domain Model Affected

### Entities (existing migrations)

academic_years (migration 05): Scoped to school, year_label, is_current (unique per school)
terms (migration 05): 1-3 per academic year, sequence_index, label, start_date, end_date
enrollments (migration 05): One active per student per term (partial unique index), enrollment_status enum
classrooms (migration 05): School + class level + stream

### New domain

internal/enrollments/{handler,service,store}/ owning domain.

### Enrollment status enum

ACTIVE, TRANSFERRED, GRADUATED, WITHDRAWN

## 4. API Contract

### New endpoints

| Method | Path | Permission | Description |
|--------|------|------------|-------------|
| GET | /api/v1/academic-years | enrollment:read | List academic years |
| POST | /api/v1/academic-years | enrollment:write | Create academic year |
| GET | /api/v1/terms | enrollment:read | List terms |
| POST | /api/v1/terms | enrollment:write | Create term |
| GET | /api/v1/enrollments | enrollment:read | List enrollments |
| GET | /api/v1/enrollments/{id} | enrollment:read | Get single enrollment |
| POST | /api/v1/enrollments | enrollment:write | Create enrollment |
| PUT | /api/v1/enrollments/{id} | enrollment:write | Update enrollment (status change) |

### Request/Response DTOs

EnrollmentDTO:
{ id, student_id, classroom_id, term_id, status, created_at, updated_at }

AcademicYearDTO:
{ id, school_id, year_label, is_current, created_at, updated_at }

TermDTO:
{ id, academic_year_id, sequence_index, label, start_date, end_date, created_at, updated_at }

### Errors

- 404 NOT_FOUND for missing OR cross-tenant record
- 403 FORBIDDEN if role lacks permission
- 409 CONFLICT if enrollment already active for student + term (DB unique violation)
- 422 VALIDATION_FAILED for invalid input

## 5. Workflow (Given/When/Then)

### Create enrollment

Given an authenticated headteacher with enrollment:write
When POST /api/v1/enrollments with student_id, classroom_id, term_id
Then returns 201 with created enrollment; student must not already have ACTIVE enrollment in that term

Given student already has ACTIVE enrollment in the term
When POST /api/v1/enrollments with duplicate
Then returns 409 CONFLICT (DB enforced partial unique index)

### Cross-tenant isolation

Given an enrollment in tenant A
When a user from tenant B requests GET /api/v1/enrollments/{id}
Then returns 404 NOT_FOUND (identical to nonexistent, BE-016)

## 6. Implementation Plan

### Backend

1. internal/enrollments/store/queries/academic_years.sql
2. internal/enrollments/store/queries/terms.sql
3. internal/enrollments/store/queries/enrollments.sql
4. internal/enrollments/store/store.go
5. internal/enrollments/service/service.go
6. internal/enrollments/handler/handler.go
7. internal/app/app.go register handlers under protectedAPI

### Frontend

8. frontend/src/features/enrollments/ feature module

### Generated files

- swagger, frontend types, sqlc gen

## 7. Tests Required

- Handler: tenant isolation + per-role tests for every endpoint
- Service: unique enrollment constraint, business rules
- Store: real PostgreSQL, RLS policies
- Contract tests against generated spec
- Frontend: per-role tests, modal soft-nav + hard-load tests

## 8. Resolved decisions

| Question | Decision |
|----------|----------|
| Active enrollment uniqueness | DB enforced via partial unique index enrollments_active_student_term_uniq |
| Term count | 1-3 terms per academic year (sequence_index 1, 2, 3) |
| Delete semantics | Soft delete via deleted_at (BE-041) |
| OpenAPI | Regenerate after endpoint changes |

## 9. Links

- SOM-AUTHZ-001 permission matrix
- SOM-ROLE-HEADTEACHER headteacher enrollment capabilities
- SOM-ROLE-TEACHER teacher enrollment read
- BE-020 permission-based authz
- backend/migrations/00000000000005_academic_domain.sql enrollment tables
