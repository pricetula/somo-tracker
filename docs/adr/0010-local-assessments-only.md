# ADR-0010: Local Teacher-Entered Assessments Only

## Metadata

| Field | Value |
|-------|-------|
| id | ADR-0010 |
| title | Local Teacher-Entered Assessments Only |
| status | proposed |
| date | 2026-11-25 |
| author | backend |
| deciders | backend, product |
| relates_to | SPEC-002, specs/001-auth-otp-request.md |

## Status

proposed

## Context

SPEC-001 references national examination bodies (KNEC, NESA, NECTA, UNEB) and national exams (KCPE, KCSE, PLE, UCE, UACE, CSEE, ACSEE). Integrating with these boards would require APIs, secure credential handling, result parsing, and cross-border certificate equivalency logic.

For the initial release, the system will not integrate with any national examination board. All assessments (quizzes, term exams, final-year exams) are created locally by teachers within a school, and results are entered manually.

## Decision

Adopt a **local-only assessment model**:

1. Assessments are created by teachers for a specific stream, subject, term, and school.
2. Scores and grades are entered manually by teachers into `assessment_results`.
3. No tables exist for external exam boards, national results, or certificate equivalency.
4. No API calls are made to KNEC, NESA, NECTA, UNEB, or any other national body.
5. Final-year results are recorded as local school assessments, not as nationally certified credentials.

## Alternatives Considered

| Alternative | Pros | Cons | Why rejected |
|-------------|------|------|--------------|
| National exam board API integration | Official results, reduces manual entry | Requires board APIs (often unavailable), secure credentials, parsing, error handling, rate limits | Rejected — out of scope for MVP |
| Hybrid (local + optional national) | Flexible | Increases complexity; partial integration is harder than none | Rejected — defer until local model is stable |
| Local-only (chosen) | Simple, fast to build, full control over data | Schools must enter all results manually | Accepted — matches MVP scope |

## Consequences

### Positive

- Fast development; no external dependencies or credential management.
- Full control over assessment lifecycle and data model.
- Works offline or in environments with limited internet access.

### Negative

- Teachers must manually enter every result; no automatic import from national exams.
- No official certification or transcript validation from national bodies.

### Neutral / Risks

- Manual entry introduces risk of data entry errors; teacher training and UI validation are important.
- Future integration with national boards would require a new ADR and schema extensions.

## Links

- Spec: `docs/specs/002-education-system-configuration.md`
- Auth spec: `docs/specs/001-auth-otp-request.md`
