# Implementation Roadmap — Specification Initiation Sequence

This document defines the order in which specifications should be initiated (approved → implemented → verified). It captures dependency relationships between specs so a future session knows what to implement next.

## Status source of truth

Specification status lives in `docs/specifications/registry.md`. This document references that status but does not duplicate it.

## Initiation phases

### Phase 0 — Foundation (no domain handler dependencies)

| # | Spec | Status | Depends on | Why first |
|---|------|--------|------------|-----------|
| 0 | SOM-AUTHZ-001 | implemented (needs update) | — | Permission matrix is data for all protected domains. Needs `timetable:read/write` and `attendance:read/write` added before timetable/attendance can be implemented. |
| 1 | SOM-SEED-001 | draft | SOM-AUTHZ-001 | Reference tables (`countries`, `education_systems`, `class_levels`) are empty and RLS-blocked in migration 06. `SOM-ACADEMICS-001` cannot create subjects without them. Seed runs post-migration in `app.go`. |

### Phase 1 — Core academic data (uses existing migrations 04 & 05 — no new tables)

| # | Spec | Status | Depends on |
|---|------|--------|------------|
| 2 | SOM-STUDENTS-001 | draft | Phase 0 complete |
| 3 | SOM-ACADEMICS-001 | draft | Phase 0 seed (education_systems + class_levels) |
| 4 | SOM-ENROLLMENTS-001 | draft | SOM-STUDENTS-001, classrooms, streams, terms |

### Phase 2 — Timetable (new migration — references Phase 1 tables)

| # | Spec | Status | Depends on |
|---|------|--------|------------|
| 5 | SOM-TIMETABLE-001 | draft | Phase 1 complete, SOM-AUTHZ-001 updated with `timetable:read/write` |

### Phase 3 — Attendance (depends on timetable_entries)

| # | Spec | Status | Depends on |
|---|------|--------|------------|
| 6 | SOM-ATTENDANCE-001 | draft | SOM-TIMETABLE-001 (timetable_entries), SOM-ENROLLMENTS-001, SOM-AUTHZ-001 updated with `attendance:read/write` |

### Phase 4 — Remaining domains

| # | Spec | Status | Depends on |
|---|------|--------|------------|
| 7 | SOM-FEES-001 | draft | SOM-STUDENTS-001, SOM-ENROLLMENTS-001 |
| 8 | SOM-HEALTH-001 | draft | SOM-STUDENTS-001 |
| 9 | SOM-BEHAVIOR-001 | draft | SOM-STUDENTS-001 |

## Dependency graph (simplified)

```
SOM-AUTHZ-001 (update)
      |
      v
SOM-SEED-001
      |
      v
SOM-STUDENTS-001  SOM-ACADEMICS-001  SOM-ENROLLMENTS-001
      |             |                 |
      v             v                 v
SOM-FEES-001   SOM-TIMETABLE-001
                    |
                    v
            SOM-ATTENDANCE-001
```

## Critical dependencies

| Dependent | Requires | Reason |
|-----------|----------|--------|
| SOM-ACADEMICS-001 | SOM-SEED-001 | `subjects` FKs to `education_systems` and `class_levels` which are empty |
| SOM-TIMETABLE-001 | SOM-AUTHZ-001 update | Needs `timetable:read` / `timetable:write` in `authz.go` permission matrix |
| SOM-ATTENDANCE-001 | SOM-TIMETABLE-001 | `attendance_records.timetable_entry_id` FK to `timetable_entries.id` |
| SOM-ATTENDANCE-001 | SOM-AUTHZ-001 update | Needs `attendance:read` / `attendance:write` in `authz.go` permission matrix |

## Current "next up"

**SOM-SEED-001** is the first spec to approve and implement. It has the fewest dependencies and unblocks `SOM-ACADEMICS-001`.

## How to use this document in a future session

1. Read `docs/specifications/registry.md` for current spec statuses.
2. Find the first `draft` spec in the initiation order above whose dependencies are all `implemented`.
3. Get human approval to move it to `approved`.
4. Implement following the spec's Implementation Plan section.
5. Update the registry status to `implemented` → `verified by human` after review.
6. Return to step 2.

## Links

- `docs/specifications/registry.md` — spec statuses
- `docs/specifications/SOM-SEED-001.md` — Phase 0 reference data
- `docs/specifications/SOM-TIMETABLE-001.md` — Phase 2 timetable
- `docs/specifications/SOM-ATTENDANCE-001.md` — Phase 3 attendance
- `backend/internal/app/app.go` — TODO comments for protected domain handler registration