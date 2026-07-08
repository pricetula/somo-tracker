# Bulk Student Import — Streaming Architecture Report

> **Date:** 2026-07-08  
> **Scope:** Frontend `features/students/components/students-import/` ↔ Backend `internal/students/` + `internal/imports/`  
> **Status:** ⚠️ Multiple critical mismatches identified

---

## Table of Contents

1. [Frontend Flow](#1-frontend-flow)
2. [Backend Flow](#2-backend-flow)
3. [How Each Side Thinks It Works](#3-how-each-side-thinks-it-works)
4. [Critical Mismatches](#4-critical-mismatches)
5. [Synchronous-vs-Async Breakdown](#5-synchronous-vs-async-breakdown)
6. [Unused Code / Dead Ends](#6-unused-code--dead-ends)
7. [Data Flow Diagram](#7-data-flow-diagram)
8. [Recommendations](#8-recommendations)

---

## 1. Frontend Flow

### 1.1 Wizard Steps

Defined in `src/features/students/components/students-import/file-importer/types.ts`:

```
upload → column_mapping → class_resolving → data_review → streaming
```

### 1.2 Streaming Step (`step-streaming.tsx`)

The "streaming" step is where the actual HTTP call happens. The logic:

1. **On mount**, loads all `StagedStudentRecord[]` with `status === "valid"` from IndexedDB.
2. **User clicks "Start Import"** → triggers `startStreaming()`.
3. **Splits records into batches of `BATCH_SIZE = 200`.**
4. **Processes batches in concurrent groups of `MAX_CONCURRENT_BATCHES = 3`.**
5. **For each batch**, calls `submitStudentImport(payload)` which is `POST /api/v1/students/import`.

```typescript
// step-streaming.tsx line ~48
const response = await submitStudentImport(payload);

// Then immediately interprets the response:
onBatchComplete(
    response.status === "completed" ? rows.length : 0,
    response.status === "completed_with_errors" ? rows.length : 0,
    batchId
);
```

6. **After all batches**, clears IndexedDB staging and calls `onComplete()`.

### 1.3 API Client (`src/lib/api/imports.ts`)

```typescript
// Submit — creates a new import job
export async function submitStudentImport(body: ImportRequest): Promise<ImportResponse> {
    return api.post<ImportResponse>("/api/v1/students/import", body);
}

// Poll job status (NEVER CALLED in step-streaming.tsx)
export async function getImportJob(jobId: string): Promise<ImportJob> { ... }

// Get failures (NEVER CALLED)
export async function getImportFailures(jobId: string, params): Promise<...> { ... }
```

### 1.4 What the Frontend Sends Per Batch

```json
{
  "academic_term_id": "",
  "rows": [
    {
      "full_name": "Alice Wanjiku",
      "gender": "F",
      "date_of_birth": null,
      "upi_number": null,
      "knec_assessment_number": null
    }
  ]
}
```

```typescript
// step-streaming.tsx line ~36
const payload: ImportRequest = {
    rows,
    academic_term_id: "", // ← Will fail backend validation
};
```

### 1.5 What the Frontend Expects Back

```typescript
export interface ImportResponse {
    job_id: string;
    total_records: number;
    total_chunks: number;
    status: ImportJobStatus; // "completed" | "completed_with_errors" | ...
}
```

The frontend checks `response.status === "completed"` and `response.status === "completed_with_errors"` to determine batch success/failure.

---

## 2. Backend Flow

### 2.1 Handler (`internal/students/handler.go`)

`POST /api/v1/students/import` → `h.BulkImport()`:

1. Parses tenant/school/user from request context
2. Validates `academic_term_id` is present and belongs to this school
3. Gets `academic_year_id` from the term
4. Builds metadata JSON with `academic_term_id` + `academic_year_id`
5. Converts `ImportRow[]` to `json.RawMessage[]`
6. **Calls `imports.Service.CreateJob()`** — which creates an async job

```go
resp, err := h.impSvc.CreateJob(c.Context(), req)
// Returns immediately with job_id, total_records, total_chunks, status
```

### 2.2 Service (`internal/imports/service.go`)

`CreateJob()`:
1. Checks idempotency key (not sent by frontend)
2. Creates `import_jobs` row with `status = "pending"`
3. **Splits rows into chunks of `ChunkSize = 100`** (backend constant)
4. Inserts all rows into `import_job_staging` table
5. Enqueues Asynq tasks (`imports:process_chunk`) — one per chunk
6. Updates status to `processing`
7. **Returns immediately** (async)

**The job is NOT complete when the response is sent.** The response status will be `"processing"`.

### 2.3 Worker (`internal/imports/module.go`)

**Asynq worker** processes each chunk:

```
imports:process_chunk → Service.ProcessChunk()
```

### 2.4 ProcessChunk Flow

1. Gets chunk staging rows from DB
2. Calls `Importer.Validate()` — schema checks (full_name, gender, grade/stream)
3. Calls `Importer.ResolveReferences()` — grade_level+stream_name → class_id
4. Calls `Importer.BulkInsert()` — purposely returns error for students (needs per-row inserts)
5. Falls back to `insertWithSavepoints()` → calls `Importer.InsertOne()` per row
6. Inserts failures into `import_job_failures`
7. Marks staging rows as succeeded
8. Calls `AtomicChunkCompletion()` — updates job counters atomically

### 2.5 Chunk Completion Status Logic

```sql
status = CASE
    WHEN processed_chunks + 1 = total_chunks AND failed_count + $4 = 0
        THEN 'completed'::import_job_status
    WHEN processed_chunks + 1 = total_chunks
        THEN 'completed_with_errors'::import_job_status
    ELSE 'processing'::import_job_status
END
```

The job transitions to `completed`/`completed_with_errors` **only when ALL chunks are done**.

### 2.6 Registered SSE/Progress Types (`internal/imports/domain.go`)

```go
// ProgressEvent is emitted via Redis Pub/Sub.
type ProgressEvent struct {
    JobID            string          `json:"job_id"`
    Status           ImportJobStatus `json:"status"`
    TotalRecords     int             `json:"total_records"`
    ProcessedRecords int             `json:"processed_records"`
    SuccessCount     int             `json:"success_count"`
    FailedCount      int             `json:"failed_count"`
    TotalChunks      int             `json:"total_chunks"`
    ProcessedChunks  int             `json:"processed_chunks"`
}
```

**This type is defined but NEVER published anywhere.** There is no Redis Pub/Sub `Publish` call, no SSE endpoint, no progress channel wiring.

---

## 3. How Each Side Thinks It Works

### Frontend's Mental Model

```
     ┌─────────────────────────────┐
     │        User clicks          │
     │      "Start Import"         │
     └──────────┬──────────────────┘
                │
     ┌──────────▼──────────────────┐
     │  Split 2000 records into    │
     │  10 batches of 200          │
     └──────────┬──────────────────┘
                │
     ┌──────────▼──────────────────┐
     │ For each batch (concurrent  │
     │ groups of 3):               │
     │ POST /students/import       │
     │ → wait for response         │
     │ → check status == completed │
     └──────────┬──────────────────┘
                │
     ┌──────────▼──────────────────┐
     │  All batches done → done!   │
     └─────────────────────────────┘
```

**Key assumption:** The POST response tells us whether the batch succeeded or failed synchronously.

### Backend's Actual Model

```
     ┌─────────────────────────────┐
     │      POST /students/import  │
     └──────────┬──────────────────┘
                │
     ┌──────────▼──────────────────┐
     │  Create import_jobs row     │
     │  Split 200 rows into        │
     │  2 chunks of 100            │
     │  Enqueue 2 Asynq tasks      │
     │  Return immediately         │
     │  status = "processing"      │
     └──────────┬──────────────────┘
                │
     ┌──────────▼──────────────────┐
     │  Asynq worker picks up      │
     │  chunk tasks (async)        │
     └──────────┬──────────────────┘
                │
     ┌──────────▼──────────────────┐
     │  Chunk 1: Validate → Resolve│
     │  → insert 100 students      │
     │  → AtomicChunkCompletion    │
     └──────────┬──────────────────┘
                │
     ┌──────────▼──────────────────┐
     │  Chunk 2: same, then        │
     │  status → "completed"       │
     └─────────────────────────────┘
```

---

## 4. Critical Mismatches

### 🔴 MISMATCH 1: Synchronous vs. Asynchronous (HIGHEST SEVERITY)

| Aspect | Frontend Expects | Backend Delivers |
|--------|-----------------|------------------|
| Response timing | POST returns after processing | POST returns immediately when job created |
| Status in response | `"completed"` or `"completed_with_errors"` | `"processing"` |
| Data persistence | Row data sent in request body | Row data stored in `import_job_staging` for async workers |

**Impact:** The frontend considers a batch "done" as soon as the HTTP response arrives, but the backend hasn't inserted anything yet. The progress (`success_count`, `failed_count`) is always wrong.

### 🔴 MISMATCH 2: Multiple Jobs Instead of One Job

| Aspect | Frontend Says | Backend Sees |
|--------|--------------|--------------|
| Number of `import_jobs` rows per import | 1 | N/BATCH_SIZE (e.g., 10 jobs for 2000 students) |
| Chunks per job | 1 | total_rows / 100 (e.g., 2 chunks for 200 rows) |
| Atomicity | Should be one transaction | Split across separate async jobs |
| Retry behavior | Retries = sends new POST | Each retry creates new jobs |

**Impact:** Instead of one import job with N chunks processed async, the system creates N/200 *separate* import jobs. The backend's sophisticated chunking/status tracking is wasted.

### 🔴 MISMATCH 3: `academic_term_id` is Empty String

```typescript
const payload: ImportRequest = {
    rows,
    academic_term_id: "", // ← step-streaming.tsx line ~41
};
```

Backend validation: `academic_term_id is required` — returns 400 error.

**Impact:** Every batch call to the backend will fail with a 400 error because `academic_term_id` is empty. The frontend would catch this and mark the whole batch as failed.

### 🔴 MISMATCH 4: No Progress Tracking (SSE / Polling)

| Component | Status |
|-----------|--------|
| Frontend defines `ImportProgressEvent` | ✅ Types exist in `src/lib/api/imports.ts` |
| Frontend calls polling endpoint | ❌ `getImportJob()` is defined but never called |
| Frontend connects to SSE | ❌ No `EventSource` usage anywhere |
| Backend defines `ProgressEvent` | ✅ Types exist in `internal/imports/domain.go` |
| Backend publishes progress | ❌ No Redis Pub/Sub publish call exists |
| Backend exposes SSE endpoint | ❌ No SSE endpoint implemented |
| Backend exposes polling endpoint | ✅ `GET /api/v1/imports/:job_id` exists |

**Impact:** The frontend has zero visibility into async job progress. Even if the architecture were corrected to use async jobs, there's no mechanism to show progress to the user.

### 🔴 MISMATCH 5: Class Resolution Location

| Step | Frontend | Backend |
|------|----------|---------|
| Where resolved | Client-side in `step-class-resolve.tsx` | Server-side in `ResolveReferences()` |
| What's sent | `class_id` as scalar | `grade_level` + `stream_name` |
| Field name | `class_id` | `grade_level` + `stream_name` |

**Impact:** The frontend builds `StagedStudentRecord.payload` with a `class_id` field. The backend's `ImportRow` struct has `GradeLevel` and `StreamName` but no `ClassID` field. The backend expects `grade_level` + `stream_name` and resolves them server-side. The frontend sends neither.

### 🔴 MISMATCH 6: Batch Size vs. Chunk Size

| Layer | Value |
|-------|-------|
| Frontend `BATCH_SIZE` | 200 |
| Backend `ChunkSize` | 100 |

**Impact:** Even if job creation were correct, the backend would split each frontend batch (200 rows) into 2 chunks (100 each). This isn't inherently wrong but doubles the chunk count unnecessarily.

### 🔴 MISMATCH 7: Idempotency Key Not Sent

| Aspect | Status |
|--------|--------|
| Backend supports idempotency | ✅ `idempotency_key` column and `uq_import_jobs_tenant_idempotency` constraint |
| Frontend sends idempotency key | ❌ `toImportRow()` never sets it, and the payload type has `idempotency_key?: string \| null` |

**Impact:** Every retry or duplicate batch call creates a brand new import job. No deduplication protection.

### 🔴 MISMATCH 8: Duplicate Detection Gap

| Where | What's checked |
|-------|---------------|
| Frontend (`validation-utils.ts`) | Client-only: duplicate UPI and name+DOB within the **file itself** |
| Backend (importer.go) | No duplicate checking against existing database records |

**Impact:** No server-side check for UPI uniqueness against the database. The `cbc_students` table may or may not have a UPI unique constraint. If it does, DB constraint violations are caught in the savepoint fallback. If not, duplicate UPIs across imports are silently accepted.

---

## 5. Synchronous-vs-Async Breakdown

### What Currently Happens End-to-End

```
StepStreaming                       Backend
    │                                  │
    │  POST /api/v1/students/import    │
    │  (batch of 200 rows)            │
    │─────────────────────────────────>│
    │                                  │  CreateJob():
    │                                  │  ├─ INSERT import_jobs (status=processing)
    │                                  │  ├─ INSERT 200 staging rows
    │                                  │  ├─ Enqueue 2 Asynq tasks
    │                                  │  └─ Return {job_id, status:"processing"}
    │<─────────────────────────────────│
    │                                  │
    │  // Frontend sees:               │  // Backend workers haven't started yet
    │  response.status === "processing"│
    │                                  │
    │  // So neither check matches:    │
    │  response.status === "completed"?│  ❌ false
    │  response.status === "completed_ │  ❌ false
    │    with_errors"?                 │
    │                                  │
    │  // Result: both success=0,      │
    │  // failed=0 for this batch      │
    │                                  │
    │  // No polling or SSE happens    │
    │  // Progress is always wrong     │
```

### What Should Happen (Corrected Architecture)

```
StepStreaming                       Backend
    │                                  │
    │  POST /api/v1/students/import    │
    │  (ALL rows, one big job)         │
    │─────────────────────────────────>│
    │                                  │  CreateJob(): create job + chunks
    │<─────────────────────────────────│  Return {job_id, status:"processing"}
    │                                  │
    │  // Connect to SSE stream        │  // Workers start processing
    │  GET /imports/:job_id/stream     │
    │─────────────────────────────────>│
    │                                  │  │
    │  Chunk 1 done: progress event    │  │
    │<─────────────────────────────────│  │
    │  Chunk 2 done: progress event    │  │
    │<─────────────────────────────────│  │
    │  ...                             │  │
    │                                  │  │
    │  Final event: status=completed   │  │
    │<─────────────────────────────────│  │
    │                                  │
    │  GET /imports/:job_id/failures   │
    │─────────────────────────────────>│
    │<─────────────────────────────────│  Return failures
    │                                  │
    │  Show final result to user       │
```

---

## 6. Unused Code / Dead Ends

### 6.1 Frontend Unused Code

| File | Code | Why Dead |
|------|------|----------|
| `src/lib/api/imports.ts` | `getImportJob()` | Never imported or called from any component |
| `src/lib/api/imports.ts` | `getImportFailures()` | Never imported or called from `step-streaming.tsx` |
| `src/lib/api/imports.ts` | `ImportProgressEvent` | Type exists, never used |
| `src/lib/api/imports.ts` | `ImportRowFailure` | Type exists, never used |
| `src/features/students/components/students-import/file-importer/types.ts` | `StreamingProgress.error_message` | Field defined but never set by the streaming code |

### 6.2 Backend Unused Code

| File | Code | Why Dead |
|------|------|----------|
| `internal/imports/domain.go` | `ProgressEvent` struct | Comment says "emitted via Redis Pub/Sub" but no Publish call exists anywhere in the codebase |
| `internal/imports/service.go` | `StreamToken` field in `CreateJobResponse` | Set to empty string, never generated or used |
| `internal/imports/service.go` | Progress publication after `AtomicChunkCompletion` | No Pub/Sub publish after chunk completion |
| All backend | SSE endpoint (`GET /imports/:job_id/stream`) | Does not exist |

---

## 7. Data Flow Diagram

```
┌───────────────────────────────────────────────────────────────┐
│                       FRONTEND                                │
│                                                               │
│  ┌─────────────┐   ┌───────────────┐   ┌──────────────────┐  │
│  │  IndexedDB   │   │ StepStreaming │   │  api.post()      │  │
│  │  staging     │──>│ (step-        │──>│  /students/import │  │
│  │  (crash      │   │  streaming    │   │  (per batch of   │  │
│  │  recovery)   │   │  .tsx)        │   │  200 rows)       │  │
│  └─────────────┘   └───────┬───────┘   └────────┬─────────┘  │
│                            │                    │            │
│                   Marks as submitted            │            │
│                            │                    │            │
│                            └────────────────────┘            │
│                                                               │
│  ❌ No polling                                              │
│  ❌ No SSE connection                                        │
└───────────────────────────────────────────────────────────────┘
                            │
                            │ HTTP POST
                            ▼
┌───────────────────────────────────────────────────────────────┐
│                       BACKEND                                 │
│                                                               │
│  ┌───────────────┐    ┌──────────────┐    ┌────────────────┐  │
│  │ student       │───>│ imports      │───>│ PostgreSQL     │  │
│  │ handler.go    │    │ Service      │    │ import_jobs    │  │
│  │ BulkImport()  │    │ CreateJob()  │    │ import_job_    │  │
│  └───────────────┘    └──────┬───────┘    │ staging        │  │
│                              │            └────────────────┘  │
│                              ▼                                │
│                     ┌────────────────┐                        │
│                     │ Asynq Redis    │                        │
│                     │ Queue          │                        │
│                     │ (chunk tasks)  │                        │
│                     └───────┬────────┘                        │
│                             │                                 │
│                             ▼                                 │
│                     ┌────────────────┐                        │
│                     │ Asynq Worker   │                        │
│                     │ ProcessChunk() │                        │
│                     │                │                        │
│                     │ 1. Validate    │                        │
│                     │ 2. ResolveRefs │                        │
│                     │ 3. BulkInsert  │──> cbc_students       │
│                     │ 4. Savepoint   │──> cbc_student_       │
│                     │    fallback    │    enrollments        │
│                     │ 5. Atomic      │                        │
│                     │    Completion   │──> import_jobs        │
│                     └────────────────┘   (counters updated)  │
│                                                               │
│  ❌ No SSE endpoint                                           │
│  ❌ No Redis Pub/Sub publish                                  │
└───────────────────────────────────────────────────────────────┘
```

---

## 8. Recommendations

### P0 — Fix the Architecture (Must Fix)

1. **Send ALL rows in one POST, not one per batch.** The frontend should submit the entire set of valid records as one import job and let the backend's chunking engine handle the splitting.

2. **Implement progress tracking.** Either:
   - **Option A (SSE):** Backend publishes `ProgressEvent` via Redis Pub/Sub → SSE endpoint, frontend connects via `EventSource`.
   - **Option B (Polling):** Frontend periodically calls `GET /imports/:job_id` and updates the progress bar.

3. **Send `academic_term_id`.** The frontend must capture and pass the actual academic term ID through the wizard to the streaming step.

4. **Align class resolution:** Either:
   - Send `grade_level` + `stream_name` to the backend and let it resolve classes server-side (recommended — matches backend design).
   - OR add a `class_id` field to the backend's `ImportRow` struct and skip server-side resolution for pre-resolved rows.

### P1 — Correct Batch/Chunk Behavior

5. **Remove client-side batching from `StepStreaming`.** The frontend should not slice records into batches of 200 — the backend's `ChunkSize = 100` handles this.

6. **Remove `MAX_CONCURRENT_BATCHES = 3`.** If there's one job, there's no need for concurrent HTTP requests.

### P2 — Leverage Existing Endpoints

7. **Wire up `getImportFailures()`** after the import completes to fetch and display per-row errors to the user.

8. **Use the idempotency key** for retry safety.

### P3 — Safety

9. **Add server-side duplicate UPI checking** in `StudentImporter.Validate()` or `InsertOne()`.

10. **Consider a DB unique constraint on `cbc_students.upi_number`** if UPI is meant to be globally unique per tenant.

---

## Appendix: Key File References

### Frontend

| File | Purpose |
|------|---------|
| `src/features/students/components/students-import/file-importer/step-streaming.tsx` | Streaming step — contains the broken batching logic |
| `src/lib/api/imports.ts` | API client with unused SSE/progress types |
| `src/features/students/components/students-import/file-importer/types.ts` | Type definitions (StagedStudentRecord, StreamingProgress) |
| `src/features/students/components/students-import/file-importer/db.ts` | IndexedDB persistence layer |
| `src/features/students/components/students-import/file-importer/utils/validation-utils.ts` | Client-side validation + duplicate detection |
| `src/features/students/components/students-import/file-importer/file-importer.tsx` | Wizard orchestrator |
| `src/lib/api/client.ts` | Base HTTP client with ApiError |

### Backend

| File | Purpose |
|------|---------|
| `internal/students/importer.go` | StudentImporter — Validate, ResolveReferences, InsertOne |
| `internal/students/handler.go` | BulkImport handler (POST /students/import) |
| `internal/students/domain.go` | ImportRow, ImportRequest, ImportResponse, ImportRepository interface |
| `internal/imports/service.go` | CreateJob, ProcessChunk, chunk management |
| `internal/imports/domain.go` | Job, Chunk, ProgressEvent, Importer interface |
| `internal/imports/handler.go` | GET /imports/:job_id, GET /imports/:job_id/failures |
| `internal/imports/repository.go` | Postgres queries for import_jobs + staging |
| `internal/imports/module.go` | Asynq worker registration |
