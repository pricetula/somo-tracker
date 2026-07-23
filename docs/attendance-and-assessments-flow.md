# Attendance & Assessments — Full Flow Analysis

> **Author:** Platform Team  
> **Date:** 2026-07-15  
> **Version:** 1.0  
> **Scope:** Backend (`backend/`) + Frontend (`frontend/`) — multi-tenant CBC school management

---

## Table of Contents

1. [Attendance — Overview](#1-attendance--overview)
2. [Attendance — Backend](#2-attendance--backend)
3. [Attendance — Database Schema](#3-attendance--database-schema)
4. [Attendance — Frontend Pages & Components](#4-attendance--frontend-pages--components)
5. [Attendance — What Each User Sees & Does](#5-attendance--what-each-user-sees--does)
6. [Attendance — Reports to Parents](#6-attendance--reports-to-parents)
7. [Attendance — Missing Features](#7-attendance--missing-features)
8. [Assessments — Overview](#8-assessments--overview)
9. [Assessments — Backend](#9-assessments--backend)
10. [Assessments — Database Schema](#10-assessments--database-schema)
11. [Assessments — Frontend Pages & Components](#11-assessments--frontend-pages--components)
12. [Assessments — What Each User Sees & Does](#12-assessments--what-each-user-sees--does)
13. [Assessments — Status Workflow](#13-assessments--status-workflow)
14. [Assessments — Reports to Parents](#14-assessments--reports-to-parents)
15. [Assessments — Missing Features](#15-assessments--missing-features)
16. [Cross-Cutting Gaps](#16-cross-cutting-gaps)

---

## 1. Attendance — Overview

Attendance is **fully implemented** on both backend and frontend. It tracks per-student, per-timetable-slot, per-date attendance with four statuses: **PRESENT**, **ABSENT**, **LATE**, **EXCUSED**.

---

## 2. Attendance — Backend

### 2.1 Domain Models (`backend/internal/attendance/domain.go`)

| Type | Purpose |
|------|---------|
| `AttendanceStatus` | Enum: `PRESENT`, `ABSENT`, `LATE`, `EXCUSED` |
| `AttendanceRecord` | Core record — one per student/slot/date |
| `RosterStudent` | Student on a class roster with optional existing mark |
| `SlotRosterResponse` | Full roster for one timetable slot on one date |
| `BulkAttendanceEntry` | Single entry in a bulk submission |
| `BulkAttendancePayload` | Request body for bulk marking |
| `StudentAttendanceRecord` | Lightweight record (date, subject, status) for parent view |
| `ChildAttendanceSummary` | Parent summary with percentage + recent periods |
| `AttendanceTermSummary` | Materialised rollup per student/term/learning-area |
| `CompletionStatus` | Marking progress for one class/day |
| `DashboardFilter` | Filter params for admin dashboard |
| `AdminDashboardResponse` | Paginated admin dashboard |
| `StudentHistoryFilter` | Filter params for student history |
| `UpdateAttendanceEntryPayload` | Admin correction payload |

### 2.2 Sentinel Errors

- `ErrNotFound`, `ErrAlreadyExists`, `ErrInvalidInput`
- `ErrUnauthorized`, `ErrForbidden`, `ErrConflict`
- `ErrSlotAlreadyMarked` — module-specific

### 2.3 Repository (`backend/internal/attendance/repository.go`)

**Interface:**
- `GetRosterForSlot` — Class roster + existing marks for a slot/date
- `BulkUpsert` — Inserts/updates all records for a slot/date in a tx
- `GetStudentHistory` — Filtered by term/date range
- `UpdateRecord` — Single record update (admin)
- `GetAdminDashboard` — School-wide completion with dynamic filters
- `GetChildAttendanceSummary` — Parent summary with fallback on-the-fly computation
- `ComputeTermSummaries` — Full school recompute (INSERT ... ON CONFLICT)
- `ComputeClassSummaries` — Single class recompute
- `GetRecordsBySlotDate`, `GetRecordByID` — Lookup helpers

**PostgreSQL implementation (`pgRepository`):**
- Uses `pgxpool.Pool`
- BulkUpsert uses `ON CONFLICT (student_id, timetable_slot_id, date) DO UPDATE`
- Admin dashboard uses dynamic WHERE + HAVING for filters
- Child summary tries materialised table first, falls back to live computation
- Summary recompute uses INSERT ... ON CONFLICT (student_id, academic_term_id, learning_area_id)

### 2.4 Service (`backend/internal/attendance/service.go`)

- Input validation on all public methods
- `BulkMarkAttendance` — Validates, calls BulkUpsert, then enqueues async summary recompute
- `UpdateAttendanceRecord` — Same-day edit window enforcement
- `enqueueClassRecompute` — Redis SET NX dedup, Asynq enqueue
- `GetAdminDashboard` — Defaults date to today, pagination

### 2.5 Handler (`backend/internal/attendance/handler.go`)

**Routes:**

| Method | Path | Handler | Auth | Purpose |
|--------|------|---------|------|---------|
| GET | `/api/v1/attendance/roster/:timetable_slot_id` | `GetRoster` | Authenticated | Roster for marking |
| POST | `/api/v1/attendance/bulk` | `BulkMark` | Authenticated | Save attendance |
| GET | `/api/v1/attendance/dashboard` | `AdminDashboard` | Authenticated | School-wide view |
| GET | `/api/v1/attendance/students/:student_id` | `StudentHistory` | Authenticated | Student history |
| PUT | `/api/v1/attendance/records/:id` | `UpdateRecord` | Authenticated | Admin correction |
| GET | `/api/v1/attendance/children/:student_id/summary` | `ChildSummary` | Authenticated | Parent view |
| POST | `/api/v1/attendance/summaries/compute` | `ComputeSummaries` | Authenticated | Trigger recompute |

### 2.6 Worker (`backend/internal/attendance/worker.go`)

- **Task Type:** `attendance:recompute_class_summaries`
- **Queue:** `"attendance"` with priority 10
- **Concurrency:** 1 (avoids DB overload)
- **Dedup:** Redis SET NX key `attendance:pending:{termID}:{classID}` with 5min TTL
- **Flow:** Bulk mark → enqueue task → worker picks up → clears Redis flag → runs `ComputeClassSummaries`

### 2.7 Module (`backend/internal/attendance/module.go`)

```go
fx.Provide(
    fx.Annotate(NewRepository, fx.As(new(Repository))),
    NewService,
    NewHandler,
    NewWorker,
)
```

- `RegisterWorkerHooks` must be invoked from `main.go` for the Asynq server lifecycle

---

## 3. Attendance — Database Schema

### Table: `attendance_records`

| Column | Type | Constraints |
|--------|------|-------------|
| `id` | UUID | PK, DEFAULT gen_random_uuid() |
| `tenant_id` | UUID | NOT NULL |
| `school_id` | UUID | NOT NULL |
| `student_id` | UUID | NOT NULL, FK → cbc_students |
| `timetable_slot_id` | UUID | NOT NULL, FK → cbc_timetable_slots |
| `academic_term_id` | UUID | NOT NULL, FK → academic_terms |
| `date` | DATE | NOT NULL |
| `status` | attendance_status (enum) | NOT NULL |
| `marked_by` | UUID | NOT NULL, FK → users |
| `marked_at` | TIMESTAMPTZ | NOT NULL, DEFAULT NOW() |
| `note` | TEXT | NULL |
| `created_at` | TIMESTAMPTZ | NOT NULL, DEFAULT NOW() |
| `updated_at` | TIMESTAMPTZ | NOT NULL, trigger-updated |

**Unique Constraint:** `(student_id, timetable_slot_id, date)`  
**Trigger:** `trg_attendance_check_non_break_slot` — prevents marking break periods  
**RLS:** Enabled (multi-tenant)

### Table: `attendance_term_summaries` (materialised rollup)

| Column | Type | Constraints |
|--------|------|-------------|
| `id` | UUID | PK |
| `tenant_id` | UUID | NOT NULL |
| `school_id` | UUID | NOT NULL |
| `student_id` | UUID | NOT NULL |
| `academic_term_id` | UUID | NOT NULL |
| `learning_area_id` | UUID | NOT NULL |
| `periods_total` | INT | NOT NULL |
| `periods_present/absent/late/excused` | INT | NOT NULL |
| `attendance_percentage` | NUMERIC(5,2) | NOT NULL |
| `last_refreshed_at` | TIMESTAMPTZ | NOT NULL |

**Unique Constraint:** `(student_id, academic_term_id, learning_area_id)`  
**Note:** Not authoritative — `attendance_records` is source of truth

---

## 4. Attendance — Frontend Pages & Components

### 4.1 API Client (`frontend/src/lib/api/attendance.ts`)

**Functions:**
- `getSlotRoster(slotId, date?)` → `SlotRosterResponse`
- `bulkMarkAttendance(payload)` → `{ message, count }`
- `listAdminAttendances(params)` → `AdminDashboardResponse` (with filters)
- `getAdminDashboard(date?)` → `AdminDashboardResponse`
- `getStudentHistory(studentId, filters?)` → `{ items, total }`
- `updateAttendanceRecord(recordId, payload)` → `{ message }`
- `getChildAttendanceSummary(studentId, termId)` → `ChildAttendanceSummary`
- `computeAttendanceSummaries(termId)` → `{ message, count }`

### 4.2 Hooks (`frontend/src/features/attendance/hooks/use-attendance.ts`)

| Hook | Query/Mutation | Purpose |
|------|---------------|---------|
| `useSlotRoster` | Query | Fetch roster |
| `useBulkMarkAttendance` | Mutation | Save marks + toast |
| `useAdminDashboard` | Query | School-wide view |
| `useStudentHistory` | Query | Student records |
| `useUpdateAttendanceRecord` | Mutation | Admin correction |
| `useChildAttendanceSummary` | Query | Parent summary |
| `useComputeAttendanceSummaries` | Mutation | Trigger recompute |

### 4.3 Components

| Component | Role(s) | Purpose |
|-----------|---------|---------|
| `TeacherAttendanceRoster` | Teacher/Admin | Radio group status selection + notes + behaviour flag + batch submit |
| `TeacherAttendanceDashboard` | Teacher | Shows today's slots from timetable |
| `TeacherHistoryView` | Teacher | Past periods with marking status |
| `AdminAttendanceDashboard` | Admin | DataTable with multi-filters (edu level, grade, class, status) |
| `AttendanceRegisterContainer` | All | Role-aware wrapper for roster |
| `ParentAttendanceView` | Parent | Multi-child selector + term-aware display |
| `ParentAttendanceSummary` | Parent | Percentage card + recent periods table |
| `StudentHistoryView` | Admin/Teacher | Raw records table with inline edit |
| `StudentHistoryContainer` | Admin | Params reader wrapper |
| `AttendanceEmptyState` | All | Reusable empty/error state |

### 4.4 Pages

| Route | Role | Purpose |
|-------|------|---------|
| `/attendance` | All | Role-dispatched landing page |
| `/attendance/register/[slotId]` | Teacher/Admin | Mark attendance for a specific slot |
| `/attendance/history` | Teacher | Past periods view |
| `/attendance/students/[student_id]` | Admin | Student attendance history |

### 4.5 Modals

| Route | Purpose |
|-------|---------|
| `@modal/(.)attendance` | Modal attendance view |
| `@modal/(.)attendance/register` | Modal register |
| `@modal/(.)attendance/register/[slotId]` | Modal register for slot |

---

## 5. Attendance — What Each User Sees & Does

### Teacher

**Landing page** (`/attendance`):
- `TeacherAttendanceDashboard` — table of today's timetable slots
- Each row: Period name, time, class name, subject, "Register" action link
- Empty state if no slots today

**Register** (`/attendance/register/[slotId]`):
- `TeacherAttendanceRoster` — full class roster
- Each student row:
  - **RadioGroup:** Present / Absent / Late / Excused (default: PRESENT)
  - **Note icon (popover):** Optional free-text note
  - **Flag icon:** Opens `CreateBehaviorNoteDialog` (behaviour module cross-link)
- **Mark All Present** button (bulk default)
- **Save Attendance** button — triggers bulk mark → success toast
- **Locked** view after same-day window closes (cannot edit)

**History** (`/attendance/history`):
- `TeacherHistoryView` — table of past periods this week
- Same-day periods: "Mark now" badge + link to register
- Older periods: "Completed" badge, read-only

### School Admin

**Landing page** (`/attendance`):
- `AdminAttendanceDashboard` — DataTable with filter bar:
  - **Education Level** filter (multi-select): Early Years, Upper Primary, Junior Secondary, Senior School
  - **Grade** filter (multi-select): PP1–G12
  - **Class** filter (single-select): From enrolled classes
  - **Status** filter (single-select): Complete / Incomplete
- Table columns:
  - Period name
  - Class name (linked to class detail)
  - Learning area (linked to curriculum)
  - Status badge: Complete (green) / Incomplete (amber)
  - Register action (pencil icon → register page)
- Pagination (50/page)
- Empty state if no records

**Register** (`/attendance/register/[slotId]`):
- Admin sees `TeacherAttendanceRoster` with `isLocked=false` (same-day restriction removed for admins)
- Can edit any day's attendance

**Student History** (`/attendance/students/[student_id]`):
- `StudentHistoryView` — raw records table with:
  - Term dropdown (all terms / specific)
  - Record count
  - Columns: Date, Status (badge), Note, Edit (pencil icon)
  - **Inline edit:** Click pencil → status dropdown → Save button
  - Same-day edit window enforced server-side

### Parent

**Landing page** (`/attendance`):
- `ParentAttendanceView` — multi-child selector (if >1 child) or direct view
- Each child shows `ParentAttendanceSummary`:
  - Large percentage card (e.g. "92.5% attendance")
  - "Recent Periods" table (last 30 days):
    - Date, Subject, Status badge
  - Empty state: "No attendance data yet"

**Prerequisites:**
- Parent profile must have linked students (from `GET /api/v1/parents/me`)
- An active academic term must exist (`is_current = true`)

### System Admin

Same as School Admin — elevated scope (no same-day restriction for attendance).

---

## 6. Attendance — Reports to Parents

### Current Implementation

1. **Real-time summary** via `ParentAttendanceSummary` component:
   - Aggregated percentage from `attendance_term_summaries` (materialised) or computed on-the-fly
   - Last 30 days of period-level records

2. **Term Reports** (`term_reports` table):
   - `attendance_snapshot` JSONB column: `{ attendance_percentage, periods_total, periods_present, periods_absent, periods_late, periods_excused }`
   - Frozen at generation time — does NOT change after report is compiled
   - **Status:** Schema exists, but no backend service or frontend UI for generating term reports yet

### Report Flow

1. Admin navigates to `/reports/terms/[term_id]` or `/reports/student/[id]`
2. Term report is generated (snapshot taken from `attendance_term_summaries` + `behavior_snapshot` + `competency_snapshot`)
3. Status: DRAFT → PUBLISHED
4. Parent sees compiled report via `/reports/terms/[term_id]`

**Status:** ✅ Schema ready, ❌ No generation backend, ❌ No parent-facing compiled report UI

---

## 7. Attendance — Missing Features

### 7.1 Existing Data But No UI

| Feature | Status | Details |
|---------|--------|---------|
| Student history inline edit | ✅ Implemented | Save button triggers mutation |
| Same-day edit window | ✅ Implemented | Server-side enforcement in `UpdateAttendanceRecord` |
| Compute summaries endpoint | ✅ Implemented | Manual trigger via `/summaries/compute` |
| Background summary recompute | ✅ Implemented | Asynq worker with Redis dedup |

### 7.2 Not Yet Implemented

#### Unimplemented Features — Attendance

| # | Feature | Backend | Frontend | Priority | User Story | Details |
|---|---------|---------|----------|----------|------------|---------|
| A1 | Term report generation | ❌ | ❌ | **High** | As an admin, I want to generate a compiled term report per student so that parents receive an official end-of-term summary. | `term_reports` table exists with `attendance_snapshot`, `behavior_snapshot`, `competency_snapshot` JSONB columns — no backend service to populate them. |
| A2 | Term report publishing workflow | ❌ | ❌ | **High** | As an admin, I want to publish compiled term reports so that they become visible to parents. | DRAFT→PUBLISHED status flow for `term_reports`. No endpoints exist. |
| A3 | Parent-facing compiled report | ❌ | ❌ | **High** | As a parent, I want to view my child's compiled term report online so that I can review their progress. | `/reports/term/[term_id]` page exists but is a placeholder. No component shows compiled attendance+behaviour+competency. |
| A4 | Bulk report export | ❌ | ❌ | **Medium** | As an admin, I want to export term reports for an entire class or grade level as a batch so that I can distribute them offline. | `/reports/bulk-export` page is placeholder. |
| A5 | CSV/PDF export of attendance | ❌ | ❌ | **Medium** | As a teacher or admin, I want to download attendance records as CSV or PDF for offline record-keeping. | No download/print functionality on any attendance page. |
| A6 | Absence notifications (SMS/email) | ❌ | ❌ | **Medium** | As a parent, I want to receive an alert when my child is marked absent or late so that I can take action. | No notification system integrated (no SMS gateway, no email templates). |
| A7 | Daily attendance roll call summary | ❌ | ❌ | **Low** | As an admin, I want to see a daily summary of attendance across all classes (total present/absent/late/excused counts). | No aggregate daily report. Dashboard only shows completion status, not counts. |
| A8 | Attendance trend charts | ❌ | ❌ | **Low** | As a teacher or parent, I want to see attendance trends over time (weekly, monthly, term) as a chart. | No charts library integration on attendance views. |
| A9 | Guardian / pickup tracking | ❌ | ❌ | **Low** | As a teacher, I want to record who picked up a student (guardian name, relationship) for safety compliance. | No check-in/check-out system. `note` field is free-text but no structured pickup data. |
| A10 | Behaviour cross-link in reports | ❌ | ❌ | **Medium** | As an admin, I want behaviour notes to be included in term reports so that parents have a holistic view. | `behavior_snapshot` column exists in `term_reports` but no service populates it from `behavior_notes`. |

#### Unimplemented Pages & Routes — Attendance

| Route | Purpose | Status | Backend Endpoint | Required Work |
|-------|---------|--------|-----------------|---------------|
| `/reports/student/[id]` | Single student compiled report | ❌ Placeholder | ✅ `GET /api/v1/attendance/students/:student_id` | Build report generation + snapshot compilation + UI |
| `/reports/terms/[term_id]` | Term report detail (admin + parent) | ❌ Placeholder | ✅ `GET /api/v1/attendance/children/:student_id/summary` | Build service to populate `term_reports`, then role-dispatched UI |
| `/reports/bulk-export` | Bulk term report export | ❌ Placeholder | ✅ None (need new) | Build backend batch generation endpoint + frontend class/grade selector |

---

## 8. Assessments — Overview

Assessment is **fully implemented on the backend** but the **frontend is entirely missing** ("Coming Soon" pages).

The system manages **CBC assessment blueprints**, **assessment sessions**, and **learner rubric results** with a full status workflow (DRAFT → PENDING_REVIEW → APPROVED → PUBLISHED) plus KNEC weight configs and grading scales.

---

## 9. Assessments — Backend

### 9.1 Domain Models (`backend/internal/assessment/domain.go`)

**Core Types:**

| Type | Purpose |
|------|---------|
| `AssessmentBlueprint` | Per-school assessment plan (title, type, grade, year, term) |
| `BlueprintDetail` | Blueprint + linked performance indicators |
| `LinkedIndicator` | Performance indicator linked to a blueprint |
| `AssessmentSession` | Concrete assessment instance (class, date, status) |
| `SessionDetail` | Session + nested rubric results |
| `SessionAdminView` | Session with blueprint/teacher/class info for admin queue |
| `LearnerRubricResult` | Single student's rubric outcome per indicator |
| `AssessmentWeightConfig` | KNEC-mandated national grading weight |
| `ParentSessionResultView` | Parent-facing view (no raw scores, no averages) |

**Status Machine Enums:**
- `cbc_assessment_type`: `Formative_Classroom`, `KNEC_Written_Assessment`, `KNEC_SBA_Project`, `National_KPSEA`, `National_KJSEA`, `National_KSSEA`
- `cbc_session_status`: `DRAFT`, `PENDING_REVIEW`, `APPROVED`, `PUBLISHED`, `REJECTED`
- `cbc_rubric_level`: `EE`, `ME`, `AE`, `BE`
- `lrr_score_type`: `Numeric_Raw`, `Rubric_Direct`

**Request/Response Types (~25):**
- `CreateBlueprintPayload`, `UpdateBlueprintPayload`, `CreateBlueprintResponse`
- `ListBlueprintsQuery`, `ListBlueprintsResponse`, `BlueprintDetailResponse`
- `LinkIndicatorPayload`, `CreateSessionPayload`, `UpdateSessionPayload`
- `ListSessionsQuery`, `ListSessionsResponse`, `SessionDetailResponse`
- `BatchUpsertResultInput`, `BatchUpsertResultsPayload`, `ListResultsResponse`
- `SubmitForReviewPayload`, `ApproveSessionPayload`, `RejectSessionPayload`
- `PublishSessionPayload`, `BatchPublishPayload`, `BatchPublishResponse`
- `StatusTransitionResponse`, `AdminQueuesQuery`, `AdminQueuesResponse`
- `ParentSessionsQuery`, `ParentPublishedResultsResponse`

**Cross-Domain Interfaces:**
- `LearningAreaResolver` — resolves indicator's education level for grade validation
- `ClassStudentResolver` — checks student membership in a class
- `BlueprintIndicatorResolver` — checks indicator is linked to blueprint

**Module-specific sentinel errors:**
- `ErrGradeLevelMismatch`, `ErrIndicatorLinked`, `ErrInvalidRubricLevel`
- `ErrScoreTypeMismatch`, `ErrStudentNotInClass`, `ErrIndicatorNotInBP`
- `ErrSessionLocked`, `ErrInvalidStatusTransition`, `ErrNoResultsToSubmit`
- `ErrRejectionReasonRequired`, `ErrNotSessionOwner`, `ErrNotAdmin`, `ErrBatchPublishEmpty`

### 9.2 Repository (`backend/internal/assessment/repository.go`)

**Interface methods:**
- Blueprints: `CreateBlueprint`, `GetBlueprintByID`, `ListBlueprints`, `UpdateBlueprint`, `DeleteBlueprint`
- Detail: `GetBlueprintDetail`
- Indicator Linking: `LinkIndicators`, `UnlinkIndicator`, `IsIndicatorLinked`, `ListBlueprintIndicators`
- Sessions: `CreateSession`, `GetSessionByID`, `ListSessions`, `UpdateSession`, `DeleteSession`
- Detail: `GetSessionDetail`
- Status Transitions: `SubmitForReview`, `ApproveSession`, `RejectSession`, `PublishSession`, `PublishSessions`
- Results: `BatchUpsertResults`, `ListResults`
- Weight Configs: `ListWeightConfigs`
- Admin Queues: `ListSessionsForAdminQueue`
- Parent-Facing: `ListPublishedResultsForStudent`

**Cross-Resolver implementations on `PgRepository`:**
- `IsStudentInClass(ctx, studentID, classID)` — EXISTS query on `cbc_student_enrollments`

### 9.3 Service (`backend/internal/assessment/service.go`)

- Full input validation on all methods
- Grade level → education level mapping (`gradeLevelToEducationLevel`)
- `CreateBlueprint`: Validates title, term (1-3), year (≥2017), type, grade
- `LinkIndicators`: Validates each indicator isn't already linked AND matches grade level via `LearningAreaResolver`
- `CreateSession`: Validates date format (YYYY-MM-DD)
- `BatchUpsertResults`: Only allowed on DRAFT/REJECTED sessions; validates rubric level, score type, indicator membership, student membership
- `SubmitForReview`: Checks status transition allowed, only session owner, must have ≥1 result
- `ApproveSession`/`RejectSession`: Admin-only, checks transition validity
- `PublishSession`/`PublishSessions`: Admin-only, APPROVED→PUBLISHED only
- `ListSessionsForAdminQueue`: Only PENDING_REVIEW or APPROVED status
- `ListPublishedResultsForStudent`: Hard-filters PUBLISHED server-side

**Status Transition Rules:**

```
DRAFT ──────────→ PENDING_REVIEW ←────────── REJECTED
                     │                              ↑
                     │                              │
                     ↓                              │
                  APPROVED                          │
                     │                              │
                     ↓                              │
                  PUBLISHED (terminal) ──────────────┘
```

### 9.4 Handler (`backend/internal/assessment/handler.go`)

**Blueprint Routes:**

| Method | Path | Handler | Auth | Purpose |
|--------|------|---------|------|---------|
| POST | `/api/v1/assessment/blueprints` | `CreateBlueprint` | Authenticated | Create |
| GET | `/api/v1/assessment/blueprints` | `ListBlueprints` | Authenticated | List (filter: school, grade, term, year) |
| GET | `/api/v1/assessment/blueprints/:id` | `GetBlueprintDetail` | Authenticated | Detail + indicators |
| PUT | `/api/v1/assessment/blueprints/:id` | `UpdateBlueprint` | Authenticated | Update title/type |
| DELETE | `/api/v1/assessment/blueprints/:id` | `DeleteBlueprint` | Authenticated | Delete (blocked if sessions exist) |

**Indicator Linking Routes (nested under blueprints):**

| Method | Path | Handler | Auth | Purpose |
|--------|------|---------|------|---------|
| POST | `/api/v1/assessment/blueprints/:id/indicators` | `LinkIndicators` | Authenticated | Link indicators |
| DELETE | `/api/v1/assessment/blueprints/:id/indicators/:indicator_id` | `UnlinkIndicator` | Authenticated | Unlink one |

**Session Routes:**

| Method | Path | Handler | Auth | Purpose |
|--------|------|---------|------|---------|
| POST | `/api/v1/assessment/sessions` | `CreateSession` | Authenticated | Create from blueprint + class |
| GET | `/api/v1/assessment/sessions` | `ListSessions` | Authenticated | List (filter: class, blueprint, status) |
| GET | `/api/v1/assessment/sessions/:id` | `GetSessionDetail` | Authenticated | Detail + results |
| PUT | `/api/v1/assessment/sessions/:id` | `UpdateSession` | Authenticated | Update date/KNEC reference |
| DELETE | `/api/v1/assessment/sessions/:id` | `DeleteSession` | Authenticated | Delete + cascade results |
| POST | `/api/v1/assessment/sessions/:id/results/batch` | `BatchUpsertResults` | Authenticated | Batch upsert rubric results |
| GET | `/api/v1/assessment/sessions/:id/results` | `ListResults` | Authenticated | List results |

**Status Transition Routes:**

| Method | Path | Handler | Auth | Purpose |
|--------|------|---------|------|---------|
| POST | `/api/v1/assessment/sessions/:id/submit` | `SubmitForReview` | Authenticated | Teacher → PENDING_REVIEW |
| POST | `/api/v1/assessment/sessions/:id/approve` | `ApproveSession` | SchoolAdmin | Admin → APPROVED |
| POST | `/api/v1/assessment/sessions/:id/reject` | `RejectSession` | SchoolAdmin | Admin → REJECTED (reason required) |
| POST | `/api/v1/assessment/sessions/:id/publish` | `PublishSession` | SchoolAdmin | Admin → PUBLISHED |
| POST | `/api/v1/assessment/sessions/batch-publish` | `BatchPublishSessions` | SchoolAdmin | Batch PUBLISH |

**Admin Queue Routes:**

| Method | Path | Handler | Auth | Purpose |
|--------|------|---------|------|---------|
| GET | `/api/v1/assessment/admin/sessions` | `ListAdminQueue` | SchoolAdmin | PENDING_REVIEW or APPROVED queue |

**Parent-Facing Routes:**

| Method | Path | Handler | Auth | Purpose |
|--------|------|---------|------|---------|
| GET | `/api/v1/parent/students/:student_id/results` | `ListPublishedResultsForStudent` | Parent | Published results only |

**Weight Config Routes:**

| Method | Path | Handler | Auth | Purpose |
|--------|------|---------|------|---------|
| GET | `/api/v1/assessment/weight-configs` | `ListWeightConfigs` | Authenticated | Read-only KNEC weights |

### 9.5 Module (`backend/internal/assessment/handler.go` — bottom)

```go
var Module = fx.Module("assessment",
    fx.Provide(
        fx.Annotate(
            NewRepository,
            fx.As(new(Repository)),
            fx.As(new(ClassStudentResolver)),
        ),
        NewService,
        NewHandler,
    ),
)
```

Note: The repository is registered as both `Repository` and `ClassStudentResolver` in a single `fx.Annotate` call.

---

## 10. Assessments — Database Schema

### Table: `assessment_blueprints`

| Column | Type | Constraints |
|--------|------|-------------|
| `id` | UUID | PK |
| `tenant_id` | UUID | NOT NULL |
| `school_id` | UUID | NOT NULL |
| `title` | VARCHAR(255) | NOT NULL |
| `type` | cbc_assessment_type | NOT NULL |
| `grade_level` | cbc_grade_level | NOT NULL |
| `academic_year` | SMALLINT | NOT NULL, ≥2017 |
| `term` | SMALLINT | NOT NULL, 1-3 |

**Unique:** `(school_id, type, grade_level, academic_year, term)`  
**RLS:** Enabled

### Table: `assessment_blueprint_indicators`

| Column | Type | Constraints |
|--------|------|-------------|
| `blueprint_id` | UUID | PK, FK → assessment_blueprints ON DELETE CASCADE |
| `indicator_id` | UUID | PK, FK → performance_indicators ON DELETE CASCADE |

### Table: `assessment_sessions`

| Column | Type | Constraints |
|--------|------|-------------|
| `id` | UUID | PK |
| `tenant_id` | UUID | NOT NULL |
| `blueprint_id` | UUID | NOT NULL, FK → assessment_blueprints ON DELETE CASCADE |
| `class_id` | UUID | NOT NULL |
| `assessed_by_user_id` | UUID | NOT NULL, FK → users |
| `date_administered` | DATE | NOT NULL |
| `knec_upload_reference` | VARCHAR(50) | NULL |
| `status` | cbc_session_status | NOT NULL, DEFAULT 'DRAFT' |
| `submitted_at` | TIMESTAMPTZ | NULL |
| `reviewed_by_user_id` | UUID | NULL |
| `reviewed_at` | TIMESTAMPTZ | NULL |
| `rejection_reason` | TEXT | NULL |
| `published_by_user_id` | UUID | NULL |
| `published_at` | TIMESTAMPTZ | NULL |

**Check Constraints:**
- `chk_asessions_rejection_reason_required`: status=REJECTED → rejection_reason IS NOT NULL
- `chk_asessions_approved_has_reviewer`: status in (APPROVED, REJECTED) → reviewed_by_user_id IS NOT NULL
- `chk_asessions_published_has_publisher`: status=PUBLISHED → published_by_user_id IS NOT NULL

**Indexes:** tenant, blueprint, class, teacher, class+date, status, class+status, reviewer, publisher, published-only partial  
**RLS:** Enabled

### Table: `learner_rubric_results`

| Column | Type | Constraints |
|--------|------|-------------|
| `id` | UUID | PK |
| `tenant_id` | UUID | NOT NULL |
| `session_id` | UUID | NOT NULL, FK → assessment_sessions ON DELETE CASCADE |
| `student_id` | UUID | NOT NULL |
| `indicator_id` | UUID | NOT NULL, FK → performance_indicators |
| `score_type` | lrr_score_type | NOT NULL |
| `raw_score` | NUMERIC(5,2) | NULL, CHECK ≥0 |
| `rubric_level` | cbc_rubric_level | NOT NULL |
| `teacher_observation_notes` | TEXT | NULL |

**Unique:** `(session_id, student_id, indicator_id)`  
**RLS:** Enabled

### Table: `cbc_assessment_grading_scales`

| Column | Type | Constraints |
|--------|------|-------------|
| `id` | UUID | PK |
| `tenant_id` | UUID | NOT NULL |
| `school_id` | UUID | NOT NULL |
| `grade_level` | cbc_grade_level | NOT NULL |
| `rubric_level` | cbc_rubric_level | NOT NULL |
| `min_percentage` | NUMERIC(5,2) | NOT NULL, ≥0 |
| `max_percentage` | NUMERIC(5,2) | NOT NULL, ≤100 |

**Exclusion Constraint** (`excl_grading_scales_no_overlap`):
- GiST exclusion with `numrange(min, max, '[)')` — half-open intervals must not overlap per tenant/school/grade

### Table: `assessment_weight_configs`

| Column | Type | Constraints |
|--------|------|-------------|
| `id` | UUID | PK |
| `grade_level` | cbc_grade_level | NOT NULL |
| `assessment_type_code` | cbc_assessment_type | NOT NULL |
| `target_exam` | knec_target_exam | NOT NULL |
| `weight_percent` | NUMERIC(5,2) | NOT NULL, 0-100 |
| `effective_from` | SMALLINT | NOT NULL, ≥2017 |

**Unique:** `(grade_level, assessment_type_code, target_exam, effective_from)`  
**Note:** Global table (no tenant_id) — KNEC weights are nationally mandated

### Table: `term_reports`

| Column | Type | Constraints |
|--------|------|-------------|
| `id` | UUID | PK |
| `tenant_id` | UUID | NOT NULL |
| `school_id` | UUID | NOT NULL |
| `student_id` | UUID | NOT NULL |
| `academic_term_id` | UUID | NOT NULL |
| `attendance_snapshot` | JSONB | NOT NULL, DEFAULT '{}' |
| `behavior_snapshot` | JSONB | NOT NULL, DEFAULT '[]' |
| `competency_snapshot` | JSONB | NOT NULL, DEFAULT '[]' |
| `status` | term_report_status | NOT NULL, DEFAULT 'DRAFT' |
| `generated_at` | TIMESTAMPTZ | NOT NULL, DEFAULT NOW() |
| `published_at` | TIMESTAMPTZ | NULL |
| `generated_by` | UUID | NOT NULL |

**Unique:** `(student_id, academic_term_id)`  
**RLS:** Enabled

---

## 11. Assessments — Frontend Pages & Components

### 11.1 Pages (All "Coming Soon" placeholders)

| Route | Status | File |
|-------|--------|------|
| `/assessments` | ❌ Coming Soon | `frontend/src/app/(dashboard)/assessments/page.tsx` |
| `/assessments/[id]` | ❌ Coming Soon | `frontend/src/app/(dashboard)/assessments/[id]/page.tsx` |
| `/assessments/add` | ❌ Coming Soon | `frontend/src/app/(dashboard)/assessments/add/page.tsx` |

### 11.2 Modals (Directories exist, likely empty)

| Route | Status |
|-------|--------|
| `@modal/(.)assessments` | ❌ Empty |
| `@modal/(.)assessments/add` | ❌ Empty |

### 11.3 Feature Directory (All empty)

| Path | Status |
|------|--------|
| `frontend/src/features/assessments/` | ❌ Empty |
| `frontend/src/features/assessments/components/` | ❌ Empty |
| `frontend/src/features/assessments/hooks/` | ❌ Empty |
| `frontend/src/features/assessments/types/` | ❌ Empty |

### 11.4 API Client

**No `assessments.ts` file exists** in `frontend/src/lib/api/`. There is no frontend API client for any assessment endpoint.

---

## 12. Assessments — What Each User Sees & Does

> ⚠️ **All frontend views are not implemented.** What follows is the *intended* user flow based on backend capabilities.

### Teacher

**Blueprint Management:**
- View a list of assessment blueprints filtered by grade/term/year
- Create new blueprints (title, type, grade, year, term)
- View blueprint detail including linked indicators
- Link/unlink performance indicators to a blueprint

**Session Management:**
- Create assessment sessions from blueprints (select class, enter date)
- View list of sessions (filter by class, blueprint, status)
- View session detail with rubric results
- Enter/edit rubric results per student per indicator (batch upsert)
- Submit session for admin review (DRAFT → PENDING_REVIEW)

### School Admin

**Review Queue:**
- View `PENDING_REVIEW` sessions queue
- Approve or reject sessions (requires rejection reason for rejections)
- View `APPROVED` sessions queue ready for publishing

**Publishing:**
- Publish individual sessions (APPROVED → PUBLISHED)
- Batch publish multiple sessions at once

**Blueprint Management:**
- Same as teacher (create, edit, delete blueprints)

**Grading Scales:**
- Configure per-school grading scale brackets (EE/ME/AE/BE)

### Parent

**Viewing Published Results:**
- See published assessment results per child
- Data includes: date administered, learning area, blueprint title/type, rubric level (EE/ME/AE/BE), indicator description, teacher observation notes
- NO raw scores visible (CBC compliance)
- NO computed averages visible (CBC compliance)
- Filterable by academic year and term

### System Admin

Same as School Admin (no additional assessment privileges).

---

## 13. Assessments — Status Workflow

```
Teacher flow:
  DRAFT ──(submit for review)──→ PENDING_REVIEW
                                    │
Admin flow:                         │
                          ┌─────────┴──────────┐
                          │                    │
                    (approve)             (reject)
                          │                    │
                          ↓                    ↓
                      APPROVED            REJECTED ──(resubmit)──→ PENDING_REVIEW
                          │
                    (publish)
                          │
                          ↓
                     PUBLISHED (terminal)
```

**Validation rules at each step:**

| Transition | From | To | Requirements |
|-----------|------|----|-------------|
| Submit for review | DRAFT, REJECTED | PENDING_REVIEW | Must be session owner; session must have ≥1 result |
| Approve | PENDING_REVIEW | APPROVED | Admin role; reviewer user_id recorded |
| Reject | PENDING_REVIEW | REJECTED | Admin role; rejection_reason required |
| Publish | APPROVED | PUBLISHED | Admin role; publisher user_id recorded |
| Batch publish | APPROVED | PUBLISHED | Admin role; skips non-APPROVED sessions |

**Editability:** Results can only be upserted when session is DRAFT or REJECTED. Once PENDING_REVIEW or beyond, results are locked.

---

## 14. Assessments — Reports to Parents

### Current Implementation

**Backend:** ✅ `ListPublishedResultsForStudent` endpoint exists  
**Frontend:** ❌ No UI to display published results to parents

### How It Should Work

1. Teacher creates blueprint → links indicators → creates session → enters results → submits for review
2. Admin reviews → approves → publishes
3. Parent sees published results via `/assessments` or `/reports/student/[id]`
4. Each result shows: date administered, learning area, blueprint title/type, rubric level (EE/ME/AE/BE), indicator description, teacher observation notes

### CBC Compliance Rules

- Raw scores (`raw_score`) are NEVER exposed to parents
- Computed averages across indicators are NEVER computed or shown (CBC violation)
- Only rubric levels (EE/ME/AE/BE) are shown

### Term Reports Integration

- `term_reports.competency_snapshot` is the JSONB column for assessment summaries
- Should contain: `{ learning_area_name, calculated_level, final_level, teacher_narrative_summary }`
- Not yet populated by any backend service

---

## 15. Assessments — Missing Features

### 15.1 Unimplemented Features — Assessment Blueprints & Sessions

| # | Feature | Backend | Frontend | Priority | User Story | Details |
|---|---------|---------|----------|----------|------------|---------|
| B1 | Assessment API client | ✅ Complete | ❌ **Missing** | **Critical** | As a developer, I want a typed API client for all assessment endpoints so that frontend components can call them. | No `frontend/src/lib/api/assessments.ts`. 23 endpoints exist in backend with zero frontend consumption. |
| B2 | Blueprint list page | ✅ Complete | ❌ **Missing** | **Critical** | As a teacher/admin, I want to see all assessment blueprints for my school filtered by grade, term, and year so that I can manage them. | `GET /api/v1/assessment/blueprints` ready. Page `/assessments` shows "Coming Soon". Needs DataTable + filter bar. |
| B3 | Blueprint create form | ✅ Complete | ❌ **Missing** | **Critical** | As a teacher/admin, I want to create a new assessment blueprint with title, type, grade, year, and term so that I can plan assessments. | `POST /api/v1/assessment/blueprints` ready. `/assessments/add` shows "Coming Soon". Needs form with validation. |
| B4 | Blueprint detail page | ✅ Complete | ❌ **Missing** | **Critical** | As a teacher/admin, I want to view a blueprint's detail including its linked performance indicators so that I can verify the assessment scope. | `GET /api/v1/assessment/blueprints/:id` returns `BlueprintDetail` with indicators. `/assessments/[id]` is placeholder. |
| B5 | Blueprint edit | ✅ Complete | ❌ **Missing** | **High** | As a teacher/admin, I want to update a blueprint's title and type so that I can correct mistakes. | `PUT /api/v1/assessment/blueprints/:id` ready. No edit form UI. |
| B6 | Blueprint delete | ✅ Complete | ❌ **Missing** | **High** | As a teacher/admin, I want to delete a blueprint so that I can remove unused plans (only if no sessions exist). | `DELETE /api/v1/assessment/blueprints/:id` ready with conflict check. No delete button UI. |
| B7 | Indicator linking UI | ✅ Complete | ❌ **Missing** | **High** | As a teacher/admin, I want to search and link performance indicators to a blueprint so that the assessment scope is defined. | `POST /api/v1/assessment/blueprints/:id/indicators` ready with grade-level validation. Needs multi-select with search + curriculum tree. |
| B8 | Indicator unlink UI | ✅ Complete | ❌ **Missing** | **High** | As a teacher/admin, I want to remove an indicator from a blueprint so that I can refine the assessment scope. | `DELETE /api/v1/assessment/blueprints/:id/indicators/:indicator_id` ready. No unlink button in indicator list. |

### 15.2 Unimplemented Features — Assessment Sessions & Results

| # | Feature | Backend | Frontend | Priority | User Story | Details |
|---|---------|---------|----------|----------|------------|---------|
| B9 | Session create flow | ✅ Complete | ❌ **Missing** | **Critical** | As a teacher, I want to create an assessment session from a blueprint for a specific class on a specific date so that I can record results. | `POST /api/v1/assessment/sessions` ready. Needs step-by-step or inline form (select class + date). |
| B10 | Session list with filters | ✅ Complete | ❌ **Missing** | **Critical** | As a teacher/admin, I want to list assessment sessions filtered by class, blueprint, and status so that I can find and manage them. | `GET /api/v1/assessment/sessions` ready with pagination + filters. Needs DataTable with filter bar. |
| B11 | Session detail page | ✅ Complete | ❌ **Missing** | **Critical** | As a teacher, I want to see all rubric results for a session so that I can review entered scores. | `GET /api/v1/assessment/sessions/:id` returns `SessionDetail` with nested `LearnerRubricResult[]`. No UI. |
| B12 | Session edit (date, KNEC ref) | ✅ Complete | ❌ **Missing** | **Medium** | As a teacher, I want to update the administered date or KNEC upload reference for a session. | `PUT /api/v1/assessment/sessions/:id` ready. No edit UI. |
| B13 | Session delete | ✅ Complete | ❌ **Missing** | **Medium** | As a teacher, I want to delete a session (and cascade its results) so that I can remove erroneous entries. | `DELETE /api/v1/assessment/sessions/:id` ready with cascade. No delete button. |
| B14 | Batch result entry (matrix/grid) | ✅ Complete | ❌ **Missing** | **Critical** | As a teacher, I want to enter rubric levels for all students × all indicators at once so that I can quickly record assessment outcomes. | `POST /api/v1/assessment/sessions/:id/results/batch` ready. Needs a matrix where rows=students, cols=indicators, cells=rubric level dropdown. |
| B15 | Result list view | ✅ Complete | ❌ **Missing** | **High** | As a teacher/admin, I want to view all rubric results for a session in a structured table so that I can verify entries. | `GET /api/v1/assessment/sessions/:id/results` ready. Needs table with student, indicator, rubric level, notes columns. |

### 15.3 Unimplemented Features — Status Workflow

| # | Feature | Backend | Frontend | Priority | User Story | Details |
|---|---------|---------|----------|----------|------------|---------|
| B16 | Submit for review (DRAFT→PENDING_REVIEW) | ✅ Complete | ❌ **Missing** | **Critical** | As a teacher, I want to submit a completed session for admin review so that it can be approved. | `POST /api/v1/assessment/sessions/:id/submit` ready. Needs submit button with validation (must have ≥1 result, must be session owner). |
| B17 | Approve session (PENDING_REVIEW→APPROVED) | ✅ Complete | ❌ **Missing** | **High** | As an admin, I want to approve a submitted session so that it can proceed to publishing. | `POST /api/v1/assessment/sessions/:id/approve` ready with admin role check. Needs approve button in admin queue. |
| B18 | Reject session (PENDING_REVIEW→REJECTED) | ✅ Complete | ❌ **Missing** | **High** | As an admin, I want to reject a session with a reason so that the teacher can correct and resubmit. | `POST /api/v1/assessment/sessions/:id/reject` ready with required rejection reason. Needs reject dialog with text input. |
| B19 | Single publish (APPROVED→PUBLISHED) | ✅ Complete | ❌ **Missing** | **High** | As an admin, I want to publish a single approved session so that results become visible to parents. | `POST /api/v1/assessment/sessions/:id/publish` ready. Needs publish button. |
| B20 | Batch publish multiple sessions | ✅ Complete | ❌ **Missing** | **Medium** | As an admin, I want to select and publish multiple approved sessions at once for efficiency. | `POST /api/v1/assessment/sessions/batch-publish` ready (skips non-APPROVED). Needs multi-select + batch publish button. |
| B21 | PENDING_REVIEW admin queue | ✅ Complete | ❌ **Missing** | **Critical** | As an admin, I want to see all sessions pending my review so that I can approve or reject them. | `GET /api/v1/assessment/admin/sessions?status=PENDING_REVIEW` ready. Needs filtered DataTable with approve/reject actions. |
| B22 | APPROVED (ready-to-publish) admin queue | ✅ Complete | ❌ **Missing** | **High** | As an admin, I want to see all approved sessions ready to be published so that I can batch-publish them. | `GET /api/v1/assessment/admin/sessions?status=APPROVED` ready. Needs filtered DataTable with publish actions. |

### 15.4 Unimplemented Features — Parent-Facing & Weight Configs

| # | Feature | Backend | Frontend | Priority | User Story | Details |
|---|---------|---------|----------|----------|------------|---------|
| B23 | Published results view (parent) | ✅ Complete | ❌ **Missing** | **High** | As a parent, I want to see my child's published assessment results so that I can track their academic progress. | `GET /api/v1/parent/students/:student_id/results` ready. Needs page showing rubric levels per indicator with learning area grouping. Raw scores MUST NOT be shown (CBC compliance). |
| B24 | Weight configs viewer | ✅ Complete | ❌ **Missing** | **Low** | As a teacher/admin, I want to see the KNEC weight configuration table so that I understand assessment scoring. | `GET /api/v1/assessment/weight-configs` ready. Needs read-only table with grade, type, target exam, weight columns. |

### 15.5 Unimplemented Features — Backend Gaps

| # | Feature | Status | Priority | User Story | Details |
|---|---------|--------|----------|------------|---------|
| B25 | Grading scales CRUD endpoints | ❌ Missing | **High** | As an admin, I want to configure and manage grading scale brackets (percentage → rubric level) for my school so that raw scores are mapped correctly. | `cbc_assessment_grading_scales` table exists with exclusion constraints. No endpoints for CRUD. Frontend needs a table editor with overlap detection. |
| B26 | Competency snapshot computation | ❌ Missing | **High** | As an admin, I want term reports to contain compiled assessment competency summaries so that parents see a holistic view. | `term_reports.competency_snapshot` should contain `{learning_area_name, calculated_level, final_level, teacher_narrative_summary}`. No service populates it. |
| B27 | Score conversion (raw → rubric) service | ❌ Partial | **Medium** | As a teacher, I want raw scores to be automatically converted to rubric levels (EE/ME/AE/BE) based on the school's grading scale. | DB function `fn_convert_raw_score_to_rubric` exists but no backend service calls it. Teachers must manually pick rubric level. |
| B28 | KNEC portal API integration | ❌ Missing | **Low** | As a teacher, I want SBA project scores to be automatically uploaded to the KNEC portal (cba.knec.ac.ke). | `knec_upload_reference` is a manual text field. No API integration to the KNEC portal exists. |
| B29 | Assessment analytics / class-wide stats | ❌ Missing | **Low** | As an admin, I want to see class-wide assessment statistics (distribution of rubric levels per indicator, per learning area) for trend analysis. | No computed averages or class-wide statistics endpoints exist. |

### 15.6 Backend Endpoints Without Frontend Counterpart

| # | Endpoint | Purpose | User Action | Priority |
|---|----------|---------|-------------|----------|
| B30 | `GET /api/v1/assessment/blueprints` | List blueprints | View assessment plans | Critical |
| B31 | `POST /api/v1/assessment/blueprints` | Create blueprint | Define new assessment plan | Critical |
| B32 | `GET /api/v1/assessment/blueprints/:id` | Blueprint detail + indicators | View linked indicators | Critical |
| B33 | `PUT /api/v1/assessment/blueprints/:id` | Update blueprint | Edit title/type | High |
| B34 | `DELETE /api/v1/assessment/blueprints/:id` | Delete blueprint | Remove unused plan | High |
| B35 | `POST /api/v1/assessment/blueprints/:id/indicators` | Link indicators | Define assessment scope | High |
| B36 | `DELETE /api/v1/assessment/blueprints/:id/indicators/:indicator_id` | Unlink indicator | Refine scope | High |
| B37 | `POST /api/v1/assessment/sessions` | Create session | Administer assessment | Critical |
| B38 | `GET /api/v1/assessment/sessions` | List sessions | Find & manage sessions | Critical |
| B39 | `GET /api/v1/assessment/sessions/:id` | Session detail + results | Review entered scores | Critical |
| B40 | `PUT /api/v1/assessment/sessions/:id` | Update session | Edit date/KNEC ref | Medium |
| B41 | `DELETE /api/v1/assessment/sessions/:id` | Delete session | Remove erroneous entry | Medium |
| B42 | `POST /api/v1/assessment/sessions/:id/submit` | Submit for review | Submit completed session | Critical |
| B43 | `POST /api/v1/assessment/sessions/:id/approve` | Approve session | Approve pending session | High |
| B44 | `POST /api/v1/assessment/sessions/:id/reject` | Reject session | Reject session with reason | High |
| B45 | `POST /api/v1/assessment/sessions/:id/publish` | Publish session | Publish approved session | High |
| B46 | `POST /api/v1/assessment/sessions/batch-publish` | Batch publish | Publish multiple sessions | Medium |
| B47 | `GET /api/v1/assessment/admin/sessions` | Admin queue | Review/publish sessions | Critical |
| B48 | `GET /api/v1/parent/students/:student_id/results` | Parent published results | View child's results | High |
| B49 | `GET /api/v1/assessment/weight-configs` | Weight configs | View KNEC weights | Low |
| B50 | `POST /api/v1/assessment/sessions/:id/results/batch` | Batch upsert results | Enter rubric scores | Critical |
| B51 | `GET /api/v1/assessment/sessions/:id/results` | List results | View entered results | High |

**Total: 23 endpoints with no frontend (numbered B30–B51, plus B25–B29 missing backend).**

### 15.7 Unimplemented Pages & Routes — Assessments

| Route | Purpose | Status | Backend Endpoint(s) | Required Work |
|-------|---------|--------|---------------------|---------------|
| `/assessments` | Blueprint list + session list (role-dispatched) | ❌ Placeholder | B30, B38, B47 | API client + role-dispatched landing page with filterable DataTables |
| `/assessments/add` | Create blueprint form | ❌ Placeholder | B31 | Form with validation (title, type, grade, year, term) |
| `/assessments/[id]` | Blueprint detail + indicator management | ❌ Placeholder | B32, B33, B35, B36 | Detail view + indicator linking UI |
| `/assessments/sessions/new` | Create session (from blueprint) | ❌ Missing | B37 | Step selector: pick blueprint → pick class → pick date |
| `/assessments/sessions/[id]` | Session detail + results entry matrix | ❌ Missing | B39, B40, B41, B50, B51 | Results grid (students × indicators) + status action buttons |
| `/assessments/admin/queue` | Admin review queue (PENDING_REVIEW + APPROVED) | ❌ Missing | B42–B47 | Two-tab DataTable with approve/reject/publish actions |
| `/reports/student/[id]` | Compiled student report (includes assessment) | ❌ Placeholder | B48, B26 | Competency snapshot compilation + report UI |

---

## 16. Cross-Cutting Gaps

### 16.1 Term Reports (Both Modules)

`term_reports` is a compiled snapshot table that should aggregate:
- `attendance_snapshot` (JSONB) — from `attendance_term_summaries`
- `behavior_snapshot` (JSONB) — from `behavior_notes` (approved only)
- `competency_snapshot` (JSONB) — from `learner_rubric_results` or `cbc_term_competency_summaries`

| # | Feature | Backend | Frontend | Priority | User Story | Details |
|---|---------|---------|----------|----------|------------|---------|
| C1 | Term report generation service | ❌ Missing | ❌ | **High** | As an admin, I want to generate compiled term reports for one or all students so that parents receive official end-of-term summaries. | Needs service to query `attendance_term_summaries`, `behavior_notes` (approved), and assessment results; compile into JSONB; upsert into `term_reports`. |
| C2 | Term report publishing workflow | ❌ Missing | ❌ | **High** | As an admin, I want to publish generated reports so they become visible to parents. | Needs DRAFT→PUBLISHED transition endpoints + permissions. |
| C3 | Term report viewing (admin) | ❌ | ❌ Missing | **High** | As an admin, I want to view and download a compiled report before publishing. | `/reports/terms/[term_id]` (admin branch) is placeholder. Needs report preview UI. |
| C4 | Term report viewing (parent) | ❌ | ❌ Missing | **High** | As a parent, I want to see my child's published term report so that I can review their term progress. | `/reports/terms/[term_id]` (parent branch) is placeholder. Needs compiled report viewer. |
| C5 | Bulk term report generation | ❌ | ❌ Missing | **Medium** | As an admin, I want to generate reports for an entire class or grade level at once. | `/reports/bulk-export` is placeholder. Needs class/grade selector + batch generation endpoint. |
| C6 | Term report PDF export | ❌ | ❌ Missing | **Medium** | As an admin/parent, I want to download a term report as PDF for printing or offline storage. | Needs PDF generation service + download button. |

### 16.2 Behavior Cross-Link

| # | Feature | Status | Priority | Details |
|---|---------|--------|----------|---------|
| C7 | Behavior notes in term reports | ❌ Missing | **Medium** | `behavior_snapshot` column exists in `term_reports` but no service populates it from `behavior_notes`. Should include approved notes only (category, description, date, subject). |
| C8 | Behavior summary in parent attendance view | ❌ Missing | **Low** | No behaviour summary card in `ParentAttendanceSummary`. Could show recent flags alongside attendance. |

*Note: Attendance already integrates with behaviour via the flag icon in `TeacherAttendanceRoster` (opens `CreateBehaviorNoteDialog`). The missing part is the behaviour data flowing into reports.*

### 16.3 Notifications

| # | Feature | Status | Priority | User Story | Details |
|---|---------|--------|----------|------------|---------|
| C9 | Absence notification (SMS/email) | ❌ Missing | **Medium** | As a parent, I want to be notified when my child is marked absent or late so that I can follow up. | No SMS gateway or email template system integrated. Real-time notification would require WebSocket/polling. |
| C10 | Assessment results published notification | ❌ Missing | **Medium** | As a parent, I want to be notified when new assessment results are published so that I can view them. | Needs to hook into the PUBLISH status transition. |
| C11 | Term report available notification | ❌ Missing | **Medium** | As a parent, I want to be notified when my child's term report is published. | Needs to hook into term report PUBLISH transition. |
| C12 | Daily digest (admin) | ❌ Missing | **Low** | As an admin, I want a daily email summary of attendance completion rates across classes. | No scheduled job or digest template. |

### 16.4 Row-Level Security (RLS) Policies

| Table | RLS Enabled | Policy Scope | Verification Status |
|-------|-------------|-------------|-------------------|
| `attendance_records` | ✅ | Multi-tenant (tenant_id) | ⚠️ Verify |
| `attendance_term_summaries` | ✅ | Multi-tenant (tenant_id) | ⚠️ Verify |
| `assessment_blueprints` | ✅ | Multi-tenant (tenant_id) | ⚠️ Verify |
| `assessment_sessions` | ✅ | Multi-tenant + parent scope | ⚠️ Verify parent-scoped policy |
| `learner_rubric_results` | ✅ | Multi-tenant (tenant_id) | ⚠️ Verify |
| `term_reports` | ✅ | Multi-tenant (tenant_id) | ⚠️ Verify |

**Known concerns:**
- Parent-scoped policy for `assessment_sessions` exists in migration (line 3039) — needs verification
- Teacher scope: should teachers see only their own classes' sessions? Currently filtered at handler level via `assessed_by_user_id`, not via RLS

### 16.5 Navigation & Routing

| # | Feature | Status | Priority | Details |
|---|---------|--------|----------|---------|
| C13 | Assessment pages in sidebar nav | ❌ Missing | **High** | Once frontend is built, sidebar needs links to `/assessments` (teacher: my sessions) and `/assessments/admin/queue` (admin: review queue). |
| C14 | Reports in sidebar nav | ❌ Partial | **Medium** | `/reports/terms/[term_id]` exists but not linked from nav. Admin needs ability to browse students and generate reports. |
| C15 | Role-based route gating | ✅ Complete | — | All assessment routes use `middleware.RequireAuth` or `middleware.RequireRole`. Frontend uses `getVerifiedRole()`. |

---

## Summary

### Status by Module

| Module | Backend Implementation | Frontend Implementation | Overall Status |
|--------|----------------------|------------------------|----------------|
| **Attendance** | ✅ Complete (7 endpoints + worker) | ✅ Complete (6 components + 4 pages + hooks + API client) | ✅ **Production-ready** |
| **Assessments** | ✅ Complete (23 endpoints) | ❌ Not started (3 "Coming Soon" pages) | ❌ **Not usable** |
| **Term Reports** | ❌ Schema only (no service) | ❌ Not started | ❌ **Not implemented** |
| **Grading Scales** | ❌ Schema only (no CRUD endpoints) | ❌ Not started | ❌ **Not implemented** |
| **Parent Results View** | ✅ Endpoint exists | ❌ Not started | ❌ **Not usable** |
| **Notifications** | ❌ Missing | ❌ Missing | ❌ **Not implemented** |

### Implementation Priority Matrix

| Priority | Feature Count | Items |
|----------|--------------|-------|
| **Critical** | 12 | B1–B4, B9–B11, B14, B16, B21, B30–B32, B37–B39, B42, B47, B50 |
| **High** | 20 | A1–A3, B5–B8, B15, B17–B19, B22–B23, B25–B26, B33–B36, B43–B45, B48, B51, C1–C4 |
| **Medium** | 15 | A4–A6, A10, B12–B13, B20, B27, B40–B41, B46, C5–C7, C9–C11, C13–C14 |
| **Low** | 8 | A7–A9, B24, B28–B29, C8, C12 |

**Total unimplemented features: 55** (across attendance, assessments, term reports, grading scales, notifications, and navigation)
