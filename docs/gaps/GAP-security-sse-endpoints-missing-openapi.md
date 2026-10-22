# Gap: SSE Endpoints Missing OpenAPI Annotations

**Area:** backend
**Severity:** Low
**Status:** Open
**Owner:** 
**Created:** 2025-10-19
**Related Issues:** #security-audit-2025-10-19

## Description
Server-Sent Events (SSE) endpoints for bulk invitation job progress (`GET /api/admins/invitations/jobs/:job_id/events`, `GET /api/teachers/invitations/jobs/:job_id/events`, etc.) lack OpenAPI/Swagger annotations.

## Current Behavior
In `admin_invitation_handler.go:238` and `teacher_invitation_handler.go:238`:
```go
// Events method has NO Swagger annotations
func (h *AdminInvitationHandler) Events(c fiber.Ctx) error {
    // ...
}
```

## Expected Behavior
SSE endpoints should have annotations documenting:
- `@Summary` / `@Description`
- `@Param job_id path string true "Job ID"`
- `@Success 200 {string} string "text/event-stream"`
- `@Header Content-Type {string} text/event-stream`
- `@Security ApiKeyAuth`

## Impact
- **API Inventory**: Endpoints not documented in generated OpenAPI spec
- **Client Generation**: Frontend SDK missing SSE types
- **Compliance**: OWASP API9:2023 — Improper Inventory Management

## Reproduction Steps
1. Run `swag init` or view generated OpenAPI spec
2. Search for `/events` paths — not present
3. Compare with other endpoints — all have annotations

## Proposed Solutions
- **Option A (Add annotations)**: Add Swagger comments to `Events` methods
- **Option B (SSE-specific extension)**: Use `x-sse-event` extension for event types

## Decision
Pending — recommend Option A.

## Acceptance Criteria
- [ ] `Events` method in admin, teacher, finance, guardian, student import handlers have Swagger annotations
- [ ] Generated OpenAPI spec includes SSE endpoints
- [ ] `@Security ApiKeyAuth` present
- [ ] Response documents event stream format

## Links
- Security audit report: `security-report-extended.md`
- OWASP API9:2023 reference: `docs/reference/owasp-api-top10-2023.md#api92023---improper-inventory-management`
- Admin handler: `backend/internal/api/admin_invitation_handler.go:238`
- Teacher handler: `backend/internal/api/teacher_invitation_handler.go:238`