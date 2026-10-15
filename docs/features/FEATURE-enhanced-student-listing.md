# Feature: Enhanced Student Listing

**Status:** Proposed
**Owner:** Product / Platform
**Created:** 2026-10-15
**Last Updated:** 2026-10-15
**Related Issues:** #
**Affected Layers:** backend | frontend

## Summary
Enhance the student list view with richer columns, inline KPIs, bulk actions, and improved filtering to support faster triage in school management.

## Current State
**Backend:** `ListStudents` service returns `student_id, admission_number, full_name, gender, class_id, class_name, grade_level`. No `date_of_birth`, `guardian_linked`, or `attendance_rate`.
**Frontend:** `StudentsTable` renders columns Name, Admission No., Class, Grade Level, Gender, Actions. Type `StudentListItem` includes `date_of_birth` in TypeScript but backend does not populate it consistently.
**Docs:** Gap documented in `docs/gaps/GAP-student-list-missing-fields.md`

## Problem Statement
Current `StudentsTable` shows Name, Admission No., Class, Grade Level, Gender, Actions. No DOB/age, no guardian status indicator, no attendance hint, no bulk export. Admins need to open detail page for basic context.

## Solution Overview
Improve list table with:
* Additional columns: Date of Birth, Guardian status, Attendance %
* Inline badges for without class / without guardian
* Bulk actions: Bulk assign class, Bulk delete
* Enhanced filters: Attendance threshold
* Column visibility toggle
* Row actions quick edit

### User Stories
- As Admin, I want to see attendance % in list to spot at-risk students quickly
- As Admin, I want to bulk assign students to a class

### Acceptance Criteria
- [ ] List API returns date_of_birth and guardian_linked flag
- [ ] List API returns term attendance % per student
- [ ] Inline badges for missing class/guardian
- [ ] Bulk actions toolbar visible when rows selected
- [ ] Column visibility toggle works
- [ ] Filters: Attendance < X%
- [ ] No dumb back buttons

## UI Wireframe
```
[Students List Toolbar]
Search: [Search by name or admission...]
Filters: [Class ▼] [Attendance < 75% ▼] [Without Class] [Without Guardian]
Actions: [Bulk Assign Class] [Bulk Delete]   Column Visibility: [▼]

Table Columns:
☑ Name | Admission No. | DOB | Class | Grade | Guardian | Attendance % | Actions

Row example:
☑ Jane Doe | 2024-001 | 2011-04-12 | Grade 7A | Grade 7 | ✅ | 92% 🟢 | ⋮
☑ John Smith | 2024-015 | 2010-09-03 | — ⚠️ | Grade 8 | ❌ | 64% 🔴 | ⋮
```

## Proposed API Response Shape

Enhanced `StudentListItem`:
```json
{
  "student_id": "uuid",
  "admission_number": "string",
  "full_name": "string",
  "date_of_birth": "YYYY-MM-DD",
  "gender": "M|F|OTHER",
  "class_id": "uuid|null",
  "class_name": "string|null",
  "grade_level": "string|null",
  "guardian_linked": true|false,
  "attendance_rate": 92.5|null
}
```
List response wraps items + total.

## Proposed SQL Changes for ListStudents

Current query returns: student_id, admission_number, full_name, gender, class_id, class_name, grade_level.

Proposed extensions:
* Add `date_of_birth` from `students.date_of_birth`
* Add `guardian_linked` boolean via EXISTS subquery on `guardian_student_links` where student_id matches and link is active
* Add `attendance_rate` numeric computed per student for current academic term:
  - Join `timetable_attendance` filtered by student_id, academic_term_id resolved for school
  - Aggregate counts: present, absent, late
  - Compute rate = present / (present + absent + late) * 100
  - Use LEFT JOIN to allow students with no attendance records → rate NULL
* Keep existing filters: school_id, search on full_name or admission_number, class_id, without_class, without_guardian
* Maintain pagination with LIMIT/OFFSET

Performance considerations:
* Attendance aggregation per row is expensive; consider subquery with GROUP BY student_id or materialized view `student_term_attendance_stats`
* Index needed on `timetable_attendance(student_id, academic_term_id, status)`

## Scope

### Backend
- Extend `ListStudents` query to include date_of_birth, guardian_linked, attendance_rate
- Performance: pagination + computed attendance may need materialized view

### Frontend
- Extend `StudentListItem` type
- Add new columns with formatting
- Add bulk action toolbar
- Add column visibility menu

### Non-Goals
- Student photos in list
- Document management

## Design Notes
Attendance % computation uses summary table `student_term_attendance_stats` updated by background workers per ADR-2026-10-15-stats-architecture. `guardian_linked` sourced from `guardian_link_stats`. No on-the-fly aggregation.

## Open Gaps / Risks
- Attendance computation per row may be expensive
- Note: Background-updated summary tables are preferred long term. Deferred to later doc.

## Test Plan
- API contract tests
- E2E bulk assign class
- Performance test with 5k students

## References
- Current list: `frontend/src/features/students/components/students-list.tsx`
- Service: `backend/internal/services/students_service.go`
