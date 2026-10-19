# Gap: Curriculum List Endpoints Have Pagination Cap of 200 (Should Be 100)

**Area:** backend
**Severity:** High
**Status:** Open
**Owner:** 
**Created:** 2025-10-19
**Related Issues:** #security-audit-2025-10-19

## Description
Curriculum list endpoints (`GET /api/subjects`, `GET /api/topics`, `GET /api/sub-topics`) cap pagination at 200 items per page instead of the security standard of 100.

## Current Behavior
**subjects_handler.go:20**:
```go
limit, _ := strconv.Atoi(c.Query("limit", "50"))
if limit < 1 || limit > 200 {
    limit = 50
}
```

**topics_handler.go:18**:
```go
limit, _ := strconv.Atoi(c.Query("limit", "50"))
if limit < 1 || limit > 200 {
    limit = 50
}
```

**sub_topics_handler.go:30**:
```go
limit, _ := strconv.Atoi(c.Query("limit", "50"))
if limit < 1 || limit > 200 {
    limit = 50
}
```

## Expected Behavior
Pagination limit capped at 100 items per page across all list endpoints.

## Impact
- **DoS Risk**: Larger page sizes increase memory/CPU
- **Inconsistency**: Different caps across endpoints
- **Compliance**: Violates OWASP API4:2023

## Proposed Solutions
- **Option A (Quick)**: Change 200 → 100 in all three handlers
- **Option B (Shared helper)**: Create `ValidatePagination` in `validator.go` and use everywhere

## Acceptance Criteria
- [ ] `GET /api/subjects?limit=100` returns 200 OK
- [ ] `GET /api/subjects?limit=101` returns 400 Bad Request
- [ ] Same for `/api/topics` and `/api/sub-topics`
- [ ] Shared validation helper used (Option B)

## Links
- Security audit: `security-report-complete.md`
- Handlers: `subjects_handler.go:20`, `topics_handler.go:18`, `sub_topics_handler.go:30`
- Related: `GAP-security-pagination-cap-too-high.md`