# ADR-0011: Education Scope Constraints

## Metadata

| Field | Value |
|-------|-------|
| id | ADR-0011 |
| title | Education Scope Constraints |
| status | proposed |
| date | 2026-11-25 |
| author | product, backend |
| deciders | product, backend |
| relates_to | SPEC-002, ADR-0007, ADR-0010 |

## Status

proposed

## Context

Somotracker supports Kenya, Rwanda, Tanzania, and Uganda. Each country has early childhood education, primary, lower secondary, upper secondary, and tertiary levels. There are also legacy education systems (e.g. Kenya’s 8-4-4) still in use alongside current systems (e.g. CBC).

The product must define clear boundaries for what is in scope for the initial release to avoid scope creep and ensure the data model remains coherent.

## Decision

Adopt the following **scope constraints**:

1. **Primary to high school only.** Configuration covers Primary, Lower Secondary, and Upper Secondary levels. Early childhood and tertiary are out of scope.
2. **No cross-border transfers.** Students and schools belong to one country context. No equivalency or migration logic between Kenyan, Rwandan, Tanzanian, or Ugandan certificates.
3. **Ignore legacy systems.** Only the current education system per country is modeled (e.g. Kenya CBC, not 8-4-4).
4. **English by default.** Language of instruction defaults to English (the education system default). Schools may override per school.
5. **Grade is by level; age is informational.** `students.date_of_birth` is stored but never used to determine grade level, stream assignment, or progression. Placement is driven entirely by school-configured grade years and levels.

## Alternatives Considered

| Alternative | Pros | Cons | Why rejected |
|-------------|------|------|--------------|
| Full spectrum (early childhood to tertiary) | Covers all education levels | Massive scope; different assessment and progression rules per level | Rejected — focus on primary-to-high-school first |
| Cross-border certificate equivalency | Enables student transfers between countries | Requires legal/regulatory research, equivalency mappings, and ongoing maintenance | Rejected — out of scope; no transfers for now |
| Support legacy and current systems simultaneously | Accommodates schools still on old systems | Doubles configuration complexity; confusing for users | Rejected — model current systems only |
| Age-based grade placement | Automates stream assignment | Inflexible; does not account for late entry, repetition, or acceleration | Rejected — age is informational only |

## Consequences

### Positive

- Clear boundaries simplify the data model and UI.
- No need to build equivalency engines or cross-country transfer workflows.
- Single education system per school reduces configuration complexity.

### Negative

- Schools on legacy systems cannot be modeled accurately until legacy support is added.
- Students cannot transfer between Kenyan and Ugandan schools within the system (by design).

### Neutral / Risks

- Age data is collected but unused; may feel like dead weight to users until future features need it.
- Scope may need to expand later; migrations will be required to add early childhood, tertiary, or legacy systems.

## Links

- Spec: `docs/specs/002-education-system-configuration.md`
- Parent ADR: `docs/adr/0007-table-driven-education-configuration.md`
- Local assessments: `docs/adr/0010-local-assessments-only.md`
