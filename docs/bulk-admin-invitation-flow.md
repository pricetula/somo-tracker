# Bulk Admin Invitation Flow — Documentation

## Overview

The bulk admin invitation feature allows an authenticated SCHOOL_ADMIN to submit up to 10,000 invitation rows (email + full name) in a single request. The backend validates the rows, deduplicates them, creates a bulk job with per-row items stored in the database, and enqueues Asynq tasks to process each row asynchronously against the Stytch API. Progress is broadcast to the frontend via Redis pub/sub (SSE).

---

## 1. Request Flow (Line by Line)

### 1.1 Extract Session Context

The `HandleInvites` handler first reads three values from the Fiber context locals:

- `active_school_id` — the school the admin is acting in
- `tenant_id` — the tenant (Somo account)
- `user_id` — the authenticated admin's user ID

These are parsed as UUIDs. If any parse fails (missing context or wrong type), the request is rejected with `401 Unauthorized` and the canonical error body:

```json
{
  "code": "unauthorized",
  "message": "missing session context",
  "errors": {}
}
```

### 1.2 Authorization Guard

The service `UserHasAdminRole` queries `school_memberships` for the given school and user with `is_active = true`.

- If the DB query fails → `500 Internal Error` (`internal_error`)
- If no membership row exists (`pgx.ErrNoRows`) → treated as no role
- If role is not `"ADMIN"` → `403 Forbidden` (`forbidden`)

### 1.3 Parse Request Body

The request body is bound into:

```go
type BulkInvitationRequest struct {
  Invitations []InvitationRow `json:"invitations"`
}
```

Malformed JSON → `400 Bad Request` (`bad_request`, errors.body = ["malformed json"]).

### 1.4 Validate Array Constraints

- `len(Invitations) == 0` → `400 Bad Request` (`validation_failed`, field `invitations` = ["required"])
- `len(Invitations) > 10000` → `400 Bad Request` (`validation_failed`, field `invitations` = ["max 10000"])

### 1.5 Idempotency Check

The `Idempotency-Key` header is required. Missing → `400 Bad Request` (`bad_request`, field `Idempotency-Key` = ["required"]).

If a job with the same `(tenant_id, idempotency_key)` already exists, the handler returns `202 Accepted` with the existing job's details — no new job is created, no duplicate rows are inserted.

The database enforces this with a unique constraint on `(tenant_id, idempotency_key)` on `bulk_jobs`.

### 1.6 Row-Level Validation

Each row is validated:

- `email` trimmed; must be non-empty and pass `net/mail.ParseAddress`
- `full_name` trimmed; must be non-empty

All validation errors are collected per field with the row index:

```json
{
  "code": "validation_failed",
  "message": "some invitation rows are invalid",
  "errors": {
    "invitations[2].email": ["invalid email format"],
    "invitations[5].full_name": ["required"]
  }
}
```

Status: `400 Bad Request`.

### 1.7 Deduplication

Valid rows are deduplicated by lowercased email address. Duplicates are silently skipped (first occurrence wins). If all rows were duplicates → `400 Bad Request` (`bad_request`, field `invitations` = ["no valid rows after deduplication"]).

### 1.8 Build Items

Each valid row becomes a `services.InvitationItem`:

```go
type InvitationItem struct {
  ID       uuid.UUID
  Email    string
  FullName string
  Role     string // always "ADMIN"
}
```

### 1.9 Create Bulk Job + Items Atomically

`CreateBulkJobWithItems` opens a transaction and, in one atomic unit:

1. Inserts a row into `bulk_jobs` with:
   - `job_type = "ADMIN_INVITATION"`
   - `status = "QUEUED"`
   - `total_records = len(items)`
   - `metadata = {"source":"bulk_invitation"}`
2. Inserts a row into `bulk_job_items` for each item with `status = "PENDING"` and the payload as JSON
3. Inserts `bulk_job_outbox` rows in batches of 40 for reliable enqueueing (`ON CONFLICT (job_id, batch_index) DO NOTHING`)

If any step fails, the transaction rolls back and the handler returns `500 Internal Error` (`internal_error`).

### 1.10 Enqueue Asynq Tasks

The handler enqueues one Asynq task per batch of 40 items:

- Task type: `admin:invitation:batch`
- Queue: `admin_invitation`
- Task ID: `<job_id>_<batch_index>` (idempotent enqueue)
- Payload: `{"job_id": "...", "batch_index": 0, "item_ids": ["..."]}`

Enqueue failures are logged and the request still returns `202 Accepted` (graceful degradation).

### 1.11 Publish Initial Progress

If Redis is available, the handler publishes `{"status":"QUEUED","total":N}` to channel `bulk_progress_<job_id>`.

### 1.12 Response

`202 Accepted`:

```json
{
  "job_id": "...",
  "status": "QUEUED",
  "total_records": 123,
  "message": "Bulk invitation job registered successfully"
}
```

---

## 2. Processing Mechanics

### 2.1 Batch Processing

`AdminInvitationProcessor.ProcessTask` handles `admin:invitation:batch` tasks:

1. Unmarshals the task payload; bad `job_id` → task fails (Asynq retries the whole task)
2. Logs batch info
3. Marks the job as `PROCESSING` if still `QUEUED`
4. Processes each item in the batch with limited concurrency (10 workers)
5. After all items finish, derives and writes the job's terminal status
6. Publishes progress

### 2.2 Per-Item Processing

For each item ID in the batch:

1. `TryAcquireItem` atomically moves the item from `PENDING` to `PROCESSING` (conditional update on `status='PENDING'`, incrementing `attempt_count`). If the item is already being processed or terminal, it is skipped (returns `acquired = false`).
2. The item's payload JSON is unmarshaled. If it fails → item is marked `FAILED` with `last_error = "invalid payload"`.
3. The job is fetched to get the tenant ID.
4. The Stytch organization ID is fetched for the tenant.
5. `stytchCli.InviteMember(email, fullName, "ADMIN", orgID)` is called.
6. On success: item → `SUCCEEDED`, result stored (`stytch_invite_id`, `stytch_member_id`), job counters incremented.
7. On error: `ClassifyStytchError` categorizes the error and the item is marked accordingly (see §3).

### 2.3 Job Status Derivation

After each batch completes, `deriveJobStatus` scans all items of the job:

- If any item is `PENDING` or `PROCESSING` → job stays `PROCESSING`
- Else if any item is `DEFERRED` → job → `COMPLETED_WITH_ERRORS`
- Else (all terminal) → if `FailedCount > 0` → `COMPLETED_WITH_ERRORS`, else `COMPLETED`

### 2.4 Progress Broadcasting

After each item and after each batch, `publishProgress` reads the job's counters and publishes to Redis channel `bulk_progress_<job_id>`:

```json
{
  "status": "QUEUED",
  "succeeded": 50,
  "failed": 2,
  "deferred": 1,
  "total": 100
}
```

The frontend subscribes via SSE (`GET /events/:job_id`) and receives `event: progress` messages. Heartbeats are sent every 15s if no state changes.

---

## 3. Error Classification (Stytch)

`stytch.Client.ClassifyStytchError` categorizes Stytch errors into four classes:

| Category | Conditions | Result |
|---|---|---|
| Retryable | HTTP 429, "rate_limit", "too many", HTTP >= 500, "timeout", "network", "connection" | Item → `DEFERRED`, `last_error` = reason (`rate_limited`, `server_error`, `network_error`, `timeout`) |
| Permanent | HTTP 400, "invalid_email", "bad_request", "invalid" | Item → `FAILED`, `last_error` = reason (`invalid_email`) |
| Duplicate | "duplicate_user_email", HTTP 409, error type contains "duplicate" | Item → `SUCCEEDED` with result note "member already exists" (idempotent) |
| Unknown | Anything else | Treated as retryable with reason `unknown: <msg>` |

### Retry Budget

`maxAttempts = 5`. On each attempt, `attempt_count` is incremented when the item is acquired. When `attempt_count >= maxAttempts`, a permanent failure archives the item to `bulk_job_dead_letter` before marking it `FAILED`.

Note: the worker unit tests contain a TODO to confirm whether the retry budget is 3 or 5 attempts — the implementation uses `maxAttempts = 5`.

### Retry Mechanism

- `DEFERRED` items are reprocessed by the `admin:invitation:retry` task.
- `RetryFailed` handler (`POST /retry-failed/:job_id`) loads all `FAILED`/`DEFERRED` items and enqueues one retry task with their IDs.
- The retry task re-enqueues them and `TryAcquireItem` atomically moves them back to `PROCESSING`.

---

## 4. Job-Level States

| State | Meaning |
|---|---|
| `QUEUED` | Job created, items pending, tasks may not have been enqueued yet |
| `PROCESSING` | At least one item is `PENDING` or `PROCESSING` |
| `COMPLETED` | All items `SUCCEEDED` |
| `COMPLETED_WITH_ERRORS` | All items terminal but at least one is `FAILED` or `DEFERRED` |

### Item-Level States

| State | Meaning |
|---|---|
| `PENDING` | Row created, not yet processed |
| `PROCESSING` | Currently being processed (acquired by a worker) |
| `SUCCEEDED` | Invitation created successfully in Stytch |
| `DEFERRED` | Transient error; will be retried |
| `FAILED` | Permanent failure; may be retried manually |

---

## 5. Error Response Contract (HTTP)

All non-2xx responses from the handler follow the canonical contract:

```json
{
  "code": "snake_case_error_code",
  "message": "human readable message",
  "errors": { "field_name": ["Specific field validation message"] }
}
```

### Handler Error Responses

| Scenario | Status | Code | Message | Errors |
|---|---|---|---|---|
| Missing/wrong session context | 401 | `unauthorized` | missing session context | `{}` |
| Role check DB failure | 500 | `internal_error` | failed to verify permissions | `{}` |
| Not an admin | 403 | `forbidden` | insufficient permissions | `{}` |
| Malformed JSON body | 400 | `bad_request` | invalid request body | `body: ["malformed json"]` |
| Empty invitations array | 400 | `validation_failed` | invitations array is required | `invitations: ["required"]` |
| >10000 rows | 400 | `validation_failed` | exceeds maximum of 10000 rows | `invitations: ["max 10000"]` |
| Missing Idempotency-Key | 400 | `bad_request` | Idempotency-Key header is required | `Idempotency-Key: ["required"]` |
| Validation errors in rows | 400 | `validation_failed` | some invitation rows are invalid | `invitations[i].email/full_name` |
| No valid rows after dedup | 400 | `bad_request` | no valid invitation rows after deduplication | `invitations: ["no valid rows"]` |
| DB job creation failure | 500 | `internal_error` | failed to create bulk job | `{}` |
| Invalid job_id param | 400 | `bad_request` | invalid job_id | `{}` |
| Job not found / wrong tenant | 404 | `not_found` | job not found | `{}` |
| Retry load DB failure | 500 | `internal_error` | failed to load retry items | `{}` |
| Retry enqueue failure | 500 | `internal_error` | retry enqueue failed | `{}` |

### Worker-Level Errors (logged, task fails)

- `unmarshal payload` — malformed Asynq task payload
- `bad job_id` — invalid job UUID in payload
- `bad item id` — invalid UUID in item_ids (item skipped)
- `failed to acquire item` — DB error acquiring item
- `failed to unmarshal item payload` — invalid item payload JSON → item `FAILED`
- `job not found for tenant` — job missing → item `FAILED`
- `stytch org not found for tenant` — tenant has no Stytch org → item `FAILED`
- `failed to update item status to SUCCEEDED` — DB write error after successful Stytch call
- `failed to increment job counters` — DB write error after success
- `failed to update item status to DEFERRED` / `FAILED` — DB write error
- `failed to archive dead letter item` — DLQ insert error after max attempts
- `failed to fetch pending outbox` — outbox reconciliation failure
- `failed to mark outbox enqueued` — outbox status update failure
- `failed to get items for status derivation` — status derivation failure

### Outbox / Enqueue Mechanics

The `bulk_job_outbox` table is a durable outbox pattern: the service writes outbox rows when creating the job, and the `ProcessOutboxTask` reconciles pending outbox entries (marks them `ENQUEUED`) so tasks can be enqueued reliably. Currently the handler enqueues tasks directly and the outbox path is a fallback.

---

## 6. Data Model

### `bulk_jobs`

| Field | Meaning |
|---|---|
| `id` | Job UUID |
| `job_type` | `ADMIN_INVITATION` |
| `idempotency_key` | Unique per tenant |
| `school_id` | School the admin belongs to |
| `tenant_id` | Tenant owning the job |
| `created_by` | Admin user who submitted |
| `status` | QUEUED / PROCESSING / COMPLETED / COMPLETED_WITH_ERRORS |
| `total_records` | Number of rows submitted |
| `succeeded_count` | Denormalized count |
| `failed_count` | Denormalized count |
| `deferred_count` | Denormalized count |
| `metadata` | JSON (`{"source":"bulk_invitation"}`) |
| `created_at` / `updated_at` | Timestamps |

### `bulk_job_items`

| Field | Meaning |
|---|---|
| `id` | Item UUID |
| `job_id` | Parent job |
| `row_index` | Original row position |
| `payload` | JSON (`{"email","full_name"}`) |
| `status` | PENDING / PROCESSING / SUCCEEDED / DEFERRED / FAILED |
| `attempt_count` | Number of acquisition attempts |
| `result` | JSON on success (invite/member IDs) or error |
| `last_error` | Last error reason string |
| `created_at` / `updated_at` | Timestamps |

### `bulk_job_outbox`

| Field | Meaning |
|---|---|
| `id` | Outbox entry UUID |
| `job_id` | Parent job |
| `batch_index` | Which batch of 40 |
| `item_ids` | UUID[] of item IDs in the batch |
| `status` | PENDING / ENQUEUED |

### `bulk_job_dead_letter`

Archived items that hit `maxAttempts` (5) without success. Fields: `item_id`, `job_id`, `payload`, `last_error`, `attempt_count`.

---

## 7. Concurrency & Safety Notes

- `TryAcquireItem` uses a conditional atomic update (`WHERE status='PENDING'`) so only one worker per item can process it.
- Batch processing uses a semaphore of 10 concurrent goroutines to avoid overwhelming Stytch.
- Job counters are incremented per item; concurrent increments are atomic in the DB.
- Idempotency is guaranteed by the unique constraint on `(tenant_id, idempotency_key)` and the `ON CONFLICT` update in job creation.
- Task IDs (`<job_id>_<batch_index>`) make Asynq enqueue idempotent.

---

## 8. Known TODOs / Open Questions

1. Retry budget in tests is documented as 3 attempts, implementation uses `maxAttempts = 5`.
2. Circuit breaker (gobreaker) wiring is not yet implemented in the worker — tests reference TODOs.
3. `CreateBulkJob`