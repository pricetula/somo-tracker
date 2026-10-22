# Gaps & Backlog Index
This is the high-level inventory of known gaps and missing functionality. Detailed items live under `docs/gaps/GAP-<slug>.md`.
## How to use
- Add new gaps via `docs/templates/gap.md`
- Link to GitHub Issues for tracking and prioritization
- Move to `docs/features/` when a gap is accepted and specced
## Open Gaps
| Gap | Area | Severity | Status | Owner |
|-----|------|----------|--------|-------|
| Student Timetable View Missing | frontend | High | In Progress | ... |
| Assessment and Report Card Management | backend | frontend | High | Open | Product / Platform |
| Health / Nurse Domain — Orphan Schema | backend | frontend | High | Open | Platform team |
| CSP Header Configuration Unverified / Missing | backend | Medium | Open | ... |
| Timetable Conflict Detection Missing | backend | High | In Progress | Platform team |
| Invitation Handlers Unsafe Type Assertion | backend | High | Open | ... |
| Finance Handler Error Inconsistency | backend | Medium | Open | ... |
| GetTemplate Returns 400 Instead of 404 | backend | Medium | Open | ... |
| SSE Endpoints Missing OpenAPI Annotations | backend | Low | Open | ... |
| Timetable Inline DTOs No Validation | backend | Low | Open | ... |
| Students Import Inline Errors | backend | Medium | Open | ... |
| Events Handler Inline Auth Errors | backend | Medium | Open | ... |
| Me Handler Inline Auth Errors | backend | Medium | Open | ... |
| School Handler ListSchools Uses fiber.NewError | backend | Medium | Open | ... |
| User Counts Handler Uses fiber.NewError | backend | Medium | Open | ... |
| ... | ... | ... | Open | ... |
## Closed Gaps
| Gap | Area | Severity | Status | Owner |
|-----|------|----------|--------|-------|
| ... | ... | ... | ... | ... |
## Deferred / Won't Do
| Gap | Reason |
|-----|--------|
| ... | ... |
## References
- `docs/planning/student-module-roadmap.md` - Consolidated student module gaps and roadmap
## Process Notes
- Severity guidance:
  - Critical: Blocks release / security / data loss
  - High: Core user flow broken
  - Medium: Nice to have / workaround exists
  - Low: Polish / tech debt
- Always document impact and reproduction steps
