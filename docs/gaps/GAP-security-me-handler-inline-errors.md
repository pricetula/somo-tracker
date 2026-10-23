# Gap: Me Handler Uses Inline Errors for Auth Checks

**Area:** backend
**Severity:** Medium
**Status:** Resolved
**Owner:** platform-team
**Created:** 2025-10-19
**Resolved:** 2026-10-22
**Related Issues:** #security-audit-2025-10-19

## Summary
Resolved. All inline `c.Status(...).JSON(fiber.Map{...})` responses in `me_handler.go:getMe` and `me_handler.go:mapMeError` were replaced with `WriteError()` using canonical constructors (`ErrUnauthorized`, `ErrBadRequest`, `ErrNotFound`, `ErrInternal`). All error paths now include `request_id` and a non-nil `map[string][]string` `errors` field.

## Changes
- `backend/internal/api/me_handler.go`: replaced inline auth checks with `WriteError(c, ErrUnauthorized(...))`; replaced inline JSON in `mapMeError` with `WriteError(c, ErrBadRequest(...))` / `WriteError(c, ErrNotFound(...))`.
- Build: `go build ./internal/api/...` passes.

## Acceptance Criteria
- [x] All error responses use `WriteError()`
- [x] Uses `ErrUnauthorized()`, `ErrBadRequest()`, `ErrNotFound()` as appropriate
- [x] Error responses include `request_id`
- [x] `errors` field is `map[string][]string`
- [x] Consistent with other handlers

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