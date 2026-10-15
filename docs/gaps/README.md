# Gaps & Backlog Index

This is the high-level inventory of known gaps and missing functionality. Detailed items live under `docs/gaps/GAP-<slug>.md`.

## How to use
- Add new gaps via `docs/templates/gap.md`
- Link to GitHub Issues for tracking and prioritization
- Move to `docs/features/` when a gap is accepted and specced

## Open Gaps

| Gap | Area | Severity | Status | Owner |
|-----|------|----------|--------|-------|
| Bulk Import UI | frontend | High | Open | ... |
| Student Detail Page is Placeholder | frontend | High | Open | ... |
| Student Schema Missing Structured Fields | backend | High | Open | ... |
| Student List Missing Fields for Enhanced Listing | backend | High | Open | ... |
| Timetable Class Assignment UI Missing | frontend | Critical | Open | ... |
| Timetable Conflict Detection Missing | backend | High | Open | ... |
| Student Timetable View Missing | frontend | High | Open | ... |
| Assessment and Report Card Management | backend | frontend | High | Open | ... |
| ... | ... | ... | Open | ... |

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
