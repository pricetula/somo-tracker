# Gap: Students Import Handler Uses Inline Error Responses

**Area:** backend
**Severity:** Medium
**Status:** Open
**Owner:** 
**Created:** 2025-10-19
**Related Issues:** #security-audit-2025-10-19

## Description
The `students_import_handler.go` uses inline `c.Status(...).JSON(fiber.Map{...})` for all error responses instead of the canonical `WriteError()` function.

## Current Behavior
All methods (`HandleImport`, `GetJob`, `Events`) use inline responses:
```go
return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
    "code": "bad_request", "message": "...", "errors": fiber.Map{},
})
```

## Expected Behavior
Use `WriteError(c, ErrBadRequest(...))`, `WriteError(c, ErrUnauthorized(...))`, etc. for consistency with error contract.

## Impact
- **Missing `request_id`** in error responses
- **Errors map type**: `fiber.Map{}` (map[string]interface{}) vs required `map[string][]string`
- **Inconsistent** with other handlers using `WriteError()`

## Proposed Solutions
- **Option A**: Refactor all error returns to use `WriteError()` with canonical constructors

## Acceptance Criteria
- [ ] All error responses use `WriteError()`
- [ ] Error responses include `request_id`
- [ ] `errors` field is `map[string][]string`
- [ ] Consistent with `students_handler.go`, `finance_handler.go` patterns

## Links
- Security audit: `security-report-complete.md`
- Handler: `backend/internal/api/students_import_handler.go`
- Error contract: `backend/internal/api/errors.go`
- Related: `GAP-security-invitation-inline-error-responses.md`