# Status Mismatch Audit - 2026-10-15

## Findings
Features marked `Status: Done` with incomplete or missing acceptance criteria.

| Feature | Status | Acceptance Criteria Checked | Issue |
|---------|--------|------------------------------|-------|
| FEATURE-attendance-marking-integration.md | Done | 0/3 | All unchecked |
| FEATURE-attendance-reporting.md | Done | 0/2 | All unchecked |
| FEATURE-attendance-summary-stats.md | Done | 0/2 | All unchecked |
| FEATURE-classes-enrollment-polish.md | Done | 0/3 | All unchecked |
| FEATURE-enhanced-student-profile.md | Done | 0/9 | All unchecked |
| FEATURE-student-timetable-view.md | Done | 0/6 | All unchecked |
| FEATURE-timetable-conflict-detection.md | Done | 0/6 | All unchecked, Implementation Notes mentions pending frontend handling |

## Root Cause
Status was set to Done without verifying Acceptance Criteria, Test Pass, Error Contract, Security Audit, and Compliance. Planning checklists were marked complete prematurely.

## Remediation
1. Revert Status to `In Progress` for all above features
2. Complete implementation to satisfy all Acceptance Criteria
3. Validate Error Contract, Security Audit, Migration Tests, and Compliance via `/review`
4. Only then mark Status `Done` and sync GAP status

## Remediation — Updated 2026-10-20
- `FEATURE-timetable-conflict-detection.md`: Status reverted to `In Progress`; `SetupClassTimetableSlot` conflict detection verified (service tests PASS); handler now returns structured `errors.conflicts[]`; error-contract validated.
- `FEATURE-student-timetable-view.md`: Status reverted to `In Progress`; endpoint `GET /api/students/:id/timetable` exists (`GetStudentTimetable`); pending RLS/test/E2E verification.
- `GAP-*` files updated to `In Progress` with open criteria listed.
- Only mark `Done` after all AC checked, `/review` passed, and GAP synced.

## Definition of Done
- All Acceptance Criteria `[x]`
- Implementation Notes contain no pending items
- Tests pass: `pnpm run test:all`
- Error contract validated via `error-contract` skill
- Security audit passed via `security-audit` skill
- Compliance passed via `agendas-compliance` skill
- GAP status synced
