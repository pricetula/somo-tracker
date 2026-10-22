# Gap: Invitation Handlers Use Unsafe Type Assertion for Session Locals

**Area:** backend
**Severity:** High
**Status:** Open
**Owner:** 
**Created:** 2025-10-19
**Related Issues:** #security-audit-2025-10-19

## Description
All bulk invitation handlers use `fmt.Sprintf("%v", c.Locals("key"))` to convert session locals to strings before UUID parsing. This is unsafe — if the local is not a string (e.g., nil, wrong type), it produces garbage like `<nil>` or `map[]` which then fails UUID parsing with unclear error.

## Current Behavior
In `admin_invitation_handler.go:45-52` (and teacher/finance/guardian equivalents):
```go
schoolIDStr := c.Locals("active_school_id")
tenantIDStr := c.Locals("tenant_id")
adminUserIDStr := c.Locals("user_id")

schoolID, err1 := uuid.Parse(fmt.Sprintf("%v", schoolIDStr))
tenantID, err2 := uuid.Parse(fmt.Sprintf("%v", tenantIDStr))
adminUserID, err3 := uuid.Parse(fmt.Sprintf("%v", adminUserIDStr))
if err1 != nil || err2 != nil || err3 != nil {
    return WriteError(c, ErrUnauthorized("missing session context"))
}
```

## Expected Behavior
Use safe type assertion like other handlers (`students_handler.go`, `finance_handler.go`):
```go
schoolIDStr, ok := c.Locals("active_school_id").(string)
if !ok || schoolIDStr == "" {
    return WriteError(c, ErrUnauthorized("active school not found"))
}
schoolID, err := uuid.Parse(schoolIDStr)
```

## Impact
- **Obscure Errors**: Wrong local type → "invalid UUID" instead of "missing context"
- **Debugging Difficulty**: Hard to distinguish missing local vs type mismatch
- **Inconsistency**: Other handlers use safe pattern
- **Potential Panic**: If local is unexpected type (though `%v` handles most)

## Reproduction Steps
1. Manually inject non-string into `c.Locals("active_school_id")` (e.g., int)
2. Call bulk invitation endpoint
3. Observe 401 with "missing session context" (misleading)

## Proposed Solutions
- **Option A (Refactor)**: Replace all `fmt.Sprintf("%v", ...)` with safe type assertions
- **Option B (Helper)**: Create `GetLocalString(c, key) (string, error)` utility
- **Option C (Middleware)**: Validate all required locals in session middleware

## Decision
Pending — recommend Option A for immediate consistency.

## Acceptance Criteria
- [ ] All invitation handlers (admin, teacher, finance, guardian, student import) use safe type assertion
- [ ] Returns proper `ErrUnauthorized` with specific message ("active school not found", "tenant not found", "user not found")
- [ ] Consistent with `students_handler.go`, `finance_handler.go`, `timetable_handler.go`
- [ ] Unit test for missing/invalid locals

## Links
- Security audit report: `security-report-extended.md`
- OWASP API2:2023 reference: `docs/reference/owasp-api-top10-2023.md#api22023---broken-authentication`
- Admin handler: `backend/internal/api/admin_invitation_handler.go:45`
- Teacher handler: `backend/internal/api/teacher_invitation_handler.go:45`
- Good pattern: `backend/internal/api/students_handler.go:35`