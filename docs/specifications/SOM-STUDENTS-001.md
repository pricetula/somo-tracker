---
id: SOM-STUDENTS-001
title: Students Domain - CRUD and Guardian Links
status: draft
author: agent
date: 2026-11-11
updated: 2026-11-11
fr_ids: [BE-006, BE-007, BE-009, BE-010, BE-012, BE-013, BE-014, BE-015, BE-016, BE-020, BE-021, BE-025, BE-036, FE-007, FE-008, FE-009, FE-010, FE-011, FE-014, FE-019, FE-020, FE-026, FE-027]
nf_ids: [BE-031, BE-032]
api_contract: yes
migrations: no (existing students, guardian_student_links tables)
generated_files: docs/api/openapi.yaml, backend/docs/swagger.*, frontend/src/features/students/api/generated/types.ts
---

## 1. Summary

Add a students domain with full CRUD (list, get, create, update, delete) and guardian-student links. This is the first protected domain built on top of the SOM-AUTHZ-001 permission matrix. Students are scoped to a school; guardian links associate a guardian membership with one or more students.

## 2. Requirements

| FR/NFR ID | Rationale |
|-----------|-----------|
| BE-006 | Handlers are thin: bind/validate input, read session context, call one service, map to DTO |
| BE-007 | Handlers return named DTO structs, never sqlc rows or domain types |
| BE-009 | SQL only in `store/queries/*.sql` via sqlc |
| BE-010 | Stores map sqlc rows to domain types, no business logic |
| BE-012 | One transaction per service call via `TxManager.Run` |
| BE-013 | UUIDv7 PKs, `timestamptz` UTC, date-only `date` Africa/Nairobi |
| BE-014 | Keyset pagination: `limit`/`cursor`/`sort`, `{items, nextCursor}` |
| BE-015 | Invariants in DB constraints; atomic operations |
| BE-016 | Every query tenant-scoped; cross-tenant returns identical 404 |
| BE-020 | Routes register with `authz.Require(perm)` or `Public()` |
| BE-021 | Services obtain `Scope` from `authz.ScopeFor(ctx, perm)` |
| BE-025 | snake_case JSON, camelCase query params, PUT full body, success codes |
| BE-036 | Per-endpoint tenant-isolation tests + per-role permission tests |
| FE-007 | Frontend never sends school/tenant ID; base path `/api/v1` |
| FE-008 | Generated API types only; snake→camel mapping at API-function layer |
| FE-009 / FE-019 | `useInfiniteQuery` + keyset pagination |
| FE-010 | `ApiError` from `{code,message,errors?,request_id}` |
| FE-011 | Active school from `useActiveSchool()`, never from URL |
| FE-014 | Add/edit/view via intercepting + parallel routes in Dialog/Sheet |
| FE-019 | Tables use shared `<DataTable>` |
| FE-020 | Forms use `react-hook-form` + `zodResolver` |
| FE-026 | FR/NFR IDs referenced; Given/When/Then AC |
| FE-027 | Role-gated screens have per-role tests; modal routes have soft-nav + hard-load tests |
| BE-031 / BE-032 | Logging and Sentry rules |

## 3. Domain Model Affected

### Entities (existing migrations)

**`students`** (`backend/migrations/00000000000004_student_domain.sql`):
- `student_id` UUID PK (uuidv7)
- `school_id` UUID FK → schools (CASCADE)
- `admission_number` VARCHAR (unique within school)
- `national_number` VARCHAR nullable
- `full_name` VARCHAR
- `date_of_birth` DATE
- `gender` VARCHAR
- `metadata` JSONB default `{}`
- `created_at`, `updated_at`, `deleted_at`

**`guardian_student_links`** (existing):
- `id` UUID PK
- `guardian_id` UUID FK → memberships (CASCADE)
- `student_id` UUID FK → students (CASCADE)
- `relationship_type` VARCHAR
- `is_emergency_contact` BOOLEAN (at most one per student)

### New domain

`internal/students/{handler,service,store}/` — owning domain for students and guardian links. Only this domain's store writes to `students` and `guardian_student_links` tables (BE-003).

### Business rules

- `admission_number` unique within a school (DB constraint)
- At most one emergency contact per student (DB partial unique index)
- Student soft-delete via `deleted_at` (BE-041)
- **Guardian scope:** Guardian can `student:read` only for students linked via `guardian_student_links`. Applies to both list (`GET /api/v1/students` returns only linked students via a dedicated sqlc query) and single-record (`GET /api/v1/students/{id}` returns 404 if not linked). Guardian cannot see another guardian's students.
- **Teacher scope:** Teacher can `student:read` all students in their school.
- Platform operator can read/write across all schools in tenant.
- Owner/admin/headteacher/finance can read/write students in their school per permission matrix.

## 4. API Contract

### New endpoints

| Method | Path | Permission | Description |
|--------|------|------------|-------------|
| GET | `/api/v1/students` | `student:read` | List students (keyset pagination) |
| GET | `/api/v1/students/{id}` | `student:read` | Get single student |
| POST | `/api/v1/students` | `student:write` | Create student |
| PUT | `/api/v1/students/{id}` | `student:write` | Update student (full body) |
| DELETE | `/api/v1/students/{id}` | `student:write` | Delete student (soft delete) |
| GET | `/api/v1/students/{id}/guardians` | `student:read` | List guardians linked to student |
| POST | `/api/v1/students/{id}/guardians` | `student:write` | Link a guardian to a student |
| DELETE | `/api/v1/students/{id}/guardians/{guardian_id}` | `student:write` | Unlink guardian from student |

### Request/Response DTOs

All JSON snake_case (BE-025). Query params camelCase.

**CreateStudentRequest:**
```json
{
  "admission_number": "string",
  "national_number": "string|null",
  "full_name": "string",
  "date_of_birth": "yyyy-MM-dd",
  "gender": "string",
  "metadata": {}
}
```

**CreateStudentResponse:** bare student object (201)

**StudentDTO:**
```json
{
  "student_id": "uuid",
  "school_id": "uuid",
  "admission_number": "string",
  "national_number": "string|null",
  "full_name": "string",
  "date_of_birth": "yyyy-MM-dd",
  "gender": "string",
  "metadata": {},
  "created_at": "RFC3339",
  "updated_at": "RFC3339"
}
```

**List response:** `{ items: StudentDTO[], nextCursor: string|null }`

**GuardianLinkDTO:**
```json
{
  "guardian_id": "uuid",
  "student_id": "uuid",
  "relationship_type": "string",
  "is_emergency_contact": boolean
}
```

### Errors

- 404 NOT_FOUND for missing OR cross-tenant student (identical, BE-016)
- 403 FORBIDDEN if role lacks permission (BE-020)
- 422 VALIDATION_FAILED for invalid input
- 409 CONFLICT if admission_number already exists in school

## 5. Workflow (Given/When/Then)

### List students

**Given** an authenticated teacher with `student:read` permission
**When** `GET /api/v1/students?limit=50&sort=name_asc`
**Then** returns `{ items, nextCursor }` with all students in their school

**Given** an authenticated guardian with membership `membership-123`
**When** `GET /api/v1/students`
**Then** returns only students linked via `guardian_student_links` where `guardian_id` = guardian's membership, using a dedicated sqlc query

**Given** an authenticated guardian requesting `GET /api/v1/students/{id}` for a student they are not linked to
**Then** returns 404 NOT_FOUND (identical to nonexistent student, BE-016)

### Create student

**Given** an authenticated headteacher with `student:write` permission
**When** `POST /api/v1/students` with valid body
**Then** returns 201 with created student; `admission_number` unique within school

**Given** admission_number already exists in the school
**When** `POST /api/v1/students` with duplicate
**Then** returns 409 CONFLICT

### Cross-tenant isolation

**Given** a student in tenant A
**When** an authenticated user from tenant B requests `GET /api/v1/students/{id}`
**Then** returns 404 NOT_FOUND (identical to nonexistent student, BE-016)

### Guardian link

**Given** an authenticated guardian with membership `membership-123`
**When** `POST /api/v1/students/{student_id}/guardians`
**Then** creates link only if guardian's membership matches; returns 403 if guardian tries to link to a student not theirs

## 6. Implementation Plan

### Backend

1. **`backend/internal/students/store/queries/students.sql`** — sqlc queries: `GetStudent`, `ListStudents`, `CreateStudent`, `UpdateStudent`, `DeleteStudent`, `ListGuardianLinks`, `CreateGuardianLink`, `DeleteGuardianLink`. All queries tenant-scoped via `current_tenant_id()` GUC.

2. **`backend/internal/students/store/queries/guardian_links.sql`** — guardian link queries.

3. **`backend/internal/students/store/store.go`** — `New(queries)`, store methods mapping sqlc rows → domain types, Postgres errors → `apperr`.

4. **`backend/internal/students/service/service.go`** — business logic: admission_number uniqueness, guardian link validation, soft delete, transaction management via `TxManager.Run`.

5. **`backend/internal/students/handler/handler.go`** — thin handlers: bind DTO, call service, map to response DTO. Register with `authz.Require(perm)`.

6. **`backend/internal/app/app.go`** — register protected domain handlers:
   ```go
   studentsAPI := protectedAPI.Group("/students")
   studentsAPI.Use(authz.Require(authz.PermStudentRead)) // or per-route
   studentsHandler.RegisterRoutes(studentsAPI)
   ```

### Frontend

7. **`frontend/src/features/students/`** — feature module:
   - `api/get-students.ts`, `api/create-student.ts`, etc. (snake→camel mapping)
   - `api/generated/types.ts` (regenerated from OpenAPI)
   - `components/`, `hooks/`, `query-keys.ts`, `schemas.ts`, `index.ts`
   - Routes: `/students` (list), `/students/@modal/(.)add`, `/students/@modal/(.)[id]`, `/students/@modal/(.)[id]/edit`

8. **`docs/api/openapi.yaml`** — add student endpoints (backend-owned)

### Generated files

- `backend/docs/swagger.*` via `make generate-swag`
- `frontend/src/features/students/api/generated/types.ts` via `pnpm generate:api`
- `backend/internal/students/store/gen/` via `sqlc generate`

## 7. Tests Required

### Backend (per BE-036)

- `handler/` — tenant isolation: cross-tenant returns 404 for every endpoint
- `handler/` — per-role permission matrix: owner/admin/headteacher/teacher/finance/guardian/platformoperator
- `service/` — repository fakes, business rules (admission_number uniqueness, guardian link validation)
- `store/` — real PostgreSQL (testcontainers), RLS policies
- Contract tests against generated spec

### Frontend (per FE-026 / FE-027)

- List page: per-role tests (roles that can/cannot see students)
- Add/Edit modal: soft-nav and hard-load tests
- Form validation error mapping (`errors[].field` → `setError`)

## 8. Resolved decisions

| Question | Decision |
|----------|----------|
| Guardian scoping | Guardian checks role first, then a dedicated sqlc query fetches only linked students. Applies to both list and single-record views; guardian cannot see unlinked students (404). |
| Teacher scope | Teacher can `student:read` all students in their school. |
| Delete semantics | Soft delete via `deleted_at` (BE-041). |
| OpenAPI regeneration | Yes — `docs/api/openapi.yaml` updated, then `make generate-swag` and `pnpm generate:api` to regenerate frontend types and swagger docs. |

## 9. Links

- [SOM-AUTHZ-001](./SOM-AUTHZ-001.md) — authz permission matrix (prerequisite)
- [BE-016](../requirements/backend.md) — tenant-scoped queries
- [BE-020](../requirements/backend.md) — permission-based authz
- [BE-021](../requirements/backend.md) — ScopeFor
- [backend/migrations/00000000000004_student_domain.sql](../../../backend/migrations/00000000000004_student_domain.sql) — students table
- [docs/api/openapi.yaml](../../api/openapi.yaml) — API contract (to be updated)
