# Schema → Implementation Gap Report

> **Generated:** 2026-06-30
> **Scope:** Full-stack audit comparing the PostgreSQL schema (`backend/internal/database/migrations/000001_initial_schema.up.sql`) against implemented backend modules and frontend features.
> **Monorepo:** Somotracker — Kenya CBC/CBE academic platform

---

## Executive Summary

The schema defines **41 tables** across 10 logical layers. Of these:

| Layer | Tables | Backend Module | Frontend Feature |
|---|---|---|---|
| Layer 1 — Platform Infrastructure | 3 (tenants, users, sessions) | ✅ auth | ✅ login/register |
| Layer 2 — Core CBC Actors | 10 | ✅ partial (6/10) | ✅ partial (3/10) |
| Layer 3 — Academic Calendar | 2 | ✅ academicyears | ❌ missing |
| Layer 4 — Health & Financials | 7 | ❌ missing entirely | ❌ missing entirely |
| Layer 5 — CBC Curriculum Structure | 4 | ❌ missing entirely | ❌ missing entirely |
| Layer 6 — Teacher Assignments, Attendance, Timetable | 4 | ✅ partial (attendance + timetable) | ❌ missing |
| Layer 7 — CBC Assessment Architecture | 3 | ❌ missing entirely | ❌ missing entirely |
| Layer 8 — CBC Assessment Execution & Results | 3 | ❌ missing entirely | ❌ missing entirely |
| Layer 9 — CBC Aggregation & Reporting | 2 | ❌ missing entirely | ❌ missing entirely |
| Layer 10 — User Active School Context | 1 | ✅ activeschool | ❌ missing |

**Backend modules implemented:** 14 of ~26 needed  
**Frontend features implemented:** 7 of ~20 needed  

---

## Layer-by-Layer Breakdown

### Layer 1 — Platform Infrastructure ✅

| Table | Backend | Frontend | Status |
|---|---|---|---|
| `tenants` | ✅ `internal/tenant/` | ✅ via auth flow | Complete |
| `users` | ✅ `internal/auth/` | ✅ `src/features/auth/` | Complete |
| `sessions` | ✅ `internal/auth/` | ✅ `src/features/auth/` | Complete |

---

### Layer 2 — Core CBC Actors ⚠️ Partial (6 of 10 tables)

| Table | Backend | Frontend | Status |
|---|---|---|---|
| `cbc_schools` | ✅ `internal/cbcschools/` | ✅ types in `generated.ts` | Complete |
| `cbc_streams` | ✅ `internal/cbcstreams/` | ✅ types in `generated.ts` | Complete |
| `cbc_classes` | ✅ `internal/cbcclasses/` | ✅ types in `generated.ts` | Complete — but **no frontend UI** |
| `memberships` | ✅ `internal/members/` | ✅ `src/features/staff/` | Complete |
| `import_jobs` | ✅ `internal/imports/` | ✅ `src/features/staff-import/` | Complete |
| `import_job_failures` | ✅ `internal/imports/` | ✅ via staff-import | Complete |
| `import_job_staging` | ✅ `internal/imports/` | ✅ via student-import | Complete |
| `invitations` | ✅ `internal/invitations/` | ✅ `src/features/staff-import/` | Complete |
| **`cbc_parents`** | ❌ **No module** | ❌ **No feature** | **MISSING** |
| **`cbc_student_parents`** | ❌ **No module** | ❌ **No feature** | **MISSING** |
| **`cbc_student_enrollments`** | ❌ **No module** | ❌ **No feature** | **MISSING** |
| `cbc_students` | ✅ `internal/students/` | ✅ `src/features/students/` | Minimal (list-only, no CRUD) |

**Gaps:**
- **`cbc_parents`** — No dedicated module. Referenced only via `imports/domain.go` for lookup types. No CRUD, no frontend parent management.
- **`cbc_student_parents`** — No module at all. No frontend for managing parent–student relationships.
- **`cbc_student_enrollments`** — No module. Currently only inserted as part of bulk student import. No enrollment management UI, no transfer/suspension workflows.
- **`cbc_classes`** — Backend and types exist, but there is **no frontend UI** for managing classes (create, list, edit, bulk delete).
- **`cbc_students`** — Only a list endpoint exists. No create, update, view detail, or deactivate.

---

### Layer 3 — Academic Calendar ⚠️ Partial

| Table | Backend | Frontend | Status |
|---|---|---|---|
| `academic_years` | ✅ `internal/academicyears/` | ✅ types in `generated.ts` | Complete |
| `academic_terms` | ✅ `internal/academicyears/` | ✅ types in `generated.ts` | Complete |

**Gaps:**
- **No frontend UI** for managing academic years or terms. The backend has full CRUD with optimistic locking, soft-delete, and term validation — but there is no page or dialog to create/edit/list academic years or terms.

---

### Layer 4 — Health & Financials ❌ Entirely Missing

| Table | Backend | Frontend | Status |
|---|---|---|---|
| **`medical_incidents`** | ❌ No module | ❌ No feature | **MISSING** |
| **`student_health_profiles`** | ❌ No module | ❌ No feature | **MISSING** |
| **`fee_categories`** | ❌ No module | ❌ No feature | **MISSING** |
| **`fee_templates`** | ❌ No module | ❌ No feature | **MISSING** |
| **`invoices`** | ❌ No module | ❌ No feature | **MISSING** |
| **`invoice_items`** | ❌ No module | ❌ No feature | **MISSING** |
| **`payments`** | ❌ No module | ❌ No feature | **MISSING** |

**Gaps:**
- All 7 tables in this layer have **zero backend code** and **zero frontend code**.
- The schema includes sophisticated triggers (`fn_sync_invoice_payment_status_insert/delete/update`) and M-Pesa reconciliation indexes — none of this is consumed.
- The `NURSE` and `FINANCE` roles exist in the user_role enum and in memberships, but both are stub dashboards with no actual functionality.

---

### Layer 5 — CBC Curriculum Structure ❌ Entirely Missing

| Table | Backend | Frontend | Status |
|---|---|---|---|
| **`cbc_learning_areas`** | ❌ No module | ❌ No feature | **MISSING** |
| **`cbc_strands`** | ❌ No module | ❌ No feature | **MISSING** |
| **`cbc_sub_strands`** | ❌ No module | ❌ No feature | **MISSING** |
| **`performance_indicators`** | ❌ No module | ❌ No feature | **MISSING** |

**Gaps:**
- This is the entire CBC curriculum hierarchy (Learning Area → Strand → Sub-Strand → Performance Indicator). Without it, no assessment blueprinting, no rubric scoring, and no KNEC SBA uploads are possible.
- The seed file (`cbc_curriculum.json`) exists but nothing consumes it.

---

### Layer 6 — Teacher Assignments, Attendance, Timetable ⚠️ Partial

| Table | Backend | Frontend | Status |
|---|---|---|---|
| `cbc_class_teachers` | ✅ via `internal/timetable/` | ✅ types in `generated.ts` | Backend complete — **no frontend UI** |
| `cbc_attendance_periods` | ✅ `internal/attendance/` | ✅ types in `generated.ts` | Backend complete — **no frontend UI** |
| `cbc_attendance_logs` | ✅ `internal/attendance/` | ✅ types in `generated.ts` | Backend complete — **no frontend UI** |
| `cbc_timetable_slots` | ✅ `internal/timetable/` | ✅ types in `generated.ts` | Backend complete — **no frontend UI** |

**Gaps:**
- Backend is solid for all 4 tables — full CRUD or bulk operations, FK validation, GiST exclusion constraints, auto-registration triggers.
- Frontend has **zero UI** for any of these. No timetable grid, no attendance marking, no teacher assignment UI.

---

### Layer 7 — CBC Assessment Architecture ❌ Entirely Missing

| Table | Backend | Frontend | Status |
|---|---|---|---|
| **`assessment_weight_configs`** | ❌ No module | ❌ No feature | **MISSING** |
| **`assessment_blueprints`** | ❌ No module | ❌ No feature | **MISSING** |
| **`assessment_blueprint_indicators`** | ❌ No module | ❌ No feature | **MISSING** |

**Gaps:**
- The `assessment_weight_configs` table is seeded with official KNEC weights in the seed migration (KPSEA: 60% SBA + 40% written; KJSEA: 20% SBA G7 + 20% SBA G8 + 20% KPSEA result + 60% written). Nothing reads this data.
- No assessment blueprint creation workflow exists.

---

### Layer 8 — CBC Assessment Execution & Results ❌ Entirely Missing

| Table | Backend | Frontend | Status |
|---|---|---|---|
| **`assessment_sessions`** | ❌ No module | ❌ No feature | **MISSING** |
| **`learner_rubric_results`** | ❌ No module | ❌ No feature | **MISSING** |
| **`learner_portfolios`** | ❌ No module | ❌ No feature | **MISSING** |

**Gaps:**
- The core assessment execution layer. No way to administer assessments, record rubric scores (EE/ME/AE/BE), or attach portfolio evidence.
- The schema enforces CBC compliance rules (no raw score averaging, no sub-levels in final KNEC submissions).

---

### Layer 9 — CBC Aggregation & Reporting ❌ Entirely Missing

| Table | Backend | Frontend | Status |
|---|---|---|---|
| **`cbc_term_competency_summaries`** | ❌ No module | ❌ No feature | **MISSING** |
| **`school_member_counts`** | ❌ No module | ❌ No feature | **MISSING** |

**Gaps:**
- `cbc_term_competency_summaries` is the definitive per-term competency record with KNEC sync status — the central artifact for KNEC SBA uploads. Nothing consumes it.
- `school_member_counts` is maintained by triggers but has no read endpoint or frontend display.

---

### Layer 10 — User Active School Context ⚠️ Partial

| Table | Backend | Frontend | Status |
|---|---|---|---|
| `member_active_school` | ✅ `internal/activeschool/` | ✅ types in `generated.ts` | Backend complete — **no frontend UI** |

**Gaps:**
- Backend has full support for active school switching. The `MeResponse` includes `school_id` and `school_name`.
- No frontend school-switcher UI exists. No multi-school navigation.

---

## Summary of What Needs to Be Built

### Priority 1 — Critical (blocks core CBC workflow)

| # | Component | Tables | Est. Effort |
|---|---|---|---|
| 1 | **Curriculum Structure Module** (backend + frontend) | `cbc_learning_areas`, `cbc_strands`, `cbc_sub_strands`, `performance_indicators` | 3-4 weeks |
| 2 | **Assessment Blueprinting Module** (backend + frontend) | `assessment_blueprints`, `assessment_blueprint_indicators`, `assessment_weight_configs` | 2-3 weeks |
| 3 | **Assessment Execution Module** (backend + frontend) | `assessment_sessions`, `learner_rubric_results`, `learner_portfolios` | 3-4 weeks |
| 4 | **Term Competency Summaries Module** (backend + frontend) | `cbc_term_competency_summaries` | 2 weeks |
| 5 | **Student Enrollment Management** (backend + frontend) | `cbc_student_enrollments` | 1-2 weeks |

### Priority 2 — High (required for daily operations)

| # | Component | Tables | Est. Effort |
|---|---|---|---|
| 6 | **Academic Calendar UI** (frontend only) | `academic_years`, `academic_terms` | 1-2 weeks |
| 7 | **Classes Management UI** (frontend only) | `cbc_classes` | 1 week |
| 8 | **Streams Management UI** (frontend only) | `cbc_streams` | 0.5 week |
| 9 | **Timetable UI** (frontend only) | `cbc_timetable_slots` | 2 weeks |
| 10 | **Attendance UI** (frontend only) | `cbc_attendance_periods`, `cbc_attendance_logs` | 2 weeks |
| 11 | **Teacher Assignment UI** (frontend only) | `cbc_class_teachers` | 1 week |

### Priority 3 — Medium (role-specific features)

| # | Component | Tables | Est. Effort |
|---|---|---|---|
| 12 | **Parents Module** (backend + frontend) | `cbc_parents`, `cbc_student_parents` | 1-2 weeks |
| 13 | **Finance Module** (backend + frontend) | `fee_categories`, `fee_templates`, `invoices`, `invoice_items`, `payments` | 3-4 weeks |
| 14 | **Health Module** (backend + frontend) | `medical_incidents`, `student_health_profiles` | 1-2 weeks |
| 15 | **Active School Switcher UI** (frontend only) | `member_active_school` | 0.5 week |

### Priority 4 — Polish & Non-Blocking

| # | Component | Tables | Est. Effort |
|---|---|---|---|
| 16 | School Member Counts Display | `school_member_counts` | 0.5 week |
| 17 | Student Profile (detail/edit) | `cbc_students` | 1 week |
| 18 | KNEC Sync Dashboard | `cbc_term_competency_summaries` | 1-2 weeks |

---

## Backend Module Checklist (by layer)

### Implemented ✅

| Module | Tables | Notes |
|---|---|---|
| `academicyears` | `academic_years`, `academic_terms` | Full CRUD, optimistic locking, soft delete, validation |
| `activeschool` | `member_active_school` | Active school context |
| `attendance` | `cbc_attendance_periods`, `cbc_attendance_logs` | Mark/retrieve attendance |
| `auth` | `users`, `sessions` | Stytch B2B auth |
| `cbcclasses` | `cbc_classes` | Create, list, update, bulk delete |
| `cbcschools` | `cbc_schools` | School CRUD |
| `cbcstreams` | `cbc_streams` | Stream CRUD |
| `imports` | `import_jobs`, `import_job_failures`, `import_job_staging` | Staff + student bulk import with SSE, Asynq workers |
| `invitations` | `invitations` | Invitation CRUD |
| `members` | `memberships` | Member listing by role |
| `students` | `cbc_students` | List only — no create/update/detail |
| `teachers` | extends `members` | Teacher-specific fields (TSC, KNEC) |
| `tenant` | `tenants` | Tenant CRUD |
| `timetable` | `cbc_timetable_slots`, `cbc_class_teachers` | Bulk upsert slots, GiST exclusion, auto-register teachers |

### Not Implemented ❌

| Module Needed | Tables | Dependencies |
|---|---|---|
| `learningareas` | `cbc_learning_areas` | Requires `cbcschools` |
| `curriculum` | `cbc_strands`, `cbc_sub_strands`, `performance_indicators` | Requires `learningareas` |
| `parents` | `cbc_parents`, `cbc_student_parents` | Requires `auth`, `students` |
| `enrollments` | `cbc_student_enrollments` | Requires `students`, `cbcclasses`, `academicyears` |
| `health` | `medical_incidents`, `student_health_profiles` | Requires `students` |
| `finance` | `fee_categories`, `fee_templates`, `invoices`, `invoice_items`, `payments` | Requires `students`, `academicyears`, `parents` |
| `assessment` | `assessment_blueprints`, `assessment_blueprint_indicators`, `assessment_weight_configs` | Requires `learningareas`, `curriculum`, `cbcschools` |
| `assessmentsessions` | `assessment_sessions`, `learner_rubric_results` | Requires `assessment`, `cbcclasses` |
| `portfolios` | `learner_portfolios` | Requires `assessmentsessions` |
| `competencysummaries` | `cbc_term_competency_summaries` | Requires `assessmentsessions` |

---

## Frontend Feature Checklist (by route area)

### Implemented ✅

| Route | Feature | API Client |
|---|---|---|
| `/(auth)/login` | Magic-link login | `src/lib/api/auth.ts` |
| `/(auth)/register` | Registration | `src/lib/api/auth.ts` |
| `/(dashboard)/` | Role-based home | — |
| `/(dashboard)/admins/` | Admin listing | `src/lib/api/admins.ts` |
| `/(dashboard)/teachers/` | Teacher listing | `src/lib/api/teachers.ts` |
| `/(dashboard)/nurses/` | Nurse listing | `src/lib/api/nurses.ts` |
| `/(dashboard)/finance/` | Finance staff listing | `src/lib/api/finance.ts` |
| `/(dashboard)/students/` | Student listing + import | `src/features/students/` + `student-import/` |
| `/(dashboard)/settings/` | Settings page | — |

### Not Implemented ❌

| Route Needed | Feature | Priority |
|---|---|---|
| `/(dashboard)/academic-years/` | Academic year + term management | P2 |
| `/(dashboard)/streams/` | Stream CRUD | P2 |
| `/(dashboard)/classes/` | Class management (create from stream, assign teachers) | P2 |
| `/(dashboard)/timetable/` | Timetable grid (weekly view, bulk upsert) | P2 |
| `/(dashboard)/attendance/` | Attendance marking + history | P2 |
| `/(dashboard)/learning-areas/` | Learning area management | P1 |
| `/(dashboard)/curriculum/` | Strands/sub-strands/PIs management | P1 |
| `/(dashboard)/assessments/` | Blueprinting + rubric scoring | P1 |
| `/(dashboard)/competencies/` | Term competency summaries + KNEC sync | P1 |
| `/(dashboard)/parents/` | Parent management + student linking | P3 |
| `/(dashboard)/invoices/` | Fee templates → invoice generation | P3 |
| `/(dashboard)/payments/` | Payment recording + M-Pesa reconciliation | P3 |
| `/(dashboard)/health/` | Medical incidents + health profiles | P3 |
| `/(dashboard)/students/[id]/` | Student detail/profile view | P4 |

---

## Cross-Cutting Concerns

### 1. Error Handling Compliance
The root `AGENTS.md` error handling contract is clear. The implemented modules appear to follow it, but all new modules must be built with:
- Sentinel errors in every `domain.go`
- `fmt.Errorf("Module.Type.Method: %w", err)` wrapping at every boundary
- Single `HTTPError()` call per handler
- No log-and-return patterns

### 2. Dependency Injection (Backend)
All new modules must follow the `fx.Annotate` rules:
- One constructor per lifecycle
- Multiple `fx.As(new(Interface))` on a single `fx.Annotate` call
- No duplicate constructor registration

### 3. Testing (Backend)
Every new module must ship:
- Unit tests (`*_service_test.go`) with in-memory mocks
- Integration tests (`*_repository_test.go`) against live Postgres

### 4. Frontend Architecture (Frontend)
All new features must follow:
- Self-contained feature modules under `src/features/<feature>/`
- Exports only via `index.ts`
- No `setState` in `useEffect`
- `ApiError` handling, `getErrorMessage`, `onError` on mutations
- Feature help derived from `content/docs/*.mdx` frontmatter

### 5. Database Migration Policy
No new migration files. All schema changes go into `000001_initial_schema.up.sql` using `CREATE TABLE IF NOT EXISTS` and `ALTER TABLE … ADD COLUMN IF NOT EXISTS`.

---

## Appendices

### A. Schema Tables vs. Implementation Matrix

| Table | Backend | Frontend | Priority |
|---|---|---|---|
| `tenants` | ✅ | ✅ | — |
| `users` | ✅ | ✅ | — |
| `sessions` | ✅ | ✅ | — |
| `cbc_schools` | ✅ | ✅ types, ⬜ UI | P4 |
| `cbc_streams` | ✅ | ✅ types, ⬜ UI | P2 |
| `cbc_classes` | ✅ | ✅ types, ⬜ UI | P2 |
| `memberships` | ✅ | ✅ | — |
| `import_jobs` | ✅ | ✅ | — |
| `import_job_failures` | ✅ | ✅ | — |
| `import_job_staging` | ✅ | ✅ | — |
| `invitations` | ✅ | ✅ | — |
| `cbc_parents` | ❌ | ❌ | P3 |
| `cbc_student_parents` | ❌ | ❌ | P3 |
| `cbc_student_enrollments` | ❌ | ❌ | P1 |
| `cbc_students` | ✅ partial | ✅ partial | P4 |
| `academic_years` | ✅ | ✅ types, ⬜ UI | P2 |
| `academic_terms` | ✅ | ✅ types, ⬜ UI | P2 |
| `medical_incidents` | ❌ | ❌ | P3 |
| `student_health_profiles` | ❌ | ❌ | P3 |
| `fee_categories` | ❌ | ❌ | P3 |
| `fee_templates` | ❌ | ❌ | P3 |
| `invoices` | ❌ | ❌ | P3 |
| `invoice_items` | ❌ | ❌ | P3 |
| `payments` | ❌ | ❌ | P3 |
| `cbc_learning_areas` | ❌ | ❌ | P1 |
| `cbc_strands` | ❌ | ❌ | P1 |
| `cbc_sub_strands` | ❌ | ❌ | P1 |
| `performance_indicators` | ❌ | ❌ | P1 |
| `cbc_class_teachers` | ✅ | ✅ types, ⬜ UI | P2 |
| `cbc_attendance_periods` | ✅ | ✅ types, ⬜ UI | P2 |
| `cbc_attendance_logs` | ✅ | ✅ types, ⬜ UI | P2 |
| `cbc_timetable_slots` | ✅ | ✅ types, ⬜ UI | P2 |
| `assessment_weight_configs` | ❌ | ❌ | P1 |
| `assessment_blueprints` | ❌ | ❌ | P1 |
| `assessment_blueprint_indicators` | ❌ | ❌ | P1 |
| `assessment_sessions` | ❌ | ❌ | P1 |
| `learner_rubric_results` | ❌ | ❌ | P1 |
| `learner_portfolios` | ❌ | ❌ | P1 |
| `cbc_term_competency_summaries` | ❌ | ❌ | P1 |
| `school_member_counts` | ❌ | ❌ | P4 |
| `member_active_school` | ✅ | ✅ types, ⬜ UI | P3 |

### B. Existing API Client Coverage

API clients exist in `src/lib/api/` for:
- `auth.ts` — discovery, verify, register, me, logout
- `admins.ts` — list, toggle active
- `teachers.ts` — list, toggle active
- `nurses.ts` — list, toggle active
- `finance.ts` — list, toggle active
- `members.ts` — list by role
- `invitations.ts` — list by role
- `imports.ts` — staff import (start, track, SSE, failures)

API clients **missing** for:
- Academic years / terms
- Schools (update, details)
- Streams
- Classes
- Timetable
- Attendance
- Learning areas / curriculum
- Assessments
- Parents
- Enrollments
- Fee categories / templates
- Invoices / payments
- Health / medical
- Competency summaries
- Active school switch

### C. Total Effort Estimate

| Priority | Items | Estimated Effort |
|---|---|---|
| P1 — Critical | 5 items | 11-15 weeks |
| P2 — High | 6 items | 7.5-12 weeks |
| P3 — Medium | 4 items | 6-9 weeks |
| P4 — Polish | 3 items | 2.5-3.5 weeks |
| **Total** | **18 items** | **~27-40 weeks** |
