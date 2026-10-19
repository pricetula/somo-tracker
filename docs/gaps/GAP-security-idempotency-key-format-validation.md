# Gap: Idempotency Key Not Validated for Format

**Area:** backend
**Severity:** High
**Status:** Open
**Owner:** 
**Created:** 2025-10-19
**Related Issues:** #security-audit-2025-10-19

## Description
Bulk invitation endpoints require an `Idempotency-Key` header but do not validate its format. Weak keys (short, predictable) could collide; overly long keys waste Redis memory.

## Current Behavior
In `admin_invitation_handler.go:93` and `teacher_invitation_handler.go:93`:
```go
idempotencyKey := c.Get("Idempotency-Key")
if idempotencyKey == "" {
    return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
        "code": "bad_request", "message": "Idempotency-Key header is required", "errors": fiber.Map{"Idempotency-Key": []string{"required"}},
    })
}
```
No format validation — accepts any non-empty string.

## Expected Behavior
Idempotency key must be a valid UUID (v4) or meet minimum entropy requirements (min 32 chars, max 128, alphanumeric + hyphen).

## Impact
- **Collision Risk**: Short keys (e.g., "1", "retry") collide across users/requests
- **Memory Waste**: Very long keys (10KB+) stored in Redis for deduplication
- **Predictability**: Sequential keys allow enumeration of existing jobs

## Reproduction Steps
1. POST `/api/admins/invitations` with `Idempotency-Key: 1` → accepted
2. POST same endpoint with `Idempotency-Key: 1` → returns existing job (collision)
3. POST with 5000-char key → stored in Redis

## Proposed Solutions
- **Option A (UUID validation)**: Require RFC4122 UUID format: `^[a-f0-9-]{36}$`
- **Option B (Entropy-based)**: Min 32 chars, max 128, regex `^[a-zA-Z0-9\-_]+$`
- **Option C (Client-generated UUID)**: Document that clients MUST generate UUID v4

## Decision
Pending — recommend Option A (UUID) for simplicity and collision resistance.

## Acceptance Criteria
- [ ] Validation added: `Idempotency-Key` must match UUID regex
- [ ] Returns 400 with `errors: {"Idempotency-Key": ["must be a valid UUID"]}` for invalid format
- [ ] Integration test: invalid key → 400
- [ ] Integration test: valid UUID → 202
- [ ] Applied to all bulk endpoints (admin, teacher, finance, guardian, student import)

## Links
- Security audit report: `security-report-extended.md`
- OWASP API6:2023 reference: `docs/reference/owasp-api-top10-2023.md#api62023---unrestricted-access-to-sensitive-business-flows`
- Admin handler: `backend/internal/api/admin_invitation_handler.go:93`
- Teacher handler: `backend/internal/api/teacher_invitation_handler.go:93`