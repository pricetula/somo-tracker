# Gap: School Handler ListSchools Uses fiber.NewError Instead of WriteError()

**Area:** backend
**Severity:** Medium
**Status:** Open
**Owner:** 
**Created:** 2025-10-19
**Related Issues:** #security-audit-2025-10-19

## Description
The `school_handler.go:ListSchools` method uses `fiber.NewError()` for error responses instead of the canonical `WriteError()` function.

## Current Behavior
```go
userID, ok := c.Locals("user_id").(string)
if !ok || userID == "" {
    return fiber.NewError(fiber.StatusUnauthorized, "user_id not found in session")
}
tenantID, ok := c.Locals("tenant_id").(string)
if !ok || tenantID == "" {
    return fiber.NewError(fiber.StatusUnauthorized, "tenant_id not found in session")
}

schools, err := h.service.ListSchools(c.Context(), userID, tenantID)
if err != nil {
    return fiber.NewError(fiber.StatusInternalServerError, err.Error())
}
```

## Expected Behavior
Use `WriteError(c, ErrUnauthorized(...))` and `WriteError(c, ErrInternal(...))`.

## Impact
- **Wrong error format**: `fiber.NewError` returns Fiber's default error format, not canonical `{code, message, errors}`
- **Missing `request_id`** correlation
- **Inconsistent** with `RegisterSchool` and `SetActiveSchool` which use `WriteError()`

## Proposed Solutions
- **Option A**: Refactor to use `WriteError()` with canonical constructors

## Acceptance Criteria
- [ ] All error responses use `WriteError()`
- [ ] Uses `ErrUnauthorized()`, `ErrInternal()` as appropriate
- [ ] Error responses include `request_id`
- [ ] `errors` field is `map[string][]string`
- [ ] Consistent with other methods in same handler

## Links
- Security audit: `security-report-complete.md`
- Handler: `backend/internal/api/school_handler.go:85`
- Error contract: `backend/internal/api/errors.go`