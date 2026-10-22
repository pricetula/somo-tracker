# Feature: Assessment and Report Card Management

**Status:** Draft
**Owner:** Product / Platform
**Created:** 2026-10-15
**Last Updated:** 2026-10-15
**Related Issues:** #
**Related Gaps:** GAP-assessment-and-report-card-management.md, GAP-teacher-profile-enhancement.md, GAP-teacher-class-subject-linking.md, GAP-class-profile-enhancement.md
**Affected Layers:** backend | frontend | docs | reference

## Summary
Enable teachers to create assessments (formative/summative), enter percentage marks tied to frozen grade-scale snapshots, aggregate approved results into term report cards, and publish batch results to students / guardians via the existing bulk-job + SSE infrastructure.

## Problem Statement
- Teachers cannot record a single mark; there is no assessment domain.
- No grade scale, no percentage-to-grade conversion, no term report card.
- Student profile (`features/students/`) has tabs/sections for Stats & KPIs, but no results or report card section.
- Attendance exists (`attendance_service.go`) but has no relationship to performance reporting.
- The product is an administration and timetable platform; the academic core (marks, grades, reports) is missing, blocking adoption for schools that need reporting.

## Solution Overview
Add an `assessment` domain that plugs into the existing `academic_period_resolver` (for term binding), RLS policies (`sqlc` + `queries`), bulk import/job framework (`Asynq` + Redis SSE), and feature-module UI architecture (`features/students/` reference pattern).

### User Stories
- As a **teacher**, I want to create an assessment for my class/subject with a frozen grade-scale snapshot, so marks are consistent and audit-able.
- As a **teacher**, I want to enter percentage marks per student, so the system converts them using the frozen scale.
- As a **teacher/admin**, I want to batch-publish approved results to selected students, with server-side generation of a report card view, so guardians see term progress.
- As a **guardian/student**, I want to see an approved report card (assessment aggregates + attendance rate + current class), so I understand progress.
- As a **school admin**, I want corrections after publish to require a reason + audit trail + re-approval, so data integrity is maintained.

### Acceptance Criteria
- [ ] DB: `grade_scales` (school-wide, frozen snapshot per assessment), `assessments`, `assessment_marks`, `report_cards`, `report_card_items`, `mark_corrections` (audit: old_value, new_value, reason, user_id, timestamp).
- [ ] DB migrations use `.up`/`.down`, include RLS (`app.current_tenant_id`), and have integration tests (`migrator_integration_test.go` pattern).
- [ ] Backend: `AssessmentsService`, `AssessmentHandler`, `ReportCardService`, `ReportCardHandler` through `internal/services/` and `internal/api/`; no direct `pgxpool` in handlers; `WriteError` only.
- [ ] Backend: assessment resolution uses `academic_period_resolver` (current / upcoming / active / break / expired / none) — never client-supplied `term_id`.
- [ ] Backend: write rules: active = current term; break/upcoming = next term; expired/none = rejected with typed error `no_next_academic_term`.
- [ ] Bulk publish: uses existing `Asynq` bulk-job framework + SSE (`/jobs/:id/events`); batch target is `report_cards`.
- [ ] Frontend: `features/assessment/` feature module (`components/ hooks/ services/ types/ index.ts`); page is stacked sections (`h2` headings, `space-y-*`), **no `<Tabs>`** (per `frontend/AGENTS.md` §8).
- [ ] Frontend: mutation errors use `getErrorMessage(err)`; no hardcoded `toast.error("...")`.
- [ ] Frontend: `ApiError` surfaces 400 `errors` map at field level; no generic toast for validation failures.
- [ ] No `ArrowLeft` / back buttons in assessment feature containers.

## Scope

### Backend
- **Service layer**: `services.AssessmentsService` (create, list, get, approve, publish batch), `services.ReportCardService` (aggregate marks + attendance stats per student/term), `services.AssessmentMarksService` (CRUD marks, conversion, audit on correction).
- **Handlers**: `AssessmentsHandler` (`POST /assessments`, `GET /assessments`, `PATCH /assessments/:id`, `POST /assessments/:id/marks`); `ReportCardHandler` (`GET /report-cards`); `AssessmentMarksHandler` (`PATCH /marks/:id`).
- **DB migrations**: `assessments`, `assessment_marks`, `grade_scales`, `report_cards`, `report_card_items`, `mark_corrections` with FK to `academic_terms` and `subjects`.
- **RLS**: all new queries use `sqlc` with tenant isolation; `NewAssessmentsService` takes `*sqlc.Queries`, not `*pgxpool.Pool`.
- **Error contract**: every non-2xx returns `{code, message, errors, request_id}` via `WriteError`; never `c.Status(...).JSON(...)` directly.
- **Period resolution**: `academic_period_resolver` controls which term receives writes; `GET /assessments` default `current` period; optional `period=current` or `period=upcoming` only; any other value = 400.

### Frontend
- **Feature module**: `frontend/src/features/assessment/` (new) with reference to `features/students/components/student-detail.tsx`.
- **Pages / routes**: teacher assessment creation/edit, marks entry (table/virtual list if > 50 students — `tanstack-virtual` per `frontend/AGENTS.md` §7), batch publish flow using `features/import/components/import-orchestrator.tsx` pattern.
- **Report card**: rendered in student detail (`features/students/components/student-detail.tsx`) as an additional stacked section (or dedicated sub-route `/students/[id]/report-card` — no back button, no tab component).
- **No tabs**; use vertical sections (`h2` headings, `space-y-4`). No custom Tailwind colors; shadcn primitives only.
- **Form**: compound `Form` (`FormField`, `FormItem`, `FormLabel`, `FormControl`, `FormMessage`) with `zod` schema + `zodResolver`; pending state on submit; no `useEffect` watching `isSuccess`/`isError`.
- **Navigation**: none inside feature components. Navigation belongs to `layout.tsx` / global shell.

### Public / Marketing
- Not required for this feature.

### Non-Goals
- Live exam-room management / proctoring.
- PDF generation server-side (defer; report card is HTML view in this phase).
- Parent portal (separate domain; guardian linking exists but portal is out of scope).
- Custom grade-scale import from CSV (assume manual entry in first pass).

## Design Notes
- Reuse `features/import/` bulk-upload orchestration for batch-publish.
- Reuse `internal/services/academic_period_resolver.go` for all period-bound writes.
- Reference ADR: `docs/adr/ADR-2026-10-15-stats-architecture.md` for stats aggregation pattern (`stats_kpis`, `summary` tables).
- Assessment creation must snapshot the grade scale (`grade_scales` table with `frozen_at` = `assessment.created_at`) so changes to school-wide scale do not retroactively alter marks.

## Open Gaps / Risks
- `GAP-teacher-class-subject-linking.md` (open) — teachers must have linked subjects/classes before creating assessments; build or link must close first.
- `GAP-assessment-and-report-card-management.md` — this feature closes the gap.
- `GAP-health-orphan-schema.md` — health/nurse module should be decided (build or delete) before adding more domain modules.
- Risk: adding assessment domain increases DB load; bulk publish uses existing `Asynq` job type (`bulk_import`) and should reuse, not duplicate, queue patterns.

## Test Plan
- Unit tests: service-level QUERY verification (`SELECT 1 FROM assessments WHERE ...`); conversion logic (percentage → grade descriptor) tested independently.
- Integration tests: `migrator_integration_test.go` pattern — `TestMigrator_AssessmentDomain` asserts `grade_scales`, `assessments`, `assessment_marks` tables + FK constraints + RLS isolation (`SET LOCAL app.current_tenant_id`).
- E2E / Playwright: (Ignore) teacher creates assessment, adds marks, publishes batch; student profile shows report card section; guardian profile (if portal exists later) sees aggregated stats.
- Contract test: every handler non-2xx returns `{code, message, errors, request_id}`; `getErrorMessage(err)` surfaces to UI; no hardcoded `toast.error`.

## Rollout
- No feature flag required (new domain). Migration sequence: 026 (grade_scales), 027 (assessments + marks), 028 (report_cards + corrections).
- Rollback: down files reverse in order; assessment data loss acceptable only before first term assessment is published (soft-delete preferred).
- Monitoring: same `metrics.go` endpoint labels; add `assessment_marks_total`, `published_report_cards_total` counters.

## References
- `docs/gaps/GAP-assessment-and-report-card-management.md`
- `docs/gaps/GAP-teacher-class-subject-linking.md`
- `docs/adr/ADR-2026-10-15-stats-architecture.md`
- `backend/internal/services/academic_period_resolver.go`
- `backend/internal/api/errors.go`
- `frontend/src/features/students/components/student-detail.tsx` (reference component)
- `frontend/src/lib/api/client.ts`
- `docs/templates/feature-spec.md`
