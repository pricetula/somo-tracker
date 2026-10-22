# Gap: Streams List Endpoint Has No Pagination

**Area:** backend
**Severity:** High
**Status:** Open
**Owner:** 
**Created:** 2025-10-19
**Related Issues:** #security-audit-2025-10-19

## Description
The `GET /api/streams` endpoint (`streams_handler.go:ListStreams`) returns all streams for a school without any pagination parameters.

## Current Behavior
In `streams_handler.go:45`:
```go
func (h *StreamsHandler) ListStreams(c fiber.Ctx) error {
    schoolID, ok := c.Locals("active_school_id").(string)
    // ...
    streams, err := h.service.ListStreams(c.Context(), schoolID)
    // ...
    return c.JSON(fiber.Map{"streams": streams})
}
```
No `page` or `limit` query parameters accepted.

## Expected Behavior
Endpoint should support pagination with `page` (default 1) and `limit` (default 50, max 100) parameters.

## Impact
- **DoS Risk**: School with many streams returns unbounded response
- **Inconsistency**: Other list endpoints support pagination
- **Compliance**: Violates OWASP API4:2023

## Proposed Solutions
- **Option A**: Add `page`, `limit` params with validation; pass to service

## Acceptance Criteria
- [ ] `GET /api/streams?page=1&limit=50` returns paginated results
- [ ] `GET /api/streams?limit=101` returns 400 Bad Request
- [ ] Response includes pagination metadata (total, page, limit)
- [ ] Service method updated to support pagination

## Links
- Security audit: `security-report-complete.md`
- Handler: `backend/internal/api/streams_handler.go:45`
- Service: `backend/internal/services/streams_service.go` (ListStreams)
- Related: `GAP-security-pagination-cap-too-high.md`