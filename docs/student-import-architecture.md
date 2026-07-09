# Student Import — Architecture & Data Flow

> **Last updated:** 2026-07-09  
> **Scope:** Frontend `features/students/components/students-import/` ↔ Backend `internal/students/` + `internal/imports/`  
> **Owner:** Platform team

---

## Table of Contents

1. [Overview](#1-overview)
2. [Architecture Diagram](#2-architecture-diagram)
3. [Import Paths](#3-import-paths)
4. [Data Flow: End-to-End](#4-data-flow-end-to-end)
5. [Frontend Components](#5-frontend-components)
6. [Backend Components](#6-backend-components)
7. [Key API Endpoints](#7-key-api-endpoints)
8. [Request/Response Contracts](#8-requestresponse-contracts)
9. [Error Handling](#9-error-handling)
10. [Important Rules for Agents](#10-important-rules-for-agents)
11. [Database Schema](#11-database-schema)
12. [Importer Interface & Registry](#12-importer-interface--registry)
13. [File Parsing Pipeline](#13-file-parsing-pipeline)
14. [Duplicate Detection Matrix](#14-duplicate-detection-matrix)
15. [Frontend Validation Rules](#15-frontend-validation-rules)
16. [IndexedDB Store Schema](#16-indexeddb-store-schema)
17. [Environment Configuration Reference](#17-environment-configuration-reference)
18. [Asynq Module Wiring](#18-asynq-module-wiring-fx-lifecycle)
19. [Testing Strategy](#19-testing-strategy)

---

## 1. Overview

Student import allows bulk-adding students via two paths:

- **Manual Entry** — user types student data into a table in the browser
- **File Import** — user uploads a CSV/Excel file, maps columns, resolves classes, reviews, then imports

Both paths ultimately call the same backend endpoint (`POST /api/v1/students/import`) and share the same progress tracking component (`<ImportProgress>`).

### Design principles

- **The frontend NEVER chunks data.** All rows are sent in a single POST request. The backend handles splitting into chunks for async processing.
- **The frontend polls for progress.** After submitting, the frontend polls `GET /api/v1/imports/:job_id` with a backoff schedule: every 1.5s for the first 30s, every 3s from 30s to 2min, then every 10s for long-running imports. This reduces load on both frontend and backend for jobs that take a while.
- **Failures are fetched after completion.** The frontend calls `GET /api/v1/imports/:job_id/failures` to display per-row error details.

---

## 2. Architecture Diagram

```
┌────────────────────────────────────────────────────────────────────┐
│                         FRONTEND                                   │
│                                                                    │
│  StudentsImportForm (parent orchestrator)                          │
│  ├── ImportSelector          ← pick "Manual" or "File"            │
│  ├── StudentManualImportForm ← user types rows in a table         │
│  │   └── submitStudentImport({ rows }) → job_id                    │
│  ├── FileImporter (wizard)                                         │
│  │   ├── StepUpload          ← CSV/Excel file parsing              │
│  │   ├── StepColumnMapping   ← map file columns to student fields  │
│  │   ├── StepClassResolve    ← resolve class names → IDs           │
│  │   ├── StepDataReview      ← review + edit records               │
│  │   └── StepStreaming       ← submitStudentImport({ rows })       │
│  │       └── submitStudentImport({ rows }) → job_id                │
│  └── ImportProgress (shared)  ← polls GET /imports/:job_id         │
│                                                                    │
│  API layer (src/lib/api/imports.ts):                               │
│  ├── submitStudentImport()    → POST /api/v1/students/import       │
│  ├── getImportJob()           → GET  /api/v1/imports/:job_id       │
│  └── getImportFailures()      → GET  /api/v1/imports/:job_id/failures │
└────────────────────────────────────────────────────────────────────┘
                              │  HTTP
                              ▼
┌────────────────────────────────────────────────────────────────────┐
│                         BACKEND                                    │
│                                                                    │
│  internal/students/handler.go                                      │
│  └── BulkImport()                                                  │
│      ├── Parse ImportRequest { rows: [...] }                       │
│      ├── Resolve current academic year + term (server-side)        │
│      └── Call imports.Service.CreateJob()                          │
│                                                                    │
│  internal/imports/service.go                                       │
│  └── CreateJob()                                                   │
│      ├── Create import_jobs row (status: processing)               │
│      ├── Write all rows to import_job_staging                      │
│      ├── Split into chunks of 100 (backend constant)               │
│      ├── Enqueue Asynq tasks (1 per chunk)                         │
│      └── Return { job_id, total_records, total_chunks, status }    │
│                                                                    │
│  Asynq Worker (internal/imports/module.go)                         │
│  └── ProcessChunk()                                                │
│      ├── 1. Load staging rows for chunk range                      │
│      ├── 2. Call Importer.Validate() → schema checks               │
│      ├── 3. Call Importer.ResolveReferences() → inject context     │
│      ├── 4. Call Importer.BulkInsert() → bulk or savepoint         │
│      │      fallback per row                                       │
│      ├── 5. Insert failures into import_job_failures               │
│      ├── 6. Mark staging rows as succeeded                         │
│      └── 7. AtomicChunkCompletion() → update job counters          │
│                                                                    │
│  internal/students/importer.go (Importer implementation)           │
│  └── StudentImporter                                               │
│      ├── Validate() → checks full_name, gender, class_id           │
│      ├── ResolveReferences() → injects tenant/school/term/context  │
│      ├── BulkInsert() → returns error (forces savepoint fallback)  │
│      └── InsertOne() → inserts student + optional enrollment       │
│                                                                    │
│  GET /api/v1/imports/:job_id                                       │
│  └── Returns current state (status, counters) for polling          │
│                                                                    │
│  GET /api/v1/imports/:job_id/failures                              │
│  └── Returns paginated failure rows                                │
└────────────────────────────────────────────────────────────────────┘
```

---

## 3. Import Paths

### 3.1 Manual Entry (`/students/import`)

**User flow:**
1. Opens the import page/dialog
2. Selects "Manual Entry"
3. Adds rows via the table (full_name required, other fields optional)
4. Clicks "Import N Students"
5. Sees shared `<ImportProgress>` component with polling progress

**Key files:**
- `manual-import-form.tsx` — the form component
- `import-progress.tsx` — shared progress display

**Row limit (MaxImportRows = 5000):** The manual entry form blocks adding
rows once the count reaches `MaxImportRows`, showing a visible count of
`N / 5,000 rows` in the footer. The backend also enforces this limit at
the handler level — see [Section 6.3](#63-internalimportsservicego).

### 3.2 File Import (`/students/import`)

**User flow:**
1. Opens the import page/dialog
2. Selects "Import File"
3. Uploads CSV/Excel file
4. Maps file columns to student fields (e.g., "Student Name" → `full_name`)
5. Resolves class names to class IDs
6. Reviews and corrects records in a data table
7. Clicks "Start Import"
8. Sees shared `<ImportProgress>` component with polling progress

**Key files:**
- `file-importer.tsx` — wizard orchestrator
- `step-upload.tsx` — file parsing
- `step-column-mapping.tsx` — column mapping UI
- `step-class-resolve.tsx` — class name resolution
- `step-data-review.tsx` — record review/correction
- `step-streaming.tsx` — submit button (minimal, delegates to parent)
- `db.ts` — IndexedDB persistence for crash recovery

#### IndexedDB crash recovery (`db.ts`)

The wizard persists session state to IndexedDB for crash recovery across
page reloads and browser restarts. Three object stores are used:

| Store | Key | Contents |
|-------|-----|----------|
| `import_meta` | `session:<school_id>` | Current step, column/class mappings, file metadata, `updated_at` |
| `student_import_staging` | Auto-increment ID | Per-row staged records with payload, raw data, validation status |
| `parsed_file` | `parsed_file:<school_id>` | Raw parsed headers + rows (up to 500 rows; larger files skip persistence) |

**Session lifecycle:**
- **Saved after every step transition** — upload→mapping, mapping→class resolve, class resolve→data review, data review→streaming.
- **Auto-restored on mount** if a session exists for the current `school_id` and is less than 24 hours old (see staleness below).
- **Cleared on any terminal import status** — `completed`, `completed_with_errors`, `failed`, or `cancelled` — via the `onTerminalStatus` callback on `ImportProgress`, which fires as soon as the polling loop detects the terminal state.
- **Size guard:** If a parsed file exceeds 500 rows, the full row data is not persisted. The session is marked `parsed_file_too_large`. On resume, the user sees a message explaining the draft cannot be fully restored and is prompted to start from upload.
- **Foreign-school detection:** Sessions are keyed by `school_id`. If a persisted session exists for a different school than the currently active one, a distinct message is shown ("You have an unfinished import for a different school") with an option to discard it, rather than silently overwriting.
- **Staleness threshold:** Sessions older than 24 hours (`SESSION_STALE_MS`) are not auto-resumed. Instead, a prompt asks "Resume this draft or start fresh?", giving the user a choice. This prevents stale data from silently resurfacing while still allowing intentional resume of older drafts.

**Row limit (MaxImportRows = 5000):** The upload step (`step-upload.tsx`)
checks the parsed row count immediately after file parsing. If it exceeds
`MaxImportRows`, progression to column mapping is blocked and a clear
error message is shown telling the user to split the file. The parsed
count is displayed as `N / 5,000 rows` in the success alert for files
under the limit. See also backend enforcement in [Section 6.3](#63-internalimportsservicego).

---

## 4. Data Flow: End-to-End

### Step 1: Submit (Frontend)

Both paths convert student data to `ImportRow[]` and generate an
`idempotency_key` via `crypto.randomUUID()`, then call:

```typescript
const result = await submitStudentImport({
    idempotency_key: idempotencyKeyRef.current,
    rows: importRows,
});
// result = { job_id, total_records, total_chunks, status: "processing", is_replay: false }
```

The key is kept in a `useRef` and reused if the submit is retried due to
a transient network failure. See [Section 8 — Idempotency semantics](#idempotency-semantics)
for the full key lifecycle.

### Step 2: Parent captures job (Frontend)

The child component calls `onJobCreated(jobId, totalRecords)` which is handled by `StudentsImportForm`. This component sets `activeJob` state and renders `<ImportProgress>` instead of the form.

### Step 3: Poll for progress (Frontend)

`<ImportProgress>` polls:
```typescript
const current = await getImportJob(jobId);
// current = { status, total_records, processed_records, success_count, failed_count, ... }
```
Every 1.5s until `status` is one of: `completed`, `completed_with_errors`, `failed`, `cancelled`.

### Step 4: Backend creates job (Backend)

`BulkImport` handler in `internal/students/handler.go`:
1. Extracts `tenant_id`, `school_id`, `user_id` from request context
2. Resolves the current active academic year and term server-side
3. Builds metadata with `academic_term_id` and `academic_year_id`
4. Converts `ImportRow[]` → `json.RawMessage[]`
5. Calls `imports.Service.CreateJob()` which:
   - Computes a `payload_hash` from the rows (SHA-256 of the serialized array)
   - **If `idempotency_key` is present:** uses `INSERT ... ON CONFLICT DO NOTHING`
     for concurrent-safe dedup. If the INSERT succeeds the job is new;
     if it conflicts, fetches the existing job and compares `payload_hash`:
     - Hash match → returns existing job with `is_replay: true` (HTTP 200)
     - Hash mismatch → returns `duplicate_import` error (HTTP 409)
   - **If no `idempotency_key`:** always creates a new job (plain INSERT)
   - Creates `import_jobs` row with status `pending`
   - Writes all rows to `import_job_staging`
   - Splits into chunks of 100 (backend constant `ChunkSize`)
   - Enqueues Asynq tasks (1 per chunk)
   - Sets status to `processing`
   - Returns immediately

### Step 5: Async processing (Backend)

Asynq worker picks up each chunk task:

1. **Cancellation check** — Before attempting to claim the chunk, the worker checks the parent job's status. If the job status is `'cancelling'`, the chunk is atomically transitioned from `'pending'` to `'cancelled'` and the worker exits immediately without processing any rows. Chunks already `'processing'` when cancellation is requested are allowed to finish normally — cancellation is cooperative, not preemptive. (See `CancelPendingChunk` in `internal/imports/repository.go`.)

2. **Chunk claim** — Atomically transitions the chunk's `import_job_chunks.status` from `'pending'` to `'processing'` with a timestamp. If another worker or a redelivery already claimed the chunk, the UPDATE returns no rows and the worker exits immediately without processing, validating, or touching staging rows. (See `ClaimChunk` in `internal/imports/repository.go`.)

2. **Loads staging rows** for the chunk range, filtered to `status = 'pending'` only. Rows that were already marked `'succeeded'` or `'failed'` by a prior partial attempt (worker crash mid-chunk) are skipped — they are never reprocessed.

3. Calls `StudentImporter.Validate()` — schema checks (full_name required, gender M/F, class_id optional)

4. Calls `StudentImporter.ResolveReferences()` — injects tenant_id, school_id, academic_term_id, academic_year_id, and the staging_row_id into each row

5. Calls `StudentImporter.BulkInsert()` — returns error (forces savepoint fallback since student+enrollment needs per-row handling)

6. Falls back to `insertWithSavepoints()` → calls `InsertOne()` per row:
   - Each call opens a `SAVEPOINT`
   - **Inserts** into `cbc_students` with `staging_row_id` included (uses `ON CONFLICT (school_id, staging_row_id) WHERE staging_row_id IS NOT NULL DO UPDATE SET staging_row_id = EXCLUDED.staging_row_id` for defense-in-depth — if the row was already inserted by a prior attempt, this is treated as success, not a duplicate error)
   - If `class_id` is present, also inserts into `cbc_student_enrollments`
   - **Atomically marks** the staging row as `'succeeded'` via `MarkStagingRowSucceeded()` within the same savepoint — there is no code path where the insert commits but the staging row is left `'pending'`
   - On failure, the savepoint is rolled back (staging row stays `'pending'` for reprocessing)
   - On success, the savepoint is released

7. Failures go to `import_job_failures`

8. `AtomicChunkCompletion()` atomically transitions the chunk from `'processing'` to `'completed'` and, only on success, increments the job counters (`processed_records`, `success_count`, `failed_count`). If called on a chunk already `'completed'`, it no-ops — counters are never re-incremented. This mirrors the claim pattern: a single atomic `UPDATE import_job_chunks SET status = 'completed' WHERE id = $1 AND status = 'processing'` inside a CTE, gating the job counter UPDATE.

   **Cancellation detection:** When the last chunk for a job reaches a terminal state (`completed` or `cancelled`) and the job status is `'cancelling'`, `AtomicChunkCompletion` transitions the job to `'cancelled'`. Job counters reflect whatever actually completed before cancellation took effect — successfully-inserted rows are not zeroed out.

**Redelivery safety guarantee:** A chunk can never be processed twice in a way that duplicates data or double-counts progress, even under:
- Worker crash (mid-insert or mid-mark)
- Asynq visibility-timeout retry (same task delivered to another worker)
- Duplicate delivery (the same task arriving twice)

### Step 6: Completion (Frontend)

When poll detects terminal status:
- If `completed`: green success banner
- If `completed_with_errors` or `failed`: amber banner with:
  - Success/failure counts
  - Failed rows list (from `getImportFailures`)
  - "Retry failed" button
- If `cancelled`: slate banner with cancellation message indicating how many students were already added before cancellation took effect — no rollback occurred.
- "Done" button resets to the import selector

**Intermediate state:** The polling loop also recognizes `'cancelling'` as a non-terminal state. While `'cancelling'`, the progress bar continues to update, and a "Waiting for in-flight chunks to finish…" message is shown. Existing polling logic transitions naturally to `'cancelled'` once all chunks complete.

---

## 5. Frontend Components

### 5.1 `StudentsImportForm` (`students-import.tsx`)

The parent orchestrator. Manages:
- `selectedImportType`: "manual" | "file" | null
- `activeJob`: `{ jobId, totalRecords } | null`

When `activeJob` is set, renders `<ImportProgress>` instead of the child forms.

**Active-job redirect on mount:**
- On mount, `StudentsImportForm` calls `GET /api/v1/schools/:school_id/imports/active` using the current `school_id` from `useMe()`.
- If an active job is returned, it immediately sets `activeJob` and renders `<ImportProgress>` — the user never sees the import selector.
- This handles the common case: a user opens the import page while a colleague's import is already running.

**Active-job redirect on submit error:**
- When `submitStudentImport()` throws `import_already_in_progress` (a job was started between the mount check and the submit), both `manual-import-form.tsx` and `step-streaming.tsx` call `onJobCreated(activeJobId, totalRecords)` instead of showing a generic error toast.
- This effectively "adopts" the existing job, transitioning the parent to show `<ImportProgress>` for that job.

**Props:**
- `isDialogVersion: boolean` — whether rendered inside a modal

### 5.2 `StudentManualImportForm` (`manual-import-form.tsx`)

Table-based form where users type student data. Each row has:
- full_name (required)
- gender (select: M/F)
- date_of_birth (date picker)
- upi_number (text)
- knec_assessment_number (text)
- admission_number (text)
- class_id (ClassCombobox: optional, skip for no enrollment)

**Props:**
- `onReset: () => void`
- `onJobCreated: (jobId: string, totalRecords: number) => void`

**Duplicate detection (client-side):**
- **Within-batch duplicates** are detected on every render. Two or more rows sharing
  the same non-empty `admission_number`, `upi_number`, or `knec_assessment_number` are
  flagged inline with messages like "Duplicate admission number — also used in row 3".
  Rows with unresolved duplicates block submission.
- **Against-existing records** check runs once on submit click via
  `POST /api/v1/students/check-duplicates`. If any values already exist in the DB,
  matching rows are flagged inline ("already exists for this school") and submission
  is blocked.

### 5.3 `ImportProgress` (`import-progress.tsx`)

Shared progress component used by both import paths. Handles:
- Polling `GET /api/v1/imports/:job_id` with polling backoff (see below)
- Displaying progress bar with `processed / total` counts
- Showing stalled-job message when `last_progress_at` is stale (processing but no recent progress)
- Showing result banner on completion (green/amber/slate for cancelled)
- Fetching failures via `GET /api/v1/imports/:job_id/failures`
- "Cancel Import" button (visible only while status is `'processing'`)
- "Done" and "Retry failed" buttons

**Polling backoff schedule:**

To reduce load on long-running imports, the polling interval backs off over the job's total elapsed time (measured since the component mounted):

| Elapsed time | Poll interval |
|---|---|
| 0–30s | 1.5s (unchanged, fast for small imports) |
| 30s–2min | 3s |
| Beyond 2min | 10s (ceiling — never grows unbounded) |

The interval is recomputed every second. Once the job reaches a terminal status, polling stops entirely.

**Stalled-job messaging:**

While the job status is `'processing'` or `'cancelling'`, if `last_progress_at` is more than 2 minutes old (indicating no chunk has completed recently), a non-alarming inline message is shown:

> "This import is taking longer than usual — you can keep waiting or cancel it."

The amber banner appears alongside the progress bar and Cancel button. It disappears automatically when any of these conditions are met:
- `last_progress_at` updates (a chunk completes, resuming progress)
- The job reaches a terminal status

This is purely a messaging addition — polling behavior is unchanged for stalled jobs beyond the existing backoff schedule.

**`last_progress_at` field:**

The backend sets `last_progress_at` on every successful chunk completion via `AtomicChunkCompletion()`. When the last chunk completes (job reaches a terminal status), `last_progress_at` is also set as part of the final UPDATE. This gives the frontend a reliable "last meaningful progress" signal regardless of `created_at` age.

**Props:**
- `jobId: string`
- `totalRecords: number`
- `onDone: () => void`
- `onRetry?: (failedPayloads: Record<string, unknown>[]) => void`
- `onTerminalStatus?: () => void` — fires exactly once when the polling loop first detects a terminal status (`completed`, `completed_with_errors`, `failed`, or `cancelled`). Used by the parent to trigger IndexedDB cleanup (G2). Fires regardless of which button the user clicks, so an abandoned tab also triggers cleanup.

**Cancellation behavior:
- The "Cancel Import" button calls `POST /api/v1/imports/:job_id/cancel`.
- On click, the button is immediately disabled (prevents double-clicks).
- The existing polling loop picks up the `'cancelling'` → `'cancelled'` transition naturally — no special-case polling needed.
- When the job reaches `'cancelled'`, the result banner displays: "Import cancelled — N students were already added before cancellation took effect." It does not imply a full rollback.

### 5.4 `StepStreaming` (`file-importer/step-streaming.tsx`)

Minimal submit component for file import. Loads valid staged records from IndexedDB, shows "Import N Students" button, then delegates to parent via `onJobCreated`.

**Props:**
- `onComplete: () => void`
- `onError: (error: string) => void`
- `onJobCreated: (jobId: string, totalRecords: number) => void`

### 5.5 `FileImporter` (`file-importer/file-importer.tsx`)

6-step wizard orchestrator. Important details:
- Gender normalization: when mapping the gender column, the `normalizeGender()` function converts common variants ("Male", "Boy", "female", "F", etc.) to "M" or "F"
- Builds staged records via `buildStagedRecords()` which applies column mappings, class resolution, and gender normalization
- Validates records via `validateAndDetectDuplicates()` before the review step
- **School isolation (G4):** The component reads `school_id` from `useMe()` and passes it to all IndexedDB operations. Session keys are scoped as `session:<school_id>`, so switching schools shows a foreign-session message rather than corrupting data.
- **Session recovery on mount:** Checks for an existing IndexedDB session for the current `school_id`. If found and under 24 hours old, auto-restores the step, mappings, and (for early steps) the parsed file data. If over 24 hours old, shows a resume-or-restart prompt. If for a different school, shows a discard prompt.
- **Parsed file persistence (G1):** The raw parsed file (headers + rows) is persisted to a `parsed_file` store for files up to 500 rows. On resume from `column_mapping` or `class_resolving`, this data is loaded back into the `parsedFile` state, allowing the user to continue without re-uploading. The parsed-file data is deleted once staged records are built in `handleClassResolveComplete`.
- **Quota pre-check (G5):** Before `bulkWriteStagedRecords` writes validated records to IndexedDB, a `checkStorageForBulkWrite()` call estimates available storage and blocks the write if insufficient, showing a user-facing error message. The write itself is also wrapped in try/catch for `QuotaExceededError`.
- **Error handling on save operations (G7):** All `saveSessionMeta`, `updateSessionStep`, and parsed-file write calls are wrapped in try/catch. On failure, a toast is shown ("Couldn't save your progress") but the wizard continues to function in-memory.
- **Import-complete cleanup (G2):** When an import job reaches a terminal status, `clearAllSessions()` is called from the parent's `onTerminalStatus` callback on `ImportProgress`. A secondary safety net in `FileImporter` tracks `jobSubmittedRef` and clears sessions on reset if the primary callback missed it.

**Duplicate detection in file import:**
- **Within-batch duplicates** are detected by `validateAndDetectDuplicates()` in
  `file-importer/utils/validation-utils.ts`. The `detectDuplicates()` function checks
  `admission_number`, `upi_number`, `knec_assessment_number`, and the
  `full_name` + `date_of_birth` combination. Conflicting rows get a "duplicate" status
  with messages naming the colliding rows (e.g. "Duplicate admission number — also used
  in row 3").
- **Against-existing records** check runs once when entering the review step
  (`step-data-review.tsx`). All non-empty values are sent to
  `POST /api/v1/students/check-duplicates`. Any values that already exist in the DB
  cause the matching records to be marked with a distinguishable message
  (e.g. "Admission number ADM001 already exists for this school").

---

## 6. Backend Components

### 6.1 `internal/students/domain.go`

Key types:
```go
type ImportRow struct {
    FullName             string  `json:"full_name"`
    Gender               string  `json:"gender"`
    DateOfBirth          *string `json:"date_of_birth,omitempty"`
    UPINumber            *string `json:"upi_number,omitempty"`
    KNECAssessmentNumber *string `json:"knec_assessment_number,omitempty"`
    AdmissionNumber      *string `json:"admission_number,omitempty"`
    ClassID              string  `json:"class_id,omitempty"`
}

type ImportRequest struct {
    IDempotencyKey *string     `json:"idempotency_key,omitempty"`
    Rows           []ImportRow `json:"rows"`
}

type ImportResponse struct {
    JobID        string `json:"job_id"`
    TotalRecords int    `json:"total_records"`
    TotalChunks  int    `json:"total_chunks"`
    Status       string `json:"status"`
}
```

### 6.2 `internal/students/importer.go` (StudentImporter)

Implements `imports.Importer` interface:

- **Validate()**: Per-row schema and business-rule checks:
   - `full_name` must be non-empty
   - `gender` must be "M" or "F"
   - `date_of_birth` (if present) must be a parseable ISO date (`YYYY-MM-DD`), not in the
     future, and not older than 25 years from today — all produce `SCHEMA_VALIDATION` failures.
   - `class_id` (if present) must be a well-formed UUID — fail fast here rather than deeper
     in `ResolveReferences` or `InsertOne`. A malformed class_id produces `SCHEMA_VALIDATION`.
   - No format/pattern validation is applied to `admission_number`, `upi_number`, or
     `knec_assessment_number`; their format is unknown/variable.
- **ResolveReferences()**: Injects tenant_id, school_id, academic_term_id, academic_year_id
   from job metadata into each row. **Class existence + tenant scope check**: if `class_id`
   is present, queries `cbc_classes` to verify the class exists AND belongs to the same
   `tenant_id`/`school_id`. If it doesn't, the row is failed with `INVALID_CLASS_REFERENCE`.
- **BulkInsert()**: Returns error to force the per-row savepoint fallback.
- **InsertOne()**: Inserts student into `cbc_students`. If class_id is present, also inserts
   enrollment into `cbc_student_enrollments`. **DB constraint translation**: Postgres driver
   errors (`pgconn.PgError`) are inspected for constraint name — known constraints map to
   friendly messages with typed error types (`INVALID_CLASS_REFERENCE` for FK violations on
   `fk_enrollments_tenant_class`, `BUSINESS_RULE_VIOLATION` for
   `unique_student_term_enrollment`). Any unmapped constraint falls back to
   `DB_CONSTRAINT_VIOLATION` with a generic message — raw SQL/driver text is never leaked.

### 6.3 `internal/imports/service.go`

- **CreateJob()**: Creates import job, writes staging rows, writes chunk rows to `import_job_chunks`, splits into chunks, enqueues Asynq tasks.
  - `ChunkSize = 100` (rows per chunk)
  - `MaxImportRows = 5000` — maximum rows accepted in a single import.
    Enforced by the handler before `CreateJob()` is called. The handler
    returns HTTP 400 with code `import_row_limit_exceeded`.
  - `maxImportBodyBytes = 15 MB` — request body size cap for the import
    endpoint. Sized for 5000 rows × ~2KB/row with 50% margin. Enforced
    by a per-route middleware, not a codebase-wide limit.
- **GetJob()**: Returns current job state (for polling).
- **GetFailures()**: Returns paginated failure records.
- **CancelJob()**: Atomically transitions a job from `'processing'` to `'cancelling'`. Returns the updated job immediately — does not wait for in-flight chunks.
- **ProcessChunk()**: The worker entry point. Checks parent job status before claiming the chunk — if `'cancelling'`, marks the chunk `'cancelled'` instead of processing it. Uses `ClaimChunk` → pending-only staging rows → savepoint insert+mark → idempotent `AtomicChunkCompletion` for redelivery safety.
- **AtomicChunkCompletion()**: Idempotent — only increments job counters when transitioning the chunk from `'processing'` to `'completed'`. If the chunk is already `'completed'`, no-ops. Detects `'cancelling'` job status on the last chunk and transitions the job to `'cancelled'`.

#### One active job per school

`CreateJob()` enforces that at most one import job may be active (status `'processing'` or `'cancelling'`) per `school_id` at any time. Enforcement uses a **DB-level partial unique index** — not a check-then-insert pattern — ensuring race safety:

```sql
CREATE UNIQUE INDEX IF NOT EXISTS uq_import_jobs_one_active_per_school
    ON import_jobs (school_id)
    WHERE status IN ('processing'::import_job_status, 'cancelling'::import_job_status);
```

- `CreateJob` (repo) inserts the row with `status = 'processing'` directly (the old `'pending'` → `'processing'` transition is removed for new jobs; status is set at insert time).
- If the partial index is violated, Postgres returns SQLSTATE 23505.
- `CreateJob()` (service) catches this error and calls `GetActiveJobBySchoolID()` to retrieve the conflicting job.
- Returns `*ImportInProgressError` (HTTP 409, code `import_already_in_progress`) with the active job's ID so the frontend can redirect to its progress.
- The idempotent path (`CreateJobIdempotent`) also catches violations both at INSERT time (if the partial index fires before the idempotency-key index) and at `UpdateJobStatus` time (when transitioning from `'pending'` to `'processing'`). In all cases the same `ImportInProgressError` is returned.
- See `isUniqueConstraintViolation()` in `service.go` for the Postgres error-code check.

### 6.3b Asynq queue / concurrency configuration

Import chunk tasks are enqueued to the `"imports"` Asynq queue with a
**global concurrency cap of 3** (`internal/imports/module.go`). This means
at most 3 `imports:process_chunk` tasks can run simultaneously across all
tenants, regardless of how many import jobs are enqueued or how many chunks
they contain.

```go
asynq.Config{
    Concurrency: 3,                          // max concurrent chunk workers
    Queues: map[string]int{"imports": 10},  // single queue, weight=10
}
```

**Reasoning:** With `ChunkSize = 100`, 3 concurrent workers can process
up to 300 rows at a time — sufficient for `MaxImportRows = 5000` even
with slow per-row inserts. The cap prevents a single large import from
starving unrelated background work or other tenants' smaller imports
enqueued at the same time.

**Future expansion:** When additional queue types are added (e.g.,
`"notifications"`, `"exports"`), increase `Concurrency` and use the
`Queues` map to assign relative priorities. The `"imports"` queue weight
of 10 gives it the highest priority among all queues when multiple types
are active.

### 6.3a Chunk claim states (`import_job_chunks` table)

| Status | Meaning |
|--------|---------|
| `pending` | Chunk created, waiting for worker to pick it up |
| `processing` | Worker claimed this chunk and is processing rows |
| `cancelled` | Chunk was pending when the parent job was cancelled; skipped without processing |
| `completed` | All rows in the chunk have been processed; counters applied to the job |

The `ClaimChunk` UPDATE (status = `'processing'`) and `AtomicChunkCompletion` UPDATE (status = `'completed'`) both use `WHERE status = 'previous_state'` with `RETURNING`, ensuring at-most-once semantics. If two workers race to claim the same chunk, exactly one succeeds. If `AtomicChunkCompletion` is called twice for the same chunk, the second call no-ops. A pending chunk that encounters a `'cancelling'` parent job is transitioned to `'cancelled'` via `CancelPendingChunk`.

### 6.4 Retention policy & cleanup job

**Policy:** `import_job_staging` and `import_job_failures` rows are retained for 30 days (`RetentionDays` = 30). After a job reaches a terminal status (`completed`, `completed_with_errors`, `failed`, `cancelled`) and its `completed_at` is older than the retention window, its associated staging and failure rows become eligible for deletion.

**`import_jobs` rows are never deleted** — they are small summary records worth keeping indefinitely for audit/history (e.g. "did our import from March succeed?"). Only the heavier per-row staging/failure data is purged.

**Rules:**

1. Only terminal-status jobs are eligible. A job still `'processing'` or `'cancelling'` is never touched, regardless of its `created_at` age — even a very long-running or previously stuck job.
2. `import_jobs` (the job summary rows) are never affected by this cleanup.
3. Cleanup runs in batches of 1000 rows (`CleanupBatchSize`) using a loop — each iteration deletes up to 1000 rows and stops when 0 rows are affected, keeping individual transactions short.

**Implementation:**

- **`CleanupExpiredData(ctx)`** in `internal/imports/service.go` — the entry point. Computes the cutoff as `now - 30 days`, then calls both cleanup methods below. Logs a summary of rows deleted per run.
- **`CleanupStagingData(ctx, cutoff, batchSize)`** in `internal/imports/repository.go` — `DELETE FROM import_job_staging WHERE job_id IN (SELECT id FROM import_jobs WHERE completed_at IS NOT NULL AND completed_at < $1 AND status IN ('completed', 'completed_with_errors', 'failed', 'cancelled')) LIMIT $2`. The subquery ensures only terminal jobs past the cutoff are included.
- **`CleanupFailureData(ctx, cutoff, batchSize)`** in `internal/imports/repository.go` — same pattern for `import_job_failures`.

**Scheduling:** The cleanup runs daily at 03:00 UTC via an Asynq periodic scheduler (`CleanupScheduler` in `internal/imports/module.go`). The scheduler registers a recurring task `imports:cleanup_old_data` with the `@daily` cronspec. The worker handler is registered alongside the chunk processing handler in the same Asynq mux. Lifecycle hooks are managed by fx.

### 6.5 `internal/students/handler.go` (BulkImport)

```
POST /api/v1/students/import
```

- Resolves academic year and term from server-side (current active year/term)
- Does NOT accept `academic_term_id` from the frontend — it's resolved automatically
- **Row limit enforcement:** Before calling `CreateJob()`, validates that
  `len(request.Rows) <= MaxImportRows`. Returns HTTP 400 with code
  `import_row_limit_exceeded` if exceeded.
- **Body size limit:** A per-route `bodySizeLimit` middleware checks
  `Content-Length` header against `MaxImportBodyBytes()` (15 MB). Returns
  HTTP 413 with code `request_too_large` if exceeded.
- Creates import job via `imports.Service.CreateJob()`
- Returns `ImportResponse` immediately (async processing)

---

## 7. Key API Endpoints

| Method | Path | Purpose | Response |
|--------|------|---------|----------|
| `POST` | `/api/v1/students/import` | Create import job | `{ job_id, total_records, total_chunks, status }` |
| `POST` | `/api/v1/students/check-duplicates` | Check which values already exist in DB | `{ existing_admission_numbers, existing_upi_numbers, existing_knec_assessment_numbers }` |
| `GET` | `/api/v1/imports/:job_id` | Poll job status | `ImportJob` (full state) |
| `GET` | `/api/v1/imports/:job_id/failures` | Get failure details | `{ failures: [...], total: number }` |
| `POST` | `/api/v1/imports/:job_id/cancel` | Cancel a running import | `ImportJob` (status: `cancelling`) |
| `GET` | `/api/v1/schools/:school_id/imports/active` | Proactive check: is a job active for this school? | `{ active: true, job: ImportJob }` or `{ active: false, job: null }` |

---

## 8. Request/Response Contracts

### POST /api/v1/students/import

**Request:**
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

**Response (201 — new job created):**
```json
{
  "job_id": "uuid",
  "total_records": 100,
  "total_chunks": 1,
  "status": "processing",
  "is_replay": false
}
```

**Response (200 — idempotent replay):**
```json
{
  "job_id": "uuid",
  "total_records": 100,
  "total_chunks": 1,
  "status": "processing",
  "is_replay": true
}
```

**Response (409 — duplicate key, different payload):**
```json
{
  "code": "duplicate_import",
  "message": "A job with this idempotency key already exists."
}
```

**Response (409 — import already in progress):**
```json
{
  "code": "import_already_in_progress",
  "message": "An import job is already in progress for this school. Please wait for it to complete or cancel it.",
  "active_job_id": "uuid"
}
```

### Idempotency semantics

When an `idempotency_key` is provided, the backend guarantees **exactly-once job creation**
for the same key + payload combination. The implementation uses a DB-level
`INSERT ... ON CONFLICT (tenant_id, school_id, idempotency_key) WHERE idempotency_key IS NOT NULL
DO NOTHING RETURNING *` which provides concurrent-safe deduplication without
application-level locking.

| Scenario | HTTP Status | `is_replay` | Behavior |
|----------|-------------|-------------|----------|
| No key provided | 201 | `false` | Always creates a new job (no dedup) |
| First submission with key | 201 | `false` | Job created, staging rows written, enqueued |
| Resubmission: same key + same payload | 200 | `true` | Returns existing job, no side effects |
| Resubmission: same key + different payload | 409 | — | Error (`duplicate_import`), no job created |

#### Key generation (frontend)

- Both `manual-import-form.tsx` and `step-streaming.tsx` generate the key via
  `crypto.randomUUID()` at the start of the submit action.
- The key is stored in a `useRef` that persists across re-renders during the
  in-flight request.
- On success the ref is cleared so the next submit gets a fresh key.
- On transient network failure the key is **reused** so the retry is safe.
- The ref is automatically reset when the component unmounts (user navigates
  away or clicks Cancel/Done).

#### Payload hash

A `payload_hash` (SHA-256 of the JSON-serialized row array) is computed server-side
and stored on the `import_jobs` row. When a conflict occurs the hash is compared:
- **Match** → idempotent replay (HTTP 200)
- **Mismatch** → `duplicate_import` error (HTTP 409)

#### Concurrent safety

The `INSERT ... ON CONFLICT DO NOTHING` pattern ensures that racing requests with the
same key are serialized by the database. Only one succeeds in inserting the row;
the rest see a conflict and either replay or reject based on the hash comparison.
This eliminates TOCTOU race conditions that a check-then-insert pattern would have.

### GET /api/v1/imports/:job_id

**Response:**
```json
{
  "id": "uuid",
  "tenant_id": "uuid",
  "school_id": "uuid",
  "job_type": "STUDENT_IMPORT",
  "status": "completed_with_errors",
  "total_records": 100,
  "processed_records": 100,
  "success_count": 95,
  "failed_count": 5,
  "total_chunks": 1,
  "processed_chunks": 1,
  "created_at": "2026-07-08T12:00:00Z",
  "started_at": "2026-07-08T12:00:01Z",
  "completed_at": "2026-07-08T12:00:30Z",
  "last_progress_at": "2026-07-08T12:00:28Z"
}
```

**`last_progress_at`:** Updated by `AtomicChunkCompletion()` on every chunk completion. Null if no chunks have completed yet. Used by the frontend for stalled-job detection — if the job is still `'processing'` but `last_progress_at` is older than 2 minutes, a non-alarming inline message is shown. See [Section 5.3 — Polling backoff & stalled-job messaging](#53-importprogress-import-progresstsx).
```

### POST /api/v1/imports/:job_id/cancel

**Request:** No body required.

**Response (200 — cancellation requested):** Returns the full `ImportJob` with status `"cancelling"`.

**Response (409 — job not cancellable):**
```json
{
  "code": "job_not_cancellable",
  "message": "the job is not in a cancellable state (it may already be completed, failed, or cancelled)"
}
```

**Semantics:**
- Only a job with status `'processing'` can be cancelled.
- The endpoint returns immediately with status `'cancelling'` — it does not wait for in-flight chunks to stop.
- Pending chunks are skipped by the worker (cooperative cancellation); already-processing chunks finish normally.
- Once all chunks reach a terminal state, the job auto-transitions from `'cancelling'` to `'cancelled'`.
- Counters reflect only the work that actually completed before cancellation.

### GET /api/v1/imports/:job_id/failures

**Response:**
```json
{
  "failures": [
    {
      "row_number": 5,
      "raw_payload": { "full_name": "Bad Student", "gender": "X" },
      "error_message": "invalid gender \"X\" (must be M or F)",
      "error_type": "SCHEMA_VALIDATION"
    }
  ],
  "total": 5
}
```

### Failure types (`error_type`)

| Error Type | Source | Description |
|------------|--------|-------------|
| `SCHEMA_VALIDATION` | `Validate()` | Missing required fields (`full_name`), invalid gender, malformed `class_id` (not a UUID), unparseable/future/implausibly-old `date_of_birth` |
| `BUSINESS_RULE_VIOLATION` | `ResolveReferences()` / `InsertOne()` | Missing job metadata, duplicate enrollment within the same term (`unique_student_term_enrollment`) |
| `INVALID_CLASS_REFERENCE` | `ResolveReferences()` / `InsertOne()` | `class_id` references a class that does not exist or belongs to a different tenant/school (checked via DB lookup in `ResolveReferences` and also caught by FK `fk_enrollments_tenant_class` in `InsertOne`) |
| `DATABASE_CONSTRAINT` | `InsertOne()` | Unmapped Postgres constraint violation (fallback when the error type cannot be determined); raw SQL text is never exposed |
| `DB_CONSTRAINT_VIOLATION` | `InsertOne()` | Unmapped Postgres constraint violation with a generic message: "This record could not be saved due to a data conflict". Used when a `pgconn.PgError` is detected but the constraint name is unknown or missing |
| `DUPLICATE_ADMISSION_NUMBER` | `ResolveReferences()` | A row's `admission_number` already exists in the DB (insert-time safety net) |
| `DUPLICATE_UPI_NUMBER` | `ResolveReferences()` | A row's `upi_number` already exists in the DB (insert-time safety net) |
| `DUPLICATE_KNEC_NUMBER` | `ResolveReferences()` | A row's `knec_assessment_number` already exists in the DB (insert-time safety net) |

### GET /api/v1/schools/:school_id/imports/active

**Response (200 — active job found):**
```json
{
  "active": true,
  "job": {
    "id": "uuid",
    "tenant_id": "uuid",
    "school_id": "uuid",
    "job_type": "STUDENT_IMPORT",
    "status": "processing",
    "total_records": 100,
    "processed_records": 25,
    "success_count": 22,
    "failed_count": 3,
    "total_chunks": 1,
    "processed_chunks": 1,
    "created_at": "2026-07-08T12:00:00Z",
    "started_at": "2026-07-08T12:00:01Z"
  }
}
```

**Response (200 — no active job):**
```json
{
  "active": false,
  "job": null
}
```

### POST /api/v1/students/check-duplicates

**Request:**
```json
{
  "admission_numbers": ["ADM001", "ADM002"],
  "upi_numbers": ["UPI12345"],
  "knec_assessment_numbers": ["KNEC67890"]
}
```
All three arrays are optional. Only provided values are checked.

**Response (200):**
```json
{
  "existing_admission_numbers": ["ADM001"],
  "existing_upi_numbers": [],
  "existing_knec_assessment_numbers": []
}
```
Returns only values that already exist for the caller's tenant/school. Empty arrays
when no conflicts are found.

---

## 9. Error Handling

### Backend error response shape
Every non-2xx response follows the canonical contract:
```json
{
  "code": "snake_case_error_code",
  "message": "human readable message",
  "errors": { "field_name": ["specific error"] }
}
```

### Common error codes from POST /students/import
| Code | Status | Meaning |
|------|--------|---------|
| `invalid_input` | 400 | Malformed body or validation failure |
| `import_row_limit_exceeded` | 400 | Row count exceeds `MaxImportRows` (5000). Message includes actual and max counts. |
| `no_active_academic_year` | 400 | School has no current academic year set |
| `no_active_academic_term` | 400 | No active term in the current academic year |
| `request_too_large` | 413 | Request body exceeds `MaxImportBodyBytes` (15 MB). Scoped to the import endpoint via per-route middleware. |
| `duplicate_import` | 409 | Idempotency key reused with a different payload. The same key was used for a different set of rows. |
| `import_already_in_progress` | 409 | Another import job (status `'processing'` or `'cancelling'`) is already running for this school. The response body includes `active_job_id` so the frontend can redirect to the existing job's progress instead of showing a dead-end error. |

### Per-row failure types (`import_job_failures.error_type`)

The following `error_type` values are used in import failure records returned by
`GET /api/v1/imports/:job_id/failures`:

| Error Type | Generated By | Typical Scenario |
|------------|-------------|------------------|
| `SCHEMA_VALIDATION` | `Validate()` | Missing `full_name`, invalid `gender`, malformed `class_id` (not UUID), future/unparseable `date_of_birth` |
| `INVALID_CLASS_REFERENCE` | `ResolveReferences()` / `InsertOne()` | `class_id` references a class that does not exist or belongs to a different school |
| `BUSINESS_RULE_VIOLATION` | `ResolveReferences()` / `InsertOne()` | Missing job metadata, duplicate enrollment for the same student+term |
| `DATABASE_CONSTRAINT` | `InsertOne()` | Legacy fallback — used when an InsertOne error is not a typed `ImportError` |
| `DB_CONSTRAINT_VIOLATION` | `InsertOne()` | Unmapped Postgres constraint violation (generic fallback) |
| `DUPLICATE_ADMISSION_NUMBER` | `ResolveReferences()` | A row's `admission_number` already exists in the DB for this school |
| `DUPLICATE_UPI_NUMBER` | `ResolveReferences()` | A row's `upi_number` already exists in the DB for this school |
| `DUPLICATE_KNEC_NUMBER` | `ResolveReferences()` | A row's `knec_assessment_number` already exists in the DB for this school |

> **Never leak raw SQL/driver text.** All constraint errors are translated into
> friendly messages before being stored in `import_job_failures.error_message`.

### Frontend error handling
- Submit failures → `toast.error()`
- Poll errors → silently retry on next tick (transient network issues)
- Failures fetch errors → logged to console, result banner still shows counts
- All API errors go through `ApiError` class in `src/lib/api/client.ts`

---

## 10. Important Rules for Agents

### NEVER chunk on the frontend
The frontend must NEVER split data into batches. Send ALL rows in a single `POST /api/v1/students/import`. The backend handles chunking internally via Asynq workers.

### Gender normalization
When importing from a file, gender values must be normalized to "M" or "F":
- "Male", "male", "m", "boy", "masculine" → "M"
- "Female", "female", "f", "girl", "feminine" → "F"
- Anything unrecognized → pass through (caught by validation)

### Progress is always async
The `POST /import` endpoint returns immediately with `status: "processing"`. Never treat the response as final. Always poll `GET /imports/:job_id` to track completion.

### Shared progress component
Both import paths use the same `<ImportProgress>` component. Don't duplicate polling/progress logic. The parent `StudentsImportForm` manages the active job state.

### Class ID is passed directly
The frontend sends `class_id` directly in `ImportRow`. The backend does NOT resolve grade_level + stream_name. Class selection happens on the frontend via `ClassCombobox` (manual) or `StepClassResolve` (file).

### Academic term is server-resolved
The frontend does NOT send `academic_term_id`. The backend resolves the current active year and term from the school's configuration. If no active year/term is set, the import returns an error.

### Idempotency — idempotency_key and payload_hash

The import endpoint supports optional idempotency for safe retry semantics.
See [Section 8 — Request/Response Contracts](#8-requestresponse-contracts) for
the full specification. Key points:

- **Frontend generates the key** with `crypto.randomUUID()` at submit start, not before.
- **Backend stores the key** on `import_jobs.idempotency_key` with a unique
  partial index on `(tenant_id, school_id, idempotency_key) WHERE idempotency_key IS NOT NULL`.
- **Payload hash** (`import_jobs.payload_hash`) is a SHA-256 digest of the
  JSON-serialized row array, compared on conflict to distinguish replay vs error.
- **200 vs 201 vs 409:** see the table in Section 8.
- **Concurrent safety:** guaranteed by `INSERT ... ON CONFLICT DO NOTHING`, not
  by application-level locking.

### Key files to reference

### Key files to reference

**Frontend:**
- `src/lib/api/imports.ts` — API client functions
- `src/features/students/components/students-import/students-import.tsx` — Parent orchestrator
- `src/features/students/components/students-import/import-progress.tsx` — Shared progress component
- `src/features/students/components/students-import/manual-import-form.tsx` — Manual entry form
- `src/features/students/components/students-import/file-importer/file-importer.tsx` — File import wizard orchestrator
- `src/features/students/components/students-import/file-importer/step-upload.tsx` — File upload & parsing
- `src/features/students/components/students-import/file-importer/step-column-mapping.tsx` — Column mapping UI
- `src/features/students/components/students-import/file-importer/step-class-resolve.tsx` — Class name resolution
- `src/features/students/components/students-import/file-importer/step-data-review.tsx` — Record review/correction
- `src/features/students/components/students-import/file-importer/step-streaming.tsx` — Submit step
- `src/features/students/components/students-import/file-importer/db.ts` — IndexedDB persistence layer
- `src/features/students/components/students-import/file-importer/types.ts` — All wizard types & constants
- `src/features/students/components/students-import/file-importer/utils/parse-utils.ts` — CSV/Excel parsing engine
- `src/features/students/components/students-import/file-importer/utils/validation-utils.ts` — Validation + duplicate detection
- `src/features/students/components/students-import/file-importer/utils/class-resolver-utils.ts` — Class resolution helpers

**Backend:**
- `internal/students/handler.go` — BulkImport handler + route registration
- `internal/students/domain.go` — ImportRow, ImportRequest, ImportResponse types
- `internal/students/importer.go` — StudentImporter (Validate, ResolveReferences, InsertOne)
- `internal/students/repository.go` — Student-specific DB queries (class lookup, duplicate check)
- `internal/imports/service.go` — Import job lifecycle (CreateJob, ProcessChunk, CancelJob, CleanupExpiredData)
- `internal/imports/domain.go` — Job, Chunk, RowFailure, ValidatedRow, Importer interface
- `internal/imports/repository.go` — PgRepository (all import_job* CRUD operations)
- `internal/imports/handler.go` — Job status + failures + cancel HTTP endpoints
- `internal/imports/module.go` — fx module, Asynq server/worker, cleanup scheduler
- `internal/database/migrations/000001_initial_schema.up.sql` — DDL for all import_* tables

---

## 11. Database Schema

### 11.1 `import_jobs` — Job summary

| Column | Type | Constraints | Notes |
|--------|------|-------------|-------|
| `id` | `UUID` | `PK DEFAULT gen_random_uuid()` | |
| `tenant_id` | `UUID` | `NOT NULL → tenants(id) ON DELETE CASCADE` | |
| `school_id` | `UUID` | `NOT NULL` | FK via composite `fk_import_jobs_tenant_school` |
| `job_type` | `import_job_type` | `NOT NULL` | `STUDENT_IMPORT` or `STAFF_INVITE` |
| `role` | `user_role` | `NULL` | Required when `job_type = 'STAFF_INVITE'` (CHECK constraint) |
| `created_by` | `UUID` | `→ users(id) ON DELETE SET NULL` | |
| `status` | `import_job_status` | `NOT NULL DEFAULT 'pending'` | See [status enum](#job-status-enum) |
| `total_records` | `INT` | `NOT NULL DEFAULT 0` | |
| `processed_records` | `INT` | `NOT NULL DEFAULT 0` | |
| `success_count` | `INT` | `NOT NULL DEFAULT 0` | |
| `failed_count` | `INT` | `NOT NULL DEFAULT 0` | |
| `idempotency_key` | `TEXT` | `NULL` | Partial unique index |
| `payload_hash` | `TEXT` | `NULL` | SHA-256 of JSON-serialized rows |
| `total_chunks` | `INT` | `NOT NULL DEFAULT 0` | |
| `processed_chunks` | `INT` | `NOT NULL DEFAULT 0` | |
| `metadata` | `JSONB` | `NOT NULL DEFAULT '{}'` | Contains `academic_term_id`, `academic_year_id` |
| `created_at` | `TIMESTAMPTZ` | `NOT NULL DEFAULT NOW()` | |
| `started_at` | `TIMESTAMPTZ` | `NULL` | Set when first chunk is processed |
| `completed_at` | `TIMESTAMPTZ` | `NULL` | Set when `processed_chunks = total_chunks` |
| `last_progress_at` | `TIMESTAMPTZ` | `NULL` | Updated by every `AtomicChunkCompletion()` |

**Indexes:**

```sql
CREATE INDEX idx_import_jobs_tenant_id  ON import_jobs (tenant_id);
CREATE INDEX idx_import_jobs_school_id  ON import_jobs (school_id);
CREATE INDEX idx_import_jobs_created_by ON import_jobs (created_by);
CREATE INDEX idx_import_jobs_status     ON import_jobs (status);

-- Idempotency dedup: one key per tenant
CREATE UNIQUE INDEX uq_import_jobs_tenant_idempotency
    ON import_jobs (tenant_id, idempotency_key)
    WHERE idempotency_key IS NOT NULL;

-- One active job per school at a time
CREATE UNIQUE INDEX uq_import_jobs_one_active_per_school
    ON import_jobs (school_id)
    WHERE status IN ('processing', 'cancelling');
```

#### Job status enum (`import_job_status`)

```sql
CREATE TYPE import_job_status AS ENUM (
    'pending',              -- Created, staging rows written, Asynq tasks enqueued
    'processing',           -- First chunk has started
    'completed',            -- All chunks processed, 0 failures
    'completed_with_errors',-- All chunks processed, ≥1 failures
    'failed',               -- Terminal error (not used by student import currently)
    'cancelled',            -- User requested cancellation, all chunks finished
    'cancelling'            -- User requested cancellation, some chunks still in-flight
);
```

#### Job type enum (`import_job_type`)

```sql
CREATE TYPE import_job_type AS ENUM ('STAFF_INVITE', 'STUDENT_IMPORT');
```

---

### 11.2 `import_job_staging` — Per-row staging data

| Column | Type | Constraints | Notes |
|--------|------|-------------|-------|
| `id` | `UUID` | `PK DEFAULT gen_random_uuid()` | Referenced by `cbc_students.staging_row_id` |
| `job_id` | `UUID` | `NOT NULL → import_jobs(id) ON DELETE CASCADE` | |
| `tenant_id` | `UUID` | `NOT NULL` | Denormalized for query simplicity |
| `school_id` | `UUID` | `NOT NULL` | Denormalized |
| `row_number` | `INT` | `NOT NULL` | 0-indexed position in the original request |
| `raw_data` | `JSONB` | `NOT NULL` | Original serialized `ImportRow` |
| `status` | `import_staging_status` | `NOT NULL DEFAULT 'pending'` | `pending | succeeded | failed` |
| `processed_at` | `TIMESTAMPTZ` | `NULL` | Set when status changes from `pending` |
| `created_at` | `TIMESTAMPTZ` | `NOT NULL DEFAULT NOW()` | |

**Indexes:**

```sql
CREATE INDEX idx_import_job_staging_job_id ON import_job_staging (job_id);
CREATE UNIQUE INDEX uq_import_job_staging_job_row ON import_job_staging (job_id, row_number);
```

#### Staging row state machine

```
          ┌──────────────┐
          │   pending    │
          └──────┬───────┘
                 │
       ┌─────────┴──────────┐
       │                    │
       ▼                    ▼
┌──────────────┐   ┌──────────────┐
│  succeeded   │   │   failed     │
└──────────────┘   └──────────────┘
```

- **`pending`** — Row is queued, waiting for a chunk worker.
- **`succeeded`** — Row was successfully inserted into `cbc_students` (and optionally `cbc_student_enrollments`). Set atomically within the same savepoint as the insert — there is no code path where the insert commits but the staging row stays `pending`.
- **`failed`** — Row failed validation, reference resolution, or insert. Error details are stored in `import_job_failures`.

**Redelivery safety:** The worker only loads staging rows where `status = 'pending'`. Rows already marked `succeeded` or `failed` by a prior partial attempt (worker crash mid-chunk) are skipped — they are never reprocessed.

---

### 11.3 `import_job_chunks` — Chunk claim tracking

| Column | Type | Constraints | Notes |
|--------|------|-------------|-------|
| `id` | `UUID` | `PK DEFAULT gen_random_uuid()` | |
| `job_id` | `UUID` | `NOT NULL → import_jobs(id) ON DELETE CASCADE` | |
| `chunk_index` | `INT` | `NOT NULL` | 0-based index within the job |
| `status` | `import_chunk_status` | `NOT NULL DEFAULT 'pending'` | See [chunk status states](#chunk-status-states) |
| `row_start` | `INT` | `NOT NULL DEFAULT 0` | Inclusive staging row number |
| `row_end` | `INT` | `NOT NULL DEFAULT 0` | Exclusive staging row number |
| `claimed_at` | `TIMESTAMPTZ` | `NULL` | Set when worker claims the chunk |
| `completed_at` | `TIMESTAMPTZ` | `NULL` | Set when chunk finishes processing |
| `created_at` | `TIMESTAMPTZ` | `NOT NULL DEFAULT NOW()` | |

**Constraints:**

```sql
CONSTRAINT uq_import_job_chunks_job_chunk UNIQUE (job_id, chunk_index)
```

**Indexes:**

```sql
CREATE INDEX idx_import_job_chunks_job_id  ON import_job_chunks (job_id);
CREATE INDEX idx_import_job_chunks_status ON import_job_chunks (job_id, status);
```

#### Chunk status states

| Status | Meaning |
|--------|---------|
| `pending` | Chunk created, waiting for worker to pick it up |
| `processing` | Worker claimed this chunk and is processing rows |
| `completed` | All rows in the chunk have been processed; counters applied to the job |
| `cancelled` | Chunk was pending when the parent job was cancelled; skipped without processing |

**At-most-once guarantee:** Both `ClaimChunk` (`pending → processing`) and `AtomicChunkCompletion` (`processing → completed`) use `UPDATE ... WHERE status = 'previous_state'` with `RETURNING`. If two workers race to claim the same chunk, exactly one succeeds. If `AtomicChunkCompletion` is called twice for the same chunk, the second call no-ops.

---

### 11.4 `import_job_failures` — Per-row failure records

| Column | Type | Constraints | Notes |
|--------|------|-------------|-------|
| `id` | `BIGSERIAL` | `PK` | |
| `import_job_id` | `UUID` | `NOT NULL → import_jobs(id) ON DELETE CASCADE` | |
| `raw_payload` | `JSONB` | `NOT NULL` | The original row data that failed |
| `error_message` | `TEXT` | `NOT NULL` | Human-readable description |
| `error_type` | `import_failure_type` | `NOT NULL DEFAULT 'DATABASE_CONSTRAINT'` | Categorised failure type |
| `row_number` | `INT` | `NULL` | 0-indexed position in the original request |
| `created_at` | `TIMESTAMPTZ` | `NOT NULL DEFAULT NOW()` | |

**Indexes:**

```sql
CREATE INDEX idx_import_job_failures_job_id ON import_job_failures (import_job_id);
```

#### Failure type enum (`import_failure_type`)

```sql
CREATE TYPE import_failure_type AS ENUM (
    'SCHEMA_VALIDATION',
    'DATABASE_CONSTRAINT',
    'BUSINESS_RULE_VIOLATION'
);
```

The application extends this via typed `ImportError` values (see [Section 9 — Error Handling](#9-error-handling) for the full list of frontend-visible failure types). The enum covers the three DB-level categories; application code may emit any of the extended types listed in [Section 9](#9-error-handling).

---

### 11.5 `cbc_students.staging_row_id` — Idempotent insert anchor

The `cbc_students` table includes a `staging_row_id` column used for defense-in-depth against duplicate inserts:

```sql
staging_row_id UUID REFERENCES import_job_staging(id) ON DELETE SET NULL
```

**Unique partial index:**

```sql
CREATE UNIQUE INDEX idx_cbc_students_school_staging_row
    ON cbc_students (school_id, staging_row_id)
    WHERE staging_row_id IS NOT NULL;
```

This means `INSERT INTO cbc_students (school_id, staging_row_id, ...) VALUES ($1, $2, ...) ON CONFLICT (school_id, staging_row_id) WHERE staging_row_id IS NOT NULL DO UPDATE SET staging_row_id = EXCLUDED.staging_row_id` is a no-op identity update — if the row was already inserted by a prior attempt, it is treated as success, not a duplicate error.

---

## 12. Importer Interface & Registry

### 12.1 `Importer` interface contract

The import engine is generic — it imports any domain type. The domain package (e.g., `students`) implements this interface and registers itself at startup.

```go
// Importer interface — implemented by domain packages (students, etc.)
type Importer interface {
    // JobType returns the import_job_type this importer handles.
    JobType() ImportJobType

    // Validate checks each raw row for schema-level correctness.
    // Returns validated rows and any rows that failed validation.
    Validate(ctx context.Context, tenantID, schoolID uuid.UUID, raw []json.RawMessage) ([]ValidatedRow, []RowFailure)

    // ResolveReferences enriches the validated rows with any cross-table
    // references that require DB lookups.
    // metadata contains the job-level metadata (e.g., academic_term_id).
    // Returns resolved rows and any rows that failed resolution.
    // The engine calls this after Validate and before BulkInsert.
    ResolveReferences(ctx context.Context, tenantID, schoolID uuid.UUID, metadata json.RawMessage, rows []ValidatedRow) ([]ValidatedRow, []RowFailure)

    // BulkInsert attempts to insert all validated & resolved rows in one multi-row INSERT.
    // Returns the number of rows inserted and any error.
    BulkInsert(ctx context.Context, tx pgx.Tx, rows []ValidatedRow) (inserted int, err error)

    // InsertOne inserts a single validated & resolved row inside a savepoint.
    InsertOne(ctx context.Context, tx pgx.Tx, row ValidatedRow) error
}
```

**Execution order inside `ProcessChunk()`:**

1. `Validate()` — Schema checks (required fields, format, ranges)
2. `ResolveReferences()` — Cross-table lookups (class existence, duplicate detection)
3. `BulkInsert()` — Optimistic multi-row INSERT (expected to fail for per-row FK/savepoint handling)
4. `InsertOne()` — Per-row savepoint fallback (called by the engine, not directly by domain code)

### 12.2 ImporterRegistry pattern

Domain packages register their importer at init time via a global registry:

```go
// ImporterRegistry holds all registered Importers, keyed by ImportJobType.
var ImporterRegistry = map[ImportJobType]Importer{}

// RegisterImporter registers an Importer for its JobType.
// Panics if a duplicate JobType is registered.
func RegisterImporter(imp Importer) {
    jt := imp.JobType()
    if _, exists := ImporterRegistry[jt]; exists {
        panic(fmt.Sprintf("imports: duplicate importer registration for job type %q", jt))
    }
    ImporterRegistry[jt] = imp
}
```

**Registration flow:**

1. `internal/students/module.go` (or equivalent for each domain) calls `imports.RegisterImporter(NewStudentImporter(repo))` in its `Provide` or `Invoke`.
2. `ImporterRegistry` is populated before any import jobs are created.
3. `ProcessChunk()` looks up `ImporterRegistry[job.JobType]` to find the right handler.

### 12.3 `StudentImporter` — Student-specific implementation

Located in `internal/students/importer.go`:

- **`Validate()`** — Per-row checks:
  - `full_name` must be non-empty
  - `gender` must be `"M"` or `"F"` (already normalized by frontend)
  - `date_of_birth` (if present): parseable ISO date, not in the future, ≤25 years old
  - `class_id` (if present): must be a well-formed UUID (fail fast)
  - No format validation on `admission_number`, `upi_number`, or `knec_assessment_number`

- **`ResolveReferences()`**:
  - Injects `tenant_id`, `school_id`, `academic_term_id`, `academic_year_id` from job metadata
  - **Class existence check:** if `class_id` is present, queries `cbc_classes` to verify the class exists AND belongs to the same `tenant_id`/`school_id`. Otherwise, fails with `INVALID_CLASS_REFERENCE`.
  - Converts the row to `augmentedImportRow` (includes all context fields for direct insert)

- **`BulkInsert()`** — Returns an error unconditionally to force the per-row savepoint fallback (the current implementation needs per-row FK/savepoint handling because of student + enrollment two-table insert).

- **`InsertOne()`** — Within a savepoint:
  1. Inserts into `cbc_students` with `staging_row_id` (ON CONFLICT identity-update for idempotency)
  2. If `class_id` is present, inserts into `cbc_student_enrollments`
  3. Translates Postgres constraint violations to typed `*ImportError` values
     - `fk_enrollments_tenant_class` → `ImportFailureInvalidClassReference`
     - `unique_student_term_enrollment` → `ImportFailureBusinessRule`
     - Unmapped constraints → `ImportFailureDBConstraintViolation` (generic, no SQL leak)

---

## 13. File Parsing Pipeline

All file parsing lives in `step-upload.tsx` → `parse-utils.ts`. The pipeline:

### 13.1 File type detection

```typescript
function detectFileType(file: File): FileType {
    const name = file.name.toLowerCase();
    const ext = name.split(".").pop() ?? "";
    if (["xlsx", "xls", "xlsm"].includes(ext)) return "excel";
    if (ext === "ods") return "ods";
    if (ext === "tsv" || ext === "tab") return "tsv";
    return "csv"; // default
}
```

### 13.2 CSV/TSV parsing

- **Library:** `papaparse`
- **Mode:** `header: true` (first row → column names), `dynamicTyping: false` (all values as strings)
- **BOM stripping:** UTF-8 BOM (`0xFEFF`) is stripped before parsing
- **Skip empty lines:** Enabled
- **Delimiter:** Auto-detected for CSV; `\t` for TSV; caller can override
- **Filtering:** Rows where every cell is empty after trimming are excluded

### 13.3 Excel/ODS parsing

- **Library:** `xlsx` (SheetJS)
- **Mode:** `raw: true` (get raw values — numbers for dates), `cellDates: false` (manual date conversion)
- **Sheet selection:** Picks the first non-empty sheet by default; user can pick from `availableSheets`
- **Serial date conversion:** Excel serial dates (numbers like `44123`) are converted to ISO date strings via:
  ```typescript
  function excelSerialToDate(serial: number): string | null {
      const ms = (serial - 1) * 86400000;
      const date = new Date(Date.UTC(1899, 11, 31) + ms);
      return date.toISOString().split("T")[0];
  }
  ```
- **Text preservation for UPI/KNEC columns:** Columns whose headers contain "upi", "knec", or "assessment" are forced to string values to avoid losing leading zeros from numeric auto-conversion.

### 13.4 Size limits and streaming threshold

| Constant | Value | Purpose |
|----------|-------|---------|
| `MAX_FILE_SIZE_BYTES` | 15 MB | Hard cap on file size before any parsing |
| `STREAMING_THRESHOLD` | 1,000 rows | Files above this show "large file" warning (main thread may block) |
| `MAX_IMPORT_ROWS` | 5,000 | Syncs with backend `MaxImportRows`; blocks progression if exceeded |
| `MAX_PERSISTED_ROWS` | 500 | Max rows persisted to IndexedDB for crash recovery |

### 13.5 Gender normalization

Located in `file-importer.tsx`. The `normalizeGender()` function maps common variants:

```typescript
function normalizeGender(raw: string | undefined | null): string | undefined {
    if (!raw) return undefined;
    const lower = raw.trim().toLowerCase();
    if (["m", "male", "boy", "masculine"].includes(lower)) return "M";
    if (["f", "female", "girl", "feminine"].includes(lower)) return "F";
    return raw.trim() || undefined;  // pass through unrecognized (caught by validation)
}
```

Unrecognized values pass through and are caught by backend validation. `undefined`/`null`/empty values leave the field unset.

### 13.6 Smart column matching

Defined in `types.ts`. When a file is uploaded, the `FileImporter` attempts to auto-map columns using a smart-match dictionary:

```typescript
export const SMART_MATCH_DICT: Record<string, string[]> = {
    full_name: ["full name", "name", "jina kamili", "mwanafunzi", "student name", "student", "names", "jina", "learner name", "learner"],
    gender: ["gender", "jinsia", "sex", "jeni"],
    date_of_birth: ["dob", "date of birth", "tarehe ya kuzaliwa", "birth date", "birthday"],
    upi_number: ["upi", "unique identifier", "nambari ya usajili", "upi number", "upi no", "upi#", "unique pupil identifier"],
    knec_assessment_number: ["knec assessment number", "knec number", "knec no", "assessment number", "knec#", "knec", "nambari ya kn", "nambari ya mtihani"],
    class_id: ["class", "stream", "grade", "class/stream", "darasa", "grade level", "level", "form", "class name", "stream name", "daraja", "kidato"],
};
```

Matching is case-insensitive. The column mapping step (`step-column-mapping.tsx`) presents unmatched columns for manual assignment.

---

## 14. Duplicate Detection Matrix

Duplicates are detected at **three separate points** in the pipeline, each with a different scope and timing:

| Phase | Scope | Detection Point | Method | Fields Checked |
|-------|-------|-----------------|--------|----------------|
| **Within-file** | Current batch only | On render (file) / On submit (manual) | `detectDuplicates()` in `validation-utils.ts` | `admission_number`, `upi_number`, `knec_assessment_number`, `full_name`+`date_of_birth` combination |
| **Against DB** | All records in the school | On submit click (manual) / On entering review step (file) | `POST /api/v1/students/check-duplicates` → backend DB query | Same 4 fields |
| **Insert-time safety net** | All records in the school | During `ResolveReferences()` | DB query for existing values + typed `ImportError` | `admission_number`, `upi_number`, `knec_assessment_number` |

**Client-side duplicate messages:**
- "Duplicate admission number — also used in row 3"
- "Duplicate UPI number — also used in row 3, 7"
- "Duplicate name + date of birth — also used in row 5"

**DB-existing messages:**
- "Admission number ADM001 already exists for this school"
- "UPI number UPI12345 already exists for this school"
- "KNEC number KNEC67890 already exists for this school"

**No format/pattern validation** is applied to `admission_number`, `upi_number`, or `knec_assessment_number` — their format is unknown/variable.

---

## 15. Frontend Validation Rules

Located in `validation-utils.ts`. These run on the client before submission:

### 15.1 `full_name`

| Rule | Value | Action |
|------|-------|--------|
| Required | Non-empty after trim | Hard error |
| Min length | 2 characters | Hard error if < 2 |
| Max length | 100 characters | Hard error if > 100 |
| Character set | Unicode letters, spaces, hyphens, apostrophes (`/^[\p{L}\p{M}'\-\s]+$/u`) | Hard error if contains digits or special characters |

### 15.2 `date_of_birth`

| Rule | Value | Action |
|------|-------|--------|
| Format | ISO date parseable by `new Date(s)` | Hard error if unparseable |
| Future dates | Must be ≤ today | Hard error if future |
| Too young | < 3 years old | Warning (not block) |
| Too old | > 20 years old | Warning (not block) — "please verify" |

### 15.3 `gender`

| Value | Normalised |
|-------|------------|
| `M`, `Male`, `male`, `m`, `boy`, `masculine` | `M` |
| `F`, `Female`, `female`, `f`, `girl`, `feminine` | `F` |
| Anything else (still unrecognized after normalization) | Hard error if non-empty: "Gender must be \"M\" or \"F\"" |

---

## 16. IndexedDB Store Schema

The wizard persists state to IndexedDB via `db.ts` using the `idb` library. Three object stores in database `somo_student_import` (version 3):

### 16.1 `import_meta` — Session metadata

| Key | Value |
|-----|-------|
| `id` | `session:<school_id>` (keyPath) |
| `current_step` | `WizardStep` — current wizard step |
| `file_name` | Original uploaded file name |
| `source_sheet_name` | Selected sheet name for multi-sheet workbooks |
| `total_rows` | Total parsed row count |
| `column_mappings` | `Record<string, string \| string[]>` — file column → target field |
| `class_mappings` | `Record<string, string>` — raw class name → resolved class UUID |
| `updated_at` | ISO timestamp of last update |
| `school_id` | School UUID (scoping key) |
| `parsed_file_too_large` | `boolean` — set if file exceeded `MAX_PERSISTED_ROWS` |

Saved after every step transition. Auto-restored on mount if < 24 hours old.

### 16.2 `student_import_staging` — Staged student records

| Key | Value |
|-----|-------|
| `id` | Auto-increment number (keyPath) |
| `school_id` | School UUID |
| `payload` | `CreateStudentPayload` — the structured import row |
| `raw_row_data` | `Record<string, string>` — original file row |
| `status` | `"valid" | "error" | "duplicate" | "submitted"` |
| `errors` | `string[]` — validation/duplicate error messages |

### 16.3 `parsed_file` — Raw parsed file data (crash recovery)

| Key | Value |
|-----|-------|
| `id` | `parsed_file:<school_id>` (keyPath) |
| `file_name` | Original file name |
| `sheet_name` | Selected sheet name |
| `headers` | `string[]` — column headers |
| `rows` | `Record<string, string>[]` — all parsed rows (up to 500) |
| `total_rows` | Total row count |

Persisted only for files ≤ 500 rows (`MAX_PERSISTED_ROWS`). Larger files set `parsed_file_too_large` on the session meta instead.

### Session lifecycle summary

1. **Saved** after every step transition (upload→mapping→class→review→streaming)
2. **Auto-restored** on mount if session exists for current `school_id` and age < 24h
3. **Stale prompt** shown if session age > 24h (user can choose to resume or discard)
4. **Foreign-school detection** — session for different `school_id` shows discard prompt
5. **Cleared** on any terminal import status via `onTerminalStatus` callback
6. **Size guard** — `checkStorageForBulkWrite()` pre-checks available IndexedDB quota before writing

---

## 17. Environment Configuration Reference

All tunable constants, categorised by layer:

### Backend constants (`internal/imports/service.go`)

| Constant | Value | Location | Purpose |
|----------|-------|----------|---------|
| `ChunkSize` | `100` | `service.go` | Rows per Asynq chunk task |
| `MaxImportRows` | `5000` | `service.go` | Maximum rows in a single import request |
| `RetentionDays` | `30` | `service.go` | Days after terminal status before staging/failure rows are eligible for cleanup |
| `CleanupBatchSize` | `1000` | `service.go` | Rows per DELETE batch during retention cleanup |
| `MaxImportBodyBytes` | `15 MB` (returned by `MaxImportBodyBytes()`) | `domain.go` | Per-route request body size limit for import endpoint |
| `maxStudentAgeYears` | `25` | `students/importer.go` | Upper bound for student age validation (date_of_birth) |

### Asynq configuration (`internal/imports/module.go`)

| Parameter | Value | Purpose |
|-----------|-------|---------|
| Concurrency | `3` | Max concurrent chunk workers across all tenants |
| Queue | `"imports"` with weight `10` | Single queue; weight 10 = highest priority among all queue types |
| MaxRetry | `3` | Asynq max retries per chunk task |
| Unique TTL | `24h` | Asynq unique task TTL (prevents double-enqueue of same chunk) |
| Cleanup schedule | `@daily` (03:00 UTC) | Periodic retention cleanup |

### Frontend constants (`types.ts`, `parse-utils.ts`)

| Constant | Value | Location | Purpose |
|----------|-------|----------|---------|
| `MAX_IMPORT_ROWS` | `5000` | `step-upload.tsx` | Syncs with backend MaxImportRows; blocks progression if exceeded |
| `MAX_PERSISTED_ROWS` | `500` | `types.ts` | Max parsed file rows persisted to IndexedDB |
| `BYTES_PER_PERSISTED_ROW` | `2048` | `types.ts` | ~2 KB per row for storage estimation |
| `SESSION_STALE_MS` | `24h` | `types.ts` | Session staleness threshold |
| `MAX_FILE_SIZE_BYTES` | `15 MB` | `parse-utils.ts` | Hard file size cap before parsing |
| `STREAMING_THRESHOLD` | `1,000` | `parse-utils.ts` | Row count threshold for "large file" warning |
| `FULL_NAME_MIN` | `2` | `validation-utils.ts` | Minimum full_name length |
| `FULL_NAME_MAX` | `100` | `validation-utils.ts` | Maximum full_name length |
| `MIN_AGE_YEARS` | `3` | `validation-utils.ts` | Minimum student age (warning threshold) |
| `WARNING_AGE_YEARS` | `20` | `validation-utils.ts` | Maximum credible student age (warning threshold) |

---

## 18. Asynq Module Wiring (fx lifecycle)

The import engine is wired into the application via `go.uber.org/fx` in `internal/imports/module.go`:

```go
var Module = fx.Module("imports",
    fx.Provide(
        fx.Annotate(NewRepository, fx.As(new(ServiceRepository))),
        NewAsynqClient,
        NewAsynqServer,
        NewCleanupScheduler,
        NewService,
        NewHandler,
        NewWorker,
    ),
)
```

### Component responsibilities

| Component | Role |
|-----------|------|
| `NewRepository` | Creates `PgRepository` (implements `ServiceRepository`) |
| `NewAsynqClient` | Creates `*asynq.Client` for enqueuing chunk tasks |
| `NewAsynqServer` | Creates `*asynq.Server` with concurrency=3, imports queue weight=10 |
| `NewCleanupScheduler` | Registers `imports:cleanup_old_data` task on `@daily` schedule |
| `NewService` | Creates the import job lifecycle service |
| `NewHandler` | HTTP handler for job status/failures/cancel endpoints |
| `NewWorker` | Wraps `*asynq.Server` with task handlers + fx lifecycle hooks |

### Worker lifecycle

The `Worker` has `Start()` and `Stop()` methods registered via `fx.Lifecycle`:

```go
func RegisterWorkerHooks(lc fx.Lifecycle, worker *Worker) {
    lc.Append(fx.Hook{
        OnStart: worker.Start,  // w.server.Start(w.mux)
        OnStop:  worker.Stop,   // w.server.Shutdown()
    })
}
```

### Task handlers (mux registration)

```go
mux := asynq.NewServeMux()
mux.HandleFunc("imports:process_chunk", func(ctx context.Context, t *asynq.Task) error {
    var payload ChunkTaskPayload
    json.Unmarshal(t.Payload(), &payload)
    return svc.ProcessChunk(ctx, payload)
})
mux.HandleFunc("imports:cleanup_old_data", func(ctx context.Context, t *asynq.Task) error {
    return svc.CleanupExpiredData(ctx)
})
```

### Task payload (ChunkTaskPayload)

```go
type ChunkTaskPayload struct {
    JobID          string `json:"job_id"`
    ChunkIndex     int    `json:"chunk_index"`
    RowNumberStart int    `json:"row_number_start"`
    RowNumberEnd   int    `json:"row_number_end"`
}
```

### Enqueue flow (deterministic task IDs)

```go
func (s *Service) enqueueChunks(ctx context.Context, jobID uuid.UUID, totalChunks int) error {
    for i := 0; i < totalChunks; i++ {
        payload := ChunkTaskPayload{
            JobID:          jobID.String(),
            ChunkIndex:     i,
            RowNumberStart: i * ChunkSize,
            RowNumberEnd:   (i * ChunkSize) + ChunkSize,
        }
        task := asynq.NewTask("imports:process_chunk", toBytes(payload),
            asynq.Queue("imports"),
            asynq.MaxRetry(3),
            asynq.Unique(24*time.Hour),
        )
        // Deterministic task ID for idempotent redelivery
        taskID := fmt.Sprintf("import:%s:chunk:%d", jobID, i)
        s.asynq.Enqueue(task, asynq.TaskID(taskID))
    }
}
```

---

## 19. Testing Strategy

### Unit tests
- **`Service.ProcessChunk`** — Mock `ServiceRepository` to verify chunk claiming, row filtering, savepoint fallback, and counter increments
- **`StudentImporter.Validate`** — Test invalid inputs (missing name, bad gender, future DOB) return correct `RowFailure` types
- **`StudentImporter.ResolveReferences`** — Mock class lookup to test `INVALID_CLASS_REFERENCE` path
- **`insertWithSavepoints`** — Verify that savepoint rollback does not affect other rows, and `MarkStagingRowSucceeded` is called atomically

### Integration tests
- **Full import E2E (Go side):** Create job → poll chunks → verify `cbc_students` and `cbc_student_enrollments` rows
- **Idempotency:** Submit same key+payload twice → second returns `is_replay: true`
- **Idempotency collision:** Submit same key+different payload → HTTP 409 `duplicate_import`
- **One active job:** Submit two jobs for same school → second receives HTTP 409 `import_already_in_progress`
- **Cancellation:** Submit job → cancel → verify chunks cancel cooperatively
- **Retention cleanup:** Backdate `completed_at` to >30 days ago → run cleanup → verify staging/failure rows deleted

### Frontend tests
- **`validateRecord()`** — All field rules (required, length, character set, DOB bounds)
- **`detectDuplicates()`** — Within-file duplicate detection for all 4 key combinations
- **`parseFile()`** — CSV, Excel, TSV parsing correctness with edge cases (BOM, serial dates, multi-sheet)
- **`normalizeGender()`** — All variant mappings
- **IndexedDB** — Session save/restore/clear lifecycle
