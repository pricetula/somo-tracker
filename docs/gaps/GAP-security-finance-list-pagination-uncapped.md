# Gap: Finance List Endpoint Has No Pagination Cap

**Area:** backend
**Severity:** High
**Status:** Open
**Owner:** 
**Created:** 2025-10-19
**Related Issues:** #security-audit-2025-10-19

## Description
The `GET /api/finance` endpoint (`finance_handler.go:ListFinance`) accepts a `limit` query parameter but does not validate an upper bound, allowing unbounded result sets.

## Current Behavior
In `finance_handler.go:88`:
```go
limit, _ := strconv.Atoi(c.Query("limit", "50"))
```
No upper bound check. Client can request `?limit=10000` or higher.

## Expected Behavior
Pagination limit capped at 100 items per page (consistent with security standard).

## Impact
- **DoS Risk**: Large `limit` causes full table scan, high memory/CPU
- **Inconsistency**: Other endpoints have caps (students: 200, but should be 100)
- **Compliance**: Violates OWASP API4:2023

## Reproduction Steps
1. Call `GET /api/finance?limit=5000`
2. Observe 5000 items returned (or query timeout)
3. Compare with `GET /api/students?limit=5000` → 400 (capped at 200)

## Proposed Solutions
- **Option A (Quick)**: Add `if limit > 100 { limit = 100 }` or return 400
- **Option B (Shared helper)**: Create `ValidatePagination(page, limit int) (int, int, error)` in `validator.go` and use across all list endpoints

## Decision
Pending — recommend Option B for consistency (see GAP-security-pagination-cap-too-high).

## Acceptance Criteria
- [ ] `GET /api/finance?limit=100` returns 200 OK with up to 100 items
- [ ] `GET /api/finance?limit=101` returns 400 Bad Request (or clamped to 100)
- [ ] Shared validation helper created and used
- [ ] Integration test for pagination boundary

## Links
- Security audit report: `security-report-extended.md`
- OWASP API4:2023 reference: `docs/reference/owasp-api-top10-2023.md#api42023---unrestricted-resource-consumption`
- Handler: `backend/internal/api/finance_handler.go:88`
- Related gap: `GAP-security-pagination-cap-too-high.md`