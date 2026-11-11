---
id: SOM-TIMETABLE-001
title: Timetable Management
status: draft
author: agent
date: 2026-11-11
updated: 2026-11-11
fr_ids: [BE-006, BE-007, BE-009, BE-010, BE-012, BE-013, BE-014, BE-015, BE-016, BE-020, BE-021, BE-025, BE-029, BE-036, FE-007, FE-008, FE-009, FE-011, FE-014, FE-019, FE-020, FE-026, FE-027]
nf_ids: [BE-031, BE-032]
api_contract: yes
migrations: yes (new timetable tables required)
generated_files: docs/api/openapi.yaml, backend/docs/swagger.*, frontend/src/features/timetable/api/generated/types.ts, backend/internal/timetable/store/gen/
---

## 1. Summary

Timetable management allows teachers to know which class and subject they have at a given time. It consists of timetable templates (named sets of time slots), time slots (start/end pairs), day cycles (1-7, Monday-Sunday), and timetable entries (class + day + slot + subject + teacher). Every day uses the same slots; no per-day slot variations. Templates are reusable across classes but not linked to them — the user selects a class manually.

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
| BE-029 | Audit: create/update/delete on timetable entries in same transaction |
| BE-036 | Per-endpoint tenant-isolation + per-role tests |
| FE-007 | Frontend never sends school/tenant ID |
| FE-008 | Generated API types only |
| FE-009 / FE-019 | useInfiniteQuery + keyset pagination |
| FE-010 | ApiError from {code,message,errors?,request_id} |
| FE-011 | Active school from useActiveSchool() |
| FE-014 | Add/edit via Dialog/Sheet intercepting routes |
| FE-019 | Tables use shared DataTable |
| FE-020 | Forms use react-hook-form + zodResolver |
| FE-026 | FR/NFR IDs referenced; Given/When/Then AC |
| FE-027 | Role-gated screens have per-role tests; modal routes have soft-nav + hard-load tests |
| BE-031 / BE-032 | Logging and Sentry rules |

## 3. Domain Model Affected

### New tables (migration required)

| Table | Description |
|-------|-------------|
| timetable_templates | Named set of time slots defining a school day shape |
| time_slots | Start/end time pair belonging to one template |
| timetable_entries | One cell in the grid: class + day + slot + subject + teacher |

### Existing tables referenced

- `classrooms` (migration 05) — class_id references classrooms.id
- `subjects` (migration 05) — subject_id references subjects.id
- `memberships` (migration 03) — teacher_id references memberships.id (teacher's membership in a school)
- `schools` (migration 03) — tenant scoping

### RLS

New timetable tables are enabled for RLS with policies keyed on `tenant_id` (migration 06 pattern via `current_tenant_id()`). The service sets `app.current_tenant_id` from session context at transaction start. Cross-tenant lookups return identical 404 per BE-016.

### New domain

`internal/timetable/{handler,service,store}/` owning domain per BE-003.

### Data model

TimetableTemplate:
  id, tenant_id, school_id, name, created_at, updated_at, deleted_at

TimeSlot:
  id, tenant_id, template_id, start_time, end_time, order, created_at, updated_at, deleted_at
  -- unique (template_id, order)
  -- no-overlap validation in service layer
  -- tenant_id denormalized from template for direct tenant-scoped queries and RLS

TimetableEntry:
  id, tenant_id, time_slot_id, class_id, day (1-7), subject_id, teacher_id, created_at, updated_at, deleted_at
  -- unique (class_id, time_slot_id, day)
  -- unique (teacher_id, time_slot_id, day)
  -- tenant_id denormalized for direct tenant-scoped queries and RLS; matches slot's tenant

### Day cycle

Grid columns numbered 1-7 (Monday-Sunday). Stored as numbers; labelled on the frontend. Every day uses the same slots; no per-day slot variations.

## 4. UI

Grid page:
- Rows are time slots with editable start/end inputs.
- Columns are days 1-7 rendered with day labels.
- Cells are entries, empty until assigned.
- Template combobox lists templates; class combobox lists classes.

Template combobox + [Save template] button + Class combobox at top.
[+ Add time slot] button below the grid.

Clicking a cell opens a Dialog/Sheet with Teacher select and Subject select. Class comes from the class combobox.

## 5. Business Rules

### Time slot rules

1. Slots within a template must never overlap. Gaps are allowed.
2. [+ Add time slot] creates a slot starting at the previous slot's end time with the same duration as the previous slot. Default duration 1 hour when no slots exist.
3. Editing a slot is rejected if it overlaps a neighbouring slot or if start >= end. Inline error shown.
4. If a new slot would pass 23:59, disable the add button or show a message.

### Template rules

- [Save template] enabled only when slots are modified (dirty state).
- Switching template with unsaved changes prompts save or discard.
- Saving validates no overlaps before persisting.

### Assignment rules

- A class must be selected before any cell is editable.
- Clicking a cell opens a form with Teacher select and Subject select.
- Saving creates/updates entry for (class, day, slot, teacher, subject). Clearing deletes entry.

### Protecting existing data (backend-enforced)

- A slot's start/end times cannot be changed if any entry references it. Return 409 Conflict: slot has assignments.
- A slot can be deleted only if it has no entries. Otherwise reject with 409.
- Removing all entries from a slot makes it editable and deletable again.

### Conflict checks (backend)

Same class or teacher could be scheduled under two different templates with overlapping times. Unique constraints on slot ID alone won't catch this, so service layer must check by actual time range:

- Reject an entry if the same class already has an entry on the same day whose slot time range overlaps the new slot's range (across all templates, same school).
- Reject an entry if the same teacher already has an entry on the same day whose slot time range overlaps (across all templates, same school).
- **Cross-school teacher protection:** resolve the teacher's user → all their active memberships across schools; reject an entry if that teacher already has an entry on the same day whose slot time range overlaps in *any* school. A teacher must not be double-booked across schools at the same time.
- DB unique constraints above as a backstop for exact-slot duplicates.

## 6. API Contract

### Endpoints

| Method | Path | Permission | Description |
|--------|------|------------|-------------|
| GET | /api/v1/timetable/templates | timetable:read | List templates |
| GET | /api/v1/timetable/templates/{id} | timetable:read | Template with slots |
| PUT | /api/v1/timetable/templates/{id} | timetable:write | Save slots |
| DELETE | /api/v1/timetable/templates/{id}/slots/{slotId} | timetable:write | Delete slot (409 if entries exist) |
| GET | /api/v1/timetable/entries | timetable:read | Entries for grid (?templateId=&classId=) |
| PUT | /api/v1/timetable/entries | timetable:write | Create/update entry |
| DELETE | /api/v1/timetable/entries/{id} | timetable:write | Remove entry |

### Request/Response DTOs

All JSON snake_case. Query params camelCase.

TimetableTemplateDTO:
{ id, name, created_at, updated_at }

TimeSlotDTO:
{ id, template_id, start_time, end_time, order, created_at, updated_at }

TimetableEntryDTO:
{ id, time_slot_id, class_id, day, subject_id, teacher_id, created_at, updated_at }

### Errors

- 404 NOT_FOUND for missing OR cross-tenant record
- 403 FORBIDDEN if role lacks permission
- 409 CONFLICT for slot lock, slot has assignments, or class/teacher conflict
- 422 VALIDATION_FAILED for invalid times (start >= end, overlap, past 23:59)

## 7. Workflow (Given/When/Then)

### Add time slot

Given a template with last slot 08:00-09:00
When [+ Add time slot] is clicked
Then a new slot 09:00-10:00 is created

Given a template with last slot edited to 11:00-11:30
When [+ Add time slot] is clicked
Then a new slot 11:30-12:00 is created

Given no slots exist
When [+ Add time slot] is clicked
Then a new slot with 1 hour duration is created

### Edit slot overlap

Given a slot 08:00-09:00 and neighbour 09:00-10:00
When editing first slot to 08:30-09:30
Then request is rejected with 422 (overlaps neighbour)

### Slot lock

Given a slot with existing entries
When PUT to change start/end times
Then request is rejected with 409 Conflict: slot has assignments

### Class/teacher conflict across templates

Given class has an entry Monday 08:00-09:00 under template A
When creating entry for same class Monday 08:30-09:30 under template B
Then request is rejected with 409 (time range overlaps across templates)

## 8. Implementation Plan

### Backend

1. New migration: backend/migrations/0000000000000X_timetable_domain.sql
2. internal/timetable/store/queries/timetable_templates.sql
3. internal/timetable/store/queries/time_slots.sql
4. internal/timetable/store/queries/timetable_entries.sql
5. internal/timetable/store/store.go
6. internal/timetable/service/service.go — overlap validation, conflict checks, audit
7. internal/timetable/handler/handler.go — thin handlers with authz.Require(perm)
8. internal/app/app.go — register handlers under protectedAPI group

### Frontend

9. frontend/src/features/timetable/ — feature module with grid page, API, hooks

### Generated files

- docs/api/openapi.yaml updated (backend-owned)
- backend/docs/swagger.* via make generate-swag
- frontend/src/features/timetable/api/generated/types.ts via pnpm generate:api
- backend/internal/timetable/store/gen/ via sqlc generate

## 8. Resolved decisions

| Question | Decision |
|----------|----------|
| Templates not linked to classes | User selects class manually each time |
| Day cycle | Numbers 1-7, same slots every day |
| Add slot default | 1 hour when no slots exist; previous end time + previous duration otherwise |
| Delete semantics | Soft delete via deleted_at (BE-041) |
| Cross-template conflicts | Service-layer check by actual time range, not just slot ID |
| `class_id` FK | References `classrooms.id` (confirmed) |
| `teacher_id` FK | References `memberships.id` (teacher's membership in one school; user may hold memberships in multiple schools, so cross-school double-booking is checked at service layer) |
| Tenant scoping | `tenant_id` denormalized on `time_slots` and `timetable_entries`; RLS policies keyed to `tenant_id` per BE-016; frontend never sends tenant/school ID |
| API paths | `/api/v1/timetable/templates` (kebab-case plural nouns per BE-025) |
| `order` field | User-defined arrangement: 1 (top-most) to N (last) |
| Permission name | `timetable:read` / `timetable:write` (granted per SOM-AUTHZ-001 matrix) |

## 9. Open Questions

None.

## Links

- SOM-AUTHZ-001 permission matrix
- SOM-ROLE-HEADTEACHER / SOM-ROLE-TEACHER — timetable users
- BE-020 permission-based authz
- BE-029 audit for timetable entries
- backend/migrations/00000000000005_academic_domain.sql — classrooms, subjects tables
