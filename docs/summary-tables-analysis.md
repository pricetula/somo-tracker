# SomoTracker Summary Tables — Analysis & Visualisation Guide

> **Author:** Platform Team  
> **Date:** June 2026  
> **Version:** 1.0.0  
> **Scope:** All 11 materialised summary tables across the student academic, behaviour, and teacher domains.

---

## Table of Contents

1. [Introduction](#1-introduction)
2. [Student Academic Summary Tables](#2-student-academic-summary-tables)
   - 2.1 — `attendance_term_summaries`
   - 2.2 — `class_daily_attendance_summaries`
   - 2.3 — `student_term_subject_summaries`
   - 2.4 — `student_term_overall_summaries`
   - 2.5 — `student_cohort_position_summaries`
   - 2.6 — `student_subject_strand_summaries`
   - 2.7 — `student_performance_projections`
3. [Student Behaviour Summary Tables](#3-student-behaviour-summary-tables)
   - 3.1 — `student_behavior_term_summaries`
4. [Teacher Summary Tables](#4-teacher-summary-tables)
   - 4.2 — `teacher_subject_performance_summaries`
   - 4.3 — `teacher_delivery_summaries`
   - 4.4 — `teacher_workload_summaries`
5. [Cross-Cutting Visualisation Patterns](#5-cross-cutting-visualisation-patterns)

---

## 1. Introduction

SomoTracker's analytics layer is built around **11 materialised summary tables**, each serving a distinct analytical grain. They fall into three categories:

| Category | Tables | Grain | Refresh Model |
|---|---|---|---|
| **Student Academic** | 7 tables | Student × Term × (Learning Area) | Incremental (triggers on PUBLISH) + Batch |
| **Student Behaviour** | 1 table | Student × Term | Incremental (triggers on INSERT/UPDATE) |
| **Teacher** | 3 tables | Teacher × (Term/Year) × (Subject/Class) | Batch-only (periodic or on-demand) |

This document explains each table's calculation logic in plain English with worked examples, then proposes visualisations.

---

## 2. Student Academic Summary Tables

### 2.1 `attendance_term_summaries`

**Grain:** `(student_id, academic_term_id, learning_area_id)`  
**Refresh:** Background task (nightly or on-demand)

#### How it works

For each student, each term, and each learning area (subject), this table counts the total number of instructional periods, then breaks them into present, absent, late, and excused counts. The **attendance_percentage** is:

```
attendance_percentage = periods_present / periods_total × 100
```

**Example:**  
A student in G4 has 120 Maths periods in Term 1. They attended 108, arrived late 4 times, were excused 2 times (medical), and were absent for 6 periods.

| Field | Value |
|---|---|
| periods_total | 120 |
| periods_present | 108 |
| periods_late | 4 |
| periods_excused | 2 |
| periods_absent | 6 |
| attendance_percentage | 90.00% |

Note: Only **non-break** instructional periods count (enforced by a trigger that rejects attendance records for break periods).

#### Visualisation Ideas

| Chart Type | What It Shows | User |
|---|---|---|
| **Gauge / Ring chart** | Single student's attendance % for current term | Parent, Teacher |
| **Stacked bar (weekly)** | Present / Late / Absent / Excused breakdown per week | Teacher, Admin |
| **Heatmap (day × slot)** | Attendance patterns by day of week + time slot | Teacher, Admin |
| **Line chart (over terms)** | Attendance trend across three terms | Parent, Headteacher |
| **Bar chart (subject comparison)** | Attendance % by learning area — reveals subject-specific truancy | Teacher, Admin |
| **Scatter plot** | Attendance % vs overall_mean_percentage — correlation check | Admin, Headteacher |

---

### 2.2 `class_daily_attendance_summaries`

**Grain:** `(class_id, date)`  
**Refresh:** Incremental (triggered when all attendance for a class-date is marked, or on timeout)

#### How it works

For each class on each school day, this table provides a single-row snapshot:

- `total_enrolled` — distinct students who have attendance records that day (note: derived from records, not enrollment status, because a mid-term suspension shouldn't retroactively erase earlier days)
- `present_count` / `absent_count` / `late_count` / `excused_count`
- `daily_attendance_rate` = `present_count / (present + absent + late + excused) × 100`

**Example:**

A class of 35 students on Monday:

| Group | Count |
|---|---|
| Present | 30 |
| Absent | 2 |
| Late | 2 |
| Excused | 1 |
| **Rate** | **85.71%** (30/35) |

#### Visualisation Ideas

| Chart Type | What It Shows | User |
|---|---|---|
| **Calendar heatmap** | Daily attendance rate over the term (green → red gradient) | Teacher, Admin |
| **Line chart (daily)** | Time-series of attendance rate, with school holidays | Teacher, Admin |
| **Bar chart (day-of-week)** | Average attendance by day — identifies "Friday slump" | Headteacher |
| **Sparklines (per class grid)** | Side-by-side mini trends for all classes in a grade | Admin |
| **Threshold alert badges** | Red badge when a class drops below 80% attendance | Class Teacher |
| **Week-over-week comparison** | Current week vs previous week vs same week last term | Admin |

---

### 2.3 `student_term_subject_summaries`

**Grain:** `(student_id, academic_term_id, learning_area_id)`  
**Refresh:** Incremental trigger on assessment session → PUBLISHED

#### How it works

This is a **blended rollup** of two assessment sources:

**Quantitative scores** (from `student_assessment_scores`):  
The `calculated_percentage` is used directly (e.g., a student scored 34/40 = 85.00%).

**Rubric outcome grades** (from `student_assessment_outcome_grades`):  
Each awarded level (EE/ME/AE/BE) is converted to a percentage via the `default_percentage_mapping` from `grading_scale_ranges`. If no mapping exists, a midpoint of the range is used.

Both sources are then **averaged together**:

```
average_percentage = AVG(resolved_pct) across all PUBLISHED sessions
                     for this student + term + learning area
```

**Flags:**
- `has_quantitative_data` — true if any QUANTITATIVE session contributed
- `has_rubric_data` — true if any RUBRIC session contributed
- `mapped_performance_level` — the CBC level (EE/ME/AE/BE) mapped from the average percentage via the active grading scale

**Example (blended):**

A student in G4 English has 3 PUBLISHED sessions:

| Session | Type | Raw Score | Rubric Level | Resolved % |
|---|---|---|---|---|
| Spelling Test | QUANTITATIVE | 18/20 | — | 90.00% |
| Oral Reading | RUBRIC | — | ME (default 65%) | 65.00% |
| Composition | RUBRIC | — | EE (default 90%) | 90.00% |

**Result:** Average = (90 + 65 + 90) / 3 = **81.67% → AE** (Approaching Expectation, assuming a typical scale where 80-89 = AE)

#### Visualisation Ideas

| Chart Type | What It Shows | User |
|---|---|---|
| **Radar / Spider chart** | Student's performance across all subjects | Parent, Teacher |
| **Bar chart (subject comparison)** | Average % per learning area side-by-side | Parent, Teacher |
| **Treemap** | Subject contribution to overall with colour by level | Parent |
| **Dot plot with threshold lines** | Each subject's score with ME/AE/EE threshold zones | Parent, Teacher |
| **Stacked bar (source composition)** | How much of each subject's score is quantitative vs rubric | Teacher, Admin |
| **Progress bar per subject** | Simple parent-friendly "you are here" visual | Parent |

---

### 2.4 `student_term_overall_summaries`

**Grain:** `(student_id, academic_term_id)`  
**Refresh:** On-demand (triggered by subject summary changes or batch)

#### How it works

This is a **second-level rollup** — it aggregates across all subject summaries for a student in one term.

**Non-weighted (regular terms):**
```
overall_mean_percentage = AVG(average_percentage) across all subjects
```

**Weighted (final exam terms — G6→KPSEA, G9→KJSEA, G12→KSSEA):**
```
overall_mean_percentage = Σ(subject_percentage × config_weight) / Σ(config_weight)
```

The weights come from `assessment_weight_configs` — these mirror official KNEC formulas:
- KPSEA: 60% SBA (G4+G5) + 40% KPSEA written
- KJSEA: 20% SBA (G7+G8) + 20% KPSEA result + 60% KJSEA written
- KSSEA: Similar CBAF-FL weighting

The `is_weighted_exam_score` flag makes transparent whether weighting was applied.

**Level counts:** The table also stores counts — `exceeding_count`, `meeting_count`, `approaching_count`, `below_count` — so you know how many subjects fall at each level.

**Example (non-weighted, Term 1):**

| Subject | Average % | Level |
|---|---|---|
| English | 81.67 | AE |
| Maths | 72.00 | AE |
| Kiswahili | 55.00 | BE |
| Science | 88.00 | ME |
| Social Studies | 91.00 | EE |

**Result:**
- Overall mean: (81.67 + 72 + 55 + 88 + 91) / 5 = **77.53% → AE**
- Level breakdown: EE=1, ME=1, AE=2, BE=1

#### Visualisation Ideas

| Chart Type | What It Shows | User |
|---|---|---|
| **Gauge with level zones** | Single overall % with EE/ME/AE/BE colour bands | Parent, Student |
| **Donut chart** | Proportion of subjects at each performance level | Parent, Teacher |
| **Horizontal bar (level distribution)** | Count of subjects per level — "at a glance" report card | Parent |
| **Weighted vs unweighted toggle** | Show or hide KNEC formula impact on the score | Parent, Admin |
| **Term-over-term comparison** | Side-by-side bar: Term 1 vs Term 2 vs Term 3 | Parent, Teacher, Headteacher |
| **Waterfall chart** | How each subject contributes to (or drags down) the overall | Teacher, Parent |
| **Headteacher remark badge** | A callout displaying the headteacher's remark | Parent |

---

### 2.5 `student_cohort_position_summaries`

**Grain:** `(student_id, class_id, academic_term_id)`  
**Refresh:** Batch-only (scheduled — never incremental)

#### How it works

This is the **ranking engine**. Using window functions (`RANK() OVER`), it computes:

- **Class rank:** position within the student's immediate class (1 = highest score)
- **Class percentile:** `(class_headcount - class_rank) / class_headcount × 100`
- **Grade rank:** position within the same grade across all streams
- **Grade percentile:** same formula but grade-wide
- **Class average / Grade average:** mean scores
- **Variance from grade mean:** `student_score - grade_average` (positive = above average)

**Example:**

Student Aisha scores 85% in G4. Her class has 40 students. The grade (all G4 streams) has 120 students.

| Metric | Value | Explanation |
|---|---|---|
| Class rank | 5th | 4 students scored higher |
| Class headcount | 40 | |
| Class percentile | 87.50 | (40-5)/40 × 100 — she's in the top 12.5% |
| Grade rank | 18th | 17 students across all streams scored higher |
| Grade headcount | 120 | |
| Grade percentile | 85.00 | (120-18)/120 × 100 |
| Class average | 62.30 | |
| Grade average | 60.10 | |
| Variance | +24.90 | Well above grade average |

#### Visualisation Ideas

| Chart Type | What It Shows | User |
|---|---|---|
| **Distribution curve (bell)** | Student's position highlighted on the grade distribution | Parent, Teacher, Admin |
| **Ranks over terms (line)** | How class rank has changed across terms — improving or slipping? | Parent, Teacher |
| **Bar chart (class vs grade)** | Side-by-side comparison of class rank vs grade rank | Parent |
| **Percentile gauge** | "You are in the top X% of your grade" — parent-friendly | Parent |
| **Scatter plot (class × score)** | All students in a class plotted — identifies outliers | Teacher |
| **Heatmap (stream comparison)** | All streams in a grade, with average scores and ranges | Admin, Headteacher |
| **Variance bar** | Green/red bar showing distance from grade mean | Parent, Teacher |
| **Top/Bottom N list** | Class top 10 and bottom 10 with scores | Teacher |

---

### 2.6 `student_subject_strand_summaries`

**Grain:** `(student_id, academic_term_id, sub_strand_id)`  
**Refresh:** Incremental trigger on RUBRIC session → PUBLISHED

#### How it works

This is a **rubric-only** summary at the **sub-strand** level (the finest grain in the curriculum hierarchy: Learning Area → Strand → Sub-Strand → Performance Indicator).

For each student + term + sub-strand, it counts how many performance indicators were awarded at each CBC level:

```
mastery_percentage = (exceeding_count + meeting_count) / total_count × 100
```

**requires_remediation = TRUE when:**
- Any indicator is Below Expectations, OR
- mastery_percentage < 50%

**has_data = TRUE** only when rubric outcome grades exist — pure-quantitative subjects leave this FALSE to avoid misleading 0%.

**Example:**

A student in G4 English, sub-strand "Reading Comprehension" has 5 performance indicators assessed:

| Indicator | Awarded Level |
|---|---|
| Identify main idea | ME |
| Make inferences | BE |
| Summarise passage | AE |
| Predict outcomes | ME |
| Analyse character | AE |

| Metric | Value |
|---|---|
| EE count | 0 |
| ME count | 2 |
| AE count | 2 |
| BE count | 1 |
| mastery_percentage | 40% (2+0)/5 |
| requires_remediation | TRUE (BE > 0 AND mastery < 50%) |
| mapped_performance_level | BE (falls in 0-49 range) |

#### Visualisation Ideas

| Chart Type | What It Shows | User |
|---|---|---|
| **Heatmap (sub-strand × level)** | Colour-coded grid of all sub-strands, one row per strand | Teacher, Admin |
| **Bar chart (mastery % per sub-strand)** | Green/red bars showing mastery per sub-strand | Teacher, Parent |
| **Treemap (hierarchical)** | Subject → Strand → Sub-strand drill-down with colour by level | Teacher, Admin, Parent |
| **Gauge per sub-strand** | Individual mastery gauges for each sub-strand | Parent, Teacher |
| **Flag/alert list** | Sub-strands requiring remediation, sorted by urgency | Teacher, Admin |
| **Radar (skill profile)** | Multi-axis radar comparing sub-strand mastery | Teacher, Parent |
| **Pie chart (level distribution)** | Proportion of indicators at each level for a sub-strand | Parent |
| **Before/after comparison** | Mastery change after remediation intervention | Teacher, Admin |

---

### 2.7 `student_performance_projections`

**Grain:** `(student_id, academic_term_id, learning_area_id)` — learning_area_id can be NULL for overall projection  
**Refresh:** Batch-only (scheduled — once per term close)

#### How it works

This table uses **simple linear regression** over the last 2-3 terms to predict future performance.

**Algorithm per student + learning area:**

1. Collect the last 2-3 terms of scores (including current term)
2. `x` = sequential term index (0, 1, 2), `y` = average_percentage
3. Compute **momentum_score** (the slope `m`):

```
m = (n·Σxy - Σx·Σy) / (n·Σxx - Σx·Σx)
```

4. **projected_score** = last_term_score + momentum_score (clamped to 0-100)
5. **target_gap_points** = projected_score - ME_threshold (negative = at risk)
6. **confidence_percentage** = 85% (3 terms) / 60% (2 terms) / 15% (<2 terms)
7. **risk_level**:
   - `High`: confidence < 30% OR gap < -15 points
   - `Medium`: gap < -5 points OR confidence < 60%
   - `Low`: otherwise

**Example:**

| Term | Term Index (x) | Score (y) | x² | xy |
|---|---|---|---|---|
| Term 1 (G4) | 0 | 62.00 | 0 | 0 |
| Term 2 (G4) | 1 | 70.00 | 1 | 70 |
| Term 3 (G4) | 2 | 75.00 | 4 | 150 |

n = 3, Σx = 3, Σy = 207, Σxy = 220, Σxx = 5

```
m = (3×220 - 3×207) / (3×5 - 3×3)
m = (660 - 621) / (15 - 9)
m = 39 / 6
m = 6.50
```

Last score = 75.00  
Projected (Term 1, G5) = 75.00 + 6.50 = **81.50% → AE**

ME threshold = 50% (default)  
Target gap = 81.50 - 50.00 = **+31.50** (comfortably above)  
Risk = **Low** (confidence 85%, gap is positive and large)

#### Visualisation Ideas

| Chart Type | What It Shows | User |
|---|---|---|
| **Scatter + trend line** | Historical scores plotted with regression line and projected point | Parent, Teacher, Admin |
| **"Report Card Forecast" card** | Next-term projected level with confidence badge | Parent |
| **Risk indicator (traffic light)** | Red/Amber/Green badge for each subject's projection | Teacher, Admin |
| **Gap bar chart** | Visual distance from ME threshold — positive = green, negative = red | Teacher, Parent |
| **Momentum arrow** | Up/Flat/Down arrow per subject showing trend direction | Parent |
| **Table with sparklines** | Per-subject mini trend charts in a dashboard table | Teacher |
| **Confidence badge** | "High confidence (3 terms of data)" or "Low confidence (new student)" | Teacher, Admin |
| **Comparison grid** | All subjects in one view: projected score + risk level + trend | Teacher, Admin, Parent |
| **Waterfall (from actual to projected)** | Shows the projected jump/decline from current term | Parent |

---

## 3. Student Behaviour Summary Tables

### 3.1 `student_behavior_term_summaries`

**Grain:** `(student_id, academic_term_id)`  
**Refresh:** Incremental trigger on INSERT/UPDATE of `behavior_notes`

#### How it works

For each student per term, this table summarises behaviour notes that have been **APPROVED** or **INCLUDED_IN_REPORT**:

- **total_incidents:** count of approved notes
- **urgent_count:** approved notes flagged as urgent
- **commendations_count:** approved notes whose category has `category_type = 'COMMENDATION'`
- **disciplinary_count:** approved notes whose category has `category_type = 'DISCIPLINARY'`
- **pending_review_count:** notes still in PENDING_REVIEW (admin visibility)
- **resolved_count:** notes with any terminal status
- **primary_category_id:** the category with the highest count (ties broken by most recent)

Categories are school-managed — a school might define: "Helping Others" (COMMENDATION), "Late to Class" (DISCIPLINARY), "Bullying" (DISCIPLINARY), "Class Participation" (COMMENDATION).

**Example:**

A student has these notes in Term 1:

| Note | Category | Type | Urgent? | Status |
|---|---|---|---|---|
| Helped classmate with maths | Helping Others | COMMENDATION | No | APPROVED |
| Late 3 times this week | Punctuality | DISCIPLINARY | No | APPROVED |
| Disruptive in class | Conduct | DISCIPLINARY | No | APPROVED |
| Won spelling bee | Academic Achievement | COMMENDATION | No | APPROVED |
| Caught fighting | Violence | DISCIPLINARY | Yes | APPROVED |

| Metric | Value |
|---|---|
| total_incidents | 5 |
| urgent_count | 1 |
| commendations_count | 2 |
| disciplinary_count | 3 |
| pending_review_count | 0 |
| resolved_count | 5 |
| primary_category | DISCIPLINARY (highest count: 3) |

#### Visualisation Ideas

| Chart Type | What It Shows | User |
|---|---|---|
| **Commendations vs Disciplinary (bar)** | Side-by-side comparison — quick moral temperature check | Parent, Teacher |
| **Stacked bar (urgent breakdown)** | Urgent vs non-urgent within disciplinary and commendation | Teacher, Admin |
| **Pie/donut chart** | Proportion of behaviour types overall | Parent, Admin |
| **Trend line over terms** | Commendation and disciplinary counts across terms | Parent, Headteacher |
| **Category breakdown (horizontal bar)** | Each behaviour category with its count | Teacher, Admin |
| **Alert badge** | "3 disciplinary incidents this term — urgent flag raised" | Parent, Admin |
| **Calendar heatmap** | When in the term do incidents cluster? | Teacher, Admin |
| **Class comparison (box plot)** | Distribution of disciplinary counts across the class | Teacher, Headteacher |
| **Net sentiment score** | commendations_count - disciplinary_count (positive = good) | Parent, Headteacher |

---

## 4. Teacher Summary Tables

### 4.1 `teacher_subject_performance_summaries`

**Grain:** `(user_id, learning_area_id, class_id, academic_term_id)`  
**Refresh:** Batch-only (periodic — once per term close)

#### How it works

For each teacher-subject-class-term combination, five metrics are computed:

**1. subject_mean_score** — Average of all students' `average_percentage` in this class+subject:

```
AVG(stss.average_percentage) for all students enrolled
```

**2. cohort_mastery_rate** — Percentage of students achieving ME or EE:

```
COUNT(students at ME or EE) / COUNT(all scored students) × 100
```

**3. student_growth_rate** — Average percentage-point change for students with data in both current and prior term:

```
AVG(current_term_avg% - prior_term_avg%) for matched students
```

**4. assessment_timeliness_index** — Percentage of published sessions that were published on or before their scheduled date

**5. strand_coverage_rate** — Of all strands in the learning area, how many had at least one PUBLISHED RUBRIC session:

```
COUNT(covered_strands) / COUNT(total_strands) × 100
```

**Example:**

Teacher Mr. Kamau teaches G4 Maths in class "Blue":

| Metric | Calculation | Value |
|---|---|---|
| subject_mean_score | 30 students, avg of 72.50 | **72.50%** |
| cohort_mastery_rate | 18 of 30 at ME+ | **60.00%** |
| student_growth_rate | 25 matched students, avg +3.2 pts | **+3.20** |
| assessment_timeliness | 8 of 10 sessions published on time | **80.00%** |
| strand_coverage | 4 of 5 strands assessed via rubric | **80.00%** |

#### Visualisation Ideas

| Chart Type | What It Shows | User |
|---|---|---|
| **Radar chart (5 metrics)** | Multi-axis performance profile for a teacher | Admin, Headteacher |
| **Dashboard card grid** | Each metric as a large KPI card with trend sparkline | Teacher, Admin |
| **Bar chart (teacher comparison)** | subject_mean_score across all teachers in a grade | Admin, Headteacher |
| **Scatter (mastery vs growth)** | Plots each teacher — identifies high-growth/high-mastery standouts | Admin |
| **Gauge per metric** | Individual gauges with target markers | Teacher, Admin |
| **Time-series (over terms)** | Each metric trended over 3 terms | Admin, Headteacher |
| **Strand coverage treemap** | Visual of which strands were/were not assessed | Teacher, Admin |
| **Class heatmap (subject × teacher)** | All teachers × subjects, coloured by mean score | Headteacher |

---

### 4.2 `teacher_delivery_summaries`

**Grain:** `(user_id, academic_term_id)`  
**Refresh:** Batch (can also be triggered incrementally)

#### How it works

For each teacher per term, this table tracks lesson delivery faithfulness:

- **total_assigned_slots:** Number of timetable slot occurrences the teacher was assigned (slot × weeks in term)
- **marked_slots:** Slot+date combinations where attendance was actually taken
- **missed_slots:** Slot+date combinations where the lesson was marked SKIPPED (teacher absent, assembly, etc.)
- **sessions_created / sessions_approved:** Counts of attendance session records
- **on_time_submission_rate:** `(marked_slots + missed_slots) / total_assigned_slots × 100`

**Example:**

Teacher Ms. Ochieng has 8 slots per week (one per weekday period). Term has 13 weeks:

| Metric | Calculation | Value |
|---|---|---|
| total_assigned_slots | 8 slots × 13 weeks | 104 |
| marked_slots | Attendance taken | 92 |
| missed_slots | Lesson skipped | 8 |
| on_time_submission_rate | (92+8)/104 | 96.15% |

The 4 unaccounted slots represent lessons held but attendance not yet recorded.

#### Visualisation Ideas

| Chart Type | What It Shows | User |
|---|---|---|
| **Gauge** | on_time_submission_rate with target line (e.g. 95%) | Teacher, Admin |
| **Stacked bar** | Marked vs Missed vs Unaccounted per week | Teacher, Admin |
| **Calendar / week grid** | Visual of which days had issues — patterns emerge | Teacher, Admin |
| **Trend line (weekly)** | Submission rate week-over-week — is it improving? | Admin |
| **Comparison bar (teacher × teacher)** | Side-by-side delivery rates across teaching staff | Headteacher |
| **Heatmap (day × slot)** | Which time slots are most commonly missed? | Admin |
| **Missed lesson reasons (pie)** | Breakdown of skip reasons (assembly, sick, training, etc.) | Admin |

---

### 4.3 `teacher_workload_summaries`

**Grain:** `(user_id, academic_year_id)`  
**Refresh:** Batch (on-demand — reassignments are infrequent)

#### How it works

For each teacher per academic year, this table measures workload:

- **total_assigned_periods:** Number of weekly timetable slots (e.g., 24 periods/week)
- **unique_subjects:** Count of distinct learning areas assigned
- **classes_taught:** Count of distinct classes
- **utilization_percentage:** `teacher_periods / total_school_periods × 100` — what share of all instructional periods this teacher covers
- **is_overcapacity:** True when `total_assigned_periods > total_school_periods / active_teachers` (simple heuristic)

**Example:**

School has 200 total weekly periods across all teachers (non-break slots). 25 active teachers.

Teacher Mr. Juma:

| Metric | Value | Notes |
|---|---|---|
| total_assigned_periods | 32 | Highest in the school |
| unique_subjects | 3 | Maths, Physics, Computer Science |
| classes_taught | 6 | |
| utilization_percentage | 16.00% | 32/200 — |
| is_overcapacity | TRUE | Avg is 200/25 = 8 periods; 32 > 8 |

#### Visualisation Ideas

| Chart Type | What It Shows | User |
|---|---|---|
| **Horizontal bar (all teachers)** | Workload comparison — longest bar = most loaded | Admin, Headteacher |
| **Histogram** | Distribution of periods/teacher — identifies outliers | Admin |
| **Gauge** | Utilization % with overcapacity threshold line | Teacher, Admin |
| **Bubble chart** | Periods vs subjects vs classes — size = utilization | Admin |
| **Treemap** | All teachers tiled by utilization, coloured by overcapacity | Headteacher |
| **Scatter (periods × subjects)** | Identifies teachers with too many subjects for too few periods | Admin |
| **Radar (per teacher)** | Workload dimensions: periods, subjects, classes | Teacher |
| **What-if slider** | Interactive: reassign 5 periods from Mr Juma → who can absorb? | Admin |

---

## 5. Cross-Cutting Visualisation Patterns

### 5.1 Dashboard Layers by User Role

| Role | Priority Tables | Suggested Dashboard |
|---|---|---|
| **Parent** | overall_summaries, subject_summaries, cohort_positions, behavior_summaries | Single-child report card: gauge + radar + trend + behaviour summary |
| **Student** | subject_summaries, strand_summaries, projections | Self-improvement dashboard: mastery heatmap + projection arrow + remediation list |
| **Class Teacher** | subject_summaries, strand_summaries, attendance (daily), behavior_summaries | Class overview: student grid (scores × subjects) + attendance alert + remediation needs |
| **Subject Teacher** | teacher_performance_summaries, teacher_delivery_summaries, subject_summaries | My subjects: mastery rates + timeliness + strand coverage + student-level drill-down |
| **School Admin** | ALL summary tables | Cross-cutting: class comparisons, teacher rankings, behaviour trends, grade distributions |
| **Headteacher** | overall_summaries (aggregated), cohort_positions, teacher_workload, attendance | School leadership: grade-wide KPIs, top/bottom performers, workload balance, exam readiness |

### 5.2 Reusable Widget Patterns

These charts appear across multiple contexts and could be built as reusable components:

1. **Level Distribution Donut** — Shows EE/ME/AE/BE proportions (used in overall, subject, and strand summaries)
2. **Trend Sparkline** — Tiny line chart for term-over-term score changes (embedded in tables, cards, headers)
3. **Traffic Light Indicator** — Red/Amber/Green for risk flags, attendance thresholds, remediation needs
4. **Score Gauge with Threshold Bands** — Score needle on a coloured continuum (EE green, ME blue, AE yellow, BE red)
5. **Hierarchical Drill-Down** — Subject → Strand → Sub-Strand → Indicator breadcrumb navigation

### 5.3 Refresh & Real-Time Indicators

Since some tables are batch (minutes/hours stale) and others are trigger-incremental (near real-time), every visualisation should display a **"Last refreshed"** timestamp and a staleness indicator:

- **Green badge:** Refreshed within the last hour
- **Amber badge:** Refreshed within the last 24 hours
- **Red badge:** Stale > 24 hours

Tables that are batch-only (cohort_positions, projections, teacher_performance, teacher_workload) should have a prominent **"Refresh now"** button for admins.

### 5.4 Data Export & Reporting

Every summary table can feed into:

- **Term-end report cards** (PDF) — overall_summaries + subject_summaries + behavior_summaries
- **KNEC SBA upload** — subject_summaries aggregated by class
- **TSC teacher appraisal** — teacher_performance_summaries + teacher_delivery_summaries
- **MoE quarterly reporting** — attendance_summaries + overall_summaries (aggregated)
- **Parent-teacher conference pack** — subject_strand_summaries + cohort_positions + projections
