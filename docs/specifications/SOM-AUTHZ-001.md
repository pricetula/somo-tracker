---
id: SOM-AUTHZ-001
title: Authz Permission Matrix
status: implemented
author: agent
date: 2026-11-11
updated: 2026-11-11
fr_ids: [BE-020, BE-021, BE-016, BE-017, BE-018]
nf_ids: [BE-031, BE-032]
api_contract: no
migrations: no
generated_files: []
---

## 1. Summary

Replace the placeholder `platform/authz` package with a real permission-based authorization system. Every route registers with `authz.Require(perm)` or an explicit `Public()` (BE-020). Services obtain a `Scope` from `authz.ScopeFor(ctx, perm)` and pass it to the store (BE-021). The permission matrix is data in `platform/authz`, not role lists in handlers.

Also aligns role naming between the database enum (`OWNER, ADMIN, HEADTEACHER, TEACHER, GUARDIAN, FINANCE, PLATFORMOPERATOR`) and the RBAC middleware's role strings (`tenant_owner`, `school_admin`, `finance_officer`, `teacher`, `guardian`), which currently diverge.

## 2. Requirements

| FR/NFR ID | Rationale |
|-----------|-----------|
| BE-020 | Every route must register with `authz.Require(perm)` or explicit `Public()`; permissions never role lists; matrix is data in `platform/authz` |
| BE-021 | Services obtain `Scope` from `authz.ScopeFor(ctx, perm)` and pass to store; out-of-scope single records return 404 |
| BE-016 | Every query is tenant-scoped; cross-tenant returns identical 404 |
| BE-017 | Session context (user, tenant, school, role) loaded once per request; never accept school/tenant ID from client |
| BE-018 | Role always from live membership; never cached in session or Redis |
| BE-031 | Log IDs (request, tenant, school, user), route pattern, status, latency, error code — never PII |
| BE-032 | Sentry: panics and 5xx only; scrub PII; attach request/tenant/school/user IDs as tags |

## 3. Domain Model Affected

No database entities change. The `memberships` table's `role` column (`user_role` enum: `OWNER, ADMIN, HEADTEACHER, TEACHER, GUARDIAN, FINANCE, PLATFORMOPERATOR`) is already the source of truth for roles.

The authz package introduces in-memory data structures (no persistence):

- `Permission` — string permission identifier
- `rolePermissions` — map from role → set of permissions (the permission matrix)
- `Require(perm)` — Fiber handler that checks session permissions
- `ScopeFor(ctx, perm)` — returns tenant-scoped context key for store queries

## 4. API Contract

No OpenAPI contract change (no endpoints added/removed). The authz middleware is applied at the route-group level in `app.go`, consistent with existing `/api/v1` route structure.

### Permissions

| Permission | Description | Roles granted |
|------------|-------------|---------------|
| `auth:read` | Read own session/profile | all authenticated roles |
| `auth:logout` | Log out / revoke session | all authenticated roles |
| `school:read` | Read school data | OWNER, ADMIN, HEADTEACHER, TEACHER, GUARDIAN, FINANCE, PLATFORMOPERATOR |
| `school:write` | Create/update school | OWNER, ADMIN, PLATFORMOPERATOR |
| `user:read` | Read user data | OWNER, ADMIN, HEADTEACHER, FINANCE, PLATFORMOPERATOR |
| `user:write` | Create/update users | OWNER, ADMIN, PLATFORMOPERATOR |
| `student:read` | Read student records | OWNER, ADMIN, HEADTEACHER, TEACHER, GUARDIAN (own children), FINANCE, PLATFORMOPERATOR |
| `student:write` | Create/update students | OWNER, ADMIN, HEADTEACHER, PLATFORMOPERATOR |
| `academic:read` | Read academic data (subjects, topics, etc.) | OWNER, ADMIN, HEADTEACHER, TEACHER, PLATFORMOPERATOR |
| `academic:write` | Create/update academic data | OWNER, ADMIN, PLATFORMOPERATOR |
| `enrollment:read` | Read enrollments | OWNER, ADMIN, HEADTEACHER, TEACHER, GUARDIAN (own children), PLATFORMOPERATOR |
| `enrollment:write` | Create/update enrollments | OWNER, ADMIN, HEADTEACHER, PLATFORMOPERATOR |
| `fee:read` | Read fee records | OWNER, ADMIN, HEADTEACHER, FINANCE, PLATFORMOPERATOR |
| `fee:write` | Create/update fee records | OWNER, ADMIN, FINANCE, PLATFORMOPERATOR |
| `health:read` | Read health records | OWNER, ADMIN, HEADTEACHER, TEACHER, GUARDIAN (own children), PLATFORMOPERATOR |
| `health:write` | Create/update health records | OWNER, ADMIN, HEADTEACHER, PLATFORMOPERATOR |
| `behavior:read` | Read behavior records | OWNER, ADMIN, HEADTEACHER, TEACHER, GUARDIAN (own children), PLATFORMOPERATOR |
| `behavior:write` | Create/update behavior records | OWNER, ADMIN, HEADTEACHER, PLATFORMOPERATOR |
| `audit:read` | Read audit logs | OWNER, ADMIN, PLATFORMOPERATOR |
| `system:manage` | System configuration | PLATFORMOPERATOR |

### Role naming alignment

Database enum values (SCREAMING_SNAKE per BE-025): `OWNER, ADMIN, HEADTEACHER, TEACHER, GUARDIAN, FINANCE, PLATFORMOPERATOR`

Middleware role strings (currently `tenant_owner`, `school_admin`, `finance_officer`, `teacher`, `guardian`) will be aligned to the database enum values, lowercased for use as strings in `c.Locals("roles")`: `owner`, `admin`, `headteacher`, `teacher`, `guardian`, `finance`, `platformoperator`.

## 5. Workflow (Given/When/Then)

### Route authorization

**Given** a route registered with `authz.Require(perm: "student:read")`
**When** an authenticated user with role `teacher` makes a request to that route
**Then** the request proceeds (teacher has `student:read`)

**Given** a route registered with `authz.Require(perm: "student:write")`
**When** an authenticated user with role `teacher` makes a request
**Then** the request returns 403 `FORBIDDEN` (teacher lacks `student:write`)

**Given** a route registered with `authz.Public()`
**When** an unauthenticated request arrives
**Then** the request proceeds without session validation

### Scope-based store queries

**Given** a service calls `authz.ScopeFor(ctx, "student:read")`
**When** the session tenant is `tenant-123`
**Then** the returned scope key is `tenant:tenant-123:student:read` and the store applies `WHERE tenant_id = 'tenant-123'` (or school-scoped equivalent)

**Given** a service requests a single record by ID
**When** the record belongs to a different tenant
**Then** the store returns 404 (identical to nonexistent record, per BE-016)

### No client-scope injection

**Given** a request includes a school ID in the body or a header
**When** the handler attempts to use it as a scope
**Then** the request is rejected; scope comes only from session context (BE-017)

## 6. Implementation Plan

### Backend

1. **`backend/internal/platform/authz/authz.go`** — Replace placeholder:
   - Define `Permission` type (already exists)
   - Define `rolePermissions` map: role → set of `Permission`
   - Implement `Require(perm Permission) fiber.Handler` — reads session from `c.Locals` (set by `AuthMiddleware.RequireAuth`), checks if user's roles collectively have `perm`; returns 403 `FORBIDDEN` if not
   - Implement `ScopeFor(ctx context.Context, perm Permission) string` — reads tenant ID from context, returns `tenant:<tenant-id>:<perm>`; also handles school-scoped perms
   - Add `Public()` marker type/func for route groups that don't require auth
   - Add `HasPermission(ctx, perm) bool` helper for service-layer checks

2. **`backend/internal/platform/authz/authz_test.go`** — Table-driven tests:
   - Each role has expected permissions
   - `Require` allows when role has perm, denies when not
   - `ScopeFor` returns correct key format
   - Unknown role returns empty permission set

3. **`backend/internal/app/app.go`** — Wire authz into route groups:
   - Protected group: `protectedAPI.Use(authz.RequireAuthWrapper())` — wait, RequireAuth already exists in identity middleware. The authz `Require` is applied per-route or per-group. Actually, per BE-020, each route registers with `authz.Require(perm)` or `Public()`. So protected domain handlers should call `authz.Require(specificPerm)` at registration.
   - Remove `_ = rbacMiddleware` unused import once RBAC middleware is replaced or kept as-is for object-level checks
   - The existing `RBACMiddleware` (RequireRole, etc.) can remain for object-level access (guardian/teacher) but route-level authz uses `authz.Require(perm)`

4. **`backend/internal/identity/middleware/rbac.go`** (new or update `auth.go`) — Align role strings with database enum:
   - `tenant_owner` → `owner`
   - `school_admin` → `admin`
   - `finance_officer` → `finance`
   - `teacher` → `teacher` (unchanged)
   - `guardian` → `guardian` (unchanged)
   - Update all `RequireRole` calls and tests accordingly

### Frontend

No changes required (authz is backend-only; frontend role handling via `/api/v1/me` unchanged).

## 7. Tests Required

### Backend (per BE-036)

- `platform/authz/authz_test.go`:
  - Permission matrix: every role maps to expected permissions (table-driven)
  - `Require(perm)`: allows when any user role has perm; 403 otherwise
  - `Require(perm)`: 401 when no session (no `c.Locals("roles")`)
  - `ScopeFor(ctx, perm)`: returns `tenant:<id>:<perm>` format; reads tenant from `httpserver.TenantIDKey{}` context
  - Unknown role: empty permission set, all `Require` calls deny
- `platform/authz/` contract test: every route group in `app.go` either uses `authz.Require(perm)` or explicit `Public()` (no unprotected routes under `/api/v1` except health)
- `app.go` protected group: verify all registered handlers have an `authz.Require(perm)` wrapper
- Identity middleware tests updated for role name alignment (`owner`, `admin`, `finance`, etc.)

### Frontend (per BE-026 / FE-026)

- No frontend tests required for this spec (backend-only authz)

### Integration

- Store and RLS tests use real PostgreSQL (testcontainers) per BE-036: cross-tenant queries return identical 404

## 8. Open Questions

1. **Role naming:** Should middleware role strings exactly match lowercase database enum values (`owner`, `admin`, `headteacher`, `teacher`, `guardian`, `finance`, `platformoperator`), or keep the current `tenant_owner`, `school_admin`, `finance_officer` aliases? Recommendation: align to lowercase enum values for single source of truth.

2. **Object-level vs route-level:** The existing `RBACMiddleware` (`RequireGuardianAccess`, `RequireTeacherAccess`, `RequireSchoolAccess`) does object-level checks via service calls. Should these be migrated to permission-based checks (`authz.Require("student:read")` + object-level scope in service), or kept as-is alongside the new permission matrix? Recommendation: keep RBAC middleware for object-level checks; add `authz.Require(perm)` for route-level checks.

3. **Guardian-specific scoping:** Guardian has `student:read` but only for linked students. The permission matrix grants `student:read` to guardian, but object-level enforcement (via `guardian_student_links`) happens in the service/store layer. Is this clear enough, or does the spec need a separate "object scope" concept?

4. **Multiple schools per user:** A user can have memberships in multiple schools. `authz.Require(perm)` checks role across all active memberships. Should scope be per-school (user must have perm in the specific school being accessed)? The `ScopeFor` returns tenant-level scope; school-level scope is handled by `TenantMiddleware` + RLS. Confirm this is sufficient.

## 9. Links

- [BE-020](../requirements/backend.md) — permissions, never role lists
- [BE-021](../requirements/backend.md) — ScopeFor, store passes scope
- [BE-016](../requirements/backend.md) — tenant-scoped queries, identical 404
- [BE-017](../requirements/backend.md) — session context from middleware, no client scope
- [BE-018](../requirements/backend.md) — role from live membership
- [backend/internal/platform/authz/authz.go](../../../backend/internal/platform/authz/authz.go) — placeholder to replace
- [backend/internal/app/app.go](../../../backend/internal/app/app.go) — route groups to wire
- [backend/internal/identity/middleware/auth.go](../../../backend/internal/identity/middleware/auth.go) — existing AuthMiddleware + RBACMiddleware
