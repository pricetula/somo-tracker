# Gap: GetTemplate Returns 400 Instead of 404 for Not Found

**Area:** backend
**Severity:** Medium
**Status:** Open
**Owner:** 
**Created:** 2025-10-19
**Related Issues:** #security-audit-2025-10-19

## Description
The `GET /api/timetable/templates/:id` endpoint (`timetable_handler.go:GetTemplate`) returns 400 Bad Request when template is not found, but should return 404 Not Found.

## Current Behavior
In `timetable_handler.go:85`:
```go
template, err := h.svc.GetTemplate(c.Context(), templateID)
if err != nil {
    h.logger.Error("get timetable template failed", zap.Error(err))
    return WriteError(c, ErrBadRequest(err.Error(), nil))  // WRONG: should be ErrNotFound
}
```

## Expected Behavior
Return 404 Not Found when resource doesn't exist. 400 should be reserved for invalid request format (e.g., malformed UUID).

## Impact
- **Semantic Incorrectness**: 400 implies client error in request format; 404 means resource not found
- **Client Handling**: Frontend may retry 400 (thinking malformed request) vs handle 404 gracefully
- **API Contract**: Violates REST conventions
- **Swagger Mismatch**: `@Failure 400` documented but should be 404

## Reproduction Steps
1. Call `GET /api/timetable/templates/invalid-uuid` → 400 (correct — malformed UUID)
2. Call `GET /api/timetable/templates/00000000-0000-0000-0000-000000000000` (valid UUID, not found) → 400 (incorrect — should be 404)

## Proposed Solutions
- **Option A (Service returns typed error)**: Service returns `ErrNotFound` sentinel; handler maps to `ErrNotFound()`
- **Option B (Handler checks error type)**: Check `errors.Is(err, pgx.ErrNoRows)` or string match "not found"
- **Option C (Repository pattern)**: Repository returns `ErrNotFound`; service propagates; handler maps

## Decision
Pending — recommend Option A (service-level typed errors).

## Acceptance Criteria
- [ ] Service `GetTemplate` returns typed `ErrNotFound` when template doesn't exist
- [ ] Handler maps to `WriteError(c, ErrNotFound("template not found"))`
- [ ] Returns 404 status code
- [ ] Swagger `@Failure 404` annotation added
- [ ] Integration test: valid UUID not found → 404

## Links
- Security audit report: `security-report-extended.md`
- OWASP API8:2023 reference: `docs/reference/owasp-api-top10-2023.md#api82023---security-misconfiguration`
- Handler: `backend/internal/api/timetable_handler.go:85`
- Service: `backend/internal/services/timetable_service.go` (GetTemplate)