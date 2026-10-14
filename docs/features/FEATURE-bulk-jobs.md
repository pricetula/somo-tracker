# Feature: Bulk Operations Framework

**Status:** Done
**Owner:** Platform team
**Created:** 2026-06-01
**Last Updated:** 2026-10-15
**Related Issues:** #
**Affected Layers:** backend

## Summary
A generic bulk ingestion framework for large-scale data imports with idempotency, per-row status tracking, and audit trail for admin invitations, student imports, teacher invitations, guardian invitations, and finance invitations.

## Problem Statement
Manual CSV imports were ad-hoc and not auditable. Operators needed a consistent way to import thousands of records with retry, partial failure handling, and tenant isolation.

## Solution Overview
Two core tables `bulk_jobs` and `bulk_job_items` plus worker processors. Jobs are created with an `idempotency_key` per tenant to prevent duplicates. Each row is tracked individually with status, attempt count, and last error.

### User Stories
- As an Admin, I want to upload a CSV of students so that they are created in batches with error reporting per row
- As an Operator, I want to see job progress and failures without re-submitting

### Acceptance Criteria
- [x] Job can be created with type `ADMIN_INVITATION | STUDENT_IMPORT | TEACHER_INVITATION | GUARDIAN_INVITATION | FINANCE_INVITATION`
- [x] Idempotency key prevents duplicate submissions per tenant
- [x] Per-row status: PENDING, PROCESSING, SUCCEEDED, FAILED, DEFERRED
- [x] Counters: total_records, succeeded_count, failed_count, deferred_count
- [x] RLS enforced via tenant_id / school_id
- [x] `bulk_job_items.payload` stores input JSONB, `result` stores output on success

## Scope

### Backend
- API changes: Bulk job creation and status endpoints
- DB changes: `bulk_jobs`, `bulk_job_items` tables with cascade delete
- Error contract compliance: returns canonical error JSON on validation failures
- RLS / multi-tenancy: jobs scoped to tenant_id and school_id

### Frontend
- Not yet implemented. Gap: UI for upload and progress.

### Public / Marketing
- N/A

### Non-Goals
- Real-time streaming ingestion
- Cross-tenant bulk operations

## Design Notes
See `docs/backend_schema.md` sections `bulk_jobs` and `bulk_job_items`. Worker example: `backend/internal/worker/bulk_invitation_processor.go`. Triggers maintain `updated_at`.

## Open Gaps / Risks
- Gap: No frontend UI for bulk upload / monitoring. See `docs/gaps/GAP-bulk-ui.md` when created.
- Gap: No retry UI for failed items.

## Test Plan
- Unit tests for idempotency key enforcement
- Integration tests for worker processing with partial failures
- Playwright not applicable – backend only

## Rollout
- Feature flag not required; backend API ready
- Migration applied via database migrator

## References
- `docs/backend_schema.md` - Backend Schema Summary
- `backend/internal/database/sqlc/bulk_jobs.sql.go`
