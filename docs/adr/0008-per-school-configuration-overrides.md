# ADR-0008: Per-School Configuration Overrides

## Metadata

| Field | Value |
|-------|-------|
| id | ADR-0008 |
| title | Per-School Configuration Overrides |
| status | proposed |
| date | 2026-11-25 |
| author | engineering |
| deciders | backend, product |
| relates_to | SPEC-002, ADR-0007, ADR-0009 |

## Status

proposed

## Context

When a school is created, it selects an `education_system_id`. The system seeds per-school configuration tables based on that education system’s defaults. However, individual schools need flexibility to adapt the configuration to their local context:

- Some schools may want a different grading scale than the system default.
- Some schools may want percentage-based grading (A–F, 0–100) while others use rubric-based competency levels (EE1/ME1/AE1/BE1) mapped to percentage ranges.
- Some schools may want to make a system-default-optional subject compulsory, or vice versa.
- Term dates vary by school even within the same country and education system.
- Language of instruction may differ from the system default (English) at specific schools.

These overrides must be configurable per school without code changes.

## Decision

Adopt a **school-level override model**:

1. A school adopts exactly one education system at creation.
2. Per-school configuration tables are seeded from education system defaults at school creation:
   - `school_grading_scales` + `school_grading_bands`
   - `school_subjects` (compulsory/optional flags)
   - `school_academic_years` + `school_academic_terms` (dates)
   - `school_grade_years` (local grade names)
   - `schools.language_of_instruction` (default: system default)
3. A school admin may later edit these per-school settings. Changes affect only that school.

## Alternatives Considered

| Alternative | Pros | Cons | Why rejected |
|-------------|------|------|--------------|
| Global-only configuration (no school overrides) | Simpler, consistent across all schools | Cannot accommodate local term dates, grading preferences, or subject choices | Rejected — schools need local autonomy |
| Per-student or per-class overrides | Maximum flexibility | Excessively complex; inconsistent grading within a school | Rejected — not needed for scope |
| School admin edits code-backed constants | No config tables | Requires deploy for every school-specific change | Rejected — violates table-driven principle |

## Consequences

### Positive

- Schools can tailor grading, subjects, terms, and language to local needs.
- Overrides are isolated to one school; no leakage to sibling schools sharing the same education system.
- Generic admin UI manages rows, not hardcoded country logic.

### Negative

- Configuration drift between schools using the same education system is possible (by design).
- More data to validate (e.g. ensuring term dates don’t overlap within a school year).

### Neutral / Risks

- School admins may create invalid combinations (e.g. overlapping terms); UI validation required.

## Links

- Spec: `docs/specs/002-education-system-configuration.md`
- Parent ADR: `docs/adr/0007-table-driven-education-configuration.md`
- No JSONB: `docs/adr/0009-normalized-relational-configuration.md`
