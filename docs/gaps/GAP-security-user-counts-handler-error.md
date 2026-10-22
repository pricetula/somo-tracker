# Gap: User Counts Handler Uses fiber.NewError Instead of WriteError()

**Area:** backend
**Severity:** Medium
**Status:** Open
**Owner:** 
**Created:** 2025-10-19
**Related Issues:** #security-audit-2025-10-19

## Description
The `user_counts_handler.go:GetUserCounts` method uses `fiber.NewError()` for all error responses instead of the canonical `WriteError()` function.

## Current Behavior
```go
schoolID, ok := c.Locals("active_school_id").(string)
if !ok || schoolID == "" {
    return fiber.NewError(fiber.StatusUnauthorized, "active_school_id not found in session")
}
counts, err := h.service.GetUserCounts(c.Context(), schoolID)
if err != nil {
    return fiber.NewError(fiber.StatusInternalServerError, err.Error())
}
```

## Expected Behavior
Use `WriteError(c, ErrUnauthorized(...))` and `WriteError(c, ErrInternal(...))`.

## Impact
- **Wrong error format**: `fiber.NewError` returns Fiber's default error format
- **Missing `request_id`** correlation
- **Inconsistent** with other handlers

## Proposed Solutions
- **Option A**: Refactor to use `WriteError()` with canonical constructors

## Acceptance Criteria
- [ ] All error responses use `WriteError()`
- [ ] Uses `ErrUnauthorized()`, `ErrInternal()` as appropriate
- [ ] Error responses include `request_id`
- [ ] `errors` field is `map[string][]string`

## Links
- Security audit: `security-report-complete.md`
- Handler: `backend/internal/api/user_counts_handler.go`
- Error contract: `backend/internal/api/errors.go`