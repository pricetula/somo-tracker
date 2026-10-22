# Gap: Invitation Handlers Use Inline Error Responses Instead of WriteError()

**Area:** backend
**Severity:** Medium
**Status:** Open
**Owner:** 
**Created:** 2025-10-19
**Related Issues:** #security-audit-2025-10-19

## Description
Bulk invitation handlers (`admin_invitation_handler.go`, `teacher_invitation_handler.go`, etc.) return inline `c.Status(...).JSON(fiber.Map{...})` for validation errors instead of using the canonical `WriteError()` function with `ErrBadRequest()`, `ErrForbidden()`, `ErrInternal()`.

## Current Behavior
Example from `admin_invitation_handler.go:62`:
```go
if !hasAdmin {
    return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
        "code": "forbidden", "message": "insufficient permissions", "errors": fiber.Map{},
    })
}
```

Other occurrences: lines 78, 88, 97, 123, 130, and all equivalents in teacher/finance/guardian handlers.

## Expected Behavior
Use `WriteError(c, ErrForbidden("insufficient_permissions", "insufficient permissions"))` for consistency with:
- Canonical error contract (`backend/internal/api/errors.go`)
- Other handlers (`students_handler.go`, `finance_handler.go`, `timetable_handler.go`)
- Frontend error parser expectation

## Impact
- **Inconsistent Error Shape**: Missing `request_id` field in response
- **Errors Map Type**: Uses `fiber.Map{}` (map[string]interface{}) vs required `map[string][]string`
- **Logging**: No structured error logging via `WriteError()` path
- **Maintenance**: Error codes scattered, hard to update

## Reproduction Steps
1. Call `POST /api/admins/invitations` without admin role
2. Observe response: `{"code":"forbidden","message":"insufficient permissions","errors":{}}`
3. Compare with `PATCH /api/students/:id` without permission → includes `request_id`, `errors: []`

## Proposed Solutions
- **Option A (Refactor)**: Replace all inline responses with `WriteError()` calls
- **Option B (Helper)**: Create `WriteForbidden()`, `WriteBadRequest()` wrappers (already exist as `ErrForbidden()`, `ErrBadRequest()`)

## Decision
Pending — recommend Option A (direct refactor to existing constructors).

## Acceptance Criteria
- [ ] All inline `c.Status().JSON(fiber.Map{...})` replaced with `WriteError(c, ErrXxx(...))`
- [ ] Error responses include `request_id` field
- [ ] `errors` field is `map[string][]string` (empty `{}` not `null`)
- [ ] Error codes match `backend/internal/api/errors.go` constants
- [ ] Applied to admin, teacher, finance, guardian, student import handlers

## Links
- Security audit report: `security-report-extended.md`
- Error contract: `backend/internal/api/errors.go`
- Admin handler: `backend/internal/api/admin_invitation_handler.go:62,78,88,97,123,130`
- Teacher handler: `backend/internal/api/teacher_invitation_handler.go` (same lines)
- Good pattern: `backend/internal/api/students_handler.go`