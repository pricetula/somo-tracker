# Gap: Bulk Invitation Service Missing Authorization Check (BFLA)

**Area:** backend
**Severity:** Critical
**Status:** Closed
**Owner:** 
**Created:** 2025-10-19
**Related Issues:** #security-audit-2025-10-19

## Description
Bulk invitation handlers (`admin_invitation_handler.go`, `teacher_invitation_handler.go`, etc.) check `UserHasAdminRole` in the handler layer, but the service methods (`CreateBulkJobWithItems`, `GetJob`, `GetFailedOrDeferredItems`) do not re-verify authorization. If service methods are called directly (tests, internal tools, future handlers), the authorization check is bypassed.

## Current Behavior
**Handler** (`admin_invitation_handler.go:58-65`):
```go
hasAdmin, err := h.svc.UserHasAdminRole(c.Context(), schoolID, adminUserID)
if err != nil { return WriteError(c, ErrInternal(...)) }
if !hasAdmin {
    return WriteError(c, ErrForbidden("insufficient_permissions", "insufficient permissions"))
}
```

**Service** (`services/admin_invitation_service.go` - not shown but inferred):
```go
func (s *adminInvitationService) CreateBulkJobWithItems(ctx, schoolID, tenantID, adminUserID, idempotencyKey, items) {
    // NO authorization check — assumes caller verified
}
```

## Expected Behavior
Service layer must enforce authorization as defense in depth. Every mutating method should verify the caller has appropriate role for the given school/tenant.

## Impact
- **Privilege Escalation**: Direct service call bypasses role check
- **Test Gaps**: Unit tests may not catch missing authz
- **Future Risk**: New handler using same service forgets check
- **Compliance**: OWASP API5:2023 BFLA — Critical severity

## Reproduction Steps
1. In test or internal tool, call `svc.CreateBulkJobWithItems(ctx, schoolID, tenantID, userID, key, items)` where `userID` is NOT an admin
2. Observe job created successfully (no error)
3. Compare with handler path → 403 Forbidden

## Proposed Solutions
- **Option A (Service-level authz)**: Add `schoolID`, `actorUserID` to all service methods; verify role inside service
- **Option B (Context-based)**: Pass `actorUserID` via context; service reads and validates
- **Option C (Policy engine)**: Use OPA/Cedar for centralized authz decisions

## Decision
Pending — recommend Option A for explicit, testable authorization.

## Acceptance Criteria
- [x] All mutating service methods (`CreateBulkJobWithItems`, `RetryFailed`, etc.) accept `actorUserID` parameter
- [x] Service verifies `UserHasAdminRole(ctx, schoolID, actorUserID)` before operation
- [x] Returns `ErrForbidden` if not admin
- [x] Handler passes `adminUserID` from context to service
- [x] Unit tests verify service rejects non-admin callers
- [x] Applied to Admin, Teacher, Finance, Guardian invitation services

## Links
- Security audit report: `security-report-extended.md`
- OWASP API5:2023 reference: `docs/reference/owasp-api-top10-2023.md#api52023---broken-function-level-authorization-bfla`
- Admin handler: `backend/internal/api/admin_invitation_handler.go:58`
- Teacher handler: `backend/internal/api/teacher_invitation_handler.go:58`
- Service interface: `backend/internal/services/admin_invitation_service.go`