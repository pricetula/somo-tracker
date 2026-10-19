# Gap: Timetable Templates List Endpoint Has No Pagination

**Area:** backend
**Severity:** High
**Status:** Open
**Owner:** 
**Created:** 2025-10-19
**Related Issues:** #security-audit-2025-10-19

## Description
The `GET /api/timetable/templates` endpoint (`timetable_handler.go:ListTemplates`) returns all templates for a school without any pagination parameters.

## Current Behavior
In `timetable_handler.go:52-62`:
```go
func (h *TimetableHandler) ListTemplates(c fiber.Ctx) error {
    schoolIDStr, ok := c.Locals("active_school_id").(string)
    // ...
    items, err := h.svc.ListTemplates(c.Context(), schoolID)
    // ...
    return c.Status(fiber.StatusOK).JSON(items)
}
```
No `page` or `limit` query parameters accepted.

## Expected Behavior
Endpoint should support pagination with `page` (default 1) and `limit` (default 50, max 100) parameters.

## Impact
- **DoS Risk**: School with many templates returns unbounded response
- **Performance**: Full table scan, large JSON serialization
- **Inconsistency**: Other list endpoints support pagination
- **Compliance**: Violates OWASP API4:2023

## Reproduction Steps
1. Create 500+ timetable templates for a school
2. Call `GET /api/timetable/templates`
3. Observe all 500+ returned in single response

## Proposed Solutions
- **Option A (Quick)**: Add `page`, `limit` params with validation; pass to service
- **Option B (Cursor-based)**: Use cursor pagination for better performance on large datasets
- **Option C (Shared helper)**: Use same pagination validation as other endpoints

## Decision
Pending — recommend Option A for parity with other endpoints.

## Acceptance Criteria
- [ ] `GET /api/timetable/templates?page=1&limit=50` returns paginated results
- [ ] `GET /api/timetable/templates?limit=101` returns 400 Bad Request
- [ ] Response includes pagination metadata (total, page, limit)
- [ ] Service method updated to support pagination
- [ ] Integration test for pagination

## Links
- Security audit report: `security-report-extended.md`
- OWASP API4:2023 reference: `docs/reference/owasp-api-top10-2023.md#api42023---unrestricted-resource-consumption`
- Handler: `backend/internal/api/timetable_handler.go:52`
- Service: `backend/internal/services/timetable_service.go` (ListTemplates)
- Related gap: `GAP-security-pagination-cap-too-high.md`