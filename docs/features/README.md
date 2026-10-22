# Features Index

This index lists all documented features for Somotracker. Each feature lives under `docs/features/FEATURE-<slug>.md` using the template in `docs/templates/feature-spec.md`.

## Legend
- Status: Draft | Proposed | Accepted | In Progress | Done | Deprecated

## Features

| Feature | Slug | Status | Owner | Layer |
|---------|------|--------|-------|-------|
| Backend Schema Overview | backend-schema | Done | Platform | backend |
| Bulk Operations Framework | bulk-jobs | Done | Platform | backend |
| Enhanced Student Profile | enhanced-student-profile | Done | Product / Platform | backend, frontend |
| Enhanced Student Listing | enhanced-student-listing | Proposed | Product / Platform | backend, frontend |
| Timetable Conflict Detection | timetable-conflict-detection | In Progress | Platform | backend, frontend |
| Student Timetable View | student-timetable-view | In Progress | Product / Platform | backend, frontend |
| Student Schema Enhancement | student-schema-enhancement | Proposed | Platform | backend |
| Timetable Class Assignment UI Polish | timetable-class-assignment-ui | Done | Product / Platform | frontend, backend |
| Teacher Timetable View | teacher-timetable-view | Proposed | Product / Platform | backend, frontend |
| Class Timetable View | class-timetable-view | Proposed | Product / Platform | backend, frontend |
| Guardian Linking | guardian-linking | Proposed | Product / Platform | backend, frontend |
| Guardian Profile Enhancement | guardian-profile-enhancement | Proposed | Product / Platform | backend, frontend |
| Guardian Summary Stats | guardian-summary-stats | Proposed | Product / Platform | backend, frontend |
| Teacher Profile Enhancement | teacher-profile-enhancement | Proposed | Product / Platform | backend, frontend |
| Teacher Class/Subject Linking | teacher-class-subject-linking | Proposed | Product / Platform | backend, frontend |
| Teacher Summary Stats | teacher-summary-stats | Proposed | Product / Platform | backend, frontend |
| Classes Enrollment Polish | classes-enrollment-polish | Proposed | Product / Platform | backend, frontend |
| Class Profile Enhancement | class-profile-enhancement | Proposed | Product / Platform | backend, frontend |
| Class Summary Stats | class-summary-stats | Proposed | Product / Platform | backend, frontend |
| Attendance Marking Integration | attendance-marking-integration | Proposed | Product / Platform | backend, frontend |
| Attendance Summary Stats | attendance-summary-stats | Proposed | Product / Platform | backend, frontend |
| Attendance Reporting | attendance-reporting | Proposed | Product / Platform | backend, frontend |
| Events Management Enhancement | events-management-enhancement | Proposed | Product / Platform | backend, frontend |
| Rooms Management Enhancement | rooms-management-enhancement | Proposed | Product / Platform | backend, frontend |
| Curriculum Enhancement | curriculum-enhancement | Proposed | Product / Platform | backend, frontend |
| Assessment and Report Card Management | assessment-and-report-card | Draft | Product / Platform | backend, frontend, docs |
| ... | ... | ... | ... | ... |

## How to add a feature
1. Copy `docs/templates/feature-spec.md` to `docs/features/FEATURE-<slug>.md`
2. Fill in Problem, Solution, Acceptance Criteria
3. Link related GitHub Issues
4. Update this index

## References
- `docs/reference/backend_schema.md` - Current database schema summary
- `docs/planning/student-module-roadmap.md` - Student module consolidated roadmap
- Monorepo structure: backend/, frontend/, public/, docs/
