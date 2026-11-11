---
id: SOM-HEALTH-001
title: Health Domain - Health Records
status: draft
author: agent
date: 2026-11-11
updated: 2026-11-11
fr_ids: [BE-006, BE-007, BE-009, BE-010, BE-012, BE-013, BE-014, BE-015, BE-016, BE-020, BE-021, BE-025, BE-029, BE-036, FE-007, FE-008, FE-009, FE-011, FE-012, FE-014, FE-019, FE-020, FE-023, FE-026, FE-027]
nf_ids: [BE-031, BE-032]
api_contract: yes
migrations: yes (new health records table required)
generated_files: docs/api/openapi.yaml, backend/docs/swagger.*, frontend/src/features/health/api/generated/types.ts, backend/internal/health/store/gen/
---

## 1. Summary

Health domain manages student health records. Sensitive data: staleTime 0, gcTime 0, never persisted. Audit events for create/update/delete on health records written in same transaction per BE-029.

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
| BE-015 | Invariants in DB constraints |
| BE-016 | Tenant-scoped queries; cross-tenant identical 404 |
| BE-020 | Routes register with authz.Require(perm) |
| BE-021 | Services obtain Scope from authz.ScopeFor |
| BE-025 | snake_case JSON, camelCase query params, PUT full body |
| BE-029 | Audit: create/update/delete on health records in same transaction; audit rows hold IDs and changed field names, never sensitive values |
| BE-036 | Per-endpoint tenant-isolation + per-role tests |
| FE-007 | Frontend never sends school/tenant ID |
| FE-008 | Generated API types only |
| FE-009 / FE-019 | useInfiniteQuery + keyset pagination |
| FE-010 | ApiError from {code,message,errors?,request_id} |
| FE-011 | Active school from useActiveSchool() |
| FE-012 | Health queries staleTime: 0, gcTime: 0, never persisted; no PII in logs/URLs/analytics |
| FE-014 | Add/edit/view via intercepting + parallel routes in Dialog/Sheet |
| FE-019 | Tables use shared DataTable |
| FE-020 | Forms use react-hook-form + zodResolver |
| FE-023 | Dates use date-fns + @date-fns/tz Africa/Nairobi; yyyy-MM-dd strings |
| FE-026 | FR/NFR IDs referenced; Given/When/Then AC |
| FE-027 | Role-gated screens have per-role tests; modal routes have soft-nav + hard-load tests |
| BE-031 / BE-032 | Logging and Sentry rules |

## 3. Domain Model Affected

### New tables (migration required)

student_health_records: Health events per student

### Existing tables referenced

students (migration 04)

### New domain

internal/health/{handler,service,store}/ owning domain.

## 4. API Contract

### New endpoints

| Method | Path | Permission | Description |
|--------|------|------------|-------------|
| GET | /api/v1/health-records | health:read | List health records |
| GET | /api/v1/health-records/{id} | health:read | Get single health record |
| POST | /api/v1/health-records | health:write | Create health record |
| PUT | /api/v1/health-records/{id} | health:write | Update health record |

### Request/Response DTOs

HealthRecordDTO:
{ id, student_id, record_type, record_date, description, created_at, updated_at }

### Errors

- 404 NOT_FOUND for missing OR cross-tenant record
- 403 FORBIDDEN if role lacks permission
- 422 VALIDATION_FAILED for invalid input

## 5. Workflow (Given/When/Then)

### Create health record

Given an authenticated headteacher with health:write
When POST /api/v1/health-records with student_id, record_type, record_date, description
Then returns 201 with created health record; audit event written in same transaction

### Cross-tenant isolation

Given a health record in tenant A
When a user from tenant B requests GET /api/v1/health-records/{id}
Then returns 404 NOT_FOUND (identical to nonexistent, BE-016)

## 6. Implementation Plan

### Backend

1. New migration for health table (backend/migrations/00000000000008_health_domain.sql)
2. internal/health/store/queries/*.sql sqlc queries
3. internal/health/store/store.go store methods
4. internal/health/service/service.go business logic, audit
5. internal/health/handler/handler.go thin handlers with authz.Require(perm)
6. internal/app/app.go register handlers under protectedAPI

### Frontend

7. frontend/src/features/health/ feature module

### Generated files

- swagger, frontend types, sqlc gen

## 7. Tests Required

- Handler: tenant isolation + per-role tests for every endpoint
- Service: audit trail in same transaction
- Store: real PostgreSQL (testcontainers), RLS policies
- Contract tests against generated spec
- Frontend: per-role tests, modal soft-nav + hard-load tests

## 8. Resolved decisions

| Question | Decision |
|----------|----------|
| Data sensitivity | staleTime 0, gcTime 0, never persisted (FE-012) |
| Audit | IDs and changed field names only, never sensitive values (BE-029) |
| Delete semantics | Soft delete via deleted_at (BE-041) |
| New migration | Required for health table |
| OpenAPI | Regenerate after endpoint changes |

## 9. Links

- SOM-AUTHZ-001 permission matrix
- SOM-ROLE-HEADTEACHER headteacher capabilities
- BE-020 permission-based authz
- BE-029 audit for health records
