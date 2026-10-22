# Gap: Events Handler Uses Inline Errors for Auth Checks

**Area:** backend
**Severity:** Medium
**Status:** Open
**Owner:** 
**Created:** 2025-10-19
**Related Issues:** #security-audit-2025-10-19

## Description
The `events_handler.go` uses inline `c.Status(...).JSON(fiber.Map{...})` for authentication/authorization checks but uses `WriteError()` for service-layer errors, creating inconsistency.

## Current Behavior
Auth checks in `ListEvents`, `CreateEvent`, `UpdateEvent`, `DeleteEvent`:
```go
schoolIDStr, ok := c.Locals("active_school_id").(string)
if !ok || schoolIDStr == "" {
    return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
        "code": "unauthorized", "message": "...", "errors": fiber.Map{},
    })
}
```

But service errors use:
```go
return WriteError(c, ErrInternal("failed to list events"))
```

## Expected Behavior
All error paths should use `WriteError()` with canonical constructors.

## Impact
- **Inconsistent error shape**: Auth errors missing `request_id`
- **Errors map type**: `fiber.Map{}` vs `map[string][]string`

## Proposed Solutions
- **Option A**: Replace inline auth checks with `WriteError(c, ErrUnauthorized(...))`

## Acceptance Criteria
- [ ] All error responses use `WriteError()`
- [ ] Uses `ErrUnauthorized()`, `ErrBadRequest()`, `ErrForbidden()` as appropriate
- [ ] Error responses include `request_id`
- [ ] `errors` field is `map[string][]string`

## Links
- Security audit: `security-report-complete.md`
- Handler: `backend/internal/api/events_handler.go`
- Error contract: `backend/internal/api/errors.go`