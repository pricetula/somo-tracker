---
id: SOM-ACADEMICS-001
title: Academics Domain - Subjects, Topics, Sub-topics
status: draft
author: agent
date: 2026-11-11
updated: 2026-11-11
fr_ids: [BE-006, BE-007, BE-009, BE-010, BE-012, BE-013, BE-014, BE-015, BE-016, BE-020, BE-021, BE-025, BE-036, FE-007, FE-008, FE-009, FE-011, FE-014, FE-019, FE-020, FE-024, FE-026, FE-027]
nf_ids: [BE-031, BE-032]
api_contract: yes
migrations: no (existing subjects, topics, subtopics, class_levels, education_systems, countries, streams, classrooms)
generated_files: docs/api/openapi.yaml, backend/docs/swagger.*, frontend/src/features/academics/api/generated/types.ts, backend/internal/academics/store/gen/
---

## 1. Summary

Academics domain manages the curriculum hierarchy: countries -> education systems -> class levels -> subjects -> topics -> sub-topics. Also manages streams and classrooms (school + class level + stream grouping). Supports multiple countries' curricula (Kenya CBC, Tanzania, Uganda later) via the countries -> education_systems hierarchy.

## 2. Requirements

| FR/NFR ID | Rationale |
|-----------|-----------|
| BE-006 | Handlers are thin |
| BE-007 | Handlers return named DTO structs |
| BE-009 | SQL only in store/queries/*.sql via sqlc |
| BE-010 | Stores map sqlc rows to domain types, no business logic |
| BE-012 | One transaction per service call via TxManager.Run |
| BE-013 | UUIDv7 PKs, timestamptz UTC |
| BE-014 | Keyset pagination for lists |
| BE-015 | Invariants in DB constraints |
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
| FE-024 | CBC terms: subject (learning area), topic (strand), sub-topic (sub-strand) |
| FE-026 | FR/NFR IDs referenced; Given/When/Then AC |
| FE-027 | Role-gated screens have per-role tests; modal routes have soft-nav + hard-load tests |
| BE-031 / BE-032 | Logging and Sentry rules |

## 3. Domain Model Affected

### Entities (existing migrations)

countries (migration 03): ISO 3166-1 alpha-2 reference table
education_systems (migration 03): Belongs to one country (RESTRICT)
class_levels (migration 03): Grade level within education system
subjects (migration 05): Learning area within education system + class level
topics (migration 05): Strand within subject, has sequence_index
subtopics (migration 05): Sub-strand within topic, has sequence_index
streams (migration 05): Class group (color/name) scoped to school
classrooms (migration 05): School + class level + stream combination

### New domain

internal/academics/{handler,service,store}/ owning domain for academics tables.

### Terminology (adopted per product decision)

| CBC term | Adopted term |
|----------|--------------|
| Learning area | Subject |
| Strand | Topic |
| Sub-strand | Sub-topic |

Subjects are scoped to education_system_id + class_level_id. Each school references one education_system_id, fixing its curriculum.

## 4. API Contract

### New endpoints

| Method | Path | Permission | Description |
|--------|------|------------|-------------|
| GET | /api/v1/subjects | academic:read | List subjects (keyset pagination) |
| GET | /api/v1/subjects/{id} | academic:read | Get single subject |
| POST | /api/v1/subjects | academic:write | Create subject |
| PUT | /api/v1/subjects/{id} | academic:write | Update subject |
| GET | /api/v1/topics | academic:read | List topics |
| POST | /api/v1/topics | academic:write | Create topic |
| GET | /api/v1/subtopics | academic:read | List sub-topics |
| POST | /api/v1/subtopics | academic:write | Create sub-topic |
| GET | /api/v1/class-levels | academic:read | List class levels |
| GET | /api/v1/streams | academic:read | List streams |
| POST | /api/v1/streams | academic:write | Create stream |
| GET | /api/v1/classrooms | academic:read | List classrooms |
| POST | /api/v1/classrooms | academic:write | Create classroom |

### Request/Response DTOs

All JSON snake_case. Query params camelCase.

SubjectDTO:
{ id, education_system_id, class_level_id, name, code, description, created_at, updated_at }

TopicDTO:
{ id, subject_id, name, sequence_index, created_at, updated_at }

SubTopicDTO:
{ id, topic_id, name, sequence_index, created_at, updated_at }

StreamDTO / ClassroomDTO: similar structure with school scoping.

### Errors

- 404 NOT_FOUND for missing OR cross-tenant record
- 403 FORBIDDEN if role lacks permission
- 422 VALIDATION_FAILED for invalid input
- 409 CONFLICT if duplicate (subject name within education system + class level)

## 5. Workflow (Given/When/Then)

### List subjects

Given an authenticated headteacher with academic:read
When GET /api/v1/subjects?limit=50&sort=name_asc
Then returns { items, nextCursor } with subjects from their school's education system

### Create subject

Given an authenticated admin with academic:write
When POST /api/v1/subjects with valid body
Then returns 201 with created subject

Given subject name already exists for education system + class level
When POST /api/v1/subjects with duplicate
Then returns 409 CONFLICT

### Cross-tenant isolation

Given a subject in tenant A
When a user from tenant B requests GET /api/v1/subjects/{id}
Then returns 404 NOT_FOUND (identical to nonexistent, BE-016)

## 6. Implementation Plan

### Backend

1. internal/academics/store/queries/subjects.sql sqlc queries
2. internal/academics/store/queries/topics.sql topic queries
3. internal/academics/store/queries/subtopics.sql sub-topic queries
4. internal/academics/store/queries/streams.sql stream queries
5. internal/academics/store/queries/classrooms.sql classroom queries
6. internal/academics/store/store.go store methods
7. internal/academics/service/service.go business logic
8. internal/academics/handler/handler.go thin handlers with authz.Require(perm)
9. internal/app/app.go register academics handlers under protectedAPI group

### Frontend

10. frontend/src/features/academics/ feature module with API, components, hooks, routes

### Generated files

- backend/docs/swagger.* via make generate-swag
- frontend/src/features/academics/api/generated/types.ts via pnpm generate:api
- backend/internal/academics/store/gen/ via sqlc generate

## 7. Tests Required

- Handler tests: tenant isolation (404 cross-tenant) for every endpoint
- Handler tests: per-role permission matrix
- Service tests: repository fakes, business rules
- Store tests: real PostgreSQL (testcontainers), RLS policies
- Contract tests against generated spec
- Frontend: per-role tests, modal soft-nav + hard-load tests

## 8. Resolved decisions

| Question | Decision |
|----------|----------|
| Terminology | Subject/Topic/Sub-topic adopted (replaces CBC learning area/strand/sub-strand) |
| Multi-country | Supported via countries -> education_systems -> subjects hierarchy |
| Delete semantics | Soft delete via deleted_at (BE-041) |
| OpenAPI | Regenerate after endpoint changes |

## 9. Links

- SOM-AUTHZ-001 permission matrix
- SOM-ROLE-HEADTEACHER headteacher capabilities
- SOM-ROLE-TEACHER teacher capabilities
- BE-020 permission-based authz
- backend/migrations/00000000000005_academic_domain.sql academic tables
