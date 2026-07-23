# Summary Tables Implementation — Gap Audit

> **Date:** July 2026  
> **Audit of:** `src/features/analytics/` against `docs/summary-tables-analysis.md`  
> **Status:** Sections 2 & 3 complete. Section 4 partially done.

---

## ✅ Complete — Sections 2 & 3

All visualisations for the following tables have been implemented (68 component files, zero TypeScript errors):

| Section | Table | Components |
|---------|-------|-----------|
| 2.1 | `attendance_term_summaries` | 6 ✓ |
| 2.2 | `class_daily_attendance_summaries` | 6 ✓ |
| 2.3 | `student_term_subject_summaries` | 6 ✓ |
| 2.4 | `student_term_overall_summaries` | 7 ✓ |
| 2.5 | `student_cohort_position_summaries` | 8 ✓ |
| 2.6 | `student_subject_strand_summaries` | 8 ✓ |
| 2.7 | `student_performance_projections` | 9 ✓ |
| 3.1 | `student_behavior_term_summaries` | 9 ✓ |

---

## ❌ Missing — Section 4 (Teacher Tables)

### 4.1 `teacher_subject_performance_summaries` — 4 missing

| # | Visualisation | Notes |
|---|---------------|-------|
| 1 | **Gauge per metric** | 5 individual gauges (subject_mean_score, cohort_mastery_rate, student_growth_rate, assessment_timeliness, strand_coverage) with target markers |
| 2 | **Time-series (over terms)** | Each of the 5 metrics trended over 3 terms as a multi-line chart |
| 3 | **Strand coverage treemap** | Treemap showing which strands were/were not assessed |
| 4 | **Class heatmap (subject × teacher)** | Grid of teachers × subjects, cell colour = mean score |

**Already built (4):** `TeacherPerformanceRadar`, `TeacherKpiCards`, `TeacherComparisonBar`, `TeacherMasteryGrowthScatter`

---

### 4.2 `teacher_delivery_summaries` — 4 missing

| # | Visualisation | Notes |
|---|---------------|-------|
| 1 | **Calendar / week grid** | Visual grid of which days had issues (sick, assembly, training, etc.) |
| 2 | **Trend line (weekly)** | on_time_submission_rate week-over-week line chart |
| 3 | **Heatmap (day × slot)** | Which time slots (Monday Period 1, etc.) are most commonly missed |
| 4 | **Missed reasons pie** | Breakdown of skip reasons (assembly, sick, training, etc.) as a pie/donut |

**Already built (3):** `DeliveryGauge`, `DeliveryWeeklyStackedBar`, `DeliveryComparisonBar`

---

### 4.3 `teacher_workload_summaries` — 6 missing

| # | Visualisation | Notes |
|---|---------------|-------|
| 1 | **Histogram** | Distribution of periods/teacher — identifies outliers |
| 2 | **Bubble chart** | Periods (x) vs subjects (y) vs classes (bubble size) — area = utilization |
| 3 | **Treemap** | All teachers tiled by utilization %, coloured by is_overcapacity |
| 4 | **Scatter (periods × subjects)** | Identifies teachers with too many subjects for too few periods |
| 5 | **Radar (per teacher)** | Multi-axis: total_assigned_periods, unique_subjects, classes_taught, utilization, overcapacity flag |
| 6 | **What-if slider** | Interactive: reassign N periods from teacher A → show who can absorb |

**Already built (2):** `WorkloadComparisonBar`, `WorkloadUtilizationGauge`

---

## Total Gap

| Domain | Missing |
|--------|---------|
| Section 4.1 — Teacher Performance | 4 components |
| Section 4.2 — Teacher Delivery | 4 components |
| Section 4.3 — Teacher Workload | 6 components |
| **Total** | **14 components** |

All missing components follow the established patterns:
- Location: `src/features/analytics/components/{section}/`
- Stack: shadcn `ChartContainer` + recharts, flat layout, semantic CSS vars only
- Each includes a typed `Props` interface, skeleton loading state, and empty state
