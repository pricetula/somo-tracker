# MVP Launch Audit Report

**Date:** July 2025  
**Scope:** Backend (`./backend`) — Go (Fiber) REST API  
**Reviewer:** AI Agent  
**Status:** ⚠️ **Launch-blocking issues found — remediate before going live**

---

## Table of Contents

1. [🔴 Critical — Launch Blockers](#1--critical--launch-blockers)
2. [🟠 High — Must Fix](#2--high--must-fix)
3. [🟡 Medium — Should Fix](#3--medium--should-fix)
4. [🔵 Low — Post-MVP](#4--low--post-mvp)
5. [📋 Summary Table](#5--summary-table)

---

## 1. 🔴 Critical — Launch Blockers

### C1. Production COOKIE_SECRET defaults to `dev-insecure-change-in-production`

**File:** `internal/config/config.go:56`  
**Severity:** 🔴 BLOCKER  

```go
CookieSecret: getEnv("COOKIE_SECRET", "dev-insecure-change-in-production"),
```

If `COOKIE_SECRET` is not explicitly set in production, the `somo_role` cookie (which frontend uses for routing decisions) is signed with a publicly-known secret. An attacker can forge role cookies to escalate privileges.

**Fix:** Add `COOKIE_SECRET` to production env vars. Add a startup check that panics if the default is used when `APP_ENV != "development"`.

---

### C2. Hardcoded Stytch secret committed to `.env`

**File:** `backend/.env`  
**Severity:** 🔴 BLOCKER  

```
STYTCH_SECRET=secret-test-SPs8RKUzwXQkL6eHYh0jjJq6DDPH3Z637I0=
STYTCH_PROJECT_ID=project-test-d25c4b6a-2f3a-4fff-aeaa-2dda82da9cec
```

A `.env` file with real credentials is committed to the repository. Anyone with repo access can authenticate as any user via the Stytch API.

**Fix:** Remove `.env` from git tracking (add to `.gitignore`). Use environment variables or a secrets manager in production. If these are test credentials, they should still not be committed.

---

### C3. Empty tenant_id hardcoded in student repository INSERTs

**Files:**
- `internal/students/repository.go:184` — `Create()`: `""` as tenant_id
- `internal/students/repository.go:216` — `CreateBatch()`: `""` as tenant_id
- `internal/students/repository.go:280` — `CreateEnrollment()`: `""` as tenant_id, `""` as school_id

**Severity:** 🔴 BLOCKER — **data integrity failure**

```go
// repository.go:184
err := r.pool.QueryRow(ctx, query,
    "",  // tenant_id will be set by the service if needed — in production use real tenant_id
    student.FullName,
    student.Gender,
    ...
```

The comment says "in production use real tenant_id" but the code passes an empty string. If the DB allows it (tenant_id is NOT NULL in `cbc_students`), this violates tenant isolation. **No student created via the API will have a tenant_id.**

Wait — actually the `cbc_students` table has `tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE`. So if `""` is passed, Postgres will try to cast `''` to UUID, which will fail with a type error. So these operations will always fail with a 500 error. **The create/update/enrollment endpoints are broken.**

**Fix:** Thread the real `tenantID` from the handler through the service to the repository.

---

### C4. Financial amounts stored/passed as strings, not decimal types

**Files:**
- `internal/billing/domain.go` — `FeeTemplate.Amount`, `Invoice.AmountDue`, `Invoice.AmountPaid`, `Payment.Amount` are all `string`
- `internal/billing/repository.go` — Amounts are cast via `$1::NUMERIC(12,2)` everywhere

**Severity:** 🔴 BLOCKER  

When amounts are parsed from JSON (string) and cast to NUMERIC, the application layer cannot perform arithmetic or comparisons. If the string is malformed (`"12.5.00"`, `"abc"`), the DB cast will produce a 500 error with no validation in the service layer.

For example, `RecordPayment` accepts: `Amount string` — there is no validation that it's a valid positive number before reaching the DB.

**Fix:** Use `decimal.Decimal` from `github.com/shopspring/decimal` in Go and validate in the service layer before DB insertion.

---

### C5. Dynamic SQL generation in billing repository — potential SQL injection

**File:** `internal/billing/repository.go:73-89`  

```go
func (r *PgRepository) UpdateFeeCategory(ctx context.Context, id, tenantID, schoolID string, name *string, isMandatory *bool) error {
    setClauses := []string{}
    args := []interface{}{}
    argIdx := 1

    if name != nil {
        setClauses = append(setClauses, fmt.Sprintf("name = $%d", argIdx))
        args = append(args, *name)
        argIdx++
    }
    ...
    query := fmt.Sprintf(`
        UPDATE fee_categories
        SET %s
        WHERE id = $%d AND tenant_id = $%d AND school_id = $%d
    `, joinClauses(setClauses, ", "), argIdx, argIdx+1, argIdx+2)
```

While the column names are hardcoded (safe), the `fmt.Sprintf` approach is fragile. More critically, the same pattern in `ListFeeTemplates`:

```go
baseQuery += fmt.Sprintf(" AND academic_term_id = $%d", argIdx)
```

This is not actually injectable since `argIdx` is a number, but it's a dangerous pattern that could be copied. Several other repositories use this `fmt.Sprintf` with `$%d` pattern for dynamic WHERE clauses.

**Fix:** Use a structured query builder or at least add a linting rule forbidding `fmt.Sprintf` with SQL fragments.

---

## 2. 🟠 High — Must Fix

### H1. PARENT role missing from all role-priority CASE statements

**Files (10+ locations):**
- `internal/middleware/security.go:166-171`
- `internal/auth/repository.go:37-42`, `:138-144`, `:295-300`, `:344-349`
- `internal/middleware/auth.go`

**Severity:** 🟠 HIGH  

The `PARENT` role is defined in the `user_role` enum in migration 000001, but every `CASE` statement that orders/prioritizes roles lists only `SYSTEM_ADMIN`, `SCHOOL_ADMIN`, `TEACHER`, `NURSE`, `FINANCE`. `PARENT` silently falls through to the `ELSE` or gets the lowest priority. This means:

1. `PARENT` users may incorrectly have their session role set to `TEACHER` (the fallback default in security.go:175)
2. Role-based access control for parents is effectively broken

**Fix:** Add `WHEN 'PARENT' THEN 6` to every CASE ordering statement.

---

### H2. Migration chk_score_range references undefined function `max_points_check`

**File:** `internal/database/migrations/000003_fix_review_findings.up.sql:131`  

```sql
ALTER TABLE IF EXISTS student_assessment_scores
    ADD CONSTRAINT chk_score_range CHECK (
        raw_score IS NULL OR (raw_score >= 0 AND max_points_check(session_id, raw_score))
    );
```

The constraint references `max_points_check(session_id, raw_score)` — a PostgreSQL function that is **never defined** anywhere in the migrations. This migration succeeds at CREATE time (Postgres doesn't validate CHECK constraint functions until they're actually evaluated), but the constraint is effectively a no-op or will fail at query time.

**Fix:** Either define the `max_points_check` function, or replace with a simpler subquery-based check like:

```sql
raw_score IS NULL OR (raw_score >= 0 AND raw_score <= (SELECT max_points FROM assessment_sessions WHERE id = session_id))
```

---

### H3. students.Repository.Delete deletes enrollments manually despite CASCADE

**File:** `internal/students/repository.go:300-303`  

```go
// Delete enrollments first (cascade)
_, err := r.pool.Exec(ctx, `DELETE FROM cbc_student_enrollments WHERE student_id = $1`, id)
```

The FK `fk_enrollments_tenant_student` has `ON DELETE CASCADE`, so manually deleting enrollments first is redundant. Worse, the manual delete removes **all** enrollments for that student even if the student's school_id doesn't match (there's no tenant/school scope filter on this delete). If a bogus `id` is provided, enrollments from other tenants could be deleted.

**Fix:** Remove the manual enrollment delete. The CASCADE will handle it.

---

### H4. No RLS policy on the `tenants` table

**File:** `internal/database/migrations/000001_initial_schema.up.sql`  

The `tenants` table is the root of the multi-tenant hierarchy. It has no RLS enabled. While only `SYSTEM_ADMIN` roles should be able to create/update tenants, there's no DB-level enforcement preventing a malicious query (e.g., from a compromised read-only connection) from reading all tenant names and Stytch org IDs.

**Fix:** Enable RLS on `tenants` and add a policy scoping to the current tenant's own row (or to SYSTEM_ADMIN).

---

### H5. No validation on payment `amount` in service layer

**File:** `internal/billing/service.go` (RecordPayment method)

The `Amount` field is a string that gets cast to `NUMERIC(12,2)` at the DB layer. Negative values would pass the Go layer but fail at the DB (`CHECK (amount > 0)`), producing a 500 instead of a 400. Similarly, strings like `"0"` or `"-50"` aren't caught early.

**Fix:** Parse `Amount` as `decimal.Decimal` in the handler/service and validate `> 0` before reaching the repository.

---

### H6. No request body size limit on most POST/PUT endpoints

**File:** `internal/students/handler.go` — `bodySizeLimit` exists only on the import route.

All other POST/PUT endpoints (CreateStudent, CreateEnrollment, RecordPayment, etc.) accept unlimited request bodies. A malicious client could send a multi-megabyte JSON payload to cause OOM on the API server.

**Fix:** Set a Fiber global body limit (`fiber.Config.BodyLimit`) or add per-route middleware to key mutating endpoints.

---

### H7. Session token stored in two places with potential inconsistency

**File:** `internal/auth/service.go`  

Sessions are stored in both Postgres and Redis. `GetSession` first checks Redis, then Postgres on miss. If Redis is flushed or a session is deleted from one store but not the other, the user could either be logged out unexpectedly or have a stale session.

The `Logout` function deletes from both stores separately — if one deletion fails, the session remains valid in the other store.

**Fix:** Use Redis as a pure cache with TTL. If Redis is down, fall through to Postgres without erroring. Use a single `Del` Lua script to atomically remove from both stores, or use Redis as the source of truth with Postgres as backup.

---

### H8. Missing `tenant_id` filter on `ListEnrollments` query

**File:** `internal/students/repository.go:326`  

```go
WHERE e.student_id = $1
```

The query filters by `student_id` only, not by `tenant_id`. While `student_id` is a UUID that may be globally unique, this still means any authenticated user can enumerate all enrollments for any student ID if they can guess/brute-force it.

**Fix:** Add `AND e.tenant_id = $2` and thread tenantID through.

---

## 3. 🟡 Medium — Should Fix

### M1. Magic link token appears in URL query params (server logs)

**File:** `internal/auth/handler.go`  

```go
token := c.Query("token")
```

The magic link token is passed as a URL query parameter. Most web servers and load balancers log the full URL (including query parameters), which means the token — which acts as a bearer credential — could appear in plaintext in access logs.

**Fix:** Use `POST` with the token in the request body for verification. If `GET` is required for the redirect flow, at minimum log a warning that tokens may appear in server logs.

---

### M2. Auth endpoints not rate-limited

**File:** `internal/middleware/security.go`  

The sliding-window rate limiter covers all IPs at 60 req/min. The `/api/auth/discover`, `/api/auth/verify`, and `/api/auth/register` endpoints are not rate-limited separately. An attacker can:

- Enumerate valid email addresses via `/discover` (the endpoint returns 200 for both existing and non-existing users)
- Brute-force magic link tokens on `/verify`
- Flood `/register` with registrations

**Fix:** Add stricter per-IP rate limits on auth endpoints (e.g., 10 req/min for discover, 20 req/min for verify).

---

### M3. CSRF token cookie is non-HttpOnly (readable by JS) — design gap

**File:** `internal/auth/handler.go:270-280`  

The CSRF double-submit cookie pattern requires the cookie to be readable by JavaScript so the frontend can include `X-CSRF-Token`. This is by design, but it means any XSS vulnerability in the frontend can read the CSRF token and perform state-changing requests on behalf of the user.

**Fix:** This is a known trade-off of the double-submit pattern. Document this explicitly and invest in XSS prevention (CSP, input sanitization). Consider switching to a SameSite=Strict approach for most endpoints.

---

### M4. `fee_categories` table has no `created_at` / `updated_at` timestamps

**File:** `internal/database/migrations/000001_initial_schema.up.sql`

Every other major entity has `created_at` / `updated_at` with an auto-trigger. `fee_categories` lacks both. This means there's no audit trail for when fee categories were created or modified.

**Fix:** Add columns and trigger in a follow-up migration.

---

### M5. No `tenant_id` on `cbc_learning_areas` table

**File:** `internal/database/migrations/000001_initial_schema.up.sql`

`cbc_learning_areas` has no `tenant_id` column. The table stores curriculum data (grades, learning areas) using embedded JSON seeding. While curriculum data may be shared across tenants, the lack of `tenant_id` means RLS cannot be applied. The table is accessible across all tenants.

**Fix:** If curriculum data is truly global, document this intentionally. If per-tenant customization is expected, add `tenant_id` with RLS.

---

### M6. Inconsistent error response format on some endpoints

**File:** `internal/billing/handler.go` and others  

Some handlers return non-canonical error shapes:

```go
return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{
    "code":    "VALIDATION_ERROR",  // UPPER_SNAKE instead of snake_case
    "message": "invalid request body",
})
```

The canonical contract (from `middleware/errors.go` and `AGENTS.md`) specifies `snake_case` error codes. `VALIDATION_ERROR` breaks the frontend's error handling.

**Fix:** Use `middleware.HTTPError(c, err)` consistently instead of inline error responses.

---

### M7. `PARENT` role users can't access attendance GET endpoints

**File:** `internal/attendance/handler.go`  

All attendance endpoints use `middleware.RequireAuth` which works for any authenticated role. However, the `PARENT` role is missing from the role-priority CASE statements (see H1), which means parent sessions may be loaded with role `"TEACHER"` (the fallback) or empty string. Parents viewing their children's attendance would get incorrect data scoping.

**Fix:** Add `PARENT` to the CASE ordering and test parent access to attendance/payment views.

---

### M8. Seed migration references tables that may not exist yet

**File:** `internal/database/migrations/000002_seed.up.sql`  

Migration 000002 inserts into `assessment_weight_configs`, which is created in migration 000001. This is correct, but the seed also references `cbc_schools` and `tenants` which exist in 000001. The ordering is fine, but the seed table is fragile if migrations are re-ordered.

**Note:** Since `golang-migrate` uses sequential ordering, this is not a bug per se — but documenting the dependency would help future maintainers.

---

## 4. 🔵 Low — Post-MVP

### L1. No db migration tests for the fix migration (000003)

The pre-flight validation queries in migration 000003 are thorough, but there's no automated test that runs them against a known dataset to verify they don't false-positive on valid data.

**Fix:** Add a migration test that seeds sample data, runs the pre-flight checks, and verifies they don't warn for conforming data.

---

### L2. `golang-migrate` URL scheme replacement is fragile

**File:** `internal/database/migrate.go:22-26`  

```go
if strings.HasPrefix(srcURL, "postgres://") {
    srcURL = strings.Replace(srcURL, "postgres://", "pgx5://", 1)
}
```

If the `DATABASE_URL` uses `postgresql://` (a valid PostgreSQL scheme variant), this works. But if the URL contains other `postgres://` substrings (e.g., in a query parameter), the replacement is incorrect. Also, if someone uses `pgx://` or `pgx5://` directly, this code double-wraps it.

**Fix:** Use a proper URL parser to swap the scheme.

---

### L3. Some `PARENT` memberships may be missing invite path

If a parent is added directly to `cbc_student_parents` without going through the inviter flow, no `memberships` row is created for them. The `GetSession` query relies on `memberships` for role resolution. A parent without a membership row would get the default `'TEACHER'` role from the fallback.

**Fix:** Ensure all parent role assignments create a corresponding `memberships` row.

---

### L4. No explicit `context` timeout propagation in many handlers

Several handler-to-service-to-repository chains don't propagate the request context timeout (`c.Context()` is internal to Fiber). If a DB query hangs, the goroutine leaks.

**Fix:** Use `context.WithTimeout` in service-layer calls with appropriate timeouts (e.g., 5s for queries, 30s for imports).

---

## 5. 📋 Summary Table

| ID | Severity | Area | Description | Status |
|----|----------|------|-------------|--------|
| C1 | 🔴 BLOCKER | Security | Default COOKIE_SECRET in production | Fix before launch |
| C2 | 🔴 BLOCKER | Security | Stytch secret committed in `.env` | Fix before launch |
| C3 | 🔴 BLOCKER | Data Integrity | Empty tenant_id in student creates | Fix before launch |
| C4 | 🔴 BLOCKER | Data Integrity | Amounts as strings, no validation | Fix before launch |
| C5 | 🔴 BLOCKER | Security | Dynamic SQL generation pattern | Fix before launch |
| H1 | 🟠 HIGH | Auth/RBAC | PARENT role missing from all CASE statements | Fix before launch |
| H2 | 🟠 HIGH | DB Schema | Undefined function in CHECK constraint | Fix before launch |
| H3 | 🟠 HIGH | Data Integrity | Unscoped enrollment delete without tenant filter | Fix before launch |
| H4 | 🟠 HIGH | Multi-Tenancy | No RLS on tenants table | Fix before launch |
| H5 | 🟠 HIGH | Validation | Payment amount not validated in Go | Fix before launch |
| H6 | 🟠 HIGH | Security | No body size limits on POST/PUT | Fix before launch |
| H7 | 🟠 HIGH | Auth | Dual session store inconsistency risk | Fix before launch |
| H8 | 🟠 HIGH | Data Isolation | ListEnrollments missing tenant_id filter | Fix before launch |
| M1 | 🟡 MEDIUM | Security | Token in URL query params (log leakage) | Should fix |
| M2 | 🟡 MEDIUM | Security | Auth endpoints not rate-limited separately | Should fix |
| M3 | 🟡 MEDIUM | Security | CSRF cookie readable by JS (documented trade-off) | Document |
| M4 | 🟡 MEDIUM | Audit | fee_categories missing timestamps | Should fix |
| M5 | 🟡 MEDIUM | Multi-Tenancy | cbc_learning_areas has no tenant_id | Should fix |
| M6 | 🟡 MEDIUM | API Contract | Inconsistent error code casing | Should fix |
| M7 | 🟡 MEDIUM | Auth/RBAC | Parent access to attendance broken by H1 | Should fix |
| M8 | 🟡 MEDIUM | CI/CD | Seed migration dependency undocumented | Should document |
| L1 | 🔵 LOW | Testing | No migration test for fix migration | Post-MVP |
| L2 | 🔵 LOW | Reliability | Fragile URL scheme replacement | Post-MVP |
| L3 | 🔵 LOW | Auth/RBAC | Parent membership gap via direct add | Post-MVP |
| L4 | 🔵 LOW | Reliability | Missing context timeouts in handler chains | Post-MVP |

---

## Remediation Priority

### Pre-launch (must fix):
1. **C3** — Tenant isolation in student CRUD is completely broken. Fix the empty `""` tenant_id.
2. **C1** — Set `COOKIE_SECRET` in production env, add startup guard.
3. **C2** — Remove `.env` from git, rotate Stytch secret.
4. **H1** — Add `PARENT` to all CASE statements (affects role resolution for ~1M+ users in multi-school deployments).
5. **H2** — Fix the undefined function in the CHECK constraint before it causes runtime errors.
6. **H4** — Enable RLS on tenants table.

### Day 1 (fix within 24h of launch):
7. **C4, H5** — Money-related: validate amounts as `decimal.Decimal`.
8. **H3, H8** — Fix tenant-scoped queries.
9. **H6** — Add body size limits.
10. **M2** — Per-endpoint rate limiting on auth.

### Week 1:
11. Remainder of 🟠 HIGH items.
12. All 🟡 MEDIUM items.
