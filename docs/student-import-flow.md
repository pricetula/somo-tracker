# Student Bulk Import Flow

## Overview
Students are imported as SIS-only records via the existing bulk-job / ImportOrchestrator stack. No user accounts, no Stytch provisioning, no class enrollment. Gender distribution per school is maintained in a denormalized table `student_gender_counts` and recomputed on job completion.

## Request / Response Contract

### Frontend → Backend
`POST /api/students/add`
Headers:
```
Idempotency-Key: <uuid>
Authorization: session cookie
```
Body:
```json
{
  "students": [
    {
      "admission_number": "string",   // required, unique per school
      "full_name": "string",
      "date_of_birth": "string",     // tolerant parser
      "gender": "M|F|OTHER|variants",
      "metadata": { ... }            // optional
    }
  ]
}
```
Response `202 Accepted`:
```json
{
  "job_id": "uuid",
  "status": "QUEUED",
  "total": 50
}
```

### Progress
`GET /api/students/jobs/:job_id`
`GET /api/students/jobs/:job_id/events` → SSE
Progress payload:
```json
{
  "status": "QUEUED|PROCESSING|COMPLETED|COMPLETED_WITH_ERRORS",
  "succeeded": 0,
  "failed": 0,
  "deferred": 0,
  "total": 50
}
```

## Architecture Diagram

```mermaid
flowchart LR
  U[User - Students Import UI] -->|POST /students/add + Idempotency-Key| H[StudentsImportHandler]
  H -->|validate admin, rows, dedup| S[StudentImportService]
  S -->|CreateBulkJobWithItems| DB[(bulk_jobs + bulk_job_items)]
  S -->|enqueue| A[Asynq student:import:batch]
  A --> W[StudentImportWorker]
  W -->|for each item| S2[NormalizeGender + ParseDate]
  W -->|INSERT| DB[(students)]
  W -->|update status| DB[(bulk_job_items)]
  W -->|increment counters| DB[(bulk_jobs)]
  W -->|on completion| S[RecomputeGenderCounts]
  S -->|UPSERT| DB[(student_gender_counts)]
  W -->|publish| R[Redis bulk_progress_<job_id>]
  R -->|SSE| H
  H -->|stream| U
```

## Step-by-Step Flow

### 1. Frontend Upload / Manual Import
* Page `/students/add` renders `StudentsImportOrchestrator`.
* Upload CSV or Manual Import tab builds `students[]` payload.
* `useStudentImport` posts to `/api/students/add` with `Idempotency-Key`.

### 2. Handler Validation
`backend/internal/api/students_import_handler.go`
* Auth: active school from session, user role ADMIN.
* Idempotency: `GET /students/jobs` by `idempotency_key` → return existing job if present.
* Row validation:
  * `admission_number` required, non-empty.
  * `full_name` required.
  * `date_of_birth` parsed via `services.ParseDate` tolerant layouts.
  * `gender` normalized via `services.NormalizeGender` → `M|F|OTHER`.
* Deduplication within file: first occurrence wins; subsequent duplicates marked `FAILED` immediately.
* Max rows 10k, rate limited.

### 3. Bulk Job Creation
`StudentImportService.CreateBulkJobWithItems`
* `INSERT INTO bulk_jobs` with `job_type = STUDENT_IMPORT`, `status = QUEUED`, `total_records = n`.
* For each validated row `INSERT INTO bulk_job_items` with `payload = json`, `status = PENDING`.
* Returns `job_id`.

### 4. Asynq Enqueue
Handler batches item IDs into `student:import:batch` tasks:
```json
{
  "job_id": "...",
  "batch_index": 0,
  "item_ids": ["uuid1", "uuid2", ...]
}
```
Queue `student_import`.

### 5. Worker Processing
`backend/internal/worker/student_import_worker.go` `StudentImportProcessor.ProcessTask`
* Set job status → `PROCESSING`.
* For each item:
  1. Load `bulk_job_items` payload.
  2. Normalize gender, parse DOB.
  3. `InsertStudent` → `INSERT INTO students (school_id, admission_number, full_name, date_of_birth, gender, metadata)`.
  4. On unique violation `admission_number` → item `FAILED`, job `failed++`.
  5. On success → item `SUCCEEDED`, job `succeeded++`.
  6. On transient DB error → item `DEFERRED`.
* Publish progress to Redis channel `bulk_progress_<job_id>`.

### 6. Completion & Gender Counts
When all items processed:
* Derive final job status: `COMPLETED` or `COMPLETED_WITH_ERRORS`.
* `StudentImportService.RecomputeGenderCounts(school_id)` → Option B:
  ```sql
  INSERT INTO student_gender_counts ...
  SELECT school_id, COUNT(*) FILTER (WHERE upper(gender) IN ('M','MALE')), ...
  FROM students WHERE school_id = $1
  ON CONFLICT (school_id) DO UPDATE ...
  ```
* Publish final progress.

### 7. Frontend Progress UI
`ImportOrchestrator` subscribes to `/students/jobs/:job_id/events` SSE:
* Renders progress bar, succeeded/failed counts.
* On completion shows summary and link to `/students` listing.

## Data Model Touchpoints
* `bulk_jobs` – `job_type` includes `STUDENT_IMPORT` via migration `000017_add_student_bulk_import_support`.
* `bulk_job_items` – one per student row.
* `students` – SIS record, unique `(school_id, admission_number)`.
* `student_gender_counts` – denormalized aggregates per school, updated on job completion.

## Error Handling
* Validation errors → `400` with canonical error body `{code, message, errors}`.
* Duplicate admission number in DB → item `FAILED`, job continues.
* Idempotency → repeated POST with same key returns existing job, no duplicate work.
* All errors returned up the stack with context, logged once at handler/worker layer.

## Permissions
* `ADMIN` only on school membership.
* Rate limiting `bulkInviteRate` reused for student import.

## Testing
* Migration integration test: `TestMigrator_AddStudentBulkImportSupport`.
* End-to-end: POST with Idempotency-Key → verify job status via SSE → verify rows in `students` and counts in `student_gender_counts`.

### Comprehensive Test Plan

#### Unit Tests – Backend
* **Service**: `internal/services/student_import_service_test.go`
  * `NormalizeGender` – M/male/boy → M; F/female/girl → F; empty/unknown → OTHER
  * `ParseDate` – tolerant layouts `2006-01-02`, `02/01/2006`, `01-02-2006`, `2006/01/02`, RFC3339; invalid → error
  * `CreateBulkJobWithItems` – idempotency key with `UNIQUE(school_id,idempotency_key)` returns same job_id, no duplicate items
  * `IncrementJobCounters` – atomic `UPDATE … RETURNING`
  * `TryFinalizeJob` – returns false while pending, true only once when sum == total; `FOR UPDATE` prevents double finalisation
  * `InsertStudent` – rejects duplicate admission_number via functional unique index, stores trimmed value
* **Handler**: `internal/api/students_import_handler_test.go`
  * `POST /students/add` – 401 without session, 403 non-ADMIN, 400 validation, 202 with job
  * In-file dedup – first occurrence wins
  * Idempotency – duplicate key returns existing job
  * Batch enqueue – payload batch size ≤100
* **Worker**: `internal/worker/student_import_worker_test.go`
  * Normalises gender, parses DOB, succeeds insert, marks SUCCEEDED
  * Unique violation → FAILED
  * Transient error → DEFERRED
  * Completion → `TryFinalizeJob` called once, gender counts recomputed

#### Unit Tests – Frontend
* **API client** `src/lib/api/students.test.ts` – query string building, uses `api.get('/api/students?...')`
* **Feature hooks** – `useStudentImport` builds payload with `Idempotency-Key`, handles 202 → polling
* **Component** – `StudentsTable` renders columns, correct `queryKey`, uses `api` client, `addHref="/students/add"`

#### Integration / E2E
* **Backend E2E** `backend/internal/api/students_import_e2e_test.go` `//go:build integration`
  * Seed school + admin, POST 50 rows mixed valid/invalid/duplicates
  * Poll SSE → status `COMPLETED_WITH_ERRORS`
  * Assert `bulk_jobs` counters, `bulk_job_items` statuses, rows in `students`, `student_gender_counts` matches
  * Repeat same Idempotency-Key → same job_id, no duplicate rows
* **Frontend E2E** `frontend/tests/e2e/students-import.spec.ts` Playwright
  * Login ADMIN → `/students/add` upload CSV → orchestrator progress → completion
  * `/students` DataTable shows rows, search works
  * Re-import same file → existing job returned
* **Regression**
  * Single student create/update/delete → trigger keeps `student_gender_counts` in sync
  * Delete admin user → `bulk_jobs.created_by` becomes NULL, audit trail preserved
  * Admission number case/space variation → unique constraint rejects

#### CI
* `go test ./... -tags integration` against ephemeral Postgres
* `pnpm lint` + `pnpm test`
* Playwright E2E on staging after migration `000018` applied
