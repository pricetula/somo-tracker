# Bulk Invite vs Student Import — Detailed Differences

> **Last updated:** 2026-07-09  
> **Purpose:** Documents every meaningful difference between the staff bulk invitation flow and the student bulk import flow, so developers can navigate both systems without confusion.

---

## Table of Contents

1. [Shared Infrastructure](#1-shared-infrastructure)
2. [Backend: Importer Implementation](#2-backend-importer-implementation)
3. [Backend: Handler & Routing](#3-backend-handler--routing)
4. [Frontend: Page Flows](#4-frontend-page-flows)
5. [Frontend: File Import Wizard](#5-frontend-file-import-wizard)
6. [Frontend: Manual Form](#6-frontend-manual-form)
7. [Backend: Module Wiring](#7-backend-module-wiring)
8. [API Contracts](#8-api-contracts)
9. [Database Schema](#9-database-schema)
10. [Failure Types](#10-failure-types)
11. [Validation Rules](#11-validation-rules)
12. [Idempotency & Retry Semantics](#12-idempotency--retry-semantics)
13. [Edge Cases](#13-edge-cases)

---

## 1. Shared Infrastructure

Both flows share these components **without modification**:

| Component | Location | What it provides |
|-----------|----------|------------------|
| `imports.Service` | `backend/internal/imports/service.go` | `CreateJob()`, `ProcessChunk()`, `CancelJob()`, `GetJob()`, `GetFailures()`, `CleanupExpiredData()` |
| `Importer` interface | `backend/internal/imports/domain.go` | `Validate()`, `ResolveReferences()`, `BulkInsert()`, `InsertOne()` — contract implemented by both domains |
| `ImporterRegistry` | `backend/internal/imports/domain.go` | Global `map[ImportJobType]Importer{}` — both register at startup |
| Asynq workers | `backend/internal/imports/module.go` | Same concurrency=3, same `imports:process_chunk` handler, same chunk claiming/completion |
| `import_jobs`, `import_job_staging`, `import_job_chunks`, `import_job_failures` | `migrations/000001` | Same tables, same status enums, same partial unique index for one-active-job-per-school |
| `GET /api/v1/imports/:job_id` | `backend/internal/imports/handler.go` | Same polling endpoint |
| `GET /api/v1/imports/:job_id/failures` | `backend/internal/imports/handler.go` | Same failures endpoint |
| `POST /api/v1/imports/:job_id/cancel` | `backend/internal/imports/handler.go` | Same cancellation endpoint |
| `GET /api/v1/schools/:school_id/imports/active` | `backend/internal/imports/handler.go` | Same active-job check endpoint |
| `ImportProgress` component | `frontend/src/features/students/components/students-import/import-progress.tsx` | Same progress bar, polling backoff, stalled-job messaging, cancel button, failure display, Done/Retry buttons |
| `ImportJob`, `ImportJobStatus`, `ImportRowFailure`, `ImportResponse` types | `frontend/src/lib/api/imports.ts` | Shared TypeScript types |

---

## 2. Backend: Importer Implementation

### Student: `StudentImporter` (`backend/internal/students/importer.go`)

```go
type StudentImporter struct {
    repo ImportRepository
}
```

**Dependencies:** Only the DB repository (`ImportRepository`).

**Validate()** — per-row schema checks:
- `full_name` non-empty
- `gender` is "M" or "F"
- `date_of_birth` parseable ISO date, not future, ≤25 years old
- `class_id` (if present) well-formed UUID

**ResolveReferences()** — enriches rows with context + checks:
- Injects `tenant_id`, `school_id`, `academic_term_id`, `academic_year_id` from job metadata
- **Class existence check:** queries `cbc_classes` to verify class exists + belongs to same tenant/school → fail with `INVALID_CLASS_REFERENCE`
- **Insert-time duplicate safety net:** queries `cbc_students` for existing `admission_number`, `upi_number`, `knec_assessment_number` → fail with `DUPLICATE_*` error types

**BulkInsert():**
```go
func (si *StudentImporter) BulkInsert(...) (int, error) {
    return 0, fmt.Errorf("student import requires per-row inserts")
}
```
Always returns error → forces savepoint fallback for every row.

**InsertOne()** — inside a savepoint:
1. `INSERT INTO cbc_students (...)` with `ON CONFLICT (school_id, staging_row_id) DO UPDATE SET staging_row_id = EXCLUDED.staging_row_id` for idempotent retry
2. If `class_id` present: `INSERT INTO cbc_student_enrollments (...)`
3. Translates Postgres constraint violations to typed `ImportError` (FK → `INVALID_CLASS_REFERENCE`, unique → `BUSINESS_RULE_VIOLATION`, unknown → `DB_CONSTRAINT_VIOLATION`)
4. **No external API calls** — all operations are DB-local

### Staff Invite: `StaffInviteImporter` (`backend/internal/invitations/importer.go`)

```go
type StaffInviteImporter struct {
    repo           Repository
    stytch         StytchInviteSender
    frontendURL    string
    createMemberFn func(...) (string, error)  // overridable for tests
    inviteEmailFn  func(...) (string, error)  // overridable for tests
}
```

**Dependencies:**
- DB repository (`Repository`)
- Stytch identity provider (`StytchInviteSender` — subset of `auth.IdentityProvider`)
- `frontendURL` from config (for constructing invite redirect URL)

**Validate():**
- Email non-empty after trim
- Email matches basic regex (`/^[^\s@]+@[^\s@]+\.[^\s@]+$/`)
- Trims whitespace from email and name

**ResolveReferences():**
- Injects `tenant_id`, `school_id`, `role`, `stytch_org_id`, `invited_by`, `import_job_id` from job metadata
- **Duplicate check:** batch queries `users` table (email has account?) AND `invitations` table (email has pending invite?) → fail with `DUPLICATE_EMAIL`
- Deduplicates within the batch before querying (collects unique emails only)

**BulkInsert():**
```go
func (si *StaffInviteImporter) BulkInsert(...) (int, error) {
    return 0, fmt.Errorf("staff invite requires per-row Stytch API calls")
}
```
Same pattern — always forces savepoint fallback.

**InsertOne()** — inside a savepoint:
1. **`stytch.CreateMember(ctx, orgID, email, name)`** — external API call to Stytch
   - On retry: detects `member_already_exists` error string → proceeds gracefully
2. **`stytch.InviteMemberByEmail(ctx, orgID, email, name, redirectURL)`** — external API call to send invite email
   - On retry: sends another email (acceptable)
3. **`INSERT INTO invitations (...)`** — local DB insert with `gen_random_uuid()` for token
   - Catches unique constraint violation on `uq_invitations_school_email_pending` → `DUPLICATE_EMAIL`
4. **External API calls + DB insert in same savepoint** — if DB fails, Stytch calls are "wasted" (redundant on retry, but harmless)

### Key differences in InsertOne

| Aspect | Student Import | Staff Invite |
|--------|---------------|--------------|
| Operations | 1–2 DB inserts (student + optional enrollment) | 2 Stytch API calls + 1 DB insert |
| Idempotency mechanism | `ON CONFLICT (school_id, staging_row_id)` on `cbc_students` | Stytch returns "member already exists" → proceed; unique constraint on `invitations` |
| Rollback semantics | Savepoint rollback → no side effects | Savepoint rollback → Stytch calls already made (harmless on retry) |
| External dependencies | None | Stytch API availability |
| Fallback on retry | ON CONFLICT treats duplicate as success | String-matches "member_already_exists" |

---

## 3. Backend: Handler & Routing

### Student: `internal/students/handler.go`

```
POST /api/v1/students/import  →  h.BulkImport
```

Handler flow:
1. Extracts tenant, school, user from request context
2. Resolves **academic year + term** server-side from school config (`academicYearsSvc.GetCurrentAcademicYearID` / `GetCurrentAcademicTermID`)
3. Validates row count ≤ `MaxImportRows` (5000)
4. Enforces body size limit via per-route middleware (`bodySizeLimit` — 15MB)
5. Builds `metadata` with `{ academic_term_id, academic_year_id }`
6. Calls `imports.Service.CreateJob()` with `JobType = STUDENT_IMPORT`, no `Role`
7. Supports optional `idempotency_key` for safe retry

Extra endpoints on the same handler:
- `POST /api/v1/students/check-duplicates` — proactive duplicate check before submit (used by frontend manual form)
- Full CRUD: `GET /list`, `POST /`, `GET /:id`, `PUT /:id`, `DELETE /:id`
- Enrollment: `POST /:id/enrollments`, `GET /:id/enrollments`

### Staff Invite: `internal/invitations/handler.go`

```
POST /api/v1/staff/invite  →  h.BulkInvite
```

Handler flow:
1. Extracts tenant, school, user from request context
2. **Validates role** — must be one of `SCHOOL_ADMIN`, `TEACHER`, `NURSE`, `FINANCE` (case-insensitive, uppercased)
3. Validates row count ≤ `MaxImportRows` (5000)
4. **Resolves Stytch org ID** for the tenant (`GetStytchOrgID`)
5. Builds `metadata` with `{ role, invited_by, stytch_org_id }`
6. Calls `imports.Service.CreateJob()` with `JobType = STAFF_INVITE`, `Role = body.Role`
7. **No idempotency key support** (simpler — no key in request/response)

Existing endpoints on the same handler:
- `GET /api/v1/invitations` — list invitations with filters (pre-existing)

**No check-duplicates endpoint needed** — duplicate checking happens in `ResolveReferences` during async processing.

---

## 4. Frontend: Page Flows

### Student: `StudentsImportForm`

1. **Active-job check** on mount via `GET /schools/:id/imports/active`
2. **Selector** (`StudentsImportSelector`): "Manual Entry" or "Import File"
3. **Manual** (`StudentManualImportForm`): table with 7 fields per row
4. **File** (`FileImporter`): 6-step wizard
5. **Progress** (`ImportProgress`): shared polling component
6. IndexedDB cleanup on terminal status (`clearAllSessions`)

State machine:
```
idle → selector → manual/file → active-job → idle
```

### Staff Invite: `BulkInviteForm`

1. **Active-job check** on mount via `GET /schools/:id/imports/active`
2. **Selector** (`BulkInviteSelector`): "Manual Entry" or "Import File"
3. **Manual** (`BulkInviteManualForm`): table with 2 fields per row (email, name)
4. **File** (`BulkInviteFileImporter`): 4-step wizard
5. **Progress** (`ImportProgress`): **reused directly** from students
6. **No IndexedDB cleanup** — no IndexedDB session to clean

State machine (identical pattern):
```
idle → selector → manual/file → active-job → idle
```

**Role prop:** `BulkInviteForm` accepts a `role` prop that threads through all child components to `StepStreaming` and ultimately to the API call. The student form has no equivalent — student import doesn't take a role.

---

## 5. Frontend: File Import Wizard

### Student: `FileImporter` — 6 steps

```
Upload → Column Mapping → Class Resolve → Data Review → Streaming → (Progress)
```

| Step | Component | What it does | IndexedDB? |
|------|-----------|-------------|------------|
| Upload | `StepUpload` | CSV/Excel/ODS/TSV parsing with multi-sheet selection, storage estimate, 15MB file cap, BOM stripping, serial date conversion | Persists parsed file (≤500 rows) + session meta |
| Column Mapping | `StepColumnMapping` | Smart-match auto-mapping using dictionary of 30+ synonyms, popover combobox for manual mapping | Persists mapping + step |
| Class Resolve | `StepClassResolve` | Resolves class names → class UUIDs via backend API, handles ambiguous/unmatched classes | Persists class mappings + step |
| Data Review | `StepDataReview` | Full editable table with validation, error highlighting, within-batch + against-DB duplicate markers | Persists staged records + quota pre-check |
| Streaming | `StepStreaming` | Shows "Import N Students" button, on click: saves records, calls submit | Final write + cleanup on terminal |

**IndexedDB stores used:**
- `import_meta` — session metadata (step, column/class mappings, file metadata)
- `student_import_staging` — per-row staged records
- `parsed_file` — raw parsed file (up to 500 rows)

**Crash recovery:**
- Auto-restore on mount if session < 24h old
- Prompt resume if > 24h old
- Foreign-school detection
- Size guard (parsed_file_too_large flag)

### Staff Invite: `BulkInviteFileImporter` — 4 steps

```
Upload → Column Mapping → Data Review → Streaming → (Progress)
```

| Step | Component | What it does | IndexedDB? |
|------|-----------|-------------|------------|
| Upload | `StepUpload` | CSV/Excel/ODS/TSV parsing with file type detection, BOM stripping, serial date conversion, file size limit | **None** — no IndexedDB |
| Column Mapping | `StepColumnMapping` | Smart-match auto-mapping using dictionary of ~10 synonyms, select-based manual mapping, live preview | **None** |
| Data Review | `StepDataReview` | Editable table with inline edit (pencil icon), validation (email format + duplicates), row-level error display | **None** |
| Streaming | `StepStreaming` | Shows "Send N Invitations" button, on click: calls `submitBulkInvite`, delegates to parent | **None** |

**No Class Resolve step** — invitations don't have class associations.

**No IndexedDB** — the data is simpler (just email strings), so crash recovery via IndexedDB isn't worth the complexity. If the user refreshes mid-wizard, they re-upload.

### Wizard step indicator difference

| Student | Staff Invite |
|---------|-------------|
| Uses inline step labels in the parent `FileImporter` (no visual step indicator component) | Uses a dedicated step indicator bar with active/completed/done states and icons |

---

## 6. Frontend: Manual Form

### Student: `StudentManualImportForm`

- 7 fields per row: `full_name`, `gender`, `date_of_birth`, `upi_number`, `knec_assessment_number`, `admission_number`, `class_id`
- Gender select (M/F), DatePicker, ClassCombobox
- **Within-batch duplicate detection** in `useMemo` (checks `admission_number`, `upi_number`, `knec_number`)
- **Against-DB duplicate check** via `POST /api/v1/students/check-duplicates` on submit click (blocking — shows toast if conflicts)
- **Idempotency key** generated via `crypto.randomUUID()` at submit start, reused on retry
- Complex field-level error merging: within-batch errors + API errors overlaid
- Row count: `N / 5,000 rows` footer
- Submit button: `Import N Students`

### Staff Invite: `BulkInviteManualForm`

- 2 fields per row: `email` (required), `full_name` (optional)
- Simple text Input for both (no combobox, date picker, or select)
- **Within-batch duplicate detection** in `useMemo` (checks email only)
- **No against-DB duplicate check** on submit — handled async in ResolveReferences
- **No idempotency key** — simpler, one-active-job-per-school prevents collisions
- Row count: `N of M rows ready` footer
- Submit button: `Invite N`
- **Role context** displayed/used during submit (via `role` prop)

---

## 7. Backend: Module Wiring

### Student: `internal/students/module.go`

```go
var Module = fx.Module("students",
    fx.Provide(
        fx.Annotate(NewRepository, fx.As(new(StudentRepository)), fx.As(new(ImportRepository))),
        NewService,
        NewHandler,
        NewStudentImporter,
    ),
    fx.Invoke(func(svc *Service, repo ImportRepository) { svc.SetImportRepo(repo); }),
    fx.Invoke(func(h *Handler, impSvc *imports.Service) { h.SetImportService(impSvc); }),
    fx.Invoke(func(h *Handler, aySvc *academicyears.Service) { h.SetAcademicYearsService(aySvc); }),
    fx.Invoke(registerStudentImporter),
)
```

Wires:
- Repository as both `StudentRepository` and `ImportRepository` (same `PgRepository` behind two interfaces)
- Academic years service into handler (for resolving current year/term)
- Import service into handler

### Staff Invite: `internal/invitations/handler.go`

```go
var Module = fx.Module("invitations",
    fx.Provide(
        fx.Annotate(NewRepository, fx.As(new(Repository))),
        NewService,
        NewHandler,
        NewStaffInviteImporter,
    ),
    fx.Invoke(func(h *Handler, impSvc *imports.Service) { h.SetImportService(impSvc); }),
    fx.Invoke(func(sii *StaffInviteImporter, idp auth.IdentityProvider) { sii.SetStytchAdapter(idp); }),
    fx.Invoke(func(sii *StaffInviteImporter, cfg config.Config) { sii.SetFrontendURL(cfg.FrontendURL); }),
    fx.Invoke(registerStaffInviteImporter),
)
```

Wires:
- Repository as `Repository` interface only (no dual-interface annotation)
- **Stytch adapter** (`auth.IdentityProvider`) into importer — unique to staff invite
- **Config** (for `FrontendURL`) into importer — unique to staff invite
- Import service into handler
- **No academic years service** — not needed for invitations

---

## 8. API Contracts

### Endpoint paths

| Action | Student | Staff Invite |
|--------|---------|-------------|
| Create job | `POST /api/v1/students/import` | `POST /api/v1/staff/invite` |
| Check duplicates | `POST /api/v1/students/check-duplicates` | **Not needed** (handled in async ResolveReferences) |
| Poll job status | `GET /api/v1/imports/:job_id` | Same endpoint (reused) |
| Get failures | `GET /api/v1/imports/:job_id/failures` | Same endpoint (reused) |
| Cancel job | `POST /api/v1/imports/:job_id/cancel` | Same endpoint (reused) |
| Active job check | `GET /api/v1/schools/:school_id/imports/active` | Same endpoint (reused) |

### Request bodies

**Student import:**
```json
{
  "idempotency_key": "uuid-v4-optional",
  "rows": [
    {
      "full_name": "Alice Wanjiku",
      "gender": "F",
      "date_of_birth": "2010-05-15",
      "upi_number": "UPI12345",
      "knec_assessment_number": "KNEC67890",
      "admission_number": "ADM001",
      "class_id": "uuid-of-class"
    }
  ]
}
```

**Staff invite:**
```json
{
  "role": "TEACHER",
  "rows": [
    { "email": "alice@example.com", "full_name": "Alice Wanjiku" },
    { "email": "bob@example.com" }
  ]
}
```

### Response (same shape)

Both return `ImportResponse`:
```json
{
  "job_id": "uuid",
  "total_records": 100,
  "total_chunks": 1,
  "status": "processing",
  "is_replay": false
}
```

---

## 8b. Active School Resolution

A simplification opportunity shared by both flows: the frontend currently passes `schoolId`
explicitly to the `getActiveImportJob(schoolId)` call, but this is **redundant** — the
backend auth middleware already sets `c.Locals("active_school_id")` from the
authenticated session.

**Current pattern (both flows):**

```typescript
// frontend/src/features/students/components/students-import/students-import.tsx
const schoolId = me?.school_id;
getActiveImportJob(schoolId).then(...)
```

```
GET /api/v1/schools/:school_id/imports/active
```

The endpoint takes `school_id` from the URL path and validates it against the caller's
tenant. But since the auth middleware already resolves the caller's active school from
the Stytch session, the endpoint could be simplified to:

```
GET /api/v1/imports/active
```

And resolve `school_id` from `c.Locals("active_school_id")` directly. This would
eliminate the need for the frontend to pass schoolId at all.

**Affected endpoints across both flows:**

| Endpoint | Frontend call site | Backend handler |
|----------|-------------------|----------------|
| `GET /schools/:school_id/imports/active` | `StudentsImportForm` + `BulkInviteForm` (both) | `GetActiveJobAPI` in `imports/handler.go` |

## 9. Database Schema

### Shared (identical for both)

All import infrastructure tables are shared:
- `import_jobs` — with `role` column (required for `STAFF_INVITE`, NULL for `STUDENT_IMPORT`)
- `import_job_staging` — generic `raw_data` JSONB column
- `import_job_chunks` — generic
- `import_job_failures` — generic

### Target tables differ

| Aspect | Student Import | Staff Invite |
|--------|---------------|-------------|
| Target table(s) | `cbc_students` + `cbc_student_enrollments` | `invitations` |
| Idempotent insert anchor | `staging_row_id` column on `cbc_students` with `ON CONFLICT` | `uq_invitations_school_email_pending` unique index (prevents duplicate pending invites) |
| Academic context | References `academic_years`, `academic_terms`, `cbc_classes` via FK | No academic context |
| Staging row cleanup | `ON DELETE SET NULL` from `cbc_students.staging_row_id` | `import_job_id` column on `invitations` with `ON DELETE SET NULL` |

---

## 10. Failure Types

### Shared failure types

Both can produce these:
- `SCHEMA_VALIDATION`
- `DATABASE_CONSTRAINT`
- `BUSINESS_RULE_VIOLATION`

### Student-only failure types

| Type | Source | Meaning |
|------|--------|---------|
| `INVALID_CLASS_REFERENCE` | ResolveReferences / InsertOne | class_id doesn't exist or belongs to different school |
| `DB_CONSTRAINT_VIOLATION` | InsertOne | Unmapped Postgres constraint |
| `DUPLICATE_ADMISSION_NUMBER` | ResolveReferences | admission_number already exists |
| `DUPLICATE_UPI_NUMBER` | ResolveReferences | upi_number already exists |
| `DUPLICATE_KNEC_NUMBER` | ResolveReferences | knec_assessment_number already exists |

### Staff-invite-only failure types

| Type | Source | Meaning |
|------|--------|---------|
| `DUPLICATE_EMAIL` | ResolveReferences / InsertOne | Email exists in users table OR has pending invitation |
| `INVALID_EMAIL_FORMAT` | Validate | Email doesn't match basic format regex |
| `STYTCH_API_ERROR` | InsertOne | Stytch CreateMember or InviteMemberByEmail failed |
| `INVITATION_INSERT_FAILED` | InsertOne | DB insert of invitation record failed |

### New enum values added to `import_failure_type`

Staff invite added these via migration `000003_bulk_invite`:
```sql
ALTER TYPE import_failure_type ADD VALUE 'DUPLICATE_EMAIL';
ALTER TYPE import_failure_type ADD VALUE 'INVALID_EMAIL_FORMAT';
ALTER TYPE import_failure_type ADD VALUE 'STYTCH_API_ERROR';
ALTER TYPE import_failure_type ADD VALUE 'INVITATION_INSERT_FAILED';
```

---

## 11. Validation Rules

### Frontend (client-side)

| Rule | Student Import | Staff Invite |
|------|---------------|-------------|
| Required field | `full_name` (min 2 chars, max 100, Unicode letters only) | `email` (non-empty) |
| Format validation | Gender M/F, DOB parseable + not future + ≤20yr old | Email regex `/^[^\s@]+@[^\s@]+\.[^\s@]+$/` |
| Within-batch duplicates | `admission_number`, `upi_number`, `knec_assessment_number` (case-insensitive) | `email` (case-insensitive) |
| Against-DB duplicates | Checked pre-submit via `POST /students/check-duplicates` | **Not checked on frontend** — deferred to async ResolveReferences |
| Row limit | 5,000 | 5,000 (same `MaxImportRows`) |

### Backend (in the Importer)

| Rule | Student Import | Staff Invite |
|------|---------------|-------------|
| Schema (Validate) | full_name, gender M/F, DOB, class_id UUID | Email format regex |
| References (ResolveReferences) | Class existence + tenant scope + duplicate field values | Email in users? Email in invitations? |
| Insert (InsertOne) | DB constraints (FK, unique) | Stytch API errors + DB unique constraint |

---

## 12. Idempotency & Retry Semantics

### Student Import

- **Optional `idempotency_key`** — frontend generates with `crypto.randomUUID()` at submit start
- **Backend:** `INSERT ... ON CONFLICT (tenant_id, school_id, idempotency_key) DO NOTHING` with `payload_hash` comparison
- **Same key + same payload** → HTTP 200 `is_replay: true`
- **Same key + different payload** → HTTP 409 `duplicate_import`
- **No key** → always creates a new job (no dedup)
- Frontend reuses key on transient network failure; clears on success

### Staff Invite

- **No idempotency key** — no key in request or response
- **One-active-job-per-school** is the only collision guard
- Retry semantics are handled by Asynq redelivery + Stytch idempotency:
  - Stytch `CreateMember` retry → "member already exists" → proceed
  - Stytch `InviteMemberByEmail` retry → sends another email → acceptable
  - DB insert retry → unique constraint violation → `DUPLICATE_EMAIL` error

---

## 13. Edge Cases

| Scenario | Student Import | Staff Invite |
|----------|---------------|-------------|
| **Concurrent jobs for same school** | Blocked by partial unique index → 409 `import_already_in_progress` | Same behavior (shared index) |
| **No active academic year/term** | 400 `no_active_academic_year` / `no_active_academic_term` | Not applicable (no academic context) |
| **Tenant has no Stytch org ID** | Not applicable | All rows fail in ResolveReferences with `BUSINESS_RULE_VIOLATION` |
| **Stytch API down** | Not applicable | Savepoint rolls back → Asynq retries (up to 3) → permanent failure |
| **Duplicate within Asynq retry** | `ON CONFLICT (school_id, staging_row_id)` treats as success | Stytch "member_already_exists" → proceed; DB unique constraint → `DUPLICATE_EMAIL` |
| **Crash mid-wizard (browser refresh)** | IndexedDB session recovery restores state | **No recovery** — user re-uploads the file |
| **Body size limit** | 15MB per-route middleware | **No body size middleware** — staff invite rows are small (just email strings) |
