# Bulk Invitation — Architecture & Data Flow

> **Last updated:** 2026-07-09
> **Scope:** Frontend `features/invitations/components/bulk-invite/` ↔ Backend `internal/invitations/` + `internal/imports/`
> **Design principle:** Reuse the student import engine (chunking, progress tracking, error reporting, progress polling) for the bulk invitation flow. Only the Importer implementation and frontend form are new.

---

## Table of Contents

1. [Overview](#1-overview)
2. [Key Differences from Student Import](#2-key-differences-from-student-import)
3. [Architecture Diagram](#3-architecture-diagram)
4. [Data Flow: End-to-End](#4-data-flow-end-to-end)
5. [Frontend Components](#5-frontend-components)
6. [Backend Components](#6-backend-components)
7. [Key API Endpoints](#7-key-api-endpoints)
8. [Request/Response Contracts](#8-requestresponse-contracts)
9. [Error Handling & Failure Types](#9-error-handling--failure-types)
10. [Database Schema (additions)](#10-database-schema-additions)
11. [StaffInviteImporter — Validate / ResolveReferences / InsertOne](#11-staffinviteimporter--validate--resolvereferences--insertone)
12. [Idempotency & Retry Semantics](#12-idempotency--retry-semantics)
13. [Module Wiring](#13-module-wiring)
14. [Edge Cases & Race Conditions](#14-edge-cases--race-conditions)

---

## 1. Overview

Bulk invitation allows admins to invite multiple staff members (admins, teachers, other roles) to a school in one action. The flow takes inspiration from the student import:

- **Frontend:** Admin types email addresses (plus optional names) into a simple table form. All rows are sent in a single POST request.
- **Backend:** Uses the existing `imports` engine — the same job/staging/chunk/failure tables, Asynq workers, and progress polling that student import uses.
- **Stytch integration:** Each row results in a Stytch `CreateMember` + `InviteMemberByEmail` call inside the chunk worker.
- **Validation:** Each email is checked against the existing `users` table and `invitations` table to avoid duplicate invitations.

### What we take from student import

| Feature | Student Import | Bulk Invitation |
|---------|---------------|-----------------|
| Frontend: "send all rows in one POST" | ✅ | ✅ (same pattern) |
| Backend: Asynq chunked processing | ✅ | ✅ (same engine) |
| Backend: job status + polling | ✅ | ✅ (same GET /imports/:job_id) |
| Backend: per-row failure reporting | ✅ | ✅ (same GET /imports/:job_id/failures) |
| Backend: one-active-job-per-school | ✅ | ✅ (same DB partial index) |
| Frontend: ImportProgress component | ✅ | ✅ (reused) |
| Frontend: cancellation | ✅ | ✅ (same POST /imports/:job_id/cancel) |

### What's different

| Aspect | Student Import | Bulk Invitation |
|--------|---------------|-----------------|
| Input | CSV/Excel file or manual table | Manual email table only |
| Validation | Full student schema (name, gender, DOB, etc.) | Email format + uniqueness check |
| DB writes | `cbc_students` + `cbc_student_enrollments` | `invitations` table |
| External API | None | Stytch `CreateMember` + `InviteMemberByEmail` |
| Metadata | academic_term_id, academic_year_id | None (just role) |
| Chunk action | INSERT student + enrollment | Create Stytch member + send invite + INSERT invitation |

---

## 2. Key Differences from Student Import

### 2.1 Stytch calls inside InsertOne

Unlike student import which only does DB writes, the StaffInviteImporter makes external API calls to Stytch within `InsertOne`. This introduces a **best-effort + idempotent retry** pattern:

1. `CreateMember(ctx, orgID, email, name)` — creates the member in Stytch. On retry, Stytch returns the existing member ID rather than erroring.
2. `InviteMemberByEmail(ctx, orgID, email, name, redirectURL)` — sends the invite email. On retry, Stytch sends another email (acceptable — the user gets a duplicate invite).
3. `INSERT INTO invitations (...)` — local DB record. Within the savepoint, so if it fails, Stytch calls are rolled back in effect (redundant on retry, but harmless).

### 2.2 No file import path

Unlike student import, bulk invitations have **only a manual entry path**. There is no CSV/Excel file upload, column mapping, or class resolution. The form is a simple table with email (required) and name (optional).

### 2.3 Single validation check

The only pre-submit validation is:
- **Email format** — basic format check (`/^[^\s@]+@[^\s@]+\.[^\s@]+$/`)
- **Email uniqueness** — checked against `users` and `invitations` tables (checking both prevents inviting someone who already has an account, or who already has a pending invitation)

There is no `check-duplicates` endpoint needed — the uniqueness check happens in `ResolveReferences` during async processing.

### 2.4 Per-job role

The `role` is set at the job level (stored in `import_jobs.role`), not per row. All invitations in one batch share the same role. This matches the existing `import_jobs` schema which already has a `role` column with a CHECK constraint requiring it for `STAFF_INVITE` jobs.

### 2.5 No idempotency key

For simplicity, the initial implementation does not use idempotency keys. If the user needs to retry, they simply submit again (which creates a new job — the old job already completed or is in progress, so the one-active-job-per-school constraint applies). Idempotency can be added later.

---

## 3. Architecture Diagram

```
┌────────────────────────────────────────────────────────────────────────────┐
│                           FRONTEND                                          │
│                                                                             │
│  BulkInviteForm (orchestrator)                                              │
│  ├── BulkInviteTable     ← table with email + optional name rows           │
│  ├── RoleSelector        ← pick role (admin, teacher, etc.)                │
│  ├── Submit button       ← POST /api/v1/staff/invite                      │
│  └── ImportProgress      ← reused from student import (polling)           │
│                                                                             │
│  API layer (src/lib/api/invitations.ts):                                    │
│  ├── submitBulkInvite()     → POST /api/v1/staff/invite                   │
│  ├── getImportJob()         → GET  /api/v1/imports/:job_id (reused)       │
│  └── getImportFailures()    → GET  /api/v1/imports/:job_id/failures (reused)│
└────────────────────────────────────────────────────────────────────────────┘
                             │  HTTP
                             ▼
┌────────────────────────────────────────────────────────────────────────────┐
│                           BACKEND                                           │
│                                                                             │
│  internal/invitations/handler.go                                            │
│  └── BulkInvite()                                                           │
│      ├── Parse InviteRequest { role, rows: [{ email, full_name? }] }       │
│      ├── Validate row count limit (MaxImportRows)                           │
│      └── Call imports.Service.CreateJob()                                   │
│                                                                             │
│  internal/imports/service.go (reused)                                       │
│  └── CreateJob()                                                            │
│      ├── Create import_jobs row (status: processing)                        │
│      ├── Write all rows to import_job_staging                              │
│      ├── Split into chunks of 100 (ChunkSize)                              │
│      ├── Enqueue Asynq tasks (1 per chunk)                                 │
│      └── Return { job_id, total_records, total_chunks, status }            │
│                                                                             │
│  Asynq Worker (internal/imports/module.go — reused)                         │
│  └── ProcessChunk()                                                         │
│      ├── 1. Claim chunk (at-most-once)                                     │
│      ├── 2. Load staging rows for chunk range                              │
│      ├── 3. Call StaffInviteImporter.Validate() → schema checks            │
│      ├── 4. Call StaffInviteImporter.ResolveReferences() → duplicate check │
│      ├── 5. Call StaffInviteImporter.BulkInsert() → fails, forces fallback │
│      ├── 6. Call StaffInviteImporter.InsertOne() per row:                  │
│      │      ├── CreateMember in Stytch (idempotent on retry)               │
│      │      ├── InviteMemberByEmail in Stytch (sends email)               │
│      │      └── INSERT into invitations table                              │
│      ├── 7. Insert failures into import_job_failures                       │
│      └── 8. AtomicChunkCompletion() → update job counters                  │
│                                                                             │
│  internal/invitations/importer.go (NEW — StaffInviteImporter)               │
│  └── Implements imports.Importer                                           │
│      ├── Validate() → email format                                         │
│      ├── ResolveReferences() → check existing users + invitations          │
│      ├── BulkInsert() → returns error (forces savepoint fallback)          │
│      └── InsertOne() → Stytch member + invite + DB insert                 │
│                                                                             │
│  GET /api/v1/imports/:job_id (reused)                                      │
│  GET /api/v1/imports/:job_id/failures (reused)                             │
└────────────────────────────────────────────────────────────────────────────┘
```

---

## 4. Data Flow: End-to-End

### Step 1: Submit (Frontend)

The BulkInviteForm collects:
- `role`: string (e.g., `"admin"`, `"teacher"`)
- `rows`: array of `{ email: string, full_name?: string }`

On submit, calls:

```typescript
const result = await submitBulkInvite({
    role: "admin",
    rows: inviteRows,
});
// result = { job_id, total_records, total_chunks, status: "processing" }
```

### Step 2: Parent captures job (Frontend)

BulkInviteForm captures `jobId` and `totalRecords` from the response, switches from the form view to `<ImportProgress>`.

### Step 3: Poll for progress (Frontend)

The shared `<ImportProgress>` component polls `GET /api/v1/imports/:job_id` with backoff (1.5s → 3s → 10s) until terminal status.

### Step 4: Backend creates job (Backend)

`BulkInvite` handler in `internal/invitations/handler.go`:
1. Extracts `tenant_id`, `school_id`, `user_id` from request context
2. Validates request (role required, non-empty rows, row limit)
3. Builds `json.RawMessage[]` from the invite rows
4. Calls `imports.Service.CreateJob()` with `JobType = "STAFF_INVITE"` and `Role = body.Role`
5. Returns `ImportResponse` immediately (async processing)

### Step 5: Async processing (Backend)

Asynq worker picks up each chunk and calls `StaffInviteImporter` methods:

1. **Validate()** — checks each row:
   - Email is non-empty after trim
   - Email matches basic format

2. **ResolveReferences()** — for each row:
   - Checks `users` table: does this email already exist for this tenant?
   - Checks `invitations` table: is there already a pending invitation for this email?
   - If either check matches → fails the row with `DUPLICATE_EMAIL` error
   - Otherwise → resolves the row (injects tenant/school context)

3. **BulkInsert()** — returns error (forces per-row savepoint fallback)

4. **InsertOne()** — per row inside a savepoint:
   - Creates Stytch member via `stytch.CreateMember(ctx, orgID, email, name)`
     - On retry: Stytch may return "member already exists" — handled by treating the returned member ID as success
   - Sends invite email via `stytch.InviteMemberByEmail(ctx, orgID, email, name, redirectURL)`
   - Inserts `invitations` row with status `'pending'`
   - On any failure → savepoint rollback → staging row stays `'pending'` → retry
   - On success → mark staging row `'succeeded'` → release savepoint

5. Failures go to `import_job_failures`

6. `AtomicChunkCompletion()` updates counters

### Step 6: Completion (Frontend)

Same as student import — `<ImportProgress>` shows green/amber banner with counts and failure details. "Retry failed" button re-submits failed rows.

---

## 5. Frontend Components

### 5.1 `BulkInviteForm` (`bulk-invite-form.tsx`)

The parent orchestrator. Manages:
- `role`: selected role from a dropdown
- `rows`: array of `InviteRow[]` with email + optional name
- `activeJob`: `{ jobId, totalRecords } | null`

When `activeJob` is set, renders `<ImportProgress>` instead of the form.

**Props:**
- `isDialogVersion: boolean` — whether rendered inside a modal

**Active-job redirect on mount:**
- On mount, calls `GET /api/v1/schools/:school_id/imports/active` using current `school_id`.
- If an active job is returned, immediately sets `activeJob` and renders `<ImportProgress>`.

### 5.2 `InviteRow` type

```typescript
interface InviteRow {
    email: string;
    full_name?: string;
}
```

### 5.3 `ImportProgress` (reused)

The same shared component from student import, reused as-is:
- `getImportJob()` → `GET /api/v1/imports/:job_id`
- `getImportFailures()` → `GET /api/v1/imports/:job_id/failures`
- Polling backoff, stalled-job messaging, cancel button, Done/Retry buttons

### 5.4 Frontend validation

| Rule | Value | Action |
|------|-------|--------|
| Email required | Non-empty after trim | Hard error |
| Email format | `/^[^\s@]+@[^\s@]+\.[^\s@]+$/` | Hard error if not matching |
| Duplicate in batch | Two rows with same email | Hard error: "Duplicate email — also used in row N" |
| Row limit | 5000 (shared `MaxImportRows`) | Blocks submit if exceeded |

### 5.5 API client (`src/lib/api/invitations.ts`)

```typescript
// POST /api/v1/staff/invite
export async function submitBulkInvite(body: {
    role: string;
    rows: { email: string; full_name?: string }[];
}): Promise<ImportResponse> {
    return api.post<ImportResponse>("/api/v1/staff/invite", body);
}
```

Reuses `getImportJob`, `getImportFailures`, `getActiveImportJob`, `cancelImportJob` from `src/lib/api/imports.ts`.

### 5.6 File structure

```
src/features/invitations/
├── components/
│   ├── bulk-invite/
│   │   ├── bulk-invite-form.tsx       ← Parent orchestrator
│   │   ├── bulk-invite-table.tsx      ← Email table (rows with email + optional name)
│   │   ├── invite-row-input.tsx       ← Single row component
│   │   └── validation-utils.ts        ← validateInviteRow(), detectDuplicateEmails()
│   └── ...
src/lib/api/
├── imports.ts                         ← getImportJob, getImportFailures, etc. (reused)
└── invitations.ts                     ← submitBulkInvite (new)
```

---

## 6. Backend Components

### 6.1 `internal/invitations/domain.go` — New types for bulk invite

```go
// InviteRow is a single row in a bulk invitation request.
type InviteRow struct {
    Email    string  `json:"email"`
    FullName *string `json:"full_name,omitempty"`
}

// BulkInviteRequest is the request body for POST /api/v1/staff/invite.
type BulkInviteRequest struct {
    Role string      `json:"role"`
    Rows []InviteRow `json:"rows"`
}

// InviteResponse is returned immediately after creating the bulk invite job.
// Reuses the same shape as student import's ImportResponse.
type InviteResponse struct {
    JobID        string `json:"job_id"`
    TotalRecords int    `json:"total_records"`
    TotalChunks  int    `json:"total_chunks"`
    Status       string `json:"status"`
    IsReplay     bool   `json:"is_replay,omitempty"`
}
```

### 6.2 `internal/invitations/handler.go` — New BulkInvite handler

```go
// BulkInvite handles POST /api/v1/staff/invite.
func (h *Handler) BulkInvite(c *fiber.Ctx) error {
    // 1. Extract tenant/school/user from context
    // 2. Parse request body (role + rows)
    // 3. Validate role is provided
    // 4. Validate non-empty rows
    // 5. Validate row count ≤ MaxImportRows
    // 6. Serialize rows as json.RawMessage[]
    // 7. Call imports.Service.CreateJob() with JobType = STAFF_INVITE
    // 8. Handle in-progress / duplicate / error cases
    // 9. Return InviteResponse (201)
}
```

### 6.3 `internal/invitations/importer.go` — StaffInviteImporter (NEW)

Implements `imports.Importer` interface. See [Section 11](#11-staffinviteimporter--validate--resolvereferences--insertone).

### 6.4 `internal/invitations/service.go` — Extended service

Adds methods needed by the importer:

```go
// CheckEmailExistsInUsers checks if any of the given emails already exist
// in the users table for this tenant.
func (s *Service) CheckEmailExistsInUsers(ctx context.Context, tenantID string, emails []string) ([]string, error)

// CheckEmailExistsInInvitations checks if any of the given emails already have
// a pending invitation for this school.
func (s *Service) CheckEmailExistsInInvitations(ctx context.Context, schoolID string, emails []string) ([]string, error)

// InsertInvitation creates a new invitation record within a transaction.
func (s *Service) InsertInvitation(ctx context.Context, tx pgx.Tx, params CreateInvitationParams) error

// GetStytchOrgID returns the Stytch org ID for the tenant (delegates to auth repository).
func (s *Service) GetStytchOrgID(ctx context.Context, tenantID string) (string, error)
```

### 6.5 `internal/invitations/repository.go` — Extended repository

Adds queries for the bulk invite flow:

```go
// CheckExistingEmailsInUsers returns the subset of emails that exist in the users table.
func (r *PgRepository) CheckExistingEmailsInUsers(ctx context.Context, tenantID string, emails []string) ([]string, error)

// CheckExistingEmailsInInvitations returns the subset of emails that have pending invitations.
func (r *PgRepository) CheckExistingEmailsInInvitations(ctx context.Context, schoolID string, emails []string) ([]string, error)

// InsertInvitation inserts a single invitation row.
func (r *PgRepository) InsertInvitation(ctx context.Context, tx pgx.Tx, params InsertInvitationParams) error

// GetStytchOrgID retrieves the Stytch org ID for a tenant.
func (r *PgRepository) GetStytchOrgID(ctx context.Context, tenantID string) (string, error)
```

---

## 7. Key API Endpoints

| Method | Path | Purpose | Response |
|--------|------|---------|----------|
| `POST` | `/api/v1/staff/invite` | Create bulk invitation job | `{ job_id, total_records, total_chunks, status }` |
| `GET` | `/api/v1/imports/:job_id` | Poll job status (reused) | `ImportJob` (full state) |
| `GET` | `/api/v1/imports/:job_id/failures` | Get failure details (reused) | `{ failures: [...], total: number }` |
| `POST` | `/api/v1/imports/:job_id/cancel` | Cancel running import (reused) | `ImportJob` (status: `cancelling`) |
| `GET` | `/api/v1/schools/:school_id/imports/active` | Active job check (reused) | `{ active: bool, job: ImportJob \| null }` |

Only `POST /api/v1/staff/invite` is new. The other endpoints are reused from the student import engine.

---

## 8. Request/Response Contracts

### POST /api/v1/staff/invite

**Request:**
```json
{
  "role": "admin",
  "rows": [
    { "email": "alice@example.com", "full_name": "Alice Wanjiku" },
    { "email": "bob@example.com" },
    { "email": "carol@example.com", "full_name": "Carol Mwangi" }
  ]
}
```

**Response (201 — new job created):**
```json
{
  "job_id": "uuid",
  "total_records": 3,
  "total_chunks": 1,
  "status": "processing",
  "is_replay": false
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

**Error codes:**

| Code | Status | Meaning |
|------|--------|---------|
| `invalid_input` | 400 | Malformed body, missing role, invalid email |
| `import_row_limit_exceeded` | 400 | Row count exceeds `MaxImportRows` (5000) |
| `import_already_in_progress` | 409 | Another import job active for this school |
| `no_tenant_stytch_org` | 500 | Tenant has no Stytch org configured (should never happen in normal flow) |

---

## 9. Error Handling & Failure Types

### Additional failure types for bulk invitation

These types are added alongside the existing student import failure types:

| Error Type | Source | Description |
|------------|--------|-------------|
| `DUPLICATE_EMAIL` | `ResolveReferences()` | Email already exists in `users` table OR has a pending invitation for this school |
| `INVALID_EMAIL_FORMAT` | `Validate()` | Email does not match basic format |
| `STYTCH_API_ERROR` | `InsertOne()` | Stytch `CreateMember` or `InviteMemberByEmail` failed |
| `INVITATION_INSERT_FAILED` | `InsertOne()` | DB insert of invitation record failed |

### Error response shape

Same canonical contract as the rest of the API:
```json
{
  "code": "snake_case_error_code",
  "message": "human readable message",
  "errors": { "field_name": ["specific error"] }
}
```

### Per-row failure messages

| Failure Type | Example Message |
|-------------|----------------|
| `DUPLICATE_EMAIL` | "Email alice@example.com already exists for this school" |
| `INVALID_EMAIL_FORMAT` | "Email 'not-an-email' is not a valid email address" |
| `STYTCH_API_ERROR` | "Could not create member in authentication provider" |
| `INVITATION_INSERT_FAILED` | "Could not save invitation record" |

---

## 10. Database Schema (additions)

No new tables are needed. The bulk invitation uses:

- **`import_jobs`** — already supports `STAFF_INVITE` job type and `role` column
- **`import_job_staging`** — stores raw invite rows as JSONB
- **`import_job_chunks`** — chunk tracking (same as student import)
- **`import_job_failures`** — per-row failure records (same structure)
- **`invitations`** — existing table for invitation records (created by `InsertOne`)

### Need to add new failure types to the enum

The `import_failure_type` DB enum needs to be extended with:

```sql
ALTER TYPE import_failure_type ADD VALUE 'DUPLICATE_EMAIL';
ALTER TYPE import_failure_type ADD VALUE 'INVALID_EMAIL_FORMAT';
ALTER TYPE import_failure_type ADD VALUE 'STYTCH_API_ERROR';
ALTER TYPE import_failure_type ADD VALUE 'INVITATION_INSERT_FAILED';
```

### Need to add email uniqueness index on invitations

To prevent race conditions where two concurrent chunks try to invite the same email:

```sql
CREATE UNIQUE INDEX IF NOT EXISTS uq_invitations_school_email
    ON invitations (school_id, email)
    WHERE status = 'pending';
```

This allows the same email to be invited again for a different school, and allows re-inviting after the first invitation was accepted/expired.

### `import_failure_type` enum extension

The Go-side `ImportFailureType` constants in `internal/imports/domain.go` need:

```go
const (
    ImportFailureDuplicateEmail     ImportFailureType = "DUPLICATE_EMAIL"
    ImportFailureInvalidEmailFormat ImportFailureType = "INVALID_EMAIL_FORMAT"
    ImportFailureStytchAPIError     ImportFailureType = "STYTCH_API_ERROR"
    ImportFailureInviteInsertFailed ImportFailureType = "INVITATION_INSERT_FAILED"
)
```

And the frontend `ImportFailureType` type needs the same values.

---

## 11. StaffInviteImporter — Validate / ResolveReferences / InsertOne

### 11.1 Augmented row (after ResolveReferences)

```go
type augmentedInviteRow struct {
    Email        string `json:"email"`
    FullName     string `json:"full_name,omitempty"`
    TenantID     string `json:"tenant_id"`
    SchoolID     string `json:"school_id"`
    Role         string `json:"role"`
    StytchOrgID  string `json:"stytch_org_id"`
    StagingRowID string `json:"staging_row_id,omitempty"`
}
```

### 11.2 Validate()

```go
func (si *StaffInviteImporter) Validate(ctx context.Context, tenantID, schoolID uuid.UUID, raw []json.RawMessage) ([]imports.ValidatedRow, []imports.RowFailure) {
    for i, rawData := range raw {
        var row InviteRow
        json.Unmarshal(rawData, &row)

        // Trim whitespace
        row.Email = strings.TrimSpace(row.Email)

        // Required email
        if row.Email == "" {
            failures = append(failures, RowFailure{...ErrorType: INVALID_EMAIL_FORMAT, Message: "email is required"})
            continue
        }

        // Basic email format check
        if !emailRegex.MatchString(row.Email) {
            failures = append(failures, RowFailure{...ErrorType: INVALID_EMAIL_FORMAT, ...})
            continue
        }

        validated = append(validated, ValidatedRow{RawData: rawData})
    }
    return validated, failures
}
```

### 11.3 ResolveReferences()

Two-phase check against existing records for this tenant/school:

```go
func (si *StaffInviteImporter) ResolveReferences(...) {
    // Phase 1: Unmarshal all rows, collect emails
    // Phase 2: Query once for all emails:
    //   - SELECT email FROM users WHERE tenant_id = $1 AND email = ANY($2)
    //   - SELECT email FROM invitations WHERE school_id = $1 AND email = ANY($2) AND status = 'pending'
    // Phase 3: Build augmented rows, rejecting duplicates

    for _, p := range parsed {
        if existingSet[p.row.Email] {
            failures = append(failures, RowFailure{...ErrorType: DUPLICATE_EMAIL, Message: fmt.Sprintf("Email %s already exists for this school", p.row.Email)})
            continue
        }

        // Build augmented row with tenant_id, school_id, role, stytch_org_id
        aug := augmentedInviteRow{...}
        resolved = append(resolved, ValidatedRow{RawData: augData, StagingRowID: p.stagingID})
    }
    return resolved, failures
}
```

### 11.4 BulkInsert()

Returns error to force per-row savepoint fallback:

```go
func (si *StaffInviteImporter) BulkInsert(ctx context.Context, tx pgx.Tx, rows []imports.ValidatedRow) (int, error) {
    return 0, fmt.Errorf("staff invite requires per-row Stytch API calls")
}
```

### 11.5 InsertOne()

```go
func (si *StaffInviteImporter) InsertOne(ctx context.Context, tx pgx.Tx, row imports.ValidatedRow) error {
    var aug augmentedInviteRow
    json.Unmarshal(row.RawData, &aug)

    // Step 1: Create Stytch member
    memberID, err := si.stytch.CreateMember(ctx, aug.StytchOrgID, aug.Email, aug.FullName)
    if err != nil {
        // On retry, Stytch may return "member already exists" — treat as success
        var stytchErr stytcherror.Error
        if errors.As(err, &stytchErr) && stytchErr.ErrorType == "member_already_exists" {
            // Extract existing member ID from error metadata if available
            memberID = extractMemberID(stytchErr)
        } else {
            return &imports.ImportError{Type: imports.ImportFailureStytchAPIError, Message: "Could not create member in authentication provider"}
        }
    }

    // Step 2: Send invite email
    _, err = si.stytch.InviteMemberByEmail(ctx, aug.StytchOrgID, aug.Email, aug.FullName, si.redirectURL)
    if err != nil {
        return &imports.ImportError{Type: imports.ImportFailureStytchAPIError, Message: "Could not send invitation email"}
    }

    // Step 3: Insert invitation record in DB (within savepoint)
    err = si.repo.InsertInvitation(ctx, tx, InsertInvitationParams{
        Email:          aug.Email,
        FullName:       aug.FullName,
        TenantID:       aug.TenantID,
        SchoolID:       aug.SchoolID,
        Role:           aug.Role,
        Status:         "pending",
        StytchMemberID: memberID,
        ExpiresAt:      time.Now().Add(7 * 24 * time.Hour), // 7-day expiry
    })
    if err != nil {
        if isUniqueConstraintViolation(err) {
            return &imports.ImportError{Type: imports.ImportFailureDuplicateEmail, Message: fmt.Sprintf("Email %s already has a pending invitation", aug.Email)}
        }
        return &imports.ImportError{Type: imports.ImportFailureInviteInsertFailed, Message: "Could not save invitation record"}
    }

    return nil
}
```

### 11.6 Dependencies

The `StaffInviteImporter` needs:

- `auth.IdentityProvider` (Stytch adapter) — for `CreateMember` + `InviteMemberByEmail`
- `InvitationRepository` — for `InsertInvitation`, `CheckExistingEmailsInUsers`, `CheckExistingEmailsInInvitations`
- `config.Config` — for `StytchRedirectURL`
- `auth.Repository` — for `GetTenantStytchOrgID`

---

## 12. Idempotency & Retry Semantics

### 12.1 Asynq redelivery (automatic)

If a chunk worker crashes mid-processing:
- The chunk transitions back from `'processing'` to a retryable state after Asynq's visibility timeout
- Only staging rows with `status = 'pending'` are loaded on retry
- Rows that were successfully processed (staging row marked `'succeeded'`) are skipped

### 12.2 Stytch retry safety

- **`CreateMember`** — On retry, Stytch returns `member_already_exists`. The importer handles this by extracting the existing member ID and continuing with `InviteMemberByEmail`.
- **`InviteMemberByEmail`** — On retry, Stytch sends another invite email. This is acceptable — the user receives a duplicate email but this does not cause data corruption.

### 12.3 DB insert retry safety

The `invitations` table has a unique index `uq_invitations_school_email` that prevents inserting a duplicate pending invitation for the same school+email combination. If a retry attempts to insert a row that was already created by a prior (partially-committed) attempt, the savepoint will fail on the unique constraint, the error will be caught by the savepoint handler, and the row will be re-marked as `'succeeded'`.

### 12.4 Partial chunk failure

If one row in a chunk fails (e.g., Stytch API is down for that specific call), the savepoint for that row is rolled back, other rows in the chunk continue processing. The failed row's staging status remains `'pending'` and will be retried in a subsequent chunk delivery.

---

## 13. Module Wiring

### 13.1 `internal/invitations/module.go`

```go
package invitations

import (
    "go.uber.org/fx"
    "somotracker/backend/internal/auth"
    "somotracker/backend/internal/imports"
)

var Module = fx.Module("invitations",
    fx.Provide(
        fx.Annotate(NewRepository, fx.As(new(Repository))),
        NewService,
        NewHandler,
        NewStaffInviteImporter,
    ),
    // Wire auth repository into service (for GetStytchOrgID)
    fx.Invoke(func(svc *Service, authRepo auth.Repository) {
        svc.SetAuthRepository(authRepo)
    }),
    // Wire Stytch adapter into importer
    fx.Invoke(func(imp *StaffInviteImporter, idp auth.IdentityProvider) {
        imp.SetStytchAdapter(idp)
    }),
    // Wire import service into handler
    fx.Invoke(func(h *Handler, impSvc *imports.Service) {
        h.SetImportService(impSvc)
    }),
    // Register the StaffInvite importer
    fx.Invoke(registerStaffInviteImporter),
)

func registerStaffInviteImporter(sii *StaffInviteImporter) {
    imports.RegisterImporter(sii)
}
```

### 13.2 Route registration

Add the bulk invite endpoint to the existing invitations handler:

```go
func (h *Handler) RegisterRoutes(router fiber.Router) {
    invitations := router.Group("/api/v1")
    invitations.Post("/staff/invite", middleware.RequireAuth, h.BulkInvite)
    invitations.Get("/invitations", middleware.RequireAuth, h.ListInvitations)
}
```

Note: The route is `/api/v1/staff/invite` (not `/api/v1/invitations/invite`) to follow the pattern of `/api/v1/students/import`. The RESTful resource is "staff" and the action is "invite".

---

## 14. Edge Cases & Race Conditions

### 14.1 Same email submitted in two concurrent chunks

If two chunks in the same job both contain the same email (which shouldn't happen — within-batch duplicates are caught by frontend validation), the second chunk's `InsertOne` will hit the `uq_invitations_school_email` unique constraint. The error is caught by the savepoint handler and reported as `DUPLICATE_EMAIL`.

### 14.2 Email invited in a separate job while first job is still processing

Both jobs will call `CreateMember` and `InviteMemberByEmail` for the same email. Stytch handles this gracefully (member already exists, duplicate invite email sent). The second job's `ResolveReferences` will not see the first job's invitation (it's not yet committed), so both jobs will proceed. The second job's `InsertOne` will hit the unique constraint and fail as a `DUPLICATE_EMAIL` error. This is acceptable — the user gets two invite emails, but only one invitation record persists.

### 14.3 Email invited manually while bulk import is running

Same as above — the manual creation creates an invitation record. The import's `InsertOne` will hit the unique constraint and fail that row. The import will need to be retried for that specific email (or excluded).

### 14.4 Tenant has no Stytch org ID

Should never happen in normal flow (tenants are created with a Stytch org at registration). If it does, the `ResolveReferences` step will fail all rows with a `BUSINESS_RULE_VIOLATION` error.

### 14.5 Stytch API is temporarily down

If Stytch is down during chunk processing:
- `CreateMember` fails → savepoint rollback → row stays pending → Asynq retries (up to 3 times)
- `InviteMemberByEmail` fails → savepoint rollback → same retry pattern
- If all retries exhausted → row is recorded as a permanent failure (`STYTCH_API_ERROR`)
- Admin can click "Retry failed" to re-submit only the failed rows in a new job

### 14.7 Invitation records management

- **Invitation expiry:** Set to 7 days from creation. Expired invitations can be detected and re-invited.
- **Stytch member lifecycle:** Once a member is created in Stytch, they exist in the org even if they never accept the invite. Re-inviting the same email finds the existing member and sends another invite email.
- **User acceptance:** When a user accepts the invitation (clicks the magic link, creates an account), the `invitations` row status transitions to 'accepted'. This is handled by the existing invite acceptance flow in the auth domain.

### 14.8 Job completion with all-email-failures

If all rows in the job fail at the `ResolveReferences` stage (e.g., all emails already exist), the job completes immediately with `status: 'completed_with_errors'`, `success_count: 0`, `failed_count: N`. No Stytch API calls are made. The frontend shows the failure list with DUPLICATE_EMAIL messages.
