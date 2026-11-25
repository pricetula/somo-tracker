# ADR-0016: Simple Report Cards

## Metadata

| Field | Value |
|-------|-------|
| id | ADR-0016 |
| title | Simple Report Cards |
| status | proposed |
| date | 2026-11-25 |
| author | frontend, product |
| deciders | frontend, product |
| relates_to | SPEC-002, ADR-0010, ADR-0013, ADR-0008 |

## Status

proposed

## Context

Schools need to provide parents/guardians with a summary of learner performance at the end of each term. The current assessment model (ADR-0010, SPEC-002) stores raw scores and derived grades but has no mechanism to generate or deliver report cards.

Official transcripts (for leaving students, national exam registration) are out of scope for MVP. Simple term report cards showing per-subject grades are in scope.

## Decision

Adopt a **simple term report card model**:

1. **Report card generation**: Computed on demand from `assessment_results` joined with `students`, `streams`, `school_subjects`, `school_academic_terms`, and the school's active grading scale bands (ADR-0008).

2. **Report card contents** (per student, per term):
   - Student name, admission number, stream, grade year
   - Academic year and term
   - Per subject: score, derived grade (per school's active scale), teacher comment (optional)
   - Term aggregate (average, optional)
   - Teacher signature/name
   - School name and contact
   - Guardian contact for delivery (see ADR-0013)

3. **No stored report card snapshots**: Report cards are generated on demand from live assessment data. No `report_cards` table in MVP. Historical report cards can be regenerated if assessment data is retained.

4. **Delivery**: PDF generation and email/print delivery is a frontend concern. Guardian contact data comes from ADR-0013.

5. **Grading**: Uses the school's active grading scale — either percentage bands or rubric levels (ADR-0008, B1).

## Alternatives Considered

| Alternative | Pros | Cons | Why rejected |
|-------------|------|------|--------------|
| Stored report card snapshots | Immutable historical record | Data duplication, sync complexity | Rejected — regenerate on demand |
| Official transcripts | Recognized by schools/employers | Requires national body alignment, legal review | Rejected — out of MVP scope |
| Simple on-demand report card (chosen) | Simple, no extra storage, always current | Not immutable (grade changes affect old cards) | Accepted — MVP scope |

## Consequences

### Positive

- Simple term reports for parents/guardians.
- No additional tables or storage.
- Uses existing assessment + grading data.

### Negative

- Not immutable — if a teacher changes a grade, prior report cards reflect the change when regenerated.
- No official transcript for national exams or school transfers.

### Neutral / Risks

- PDF generation adds frontend/backend complexity; print-friendly HTML is acceptable for MVP.

## Links

- Spec: `docs/specs/002-education-system-configuration.md`
- Local assessments: `docs/adr/0010-local-assessments-only.md`
- Grading scales: `docs/adr/0008-per-school-configuration-overrides.md`
- Guardians: `docs/adr/0013-guardian-management.md`
