# Gap: Attendance Reports List Endpoint Has Hardcoded Limit No Validation

**Area:** backend
**Severity:** High
**Status:** Open
**Owner:** 
**Created:** 2025-10-19
**Related Issues:** #security-audit-2025-10-19

## Description
The `GET /api/attendance/reports` endpoint (`attendance_reports_handler.go:ListStudentTermStats`) has hardcoded `limit := 50` with no query parameter support and no validation.

## Current Behavior
In `attendance_reports_handler.go:38`:
```go
page := 1
limit := 50
if p := c.Query("page"); p != "" {
    if v, err := strconv.Atoi(p); err == nil && v > 0 {
        page = v
    }
}
// limit is hardcoded, no query param support
items, _, err := h.svc.ListStudentTermStats(c.Context(), schoolID, termID, page, limit)
```

## Expected Behavior
Endpoint should support `limit` query parameter with validation (max 100).

## Impact
- **No client control**: Cannot request smaller/larger pages
- **Future risk**: If limit made configurable, no validation
- **Inconsistency**: Other list endpoints support configurable pagination

## Proposed Solutions
- **Option A**: Add `limit` query param with validation (max 100)

## Acceptance Criteria
- [ ] `GET /api/attendance/reports?term_id=...&limit=50` works
- [ ] `GET /api/attendance/reports?term_id=...&limit=101` returns 400
- [ ] Default limit 50, max 100

## Links
- Security audit: `security-report-complete.md`
- Handler: `backend/internal/api/attendance_reports_handler.go:38`
- Related: `GAP-security-pagination-cap-too-high.md`