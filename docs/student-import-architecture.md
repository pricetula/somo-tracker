# Student Import — Architecture & Data Flow

> **Last updated:** 2026-07-08  
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

---

## 1. Overview

Student import allows bulk-adding students via two paths:

- **Manual Entry** — user types student data into a table in the browser
- **File Import** — user uploads a CSV/Excel file, maps columns, resolves classes, reviews, then imports

Both paths ultimately call the same backend endpoint (`POST /api/v1/students/import`) and share the same progress tracking component (`<ImportProgress>`).

### Design principles

- **The frontend NEVER chunks data.** All rows are sent in a single POST request. The backend handles splitting into chunks for async processing.
- **The frontend polls for progress.** After submitting, the frontend polls `GET /api/v1/imports/:job_id` every 1.5s until the job reaches a terminal status.
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
1. Loads staging rows for the chunk range
2. Calls `StudentImporter.Validate()` — schema checks (full_name required, gender M/F, class_id optional)
3. Calls `StudentImporter.ResolveReferences()` — injects tenant_id, school_id, academic_term_id, academic_year_id into each row
4. Calls `StudentImporter.BulkInsert()` — returns error (forces savepoint fallback since student+enrollment needs per-row handling)
5. Falls back to `insertWithSavepoints()` → calls `InsertOne()` per row
6. Each `InsertOne()`:
   - Inserts into `cbc_students`
   - If `class_id` is present, also inserts into `cbc_student_enrollments`
7. Failures go to `import_job_failures`
8. `AtomicChunkCompletion()` updates job counters atomically

### Step 6: Completion (Frontend)

When poll detects terminal status:
- If `completed`: green success banner
- If `completed_with_errors` or `failed`: amber banner with:
  - Success/failure counts
  - Failed rows list (from `getImportFailures`)
  - "Retry failed" button
- "Done" button resets to the import selector

---

## 5. Frontend Components

### 5.1 `StudentsImportForm` (`students-import.tsx`)

The parent orchestrator. Manages:
- `selectedImportType`: "manual" | "file" | null
- `activeJob`: `{ jobId, totalRecords } | null`

When `activeJob` is set, renders `<ImportProgress>` instead of the child forms.

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

### 5.3 `ImportProgress` (`import-progress.tsx`)

Shared progress component used by both import paths. Handles:
- Polling `GET /api/v1/imports/:job_id` every 1.5s
- Displaying progress bar with `processed / total` counts
- Showing result banner on completion (green/amber)
- Fetching failures via `GET /api/v1/imports/:job_id/failures`
- "Done" and "Retry failed" buttons

**Props:**
- `jobId: string`
- `totalRecords: number`
- `onDone: () => void`
- `onRetry?: (failedPayloads: Record<string, unknown>[]) => void`

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

- **Validate()**: Checks full_name is present, gender is "M" or "F". class_id is optional.
- **ResolveReferences()**: Injects tenant_id, school_id, academic_term_id, academic_year_id from job metadata into each row.
- **BulkInsert()**: Returns error to force the per-row savepoint fallback.
- **InsertOne()**: Inserts student into `cbc_students`. If class_id is present, also inserts enrollment into `cbc_student_enrollments`.

### 6.3 `internal/imports/service.go`

- **CreateJob()**: Creates import job, writes staging rows, splits into chunks, enqueues Asynq tasks.
  - `ChunkSize = 100` (rows per chunk)
- **GetJob()**: Returns current job state (for polling).
- **GetFailures()**: Returns paginated failure records.
- **AtomicChunkCompletion()**: Atomically updates job counters and determines new status.

### 6.4 `internal/students/handler.go` (BulkImport)

```
POST /api/v1/students/import
```

- Resolves academic year and term from server-side (current active year/term)
- Does NOT accept `academic_term_id` from the frontend — it's resolved automatically
- Creates import job via `imports.Service.CreateJob()`
- Returns `ImportResponse` immediately (async processing)

---

## 7. Key API Endpoints

| Method | Path | Purpose | Response |
|--------|------|---------|----------|
| `POST` | `/api/v1/students/import` | Create import job | `{ job_id, total_records, total_chunks, status }` |
| `GET` | `/api/v1/imports/:job_id` | Poll job status | `ImportJob` (full state) |
| `GET` | `/api/v1/imports/:job_id/failures` | Get failure details | `{ failures: [...], total: number }` |

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
  "completed_at": "2026-07-08T12:00:30Z"
}
```

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
| `no_active_academic_year` | 400 | School has no current academic year set |
| `no_active_academic_term` | 400 | No active term in the current academic year |
| `duplicate_import` | 409 | Idempotency key reused with a different payload. The same key was used for a different set of rows. |

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

**Frontend:**
- `src/lib/api/imports.ts` — API client functions
- `src/features/students/components/students-import/students-import.tsx` — Parent orchestrator
- `src/features/students/components/students-import/import-progress.tsx` — Shared progress
- `src/features/students/components/students-import/manual-import-form.tsx` — Manual entry
- `src/features/students/components/students-import/file-importer/file-importer.tsx` — File import wizard
- `src/features/students/components/students-import/file-importer/step-streaming.tsx` — Submit step
- `src/features/students/components/students-import/file-importer/utils/validation-utils.ts` — Validation

**Backend:**
- `internal/students/handler.go` — BulkImport handler
- `internal/students/domain.go` — Import types
- `internal/students/importer.go` — StudentImporter implementation
- `internal/imports/service.go` — Import job lifecycle
- `internal/imports/domain.go` — Job, Chunk, Importer interface
- `internal/imports/handler.go` — Job status + failures endpoints
