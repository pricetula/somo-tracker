# Gap: Student List Missing Fields for Enhanced Listing

**Area:** backend
**Severity:** High
**Status:** Done
**Owner:** Platform team
**Created:** 2026-10-15
**Closed:** 2026-10-15

## Description
`ListStudents` API and `StudentListItem` type do not return `date_of_birth`, `guardian_linked` flag, or `attendance_rate`. Current fields: student_id, admission_number, full_name, gender, class_id, class_name, grade_level.

Frontend cannot render Age, Guardian status badge, or inline Attendance % without additional queries.

## Impact
Enhanced student listing cannot be implemented. Users must open detail page for basic context.

## Proposed Solution
Extend `ListStudents` SQL to select date_of_birth, compute guardian_linked via exists check, compute attendance_rate for current term.

## Acceptance Criteria
- [x] `StudentListItem` includes `date_of_birth`, `guardian_linked`, `attendance_rate`
- [x] API response includes new fields
- [x] Frontend table renders Age, Guardian badge, Attendance %

## Links
- Feature: `docs/features/FEATURE-enhanced-student-listing.md`
