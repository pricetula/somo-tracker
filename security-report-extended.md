# Security Audit Report — Extended Run (Sample Endpoints)

**Date**: 2025-10-19
**Auditor**: pi-agent (security-audit skill)
**Scope**: Extended audit of 15 endpoints across 5 handler files

---

## Test Endpoints Audited

### Auth (3)
1. `POST /api/auth/magic-link/send` (auth_handler.go)
2. `GET /api/auth/callback` (auth_handler.go)
3. `POST /api/auth/logout` (auth_handler.go)

### Students (4)
4. `GET /api/students` (students_handler.go:ListStudents)
5. `GET /api/students/:id` (students_handler.go:GetStudent)
6. `PATCH /api/students/:id` (students_handler.go:UpdateStudent)
7. `DELETE /api/students` (students_handler.go:DeleteStudents)

### Bulk Invitations — Admin (4)
8. `POST /api/admins/invitations` (admin_invitation_handler.go:HandleInvites)
9. `GET /api/admins/invitations/jobs/:job_id` (admin_invitation_handler.go:GetJob)
10. `POST /api/admins/invitations/jobs/:job_id/retry-failed` (admin_invitation_handler.go:RetryFailed)
11. `GET /api/admins/invitations/jobs/:job_id/events` (admin_invitation_handler.go:Events)

### Bulk Invitations — Teacher (4) — *Nearly identical to Admin*
12. `POST /api/teachers/invitations` (teacher_invitation_handler.go:HandleInvites)
13. `GET /api/teachers/invitations/jobs/:job_id` (teacher_invitation_handler.go:GetJob)
14. `POST /api/teachers/invitations/jobs/:job_id/retry-failed` (teacher_invitation_handler.go:RetryFailed)
15. `GET /api/teachers/invitations/jobs/:job_id/events` (teacher_invitation_handler.go:Events)

### Finance (2)
16. `GET /api/finance` (finance_handler.go:ListFinance)
17. `DELETE /api/finance` (finance_handler.go:DeleteFinance)

### Timetable (5)
18. `GET /api/timetable/templates` (timetable_handler.go:ListTemplates)
19. `POST /api/timetable/templates` (timetable_handler.go:CreateTemplate)
20. `GET /api/timetable/templates/:id` (timetable_handler.go:GetTemplate)
21. `PATCH /api/timetable/templates/:id` (timetable_handler.go:UpdateTemplate)
22. `DELETE /api/timetable/class-timetable-slots/:id` (timetable_handler.go:DeleteClassTimetableSlot)
23. `POST /api/timetable/setup` (timetable_handler.go:SetupClassTimetableSlot)
24. `GET /api/timetable/templates/:id/slots` (timetable_handler.go:ListTimeSlotsByTemplate)
25. `GET /api/timetable/templates/:id/classes/:classId/slots` (timetable_handler.go:GetClassSlotsByTemplate)

---

## Findings

### Critical

#### [API5-BFLA] Bulk invitation endpoints — Role check only at handler, not service
- **Files**: `admin_invitation_handler.go:58`, `teacher_invitation_handler.go:58`
- **Issue**: `UserHasAdminRole` checked in handler, but service methods don't re-verify. If service called directly (tests, internal), bypass possible.
- **Current**: Handler checks `h.svc.UserHasAdminRole(ctx, schoolID, adminUserID)` → returns 403 if false.
- **Risk**: Defense in depth missing. Service should also validate caller permissions.
- **Fix**: Add authorization check in service layer `CreateBulkJobWithItems()`.

#### [API1-BOLA] `DELETE /api/timetable/class-timetable-slots/:id` — No ownership validation
- **File**: `timetable_handler.go:168` (DeleteClassTimetableSlot)
- **Issue**: Deletes slot by ID only, no `schoolID` check. Slot could belong to another school.
- **Current**: `h.svc.DeleteClassTimetableSlot(ctx, slotID)` — only slot ID passed.
- **Fix**: Pass `schoolID` from context; service must verify slot belongs to school.

#### [API1-BOLA] `GET /api/timetable/templates/:id/classes/:classId/slots` — Class ID not validated against school
- **File**: `timetable_handler.go:145` (GetClassSlotsByTemplate)
- **Issue**: Reads `schoolID` from context but passes `classID` directly to service without verifying class belongs to school.
- **Fix**: Service must validate `classID` → `schoolID` ownership.

### High

#### [API4-Consumption] Bulk invitations — 10,000 row limit per request (too high)
- **Files**: `admin_invitation_handler.go:85`, `teacher_invitation_handler.go:85`
- **Issue**: `if len(req.Invitations) > 10000` allows 10K emails in single request.
- **Impact**: Memory pressure, long processing time, queue flooding (100 batches of 100).
- **Standard**: Cap at 1000 rows/request (10 batches max).
- **Fix**: Change limit to 1000; add pagination for larger imports.

#### [API4-Consumption] `GET /api/finance` — Pagination limit uncapped (no validation)
- **File**: `finance_handler.go:88` (ListFinance)
- **Issue**: `limit, _ := strconv.Atoi(c.Query("limit", "50"))` — no upper bound check.
- **Risk**: Client can request `?limit=10000` → full table scan.
- **Fix**: Add `if limit > 100 { limit = 100 }` or return 400.

#### [API4-Consumption] `GET /api/timetable/templates` — No pagination at all
- **File**: `timetable_handler.go:52` (ListTemplates)
- **Issue**: Returns all templates for school unbounded.
- **Fix**: Add `page`, `limit` params with cap at 100.

#### [API6-Flow] Bulk invitations — Idempotency key required but not validated for format
- **Files**: `admin_invitation_handler.go:93`, `teacher_invitation_handler.go:93`
- **Issue**: `idempotencyKey := c.Get("Idempotency-Key")` — accepts any string, no format validation (UUID, length).
- **Risk**: Weak keys (e.g., "1") could collide; long keys waste Redis memory.
- **Fix**: Validate format: `^[a-f0-9-]{36}$` (UUID) or min 32 chars, max 128.

#### [API6-Flow] Bulk invitations — Duplicate email silently dropped (first wins)
- **Files**: `admin_invitation_handler.go:115`, `teacher_invitation_handler.go:115`
- **Issue**: `seen` map deduplicates case-insensitive; duplicates skipped without warning.
- **Risk**: User submits 100 rows with 50 duplicates → 50 invitations sent, no feedback.
- **Fix**: Return warning in response with count of deduplicated rows.

#### [API2-Auth] Invitation handlers — Raw `fmt.Sprintf("%v", local)` for UUID parsing
- **Files**: All invitation handlers (lines 45-52)
- **Issue**: `uuid.Parse(fmt.Sprintf("%v", schoolIDStr))` — if local is wrong type, parses garbage.
- **Better**: Type assertion `schoolIDStr, ok := c.Locals("active_school_id").(string)`
- **Fix**: Use safe type assertion like other handlers.

#### [API7-SSRF] SSE endpoint (`/events`) — Redis channel name from user input
- **Files**: `admin_invitation_handler.go:270`, `teacher_invitation_handler.go:270`
- **Issue**: `channel := "bulk_progress_" + jobIDStr` — jobID from path param, validated as UUID. **PASS** — UUID validation prevents injection.
- **Note**: Good pattern — validate before using in Redis pub/sub.

### Medium

#### [API3-BPLA] `POST /api/timetable/templates` — Inline request struct with nested slice
- **File**: `timetable_handler.go:26` (createTimetableTemplateRequest)
- **Issue**: `TimeSlots []struct{...}` — inline struct, no validation tags on nested fields.
- **Risk**: Mass assignment if fields added; no `validate` tags on `StartTime`, `EndTime`.
- **Fix**: Extract to named struct with validation tags.

#### [API3-BPLA] `POST /api/timetable/setup` — Inline struct, no validation on UUID fields
- **File**: `timetable_handler.go:203` (SetupClassTimetableSlot)
- **Issue**: `ClassRoomID`, `TimeSlotID`, `SubjectID`, `TeacherMembershipID` as strings — no UUID format validation.
- **Fix**: Use `uuid.UUID` type or validate format.

#### [API8-Misconfig] Invitation handlers — Inline error responses instead of `WriteError()`
- **Files**: `admin_invitation_handler.go:62, 78, 88, 97, 123, 130` (and teacher equivalents)
- **Issue**: Uses `c.Status(...).JSON(fiber.Map{...})` directly instead of `WriteError(c, ErrBadRequest(...))`.
- **Impact**: Inconsistent error shape; missing `request_id`; `Errors` map may be `nil`.
- **Fix**: Use `WriteError()` with `ErrBadRequest()`, `ErrForbidden()`, `ErrInternal()`.

#### [API8-Misconfig] Finance handler — Mix of `WriteError()` and inline responses
- **File**: `finance_handler.go` — `DeleteFinance` uses `WriteError` for some, inline for auth checks.
- **Issue**: Auth checks return inline `fiber.Map` without `request_id`.
- **Fix**: Use `ErrUnauthorized()`, `ErrBadRequest()` consistently.

#### [API8-Misconfig] Timetable handler — `GetTemplate` returns `ErrBadRequest` for not found
- **File**: `timetable_handler.go:85` (GetTemplate)
- **Issue**: `return WriteError(c, ErrBadRequest(err.Error(), nil))` — should be `ErrNotFound`.
- **Fix**: Service should return typed error; handler maps to correct HTTP code.

#### [API9-Inventory] SSE endpoints missing OpenAPI annotations
- **Files**: `admin_invitation_handler.go:238`, `teacher_invitation_handler.go:238` (Events)
- **Issue**: No `@Summary`, `@Param`, `@Success` for SSE endpoints.
- **Fix**: Add Swagger annotations for EventSource responses.

#### [API10-ThirdParty] Stytch client used directly in handler (admin invitation)
- **File**: `admin_invitation_handler.go:19` — `stytch stytch.Inviter` interface
- **Assessment**: **PASS** — Uses interface, not concrete. Secrets in config. Webhook verification in service.

### Low

#### [API9-Inventory] `ListTemplates` missing `@Security ApiKeyAuth`
- **File**: `timetable_handler.go:48`
- **Issue**: No security annotation on protected endpoint.

#### [API9-Inventory] `GetTemplate` returns 400 for not found (should be 404)
- **File**: `timetable_handler.go:85`
- **Issue**: Swagger shows `@Failure 400` but semantic should be 404.

#### [API5-BFLA] `DELETE /api/finance` — Checks current user ID but no role verification
- **File**: `finance_handler.go:26` (DeleteFinance)
- **Issue**: Gets `currentUserID` from context but doesn't verify admin role.
- **Note**: Service `DeleteFinance` may check, but handler should defense-in-depth.

---

## Summary by Category

| Category | Critical | High | Medium | Low |
|----------|----------|------|--------|-----|
| API1 BOLA | 2 | 0 | 0 | 0 |
| API2 Auth | 0 | 1 | 0 | 0 |
| API3 BPLA | 0 | 0 | 2 | 0 |
| API4 Consumption | 0 | 4 | 0 | 0 |
| API5 BFLA | 1 | 0 | 0 | 1 |
| API6 Business Flows | 0 | 2 | 0 | 0 |
| API7 SSRF | 0 | 0 | 0 | 0 |
| API8 Misconfig | 0 | 0 | 4 | 0 |
| API9 Inventory | 0 | 0 | 1 | 2 |
| API10 Third-Party | 0 | 0 | 0 | 0 |
| **Total** | **3** | **7** | **7** | **3** |

---

## Remediation Priority

### Critical (Fix Immediately)
1. **BOLA**: `DeleteClassTimetableSlot` — add schoolID ownership check
2. **BOLA**: `GetClassSlotsByTemplate` — validate class belongs to school
3. **BFLA**: Bulk invitation service layer — add authorization check

### High (Fix This Sprint)
4. **Consumption**: Bulk invitation row limit 10000 → 1000
5. **Consumption**: `ListFinance` pagination cap (add 100 max)
6. **Consumption**: `ListTemplates` add pagination
7. **Flow**: Idempotency key format validation
8. **Flow**: Duplicate email warning in bulk invite response
9. **Auth**: Safe type assertion for session locals

### Medium (Fix Next Sprint)
10. **BPLA**: Extract timetable request DTOs with validation
11. **Misconfig**: Use `WriteError()` consistently in invitation handlers
12. **Misconfig**: Finance handler error consistency
13. **Misconfig**: `GetTemplate` 400 → 404
14. **Inventory**: Add OpenAPI annotations for SSE endpoints

### Low (Tech Debt)
15. **Inventory**: Add `@Security` annotations
16. **BFLA**: Finance delete role check

---

## Positive Patterns Observed (Compliant)

| Pattern | Location | Status |
|---------|----------|--------|
| **Tenant isolation** | All handlers read `active_school_id`, `tenant_id` from `c.Locals()` | ✅ |
| **Role-based access** | Invitation handlers check `UserHasAdminRole` | ✅ |
| **Idempotency keys** | Required on bulk mutations | ✅ |
| **Input validation** | Email parsing, UUID parsing, required fields | ✅ |
| **Deduplication** | Case-insensitive email dedup in bulk invites | ✅ |
| **Async processing** | Asynq batches of 100, Redis progress pub/sub | ✅ |
| **Error logging** | Structured logging with zap, no sensitive data | ✅ |
| **Rate limiting** | Per-tenant bulk invite limit (5/hr) in router | ✅ |
| **Canonical errors** | Students, timetable handlers use `WriteError()` | ✅ |
| **Request DTOs** | Dedicated structs (not DB models) | ✅ |

---

## Compliance with Project Contracts

- ✅ **Error contract**: Most handlers use `WriteError()` with canonical shape
- ✅ **Auth middleware**: Session middleware injects `user_id`, `tenant_id`, `active_school_id`
- ✅ **Rate limiting**: Tiered (IP, email, tenant) via Redis
- ✅ **DTO separation**: Request/response DTOs distinct from sqlc models
- ✅ **No mass assignment**: Update DTOs use pointers + `omitempty`
- ✅ **Tenant isolation**: Queries scoped by `schoolID` from session
- ✅ **CSRF protection**: Double-submit cookie on mutating endpoints (protected group)
- ⚠️ **Security headers**: Helmet registered; CSP config needs verification
- ⚠️ **Body limits**: Public auth routes may miss global limit (see GAP)
- ⚠️ **Pagination caps**: Inconsistent (50, 200, uncapped, none)

---

## Recommended Follow-up Audits

1. **Guardian/Finance invitations** — Same pattern as admin/teacher
2. **Student import** — Bulk CSV upload, async, similar risks
3. **Attendance/Events** — Write endpoints, conflict detection
4. **Webhook endpoints** — Stytch callbacks, signature verification
5. **All list endpoints** — Systematic pagination cap audit (target: 100 max)

---

## Gap Tracking

| Gap ID | Title | Severity | Status |
|--------|-------|----------|--------|
| GAP-security-pagination-cap-too-high | Students list limit 200 | High | Open |
| GAP-security-missing-body-limit-auth-routes | Auth routes body limit | High | Open |
| GAP-security-magic-link-ip-rate-limit-too-permissive | Magic-link IP rate 10/min | High | Open |
| GAP-security-error-contract-violation-magiclink | MagicLinkResponse Errors type | Medium | Open |
| GAP-security-csp-header-unverified | CSP header config | Medium | Open |
| **NEW** | Bulk invitation 10K row limit | High | — |
| **NEW** | Finance list pagination uncapped | High | — |
| **NEW** | Timetable templates no pagination | High | — |
| **NEW** | Timetable slot delete BOLA | Critical | — |
| **NEW** | Class slots BOLA (no school check) | Critical | — |
| **NEW** | Invitation service missing authz | Critical | — |
| **NEW** | Idempotency key format validation | High | — |
| **NEW** | Duplicate email silent drop | High | — |
| **NEW** | Unsafe type assertion in invitations | High | — |
| **NEW** | Timetable inline DTOs no validation | Medium | — |
| **NEW** | Invitation handlers inline errors | Medium | — |
| **NEW** | Finance handler error inconsistency | Medium | — |
| **NEW** | GetTemplate 400 vs 404 | Medium | — |
| **NEW** | SSE endpoints missing OpenAPI | Low | — |

**Total New Gaps**: 14 (3 Critical, 6 High, 5 Medium, 2 Low)