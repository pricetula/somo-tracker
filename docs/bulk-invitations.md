# Bulk Staff Invitations — Admin, Nurse & Finance

> **Last updated:** 2026-06-23
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

| Approach | Max records | Invite email sent? | Processing | Best for |
|---|---|---|---|---|
| **Quick invite** — `POST /api/v1/invitations` | ~100 | No (manual Stytch send on demand) | Synchronous | Small batches, individual invites |
| **Bulk import** — `POST /api/v1/imports/staff` | 5,000 | ✅ Yes (Stage 2 Stytch send) | Async (Asynq worker) | Large CSVs, hundreds of staff members |

### Supported roles

Only three staff roles can be invited through these flows:

| Role | Allowed access | Typical use |
|---|---|---|
| `SCHOOL_ADMIN` | `/admin`, `/admins`, `/dashboard`, `/settings`, `/schools`, `/docs` | Deputy head, senior teachers, operations |
| `NURSE` | `/dashboard`, `/docs` | School nurse, healthcare staff |
| `FINANCE` | `/dashboard`, `/docs` | Bursar, accounting staff |

> **Note:** `TEACHER` and `SYSTEM_ADMIN` use separate invitation flows (see [Teacher invitations](TODO) and system admin provisioning).

### Who can invite

Any **authenticated user** with a valid session can create invitations for their school. The invitee's role is assigned at invitation time and the user's session must belong to the same tenant/school.

---

## Quick Invite Flow

### `POST /api/v1/invitations` — Create invitations (per-invite roles)

This endpoint lets you create one or more invitation records with individual roles per invitee.

#### Request

```json
{
  "invites": [
    {
      "email": "nurse@school.edu",
      "first_name": "Grace",
      "last_name": "Mwangi",
      "role": "NURSE"
    },
    {
      "email": "bursar@school.edu",
      "first_name": "James",
      "last_name": "Ochieng",
      "role": "FINANCE"
    }
  ]
}
```

#### Response

```json
{
  "sent": 2,
  "failed": 0,
  "errors": []
}
```

#### Backend processing

1. **Auth check** — validates `somo_sid` cookie via `auth.Service.GetSession`.
2. **Resolve active school** — queries `memberships` for the user's highest-privilege active school.
3. **For each invite:**
   - Validates email is not empty.
   - Validates role is one of `TEACHER`, `NURSE`, `FINANCE`, `SCHOOL_ADMIN`.
   - Checks for existing active membership (same email + school → skip).
   - Checks for existing pending invite (same email + school → skip).
   - Inserts `invitations` row with `status = 'pending'`, `expires_at = now + 7 days`.
   - Token is the invitation's own ID.
4. Returns `sent` / `failed` counts.

> ⚠️ This endpoint **does not** send Stytch invite emails. The invitation record is persisted locally. The invitee must be sent a magic link through a separate mechanism (or re-sent via the member invite endpoint).

### `POST /api/v1/members/invite` — Bulk invite with shared role + Stytch email

This endpoint sends a Stytch invite email **and** persists the invitation record.

#### Request

```json
{
  "role": "FINANCE",
  "invites": [
    { "email": "bursar@school.edu", "first_name": "James", "last_name": "Ochieng" },
    { "email": "accountant@school.edu", "first_name": "Faith", "last_name": "Kiprop" }
  ]
}
```

#### Backend processing

1. Auth check + resolve active school + resolve Stytch org ID.
2. For each invite:
   - Validates membership/pending-duplicate guards.
   - Creates `invitations` row in Postgres.
   - Calls `StytchAdapter.InviteMemberByEmail` to send the invite magic link.
   - On success, stores `stytch_member_id` on the invitation row.
   - On Stytch failure, logs a warning but **keeps the invitation** (it can be retried later).

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
     │                 │                   │  (concurrent)  │
     │                 │                   │───────────────>│
     │                 │                   │                │
     │   SSE: progress │                   │                │
     │<────────────────│<──────────────────│                │
     │                 │                   │                │
     │   SSE: finished │                   │                │
     │<────────────────│<──────────────────│                │
```

### Endpoints

| Method | Path | Purpose |
|---|---|---|
| `POST` | `/api/v1/imports/staff` | Start a bulk import job |
| `GET` | `/api/v1/imports/staff/track/:id` | Poll job status |
| `GET` | `/api/v1/imports/staff/track/:id/sse` | Real-time SSE progress stream |
| `GET` | `/api/v1/imports/staff/:id/failures` | Fetch failed invitations for correction |

### Step 1: Start Import — `POST /api/v1/imports/staff`

#### Request

```json
{
  "role": "NURSE",
  "records": [
    {
      "temp_id": "a1b2c3d4-...",
      "email": "nurse1@school.edu",
      "first_name": "Grace",
      "last_name": "Mwangi",
      "phone": "+254712345678",
      "registration_number": "NRS-001"
    }
  ]
}
```

- `role` — must be `SCHOOL_ADMIN`, `NURSE`, or `FINANCE`.
- `records` — array of records, max **5,000**.
- `temp_id` — client-generated UUID for reconciliation (each row must have a unique one).
- `phone` and `registration_number` are optional.

#### Validation rules (backend)

| Field | Rule |
|---|---|
| `email` | Required, must be non-empty |
| `first_name` | Required, must be non-empty |
| `last_name` | Required, must be non-empty |
| `role` | Must be one of `SCHOOL_ADMIN`, `NURSE`, `FINANCE` |
| Record count | Between 1 and 5,000 |

#### Response (202 Accepted)

```json
{
  "import_job_id": "uuid-string",
  "status": "pending",
  "total": 42
}
```

### Step 2: Processing (Async)

After accepting the request, the backend:

1. **Creates an `import_jobs` row** with `status = 'pending'`.
2. **Enqueues an Asynq task** on the `critical` queue with `MaxRetry(3)`.
3. Returns immediately with `202 Accepted`.

The Asynq worker then processes the import in **two stages**:

#### Stage 1 — Bulk DB Ingestion

- Processes records in **batches of 200**.
- Uses a **CTE (Common Table Expression)** to bulk-insert invitations with `ON CONFLICT ... DO NOTHING` on the unique index `(tenant_id, school_id, email)` where status is not expired/revoked.
- Returns a `map[temp_id]invitation_id` for reconciliation + a list of duplicates.
- Duplicates (existing active invites) are counted as failures.
- On correction resubmit (when `parentImportJobID` is set), uses `BulkUpdateInvitations` instead to update existing rows.

#### Stage 2 — Stytch Email Dispatch

- Sends Stytch invite emails with **bounded concurrency** (8 goroutines).
- Each call has **3 retries** with exponential backoff (2s, 4s, 6s).
- Permanent Stytch errors (invalid email, blocked domain, member already exists) are detected and **not retried**.
- Failed records at this stage are marked `status = 'invite_failed'` with the error message.
- Progress is published to Redis pub/sub after each record.

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

| Status | Meaning |
|---|---|
| `pending` | Job created, not yet picked up by worker |
| `processing` | Worker is actively processing |
| `completed` | All records processed successfully |
| `completed_with_errors` | Some records failed (check failures endpoint) |

#### SSE — `GET /api/v1/imports/staff/track/:id/sse`

Server-Sent Events endpoint for real-time progress:

```json
// Event: connected (sent immediately)
{ "type": "connected", "import_job_id": "uuid" }

// Event: progress (sent per-record during Stage 2)
{ "type": "import_progress", "status": "processing", "processed_records": 42, ... }

// Event: finished (sent once on completion)
{ "type": "import_finished", "status": "completed", ... }
```

The SSE stream:
1. Sends a `connected` event immediately.
2. Listens to Redis pub/sub channel `import:progress:{jobID}` for real-time events.
3. Falls back to 3-second polling ticks as a heartbeat.
4. Sends `import_finished` and closes the connection when the job completes.

### Step 4: Retrieve Failures — `GET /api/v1/imports/staff/:id/failures`

```json
{
  "invitations": [
    {
      "id": "invitation-uuid",
      "email": "failed@somedomain.com",
      "first_name": "John",
      "last_name": "Doe",
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

| Route | Role scope | Mode |
|---|---|---|
| `/admins/invitations/new` | `SCHOOL_ADMIN` | Dialog (via `@modal` intercepting route) |
| `/admins/invitations` | `SCHOOL_ADMIN` | Page |
| `/nurses/invitations/new` | `NURSE` | Dialog (via `@modal` intercepting route) |
| `/nurses/invitations` | `NURSE` | Page |
| `/finance/invitations/new` | `FINANCE` | Dialog (via `@modal` intercepting route) |
| `/finance/invitations` | `FINANCE` | Page |

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

| Tab | Component | Description |
|---|---|---|
| **Add Manually** | `manual-entry-panel.tsx` | Row-by-row form with email, first name, last name, phone (optional). |
| **Upload File** | `file-upload-panel.tsx` | CSV/XLSX file upload with client-side parsing. |

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

Simple success message: *"Import Complete — All invitations have been processed successfully."*

---

## Backend Architecture

### Files

| File | Purpose |
|---|---|
| `backend/internal/imports/domain.go` | Models (`ImportJob`, `ImportStaffRecord`), sentinel errors, constants |
| `backend/internal/imports/handler.go` | HTTP handlers: StartImport, TrackImport, SSETrackImport, ListFailedInvitations |
| `backend/internal/imports/service.go` | Business logic: create job, enqueue task, get failures |
| `backend/internal/imports/worker.go` | Asynq task handler: ProcessImport (Stage 1 + Stage 2) |
| `backend/internal/imports/repository.go` | Pgx-backed Postgres operations |
| `backend/internal/imports/module.go` | fx dependency injection module |
| `frontend/src/features/staff-import/` | React components (bulk-staff-import-dialog, entry-view, review-view, etc.) |
| `frontend/src/features/staff-import/hooks/use-staff-import.ts` | React Query hooks + SSE Observable |
| `frontend/src/features/staff-import/lib/validation.ts` | Client-side email/phone validation |

### Worker architecture

The bulk import uses **Asynq** for reliable async task processing:

```
POST /api/v1/imports/staff
        │
        ▼
   Service.StartImport()
        │
        ├── Create import_jobs row (Postgres)
        │
        └── Enqueue Asynq task ──────> Asynq Server (10 goroutines)
                                           │
                                           ▼
                                      Worker.ProcessImport()
                                           │
                                           ├── Stage 1: BulkInsertInvitations (batches of 200)
                                           │
                                           └── Stage 2: Stytch invite send (8 concurrent)
```

Key configuration:

| Constant | Value | Description |
|---|---|---|
| `MaxRecordsPerImport` | 5,000 | Max records per import job |
| `BatchSize` | 200 | Records per DB batch insert |
| `StytchConcurrency` | 8 | Max concurrent Stytch API calls |
| `StytchMaxRetries` | 3 | Retry attempts for transient Stytch errors |
| `InvitationTTL` | 7 days | How long pending invitations are valid |

### Temporary ID reconciliation

Each import record carries a **`temp_id`** — a UUID generated by the frontend. This is crucial for two reasons:

1. **DB-to-client reconciliation** — after bulk insert, the response maps `temp_id → invitation_id` so the frontend can track which records succeeded.
2. **Correction resubmit** — on correction, the `temp_id` *is* the `invitation_id` (the DB primary key of the failed row), so the backend knows exactly which rows to update.

### Redis pub/sub for SSE

Progress events are published to Redis channel `import:progress:{jobID}`. The SSE endpoint subscribes to this channel for real-time delivery. If no pub/sub message arrives within 3 seconds, the SSE handler polls Postgres directly as a heartbeat.

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

### Duplicate guards

- **`uq_invitations_active_email`** — partial unique index on `(tenant_id, school_id, email)` WHERE `status NOT IN ('expired', 'revoked')`.
- Both the quick invite endpoint and the bulk import check this constraint and skip records where an active pending invite already exists.

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
    first_name          VARCHAR(255)      NULL,
    last_name           VARCHAR(255)      NULL,
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

| Status | Meaning |
|---|---|
| `pending` | Created, awaiting the invitee to click the magic link |
| `accepted` | Invitee has clicked the link and created their account |
| `expired` | TTL (7 days) has passed since creation |
| `revoked` | Manually revoked by an admin |
| `invite_failed` | Stytch email send permanently failed (bulk import only) |

---

## Error Handling

### Error codes

| Error | HTTP | Scenario |
|---|---|---|
| `unauthorized` | 401 | No session cookie or invalid session |
| `invalid_input` | 400 | Missing/invalid fields in the request |
| `not_found` | 404 | Job ID, tenant, or school not found |
| `internal_error` | 500 | DB or Stytch API failure |

### Per-invite errors (response body)

When creating invitations via `POST /api/v1/invitations` or `POST /api/v1/members/invite`, failures are reported per-invite in the response:

```json
{
  "sent": 1,
  "failed": 2,
  "errors": [
    { "email": "dupe@school.edu", "error": "user is already a member of this school" },
    { "email": "", "error": "email is required" }
  ]
}
```

| Error message | Cause |
|---|---|
| `email is required` | Empty email field |
| `user is already a member of this school` | Email has an active `memberships` row |
| `pending invitation already exists for this email` | Active `invitations` row with `status = 'pending'` |
| `invalid role: must be ...` | Role not in allowed set |
| `internal error checking existing membership` | DB query failed |
| `failed to create invitation` | DB insert failed |

### Bulk import failures (Stage 2)

Stytch invite failures are stored directly on the `invitations` row:

- `status = 'invite_failed'`
- `error_message` — contains the Stytch API error string
- `attempt_count` — number of retry attempts before giving up

### Permanent vs transient Stytch errors

The worker classifies Stytch errors to avoid retrying hopeless cases:

| Permanent (no retry) | Transient (retry up to 3 times) |
|---|---|
| `invalid_email` | Rate-limit errors |
| `email_invalid` | Network timeouts |
| `blocked_domain` | Server errors (5xx) |
| `domain_not_allowed` | Unclassified errors |
| `member_already_exists` | |
| `not_found` | |

---

## FAQ

### "How many people can I invite at once?"

Up to **5,000 records** per bulk import job. For smaller batches (under 100), use the quick invite endpoint instead.

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

| File | Purpose |
|---|---|
| `backend/internal/imports/domain.go` | Models, constants, sentinel errors |
| `backend/internal/imports/handler.go` | HTTP handlers for all import endpoints |
| `backend/internal/imports/service.go` | Business logic + Asynq task enqueue |
| `backend/internal/imports/worker.go` | Async processing (Stage 1 + Stage 2) |
| `backend/internal/imports/repository.go` | Pgx DB operations including CTE bulk insert |
| `backend/internal/members/domain.go` | Invitation model (`members` domain) |
| `backend/internal/members/service.go` | Quick invite and bulk invite logic |
| `backend/internal/members/handler.go` | HTTP handlers for `/api/v1/invitations` and `/api/v1/members/invite` |
| `frontend/src/features/staff-import/` | Full React component tree for bulk import UX |
| `frontend/src/features/staff-import/hooks/use-staff-import.ts` | React Query hooks + SSE Observable |
| `frontend/src/lib/api/imports.ts` | Frontend API functions for import endpoints |
| `frontend/src/lib/api/invitations.ts` | Frontend API functions for invitation endpoints |
| `backend/internal/database/migrations/000001_initial_schema.up.sql` | `import_jobs`, `import_job_failures`, `invitations` table definitions |
