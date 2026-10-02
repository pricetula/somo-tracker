# Important Features To Build — Somotracker

> Generated from navigation audit, database schema gaps, and existing documentation.
> Last updated: 2026-09-23

---

## Quick Status Matrix

| Feature Area | Backend | Frontend | Nav Link |
|--------------|---------|----------|----------|
| **Dashboard** | ✅ | ✅ | ✅ |
| **Members (Admins/Teachers/Finance/Guardians/Students)** | ✅ | ✅ | ✅ |
| **Curriculum** | ✅ | ✅ | ✅ |
| **Classes** | ✅ | ✅ | ✅ |
| **Timetable** | ⚠️ Partial | ✅ | ✅ |
| **Attendance** | ✅ | ✅ | ✅ |
| **Assessments** | ❌ | ❌ | ✅ |
| **Events** | ⚠️ Partial | ⚠️ Scaffold | ✅ |
| **Reports** | ❌ | ❌ | ✅ |
| **Behavior** | ❌ | ❌ | ✅ |
| **Finance** | ⚠️ Staff only | ❌ | ✅ |
| **Health** | ❌ | ❌ | ✅ |
| **Settings** | ⚠️ Partial | ⚠️ Partial | ✅ |

---

## 1. Assessment System (High Priority)

**Nav promises it, zero implementation exists.**

| Route | Status | Description |
|-------|--------|-------------|
| `/assessments` | ❌ Missing | Assessment sessions list/create |
| `/assessments/grading-scales` | ❌ Missing | Grading scale management |
| `/assessments/weight-configs` | ❌ Missing | Assessment weight configuration |

**Backend needed:**
- Database tables: `assessments`, `assessment_sessions`, `grading_scales`, `weight_configs`, `assessment_results`
- SQLC queries
- API endpoints: CRUD for all above
- Service layer with business logic

---

## 2. Reports / Analytics (High Priority)

**Nav promises it, zero implementation.**

| Route | Status | Description |
|-------|--------|-------------|
| `/reports` | ❌ Missing | Dashboard with academic/attendance/behavior analytics |

**Backend needed:**
- Aggregation queries for attendance rates, grade distributions, behavior trends
- Export endpoints (PDF, CSV)
- Time-series data for charts

---

## 3. Behavior Management (Medium Priority)

**Nav promises it, zero implementation.**

| Route | Status | Description |
|-------|--------|-------------|
| `/behavior` | ❌ Missing | Incident logging, disciplinary actions, parent notifications |

**Backend needed:**
- Tables: `behavior_incidents`, `behavior_categories`, `disciplinary_actions`
- SQLC queries
- API endpoints

---

## 4. Finance Module (Medium Priority)

**Partial backend (staff listing only), no frontend pages.**

| Route | Status | Description |
|-------|--------|-------------|
| `/finance/fee-categories` | ❌ Missing | Fee category CRUD |
| `/finance/fee-templates` | ❌ Missing | Fee template builder |
| `/finance/invoices` | ❌ Missing | Invoice generation, tracking, payments |

**Backend needed:**
- Tables: `fee_categories`, `fee_templates`, `invoices`, `invoice_items`, `payments`
- Integration with student enrollments
- Payment gateway hooks (future)

---

## 5. Health / Medical (Medium Priority)

**Nav promises it, zero implementation. `/nurses` route in nav but no page.**

| Route | Status | Description |
|-------|--------|-------------|
| `/health` | ❌ Missing | Medical records, immunization tracking, nurse visits |
| `/nurses` | ❌ Missing | Nurse staff management |

**Backend needed:**
- Tables: `medical_records`, `immunizations`, `nurse_visits`, `health_conditions`
- Privacy/RLS considerations (sensitive data)

---

## 6. Event Timeline / Calendar (Planned — Design Doc Exists)

**Full design in `docs/event-timeline-ui-plan.md`**

| Component | Status | Description |
|-----------|--------|-------------|
| Annual timeline view | ❌ Missing | Monthly grid + horizontal bar timeline |
| School events CRUD | ❌ Missing | `GET /api/events` exists, no create/update/delete |
| Public holidays CRUD | ❌ Missing | Table exists, no API |
| TimelineBar component | ❌ Missing | Gantt-style bar visualization |

**Backend needed:**
- `POST/PATCH/DELETE /api/events`
- Public holidays CRUD endpoints
- Combined query for timeline view

---

## 7. Missing Core CRUD (from `tables-endpoints-queries.md` gaps)

| Resource | Missing Endpoints | Priority | Blocked Features |
|----------|-------------------|----------|------------------|
| **Rooms** | `GET/POST/PATCH/DELETE /api/school/rooms` | **High** | Timetable room assignment |
| **Timetable Substitutions** | List, Create, Update, Delete | **High** | Teacher coverage workflow |
| **Public Holidays** | Full CRUD | Medium | Calendar integration |
| **Academic Terms** | `GET /api/school/academic-periods` listing | Medium | Term selection in UI |
| **Student Enrollments** | `POST /api/classes/:id/enrollments`, `PATCH /api/enrollments/:id` | **High** | Class management |
| **Guardian-Student Links** | Direct link/unlink API | Medium | Guardian portal |
| **Event Attendance** | Create/List endpoints | Medium | Event check-in |
| **Bulk Job Items** | Inspect per-row import results | Low | Import debugging |

---

## 8. Settings Pages Completion

| Route | Status | Notes |
|-------|--------|-------|
| `/settings` | ✅ Exists | General settings |
| `/settings/streams` | ✅ Exists | Stream management |
| `/settings/grade-levels` | ❌ Missing | Grade level CRUD |
| `/academic-years` | ❌ Missing | Academic year management |

---

## Suggested Implementation Priority

| Phase | Features | Rationale |
|-------|----------|-----------|
| **Phase 1** | Rooms CRUD, Timetable Substitutions, Student Enrollments | Core timetable functionality incomplete without these |
| **Phase 2** | Assessments (full module) | Major pedagogical feature, nav already promises it |
| **Phase 3** | Event Timeline + Public Holidays | Design doc ready, calendar is high-visibility |
| **Phase 4** | Finance (fee categories, templates, invoices) | Revenue-critical for schools |
| **Phase 5** | Reports/Analytics, Behavior, Health | Value-add modules, can build on core data |
| **Phase 6** | Settings completion (grade levels, academic years) | Admin polish |

---

## Related Documents

- `docs/tables-endpoints-queries.md` — Full table/endpoint inventory with gaps
- `docs/event-timeline-ui-plan.md` — Event timeline UI/UX design
- `ATTENDANCE_FEATURE_PLAN.md` — Attendance implementation reference
- `backend-sql-schema.md` — Database schema reference

---

## Notes

- All nav items in `src/components/layout/nav-main.tsx` should eventually have working pages
- Backend follows handler/service separation: thin handlers in `internal/api/`, business logic in `internal/services/`
- Frontend uses feature-module architecture under `src/features/`
- RLS-enabled tables require `database.WithTenantTx` for queries
- Never commit directly — leave changes unstaged for user review