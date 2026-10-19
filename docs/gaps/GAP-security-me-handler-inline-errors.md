# Gap: Me Handler Uses Inline Errors for Auth Checks

**Area:** backend
**Severity:** Medium
**Status:** Open
**Owner:** 
**Created:** 2025-10-19
**Related Issues:** #security-audit-2025-10-19

## Description
The `me_handler.go:getMe` method uses inline `c.Status(...).JSON(fiber.Map{...})` for token/tenant validation but uses `WriteError()` for service errors.

## Current Behavior
```go
token := c.Cookies("session_token")
if token == "" {
    return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
        "code": "unauthorized", "message": "session token missing", "errors": fiber.Map{},
    })
}
tenantID, _ := c.Locals("tenant_id").(string)
if tenantID == "" {
    return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
        "code": "unauthorized", "message": "tenant context missing", "errors": fiber.Map{},
    })
}
```

But service errors use:
```go
return WriteError(c, ErrInternal("An unexpected error occurred"))
```

## Expected Behavior
All error paths should use `WriteError()` with canonical constructors.

## Impact
- **Inconsistent error shape**: Auth errors missing `request_id`
- **Errors map type**: `fiber.Map{}` vs `map[string][]string`

## Proposed Solutions
- **Option A**: Replace inline checks with `WriteError(c, ErrUnauthorized(...))`

## Acceptance Criteria
- [ ] All error responses use `WriteError()`
- [ ] Uses `ErrUnauthorized()`, `ErrBadRequest()` as appropriate
- [ ] Error responses include `request_id`
- [ ] `errors` field is `map[string][]string`

## Links
- Security audit: `security-report-complete.md`
- Handler: `backend/internal/api/me_handler.go`
- Error contract: `backend/internal/api/errors.go`