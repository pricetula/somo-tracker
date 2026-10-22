# Gap: Finance Handler Mixes WriteError() and Inline Error Responses

**Area:** backend
**Severity:** Medium
**Status:** Open
**Owner:** 
**Created:** 2025-10-19
**Related Issues:** #security-audit-2025-10-19

## Description
The `finance_handler.go` uses `WriteError()` for service-layer errors but returns inline `c.Status().JSON(fiber.Map{...})` for authentication/authorization checks, creating inconsistent error shapes.

## Current Behavior
In `finance_handler.go:26-40` (DeleteFinance) and `finance_handler.go:80-90` (ListFinance):
```go
// Auth checks use inline responses
schoolIDStr, ok := c.Locals("active_school_id").(string)
if !ok || schoolIDStr == "" {
    return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
        "code": "unauthorized", "message": "active school not found in session", "errors": fiber.Map{},
    })
}

// But service errors use WriteError()
if err := h.svc.DeleteFinance(...); err != nil {
    return WriteError(c, ErrBadRequest(...))
}
```

## Expected Behavior
All error paths should use `WriteError()` with canonical constructors (`ErrUnauthorized()`, `ErrBadRequest()`, etc.).

## Impact
- **Inconsistent Shape**: Auth errors missing `request_id`; `errors` is `fiber.Map{}` not `map[string][]string`
- **Frontend Parsing**: Error handler expects consistent shape
- **Audit Trail**: Missing request correlation on auth failures

## Reproduction Steps
1. Call `DELETE /api/finance` without valid session
2. Observe response: missing `request_id`, `errors: {}`
3. Call with valid session but service error → has `request_id`, `errors: []`

## Proposed Solutions
- **Option A (Refactor)**: Replace inline auth checks with `WriteError(c, ErrUnauthorized(...))`
- **Option B (Middleware)**: Move auth checks to middleware (but need school-specific context)

## Decision
Pending — recommend Option A for quick fix.

## Acceptance Criteria
- [ ] All inline `c.Status().JSON(fiber.Map{...})` in `finance_handler.go` replaced with `WriteError()`
- [ ] Uses `ErrUnauthorized()`, `ErrBadRequest()`, `ErrForbidden()` as appropriate
- [ ] Error responses include `request_id`
- [ ] `errors` field is `map[string][]string`
- [ ] Consistent with other handlers

## Links
- Security audit report: `security-report-extended.md`
- Error contract: `backend/internal/api/errors.go`
- Handler: `backend/internal/api/finance_handler.go`
- Related: `GAP-security-invitation-inline-error-responses.md`