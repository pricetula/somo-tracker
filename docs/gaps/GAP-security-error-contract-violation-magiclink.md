# Gap: MagicLinkResponse Violates Canonical Error Contract

**Area:** backend
**Severity:** Medium
**Status:** Open
**Owner:** 
**Created:** 2025-10-19
**Related Issues:** #security-audit-2025-10-19

## Description
The `MagicLinkResponse` struct in `backend/internal/api/auth_handler.go:18` defines `Errors` as `map[string]interface{}`, but the canonical error contract (enforced by `error-contract` skill and `backend/internal/api/errors.go`) requires `map[string][]string`.

## Current Behavior
```go
type MagicLinkResponse struct {
    Code    string                 `json:"code"`
    Message string                 `json:"message"`
    Errors  map[string]interface{} `json:"errors"`  // WRONG TYPE
}
```

The handler returns `fiber.Map{"errors": fiber.Map{}}` which serializes to empty object `{}`, but the struct definition allows arbitrary value types.

## Expected Behavior
All error response structs must use `Errors map[string][]string` to match:
- `backend/internal/api/errors.go` `APIError` struct
- Frontend client expectation (`frontend/src/lib/api/client.ts`)
- OpenAPI spec generation

## Impact
- **Contract Violation**: Frontend error handling expects array of strings per field
- **Type Safety**: Go struct allows invalid types that would break frontend parsing
- **Consistency**: Other handlers use correct `map[string][]string` type
- **Security Audit**: Flagged as API8-Misconfig (inconsistent error format)

## Reproduction Steps
1. Inspect `MagicLinkResponse` struct definition
2. Compare with `APIError` in `errors.go:10` — `Errors map[string][]string`
3. Check frontend error parser expects `errors: Record<string, string[]>`
4. Run `make build-backend` — no compile error (interface{} accepts anything)

## Proposed Solutions
- **Option A (Fix struct)**: Change `Errors map[string]interface{}` to `Errors map[string][]string`
- **Option B (Remove struct)**: Use inline `fiber.Map` like other handlers (e.g., `logout` returns `fiber.Map{"code": "...", "errors": fiber.Map{}}`)
- **Option C (Shared type)**: Create `ErrorResponse` type in `errors.go` and reuse

## Decision
Pending — recommend Option A for minimal change, Option C for long-term consistency.

## Acceptance Criteria
- [ ] `MagicLinkResponse.Errors` changed to `map[string][]string`
- [ ] `CallbackResponse.Errors` and `LogoutResponse.Errors` also fixed (same issue)
- [ ] Handler returns use `fiber.Map{"errors": map[string][]string{}}` or `fiber.Map{}` 
- [ ] Frontend error handling tests pass
- [ ] `make build-backend` compiles without errors
- [ ] Error contract test (`error_contract_test.go`) covers auth responses

## Links
- Security audit report: `security-report-test.md`
- Error contract: `backend/internal/api/errors.go`
- Error contract test: `backend/internal/api/error_contract_test.go`
- Frontend API client: `frontend/src/lib/api/client.ts`
- OWASP API8:2023 reference: `docs/reference/owasp-api-top10-2023.md#api82023---security-misconfiguration`