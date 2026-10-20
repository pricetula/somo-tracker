# Gap: Finance Delete Endpoint Missing Role Verification (BFLA)

**Area:** backend
**Severity:** Critical
**Status:** Closed
**Owner:** 
**Created:** 2026-10-20
**Related Issues:** #security-audit-2026-10-19

## Description
The `DELETE /api/finance` endpoint retrieves `currentUserID` from context but does not verify the caller has an admin role before allowing deletion of finance records. Service layer may enforce authz, but handler lacks defense-in-depth check, enabling Broken Function Level Authorization if service is called directly or bypassed.

## Current Behavior
In `backend/internal/api/finance_handler.go:26-40` (DeleteFinance):
```go
currentUserID, ok := c.Locals("userID").(string)
if !ok {
    return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{...})
}
// No role check performed
if err := h.svc.DeleteFinance(c.Context(), schoolID, financeIDs, currentUserID); err != nil {
    return WriteError(c, ErrBadRequest(...))
}
```
Handler passes `currentUserID` to service but never validates `c.Locals("role") == "admin"`.

## Expected Behavior
Handler must verify caller role is `admin` for destructive finance operations. Service method signature should be `DeleteFinance(ctx, schoolID, financeIDs, userID, role)` or service should independently re-verify role.

## Impact
- **BFLA – Broken Function Level Authorization**: Non-admin user with valid session could delete finance records if service check is missing or changed.
- **Data Loss / Compliance**: Finance data is sensitive; unauthorized deletion is a critical security and audit risk.
- **OWASP API5:2023**

## Reproduction Steps
1. Authenticate as regular user with valid `userID` and `active_school_id`
2. Call `DELETE /api/finance` with finance IDs
3. Observe 200/204 response without 403 Forbidden

## Proposed Solutions
- **Option A (Handler check)**: Add `RequireRole("admin")` middleware to `DELETE /api/finance` route and/or explicit check:
  ```go
  role, ok := c.Locals("role").(string)
  if role != "admin" { return WriteError(c, ErrForbidden(...)) }
  ```
- **Option B (Service enforcement)**: Service `DeleteFinance` must validate role from context, defense in depth
- **Option C (Both)**: Handler role check + service re-verification + OpenAPI `@Security` annotation

## Decision
Pending — recommend Option C.

## Acceptance Criteria
- [x] Handler verifies `c.Locals("role")` is `admin` before calling service
- [x] Returns 403 with canonical error shape via `WriteError(c, ErrForbidden(...))`
- [x] Service `DeleteFinance` also validates role / caller permissions
- [x] Integration test: regular user DELETE → 403
- [ ] OpenAPI annotation updated with admin requirement
- [ ] Security audit re-run passes for `finance_handler.go:DeleteFinance`

## Links
- Security audit report: `security-report-complete.md`
- OWASP API5:2023 reference: `docs/reference/owasp-api-top10-2023.md#api5`
- Handler: `backend/internal/api/finance_handler.go:26`
- Service: `backend/internal/services/finance_service.go:DeleteFinance`
- Related: `GAP-security-bulk-invite-service-missing-authz.md`, `GAP-security-finance-handler-error-inconsistency.md`
