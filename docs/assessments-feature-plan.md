# Assessment Features — Plan (2026-09)

Status: DRAFT PLAN — not committed per AGENTS.md golden rule.
Layer: backend + frontend. Isolation rule applies.

---

## 1. What "assessments" means in this project

From `docs/cbc/*.json`, `backend/internal/database/migrations/` (000036–000061), and `docs/schema-details.md`:

- **Assessment sessions** (`assessment_sessions`) — teacher-created evaluations for a class + learning_area + academic_term. Lifecycle: `DRAFT → PENDING_APPROVAL → PUBLISHED`. Rejection returns to `DRAFT` with `rejection_comment`. Two `evaluation_method`s: `QUANTITATIVE` (marks → percentage → grading scale) and `RUBRIC` (indicator-level `cbc_performance_level` directly, no raw score).
- **Student scores** (`student_assessment_scores`) — quantitative only. `raw_score` (≤ `max_points`, enforced by `chk_score_range` + `max_points_check`). `calculated_percentage`, `final_performance_level` (snapshotted at admin approval — immutable after that). `enrollment_status` denormalized to enforce No-Grade-Ghosting (`ACTIVE` required; `ABSENT`/`EXEMPT` blocked).
- **Rubric outcome grades** (`student_assessment_outcome_grades`) — rubric only. Maps `student_id + session_id` → `performance_indicator_id` → `awarded_level` (`EE`/`ME`/`AE`/`BE`). Unique on `(session_id, student_id, performance_indicator_id)`.
- **Weight configs** (`assessment_weight_configs`) — KNEC-mandated per grade / assessment_type_code / target_exam / effective_from (`KPSEA`, `KJSEA`, `KSSEA`). Not per-school.
- **Publication triggers** (`fn_assessment_sessions_after_publish`, 000061) — after `PUBLISHED` refreshes `student_term_subject_summaries` (quantitative) and `student_subject_strand_summaries` (rubric).
- **Summary rollups** (schema-details.md §Assessment summary rollups) — term-level, cohort positions, performance projections, behavior term summaries, teacher workload/delivery.

Schema is complete; **backend module and all frontend surfaces are missing**.

---

## 2. Current gaps (verified 2026-09)

| Area | Found | Missing |
|---|---|---|
| DB schema / migrations | Yes (000036–000061) + trigger functions | — |
| Domain types / DTOs | Partially (`students/domain.go` has `KNECAssessmentNumber`; academia references in `academicyears/`) | `assessment_sessions`, `student_assessment_scores`, `outcome_grades`, weight configs, session status enums |
| Backend module (service/repo/handler/router) | **None** — no `backend/internal/assessments/` directory | All CRUD, approval workflow, score entry, rubric entry, publish, refresh, weight config read |
| API endpoints (`/api/v1/assessments/*`) | None | All |
| Frontend feature directory | None | Assessment creation, grading sheet (quant + rubric), approval queue, parent result view, cohort/summary views |
| Parent portal / result visibility | No `assessment` references in `frontend/src/features/` | Published session results, term summaries |

---

## 3. Feature plan — phases

Phase order follows dependency: schema → session lifecycle → scoring → approval/publication → reporting → integration. Each phase keeps the canonical error contract (`code`/`message`/`errors`) and avoids empty-catch / log-and-return / silent `_` per root AGENTS.md.

### Phase 1 — Core session lifecycle (backend + minimal frontend)

1.1 **Backend: `internal/assessments/` module**
- `domain.go`: `AssessmentSession`, `AssessmentSessionStatus`, `EvaluationMethod`, `CreateSessionPayload`, `UpdateSessionPayload`, `SubmitForApprovalPayload`, `ApproveSessionPayload`, `RejectSessionPayload`.
- `repository.go`: queries with `tenant_id` + `school_id` + RLS awareness; enforce `fk_` constraints from migrations.
- `service.go`: lifecycle logic (draft → submit → approve/reject → publish). Block `max_points` change after any `student_assessment_scores` exist via function-catch (`P0002`). Enforce `chk_quantitative_has_points` / `chk_rubric_no_points` at service layer for fast fail.
- `handler.go`: REST routes; return canonical error JSON for all non-2xx.
- `router.go`: wire to `backend/api` or existing router group.

1.2 **API endpoints (backend)**
- `POST /api/v1/assessments/sessions` — create session (teacher, `DRAFT` default).
- `GET /api/v1/assessments/sessions` — list with filters: `school_id`, `class_id`, `learning_area_id`, `academic_term_id`, `status`, `evaluation_method`.
- `GET /api/v1/assessments/sessions/:id` — get with embedded session metadata.
- `PUT /api/v1/assessments/sessions/:id` — update (only allowed when `DRAFT`; block edits after submit unless rejected back to `DRAFT`). Block `max_points` mutation when scores exist (catch `P0002`).
- `POST /api/v1/assessments/sessions/:id/submit` — `DRAFT → PENDING_APPROVAL`; clear any old `rejection_comment`.
- `POST /api/v1/assessments/sessions/:id/approve` — admin only; `PENDING_APPROVAL → PUBLISHED`; trigger `fn_assessment_sessions_after_publish` via DB (already exists — do not duplicate in code, but verify it fires).
- `POST /api/v1/assessments/sessions/:id/reject` — admin only; `PENDING_APPROVAL → DRAFT`; require `rejection_comment`; clear `submitted_by`/`approved_by` appropriately.
- `DELETE /api/v1/assessments/sessions/:id` — allow only when `DRAFT` and no scores exist.

1.3 **Frontend (pure-shadcn, per `pure-shadcn` skill)**
- `frontend/src/features/assessments/components/session-form.tsx` — create/edit session form (name, learning_area, evaluation_method toggle, max_points conditional, scheduled_date).
- `frontend/src/features/assessments/components/session-list.tsx` — DataTable of sessions per class/term; filter by status; actions (edit, submit, approve/reject for admin, delete).
- `frontend/src/features/assessments/pages/sessions/page.tsx` — listing + FAB for create.
- `frontend/src/app/assessments/sessions/[id]/page.tsx` — detail with approval actions.

---

### Phase 2 — Quantitative scoring (backend + frontend sheet)

2.1 **Backend**
- `POST /api/v1/assessments/sessions/:id/scores` — batch or single score entry (teacher). Validate `raw_score` against session `max_points` (use `max_points_check` logic or replicate in Go). Enforce `enrollment_status = ACTIVE`; reject `ABSENT`/`EXEMPT` (No-Grade-Ghosting). Only allowed when `DRAFT` or `PENDING_APPROVAL` (before publish — scores can be edited pre-approval; after approval, scores are read-only except if session rejected back to draft).
- `PUT /api/v1/assessments/scores/:score_id` — update single score (same rules).
- `GET /api/v1/assessments/sessions/:id/scores` — list with student info; include `calculated_percentage` computed from `grading_scale_profiles` via DB logic or application computation.
- `GET /api/v1/assessments/sessions/:id/students` — eligible students (enrolled, not absent/exempt) for grading sheet.

2.2 **Frontend**
- `frontend/src/features/assessments/components/quantitative-sheet.tsx` — grid/table: students × session; input `raw_score`; live `calculated_percentage`; disable input when session `PUBLISHED`. Show `final_performance_level` only after approval.
- Validation: non-negative, ≤ max_points, numeric.

---

### Phase 3 — Rubric grading (backend + frontend indicator sheet)

3.1 **Backend**
- `POST /api/v1/assessments/sessions/:id/outcome-grades` — batch/single. Validate `performance_indicator_id` belongs to session's `learning_area`. Validate `awarded_level` is valid `cbc_performance_level`. Enforce same enrollment / lifecycle rules as quantitative.
- `GET /api/v1/assessments/sessions/:id/outcome-grades` — with indicator names.
- `PUT /api/v1/assessments/outcome-grades/:id` — update.

3.2 **Frontend**
- `frontend/src/features/assessments/components/rubric-sheet.tsx` — student rows × performance indicators; dropdown per cell (`EE`/`ME`/`AE`/`BE`). Disable after `PUBLISHED`. Show `final_performance_level` derived from indicator grades only post-approval.

---

### Phase 4 — Weight configs & term-level rollups (read-only / admin)

4.1 **Backend**
- `GET /api/v1/assessments/weight-configs` — read-only; filter by `grade_level`, `target_exam`, `effective_from`. No mutation (national mandate).
- `GET /api/v1/assessments/term-summaries?term_id=` — read from `student_term_subject_summaries` (already computed by DB triggers).
- `GET /api/v1/assessments/cohort-positions?term_id=` — read from `student_cohort_position_summaries`.
- `GET /api/v1/assessments/performance-projections?term_id=` — read from `student_performance_projections`.

4.2 **Frontend**
- Admin/config page for viewing weights (not editing — aligned with `assessment_weight_configs` comment that changes require schema change for per-school overrides).
- Teacher/parent summary views using existing DB rollup tables.

---

### Phase 5 — Parent / student result visibility (integration)

5.1 **Requirements from schema + migrations**
- Only `PUBLISHED` sessions visible to parents.
- `student_term_overall_summaries` must blend quantitative percentages + rubric grades mapped via `grading_scale_ranges.default_percentage_mapping`.
- Cohort positions (percentile, rank, average) computed via `fn_compute_cohort_positions_for_term` — never incremental; call on-demand or after publish.

5.2 **Backend**
- `GET /api/v1/parents/students/:student_id/term-summaries?academic_term_id=` — for linked parent, with RLS check.
- `GET /api/v1/students/:student_id/assessment-results?academic_term_id=` — own results for student view.

5.3 **Frontend**
- `frontend/src/features/assessments/pages/student-results/page.tsx` — term-by-term table with subject, percentage/level, cohort rank.
- Parent-facing result card showing published session outcomes.

---

## 4. Constraints that must survive every phase

From root AGENTS.md + `internal/middleware/errors.go` + migrations:

- **Canonical error JSON** on every non-2xx: `{"code":"...","message":"...","errors":{...}}`. No custom shapes.
- **No empty catch** — every error wrapped or logged at handler, returned up stack from intermediate layers. Intermediate layers wrap and return; handler logs once.
- **No silent `_`** — `max_points_check` result, trigger return values, and DB function outputs must be handled.
- **No log-and-return duplicate** — if service logs, handler does not re-log the same error; return wrapped error only.
- **Write-once `max_points`** — service must catch DB `P0002` from trigger and return `code: "assessment_max_points_immutable"` with message.
- **No-Grade-Ghosting** — score entry must check `enrollment_status` (denormalized) and reject if not `ACTIVE`. Do not rely solely on student-table status; use the score-table column.
- **Publication = refresh** — approval of session must not duplicate DB trigger; instead ensure DB trigger fires (`AFTER UPDATE` on `assessment_sessions` when `status` becomes `PUBLISHED`). If manual refresh needed, expose admin endpoint calling `fn_refresh_term_subject_summary_for_session` / `fn_refresh_subject_strand_summary_for_session` explicitly.
- **RLS / tenant isolation** — all queries include `tenant_id`; foreign keys use composite `(tenant_id, id)`.
- **Isolation rule** — all changes to backend module only in `backend/`. All frontend only in `frontend/`. Do not touch `public/`. `docs/` is for plan/docs only.

---

## 5. Open questions before implementation

1. Does `frontend/src/lib/api/client.ts` already have assessment endpoint stubs? (Check before writing new ones.)
2. Should quantitative score computation happen in Go (replicate `grading_scale_profiles`) or rely solely on DB `calculated_percentage`? Migration comments suggest DB computes; preference: read DB-computed value, compute in Go only for pre-save validation.
3. Are `grading_scale_profiles` and `performance_indicators` already exposed via backend? (Check `curriculum/` module — they may be.)
4. Parent portal auth uses `stytch_org_id` (from `users` table) — verify parent-student linkage (`cbc_student_parents`) is already loaded before building parent result endpoints.
5. Should Phase 5 include `student_cohort_position_summaries` ranking display for parents? (Schema says it exists; decide if privacy-sensitive — probably yes, rank only, not others' names.)

---

## 6. File references (do not edit; read only)

- `docs/schema-details.md` — full table/trigger/enumeration reference.
- `backend/internal/database/migrations/000036_assessment_weight_configs.up.sql` through `000061_assessment_publish_refresh.up.sql` — DDL and trigger logic.
- `docs/cbc/*.json` — CBC curriculum / assessment instruments per grade and stream.
- Root `AGENTS.md` — error contract, isolation, never-commit rule.
- `frontend/AGENTS.md` — frontend error handling (check before implementing client).
- `.pi/skills/pure-shadcn/SKILL.md` — UI component rules for new views.

---

*Plan version: 1.0.0 — 2026-09. Not committed (unstaged modifications only).*
