# Gap: Error Contract Drift & Silent Failures in Handlers

**Area:** backend
**Severity:** High
**Status:** Done
**Owner:** Platform team
**Created:** 2026-10-15
**Closed:** 2026-10-15
**Related Issues:** #

## Description
Multiple handlers bypass the canonical error contract in `backend/internal/api/errors.go` and `backend/AGENTS.md`. Errors are returned as ad-hoc JSON, `uuid.Parse` errors are silently discarded with `_`, and errors are logged and returned inconsistently.

## Current Behavior
* `backend/internal/api/students_handler.go` `GetStudent`:
  ```go
  schoolID, _ := uuid.Parse(schoolIDStr)
  ...
  return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"code":"internal_error",...})
  ```
  No `WriteError`, no request_id, silent parse failure.
* Similar patterns exist in other handlers.
* Violates root `AGENTS.md` error handling rules: never silent `_`, never log-and-return, always return up the stack with context.

## Expected Behavior
All non-2xx responses use `WriteError` returning `{code, message, errors, request_id}` snake_case. Parse errors are handled explicitly. Errors are either returned up the stack or logged and acted upon, never both.

## Impact
Inconsistent API responses break frontend `ApiError` handling. Silent parse failures cause hard-to-debug auth/tenant issues. Observability is reduced.

## Reproduction Steps
1. Call `GET /api/students/:id` with malformed `active_school_id` in locals
2. `uuid.Parse` fails silently, `schoolID` is zero value → query returns wrong data or 500
3. Response shape differs from `backend/internal/api/errors.go` contract

## Proposed Solutions
* Replace manual `c.Status(...).JSON(...)` with `WriteError(c, APIError{...})`
* Never ignore errors: handle `uuid.Parse` explicitly, return 400 `bad_request` if invalid
* Audit handlers for silent `_` assignments and empty catch blocks

## Decision
Enforce canonical error contract across all handlers via lint/review checklist.

## Acceptance Criteria
- [x] `StudentsHandler.GetStudent` and `UpdateStudent` use `WriteError`
- [x] No `_` discards on `uuid.Parse` in handlers
- [x] All errors follow `{code, message, errors}` shape
- [x] Frontend `ApiError` client parses responses without fallback

## Links
* Error contract: `backend/internal/api/errors.go`
* Frontend contract: `frontend/src/lib/api/client.ts`
* Root rules: `AGENTS.md` Error Handling
* Example handler: `backend/internal/api/students_handler.go`
