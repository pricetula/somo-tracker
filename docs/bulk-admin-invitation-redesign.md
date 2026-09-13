# Bulk Admin Invitation — Ground-Up Redesign Plan

## Goal

Replace the current over-engineered bulk invitation pipeline with a minimal, observable, and reliable design that still meets the product requirements:
* School admin can upload up to 10,000 admin invites in one request
* Idempotent submission via Idempotency-Key
* Async processing with progress visibility
* Clear error classification and retry for transient failures
* No dead weight: outbox not used, no per-item Redis chatter, no separate dead letter table

## Current Pain Points

* Outbox table written but handler enqueues directly → dual-write, no guarantee
* Per-item Redis publish → 10k+ pub/sub messages per job
* Four tables: bulk_jobs, bulk_job_items, bulk_job_outbox, bulk_job_dead_letter
* maxAttempts drift: code =5, tests assume 3
* Circuit breaker wired in tests but not in production code
* TryAcquireItem + per-item attempt_count adds locking complexity

## Target Architecture

### Principles
* Single source of truth for enqueue: DB transaction + Asynq enqueue in the same request, relying on Asynq idempotency via TaskID
* Batch-oriented processing, not per-item locking
* Progress is batched, not per-item
* Failures are first-class statuses on the item row
* No separate outbox or dead letter tables

### Data Model

**bulk_jobs**
* id uuid PK
* job_type text = 'ADMIN_INVITATION'
* idempotency_key text
* tenant_id uuid
* school_id uuid
* created_by uuid
* status text: QUEUED, PROCESSING, COMPLETED, COMPLETED_WITH_ERRORS
* total_records int
* succeeded_count int
* failed_count int
* deferred_count int
* created_at timestamptz
* updated_at timestamptz
* Unique index: (tenant_id, idempotency_key)

**bulk_job_items**
* id uuid PK
* job_id uuid FK
* row_index int
* email text
* full_name text
* status text: PENDING, PROCESSING, SUCCEEDED, FAILED, DEFERRED
* attempt_count int default 0
* result jsonb nullable
* last_error text nullable
* stytch_invite_id text nullable
* stytch_member_id text nullable
* created_at timestamptz
* updated_at timestamptz

No outbox table. No dead letter table.

### API Contract

POST /admin/invitations/bulk
Headers: Idempotency-Key required, Authorization
Body:
{
  "invitations": [
    {"email": "...", "full_name": "..."}
  ]
}
Response 202 Accepted:
{
  "job_id": "...",
  "status": "QUEUED",
  "total_records": 123
}
Validation errors 400 with canonical error body.

GET /admin/invitations/jobs/:job_id
Returns job + counters.

GET /admin/invitations/jobs/:job_id/items?status=&limit=&offset=
Paginated items for UI.

POST /admin/invitations/jobs/:job_id/retry
Re-enqueues FAILED/DEFERRED items.

GET /admin/invitations/jobs/:job_id/events
SSE stream for progress. Publishes on batch completion only.

### Processing Flow

1. Handler
* Extract session, authorize ADMIN role
* Validate Idempotency-Key present
* Check existing job by (tenant_id, idempotency_key) → return existing if found
* Validate rows: email format, full_name non-empty, max 10k
* Dedupe by lowercased email, first wins
* Create job + items in a single transaction
  - bulk_jobs insert with status QUEUED
  - bulk_job_items bulk insert with status PENDING
* Enqueue Asynq tasks: one task per 40 items
  - Task type: admin:invitation:batch
  - TaskID = job_id_batch_index for idempotency
  - Payload = {job_id, batch_index, item_ids}
* Publish initial progress to Redis once
* Return 202

2. Worker: admin:invitation:batch
* Load job, ensure tenant has Stytch org
* Update job status to PROCESSING if still QUEUED
* Load items for batch where status IN (PENDING, DEFERRED)
* For each item:
  - Update status to PROCESSING, increment attempt_count
  - Call Stytch InviteMember
  - Classify error
    - Duplicate → SUCCEEDED
    - Retryable → DEFERRED, increment deferred_count
    - Permanent → FAILED, increment failed_count
    - Success → SUCCEEDED, increment succeeded_count
* After batch: derive job status
  - If any items PENDING/PROCESSING → PROCESSING
  - Else if any DEFERRED → COMPLETED_WITH_ERRORS
  - Else if failed_count >0 → COMPLETED_WITH_ERRORS
  - Else → COMPLETED
* Publish progress to Redis once per batch

3. Retry
* Retry endpoint loads FAILED/DEFERRED items, builds new batch tasks, enqueues with same TaskID pattern
* Max attempts enforced in worker: if attempt_count >= 5, mark FAILED permanently and stop retrying

### Error Classification

Reuse existing ClassifyStytchError logic:
* 429 / rate_limit / too many → retryable
* >=500 / timeout / network / connection → retryable
* duplicate_user_email / 409 → SUCCEEDED
* 400 / invalid_email → permanent
* else → retryable with reason unknown

### Progress & SSE

* Redis channel: bulk_progress_<job_id>
* Publish only:
  - On job creation
  - After each batch completes
  - On job terminal state change
* SSE endpoint sends initial state then subscribes to channel, heartbeats every 15s

### Idempotency & Safety

* DB unique constraint on (tenant_id, idempotency_key) prevents duplicate jobs
* Asynq TaskID prevents duplicate batch enqueue
* No per-item locking needed: batch task owns its items for the duration; if worker crashes, Asynq retries whole batch

### Observability

* Structured logs per batch with job_id, batch_index, succeeded/failed/deferred counts
* Prometheus counters for job created, batch processed, stytch errors by class
* Job status derivation is deterministic from item counts

### Migration Path

1. Create new tables schema or migrate existing:
* Drop bulk_job_outbox
* Drop bulk_job_dead_letter
* Simplify bulk_job_items to store email/full_name inline instead of payload json
2. Update handler to remove outbox writes and direct enqueue with TaskID
3. Update worker to batch process without TryAcquireItem
4. Update SSE to throttle publishes
5. Backfill existing jobs or run a one-off migration script to move outbox entries to tasks
6. Remove old service methods: TryAcquireItem, GetPendingOutbox, MarkOutboxEnqueued, ArchiveFailedItem

### Open Decisions

* Batch size: 40 is current. Could be 100 for less task overhead.
* Max attempts: standardize to 5 across code and tests.
* Deduplication policy: currently silent skip. Should we return info about skipped duplicates in response?
* Circuit breaker: do we need gobreaker at all? Could rely on Stytch retry classification + Asynq backoff.
* Item pagination: default page size for GET /items?

### Acceptance Criteria

* 10k row submission completes within time budget with mock 50-150ms Stytch latency
* Idempotent resubmission returns same job_id with no duplicate rows
* Progress SSE updates <= number of batches + 1
* Failed/Deferred items can be retried via endpoint
* No outbox table writes in code path
* All error responses follow canonical contract

## Implementation Status

Decisions confirmed:
* Batch size = 100
* Max attempts = 5
* Duplicate rows silent skip
* No circuit breaker, rely on Stytch error classification

Changes applied:
* Service: removed OutboxEntry, GetPendingOutbox, MarkOutboxEnqueued, ArchiveFailedItem, TryAcquireItem
* Worker: removed ProcessOutboxTask, removed per-item Redis publish, removed ArchiveFailedItem, processSingleItem now loads item by ID and marks PROCESSING without atomic acquire
* Handler: batch size updated to 100, outbox writes removed
* Tests: mock cleaned

Remaining:
* Drop bulk_job_outbox and bulk_job_dead_letter tables via migration
* Update integration tests referencing outbox/TryAcquireItem
* Add batch-level progress publish throttling if needed
