# Full-Stack Gap Analysis — SomoTracker

## How to read this

Each gap is rated by severity and links the SQL schema table(s), backend module, and frontend page/API that are involved. The goal is to surface every table→service→endpoint→page mismatch so work can be prioritised.

---

## CRITICAL GAPS

### 1. Reports Module — Backend Completely Missing

| Layer | Status | Details |
|-------|--------|---------|
| SQL Schema | ❌ No dedicated `term_reports` table | The report-card endpoint in assessments (`GetStudentTermGrades` at `GET /api/v1/parent/students/:studentId/report-card`) partially overlaps but uses a different path and is parent-only. |
| Backend | ❌ **No `backend/internal/reports/` module** | No handler, service, or repository exists. |
| Frontend API (`reports.ts`) | 🟡 Calls 5 routes that **will all 404** | `GET /api/v1/reports/terms/:term_id/students/:student_id`, `POST /api/v1/reports/terms/:term_id/generate`, `GET /api/v1/reports/terms/:term_id`, `POST /api/v1/reports/:report_id/publish` |
| Frontend pages | 🟡 Pages exist but broken | `/reports`, `/reports/student/[id]`, `/reports/terms/[term_id]`, `/reports/bulk-export` all render but can't fetch data. |

**What's needed:** A new `backend/internal/reports/` module (handler + service + repository) with a `term_reports` table in the schema, or polyfill the existing assessments report-card endpoint with the right contract.

---

### 2. Student DELETE Route Not Registered

| Layer | Status | Details |
|-------|--------|---------|
| Backend handler | ✅ `Delete()` function exists at `students/handler.go:504` | But **never registered in `RegisterRoutes()`**. |
| Backend routes | ❌ No `DELETE /api/v1/students/:id` | The route block in `RegisterRoutes` stops at `POST /check-duplicates`. |
| Frontend API (`students.ts:204`) | 🟡 Calls `api.delete<void>(\`/api/v1/students/${id}\`)` | Will receive a 404. |

**Fix:** Add one line:
```go
students.Delete("/:id", middleware.RequireAuth, h.Delete)
```

---

## HIGH-SEVERITY GAPS

### 3. Class Update — Missing Role Restriction (Security)

| Layer | Status | Details |
|-------|--------|---------|
| Backend route | ⚠️ `PUT /api/v1/classes/:id` has **only `RequireAuth`** | No `RequireRole("SCHOOL_ADMIN")` unlike all other mutating class endpoints (`POST`, `DELETE`). Any authenticated user (teacher, nurse, parent) could update a class. |
| Frontend | 🟡 No edit-page calls this route | But the security hole exists regardless. |

**Fix:** Add `middleware.RequireRole("SCHOOL_ADMIN", "SYSTEM_ADMIN")` to the PUT route.

---

## MODERATE-SEVERITY GAPS

### 4. RLS: Three Curriculum Tables Missing Tenant Isolation

| Layer | Status | Details |
|-------|--------|---------|
| SQL Schema | ⚠️ `cbc_strands`, `cbc_sub_strands`, `performance_indicators` | These three tables appear in `CREATE TABLE` statements and the ALTER/trigger sections, but are **not included** in either the `ENABLE ROW LEVEL SECURITY` block or the policy-creation DO block. |
| Backend | 🟡 The curriculum service scopes queries by `learning_area_id` (which is tenant-scoped) | So the app layer is safe today, but RLS wouldn't catch a missed WHERE clause. |

**Fix:** Add these three tables to both the ENABLE RLS statement list and the policy DO loop.

### 5. No Stream Detail Page (Frontend UX Gap)

| Layer | Status | Details |
|-------|--------|---------|
| Backend | ✅ `GET /api/v1/streams/:id` exists | Handler returns a single stream by ID. |
| Frontend API (`streams.ts`) | ❌ No `getStream(id)` function | Never calls `GET /api/v1/streams/:id`. |
| Frontend pages | ❌ No individual stream detail page | Only `streams/add` and the list page exist. |

**Fix:** Add `getStream` API function and optionally a `streams/[id]` page.

### 6. Class Edit — No Frontend Page (UX Gap)

| Layer | Status | Details |
|-------|--------|---------|
| Backend | ✅ `PUT /api/v1/classes/:id` exists | Handler has `Update` function. |
| Frontend pages | ❌ No `classes/[id]/edit` page | Only `classes/add`, `classes/[id]` (detail), and `classes/[id]/enroll` exist. |

**Fix:** Build a `classes/[id]/edit` page or combine edit into the detail page.

---

## MINOR GAPS / OBSERVATIONS

### 7. Invitations — `PUT /:id` Not Exposed

| Layer | Status | Details |
|-------|--------|---------|
| Backend | ❌ No `PUT /api/v1/invitations/:id` | Invitations are immutable after creation (status only transitions via accept/expire/revoke flows). Intentional, but worth noting if manual status-override is ever needed. |
| Frontend | 🟡 Invitations pages exist but are read-only | `/invitations` pages list invitations but offer no edit capability. |

### 8. Academic Terms — DELETE Commented Out

| Layer | Status | Details |
|-------|--------|---------|
| Backend route | 🟡 Commented out in `academicyears/handler.go:40` | `// terms.Delete("/:id", ...)` — likely intentional to prevent orphaned enrollments. |

### 9. `school_member_counts` Table — No Backend Write API

| Layer | Status | Details |
|-------|--------|---------|
| SQL Schema | ✅ Table exists with triggers auto-syncing from `memberships` and `cbc_students` | Fully maintained by DB triggers. |
| Backend | ❌ No dedicated read endpoint | Not critical since it's a materialised cache, but any UI wanting to show "XX Teachers, YY Students" must either query members directly or someone needs to expose `GET /api/v1/schools/:id/counts`. |

### 10. `assessment_weight_configs` — No Frontend UI

| Layer | Status | Details |
|-------|--------|---------|
| SQL Schema | ✅ Table exists | KNEC weight configs for KPSEA/KJSEA/KSSEA. |
| Backend | ✅ CRUD routes exist in assessments handler | `GET`, `GET /:id`, `POST /` |
| Frontend API | ✅ `assessments.ts` has `getWeightConfigs()` etc. | |
| Frontend pages | ❌ No UI page to manage weight configs | Only backend/system-level, so this may be intentional (SYSTEM_ADMIN only). |

---

## SUMMARY TABLE

| # | Gap | Severity | SQL Schema | Backend | Frontend API | Frontend Pages |
|---|-----|----------|-----------|---------|-------------|---------------|
| 1 | **Reports module missing** | 🔴 Critical | ❌ | ❌ | 🟡 (will 404) | 🟡 (renders broken) |
| 2 | **Student DELETE not registered** | 🔴 Critical | N/A | ✅ code, ❌ route | 🟡 (will 404) | N/A |
| 3 | **Class PUT missing role guard** | 🟠 High | N/A | ⚠️ | N/A | N/A |
| 4 | **RLS missing 3 curriculum tables** | 🟡 Medium | ⚠️ | N/A | N/A | N/A |
| 5 | **No stream detail page** | 🟡 Medium | N/A | ✅ | ❌ | ❌ |
| 6 | **No class edit page** | 🟡 Medium | N/A | ✅ | N/A | ❌ |
| 7 | **No invitation PUT** | ⚪ Minor | N/A | ❌ (intentional) | N/A | N/A |
| 8 | **Term DELETE commented** | ⚪ Minor | N/A | 🟡 | N/A | N/A |
| 9 | **Member counts not readable via API** | ⚪ Minor | ✅ (triggers) | ❌ | N/A | N/A |
| 10 | **Weight configs no UI** | ⚪ Minor | ✅ | ✅ | ✅ | ❌ (intentional) |

---

## ROUTE INVENTORY CROSS-REFERENCE

For reference, here is every backend → frontend route pair verified:

| Prefix | Backend Routes | Frontend API File | Status |
|--------|---------------|-------------------|--------|
| `/api/v1/academic-years` | LIST, CREATE, PATCH, SET-CURRENT, DELETE | `academic-terms.ts` | ✅ |
| `/api/v1/academic-terms` | LIST, CREATE, PATCH | `academic-terms.ts` | ✅ |
| `/api/v1/assessments` | Full CRUD + submit/approve/reject + scores/grades | `assessments.ts` | ✅ |
| `/api/v1/attendance` | Roster, bulk-mark, sessions, dashboard, history, summaries | `attendance.ts` | ✅ |
| `/api/auth` | Discover, verify, register, callback, me, logout | `auth.ts` | ✅ |
| `/api/v1/behavior` | Categories CRUD + Notes CRUD + review queue | `behavior.ts` | ✅ |
| `/api/v1/billing` | Fee categories, templates, invoices, payments | `billing.ts` | ✅ |
| `/api/v1/classes` | LIST, GET, CREATE, UPDATE, DELETE, roster, enroll, unenroll | `classes.ts` | ✅ |
| `/api/v1/class-teachers` | CREATE, LIST-BY-CLASS, LIST-BY-TEACHER, GET, DELETE | `classteachers.ts` | ✅ |
| `/api/v1/curriculum` | Learning areas, strands, sub-strands, indicators (full CRUD) | `curriculum.ts` | ✅ |
| `/api/v1/health` | Incidents CRUD + profiles + student health | `health.ts` | ✅ |
| `/api/v1/imports` | Active job, job detail, failures, cancel, list | `imports.ts` | ✅ |
| `/api/v1/invitations` | LIST + staff/invite | `invitations.ts` | ✅ |
| `/api/v1/members` | LIST, GET, UPDATE, TOGGLE-ACTIVE, DELETE | `admins.ts`, `finance.ts`, `nurses.ts`, `members.ts` | ✅ |
| `/api/v1/parents` | CRUD + invite + import + link/unlink students | `parents.ts` | ✅ |
| `/api/v1/students` | LIST, CREATE, GET, UPDATE, enrollments, import, check-duplicates | `students.ts` | ⚠️ DELETE missing |
| `/api/v1/teachers` | LIST, GET, UPDATE, TOGGLE-ACTIVE, DELETE | `teachers.ts` | ✅ |
| `/api/v1/timetable/structure` | Full CRUD + batch + replicate + delete-by-day/name | `timetable-structure.ts` | ✅ |
| `/api/v1/timetable/slots` | LIST, ENRICHED, GET, CREATE, BATCH, UPDATE, DELETE | `timetable-structure.ts` | ✅ |
| `/api/v1/reports` | **NONE** — all 5 frontend routes will 404 | `reports.ts` | 🔴 Critical |
| `/api/v1/streams` | LIST, GET, CREATE, UPDATE, DELETE | `streams.ts` | ⚠️ GET /:id not called |
| `/api/v1/schools` | CREATE, LIST, UPDATE, DELETE, ACTIVATE | `schools.ts` | ✅ |
