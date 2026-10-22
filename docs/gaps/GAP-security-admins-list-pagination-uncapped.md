# Gap: Admins List Endpoint Has No Pagination Cap

**Area:** backend
**Severity:** High
**Status:** Open
**Owner:** 
**Created:** 2025-10-19
**Related Issues:** #security-audit-2025-10-19

## Description
The `GET /api/admins` endpoint (`admins_handler.go:ListAdmins`) accepts a `limit` query parameter but does not validate an upper bound, allowing unbounded result sets.

## Current Behavior
In `admins_handler.go:130`:
```go
limit, _ := strconv.Atoi(c.Query("limit", "50"))
```
No upper bound check.

## Expected Behavior
Pagination limit capped at 100 items per page.

## Impact
- **DoS Risk**: Large `limit` causes full table scan
- **Inconsistency**: Other endpoints should have consistent caps
- **Compliance**: Violates OWASP API4:2023

## Reproduction Steps
1. Call `GET /api/admins?limit=5000`
2. Observe 5000 items returned (or timeout)

## Proposed Solutions
- **Option A (Shared helper)**: Create `ValidatePagination(page, limit int) (int, int, error)` in `validator.go`
- **Option B (Quick fix)**: Add `if limit > 100 { limit = 100 }`

## Acceptance Criteria
- [ ] `GET /api/admins?limit=100` returns 200 OK
- [ ] `GET /api/admins?limit=101` returns 400 or clamped to 100
- [ ] Shared validation helper used

## Links
- Security audit: `security-report-complete.md`
- Handler: `backend/internal/api/admins_handler.go:130`
- Related: `GAP-security-pagination-cap-too-high.md`