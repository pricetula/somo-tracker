# Assessments — Complete Flow Analysis

> **Author:** Platform Team  
> **Date:** 2026-07-15  
> **Version:** 1.1  
> **Scope:** Backend (`backend/`) + Frontend (`frontend/`) — multi-tenant CBC school management

---

## Table of Contents

1. [Architecture Overview](#1-architecture-overview)
2. [Domain Model Map](#2-domain-model-map)
3. [Grading Scale Profiles](#3-grading-scale-profiles)
4. [Assessment Sessions — Lifecycle](#4-assessment-sessions--lifecycle)
5. [Two Evaluation Methods](#5-two-evaluation-methods)
6. [Quantitative Score Capture](#6-quantitative-score-capture)
7. [Rubric (Indicator-Level) Grading](#7-rubric-indicator-level-grading)
8. [Raw Score to Rubric Conversion](#8-raw-score-to-rubric-conversion)
9. [Approval Workflow](#9-approval-workflow)
10. [Parent Portal — Published Results](#10-parent-portal--published-results)
11. [Report Card Aggregation](#11-report-card-aggregation)
12. [Term Reports & Bulk Export](#12-term-reports--bulk-export)
13. [Weight Configs](#13-weight-configs)
14. [Database Schema Overview](#14-database-schema-overview)
15. [API Endpoint Reference](#15-api-endpoint-reference)
16. [Frontend Route Reference](#16-frontend-route-reference)

---

## 1. Architecture Overview

The assessments module implements a **Competency-Based Curriculum (CBC)** grading engine with two parallel grading methods:

| Method                         | Description                                                                                             | Use Case                                 |
| ------------------------------ | ------------------------------------------------------------------------------------------------------- | ---------------------------------------- |
| **Quantitative (Marks-Based)** | Teacher enters raw scores (e.g. 41/50), system converts to percentage → rubrics level via scale profile | Summative exams, CATs                    |
| **Rubric (Indicator-Level)**   | Teacher directly assigns EE/ME/AE/BE per KICD performance indicator                                     | Practical skills, projects, observations |

Both methods converge into a **unified report card** that aggregates across the term using the "Last One" chronological mode.

### Tech Stack

| Layer    | Technology                                                             |
| -------- | ---------------------------------------------------------------------- |
| Backend  | Go (Fiber) — hexagonal architecture (handler → service → repository)   |
| Frontend | Next.js 14 (App Router) — React Query for data fetching                |
| Database | PostgreSQL with row-level security (RLS), exclusion constraints, enums |
| DI       | Uber Fx                                                                |

### Directory Layout

```
backend/internal/assessments/
├── domain.go       # Models, params, errors, payloads
├── handler.go      # HTTP handlers (Fiber routes)
├── service.go      # Business logic & validation
├── repository.go   # PostgreSQL persistence (pgx)
├── module.go       # Fx module wiring
├── handler_test.go
└── service_test.go

frontend/src/
├── features/assessments/
│   ├── index.ts                              # Barrel exports
│   ├── types/index.ts                        # Shared type re-exports + constants
│   ├── hooks/use-assessments.ts              # React Query hooks
│   └── components/
│       ├── create-scale-profile-form.tsx
│       ├── create-assessment-session-form.tsx
│       ├── scale-profile-detail-view.tsx
│       ├── set-scale-ranges-form.tsx
│       ├── grading-scale-profiles-list.tsx
│       ├── assessment-sessions-list.tsx
│       ├── assessment-session-detail-view.tsx
│       ├── approval-actions.tsx
│       ├── performance-level-badge.tsx
│       └── status-badge.tsx
├── lib/api/assessments.ts                    # API client (all endpoints)
└── app/(dashboard)/
    ├── assessments/
    │   ├── page.tsx                           # Assessment sessions list
    │   ├── add/page.tsx                       # Create session form
    │   ├── [id]/page.tsx                      # Session detail view
    │   ├── grading-scales/
    │   │   ├── page.tsx                       # Scale profiles list
    │   │   ├── new/page.tsx                   # Create scale profile
    │   │   └── [id]/page.tsx                  # Scale profile detail
    │   └── weight-configs/page.tsx            # KNEC weight configs (placeholder)
    ├── reports/
    │   ├── page.tsx                           # Reports dashboard
    │   ├── student/[id]/page.tsx              # Student report (placeholder)
    │   ├── terms/[term_id]/page.tsx           # Term report (placeholder)
    │   └── bulk-export/page.tsx               # Bulk export (placeholder)
    └── @modal/(.)assessments/                 # Intercepted modals
        ├── add/page.tsx
        ├── grading-scales/
        │   ├── new/page.tsx
        │   └── [id]/page.tsx
```

---

## 2. Domain Model Map

```
┌─────────────────────────────────────────────────────────────────┐
│                    GRADING SCALE PROFILE                         │
│  ┌───────────────┐                                              │
│  │  ScaleProfile │── 1 ── * ──│  ScaleRange                     │
│  │  (directory)  │            │  (EE: 80-100%, ME: 60-79%,     │
│  └───────────────┘            │   AE: 40-59%, BE: 0-39%)        │
│                               └─────────────────────────────────┘
│  Validation: EE, ME, AE required at minimum.                    │
│  Ranges must be contiguous, non-overlapping, within 0-100.     │
└─────────────────────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────────────────────┐
│                     ASSESSMENT SESSION                           │
│  ┌────────────────────┐              ┌──────────────────────────┐
│  │  AssessmentSession │── 1 ── * ───│  StudentScore             │
│  │                    │              │  (QUANTITATIVE only)      │
│  │  Status: DRAFT     │              │  - raw_score              │
│  │         PENDING    │              │  - calculated_percentage  │
│  │         PUBLISHED  │              │  - final_performance_lvl │
│  │                    │              └──────────────────────────┘
│  │  EvaluationMethod  │              ┌──────────────────────────┐
│  │  QUANTITATIVE:     │── 1 ── * ───│  OutcomeGrade             │
│  │    has max_points  │              │  (RUBRIC only)            │
│  │    + scale_profile │              │  - performance_indicator  │
│  │                    │              │  - awarded_level (EE/ME)  │
│  │  RUBRIC:           │              └──────────────────────────┘
│  │    no max_points   │
│  │    no scale        │
│  └────────────────────┘
└─────────────────────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────────────────────┐
│                    REPORT & AGGREGATION                          │
│  ┌──────────────────┐    ┌──────────────────────┐               │
│  │ StudentTermGrade │◄───│  "Last One" Mode:    │               │
│  │                  │    │  1. Mode (most freq)  │               │
│  │  per learning_   │    │  2. Tie-break: latest │               │
│  │  area per term   │    │     chronological     │               │
│  └──────────────────┘    └──────────────────────┘               │
│                                                                 │
│  ┌──────────────────────┐                                       │
│  │  ParentAssessmentView │  (published results for parent)      │
│  └──────────────────────┘                                       │
└─────────────────────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────────────────────┐
│                    WEIGHT CONFIGS (KNEC)                         │
│  ┌────────────────────────────────────┐                         │
│  │ AssessmentWeightConfig             │                         │
│  │  grade_level + type + target_exam  │                         │
│  │  + weight_percent + effective_year │                         │
│  └────────────────────────────────────┘                         │
│  Global, system-level. Populated by SYSTEM_ADMIN.               │
└─────────────────────────────────────────────────────────────────┘
```

### CBC Performance Levels

| Code   | Label                   | Numeric Order |
| ------ | ----------------------- | ------------- |
| **EE** | Exceeding Expectation   | 4 (highest)   |
| **ME** | Meeting Expectation     | 3             |
| **AE** | Approaching Expectation | 2             |
| **BE** | Below Expectation       | 1 (lowest)    |

### Session Statuses

| Status             | Description                       | Editable? | Parent Visible? |
| ------------------ | --------------------------------- | --------- | --------------- |
| `DRAFT`            | Teacher entering scores/grades    | ✅ Yes    | ❌ No           |
| `PENDING_APPROVAL` | Submitted for admin review        | ❌ Locked | ❌ No           |
| `PUBLISHED`        | Approved by admin, results frozen | ❌ Locked | ✅ Yes          |

---

## 3. Grading Scale Profiles

### Purpose

A **grading scale profile** is a named collection of percentage-to-CBC-level conversion rules. It defines the mapping between numeric scores (percentages) and rubric levels (EE/ME/AE/BE). Profiles are school-scoped and tenant-isolated.

### Create Flow

```
[SCHOOL_ADMIN]
     │
     ├── Navigate to /assessments/grading-scales
     │   → GradingScaleProfilesList: DataTable of existing profiles
     │
     ├── Click "Add Profile" → /assessments/grading-scales/new
     │   (intercepted as modal via @modal overlay)
     │
     ├── Fill CreateScaleProfileForm:
     │   ├── Profile Name (256 chars max, immutable after creation)
     │   └── Percentage Ranges (grid of 4 levels)
     │       ├── EE: min % / max % (required)
     │       ├── ME: min % / max % (required)
     │       ├── AE: min % / max % (required)
     │       └── BE: min % / max % (recommended)
     │
     ├── Client-side validation:
     │   ├── Name is required
     │   ├── EE, ME, AE ranges must be filled
     │   ├── Percentages between 0-100
     │   ├── min < max for each range
     │   ├── No overlaps (sorted by min, each starts after previous max)
     │
     ├── POST /api/v1/grading/profiles
     │   → Service.CreateScaleProfile()
     │   → Service.validateScaleRanges(): server-side dupe of validation
     │   → Repo.CreateScaleProfileWithRanges(): atomic transaction
     │       ├── INSERT grading_scale_profiles
     │       └── INSERT grading_scale_ranges (one per level)
     │
     └── Returns { id, range_ids } → redirects to /assessments/grading-scales/:id
```

### Editing Ranges

Ranges can be replaced atomically via the detail page (`SetScaleRangesForm`):

```
PUT /api/v1/grading/profiles/:id/ranges
  → Service.ReplaceScaleRanges()
  → Repo.ReplaceScaleRanges(): atomic tx
      ├── DELETE FROM grading_scale_ranges WHERE profile_id = $1
      └── INSERT new ranges
```

### Toggle Active / Delete

- **Toggle:** `PUT /api/v1/grading/profiles/:id/toggle?is_active=false` — deprecates profiles gracefully. Historical sessions referencing the profile remain interpretable.
- **Delete:** `DELETE /api/v1/grading/profiles/:id` — only allowed if **no sessions** reference this profile (`ErrScaleReferenced` otherwise). Use toggle instead.

### Validation Rules (Server-Side)

| Rule                                           | Error              |
| ---------------------------------------------- | ------------------ |
| Name required                                  | `ErrInvalidInput`  |
| Max 255 characters                             | `ErrInvalidInput`  |
| At least one range                             | `ErrInvalidInput`  |
| Must include EE, ME, AE levels                 | `ErrInvalidInput`  |
| min_percentage/max_percentage within 0–100     | `ErrInvalidInput`  |
| max > min                                      | `ErrInvalidInput`  |
| Ranges must not overlap (exclusion constraint) | `ErrConflict`      |
| Duplicate performance level in same profile    | `ErrAlreadyExists` |

---

## 4. Assessment Sessions — Lifecycle

A session represents a single assessment event (e.g. "Mathematics CAT 1" for Class 4A).

```
  ┌───────┐     Submit      ┌──────────────────┐
  │ DRAFT │────────────────→│ PENDING_APPROVAL  │
  └───┬───┘                 └────────┬─────────┘
      │                              │
      │  (teacher edits              │
      │   scores/grades)      ┌──────┴──────┐
      │                       │             │
      │                  Approve         Reject
      │                       │             │
      │                       ↓             ↓
      │                  ┌──────────┐   ┌─────────┐
      │                  │ PUBLISHED│   │  DRAFT  │◄── (returned for revision)
      │                  └──────────┘   └─────────┘
      │                  (terminal)
      │
      └── Student scores/grades editable only in DRAFT
```

### Create Session Flow

```
[TEACHER]
     │
     ├── Navigate to /assessments → AssessmentSessionsList
     ├── Click "Create" → /assessments/add
     │
     ├── Fill CreateAssessmentSessionForm:
     │   ├── Assessment Name
     │   ├── Class (combobox)
     │   ├── Learning Area / Subject (combobox)
     │   ├── Academic Year (combobox)
     │   ├── Academic Term (combobox)
     │   ├── Evaluation Method:
     │   │   ├── "Marks-Based (QUANTITATIVE)" → shows:
     │   │   │   ├── Max Points (numeric)
     │   │   │   └── Grading Scale Profile (dropdown of active profiles)
     │   │   └── "Rubric (RUBRIC)" → hides max_points and scale fields
     │   └── Scheduled Date (optional, date picker)
     │
     ├── Client-side validation: required fields, max_points validation
     │
     ├── POST /api/v1/assessments/sessions
     │   → Service.CreateSession()
     │   │   ├── Validates: tenant, school, all IDs, name, evaluation_method
     │   │   ├── QUANTITATIVE: requires max_points > 0 and scale_profile_id
     │   │   ├── RUBRIC: must NOT include max_points or scale_profile_id
     │   │   └── Term finalisation check (ErrTermFinalised if is_final = true)
     │   └── Repo.CreateSession(): INSERT with status = 'DRAFT'
     │
     └── Returns { id } → redirects to /assessments/:id
```

---

## 5. Two Evaluation Methods

### QUANTITATIVE (Marks-Based)

Used for scored assessments like exams, CATs, and standardized tests.

| Field                      | Required?     | Purpose                                |
| -------------------------- | ------------- | -------------------------------------- |
| `max_points`               | ✅ Required   | Total possible marks (e.g. 50)         |
| `grading_scale_profile_id` | ✅ Required   | Maps % to EE/ME/AE/BE                  |
| Raw scores per student     | Entered later | Each student's score out of max_points |

Data flow:

```
raw_score → calculated_percentage = (raw_score / max_points) × 100
          → scale_profile ranges → final_performance_level (at approval)
```

### RUBRIC (Indicator-Level)

Used for practical assessments where teachers evaluate per KICD performance indicator.

| Field                                  | Required?     | Purpose                                 |
| -------------------------------------- | ------------- | --------------------------------------- |
| `max_points`                           | ❌ Prohibited | No numeric scoring                      |
| `grading_scale_profile_id`             | ❌ Prohibited | Levels assigned directly                |
| Outcome grades per student × indicator | Entered later | Each (student, indicator) → EE/ME/AE/BE |

Data flow:

```
Teacher assigns EE/ME/AE/BE per indicator directly
  → stored in student_assessment_outcome_grades
  → no conversion needed
```

---

## 6. Quantitative Score Capture

### Entering Scores

```
[TEACHER] — Session must be in DRAFT
     │
     ├── POST /api/v1/assessments/sessions/:id/scores
     │   Body: { scores: [{ student_id, raw_score }, ...] }
     │
     ├── Service.BulkUpsertScores()
     │   ├── Checks status == 'DRAFT' (ErrInvalidStateTransition if not)
     │   ├── Checks term not finalised (ErrTermFinalised)
     │   ├── Validates each: student_id required, raw_score ≥ 0
     │   └── Blocks ABSENT/EXEMPT enrollment status (ErrStudentNotGradable)
     │
     └── Repo.BulkUpsertStudentScores()
         ├── Gets max_points from assessment_sessions
         ├── For each student:
         │   ├── calculated_percentage = (raw_score / max_points) × 100
         │   └── UPSERT: INSERT ... ON CONFLICT (session_id, student_id)
         │              → Only updates if final_performance_level IS NULL
         │                (prevents overwriting approved/published snapshots)
         └── Transactional: all-or-nothing
```

### Score Schema

| Column                    | Example  | Notes                                           |
| ------------------------- | -------- | ----------------------------------------------- |
| `raw_score`               | 41       | Raw marks scored                                |
| `calculated_percentage`   | 82.0     | Computed automatically                          |
| `final_performance_level` | "EE"     | Snapshotted at approval time — null before then |
| `enrollment_status`       | "ACTIVE" | ABSENT/EXEMPT cannot receive scores             |

### Upsert Idempotency

The `ON CONFLICT (session_id, student_id)` clause makes re-submission safe. The `WHERE final_performance_level IS NULL` guard prevents overwriting after the session is approved/published.

---

## 7. Rubric (Indicator-Level) Grading

### Entering Outcome Grades

```
[TEACHER] — Session must be in DRAFT
     │
     ├── POST /api/v1/assessments/sessions/:id/grades
     │   Body: { grades: [
     │     { student_id, performance_indicator_id, awarded_level: "ME" },
     │     ...
     │   ]}
     │
     ├── Service.BulkUpsertOutcomeGrades()
     │   ├── Checks status == 'DRAFT'
     │   ├── Checks term not finalised
     │   ├── Validates: student_id, indicator_id, valid CBC level
     │   └── ValidPerformanceLevels: ["EE", "ME", "AE", "BE"]
     │
     └── Repo.BulkUpsertOutcomeGrades()
         └── UPSERT per (session_id, student_id, performance_indicator_id)
```

### Outcome Grade Schema

| Column                     | Example                                | Notes                 |
| -------------------------- | -------------------------------------- | --------------------- |
| `performance_indicator_id` | "ind-001"                              | FK to KICD indicators |
| `awarded_level`            | "ME"                                   | EE/ME/AE/BE only      |
| Unique constraint          | (session_id, student_id, indicator_id) | One level per triple  |

### Key Difference from Quantitative

- **No raw scores** — teachers evaluate holistically
- **No percentage conversion** — levels are direct
- **No scale profile needed** — grading is per-indicator
- **Multiple grades per student** — one per indicator per session

---

## 8. Raw Score to Rubric Conversion

This is the critical bridge between quantitative scoring and the CBC rubric system. The conversion happens **at approval time**, not at score entry time.

### The Conversion Mechanism

```
At session approval (ApproveSession):
     │
     ├── Only for QUANTITATIVE sessions with a grading_scale_profile_id
     │
     ├── 1. Fetch the ScaleProfile (with its ranges)
     │     └── GET grading_scale_ranges WHERE profile_id = $1
     │
     ├── 2. For each score in the session:
     │     └── UPDATE student_assessment_scores
     │         SET final_performance_level = (
     │           SELECT r.performance_level
     │           FROM grading_scale_ranges r
     │           WHERE r.profile_id = $2
     │             AND score.calculated_percentage >= r.min_percentage
     │             AND score.calculated_percentage <= r.max_percentage
     │           LIMIT 1
     │         )
     │         WHERE session_id = $1
     │           AND final_performance_level IS NULL
     │           AND calculated_percentage IS NOT NULL
     │
     └── Result: each student's percentage → snapshotted CBC level
```

### Example Conversion

Given a scale profile with these ranges:

| Level | Min % | Max % |
| ----- | ----- | ----- |
| BE    | 0     | 39    |
| AE    | 40    | 59    |
| ME    | 60    | 79    |
| EE    | 80    | 100   |

| Student | Raw Score | Max Points | %     | Converted Level |
| ------- | --------- | ---------- | ----- | --------------- |
| Kamau   | 41        | 50         | 82.0% | EE              |
| Achesa  | 38        | 50         | 76.0% | ME              |
| Juma    | 25        | 50         | 50.0% | AE              |
| Otieno  | 12        | 50         | 24.0% | BE              |

### Important Design Decisions

1. **Conversion is a snapshot** — once written, `final_performance_level` is frozen. Even if the scale profile's ranges change later, historical results remain accurate.

2. **Conversion is irreversible** — the `WHERE final_performance_level IS NULL` guard in `SnapshotPerformanceLevels` and the upsert query prevents overwriting.

3. **Guard for ABSENT/EXEMPT** — these students have `calculated_percentage = NULL`, so the conversion query skips them (they get no `final_performance_level`).

4. **Rubric sessions skip conversion entirely** — outcome grades already have the level.

---

## 9. Approval Workflow

### 9.1 Submit for Approval (Teacher)

```
POST /api/v1/assessments/sessions/:id/submit
  → Service.SubmitSession()
      ├── Session must be in DRAFT
      ├── Term must not be finalised
      ├── allowedTransitions["DRAFT"]["PENDING_APPROVAL"] must exist
      └── Repo.UpdateSessionStatus():
          ├── status = 'PENDING_APPROVAL'
          ├── submitted_by = user_id
          ├── rejection_comment = NULL (clear previous)
          └── approved_by = NULL (clear previous)
```

**Frontend trigger:** `/assessments/:id` → "Submit for Approval" button (visible in DRAFT)

### 9.2 Approve & Publish (School Admin)

```
POST /api/v1/assessments/sessions/:id/approve
  → Service.ApproveSession()
      ├── Session must be in PENDING_APPROVAL
      ├── Term must not be finalised
      └── IF QUANTITATIVE:
          ├── Fetch scale profile (validates it still exists)
          └── Repo.SnapshotPerformanceLevels():
              ├── Loads grading_scale_ranges for the profile
              └── Runs UPDATE to set final_performance_level
              └── See §8 for the SQL
      └── Repo.UpdateSessionStatus():
          ├── status = 'PUBLISHED'
          └── approved_by = user_id
```

**Frontend trigger:** `/assessments/:id` → "Approve & Publish" button (ApprovalActions component, visible in PENDING_APPROVAL)

### 9.3 Reject & Return to Draft (School Admin)

```
POST /api/v1/assessments/sessions/:id/reject
  Body: { rejection_comment: "..." }
  → Service.RejectSession()
      ├── Session must be in PENDING_APPROVAL
      ├── rejection_comment required (trimmed, non-empty)
      ├── Term must not be finalised
      └── Repo.UpdateSessionStatus():
          ├── status = 'DRAFT'
          ├── rejection_comment = comment
          ├── submitted_by = NULL (unlocks for teacher)
          └── approved_by = NULL
```

**Frontend trigger:** `/assessments/:id` → "Reject" dialog (requires comment textarea)

### 9.4 State Transition Rules

```
allowedTransitions = {
  "DRAFT":              { "PENDING_APPROVAL": true },
  "PENDING_APPROVAL":   { "DRAFT": true, "PUBLISHED": true },
  "PUBLISHED":          {},  // terminal
}
```

### 9.5 User Auditing

| Field               | Set By  | When                      |
| ------------------- | ------- | ------------------------- |
| `submitted_by`      | Teacher | Submit → PENDING_APPROVAL |
| `approved_by`       | Admin   | Approve → PUBLISHED       |
| `rejection_comment` | Admin   | Reject → DRAFT            |

---

## 10. Parent Portal — Published Results

### 10.1 Fetching Published Assessments

```
GET /api/v1/parent/students/:studentId/assessments?academic_term_id=:termId
  → Service.GetParentAssessments()
      ├── Fetches all PUBLISHED sessions for this student + term
      ├── For QUANTITATIVE: includes raw_score, max_points, performance_level
      ├── For RUBRIC: fetches OutcomeGrades per student per session
      └── Returns ParentAssessmentView[]
```

**CBC Compliance:**

- Raw scores ARE shown here (for context), unlike traditional report cards
- The performance_level (EE/ME/AE/BE) is the primary output
- Outcome grades for RUBRIC sessions show per-indicator levels

### 10.2 Parent View Example

```json
{
  "items": [
    {
      "session_id": "uuid-1",
      "session_name": "Mathematics CAT 1",
      "evaluation_method": "QUANTITATIVE",
      "scheduled_date": "2026-01-15",
      "raw_score": 41,
      "max_points": 50,
      "performance_level": "EE",
      "outcome_grades": null
    },
    {
      "session_id": "uuid-2",
      "session_name": "Practical Skills",
      "evaluation_method": "RUBRIC",
      "scheduled_date": "2026-02-01",
      "raw_score": null,
      "max_points": null,
      "performance_level": null,
      "outcome_grades": [
        { "performance_indicator_id": "ind-001", "awarded_level": "ME" },
        { "performance_indicator_id": "ind-002", "awarded_level": "AE" }
      ]
    }
  ]
}
```

---

## 11. Report Card Aggregation

### "Last One" Chronological Mode

The term-end report card uses a **mode-with-tiebreaker** algorithm to compile a single final level per learning area.

```
For each learning area the student took assessments in:

  1. COLLECT all published performance levels from the term:
     - QUANTITATIVE sessions: use final_performance_level (snapshotted)
     - RUBRIC sessions: use awarded_level (per indicator)
     - All levels include assessment_date (session.created_at)

  2. GROUP BY level → COUNT occurrences → FIND mode
     - Most frequent level wins

  3. TIE-BREAK: if two+ levels have same frequency:
     - Pick the level from the chronologically LATEST assessment
     - If still tied, pick the higher level (EE > ME > AE > BE)
```

### Example

Kamau's Mathematics results for Term 1:

| Assessment  | Date   | Level |
| ----------- | ------ | ----- |
| CAT 1       | Jan 15 | EE    |
| Practical 1 | Feb 1  | ME    |
| Practical 2 | Feb 1  | AE    |
| CAT 2       | Feb 20 | ME    |
| Project     | Mar 10 | EE    |

**Calculation:**

| Level | Count |
| ----- | ----- |
| EE    | 2     |
| ME    | 2     |
| AE    | 1     |

**Tie between EE and ME (both count = 2)**  
→ Latest among tied: EE (Mar 10) vs ME (Feb 20) → **EE wins**

### SQL Implementation

The aggregation is done entirely in PostgreSQL using a CTE chain:

```
session_scores → (UNION of quantitative + rubric scores)
level_counts → (GROUP BY learning_area, level, COUNT, MAX(date))
max_counts → (per learning area: max count)
tied_levels → (filter to levels matching max count)
ranked → (ROW_NUMBER by latest_date DESC, level DESC)
final → (pick rn=1 per learning area)
```

### API Endpoint

```
GET /api/v1/parent/students/:studentId/report-card?academic_term_id=:termId

Response:
{
  "items": [
    {
      "learning_area_id": "uuid",
      "learning_area_name": "Mathematics",
      "learning_area_code": "MATH",
      "final_level": "EE",
      "assessment_count": 5
    }
  ]
}
```

**Frontend state:** The `/reports` page exists but is a placeholder. Hooks `useStudentTermGrades` and `useParentAssessments` are ready for consumption.

---

## 12. Term Reports & Bulk Export

### Current Status

| Feature                         | Backend          | Frontend       | Status                                                                                             |
| ------------------------------- | ---------------- | -------------- | -------------------------------------------------------------------------------------------------- |
| `term_reports` table            | ✅ Schema exists | N/A            | Table defined with `attendance_snapshot`, `behavior_snapshot`, `competency_snapshot` JSONB columns |
| Term report generation service  | ❌ Missing       | ❌ Missing     | No service populates the `term_reports` table                                                      |
| Term report publishing workflow | ❌ Missing       | ❌ Missing     | DRAFT → PUBLISHED not implemented                                                                  |
| Student report page             | ✅ Route exists  | ❌ Placeholder | `/reports/student/[id]` shows "Select a term"                                                      |
| Term report page                | ✅ Route exists  | ❌ Placeholder | `/reports/terms/[term_id]` shows role-switched shell                                               |
| Bulk export                     | ❌ Missing       | ❌ Placeholder | `/reports/bulk-export` shows placeholder                                                           |
| Report generation API           | ❌ Missing       | ❌ Missing     | No backend endpoint to trigger generation                                                          |

### Planned Architecture for Term Reports

```
[Admin] → Select term → Select student(s)
  → POST /api/v1/reports/generate
    → Service:
        1. Query attendance_term_summaries → attendance_snapshot
        2. Query behavior_notes (approved) → behavior_snapshot
        3. Query learner rubric results / term grades → competency_snapshot
        4. INSERT / UPDATE term_reports
        5. Status = DRAFT
  → Admin reviews → POST /api/v1/reports/:id/publish
    → Status = PUBLISHED
  → Parent sees compiled report
```

---

## 13. Weight Configs

### Purpose

Weight Configs define the KNEC-mandated national weighting formulas. They specify how different assessment type scores contribute to the **target exam placement score** (e.g. KPSEA, KJSEA, KSSEA).

### Schema

| Column                 | Example            | Notes                                 |
| ---------------------- | ------------------ | ------------------------------------- |
| `grade_level`          | `GRADE_4`          | Which grade this applies to           |
| `assessment_type_code` | `KNEC_SBA_Project` | Type of assessment                    |
| `target_exam`          | `KPSEA`            | Which national exam it contributes to |
| `weight_percent`       | `20.0`             | Percentage contribution (0-100)       |
| `effective_from`       | `2026`             | Academic year this becomes active     |

### Example Configs

| Grade   | Assessment Type         | Target Exam | Weight |
| ------- | ----------------------- | ----------- | ------ |
| GRADE_4 | KNEC_SBA_Project        | KPSEA       | 20%    |
| GRADE_4 | KNEC_Written_Assessment | KPSEA       | 80%    |
| GRADE_6 | KNEC_SBA_Project        | KPSEA       | 30%    |
| GRADE_6 | KNEC_Written_Assessment | KPSEA       | 70%    |

### API

- **GET /api/v1/assessments/weight-configs** — list with optional filters (grade_level, target_exam, effective_from)
- **GET /api/v1/assessments/weight-configs/:id** — single config
- **POST /api/v1/assessments/weight-configs** — create (SYSTEM_ADMIN only)

**Frontend state:** `/assessments/weight-configs` page is a placeholder with "coming soon" messaging.

---

## 14. Database Schema Overview

### Table: `grading_scale_profiles`

| Column       | Type                  | Notes                    |
| ------------ | --------------------- | ------------------------ |
| `id`         | UUID PK               |                          |
| `tenant_id`  | UUID NOT NULL         | Multi-tenant isolation   |
| `school_id`  | UUID NOT NULL         |                          |
| `name`       | VARCHAR(255) NOT NULL | Immutable after creation |
| `is_active`  | BOOLEAN DEFAULT true  | Toggle for deprecation   |
| `created_at` | TIMESTAMPTZ           |                          |
| `updated_at` | TIMESTAMPTZ           |                          |

### Table: `grading_scale_ranges`

| Column                       | Type                            | Notes                    |
| ---------------------------- | ------------------------------- | ------------------------ |
| `id`                         | UUID PK                         |                          |
| `profile_id`                 | UUID FK                         | → grading_scale_profiles |
| `performance_level`          | cbc_performance_level ENUM      | EE, ME, AE, BE           |
| `min_percentage`             | NUMERIC(5,2)                    | Inclusive                |
| `max_percentage`             | NUMERIC(5,2)                    | Inclusive                |
| `default_percentage_mapping` | NUMERIC(5,2) NULL               | Optional midpoint        |
| **Exclusion**                | GiST `numrange(min, max, '[)')` | No overlap per profile   |

### Table: `assessment_sessions`

| Column                                  | Type                              | Notes                              |
| --------------------------------------- | --------------------------------- | ---------------------------------- |
| `id`                                    | UUID PK                           |                                    |
| `tenant_id` / `school_id`               | UUID                              | RLS enabled                        |
| `class_id` / `learning_area_id`         | UUID                              | FK refs                            |
| `academic_term_id` / `academic_year_id` | UUID                              | FK refs                            |
| `name`                                  | VARCHAR(255)                      |                                    |
| `evaluation_method`                     | assessment_evaluation_method ENUM | QUANTITATIVE or RUBRIC             |
| `max_points`                            | NUMERIC(5,2) NULL                 | For QUANTITATIVE                   |
| `grading_scale_profile_id`              | UUID NULL                         | For QUANTITATIVE                   |
| `status`                                | assessment_session_status ENUM    | DRAFT, PENDING_APPROVAL, PUBLISHED |
| `rejection_comment`                     | TEXT NULL                         | Set on reject, cleared on resubmit |
| `submitted_by` / `approved_by`          | UUID NULL                         | User audit trail                   |
| `scheduled_date`                        | DATE NULL                         |                                    |
| `created_by`                            | UUID                              |                                    |

### Table: `student_assessment_scores`

| Column                      | Type                         | Notes                   |
| --------------------------- | ---------------------------- | ----------------------- |
| `id`                        | UUID PK                      |                         |
| `session_id` / `student_id` | UUID                         | FK refs                 |
| `raw_score`                 | NUMERIC(5,2) NULL            |                         |
| `calculated_percentage`     | NUMERIC(5,2) NULL            | (raw/max)×100           |
| `final_performance_level`   | cbc_performance_level NULL   | Snapshotted at approval |
| `enrollment_status`         | VARCHAR(20) DEFAULT 'ACTIVE' |                         |
| **Unique**                  | (session_id, student_id)     |                         |

### Table: `student_assessment_outcome_grades`

| Column                      | Type                                   | Notes                |
| --------------------------- | -------------------------------------- | -------------------- |
| `id`                        | UUID PK                                |                      |
| `session_id` / `student_id` | UUID                                   | FK refs              |
| `performance_indicator_id`  | UUID                                   | FK → KICD indicators |
| `awarded_level`             | cbc_performance_level                  | EE/ME/AE/BE          |
| **Unique**                  | (session_id, student_id, indicator_id) |                      |

### Table: `assessment_weight_configs`

| Column                 | Type                                             | Notes                      |
| ---------------------- | ------------------------------------------------ | -------------------------- |
| `id`                   | UUID PK                                          |                            |
| `grade_level`          | cbc_grade_level                                  | System-wide (no tenant_id) |
| `assessment_type_code` | VARCHAR(50)                                      |                            |
| `target_exam`          | VARCHAR(20)                                      | e.g. KPSEA                 |
| `weight_percent`       | NUMERIC(5,2)                                     | 0-100                      |
| `effective_from`       | SMALLINT                                         | Year                       |
| **Unique**             | (grade_level, type, target_exam, effective_from) |                            |

### Table: `term_reports`

| Column                                   | Type                           | Notes                     |
| ---------------------------------------- | ------------------------------ | ------------------------- |
| `id`                                     | UUID PK                        |                           |
| `tenant_id` / `school_id` / `student_id` | UUID                           |                           |
| `academic_term_id`                       | UUID                           |                           |
| `attendance_snapshot`                    | JSONB                          | Frozen attendance summary |
| `behavior_snapshot`                      | JSONB                          | Frozen behaviour notes    |
| `competency_snapshot`                    | JSONB                          | Frozen assessment grades  |
| `status`                                 | term_report_status             | DRAFT / PUBLISHED         |
| `generated_at` / `published_at`          | TIMESTAMPTZ                    |                           |
| **Unique**                               | (student_id, academic_term_id) |                           |

---

## 15. API Endpoint Reference

### Grading Scale Profiles

| Method | Path                                  | Auth     | Role         | Purpose                    |
| ------ | ------------------------------------- | -------- | ------------ | -------------------------- |
| POST   | `/api/v1/grading/profiles`            | Required | —            | Create profile + ranges    |
| GET    | `/api/v1/grading/profiles`            | Required | —            | List profiles              |
| GET    | `/api/v1/grading/profiles/:id`        | Required | —            | Get profile with ranges    |
| PUT    | `/api/v1/grading/profiles/:id/toggle` | Required | SCHOOL_ADMIN | Toggle active              |
| DELETE | `/api/v1/grading/profiles/:id`        | Required | SCHOOL_ADMIN | Delete (blocked if in use) |
| GET    | `/api/v1/grading/profiles/:id/ranges` | Required | —            | Get ranges                 |
| PUT    | `/api/v1/grading/profiles/:id/ranges` | Required | SCHOOL_ADMIN | Replace ranges atomically  |

### Assessment Sessions

| Method | Path                                       | Auth     | Role         | Purpose                  |
| ------ | ------------------------------------------ | -------- | ------------ | ------------------------ |
| POST   | `/api/v1/assessments/sessions`             | Required | —            | Create session (DRAFT)   |
| GET    | `/api/v1/assessments/sessions`             | Required | —            | List with filters        |
| GET    | `/api/v1/assessments/sessions/:id`         | Required | —            | Get session              |
| POST   | `/api/v1/assessments/sessions/:id/submit`  | Required | —            | DRAFT → PENDING_APPROVAL |
| POST   | `/api/v1/assessments/sessions/:id/approve` | Required | SCHOOL_ADMIN | → PUBLISHED + snapshot   |
| POST   | `/api/v1/assessments/sessions/:id/reject`  | Required | SCHOOL_ADMIN | → DRAFT + comment        |

### Student Scores (Quantitative)

| Method | Path                                      | Auth     | Role | Purpose                    |
| ------ | ----------------------------------------- | -------- | ---- | -------------------------- |
| POST   | `/api/v1/assessments/sessions/:id/scores` | Required | —    | Bulk upsert raw scores     |
| GET    | `/api/v1/assessments/sessions/:id/scores` | Required | —    | Get scores with % + levels |

### Outcome Grades (Rubric)

| Method | Path                                      | Auth     | Role | Purpose                   |
| ------ | ----------------------------------------- | -------- | ---- | ------------------------- |
| POST   | `/api/v1/assessments/sessions/:id/grades` | Required | —    | Bulk upsert rubric grades |
| GET    | `/api/v1/assessments/sessions/:id/grades` | Required | —    | Get outcome grades        |

### Parent & Reports

| Method | Path                                             | Auth     | Role | Purpose               |
| ------ | ------------------------------------------------ | -------- | ---- | --------------------- |
| GET    | `/api/v1/parent/students/:studentId/assessments` | Required | —    | Published assessments |
| GET    | `/api/v1/parent/students/:studentId/report-card` | Required | —    | Compiled term grades  |

### Weight Configs

| Method | Path                                     | Auth     | Role         | Purpose           |
| ------ | ---------------------------------------- | -------- | ------------ | ----------------- |
| GET    | `/api/v1/assessments/weight-configs`     | Required | —            | List with filters |
| GET    | `/api/v1/assessments/weight-configs/:id` | Required | —            | Get single config |
| POST   | `/api/v1/assessments/weight-configs`     | Required | SYSTEM_ADMIN | Create config     |

---

## 16. Frontend Route Reference

| Route                              | Component                     | Purpose                                   | Status         |
| ---------------------------------- | ----------------------------- | ----------------------------------------- | -------------- |
| `/assessments`                     | `AssessmentSessionsList`      | List all sessions with filters            | ✅ Complete    |
| `/assessments/add`                 | `CreateAssessmentSessionForm` | Create session form                       | ✅ Complete    |
| `/assessments/[id]`                | `AssessmentSessionDetailView` | Session detail + scores/grades + workflow | ✅ Complete    |
| `/assessments/grading-scales`      | `GradingScaleProfilesList`    | List scale profiles                       | ✅ Complete    |
| `/assessments/grading-scales/new`  | `CreateScaleProfileForm`      | Create profile + ranges                   | ✅ Complete    |
| `/assessments/grading-scales/[id]` | `ScaleProfileDetailView`      | Profile detail + range editor             | ✅ Complete    |
| `/assessments/weight-configs`      | Placeholder                   | KNEC weight configs viewer                | ❌ Placeholder |
| `/reports`                         | Role-dispatched               | Reports dashboard                         | ❌ Placeholder |
| `/reports/student/[id]`            | Placeholder                   | Student report                            | ❌ Placeholder |
| `/reports/terms/[term_id]`         | Role-dispatched               | Term report detail                        | ❌ Placeholder |
| `/reports/bulk-export`             | Placeholder                   | Bulk report export                        | ❌ Placeholder |

---

## Appendix A: End-to-End Flow Examples

### A.1 Quantitative Assessment — Full Lifecycle

```
┌─────────────────────────────────────────────────────────────────────────────┐
│ 1. Admin creates scale profile                                             │
│    POST /api/v1/grading/profiles → { name: "Grade 4 Standard", ranges }    │
│    → Returns profile ID: "prof-001"                                        │
├─────────────────────────────────────────────────────────────────────────────┤
│ 2. Teacher creates QUANTITATIVE session                                    │
│    POST /api/v1/assessments/sessions                                       │
│    → { name: "Maths CAT 1", evaluation_method: "QUANTITATIVE",             │
│        max_points: 50, grading_scale_profile_id: "prof-001", ... }         │
│    → Returns session ID: "sess-001"                                        │
├─────────────────────────────────────────────────────────────────────────────┤
│ 3. Teacher enters scores (DRAFT)                                           │
│    POST /api/v1/assessments/sessions/sess-001/scores                       │
│    → { scores: [{ student_id: "s1", raw_score: 41 }, ...] }                │
│    → System computes calculated_percentage automatically                   │
├─────────────────────────────────────────────────────────────────────────────┤
│ 4. Teacher submits for approval                                            │
│    POST /api/v1/assessments/sessions/sess-001/submit                       │
│    → Status: DRAFT → PENDING_APPROVAL                                      │
├─────────────────────────────────────────────────────────────────────────────┤
│ 5. Admin approves → snapshot happens                                       │
│    POST /api/v1/assessments/sessions/sess-001/approve                      │
│    → Status: PUBLISHED                                                      │
│    → Kamau: 41/50 = 82% → EE (snapshotted)                                 │
│    → Achesa: 38/50 = 76% → ME                                              │
│    → Juma: 25/50 = 50% → AE                                                │
├─────────────────────────────────────────────────────────────────────────────┤
│ 6. Parent views published results                                          │
│    GET /api/v1/parent/students/s1/assessments?academic_term_id=term-1      │
│    → Shows: { raw_score: 41, max_points: 50, performance_level: "EE" }     │
├─────────────────────────────────────────────────────────────────────────────┤
│ 7. Term-end report card aggregation                                        │
│    GET /api/v1/parent/students/s1/report-card?academic_term_id=term-1      │
│    → Collects all PUBLISHED levels → mode → tie-break → final level        │
│    → Maths: EE (from 5 assessments)                                        │
└─────────────────────────────────────────────────────────────────────────────┘
```

### A.2 Rubric Assessment — Full Lifecycle

```
┌─────────────────────────────────────────────────────────────────────────────┐
│ 1. Teacher creates RUBRIC session                                          │
│    POST /api/v1/assessments/sessions                                       │
│    → { name: "Science Practical 1", evaluation_method: "RUBRIC", ... }     │
│    → No max_points, no scale_profile_id                                    │
├─────────────────────────────────────────────────────────────────────────────┤
│ 2. Teacher assigns outcome grades (DRAFT)                                  │
│    POST /api/v1/assessments/sessions/sess-002/grades                       │
│    → { grades: [                                                           │
│        { student_id: "s1", indicator_id: "ind-001", awarded_level: "ME" }, │
│        { student_id: "s1", indicator_id: "ind-002", awarded_level: "AE" }, │
│        ... ] }                                                              │
├─────────────────────────────────────────────────────────────────────────────┤
│ 3. Submit → Approve (same as quantitative)                                 │
│    Note: No snapshot needed — levels are already assigned                  │
├─────────────────────────────────────────────────────────────────────────────┤
│ 4. Parent views: shows per-indicator levels                                │
│    → outcome_grades: [{ indicator: "ind-001", level: "ME" }, ...]          │
├─────────────────────────────────────────────────────────────────────────────┤
│ 5. Term aggregation: ALL outcome grades across ALL rubric sessions         │
│    are included in the mode calculation alongside quantitative levels       │
└─────────────────────────────────────────────────────────────────────────────┘
```

### A.3 Rejection & Resubmission Flow

```
┌─────────────────────────────────────────────────────────────────────────────┐
│ 1. Teacher submits → PENDING_APPROVAL                                      │
├─────────────────────────────────────────────────────────────────────────────┤
│ 2. Admin reviews and rejects with comment                                  │
│    POST /api/v1/assessments/sessions/sess-001/reject                       │
│    → { rejection_comment: "Please review Kamau's score — seems too high." }│
│    → Status: DRAFT, scores unlocked                                        │
├─────────────────────────────────────────────────────────────────────────────┤
│ 3. Teacher edits Kamau's score                                              │
│    POST /api/v1/assessments/sessions/sess-001/scores (idempotent)           │
├─────────────────────────────────────────────────────────────────────────────┤
│ 4. Teacher re-submits                                                      │
│    → Status: PENDING_APPROVAL again                                         │
├─────────────────────────────────────────────────────────────────────────────┤
│ 5. Admin approves → PUBLISHED                                              │
└─────────────────────────────────────────────────────────────────────────────┘
```

---

## Appendix B: Error Codes

| HTTP | Code                       | Meaning                                |
| ---- | -------------------------- | -------------------------------------- |
| 400  | `invalid_input`            | Missing or malformed fields            |
| 401  | `unauthorized`             | Authentication required                |
| 403  | `forbidden`                | Insufficient role                      |
| 403  | `term_finalised`           | Term is locked                         |
| 404  | `not_found`                | Resource not found                     |
| 409  | `conflict`                 | General conflict                       |
| 409  | `invalid_state_transition` | Wrong session status                   |
| 409  | `scores_exist`             | Scores already on session              |
| 409  | `scale_referenced`         | Profile in use by sessions             |
| 409  | `overlapping_ranges`       | Scale ranges overlap                   |
| 409  | `duplicate_level`          | Duplicate performance level in profile |
| 422  | `invalid_input`            | Unparseable request body               |

---

## Appendix C: Missing / Placeholder Features

| Feature                                     | Backend    | Frontend       | Priority |
| ------------------------------------------- | ---------- | -------------- | -------- |
| Term report generation service              | ❌ Missing | ❌ Missing     | High     |
| Term report publishing workflow             | ❌ Missing | ❌ Missing     | High     |
| Student report page content                 | ✅ Route   | ❌ Placeholder | High     |
| Term report detail content                  | ✅ Route   | ❌ Placeholder | High     |
| Bulk report export                          | ❌ Missing | ❌ Placeholder | Medium   |
| Weight config management UI                 | ✅ API     | ❌ Placeholder | Low      |
| Parent report card UI                       | ✅ API     | ❌ Not built   | High     |
| Score conversion (raw→rubric) at entry time | ❌ Partial | N/A            | Medium   |
| Asynq worker for async report generation    | ❌ Missing | N/A            | Medium   |
| CSV/PDF export for scores                   | ❌ Missing | ❌ Missing     | Low      |
