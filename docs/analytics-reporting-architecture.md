# Analytics, Reports & Quick Views — Architecture Analysis

> **Scope:** Role-based analytics, term reports, role-specific quick views (dashboards).
> **Date:** Generated from schema review — code-free discussion phase.
> **Status:** Analysis only — no implementation decisions taken yet.

---

## 1. Schema Entity Map

### Core Entities (cross-cutting)

```
tenants ────────┬── users ──────── memberships ──── cbc_schools
                │                                    │
                └── cbc_students ── cbc_student_enrollments ── cbc_classes
                                         │                           │
                                    cbc_student_parents         cbc_streams
                                         │
                                    cbc_parents
```

### Curriculum & Assessment Chain

```
cbc_learning_areas → cbc_strands → cbc_sub_strands → performance_indicators
        │                                                       │
assessment_sessions ── student_assessment_scores ── student_assessment_outcome_grades
    │                                 (quantitative)              (rubric)
    └── grading_scale_profiles ── grading_scale_ranges
    └── assessment_weight_configs
```

### Attendance Chain

```
timetable_structures ── cbc_timetable_slots
                              │
                  attendance_records ──── cbc_attendance_sessions
                              │
                  attendance_term_summaries (materialised rollup)
```

### Behavior Chain

```
behavior_categories ── behavior_notes
```

### Financial Chain

```
fee_categories ── fee_templates ── invoices ── invoice_items ── payments
```

### Denormalised Counts (already exist)

`schoool_member_counts` — admins, teachers, nurses, finance, parents, students

---

## 2. Cross-Domain Relationship Graph

```
student ──┬── enrollments ── class ── timetable_slots ── attendance_records
           │                              │
           ├── assessment_scores ─── session ── learning_area
           │                              │
           ├── outcome_grades ───── session ── performance_indicator
           │
           ├── behavior_notes ───── category
           │
           ├── invoices ─── items ─── fee_category
           │      └── payments
           │
           └── health_profile ─── medical_incidents
```

---

## 3. Role-Based Analytics Needs

### 3.1 🛡️ SCHOOL_ADMIN / SYSTEM_ADMIN — School Pulse & Operations

| Need | Data Sources |
|---|---|
| **Dashboard overview** — counts of students, teachers, classes, streams this term | `school_member_counts`, `cbc_student_enrollments` (filtered `ACTIVE`), `cbc_classes` |
| **Enrollment trends** — new enrollments vs transfers out per term/year | `cbc_student_enrollments` grouped by `status`, `academic_term_id` |
| **Assessment completion rates** — % of sessions published vs still in DRAFT | `assessment_sessions` grouped by `status`, `class_id` |
| **Behavior summary** — # of incidents, approval rate, urgent flags | `behavior_notes` grouped by `status`, `is_urgent` |
| **Attendance overview** — average attendance % across school, lowest-attendance classes | `attendance_term_summaries` aggregated |
| **Financial snapshot** — total invoiced, collected, outstanding % | `invoices` aggregated by `payment_status` |
| **Teacher workload** — # classes/subjects per teacher, timetable gaps | `cbc_class_teachers`, `cbc_timetable_slots` |
| **Term report generation** — which classes have published reports, trigger compilation | `assessment_sessions` (published), `attendance_term_summaries` |
| **Bulk export** — generate PDF/CSV for entire grade/class | All assessment + attendance + behavior data per student |

### 3.2 👩‍🏫 TEACHER — Per-Class Operational View

| Need | Data Sources |
|---|---|
| **My classes today** — timetable for today, quick attendance entry | `cbc_timetable_slots` filtered by `teacher_id` + current date |
| **Class attendance** — who's present/absent for each of my slots today | `attendance_records` per slot |
| **My assessment sessions** — draft count, pending approval, published | `assessment_sessions` filtered by `created_by` |
| **Grade entry** — quick-access to the grading grid for a session | `student_assessment_scores` or `student_assessment_outcome_grades` |
| **Class performance summary** — avg scores per learning area, distribution (EE/ME/AE/BE) | `student_assessment_scores` → `calculated_percentage` + `final_performance_level` |
| **Behavior logging** — quick incident entry for today's class | `behavior_notes` |
| **Pending reviews** — sessions I've submitted but not yet approved | `assessment_sessions` where `status='PENDING_APPROVAL'` and `submitted_by` = me |
| **Student report card preview** — what a compiled term report looks like for my class | All per-student aggregates |
| **My flagged urgent behaviors** — approved urgent notes I authored | `behavior_notes` where `is_urgent=true` and `authored_by_id` = me |

### 3.3 💰 FINANCE — Fiscal Health & Reconciliation

| Need | Data Sources |
|---|---|
| **Fee collection dashboard** — total invoiced vs collected this term, by grade level | `invoices` + `payments` grouped by `grade_level` via `cbc_student_enrollments` |
| **Outstanding balances** — list of unpaid/partial invoices, total AR | `invoices` where `payment_status IN ('UNPAID','PARTIAL')` |
| **Payment reconciliation** — M-Pesa/Bank reference codes, date range search | `payments` with `reference_code`, `payment_method` |
| **Daily/Weekly collections** — payment volume and trends | `payments` grouped by date |
| **Waived amounts** — total fees waived this term, by reason | `invoices` where `payment_status='WAIVED'` |
| **Per-student ledger** — full invoice + payment history for a student | `invoices` + `invoice_items` + `payments` per `student_id` |
| **Grade-level revenue projection** — expected revenue vs actual per grade | `fee_templates` × `cbc_student_enrollments` (active) vs actual collections |
| **Parent payment history** — which parents are consistently late | `payments` by `parent_id` with timestamps |
| **Bulk invoice generation status** — how many invoices generated this term | `invoices` grouped by term |

### 3.4 👨‍👩‍👧 PARENT — Child Progress & Finance

| Need | Data Sources |
|---|---|
| **My children's profiles** — list of linked students | `cbc_student_parents` where `parent_id` = me |
| **Published assessment results** — per-learning-area scores and performance levels | `ParentAssessmentView` (published sessions only) |
| **Term report card** — compiled grades across all learning areas | `StudentTermGrade` from `GetStudentTermGrades()` |
| **Attendance record** — attendance % per term, recent absences | `attendance_term_summaries` or `attendance_records` for current term |
| **Behavior notes** — approved/reviewed behavior notes for my child | `behavior_notes` where `status='APPROVED'` or `INCLUDED_IN_REPORT` |
| **Fee/invoice status** — current balance, payment history | `invoices` per student where parent is primary, `payments` |
| **Timetable** — my child's weekly schedule | `cbc_timetable_slots` via class enrollment |
| **Health profile & incidents** — allergies, logged incidents | `student_health_profiles`, `medical_incidents` |

---

## 4. Critical Gaps Identified

### 4.1 No dedicated analytics/reports backend domain

- The frontend has stub pages at `frontend/src/app/(dashboard)/reports/`:
  - `/reports/page.tsx` — role-gated entry (admin vs parent)
  - `/reports/terms/[term_id]/page.tsx` — term detail
  - `/reports/student/[id]/page.tsx` — single student report
  - `/reports/bulk-export/page.tsx` — bulk export placeholder
- **No** `internal/reports/` or `internal/analytics/` package exists in the backend.
- The only aggregation endpoints available:
  - `GetStudentTermGrades()` in assessments (compiled grades per learning area)
  - `GetPublishedSessionsForParent()` (parent-facing assessment view)
  - `school_member_counts` (materialised via triggers)

### 4.2 No materialised reporting tables

Only two materialised/denormalised aggregates exist:
- `attendance_term_summaries` — per student/term/learning area attendance rollup
- `school_member_counts` — school-wide staff/student headcount

Missing materialisations include:
- `term_report_snapshots` — frozen report card per student per term
- `class_assessment_summaries` — per-class performance distribution
- `behavior_term_summaries` — incident counts per student per term
- `fee_collection_summaries` — per-grade/class collection rates

### 4.3 No real-time quick-view system

- No "Teacher My Day" dashboard API endpoint
- No "School Pulse" admin overview endpoint
- No role-specific aggregated statistics endpoint

### 4.4 No export/report generation infrastructure

- No PDF generation service
- No KNEC-compliant report card template
- No CSV/XLSX bulk export endpoint
- No async job queue for report compilation

### 4.5 No role-specific dashboard API

- Frontend has `features/dashboard/components/` but no matching backend endpoint
- No per-role aggregation query exists

---

## 5. Existing Aggregation Infrastructure (What We Have)

### 5.1 Backend Domain Interfaces with Aggregation Methods

| Domain | Method | Returns |
|---|---|---|
| `assessments.Repository` | `GetStudentTermGrades()` | `[]StudentTermGrade` — compiled grades per learning area |
| `assessments.Repository` | `GetPublishedSessionsForParent()` | `[]ParentAssessmentView` — published assessments per student/term |
| `assessments.Repository` | `CountSessionsReferencingScale()` | `int` — how many sessions use a grading scale |

### 5.2 Trigger-Driven Denormalised Tables

| Table | Populated By | Contains |
|---|---|---|
| `school_member_counts` | `trg_memberships_counts_*` + `trg_cbc_students_counts_*` | Admins, teachers, nurses, finance, parents, students per school |
| `attendance_term_summaries` | Background task (nightly or on-demand) | Per-student-per-term-per-learning-area attendance rollup |

---

## 6. Suggested Implementation Phases (for discussion)

| Phase | Focus | Key Deliverables |
|---|---|---|
| **1** | Backend reports/analytics domain | New `internal/reports/` package with aggregation queries |
| **2** | Dashboard quick-views per role | Role-gated API endpoints: admin pulse, teacher my-day, finance overview, parent portal |
| **3** | Term report compilation + snapshot | `term_report_snapshots` table, background compilation job, report card view |
| **4** | Export infrastructure | PDF generation, CSV/XLSX bulk download, async job queue |
| **5** | Materialised views | Performance-optimised aggregates for common query patterns |

---

## 7. User Role Enum (Defined in Schema)

```sql
CREATE TYPE user_role AS ENUM (
    'SYSTEM_ADMIN',
    'SCHOOL_ADMIN',
    'TEACHER',
    'NURSE',
    'FINANCE',
    'PARENT'
);
```

> **Note:** `NURSE` role is defined but has no dedicated analytics needs in this document. Health-specific views (incident logs, health profile management) are separate from academic/financial reporting.
