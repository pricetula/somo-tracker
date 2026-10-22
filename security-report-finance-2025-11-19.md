# Security Audit Report — Finance Handler & Service Review

**Date**: 2025-11-19
**Auditor**: pi-agent (security-audit skill)
**Scope**: `backend/internal/api/finance_handler.go`, `backend/internal/services/finance_service.go`

## Findings

### Critical
- None newly identified. BFLA verified.

### High
- **[API5-BFLA]** Duplicate admin checks: Handler verifies `UserHasAdminRole` before calling service, service repeats check. Defense-in-depth acceptable but adds latency. Consider removing service-level check to avoid confusion.
  - **File**: `backend/internal/services/finance_service.go:48`
  - **Fix**: Keep one canonical check at handler level; service assumes caller authorized.
- **[Error Contract]** Service BadRequest errors not propagated correctly to handler.
  - Handler checks `strings.HasPrefix(err.Error(), "bad_request:")` but `ServiceError.Error()` returns message only, not prefixed. Result: validation errors returned as 500.
  - **File**: `backend/internal/api/finance_handler.go:77`
  - **Fix**: Check error type via `errors.As` for `ServiceError` and inspect `Code` field, or wrap errors with prefix.

### Medium
- **[API9-Inventory]** Missing OpenAPI annotations for `DELETE /finance` before fix. Now added.
  - **File**: `backend/internal/api/finance_handler.go:24`
  - **Fix**: Added Swagger annotations with admin role requirement.
- **[API4-Consumption]** Service pagination cap reduced from 200 to 100 to meet security standard.
  - **File**: `backend/internal/services/finance_service.go:84`
  - **Fix**: Applied.

### Low
- Success response returns `errors: fiber.Map{}` which serializes to `{}`. Consistent with contract but could reuse canonical empty map.
- Error code `insufficient_permissions` for forbidden is custom; consider standard `forbidden` code.

## Remediation Summary
- Added OpenAPI annotations requiring admin role.
- Fixed error contract for auth response structs.
- Reduced pagination cap to 100.
- Created unit tests for admin role enforcement.
- Closed gaps: Finance List Pagination Uncapped, MagicLinkResponse error contract violation.

## Next Steps
- Refactor error propagation to use typed errors instead of string prefix checks.
- Audit remaining services for pagination caps >100.
