# Gap: Guardians List Endpoint Has No Pagination Cap

**Area:** backend
**Severity:** High
**Status:** Open
**Owner:** 
**Created:** 2025-10-19
**Related Issues:** #security-audit-2025-10-19

## Description
The `GET /api/guardians` endpoint (`guardians_handler.go:ListGuardians`) accepts a `limit` query parameter but does not validate an upper bound.

## Current Behavior
In `guardians_handler.go:145`:
```go
limit, _ := strconv.Atoi(c.Query("limit", "50"))
```
No upper bound check.

## Expected Behavior
Pagination limit capped at 100 items per page.

## Impact
- **DoS Risk**: Unbounded result sets
- **Compliance**: Violates OWASP API4:2023

## Proposed Solutions
- **Option A (Shared helper)**: Use `ValidatePagination` from `validator.go`

## Acceptance Criteria
- [ ] `GET /api/guardians?limit=100` returns 200 OK
- [ ] `GET /api/guardians?limit=101` returns 400 or clamped to 100
- [ ] Shared validation helper used

## Links
- Security audit: `security-report-complete.md`
- Handler: `backend/internal/api/guardians_handler.go:145`
- Related: `GAP-security-pagination-cap-too-high.md`