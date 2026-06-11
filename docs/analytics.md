# Somotracker Architecture: Analytics Engine Specifications

This document defines the automated aggregation metrics, database computation formulas, and read-optimized snapshot tables for Somotracker’s academic, administrative, and welfare analytical layers. Financial performance analytics are excluded from this engine to prioritize internal educational velocity and operational tracking.

---

## 📈 Part 1: Operational & Progression Analytics Map

### 1. The Institutional Health Ribbon
* **Who it is for:** School Principals and Headteachers.
* **How it benefits them:** Provides a morning diagnostic tool to instantly assess campus attendance, operational risk, and teacher grading compliance within seconds of logging in.
* **How it looks:** A high-level KPI card row at the top of the main admin dashboard displaying three bold metrics: Daily Attendance %, Total Outstanding Arrears, and Term Mark Completion %.

### 2. School-Wide Absenteeism Heatmap
* **Who it is for:** Principals and Deputy Headteachers.
* **How it benefits them:** Highlights chronic attendance leaks by specific weekdays or classroom cohorts, enabling early intervention before students drop out or fall behind academically.
* **How it looks:** A color-coded calendar grid shading from cool blue (100% attendance) to deep crimson (high absenteeism) mapped across class sections and days of the week.

### 3. Teacher Grading Velocity Tracker
* **Who it is for:** Principals and Academic Directors.
* **How it benefits them:** Minimizes micro-management by identifying which faculty members have compiled their assessments and who needs a reminder before report-card deadlines.
* **How it looks:** A progress bar list where each row shows a teacher's name, assigned subject, and a percentage completion bar matching expected assessments for the active term.

### 4. The Classroom Subject Performance Matrix
* **Who it is for:** Subject Teachers, Class Teachers, and Academic Administrators.
* **How it benefits them:** Evaluates teaching efficacy by showing topical mastery immediately after an assessment, highlighting whether a concept needs to be re-taught.
* **How it looks:** For traditional tracks, a bell-curve or distribution bar showing class averages. For CBC tracks, a stacked-bar chart showing the exact percentage breakdown of students sitting in EE, ME, AE, or BE.

### 5. Topical Skill Performance Comparison
* **Who it is for:** Subject Teachers.
* **How it benefits them:** Displays historical performance trends across different topics or curriculum strands to reveal structural class weaknesses (e.g., strong in Geometry, weak in Algebra).
* **How it looks:** A multi-line or side-by-side bar chart grouping student assessment averages by curriculum strands across the active term.

### 6. Sickbay Visit Velocity Index
* **Who it is for:** School Nurses and Student Welfare Officers.
* **How it benefits them:** Tracks spikes in clinic visits to act as an early warning system for seasonal outbreaks or identify localized environmental hazards on campus.
* **How it looks:** A weekly volume chart showing logged medical incidents set against a dotted line representing the historical campus baseline average for that month.

### 7. Term-Matching Subject Mean Progression (Year-over-Year)
* **Who it is for:** School Principals, Academic Directors, and Department Heads.
* **How it benefits them:** Quantifies the direct educational return on investments (e.g., new textbooks or teaching tools) by comparing a cohort's performance this term directly against how that same cohort performed in the exact same term last year.
* **How it looks:** A dual-line chart where the horizontal axis maps Term 1, Term 2, and Term 3. A solid line plots the current academic year, while a dotted line tracks the previous year's parallel terms.

### 8. CBC Competency Level Shifts (Year-over-Year)
* **Who it is for:** CBC Class Teachers and Headteachers.
* **How it benefits them:** Evaluates the closing of skill gaps over time by demonstrating whether the percentage of struggling students (Below Expectations) has shrunk compared to where those exact same children stood a year ago.
* **How it looks:** A side-by-side grouped vertical bar chart for each term, pairing the previous year's distribution against the current year's distribution for the identical student cohort.

### 9. The Student Longitudinal Subject Arc (Year-over-Year)
* **Who it is for:** Parents, Class Teachers, and Guidance Counselors.
* **How it benefits them:** Flags individual student burnout or hidden learning hurdles early by showing if a child’s performance has dropped below their own historical baseline from the prior grade level.
* **How it looks:** A multi-tab line matrix on the student profile page. Selecting a subject plots two overlapping lines comparing the terms of the current calendar year directly against the child’s historical performance line.

### 10. Curriculum Strand Mastery Delta (Year-over-Year)
* **Who it is for:** Subject Teachers and Parents.
* **How it benefits them:** Transforms parent-teacher conferences from vague generalizations into precise, data-driven conversations about specific learning milestones.
* **How it looks:** A horizontal multi-bar "Delta Chart" showing core curriculum themes, with green bars pointing right for positive growth compared to last year's parallel term, and red bars pointing left for regressions.

---

## 📐 Part 2: Relational Schema Calculations

To calculate these metrics without choking production runtime, the analytics engine executes queries against the transactional tables established in `somotracker_schema_specifications.md`. Below are the underlying mathematical formulas and database processing rules:

### A. Daily Attendance Rate Formula
* **Involved Tables:** `attendance_logs`, `attendance_periods`.
* **Calculation:**
    $$\text{Attendance Rate} = \left( \frac{\text{COUNT}(\text{attendance\_logs.id}) \text{ WHERE status} = \text{'PRESENT'}}{\text{COUNT}(\text{attendance\_logs.id}) \text{ WHERE expected}} \right) \times 100$$
* **Execution Rule:** Filter `attendance_periods` by the current system date. Count the corresponding child state logs, ignoring any system-flagged administrative slots.

### B. Total Active Arrears Formula
* **Involved Tables:** `student_invoices`.
* **Calculation:**
    $$\text{Total Arrears} = \sum (\text{amount\_due} - \text{amount\_paid})$$
* **Execution Rule:** Scan `student_invoices` across the current `academic_term_id`, filtering strictly where `amount_paid < amount_due`.

### C. Term Mark Completion Percentage
* **Involved Tables:** `summative_assessments`, `subjects`.
* **Calculation:**
    $$\text{Completion \%} = \left( \frac{\text{COUNT}(\text{DISTINCT subject\_id IN } \text{summative\_assessments})}{\text{COUNT}(\text{id IN } \text{subjects WHERE active\_for\_grade})} \right) \times 100$$

### D. Multi-Curriculum Performance Aggregations
* **Involved Tables:** `summative_scores`, `task_evaluations`, `education_systems`.
* **Traditional Class Mean (IGCSE):**
    $$\text{Class Mean} = \frac{\sum (\text{summative\_scores.raw\_score})}{\text{COUNT}(\text{summative\_scores.id})}$$
* **CBC Competency Distribution Index:**
    $$\text{Distribution \% per Rating} = \left( \frac{\text{COUNT}(\text{task\_evaluations.id WHERE rating} = \text{TARGET\_ENUM})}{\text{TOTAL COUNT}(\text{task\_evaluations.id})} \right) \times 100$$
    *Where TARGET_ENUM represents `EE`, `ME`, `AE`, or `BE`*.

---

## 🗄️ Part 3: Read-Optimized Snapshot Tables

To bypass heavy multi-table table joins during dashboard rendering, the following specialized analytics tables cache pre-aggregated calculations.

### 1. Attendance Snapshot Ledger (`analytics_attendance_snapshots`)
Flattens live attendance logs into daily and termly summaries per class section to hydrate heatmaps instantly.

| Field Name | Data Type | Nullable | Domain Rules & Database Constraints |
| --- | --- | --- | --- |
| `id` | `UUID` | No | Primary key. |
| `class_id` | `UUID` | No | Foreign key linking to the classroom cohort. |
| `academic_term_id` | `UUID` | No | Filters evaluation boundaries per term. |
| `log_date` | `DATE` | No | The operational calendar date tracked. |
| `present_count` | `INTEGER` | No | Total students marked present on this date. |
| `absent_count` | `INTEGER` | No | Total students marked absent on this date. |
| `attendance_percentage`| `NUMERIC(5,2)` | No | Calculated daily percentage (`present / total`). |
| `last_calculated_at` | `TIMESTAMP` | No | Processing sync execution window. |
| *Constraint* | *Composite* | *No* | `UNIQUE(class_id, log_date)` to ensure one row per day. |

### 2. Subject Performance Snapshots (`analytics_subject_performance_snapshots`)
Stores pre-calculated class averages and CBC rating percentages. This table contains self-referencing tracking fields to compute year-over-year progress loops within a single read query.

| Field Name | Data Type | Nullable | Domain Rules & Database Constraints |
| --- | --- | --- | --- |
| `id` | `UUID` | No | Primary key. |
| `class_id` | `UUID` | No | Foreign key linking to the classroom cohort. |
| `subject_id` | `UUID` | No | Foreign key linking to the evaluated subject. |
| `academic_term_id` | `UUID` | No | Foreign key defining the tracking term context. |
| `traditional_mean_score`| `NUMERIC(5,2)` | Yes | Average score percentage. NULL if CBC track. |
| `cbc_ee_percentage` | `NUMERIC(5,2)` | Yes | Pre-calculated % of Exceeding Expectations. |
| `cbc_me_percentage` | `NUMERIC(5,2)` | Yes | Pre-calculated % of Meeting Expectations. |
| `cbc_ae_percentage` | `NUMERIC(5,2)` | Yes | Pre-calculated % of Approaching Expectations. |
| `cbc_be_percentage` | `NUMERIC(5,2)` | Yes | Pre-calculated % of Below Expectations. |
| `prior_year_mean_delta`| `NUMERIC(5,2)` | Yes | Current mean minus previous year's parallel term mean. |
| `last_calculated_at` | `TIMESTAMP` | No | Timestamp tracking computation velocity. |
| *Constraint* | *Composite* | *No* | `UNIQUE(class_id, subject_id, academic_term_id)`. |

### 3. Individual Student Growth Snapshots (`analytics_student_longitudinal_snapshots`)
Caches term-by-term subject results for single students, allowing immediate rendering of individual longitudinal arcs and parent progress gauges.

| Field Name | Data Type | Nullable | Domain Rules & Database Constraints |
| --- | --- | --- | --- |
| `id` | `UUID` | No | Primary key. |
| `student_id` | `UUID` | No | Foreign key linking to the unique student profile. |
| `subject_id` | `UUID` | No | Foreign key linking to the target subject catalog. |
| `academic_term_id` | `UUID` | No | Foreign key defining the current term context. |
| `calculated_score` | `NUMERIC(5,2)` | Yes | Numeric grade for traditional tracks. NULL if CBC. |
| `calculated_cbc_rating`| `VARCHAR(2)` | Yes | Derived categorical tier (`EE`,`ME`,`AE`,`BE`). NULL if IGCSE. |
| `yoy_progress_status` | `VARCHAR(12)` | No | Status string enum: `GROWTH`, `STABLE`, `REGRESSION`. |
| `last_calculated_at` | `TIMESTAMP` | No | Logged processing execution timestamp. |
| *Constraint* | *Composite* | *No* | `UNIQUE(student_id, subject_id, academic_term_id)`. |

### 4. Clinical Welfare Volume Snapshots (`analytics_welfare_snapshots`)
Aggregates medical visit data to populate the Sickbay Velocity Index without processing historical patient visit logs on the fly.

| Field Name | Data Type | Nullable | Domain Rules & Database Constraints |
| --- | --- | --- | --- |
| `id` | `UUID` | No | Primary key. |
| `school_id` | `UUID` | No | Foreign key mapping to the physical campus. |
| `week_start_date` | `DATE` | No | Calendar date marking the beginning of the tracked week. |
| `total_visit_count` | `INTEGER` | No | Total number of recorded clinic visits during this week. |
| `historical_baseline` | `NUMERIC(5,2)` | No | Averaged visit volume for this calendar week over past 3 years. |
| `last_calculated_at` | `TIMESTAMP` | No | Operational background task sync block. |
| *Constraint* | *Composite* | *No* | `UNIQUE(school_id, week_start_date)`. |

---

## 🛡️ Part 4: Data Pipeline & Synchronization Guardrails

### 1. Event-Driven Cache Invalidation
The backend runtime must listen for successful write operations targeting transactional tables (`attendance_logs`, `summative_scores`, `task_evaluations`, and medical records). Upon transaction commit, a lightweight background job must execute immediately to recalculate the matching row inside the corresponding analytics snapshot table.

### 2. Historical Term Immutability Lock
Once an `academic_term_id` transitions from `is_current = true` to `is_current = false`, the analytics pipeline flags all rows linked to that term as frozen. Background event triggers are blocked from running mutations on historical terms, enforcing absolute data immutability for past academic periods.

### 3. The Midnight Reconciliation Loop
To protect dashboards against missing event triggers or race conditions, an automated system cron worker runs daily at midnight. This worker re-aggregates raw transactional rows from the preceding 48 hours and overwrites the cached values inside the analytical snapshot tables to repair any daytime sync drift.