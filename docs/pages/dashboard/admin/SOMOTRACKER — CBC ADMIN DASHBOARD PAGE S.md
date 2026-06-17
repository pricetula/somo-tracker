# SOMOTRACKER — CBC ADMIN DASHBOARD PAGE STRUCTURE DOCUMENT

Focus: High ROI, Time-Saving, Minimal Effective Design Interface
Curriculum Context: Kenya Competency Based Curriculum (CBC)
===============================================================================

## 1. TOP SECTION: GLOBAL CONTEXT & HEALTH VITALS

---

Layout: Horizontal ribbon on desktop (large screens) / Stacked elements on mobile (small screens)
Intent: Instant anchoring of administrative timeframe and mission-critical student safety details.

Features:

### A Current Term Banner

UI to show current academic year, academic period and progress in the academic period. Below is how we handle those specific visual states and fallback behaviors cleanly using standard UI patterns:

#### A1. The Standard State (Everything is Active)

A clean stacked UI with:-

- [academic period name] / [academic year name] like Term 2 / 2026 Academic Year
- **Progress Element:** A thin shadcn Progress component directly underneath the text. It computes the percentage of time elapsed from the term's start date to its end date relative to the current date.

### A2. The Edge Case States (Missing Data Elements)

We break this down into three distinct visual scenarios based on what data is missing:

        #### Scenario A: System Has No Configurations At All (Fresh Install)

        - **Visual Layout:** The text reads **System Initialization Required**.
        - **Behavior:** A prominent amber warning icon appears next to the text. Hovering or clicking the component triggers a shadcn Hover Card or Popover.
        - **Popover Content:** _"No Academic Year or Term data exists in the system. You cannot register students, log attendance, or issue invoices without a time anchor."_ It contains a high-contrast action button: **\[Configure Academic Year\]** which links directly to the setup document page.

        #### Scenario B: Years Exist, but No Current Period (e.g., School Break)

        - **Visual Layout:** The text reads **2026 Academic Year / No Active Term**.
        - **Progress Element:** The progress bar shifts to a neutral, solid gray background state (0% filled).
        - **Behavior:** The warning icon appears. The popover states: _"The current date falls outside of any configured Term dates. Data entry is restricted."_ It provides a direct link: **\[Create New Term Period\]**.

#### Scenario C: Current Date Out of Range Entirely

- **Visual Layout:** The text reads **Calendar Date Out of Range**.
- **Progress Element:** The progress bar is hidden entirely.
- **Behavior:** The warning icon appears. The popover states: _"The system clock detects today's date is outside your active structural calendar boundaries."_ It provides a link to the calendar management documentation to adjust the school's active lifecycle ranges.

* Grade Sequence Mapping Link
  - Small, clean text link next to the banner leading to the master lookup.
  - Shows the logical order of grades to verify correct student advancement pathways.
* School Enrollment Metrics
  - Quick, low-friction counters displaying: Total Active | Suspended | Transferred students.
* Daily Attendance Snapshot
  - Real-time minimal donut graph or percentage labels showing today's attendance:
    Present % | Late % | Absent %.
* High-Risk Health Flags
  - Fixed, high-contrast micro-panel displaying a count of students with severe chronic
    conditions or critical emergency instructions. Clicking opens immediate detail view.

## 2. MAIN TWO-COLUMN WORK GRID (DESKTOP: 65% LEFT / 35% RIGHT)

---

===============================================================================
LEFT COLUMN (65% Width): THE CBC ACADEMIC ENGINE & MACRO TRENDS
===============================================================================

SUB-SECTION A: THE CBC STUDENT PERFORMANCE MATRIX
Layout: Clean, bordered block with micro-tabbed sub-navigation to alternate views.
Intent: High-density scannable text leaderboards replacing heavy multi-row tables.

Features:

- Top Performing CBC Students
  - Leaderboard ranking students by their weighted final averages.
  - Calculated automatically via numeric conversions of their score levels (EE=4, ME=3, AE=2, BE=1).
- Most Improved CBC Trajectory
  - Performance tracker highlighting students with the highest positive upward shift in
    overall score levels (e.g., jumping from AE to ME) compared to the prior term.
- Most Regressed CBC Trajectory
  - Watchlist identifying students whose general score levels have dropped significantly
    term-over-term for immediate counselor or headteacher flagging.
- CBC Academic Intervention Watchlist
  - High-priority list isolating students who consistently score at the BE (Below Expectation)
    level across multiple distinct learning areas.

SUB-SECTION B: MACRO ANALYTICS & TREND VISUALIZATIONS
Layout: Grid of small, flat card components containing minimal canvas/SVG visual vectors.
Intent: Aggregate data presentation to guide administrative optimization decisions.

Features:

- Learning Area Mastery Trends
  - A compact visual heat-matrix showing which specific CBC learning areas (e.g., Mathematical
    Activities, Language Activities) possess the highest density of EE scores school-wide.
- Stream-by-Stream CBC Performance Variance
  - Side-by-side stream comparison tool (e.g., Grade 3 East vs Grade 3 West) showing distribution
    curves of score levels to pinpoint classroom performance disparities.
- Year-Over-Year CBC Mastery Comparison
  - A clean, single-line trend graph tracking the school's overall performance level distribution
    for the current academic period against the exact same calendar timeline from last year.
- Attendance vs. CBC Score Correlation Graph
  - Scatter plot mapping individual student attendance percentages directly against their cumulative
    curriculum performance level to visually demonstrate the clear impact of missed lessons.

===============================================================================
RIGHT COLUMN (35% Width): COMPLIANCE, FINANCIALS, AUDITS & FEEDS
===============================================================================

SUB-SECTION C: SYSTEM COMPLIANCE & DATA INTEGRITY
Layout: Stacked text cards with small amber warning markers. Max 5 items per list.
Intent: Fast data clean-up action items. Fixes data fragmentation before it impacts reporting.

Features:

- Students without an assigned class in the current academic period.
- Classes without any students assigned.
- Classes without at least one subject teacher assigned.
- Teachers without at least one class subject assigned.
- Parents without at least one child student assigned.

SUB-SECTION D: TEACHER & ASSESSMENT AUDITS
Layout: List elements with interactive checkbox states.
Intent: Tracking administrative compliance of instructors and layout logic.

Features:

- CBC Task Evaluation Compliance
  - Operational audit list showing which class teachers have fully graded their administered
    formative tasks versus those with pending, overdue evaluation records.
- Formative Task Weighting Audit
  - Operational checklist flagging any CBC classrooms where active task counts or assigned types
    do not properly align with configured school assessment weights for the term.
- Chronic Absentee Watchlist
  - List of individual students whose cumulative "ABSENT" logs in cbc_attendance_logs
    exceed the school’s maximum allowed safety threshold.

SUB-SECTION E: INSTITUTIONAL FINANCE
Layout: Structured billing component with high visual priority on outstanding cash flow.
Intent: Immediate, frictionless access to school financial health variables.

Features:

- Outstanding Balance Total
  - Prominent, bold numeric metric displaying the aggregate balance due from all unpaid
    or partially paid student invoices.
- Fee Collection Progress Bar
  - Visual progress bar overlay comparing total fee currency amounts invoiced against
    actual liquid payments received for the current term.
- Mandatory Fee Categories Checklist
  - Mini summary checklist displaying structural fee categories (e.g., Tuition, Transport)
    actively applied across the school for verification.

SUB-SECTION F: REAL-TIME OPERATIONAL FEEDS
Layout: Vertical chronological timeline rows.
Intent: Tracking daily live human activity elements inside the school environment.

Features:

- Pre-Invited User Queue
  - Compact feed showing a maximum of 5 pending teacher or staff accounts who have pending
    invitations sent out but have not yet linked/activated their external Stytch identities.
- Recent Medical Incidents
  - Chronological activity feed of today's health logs, showing student names, recorded
    symptoms, and immediate actions taken by staff.

===============================================================================
DESIGN NOTE: TIMETABLE CONFLICTS
===============================================================================

- Database-level GiST exclusion constraints (excl_cbc_timetable_teacher and
  excl_cbc_timetable_room) render timetable conflicts physically impossible to
  save. Because the system actively blocks double-bookings natively on execution,
  conflict alert widgets are omitted entirely from the UI to preserve whitespace.
  ===============================================================================
