# ADR-0012: Attendance Tracking

## Metadata

| Field | Value |
|-------|-------|
| id | ADR-0012 |
| title | Attendance Tracking |
| status | proposed |
| date | 2026-11-25 |
| author | backend, product |
| deciders | backend, product |
| relates_to | SPEC-002, ADR-0007, ADR-0010 |

## Status

proposed

## Context

Schools need to track learner attendance as a core administrative function. Absenteeism is a significant concern across East African schools (rural access, pastoralist mobility, term-time absence, illness). Attendance data also supports progression decisions and early-warning for at-risk learners.

Attendance must be simple for teachers to record daily, per stream, and queryable for reporting by class, term, and learner.

## Decision

Adopt a **per-day, per-student attendance record model**:

1. **`attendance_records` table**: One row per student per school day.
   - `id` UUID PK
   - `student_id` UUID → `students.id`
   - `stream_id` UUID → `streams.id` (denormalized for fast class-level queries)
   - `academic_year_id` UUID → `school_academic_years.id`
   - `attendance_date` DATE
   - `status` TEXT: `present`, `absent`, `late`, `excused`
   - `recorded_by_teacher_id` UUID → `teachers.id`
   - `recorded_at` TIMESTAMPTZ
   - Unique constraint on `(student_id, attendance_date)` prevents duplicate entries for the same day.

2. **Daily recording**: Teachers mark attendance once per day per stream. A teacher records for streams they teach.

3. **No session-level tracking initially**: Attendance is recorded once per day, not per period/subject. This keeps the UI simple for teachers and matches the MVP scope.

4. **Aggregation**: Attendance percentages are computed on read (GROUP BY student, date range) rather than stored, avoiding drift.

## Alternatives Considered

| Alternative | Pros | Cons | Why rejected |
|-------------|------|------|--------------|
| Per-period attendance | Granular, subject-level absence | High teacher burden, complex UI | Rejected — MVP needs daily simplicity |
| Parent-reported absence | Reduces teacher workload | Unreliable, no verification | Rejected — school-owned record |
| Daily only (chosen) | Simple, fast, matches paper registers | No period-level detail | Accepted — can extend later |

## Consequences

### Positive

- Simple daily workflow for teachers (matches existing paper registers).
- Easy class-level and term-level reporting.
- Unique constraint prevents double-entry.

### Negative

- No period/subject-level absence detail (can be added in a future ADR).
- No automated absence alerts (future enhancement).

### Neutral / Risks

- Teachers must remember to mark daily; no reminder mechanism in scope.
- Attendance data quality depends on consistent teacher entry.

## Links

- Spec: `docs/specs/002-education-system-configuration.md`
- Parent ADR: `docs/adr/0007-table-driven-education-configuration.md`
- Students: `docs/specs/002-education-system-configuration.md` (`students` table)
- Streams: `docs/specs/002-education-system-configuration.md` (`streams` table)
