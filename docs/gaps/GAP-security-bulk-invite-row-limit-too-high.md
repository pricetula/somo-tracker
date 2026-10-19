# Gap: Bulk Invitation Endpoints Allow 10,000 Rows Per Request

**Area:** backend
**Severity:** High
**Status:** Open
**Owner:** 
**Created:** 2025-10-19
**Related Issues:** #security-audit-2025-10-19

## Description
Bulk invitation endpoints (`POST /api/admins/invitations`, `POST /api/teachers/invitations`, and similar for finance/guardian) accept up to 10,000 invitation rows in a single request. This creates resource consumption risks (memory, CPU, queue flooding).

## Current Behavior
In `admin_invitation_handler.go:85` and `teacher_invitation_handler.go:85`:
```go
if len(req.Invitations) > 10000 {
    return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
        "code": "bad_request", "message": "exceeds maximum of 10000 rows", "errors": fiber.Map{"invitations": []string{"max 10000"}},
    })
}
```

Each request creates a job with items, then enqueues batches of 100 → 100 Asynq tasks for 10K rows.

## Expected Behavior
Maximum rows per request should be capped at 1,000 (10 batches max) per OWASP API4:2023.

## Impact
- **Memory/CPU**: Parsing/validating 10K rows in single HTTP request
- **Queue Flooding**: 100 Asynq tasks enqueued atomically
- **Timeout Risk**: HTTP request may timeout before all batches enqueued
- **Abuse**: Single API call generates massive background work

## Reproduction Steps
1. POST `/api/admins/invitations` with 10,000 valid invitation rows
2. Observe 202 Accepted with 100 batches enqueued
3. Monitor Asynq queue depth and Redis memory

## Proposed Solutions
- **Option A (Quick)**: Change limit from 10000 to 1000 in both handlers
- **Option B (Config-driven)**: Add `config.BulkInviteMaxRows` with default 1000
- **Option C (Paginated import)**: For >1000 rows, require chunked upload via separate endpoint

## Decision
Pending — recommend Option A for immediate fix, Option B for configurability.

## Acceptance Criteria
- [ ] Limit changed to 1000 in `admin_invitation_handler.go` and `teacher_invitation_handler.go`
- [ ] Same limit applied to `finance_invitation_handler.go` and `guardian_invitation_handler.go`
- [ ] Integration test: 1001 rows → 400 Bad Request
- [ ] Integration test: 1000 rows → 202 Accepted, 10 batches enqueued
- [ ] Document limit in API docs

## Links
- Security audit report: `security-report-extended.md`
- OWASP API4:2023 reference: `docs/reference/owasp-api-top10-2023.md#api42023---unrestricted-resource-consumption`
- Admin handler: `backend/internal/api/admin_invitation_handler.go:85`
- Teacher handler: `backend/internal/api/teacher_invitation_handler.go:85`