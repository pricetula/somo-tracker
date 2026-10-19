# Closure Checklist – Timetable Conflict Detection

Feature: `docs/features/FEATURE-timetable-conflict-detection.md`
Gap: `docs/gaps/GAP-timetable-conflict-detection.md`

## Definition of Done

- [ ] Acceptance Criteria all checked
- [ ] Backend validation in `internal/services/timetable_service.go` SetupClassTimetableSlot
- [ ] Transactional validation for teacher, room, class
- [ ] Error contract compliant: 409 with code `timetable_conflict`, `errors.conflicts[]` populated
- [ ] RLS enforced, school_id from `app.current_tenant_id`, term resolved via `AcademicPeriodResolver`
- [ ] Indexes exist for validation queries
- [ ] Frontend `assign-slot-form.tsx` handles 409 `timetable_conflict` with inline field errors + toast
- [ ] No dumb back buttons
- [ ] Tests pass
  - [ ] Unit tests for each conflict type
  - [ ] Integration tests for 409 response shape
  - [ ] RLS isolation test
  - [ ] Playwright E2E test passes
- [ ] Security audit passed via `/skill:security-audit`
- [ ] Error contract validated via `/skill:error-contract`
- [ ] Compliance passed via `/review`
- [ ] Docs synced: Feature status `Done`, Gap status `Done` with Closed date
- [ ] Implementation Notes updated to remove pending items

## Current State
Status reverted to `In Progress` on 2026-10-15 due to unchecked acceptance criteria.

## Next Steps
1. Verify frontend error handling is complete and tested
2. Tick acceptance criteria in feature spec
3. Run `pnpm run test:all` and Playwright tests
4. Run `/review` and `/security`
5. Update Implementation Notes and sync GAP
6. Mark feature `Done` only after all checks pass
