# Gap: Pagination Limit Cap Exceeds Security Standard

**Area:** backend
**Severity:** High
**Status:** Open
**Owner:** 
**Created:** 2025-10-19
**Related Issues:** #security-audit-2025-10-19

## Description
The `GET /api/students` endpoint allows a `limit` query parameter up to 200 items per page. The security audit skill (OWASP API4:2023 — Unrestricted Resource Consumption) recommends a hard cap of 100 to prevent resource exhaustion via large page requests.

## Current Behavior
In `backend/internal/api/students_handler.go:48`:
```go
limit, err := strconv.Atoi(limitStr)
if err != nil || limit < 1 || limit > 200 {
    return WriteError(c, ErrBadRequest("invalid limit parameter", map[string][]string{"limit": {"must be between 1 and 200"}}))
}
```

## Expected Behavior
Pagination limit should be capped at 100 items per page across all list endpoints.

## Impact
- **DoS Risk**: A malicious or buggy client could request 200 items per page, increasing memory/CPU usage and query latency
- **Consistency**: Other endpoints may have different caps (need audit)
- **Compliance**: Violates project security standard (OWASP API4)

## Reproduction Steps
1. Call `GET /api/students?limit=200`
2. Observe 200 items returned in single response
3. Call `GET /api/students?limit=500` → returns 400 (correctly rejected)
4. Compare with security standard of 100 max

## Proposed Solutions
- **Option A (Quick Fix)**: Change cap from 200 to 100 in `students_handler.go`
- **Option B (Systematic)**: Create shared pagination validation helper in `backend/internal/api/validator.go` and apply to all list endpoints
- **Option C (Config-driven)**: Make max limit configurable via `config.Config` with default 100

## Decision
Pending — recommend Option A for immediate fix, Option B for consistency across all endpoints.

## Acceptance Criteria
- [ ] `GET /api/students?limit=100` returns 200 OK with up to 100 items
- [ ] `GET /api/students?limit=101` returns 400 Bad Request with validation error
- [ ] All other list endpoints (`/teachers`, `/guardians`, `/classes`, `/streams`, `/rooms`, `/events`, `/admins`, `/finance`) audited and capped at 100
- [ ] Shared validation helper created (if Option B chosen)
- [ ] Integration test added for pagination boundary (100 vs 101)

## Links
- Security audit report: `security-report-test.md`
- OWASP API4:2023 reference: `docs/reference/owasp-api-top10-2023.md#api42023---unrestricted-resource-consumption`
- Validator helper: `backend/internal/api/validator.go`