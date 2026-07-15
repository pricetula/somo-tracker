# Gap Analysis: DB Schema ↔ API Handlers ↔ Frontend Pages

> Generated: 2025-07-16  
> Scope: `backend/internal/*` ↔ `frontend/src/app/(dashboard)/*` ↔ `frontend/src/features/*`

---

## 1. DB Tables With NO Backend API Handler

These tables exist in the migration schema but have **zero** backend Go handler or domain module.

| Table | Schema Location | Notes |
|---|---|---|
| `assessment_weight_configs` | Layer 7 | Domain + handler exist in `assessments/`, weight configs have routes |
| `school_member_counts` | Trigger materialized | No handler needed — auto-synced via triggers |
| `member_active_school` | Layer 10 | Managed inside `auth/handler.go` via `SetActiveSchool` flow |

**All schema tables have a corresponding backend module.** ✅ No orphan tables.

---

## 2. Backend Modules With MISSING Service Methods

### 2.1 Teachers (`teachers/service.go` — 44 lines)
- **Missing:** `GetTeacherByID`, `UpdateTeacher`, `BulkImport` (uses generic imports system instead)
- **No subject/learning-area specialisation methods** — only list, toggle-active, delete

### 2.2 Members (`members/service.go` — 63 lines)
- **Missing:** `GetMemberByID`, `UpdateMember`, `BulkInvite` (delegated to `invitations/`)
- Toggle-active and delete only — minimal CRUD coverage

### 2.3 CbcStreams (`cbcstreams/service.go` — 86 lines)
- **Missing:** No `GetStreamByID` (only List, Create, Update, Delete by separate routes)
- No `Update` route actually exists in handler (only has `Put("/:id")` so it's fine)

### 2.4 CbcTimetableSlots (`cbctimetableslots/service.go` — 161 lines)
- **OK** — Full CRUD + batch + enriched list ✅

### 2.5 TimetableStructure (`timetablestructure/service.go` — 393 lines)
- **Well covered** — List, Create, Batch, Replicate, Delete, Update, DeleteByDay, DeleteByName ✅

### 2.6 Health (`health/service.go` — 204 lines)
- **Missing:** No `ListIncidentsByStudent` — the handler exposes it at route level but service may lack aggregation

### 2.7 Billing (`billing/service.go` — 416 lines)
- **Missing:** No invoice item CRUD (items are created as part of invoice generation only)
- **Missing:** No payment reconciliation or M-Pesa integration
- **Missing:** No `GetInvoiceByID` — only list and generate

### 2.8 Attendance (`attendance/service.go` — 284 lines)
- **Missing:** No `GetAttendanceRecordByID` for single-record lookup

### 2.9 Behavior (`behavior/service.go` — 101 lines)
- **Missing:** No `GetNoteByID` individual fetch (route exists)
- **Missing:** No `UpdateNote` for editing descriptions

---

## 3. API Routes With NO Frontend Page

The following backend API routes have **no corresponding frontend page** that connects to them:

| Backend Route (Handler) | Module | Missing Frontend Feature |
|---|---|---|
| `PATCH /academic-years/:id` | academicyears | No page to edit academic years |
| `POST /academic-years/:id/set-current` | academicyears | No "set current year" button |
| `DELETE /academic-years/:id` | academicyears | No delete year action |
| `POST /academic-terms/` | academicyears | No "create term" form |
| `PATCH /academic-terms/:id` | academicyears | No term edit page |
| `GET/POST/PUT/DELETE /api/v1/curriculum/learning-areas` | curriculum | Only part of tree view — **no standalone LA management page** |
| `GET /curriculum/learning-areas/:id/tree` | curriculum | Partially used in `curriculum/[id]` page |
| `POST /curriculum/weight-configs` | assessments | No weight-config UI at all |
| `PUT /schools/:id` | cbcschools | School edit page (inline in switcher only) |
| `DELETE /schools/:id` | cbcschools | No delete school action |
| `POST /schools/:id/activate` | cbcschools | Used in school switcher |
| `GET /api/v1/imports/active` | imports | No "view active imports" page |
| `POST /api/v1/imports/:job_id/cancel` | imports | No "cancel import" button |
| `POST /students/:id/enrollments` | students | Enrollment create — **partially in enroll-dialog** |
| `DELETE /api/v1/class-teachers/:id` | classteachers | Used in assign-teacher-dialog |
| `DELETE /api/v1/schools/:school_id/imports/active` | imports | No active import management |
| `POST /assessments/sessions/:id/scores` | assessments | Score bulk-upload — **grading page exists** |
| `POST /assessments/sessions/:id/grades` | assessments | Rubric grade upload — **grading page exists** |
| `POST /assessments/sessions/:id/submit` | assessments | Submit for approval — **approval-actions exists** |
| `POST /assessments/sessions/:id/approve` | assessments | Approval action — **approval-actions exists** |

---

## 4. Missing Frontend Pages (No App Route)

| Missing Page | Priority | DB Tables / Backend Routes | Rationale |
|---|---|---|---|
| **Academic Year Management** (`/academic-years`) | HIGH | `academic_years`, `academic_terms` + full CRUD routes | Currently no UI to create/edit years or terms |
| **Timetable Blueprint Page** (`/timetable/structure`) | HIGH | `timetable_structures` | Structure exists in sidebar as "Time table" but links to a page; current page may not show full blueprint |
| **Grading Scale Profiles** (`/settings/grading-scales`) | MEDIUM | `grading_scale_profiles`, `grading_scale_ranges` | Scale profiles are managed via modals only; no dedicated settings page |
| **Assessment Weight Configs** (`/settings/assessment-weights`) | MEDIUM | `assessment_weight_configs` | Backend routes exist, no frontend at all |
| **School Settings / Edit School** (`/settings/school`) | MEDIUM | `cbc_schools` — PUT route | School edit is inline only |
| **Fee Category Management** (`/finance/fee-categories`) | MEDIUM | `fee_categories` | Categories exist in DB, no dedicated page |
| **Fee Template Management** (`/finance/fee-templates`) | MEDIUM | `fee_templates` | Templates exist in DB, no dedicated page |
| **Invoice Detail Page** (`/finance/invoices/[id]`) | MEDIUM | `invoices`, `invoice_items`, `payments` | Backend has `GetInvoiceDetail` — no frontend page |
| **Payment Recording** (`/finance/payments/new`) | MEDIUM | `payments` | Backend has `RecordPayment` — no dedicated form page |
| **Stream Management** (`/settings/streams`) | LOW | `cbc_streams` | Streams managed via settings-school/streams-section but no dedicated page |
| **Behavior Categories Management** (`/settings/behavior-categories`) | LOW | `behavior_categories` | Exists as `/settings/behavior-categories` — actually present ✅ |
| **Import Job Detail / Progress** (`/imports/[job_id]`) | LOW | `import_jobs`, `import_job_failures` | Progress shown in modal only |

---

## 5. Missing Action Buttons / UI Controls

These are **backend capabilities with no corresponding UI control** on existing pages:

| Missing Button/Control | Expected Location | Backend Route |
|---|---|---|
| **"Edit Academic Year"** | Settings or dashboard | `PATCH /academic-years/:id` |
| **"Set Current Year"** | Academic year list | `POST /academic-years/:id/set-current` |
| **"Delete Academic Year"** | Academic year list | `DELETE /academic-years/:id` |
| **"Create Term"** | Academic year detail | `POST /academic-terms/` |
| **"Edit Term"** | Term list | `PATCH /academic-terms/:id` |
| **"Cancel Import"** | Import progress modal | `POST /imports/:job_id/cancel` |
| **"View Import Failures"** | Import progress modal | `GET /imports/:job_id/failures` |
| **"Manage Grading Scales"** | Settings page | `POST/GET/PUT/DELETE /grading-scale-profiles` |
| **"Set Scale Ranges"** | Scale profile detail | `POST /profiles/:id/ranges` |
| **"Edit School Details"** | School switcher / settings | `PUT /schools/:id` |
| **"Delete School"** | School list | `DELETE /schools/:id` |
| **"Edit Stream"** | Streams list | `PUT /streams/:id` |
| **"Delete Stream"** | Streams list | `DELETE /streams/:id` |
| **"View Assessment Weight Configs"** | Curriculum / Settings | `GET /weight-configs` |
| **"Edit Teacher" (TSC number, role)** | Teacher list page | No dedicated route — missing service method |
| **"Add Payment"** | Invoice detail | `POST /payments` |
| **"Waive Invoice"** | Invoice detail | `POST /invoices/:id/waive` |
| **"Generate Invoice"** | Finance dashboard | `POST /invoices/generate` |

---

## 6. Missing Frontend Feature Directories

Completely absent feature directories that don't exist under `src/features/`:

| Missing Feature Dir | What It Should Cover |
|---|---|
| `academic-years` | Year/term CRUD pages |
| `billing` / `fee-categories` | Fee categories, templates |
| `finance-invoices` | Invoice list, detail, payment recording |
| `grading-scales` | Scale profiles, ranges |
| `assessment-weights` | KNEC weight configs UI |
| `import-jobs` | Import job progress, failure viewer, cancellation |
| `school-settings` | School edit, stream management settings |

---

## 7. Service / Domain Gaps (Logical Missing Features)

### 7.1 No Academic Year & Term Management UI
- DB has full `academic_years` + `academic_terms` with versioning, FK guards
- Backend has full CRUD via `academicyears/handler.go`
- Frontend has **zero pages** for managing these — combinobox only

### 7.2 No Fee Category / Template / Invoice Management Pages
- DB has `fee_categories`, `fee_templates`, `invoices`, `invoice_items`, `payments`
- Backend has full billing CRUD via `billing/handler.go`
- Frontend finance page exists but **no sub-pages** for fee management

### 7.3 No Import Job Management UI
- Backend has full import tracking: progress, failures, cancellation
- Frontend handles import initiation but **no dedicated page** for:
  - Viewing active import jobs
  - Monitoring progress (only during initial submission)
  - Viewing/cancelling failed imports
  - Browsing import failure history

### 7.4 No Assessment Weight Configurations UI
- DB has `assessment_weight_configs` linking to KNEC formulas
- Backend has `GET/POST /weight-configs` routes
- Frontend has **zero UI** for this

### 7.5 No Dedicated Grading Scale Management Page
- Grade profiles and ranges managed through modals only
- No full settings page listing all profiles with ranges

### 7.6 No Teacher Profile Editing
- Teacher route includes only List, ToggleActive, Delete
- No way to update TSC number, KNEC panel assessor ID, or teacher role
- These fields exist in the `users` table with specific UNIQUE indexes

---

## 8. Frontend Pages That Exist But Have NO Backend Route Connected

Some frontend pages may exist but lack the corresponding backend API:

| Frontend Page | Backend Route | Status |
|---|---|---|
| `/reports` | `GET /api/v1/reports` | ✅ Exists via `reports.ts` |
| `/reports/terms/[term_id]` | `GET /reports/term/:termId` | ✅ |
| `/reports/bulk-export` | `POST /reports/bulk-export` | Unknown — check reports API |
| `/timetable` | `GET /timetable-structure/...` | ✅ All routes present |
| `/grading` | `GET/POST /assessments/sessions/...` | ✅ |
| `/grading/[id]` | `GET /assessments/sessions/:id` | ✅ |
| `/grading/new` | `POST /assessments/sessions` | ✅ |
| `/settings` | Various | ✅ |
| `/settings/behavior-categories` | `GET/POST/PUT /behavior/categories` | ✅ |

---

## 9. Consolidated Priority Actions

| Priority | Action | Effort | Impact |
|---|---|---|---|
| **P0** | Build Academic Year & Term management UI (list, create, edit, set-current) | Medium | Unblocks all time-based workflows |
| **P0** | Add Import Job tracking page (progress, failures, cancel) | Medium | Critical for bulk ops UX |
| **P1** | Build Invoice detail page with payment recording + waive action | Medium | Core finance workflow |
| **P1** | Add Fee Category & Template management pages | Medium | Prerequisite for invoice generation |
| **P1** | Build Grading Scale Profile settings page | Medium | Assessment setup requires it |
| **P2** | Add Teacher profile editing (TSC, KNEC assessor, role) | Small | Minor admin workflow |
| **P2** | Add Assessment Weight Config UI | Small | Niche but KNEC-mandated |
| **P2** | Connect "Edit School" and "Delete School" actions to existing UI | Small | School lifecycle |
| **P2** | Add "Edit Stream" / "Delete Stream" inline actions | Small | Stream management |
| **P3** | Build Member detail view (currently only list) | Small | Nice-to-have |
| **P3** | Add Behavior Note editing (update description) | Small | Teacher workflow |

---

## 10. Summary Statistics

| Metric | Count |
|---|---|
| DB Tables | ~40 tables + enums |
| Backend Modules (Go packages) | 22 domains + health |
| Backend API Routes | ~120+ routes |
| Frontend App Routes (dashboard) | ~50 page.tsx files |
| Frontend Feature Modules | 22 feature directories |
| Frontend API Client Modules | 24 lib/api/*.ts files |
| **Missing Pages** (estimated) | **10–12** |
| **Missing Buttons/Actions** (estimated) | **20+** |
| **Completely Unconnected Backend Features** | Academic Years/Terms, Fee Management, Import Job Tracking, Weight Configs |
