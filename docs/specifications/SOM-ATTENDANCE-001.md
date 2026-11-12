---
id: SOM-ATTENDANCE-001
title: Attendance Management
status: draft
author: agent
date: 2026-11-11
updated: 2026-11-11
fr_ids: [BE-006, BE-007, BE-009, BE-010, BE-012, BE-013, BE-014, BE-015, BE-016, BE-020, BE-021, BE-025, BE-029, BE-036, FE-007, FE-008, FE-009, FE-011, FE-014, FE-019, FE-020, FE-023, FE-026, FE-027]
nf_ids: [BE-031, BE-032]
api_contract: yes
migrations: yes (new attendance table required)
generated_files: docs/api/openapi.yaml, backend/docs/swagger.*, frontend/src/features/attendance/api/generated/types.ts, backend/internal/attendance/store/gen/
---

## 1. Summary

Attendance management records whether a student was present, absent, late, or excused for a specific lesson. A lesson is identified by a timetable entry (class + day + slot + subject + teacher). Attendance is marked per student per lesson per date. Only the teacher assigned to that timetable entry may mark attendance. After the end of the school day, attendance can still be marked but carries a `late_marking` flag.

## 2. Requirements

| FR/NFR ID | Rationale |
|-----------|-----------|
| BE-006 | Handlers are thin |
| BE-007 | Handlers return named DTO structs |
| BE-009 | SQL only in store/queries/*.sql via sqlc |
| BE-010 | Stores map sqlc rows to domain types |
| BE-012 | One transaction per service call via TxManager.Run |
| BE-013 | UUIDv7 PKs, timestamptz UTC, school-day dates date Africa/Nairobi |
| BE-014 | Keyset pagination for list endpoints |
| BE-015 | Invariants in DB constraints; atomic operations |
| BE-016 | Tenant-scoped queries; cross-tenant identical 404 |
| BE-020 | Routes register with authz.Require(perm) |
| BE-021 | Services obtain Scope from authz.ScopeFor |
| BE-025 | snake_case JSON, camelCase query params, PUT full body |
| BE-029 | Audit: create/update/delete on attendance records in same transaction |
| BE-036 | Per-endpoint tenant-isolation + per-role tests |
| FE-007 | Frontend never sends school/tenant ID |
| FE-008 | Generated API types only |
| FE-009 / FE-019 | useInfiniteQuery + keyset pagination |
| FE-011 | Active school from useActiveSchool() |
| FE-014 | Add/edit via Dialog/Sheet intercepting routes |
| FE-019 | Tables use shared DataTable |
| FE-020 | Forms use react-hook-form + zodResolver |
| FE-023 | Dates use date-fns + @date-fns/tz Africa/Nairobi; yyyy-MM-dd strings |
| FE-026 | FR/NFR IDs referenced; Given/When/Then AC |
| FE-027 | Role-gated screens have per-role tests; modal routes have soft-nav + hard-load tests |
| BE-031 / BE-032 | Logging and Sentry rules |

## 3. Domain Model Affected

### New tables (migration required)

| Table | Description |
|-------|-------------|
| attendance_records | One row per student per lesson per date: enrollment + timetable entry + date + status |

### Existing tables referenced

- `enrollments` (migration 05) — enrollment_id references enrollments.id (student + classroom + term)
- `timetable_entries` (new timetable migration) — timetable_entry_id references timetable_entries.id (which lesson: class + day + slot + subject + teacher)
- `memberships` (migration 03) — marked_by references memberships.id (teacher who marked the attendance)
- `students` (migration 04) — via enrollment
- `classrooms` (migration 05) — via enrollment / timetable entry

### RLS

New `attendance_records` table is enabled for RLS with policies keyed on `tenant_id` (migration 06 pattern via `current_tenant_id()`). The service sets `app.current_tenant_id` from session context at transaction start. Cross-tenant lookups return identical 404 per BE-016.

### New domain

`internal/attendance/{handler,service,store}/` owning domain per BE-003.

### Data model

AttendanceRecord:
  id, tenant_id, enrollment_id, timetable_entry_id, date, status, late_marking, marked_by, created_at, updated_at, deleted_at
  -- unique (enrollment_id, timetable_entry_id, date)
  -- tenant_id denormalized for direct tenant-scoped queries and RLS
  -- status: attendance_status ENUM (PRESENT, ABSENT, LATE, EXCUSED)
  -- late_marking BOOLEAN default false

New ENUM type:
  attendance_status: PRESENT, ABSENT, LATE, EXCUSED

### Day cycle

Attendance dates are school-day dates (DATE, Africa/Nairobi). A lesson is tied to a day-of-week (1-7) via the timetable entry; the actual attendance date must fall on that day of the week (service-layer validation).

## 4. UI

Attendance page:
- Teacher selects a class (from their assigned timetable entries) and a date.
- Grid lists students in that class for that lesson, each with a status select (Present / Absent / Late / Excused).
- [Save] button submits all marks for the lesson.
- After end of day, marks can still be saved but each record is flagged late_marking = true.
- Read-only view for guardians (own children), headteacher, admin, finance.

## 5. Business Rules

### Marking rules

1. **Teacher assignment gate:** A teacher can only mark attendance for a timetable entry where `timetable_entries.teacher_id` equals the teacher's membership id (`marked_by`). A headteacher or admin may mark for any timetable entry in their school.
2. **One record per student per lesson per date:** DB unique constraint `(enrollment_id, timetable_entry_id, date)`. Re-marking upserts the same record.
3. **Status:** Must be one of `PRESENT`, `ABSENT`, `LATE`, `EXCUSED` (DB ENUM).
4. **Enrollment scope:** Can only mark for enrollments that are ACTIVE in the term referenced by the timetable entry's class.
5. **Date validity:** The attendance date must fall within the term's start/end dates and match the day-of-week of the timetable entry.
6. **Late marking:** `late_marking = true` when attendance is marked after the end of the school day. During the school day, `late_marking = false`. After end of day, marking is still allowed but flagged.

### Protecting existing data (backend-enforced)

- Soft delete via `deleted_at` (BE-041). A deleted attendance record is excluded from reports.
- A record's `timetable_entry_id` or `enrollment_id` cannot be changed after creation; to correct, delete and re-create.
- Cross-tenant: identical 404 for any record not in the session's tenant.

### Conflict checks (backend)

- Reject if the teacher is not the assigned teacher for the timetable entry (403 FORBIDDEN).
- Reject if a record already exists for the same enrollment + timetable entry + date (409 CONFLICT); use upsert semantics instead.
- Reject if the enrollment is not ACTIVE (422 VALIDATION_FAILED).
- Reject if the date is outside the term or does not match the timetable entry's day-of-week (422 VALIDATION_FAILED).

## 6. API Contract

### Endpoints

| Method | Path | Permission | Description |
|--------|------|------------|-------------|
| GET | /api/v1/attendance | attendance:read | List attendance records (?enrollmentId=&timetableEntryId=&date=&status=&classId=) |
| GET | /api/v1/attendance/{id} | attendance:read | Get single attendance record |
| POST | /api/v1/attendance | attendance:write | Mark/upsert a single attendance record |
| POST | /api/v1/attendance/mark | attendance:write | Bulk mark attendance for a lesson (timetableEntryId + date + records[]) |
| PUT | /api/v1/attendance/{id} | attendance:write | Update attendance record (full body) |
| DELETE | /api/v1/attendance/{id} | attendance:write | Soft-delete attendance record |

### Request/Response DTOs

All JSON snake_case. Query params camelCase.

AttendanceRecordDTO:
{ id, enrollment_id, timetable_entry_id, date, status, late_marking, marked_by, created_at, updated_at }

MarkAttendanceRequest (single):
{ enrollment_id, timetable_entry_id, date, status }

MarkAttendanceRequest (bulk):
{ timetable_entry_id, date, records: [{ enrollment_id, status }] }

### Errors

- 404 NOT_FOUND for missing OR cross-tenant record
- 403 FORBIDDEN if role lacks permission OR teacher not assigned to the timetable entry
- 409 CONFLICT if duplicate attendance for same enrollment + timetable entry + date
- 422 VALIDATION_FAILED for invalid status, date outside term, day-of-week mismatch, inactive enrollment

## 7. Workflow (Given/When/Then)

### Mark attendance (single)

Given a teacher assigned to timetable entry Monday 08:00-09:00 Math for Class 4A
When POST /api/v1/attendance with enrollment_id, timetable_entry_id, date, status=PRESENT
Then returns 201 with attendance record; late_marking=false if during school day

Given the same teacher attempts to mark for a timetable entry assigned to another teacher
When POST /api/v1/attendance
Then returns 403 FORBIDDEN

### Bulk mark attendance

Given a teacher opens attendance for Class 4A, Monday 08:00-09:00, date 2026-01-12
When POST /api/v1/attendance/mark with { timetable_entry_id, date, records: [{ enrollment_id, status }] }
Then creates/upserts one attendance record per student; returns 201

### Late marking after end of day

Given it is after the end of the school day
When a teacher marks attendance for a lesson
Then each record has late_marking=true

### Cross-tenant isolation

Given an attendance record in tenant A
When a user from tenant B requests GET /api/v1/attendance/{id}
Then returns 404 NOT_FOUND (identical to nonexistent, BE-016)

### Guardian read scope

Given a guardian linked to student S via guardian_student_links
When GET /api/v1/attendance?enrollmentId=<S's enrollment>
Then returns only attendance records for that guardian's linked students

## 8. Implementation Plan

### Backend

1. New migration: backend/migrations/0000000000000X_attendance_domain.sql
   - CREATE TYPE attendance_status AS ENUM ('PRESENT', 'ABSENT', 'LATE', 'EXCUSED')
   - CREATE TABLE attendance_records (...)
   - RLS enable + policy + FORCE ROW LEVEL SECURITY
   - trigger_set_timestamp on attendance_records
2. internal/attendance/store/queries/attendance_records.sql
3. internal/attendance/store/store.go
4. internal/attendance/service/service.go — teacher assignment gate, upsert, late_marking determination, audit
5. internal/attendance/handler/handler.go — thin handlers with authz.Require(perm)
6. internal/app/app.go — register handlers under protectedAPI group

### Frontend

7. frontend/src/features/attendance/ — feature module with attendance page, API, hooks

### Generated files

- docs/api/openapi.yaml updated (backend-owned)
- backend/docs/swagger.* via make generate-swag
- frontend/src/features/attendance/api/generated/types.ts via pnpm generate:api
- backend/internal/attendance/store/gen/ via sqlc generate

### Authz matrix

Add `attendance:read` and `attendance:write` to SOM-AUTHZ-001 permission matrix:
- attendance:read: OWNER, ADMIN, HEADTEACHER, TEACHER, GUARDIAN (own children), FINANCE, PLATFORMOPERATOR
- attendance:write: OWNER, ADMIN, HEADTEACHER, TEACHER (own class), PLATFORMOPERATOR

## 9. Tests Required

### Backend (per BE-036)

- handler/ — tenant isolation: cross-tenant returns 404 for every endpoint
- handler/ — per-role permission matrix: owner/admin/headteacher/teacher/finance/guardian/platformoperator
- handler/ — teacher assignment gate: teacher attempting to mark for another teacher's timetable entry returns 403
- service/ — repository fakes, business rules (upsert, late_marking, date validation, enrollment active check)
- store/ — real PostgreSQL (testcontainers), RLS policies, unique constraint
- Contract tests against generated spec

### Frontend (per FE-026 / FE-027)

- Attendance page: per-role tests (roles that can/cannot mark/view)
- Mark modal: soft-nav and hard-load tests
- Form validation error mapping (errors[].field → setError)
- Late marking flag displayed visually after end of day

## 10. Resolved decisions

| Question | Decision |
|----------|----------|
| Granularity | Per student per lesson (timetable entry) per date (Option B) |
| Who marks | Only the teacher assigned to the timetable entry's class/subject; headteacher/admin may mark for any entry in their school |
| Status values | PRESENT, ABSENT, LATE, EXCUSED (DB ENUM attendance_status) |
| Primary FK | enrollments.id (student + classroom + academic term) |
| Lesson FK | timetable_entries.id (which lesson: class + day + slot + subject + teacher) |
| Late marking | late_marking BOOLEAN; true when marked after end of school day (UTC midnight); marking still allowed after end of day |
| Delete semantics | Soft delete via deleted_at (BE-041) |
| Tenant scoping | tenant_id denormalized on attendance_records; RLS policies keyed to tenant_id per BE-016; frontend never sends tenant/school ID |
| API paths | /api/v1/attendance (list/get/update/delete); POST /api/v1/attendance/mark for bulk marking (kebab-case plural nouns per BE-025) |
| Permission names | attendance:read / attendance:write (to be added to SOM-AUTHZ-001 matrix) |

## 11. Open Questions

None.

## Links

- SOM-AUTHZ-001 permission matrix (attendance:read / attendance:write to be added)
- SOM-ROLE-TEACHER — timetable users, attendance marking capability
- SOM-TIMETABLE-001 — timetable entries (lesson source for attendance)
- BE-020 permission-based authz
- BE-029 audit for attendance records
- BE-041 soft delete
- backend/migrations/00000000000005_academic_domain.sql — enrollments, classrooms tables
- backend/migrations/00000000000004_student_domain.sql — students table