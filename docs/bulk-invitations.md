# Bulk Staff Invitations — Admin, Nurse & Finance

> **Last updated:** 2026-06-24 (TEACHER role & TSC Number support added)
> **Owner:** Platform team

This document describes how school administrators can invite staff members in bulk — covering both the **simple invitation creation** flow and the **high-volume bulk import** pipeline with async processing, real-time progress, and correction recovery.

---

## Table of Contents

1. [Overview](#overview)
2. [Supported Roles](#supported-roles)
3. [Quick Invite Flow](#quick-invite-flow)
4. [Bulk Import Flow](#bulk-import-flow)
5. [Frontend UX Walkthrough](#frontend-ux-walkthrough)
6. [Backend Architecture](#backend-architecture)
7. [Correction & Recovery](#correction--recovery)
8. [Database Schema](#database-schema)
9. [Error Handling](#error-handling)
10. [FAQ](#faq)

---

## Overview

School administrators can invite new staff members to join their school through two complementary approaches:

| Approach                                       | Max records | Invite email sent?                | Processing           | Best for                              |
| ---------------------------------------------- | ----------- | --------------------------------- | -------------------- | ------------------------------------- |
| **Quick invite** — ad-hoc creation             | ~100        | No (manual Stytch send on demand) | Synchronous          | Small batches, individual invites     |
| **Bulk import** — `POST /api/v1/imports/staff` | 5,000       | ✅ Yes (Stage 2 Stytch send)      | Async (Asynq worker) | Large CSVs, hundreds of staff members |

### Supported roles

Four staff roles can be invited through these flows:

| Role           | Allowed access                                                      | Typical use                                    |
| -------------- | ------------------------------------------------------------------- | ---------------------------------------------- |
| `SCHOOL_ADMIN` | `/admin`, `/admins`, `/dashboard`, `/settings`, `/schools`, `/docs` | Deputy head, senior teachers, operations       |
| `TEACHER`      | `/dashboard`, `/docs`, `/teachers`                                  | Classroom teachers (KICD-compliant onboarding) |
| `NURSE`        | `/dashboard`, `/docs`                                               | School nurse, healthcare staff                 |
| `FINANCE`      | `/dashboard`, `/docs`                                               | Bursar, accounting staff                       |

> **Note:** `SYSTEM_ADMIN` uses a separate provisioning flow outside this pipeline.

### Who can invite

Any **authenticated user** with a valid session can create invitations for their school. The invitee's role is assigned at invitation time and the user's session must belong to the same tenant/school.

---

## Quick Invite Flow

> **⚠️ The quick invite endpoints (`POST /api/v1/invitations` and `POST /api/v1/members/invite`) are not yet implemented.** Currently only the bulk import pipeline (see below) is available for creating invitations. The `GET /api/v1/invitations` endpoint exists for listing invitations.

For reference, the planned quick invite flow will support:

- Creating one or more invitation records with individual roles per invitee.
- Optionally sending a Stytch invite email via `POST /api/v1/members/invite`.
- Role validation against `SCHOOL_ADMIN`, `NURSE`, `FINANCE` (not `TEACHER`).

See the [`members/` handler](backend/internal/members/handler.go) for the current invitation listing endpoint.

---

## Bulk Import Flow

The bulk import pipeline handles **up to 5,000 records** using an async worker architecture with real-time progress feedback.

### High-level flow

```
┌─────────┐     ┌──────────────┐     ┌───────────┐     ┌──────────┐
│ Browser │     │ Go API       │     │  Asynq    │     │  Stytch  │
│ (FE)    │     │ (Fiber)      │     │  Worker   │     │  (B2B)   │
└────┬────┘     └──────┬───────┘     └─────┬─────┘     └────┬─────┘
     │                 │                   │                │
     │  POST /api/v1   │                   │                │
     │  /imports/staff │                   │                │
     │────────────────>│                   │                │
     │                 │  Create import_job │               │
     │                 │  Enqueue Asynq task│               │
     │  202 Accepted   │──────────────────>│                │
     │<────────────────│                   │                │
     │                 │                   │                │
     │  ── SSE stream ──────────────────────────────────────│
     │  /track/:id/sse │                   │                │
     │                                                      │
     │                 │                   │  Stage 1:      │
     │                 │                   │  Bulk insert    │
     │                 │                   │  invitations    │
     │                 │                   │  (CTE batch)    │
     │                 │                   │       │         │
     │                 │                   │  Stage 2:       │
     │                 │                   │  Stytch send   │
     │                 │                   │  (idempotent)  │
     │                 │                   │───────────────>│
     │                 │                   │                │
     │   SSE: progress │                   │                │
     │<────────────────│<──────────────────│                │
     │                 │                   │                │
     │   SSE: finished │                   │                │
     │<────────────────│<──────────────────│                │
```

### Endpoints

| Method | Path                                  | Purpose                                 |
| ------ | ------------------------------------- | --------------------------------------- |
| `POST` | `/api/v1/imports/staff`               | Start a bulk import job                 |
| `GET`  | `/api/v1/imports/staff/track/:id`     | Poll job status                         |
| `GET`  | `/api/v1/imports/staff/track/:id/sse` | Real-time SSE progress stream           |
| `GET`  | `/api/v1/imports/staff/:id/failures`  | Fetch failed invitations for correction |

### Step 1: Start Import — `POST /api/v1/imports/staff`

#### Request

```json
{
  "role": "NURSE",
  "records": [
    {
      "temp_id": "a1b2c3d4-...",
      "email": "nurse1@school.edu",
      "full_name": "Mwangi",
      "phone": "+254712345678",
      "registration_number": "NRS-001"
    }
  ]
}
```

- `role` — must be `SCHOOL_ADMIN`, `NURSE`, or `FINANCE`.
- `records` — array of records, max **5,000**.
- `temp_id` — client-generated UUID for reconciliation (each row must have a unique one).
- `phone` is optional.
- `registration_number` (TSC Number) is **optional for SCHOOL_ADMIN, NURSE, FINANCE** but **mandatory for TEACHER**.

#### Validation rules (backend)

| Field        | Rule                                                         |
| ------------ | ------------------------------------------------------------ |
| `email`      | Required, must be non-empty                                  |
| `full_name`  | Required, must be non-empty                                  |
| `role`       | Must be one of `SCHOOL_ADMIN`, `TEACHER`, `NURSE`, `FINANCE` |
| Record count | Between 1 and 5,000                                          |

#### Response (202 Accepted)

```json
{
  "import_job_id": "uuid-string",
  "status": "pending",
  "total": 42
}
```

If the Asynq task cannot be enqueued (e.g., Redis is down), the response returns `status: "enqueue_failed"` along with the job ID. The client can retry via a fresh import or have an operator investigate the queue.

### Step 2: Processing (Async)

After accepting the request, the backend:

1. **Creates an `import_jobs` row** with `status = 'pending'`.
2. **Enqueues an Asynq task** on the `critical` queue with `MaxRetry(3)` and a **45-minute timeout** (`TaskTimeout`).
3. Returns immediately with `202 Accepted`.

The Asynq worker then processes the import in **two stages**:

#### Stage 1 — Bulk DB Ingestion

- Processes records in **batches of 200** (`BatchSize`).
- Uses a **CTE (Common Table Expression)** to bulk-insert invitations with `ON CONFLICT ... DO NOTHING` on the unique index `(tenant_id, school_id, email)` where status is not expired/revoked.
- Returns a `map[temp_id]invitation_id` for reconciliation + a list of duplicates.
- Duplicates (existing active invites) are counted as failures.
- On correction resubmit (when `parentImportJobID` is set), uses `BulkUpdateInvitations` instead to update existing rows in-place (updating email, name, phone, role, resetting `status = 'pending'`, clearing `error_message` and `attempt_count`).

#### Stage 2 — Stytch Email Dispatch (Idempotent)

Stage 2 is **re-entry safe**: before sending each invite, the worker queries `GetPendingStage2Records` to retrieve only invitations that lack a `stytch_member_id` and are not in `invite_failed` status. This means:

- **On first run**: all newly inserted records are picked up.
- **On task retry** (e.g., after a crash mid-Stage 2): only records that have not yet been sent to Stytch are processed. Already-invited records (with `stytch_member_id` set) are skipped.
- **On correction resubmit**: `BulkUpdateInvitations` resets records back to `status = 'pending'` and clears `stytch_member_id` and `error_message`, so `GetPendingStage2Records` picks them up again.

Processing details:

- Sends Stytch invite emails with **bounded concurrency** (8 goroutines via a buffered channel semaphore).
- Each call has **3 retries** (`StytchMaxRetries`) with exponential backoff (2s, 4s, 6s).
- Permanent Stytch errors (invalid email, blocked domain, member already exists) are detected via `isPermanentStytchError` and **not retried**.
- Failed records at this stage are marked `status = 'invite_failed'` with the error message and `attempt_count` set to `StytchMaxRetries`.
- Progress is published to Redis pub/sub after each record via `publishProgress` (non-fatal if Redis is down).

#### Worker Error Handler

If an Asynq task exhausts all 3 retries, the `HandleError` method (implementing `asynq.ErrorHandler`) marks the job as `status = 'failed'` with `completed_at` set, so the job is not left stuck as `'processing'` indefinitely.

### Step 3: Track Progress

#### Polling — `GET /api/v1/imports/staff/track/:id`

```json
{
  "job": {
    "id": "uuid",
    "status": "processing",
    "total_records": 500,
    "processed_records": 200,
    "success_count": 195,
    "failed_count": 5,
    ...
  },
  "failed_records": 0
}
```

Job statuses:

| Status                  | Meaning                                                                                          |
| ----------------------- | ------------------------------------------------------------------------------------------------ |
| `pending`               | Job created, not yet picked up by worker                                                         |
| `enqueue_failed`        | Asynq task could not be enqueued (Redis/queue issue); retry via a fresh import                   |
| `processing`            | Worker is actively processing                                                                    |
| `completed`             | All records processed successfully                                                               |
| `completed_with_errors` | Some records failed (check failures endpoint)                                                    |
| `failed`                | All 3 Asynq retries exhausted (e.g., persistent DB or Stytch errors) — job never fully completed |

#### SSE — `GET /api/v1/imports/staff/track/:id/sse`

Server-Sent Events endpoint for real-time progress. Uses `c.Context().SetBodyStreamWriter` for streaming with Fiber.

All events use the same `ImportProgressEvent` JSON struct with a `type` discriminator:

```json
// Event: connected (sent immediately on stream open)
{ "type": "connected", "import_job_id": "uuid" }

// Event: progress (sent per-record during Stage 2)
{ "type": "import_progress", "status": "processing", "processed_records": 42, ... }

// Event: finished (sent once on completion)
{ "type": "import_finished", "status": "completed", ... }
```

The SSE stream:

1. Sends a `connected` event immediately using the `ImportProgressEvent` struct.
2. Pings Redis; if reachable, subscribes to pub/sub channel `import:progress:{jobID}` for real-time events. If Redis is unreachable, falls back to pure DB polling immediately.
3. Falls back to 3-second polling ticks (via `GetImportJob`) as a heartbeat, continuing to send `import_progress` events.
4. Sends `import_finished` and closes the connection when the job enters a terminal status (`completed`, `completed_with_errors`, or `failed`).
5. If the Redis pub/sub channel closes mid-stream (e.g., Redis goes down), the handler transitions to DB-only polling gracefully.

### Step 4: Retrieve Failures — `GET /api/v1/imports/staff/:id/failures`

```json
{
  "invitations": [
    {
      "id": "invitation-uuid",
      "email": "failed@somedomain.com",
      "full_name": "Doe",
      "phone": "+254700000000",
      "error_message": "stytch invite failed: invalid_email"
    }
  ]
}
```

Used by the correction panel to display and retry failed records.

---

## Frontend UX Walkthrough

The bulk import UI lives in `frontend/src/features/staff-import/` and is a **multi-view, stateful dialog/page** with draft persistence.

### Entry points

| Route                      | Role scope     | Mode                                     |
| -------------------------- | -------------- | ---------------------------------------- |
| `/admins/invitations/new`  | `SCHOOL_ADMIN` | Dialog (via `@modal` intercepting route) |
| `/admins/invitations`      | `SCHOOL_ADMIN` | Page                                     |
| `/nurses/invitations/new`  | `NURSE`        | Dialog (via `@modal` intercepting route) |
| `/nurses/invitations`      | `NURSE`        | Page                                     |
| `/finance/invitations/new` | `FINANCE`      | Dialog (via `@modal` intercepting route) |
| `/finance/invitations`     | `FINANCE`      | Page                                     |

### Views (dialog states)

```
ENTRY ──(valid rows)──> REVIEW ──(submit)──> PROGRESS ──(done)──> DONE
  │                                                  │
  │    (resume prompt)                               │
  └──(if draft exists)────┐                          │
                          │                          │
                   [Resume Draft?]          CORRECTION ──(resubmit)──> PROGRESS
```

#### 1. Entry View (`entry-view.tsx`)

Two tabs for data entry:

| Tab              | Component                | Description                                                          |
| ---------------- | ------------------------ | -------------------------------------------------------------------- |
| **Add Manually** | `manual-entry-panel.tsx` | Row-by-row form with email, first name, last name, phone (optional). |
| **Upload File**  | `file-upload-panel.tsx`  | CSV/XLSX file upload with client-side parsing.                       |

Features:

- **Draft persistence** — rows are saved to IndexedDB (`@/lib/db`) every time they're edited. On reload, shows a "Resume Draft?" prompt.
- **Client-side validation** — checks email structure (`@`, domain with `.`), required first/last name, duplicate email detection.
- Phone numbers are normalized to E.164 format using `libphonenumber-js` with Kenyan country code default.

#### 2. Review View (`review-view.tsx`)

Shows a **5-row sample** of the data for visual confirmation before submission.

Buttons:

- **Back** — return to entry view.
- **Submit {N} Invitations** — calls `useStartImport` mutation → `POST /api/v1/imports/staff`.

#### 3. Progress View (`import-progress-panel.tsx`)

Real-time progress display using the SSE stream or polling fallback:

```
Importing...
  ████████████░░░░░░░░  200 / 500 records
  ✅ Sent: 195
  ❌ Failed: 5
```

- Uses `createImportProgressStream` (RxJS Observable wrapping SSE) for real-time updates.
- Falls back to polling `GET /track/:id` every 3 seconds if SSE drops.
- On completion, transitions to **Done** (all succeeded) or **Correction** (some failed).

#### 4. Correction View (`correction-panel.tsx`)

Displays failed invitations for selective retry:

- Lists each failed record with its error message (e.g., "invalid_email").
- Allows editing email/name/phone before resubmit.
- Resubmits via a **new import job** linked to the original via `parentImportJobID`.
- The correction job re-runs only the failed records through Stage 2 (Stytch send).

#### 5. Done View

Simple success message: _"Import Complete — All invitations have been processed successfully."_

---

## Backend Architecture

### Files

| File                                                           | Purpose                                                                                                                                                                                                   |
| -------------------------------------------------------------- | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `backend/internal/imports/domain.go`                           | Models (`ImportJob`, `ImportStaffRecord`), sentinel errors, constants (`MaxRecordsPerImport`, `BatchSize`, `StytchConcurrency`, `StytchMaxRetries`, `InvitationTTL`, `TaskTimeout`), repository interface |
| `backend/internal/imports/handler.go`                          | HTTP handlers: StartImport, TrackImport, SSETrackImport, ListFailedInvitations. Includes `requireAuth` middleware and SSE body stream writer                                                              |
| `backend/internal/imports/handler_test.go`                     | Integration tests for all import endpoints                                                                                                                                                                |
| `backend/internal/imports/service.go`                          | Business logic: create job, enqueue Asynq task (with `TypeProcessImport`), get failures                                                                                                                   |
| `backend/internal/imports/service_test.go`                     | Unit tests for StartImport validation and job creation                                                                                                                                                    |
| `backend/internal/imports/worker.go`                           | Asynq task handler: ProcessImport (Stage 1 + Stage 2), `HandleError` (retry exhaustion), progress publishing (`publishProgress`, `publishFinished`)                                                       |
| `backend/internal/imports/worker_test.go`                      | Tests for worker processing and error handling                                                                                                                                                            |
| `backend/internal/imports/repository.go`                       | Pgx-backed Postgres operations: bulk CTE insert (`BulkInsertInvitations`), correction update (`BulkUpdateInvitations`), Stage 2 idempotency query (`GetPendingStage2Records`)                             |
| `backend/internal/imports/module.go`                           | fx dependency injection module                                                                                                                                                                            |
| `frontend/src/features/staff-import/`                          | React components (bulk-staff-import-dialog, entry-view, review-view, etc.)                                                                                                                                |
| `frontend/src/features/staff-import/hooks/use-staff-import.ts` | React Query hooks (`useStartImport`, `useTrackImport`, `useImportFailures`) + RxJS `createImportProgressStream` SSE Observable                                                                            |
| `frontend/src/features/staff-import/lib/validation.ts`         | Client-side email/phone validation (E.164, Kenyan country code default)                                                                                                                                   |
| `frontend/src/lib/api/imports.ts`                              | Frontend API functions: `startImport`, `trackImport`, `listFailedInvitations`, `createImportSSE`                                                                                                          |
| `frontend/src/lib/api/invitations.ts`                          | Frontend API functions: `listInvitationsByRole` (list-only, no create)                                                                                                                                    |

### Worker architecture

The bulk import uses **Asynq** for reliable async task processing with retry and timeout management:

```
POST /api/v1/imports/staff
        │
        ▼
   Service.StartImport()
        │
        ├── Create import_jobs row (Postgres)
        │
        └── Enqueue Asynq task (TypeProcessImport="imports:process") ──> Asynq Server (10 goroutines)
                                            │
                                            ▼
                                       Worker.ProcessImport()
                                            │
                                            ├── Stage 1: BulkInsertInvitations / BulkUpdateInvitations
                                            │             (batches of 200, CTE ON CONFLICT DO NOTHING)
                                            │
                                            ├── GetPendingStage2Records
                                            │    (only records w/o stytch_member_id, re-entry safe)
                                            │
                                            └── Stage 2: Stytch invite send
                                              (8 concurrent goroutines, semaphore pattern)
```

On retry exhaustion (3 attempts), `Worker.HandleError` (implementing `asynq.ErrorHandler`) sets the job status to `'failed'` with `completed_at` set, so it doesn't remain stuck as `'processing'`.

Key constants (all defined in `domain.go`):

| Constant              | Value      | Description                                                                        |
| --------------------- | ---------- | ---------------------------------------------------------------------------------- |
| `MaxRecordsPerImport` | 5,000      | Max records per import job                                                         |
| `BatchSize`           | 200        | Records per DB batch insert                                                        |
| `StytchConcurrency`   | 8          | Max concurrent Stytch API calls (buffered channel semaphore)                       |
| `StytchMaxRetries`    | 3          | Retry attempts for transient Stytch errors                                         |
| `InvitationTTL`       | 7 days     | How long pending invitations are valid (`expires_at = now + 7 days`)               |
| `TaskTimeout`         | 45 minutes | Asynq task timeout (accounts for 5000 records × 8 concurrent workers with retries) |

#### Task type reference

| Constant            | Value               | Queue      | MaxRetry | Timeout    |
| ------------------- | ------------------- | ---------- | -------- | ---------- |
| `TypeProcessImport` | `"imports:process"` | `critical` | 3        | 45 minutes |

### Temporary ID reconciliation

Each import record carries a **`temp_id`** — a UUID generated by the frontend. This is crucial for two reasons:

1. **DB-to-client reconciliation** — after bulk insert, the CTE returns a `map[temp_id]invitation_id` so the frontend can track which records succeeded and which were duplicates.
2. **Correction resubmit** — on correction, the `temp_id` _is_ the `invitation_id` (the DB primary key of the failed row), so the backend knows exactly which rows to update via `BulkUpdateInvitations`.

### Redis pub/sub for SSE

Progress events are published to Redis channel `import:progress:{jobID}` (constant `RedisChannelProgress`). The SSE endpoint subscribes to this channel for real-time delivery. If no pub/sub message arrives within 3 seconds, the SSE handler polls `GetImportJob` directly as a heartbeat. Redis failures are logged but never block processing — the SSE handler falls back to pure DB polling gracefully.

---

## Correction & Recovery

### When Stage 2 (Stytch) fails

If a Stytch invite email fails to send (permanent error like `invalid_email` or `blocked_domain`), the invitation row is marked `status = 'invite_failed'` with the error message. The import job completes with `status = 'completed_with_errors'`.

### Correction flow

1. User opens the correction panel, sees the failed records with error messages.
2. User edits the problematic data (e.g., corrects a typo in the email).
3. User clicks "Resubmit" → sends a new `POST /api/v1/imports/staff` with `parentImportJobID` set to the original job ID.
4. Backend detects the correction path (`parentImportJobID != ""`) and uses `BulkUpdateInvitations` instead of `BulkInsertInvitations`:
   - Updates existing invitation rows in-place (email, name, phone, role).
   - Resets `status = 'pending'`, `error_message = NULL`, `attempt_count = 0`.
   - Re-runs only the corrected records through Stage 2 (Stytch send).
5. The new import job has `parent_import_job_id` linking it to the original for traceability.

Duplicate guards are enforced at the database level:

- **`uq_invitations_active_email`** — partial unique index on `(tenant_id, school_id, email)` WHERE `status NOT IN ('expired', 'revoked')`.
- The bulk import CTE uses `ON CONFLICT ... DO NOTHING` against this index, skipping records where an active pending invite already exists.
- Correction resubmits bypass this guard since they update existing rows by ID.

---

## Database Schema

### `import_jobs` table

```sql
CREATE TABLE IF NOT EXISTS import_jobs (
    id                   UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id            UUID        NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    school_id            UUID        NOT NULL,
    role                 user_role   NOT NULL,
    created_by           UUID        REFERENCES users(id) ON DELETE SET NULL,
    status               TEXT        NOT NULL DEFAULT 'pending',
    total_records        INT         NOT NULL DEFAULT 0,
    processed_records    INT         NOT NULL DEFAULT 0,
    success_count        INT         NOT NULL DEFAULT 0,
    failed_count         INT         NOT NULL DEFAULT 0,
    parent_import_job_id UUID        NULL,
    created_at           TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    started_at           TIMESTAMPTZ NULL,
    completed_at         TIMESTAMPTZ NULL
);
```

### `import_job_failures` table

```sql
CREATE TABLE IF NOT EXISTS import_job_failures (
    id             BIGSERIAL   PRIMARY KEY,
    import_job_id  UUID        NOT NULL REFERENCES import_jobs(id) ON DELETE CASCADE,
    raw_payload    JSONB       NOT NULL,
    error_message  TEXT        NOT NULL,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
```

### `invitations` table (relevant columns)

```sql
CREATE TABLE IF NOT EXISTS invitations (
    id                  UUID              PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id           UUID              NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    school_id           UUID              NOT NULL,
    email               VARCHAR(255)      NOT NULL,
    role                user_role         NOT NULL,
    status              invitation_status NOT NULL DEFAULT 'pending',
    invited_by          UUID              REFERENCES users(id) ON DELETE SET NULL,
    token               TEXT              NOT NULL,
    expires_at          TIMESTAMPTZ       NOT NULL,
    accepted_at         TIMESTAMPTZ       NULL,
    full_name           VARCHAR(255)      NOT NULL,
    phone               VARCHAR(50)       NULL,
    registration_number VARCHAR(100)      NULL,
    stytch_member_id    VARCHAR(255)      NULL,
    import_job_id       UUID              NULL,
    error_message       TEXT              NULL,
    attempt_count       INT               NOT NULL DEFAULT 0,
    created_at          TIMESTAMPTZ       NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX IF NOT EXISTS uq_invitations_active_email
  ON invitations (tenant_id, school_id, email)
  WHERE status NOT IN ('expired', 'revoked');
```

### Invitation statuses (`invitation_status` enum)

| Status          | Meaning                                                 |
| --------------- | ------------------------------------------------------- |
| `pending`       | Created, awaiting the invitee to click the magic link   |
| `accepted`      | Invitee has clicked the link and created their account  |
| `expired`       | TTL (7 days) has passed since creation                  |
| `revoked`       | Manually revoked by an admin                            |
| `invite_failed` | Stytch email send permanently failed (bulk import only) |

---

## Error Handling

### Error codes

| Error            | HTTP | Scenario                              |
| ---------------- | ---- | ------------------------------------- |
| `unauthorized`   | 401  | No session cookie or invalid session  |
| `invalid_input`  | 400  | Missing/invalid fields in the request |
| `not_found`      | 404  | Job ID, tenant, or school not found   |
| `internal_error` | 500  | DB or Stytch API failure              |

### Bulk import validation errors (input stage)

The bulk import endpoint validates all records before creating a job. Errors are returned immediately as `400 Bad Request` with an `invalid_input` code:

| Error message                                                          | Cause                                 |
| ---------------------------------------------------------------------- | ------------------------------------- |
| `email is required for all records`                                    | Empty email field in any record       |
| `full_name is required for all records`                                | Empty full_name in any record         |
| `registration_number (TSC Number) is required for all teacher records` | Empty TSC Number when role is TEACHER |
| `invalid role: must be one of SCHOOL_ADMIN, TEACHER, NURSE, FINANCE`   | Role not in allowed set               |
| `at least one record is required`                                      | Empty records array                   |
| `maximum 5000 records per import`                                      | Record count exceeds limit            |

Duplicates (existing active invites) are counted as failed records post-ingestion, not rejected upfront.

### Bulk import failures (Stage 2)

Stytch invite failures are stored directly on the `invitations` row:

- `status = 'invite_failed'`
- `error_message` — contains the Stytch API error string
- `attempt_count` — number of retry attempts before giving up

### Permanent vs transient Stytch errors

The worker classifies Stytch errors to avoid retrying hopeless cases:

| Permanent (no retry)    | Transient (retry up to 3 times) |
| ----------------------- | ------------------------------- |
| `invalid_email`         | Rate-limit errors               |
| `email_invalid`         | Network timeouts                |
| `blocked_domain`        | Server errors (5xx)             |
| `domain_not_allowed`    | Unclassified errors             |
| `member_already_exists` |                                 |
| `not_found`             |                                 |

---

## TSC Number Promotion (TEACHER Role)

When a TEACHER invitation is accepted, the `registration_number` value stored in the `invitations` table is automatically promoted to the `users.tsc_number` column for the newly created user account.

### Flow

```
┌──────────────┐     ┌───────────────┐     ┌──────────────┐
│  Bulk Import  │     │  Invite       │     │  User        │
│  (worker)     │     │  Acceptance   │     │  Profile     │
│               │     │  (auth svc)   │     │              │
│ Writes        │     │               │     │              │
│ registration_ │────>│ Reads inv.    │────>│ tsc_number   │
│ number to     │     │ registration_ │     │ is populated │
│ invitations   │     │ number        │     │              │
└──────────────┘     └───────────────┘     └──────────────┘
```

### Implementation details

1. **Backend Service** (`auth/service.go` — `AcceptInvite`): The `Invitation.RegistrationNumber` from the DB query is mapped to `CreateInvitedUserSessionArgs.TSCNumber`.

2. **Backend Repository** (`auth/repository.go` — `CreateInvitedUserSession`): When `args.Role == "TEACHER"` and `args.TSCNumber != ""`, the INSERT statement includes the `tsc_number` column:

   ```sql
   INSERT INTO users (email, tenant_id, full_name, external_auth_id, tsc_number)
   VALUES ($1, $2, $3, $4, $5, $6)
   ```

   For non-TEACHER roles, the standard 5-column insert is used (no TSC number).

3. **Correction Resubmit** (`imports/repository.go` — `BulkUpdateInvitations`): The `registration_number` field is now included in the CTE update so corrected records persist the TSC number.

### Database constraints

- The `users.tsc_number` column has a **partial unique index**: `CREATE UNIQUE INDEX idx_users_tsc_number ON users (tsc_number) WHERE tsc_number IS NOT NULL`.
- This means two teachers cannot share the same TSC number.
- The column accepts `NULL` for non-TEACHER users.

---

## FAQ

### "How many people can I invite at once?"

Up to **5,000 records** per bulk import job. For smaller batches, you can create invitation records individually via the bulk import endpoint with a single record.

### "Do invited users need to create an account?"

Yes. When an invitee clicks the magic link in the Stytch invite email, they go through the **invite acceptance flow** (`GET /api/auth/invite/callback?token=...`), which creates their user account, session, and membership in one transaction. See [auth-flow.md](auth-flow.md) for details.

### "What happens if the Stytch invite email doesn't arrive?"

If Stytch returns a permanent error, the invitation is marked `status = 'invite_failed'` with the error message. Open the correction panel, fix the data, and resubmit. The system will re-attempt the Stytch send.

If Stytch accepted the request but the email doesn't arrive in the inbox, check spam/junk. The invitation record has `stytch_member_id` populated, meaning Stytch considers it sent.

### "Can I invite someone who was already invited but hasn't accepted?"

No — the `uq_invitations_active_email` unique index prevents duplicate pending invitations for the same email, school, and tenant. Revoke the old invitation first, or wait for it to expire (7 days).

### "What happens after 7 days?"

Pending invitations expire naturally. The `expires_at` check in `WHERE expires_at > NOW()` prevents expired invitations from being accepted. Expired invitations show in the listing as `status = 'expired'` (when `?expired=true` is passed).

### "Can I invite a user who is already a member of this school?"

No. If the email already has an active `memberships` row for the school, the system rejects the invitation. If they were previously a member but are now `is_active = false`, contact the platform team to re-activate.

### "How long does a bulk import take?"

For 5,000 records:

- Stage 1 (DB insert): ~1–2 seconds (batch CTE).
- Stage 2 (Stytch send): ~2–5 minutes (8 concurrent goroutines, each Stytch call is ~200–500ms).

The SSE stream updates in real time so the user can watch progress.

### "Can I cancel a running import?"

Not yet. If you started an import with bad data, wait for it to complete, then use the correction panel to fix failed records.

---

## Related files

| File                                                                | Purpose                                                                          |
| ------------------------------------------------------------------- | -------------------------------------------------------------------------------- |
| `backend/internal/imports/domain.go`                                | Models, constants, sentinel errors, repository interface                         |
| `backend/internal/imports/handler.go`                               | HTTP handlers for all import endpoints (StartImport, TrackImport, SSE, Failures) |
| `backend/internal/imports/handler_test.go`                          | Integration tests for import endpoints                                           |
| `backend/internal/imports/service.go`                               | Business logic + Asynq task enqueue                                              |
| `backend/internal/imports/service_test.go`                          | Unit tests for StartImport validation                                            |
| `backend/internal/imports/worker.go`                                | Async processing (Stage 1 + Stage 2) + HandleError                               |
| `backend/internal/imports/worker_test.go`                           | Tests for worker processing and error handling                                   |
| `backend/internal/imports/repository.go`                            | Pgx DB operations including CTE bulk insert, correction update, Stage 2 query    |
| `backend/internal/members/domain.go`                                | Invitation model (`members` domain)                                              |
| `frontend/src/features/staff-import/`                               | Full React component tree for bulk import UX                                     |
| `frontend/src/features/staff-import/hooks/use-staff-import.ts`      | React Query hooks + RxJS SSE Observable                                          |
| `frontend/src/lib/api/imports.ts`                                   | Frontend API functions for import endpoints                                      |
| `frontend/src/lib/api/invitations.ts`                               | Frontend API functions for invitation listing (list only)                        |
| `backend/internal/database/migrations/000001_initial_schema.up.sql` | `import_jobs`, `import_job_failures`, `invitations` table definitions            |
