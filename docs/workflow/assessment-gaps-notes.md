# Domain Gap Notes — Session Restart Helper

Last updated: 2026-11-01

## Context

* Assessment spec (`FEATURE-ASSESSMENTS-001`) and UI/API draft (`FEATURE-ASSESSMENTS-UI-001`) committed.
* Team decision: continue feature documentation. Attendance cannot be documented until school calendar and timetable are resolved, since attendance ties to subject/teacher assignments and daily scheduling.

## Gap chain

1. **School Calendar**
   * Needed for: term dates, school-day boundaries, day close, attendance roll-call eligibility, assessment date validity.
   * Missing entities: `academic_years` exists, `terms` exists, but no explicit `school_calendar` / `school_days` or holiday list.
   * Questions:
     * Do we need a per-school `school_calendar` table with `school_day` dates, `is_school_day`, `is_holiday` flags?
     * How to handle different calendars per school within the same tenant?
     * Does day close use calendar to define “today”? Africa/Nairobi time zone?
   * Decisions:
     * Yes, model `school_calendar` per school with `date` and `is_holiday` flag only. `is_school_day` is implied by `is_holiday = false` and date falling within an active term. This accommodates school trip days for certain classes via class-level overrides later if needed.
     * Each school has its own calendar entries; calendars are scoped to `school_id`, not tenant.
     * Day close is defined as the end of the Africa/Nairobi calendar day. Store timestamps in UTC in DB, convert to Africa/Nairobi for “today”/day-close logic.

2. **Timetable**
   * Needed for: linking subject, teacher, classroom, and time slot per day; attendance is per class/day, but timetable defines which teacher is present and which subjects are scheduled.
   * Missing entities: `timetable`, `timetable_entries`, possibly `periods`.
   * Questions:
     * Is timetable per class + academic year, with subject offering mapping?
     * Do we need period start/end times, or just sequence index for ordering?
     * Does timetable drive assessment scheduling constraints (assessment date must fall within term and on a school day)?
     * Teacher assignment already partially covered by `class_teacher_assignments` and `subject_offerings`. Need clarity on multiple teachers per subject offering.

3. **Attendance**
   * Depends on calendar (valid school days) and timetable (teacher/class context).
   * Core acceptance criteria already identified in discussion: median roll-call <60s, edit after day close with reason+audit, guardian view.
   * Do not document Attendance until calendar & timetable decisions are recorded.

## Dependencies

* Attendance → Calendar → Timetable → Subject offering / Class teacher assignment.
* Assessment already depends on subject offering, term, and academic year; it can assume valid dates via calendar.
* Report cards depend on attendance aggregates? Not in v1 per PRODUCT-BRIEF, but term summary may be needed.

## Next actions

* Document `School Calendar` feature before `Timetable`.
* Capture calendar/timetable decisions in a short `docs/workflow/notes-calendar-timetable.md` file once approved.
* Then scaffold `FEATURE-ATTENDANCE-001.md`.

## Open questions

* Do we model holidays per school or per tenant?
* Are school days computed from term dates minus weekends/holidays? Or manually curated?
* Is timetable required for MVP? Could attendance be classroom-level without timetable? Team says yes — timetable needed for subject/teacher tie.
* Time zones: all Africa/Nairobi for UI, UTC in DB.

## References

* docs/product/brief.md — PP1–Grade 9 CBC, mobile-first, attendance success metrics.
* docs/features/FEATURE-ASSESSMENTS-001.md — assessment spec with term/academic year dependencies.
* docs/workflow/migration-roadmap.md — attendance table blueprint.
* docs/domain/glossary.md — academic years, terms, enrollments.
