# Calendar & Timetable Decisions

Status: draft
Last updated: 2026-11-05

## School Calendar

**Decisions**
- Model `school_off_dates` per school, scoped to `school_id`. Renamed from `school_calendar` for clarity – table stores only dates school is off.
- Table shape: `school_off_dates(school_id, date date, off_type text, note text nullable)`. Presence of a row means the date is off. `off_type` can be 'holiday','closure','inset'. No country holidays table; use off_type = 'holiday'.
- DB stores UTC, business logic in Africa/Nairobi. Yes.
- Do NOT pre-populate every calendar day. Store only exception dates. Weekends are handled later by timetable.
- `is_school_day` for attendance will be driven by timetable existence, not weekday. Timetable will fix weekend gaps. `school_off_dates` still blocks attendance on off days.
- This allows per-school holidays and school-trip days. Class-specific trip days will be handled as overrides in timetable/attendance later if needed; v1 keeps school-wide holiday flag.
- Day close = end of Africa/Nairobi calendar day. DB stores UTC; application converts to Africa/Nairobi for “today” determination and day-close checks.
- Calendar entries are manually curated per school. Admin creates holiday entries; weekends are treated as non-school days by application logic.

**Open items**
- UI for bulk import of holidays per academic year.
- Validation: assessment dates must fall on non-holiday school days within term.
- New table `assessment_dates` to be designed for assessment scheduling.
- Events for partial attendance impact: debate, music, trips for select groups of students. Needs decision on modeling.

**Events consideration**
- Use case: select group of students attend debate/musical on a school day. School remains open for others.
- Proposed model:
  * `school_off_dates(school_id, date, off_type, note)` remains exception-only, school-wide.
  * `school_events(id, school_id, date, start_time, end_time, title, event_type, reason, affects_attendance boolean)`
  * `school_event_students(event_id, student_id)`
  * `school_event_teachers(event_id, teacher_id)`
  * Multi-day events are per-day rows. Each event has start_time/end_time. No time validation required.
  * Attendance affected based on event time. Enforce no overlapping events for same student on same date: do not allow student to be linked to two events whose times overlap.
- Benefits: teachers/students can see their events and attendance can pre-mark as excused when `affects_attendance=true` and student is linked.
- Open questions:
  * Multi-day events: store per-day rows or date range + recurrence?
  * Scope to classroom vs individual students – per-student links give flexibility but increase data entry.
  * Should events be allowed on holiday dates? Validation needed.
- Recommendation for v1: keep school-wide calendar exception-only, add events with date + start/end time, optional participant links, and auto-excuse logic.

## Timetable

**Pending decisions**
- Is timetable per class + academic year with subject offering mapping? Need confirmation.
- Period start/end times vs sequence index.
- Does timetable gate assessment scheduling?
- Multiple teachers per subject offering clarity.

**Dependencies**
- Attendance → Calendar → Timetable.

## Next actions
- Document `FEATURE-SCHOOL-CALENDAR-001` with schema and API.
- Then document timetable feature.
- Then scaffold `FEATURE-ATTENDANCE-001`.
