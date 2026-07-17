# CBE/CBC Report Card Gap Verification

> **Investigation date:** 2026-06  
> **Scope:** `backend/internal/assessments/`, `backend/internal/curriculum/`, `backend/internal/database/migrations/`, `frontend/src/features/assessments/`  
> **Status:** Fact-finding pass — no fixes applied.

---

## Summary Table

| # | Item | Status | Evidence (file:line) | Blast radius if fixed |
|---|------|--------|----------------------|------------------------|
| 1 | Strand / sub-strand granularity | **Confirmed gap** — no strand or sub-strand FK in any assessment table | `backend/internal/database/migrations/000001_initial_schema.up.sql:2605-2737` — `student_assessment_scores` and `student_assessment_outcome_grades` have no `strand_id`/`sub_strand_id` column. Aggregation query at `backend/internal/assessments/repository.go:954-1016` groups by `learning_area_id` only. | **Medium** — schema migration to add nullable `strand_id`/`sub_strand_id` to `assessment_sessions` + adjust aggregation CTE to optionally group by strand. Frontend rubric matrix (`rubric-grading-matrix.tsx`) already drills to indicator level via `learning_area_id`; would need strand selector. |
| 2 | Core Competencies and Values tracking | **Confirmed gap** — zero implementation exists anywhere | No `competency_snapshot` column, no `term_reports` table, no core competencies enum in any migration, domain model, or service. Curriculum seed data (`grade4.json` etc.) uses only strand/sub-strand/indicator hierarchy; no competency or values metadata. `grep -ri "competenc"` across entire repo returned zero relevant results in code (only false positives in JSON curriculum seed data that contain "Oral Communication" as a strand name). | **Large** — new `core_competencies` enum + `student_competency_ratings` table + separate aggregation service entirely. Requires migration, new domain/repo/service, new frontend views. This is an entire feature missing. |
| 3 | Remarks / comments fields | **Confirmed gap** — only `rejection_comment` exists, which is workflow-only | `assessment_sessions.rejection_comment` (`backend/internal/database/migrations/000001_initial_schema.up.sql:2527`) is the *only* free-text field. No `teacher_remarks`, `class_teacher_remarks`, or `head_teacher_remarks` column anywhere in the schema. Frontend `ApprovalActions` (`frontend/src/features/assessments/components/approval-actions.tsx`) only collects `rejection_comment`. The `term_reports` table does not exist. Report card pages (`reports/student/[id]/page.tsx`, `reports/terms/[term_id]/page.tsx`) are stubs with no remark UI. | **Medium** — add free-text columns to `assessment_sessions` or create a `term_reports` table with remark columns. Update frontend report views. |
| 4 | Assessment type vs. evaluation method | **Confirmed orphan** — weight configs are populated but unreachable from any consumer | `assessment_sessions` has only `evaluation_method` (`backend/internal/database/migrations/000001_initial_schema.up.sql:2506`) — no `assessment_type_code` field exists. `assessment_weight_configs` table (`:1398-1424`) uses `assessment_type_code` (free VARCHAR like `"KNEC_SBA_Project"`, `"National_KPSEA"`). No join, no query, no service across the entire codebase reads weight configs in any session/score/aggregation context — only CRUD endpoints (`handler.go:1116-1197`). Only seed data (`000002_seed.up.sql:15-31`) populates 7 rows. | **Small** — if weight configs are meant to feed into KNEC placement calculation, a new service would need to map sessions to weight configs via a lookup table or heuristics. Current schema has no bridge between sessions and weight configs. |
| 5 | Report-card aggregation algorithm ("Last One" mode) | **Partially confirmed** — mode + recency tie-break is correct; **level tie-break has a bug** | CTE chain at `backend/internal/assessments/repository.go:954-1016` implements: `session_scores` → `level_counts` → `max_counts` → `tied_levels` → `ranked` → `final`. Tie-break order: `latest_date DESC, level DESC`. However `level` is cast to `::text`, so `level DESC` sorts *alphabetically* (ME > EE > BE > AE), not by the actual CBC hierarchy (EE > ME > AE > BE). No unit tests exist for this aggregation — `service_test.go` only tests weight config CRUD. | **Small** — fix `ORDER BY level DESC` to use a case expression or enum ordering. One line change in `repository.go:1010`. Add unit tests. |
| 6 | (No item 6 in prompt — separate observations) | See nuance sections below. | — | — |

---

## Per-Item Detailed Findings

### 1. Strand / Sub-Strand Granularity

**Status: Confirmed gap**

**Schema evidence:**

The curriculum hierarchy exists in full:
- `cbc_learning_areas` → `cbc_strands` → `cbc_sub_strands` → `performance_indicators`  
  (migration: lines 1171-1199)

However, `assessment_sessions` only links to `cbc_learning_areas` via `learning_area_id` (line 2514):
```sql
learning_area_id UUID NOT NULL,
CONSTRAINT fk_assessment_sessions_learning_area
    FOREIGN KEY (learning_area_id) REFERENCES cbc_learning_areas(id)
```

Similarly, `student_assessment_scores` (line 2605) and `student_assessment_outcome_grades` (line 2685) have **no** `strand_id` or `sub_strand_id` columns.

**Aggregation query evidence:**

`GetStudentTermGrades()` in `repository.go:954-1016` groups exclusively by `learning_area_id`:
```sql
GROUP BY learning_area_id, learning_area_name, learning_area_code, level
```

The `StudentTermGrade` domain struct also only has `LearningAreaID` — no strand field.

**Cross-module search:**

Searching `"strand"` across the entire repo shows strand/sub-strand are fully modelled only in the `curriculum/` module and its frontend counterpart. The `assessments/` module never references strands. The `rubric-grading-matrix.tsx` frontend component loads performance indicators per learning area for the rubric grid, but it also doesn't filter by strand.

**Blast radius:** Adding strand-level grouping would require:
1. Migration: add nullable `strand_id` to `assessment_sessions` (since a session may cover one strand)
2. Add `StrandID` to `AssessmentSession` domain model
3. Update `GetStudentTermGrades` CTE to optionally aggregate at strand level
4. Frontend: add strand selector to report card views

---

### 2. Core Competencies and Values Tracking

**Status: Confirmed gap — entirely absent**

**Search evidence:**
- `grep -ri "competenc"` across all `.go`, `.ts`, `.tsx`, `.sql` files (excluding node_modules) returned **zero** relevant results.
- `grep -ri "communication\|critical_thinking\|creativity\|citizenship\|self_efficacy\|digital_literacy"` returned hits only in:
  - Curriculum JSON seed data (e.g., `"Oral Communication and Digital Narratives"` as a **strand name**, not a competency framework)
  - False positives in test data and generated files
- `grep -ri "love\|responsibility\|respect\|unity\|peace\|patriotism\|integrity"` — zero results in code

**No `term_reports` table exists in the schema.** There is no `competency_snapshot` JSONB column or any similar construct.

**Consequence:** The CBC framework requires 7 core competencies (Communication, Critical Thinking, Creativity, Citizenship, Learning to Learn, Self-Efficacy, Digital Literacy) and 7 core values (Love, Responsibility, Respect, Unity, Peace, Patriotism, Integrity) to be assessed per term. This is an entire missing feature — not just a gap in what's populated.

**Blast radius:** Large. Requires:
1. Migration: `core_competencies` enum + `student_competency_ratings` table (student_id, term_id, competency, level)
2. Migration: `core_values` enum + `student_values_ratings` table (or combine with competencies)
3. New domain module (`competencies/`) or extend assessments
4. Teacher input UI and aggregation service
5. Report card rendering

---

### 3. Remarks / Comments Fields

**Status: Confirmed gap**

**Schema evidence:**
- `assessment_sessions.rejection_comment` (line 2527) — **only** free-text in the assessment module
- This is purely workflow-related: admin → teacher feedback when rejecting a session
- No `teacher_remarks`, `class_teacher_remarks`, `head_teacher_remarks` column anywhere
- No `term_reports` table that could carry such fields

**Frontend evidence:**
- `ApprovalActions.tsx` (line 54-96) — `rejectionComment` state + Textarea input, only for rejection workflow
- `AssessmentSessionDetailView.tsx` (line 84-93) — displays `session.rejection_comment` in amber alert box
- No remark/comment input exists outside of rejection
- All report card pages are stubs:
  - `reports/student/[id]/page.tsx` — just "Report for student {id}. Select a term to view."
  - `reports/terms/[term_id]/page.tsx` — role-conditional but both branches show placeholder text
  - `reports/bulk-export/page.tsx` — single TODO paragraph

**Blast radius:** Medium. Options:
1. Add remark columns directly to `assessment_sessions` (simpler but denormalised)
2. Create a `term_reports` table with JSONB `remarks` field holding teacher/class-teacher/head-teacher remarks
3. Update frontend report views with remark input fields and display

---

### 4. Assessment Type vs. Evaluation Method

**Status: Confirmed orphan — weight configs have no consumer**

**Schema contrast:**

| Entity | Field | Type | Purpose |
|--------|-------|------|---------|
| `assessment_sessions` | `evaluation_method` | ENUM: QUANTITATIVE, RUBRIC | How the teacher enters scores |
| `assessment_weight_configs` | `assessment_type_code` | VARCHAR(50) | KNEC classification code |

**Key findings:**

1. `assessment_sessions` has **no** `assessment_type_code` field. The only method discriminator is `evaluation_method`.
2. `assessment_weight_configs` is a standalone table with no FK relationship to any other table.
3. **No join exists** between sessions/scores and weight configs anywhere in the codebase.
4. The CRUD endpoints (`handler.go:1116-1197`) are the only consumers — they read/write the table directly.
5. Service methods (`service.go:253-310`) only validate and pass through to repository.
6. Frontend `WeightConfigsList` (`assessments/weight-configs/page.tsx`) is a standalone list page with no integration to scoring.

**Weight config route registration** (`handler.go:84-88`):
```go
wcfg := router.Group("/api/v1/assessments/weight-configs")
wcfg.Get("/", middleware.RequireAuth, h.ListWeightConfigs)
wcfg.Get("/:id", middleware.RequireAuth, h.GetWeightConfig)
wcfg.Post("/", middleware.RequireAuth, middleware.RequireRole("SCHOOL_ADMIN"), h.CreateWeightConfig)
```

Note: the handler comment says "SYSTEM_ADMIN only" but the route check is `SCHOOL_ADMIN`.

**Seed data** (`000002_seed.up.sql`) inserts 7 KNEC formula rows — but nothing reads them.

**Blast radius:** Small if weight configs are intended for future use. If KNEC placement calculation is needed, a bridge between sessions and weight configs must be built — either a new `assessment_session_weight_config` junction table or a heuristic matching by grade_level + assessment metadata.

---

### 5. Report-Card Aggregation ("Last One" Mode)

**Status: Partially confirmed — logic is correct for mode+recency, but level tie-break has an ordering bug**

**CTE chain** (`repository.go:954-1016`):

```
session_scores → level_counts → max_counts → tied_levels → ranked → final
```

This matches the documented pipeline exactly.

**Correct behaviour:**
- Mode wins (most frequent performance level for that learning area)
- Tie-break 1: most recent assessment (`latest_date DESC`)
- Tie-break 2: highest level (`level DESC`)

**Bug found — `level DESC` sorts alphabetically, not hierarchically:**

In `session_scores`, level is cast to `::text`:
```sql
sas.final_performance_level::text AS level,
```
Since it's text, `ORDER BY level DESC` sorts alphabetically:
```
ME → EE → BE → AE  (alphabetical descending)
```
But the correct hierarchical order (highest to lowest) is:
```
EE → ME → AE → BE  (CBC performance hierarchy)
```

**Concrete example:** If a student has BE, ME, ME, EE, EE:
- Mode: EE and ME both appear 2 times → tie
- Latest_date: suppose both EE assessments are from Jan 10, both ME from Feb 20
- Tie-break 1 (latest_date): ME wins because Feb 20 > Jan 10 ✓
- *But if dates are equal*, tie-break 2 uses alphabetical: ME > EE alphabetically, so ME would win. **This is wrong** — EE should win because it's the higher performance level.

**Impact:** In most real-world scenarios, the `latest_date DESC` tie-break will resolve first because assessments within a learning area usually have distinct dates. The level tie-break only matters when two different levels have equal frequency *and* their latest occurrences fall on the same date — a narrow but real edge case.

**Test coverage:**
- `service_test.go` contains **zero tests** for `GetStudentTermGrades` — the mock stubs return `nil, errors.New("not implemented in mock")` for this method (line 115)
- All tests in `service_test.go` cover only `CreateWeightConfig`, `ListWeightConfigs`, `GetWeightConfigByID`
- No integration tests exist for the aggregation query

**Fix required:** Change the tie-break from alphabetical to a case expression or use the enum's natural order:
```sql
-- Option A: map via CASE
ORDER BY latest_date DESC,
  CASE level
    WHEN 'EE' THEN 4
    WHEN 'ME' THEN 3
    WHEN 'AE' THEN 2
    WHEN 'BE' THEN 1
  END DESC

-- Option B: cast back to cbc_performance_level (enum ordinal)
-- cbc_performance_level ENUM is defined as ('EE','ME','AE','BE')
-- So ORDER BY level::cbc_performance_level DESC gives: BE, AE, ME, EE
-- That's still wrong! Need to think about enum ordering...
-- EE(1), ME(2), AE(3), BE(4) → DESC = BE(4), AE(3), ME(2), EE(1) ← also wrong!
-- So casting to enum doesn't help either.
-- The CASE expression is the safest fix.
```

---

## Additional Observations

### Lack of `term_reports` table entirely
There is no `term_reports` or `student_term_reports` table anywhere in the schema. The report card is computed on-the-fly via the `GetStudentTermGrades` aggregation query. This means:
- No persisted report card to audit later
- No place for remarks/comments
- No place for competency snapshots or behavior summaries
- Any change to assessment data after the term ends would retroactively alter the "final" report card

### Reports frontend is all placeholder
The `reports/` pages under `frontend/src/app/(dashboard)/reports/` — admin report listing, student report detail, bulk export, term report — are all stubs with TODO comments. The only working report feature is the parent portal's `ParentAssessmentsView`, which shows the live aggregation but offers no printable layout or teacher/admin management.

### `evaluation_method` naming may be misleading
The field `evaluation_method` with values `QUANTITATIVE` / `RUBRIC` describes the *grading input mechanism*, not a KNEC assessment type. This is functionally accurate for the current system but doesn't align with KNEC's classification (e.g., KNEC_SBA_Project, National_KPSEA). If weight configs are ever wired up, the system will need a way to classify sessions into KNEC assessment types.

### Weight config creation uses wrong role
`handler.go:1162` registers `CreateWeightConfig` with `middleware.RequireRole("SCHOOL_ADMIN")`, but the handler docstring at line 1120 says "Only SYSTEM_ADMIN users can create weight configs". Since weight configs are KNEC national formulas, `SYSTEM_ADMIN` is the correct role. The seed data isn't tenant-scoped, supporting the system-level intent.

---

## File Index

| File | Relevance |
|------|-----------|
| `backend/internal/database/migrations/000001_initial_schema.up.sql` | All table definitions, enums, FKs |
| `backend/internal/assessments/domain.go` | Domain models, DTOs, performance levels |
| `backend/internal/assessments/service.go` | Business logic, validation, state transitions |
| `backend/internal/assessments/repository.go:954-1016` | Report card aggregation CTE |
| `backend/internal/assessments/service_test.go` | Tests (only weight config coverage) |
| `backend/internal/assessments/handler.go` | HTTP handlers, route registration |
| `backend/internal/curriculum/domain.go` | Curriculum hierarchy models |
| `backend/internal/curriculum/cbcdata/grade4.json` | Sample curriculum seed data (strand hierarchy only) |
| `backend/internal/database/migrations/000002_seed.up.sql` | Weight config seed data |
| `frontend/src/features/assessments/components/approval-actions.tsx` | Rejection comment input (only free-text field) |
| `frontend/src/features/assessments/components/assessment-session-detail-view.tsx` | Session detail with rejection display |
| `frontend/src/features/assessments/components/parent-assessments-view.tsx` | Parent portal — only working report UI |
| `frontend/src/app/(dashboard)/reports/student/[id]/page.tsx` | Report page stub |
| `frontend/src/app/(dashboard)/reports/terms/[term_id]/page.tsx` | Term report page stub |
| `frontend/src/app/(dashboard)/reports/bulk-export/page.tsx` | Bulk export stub |
