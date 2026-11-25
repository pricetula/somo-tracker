# ADR-0009: Normalized Relational Configuration (No JSONB)

## Metadata

| Field | Value |
|-------|-------|
| id | ADR-0009 |
| title | Normalized Relational Configuration (No JSONB) |
| status | proposed |
| date | 2026-11-25 |
| author | backend |
| deciders | backend |
| relates_to | SPEC-002, ADR-0007 |

## Status

proposed

## Context

Education configuration data (grading scales, bands, subjects, terms, grade years) is inherently relational. Grading scales contain multiple bands; each band has a grade label, minimum score, maximum score, and sort order. Bands support both percentage-based grading (e.g. "A", 70–100) and rubric-based competency grading (e.g. "EE1", "Exceeds Expectations", mapped to a percentage range such as 80–100). Subjects belong to levels and have compulsory flags. Terms belong to academic years and have dates.

Storing this as JSONB would lose referential integrity, make querying and reporting difficult, and prevent database-level validation of relationships.

## Decision

Adopt **strictly normalized relational tables** for all education configuration data:

- `education_system_grading_scales` + `education_system_grading_bands` (and school-level overrides)
- `education_system_subjects` + `school_subjects`
- `education_system_term_structures` + `school_academic_terms`
- `education_system_levels` + `school_grade_years`

JSONB is prohibited for configuration data. Every grade band, subject, term, and grade year is a row with explicit foreign keys.

## Alternatives Considered

| Alternative | Pros | Cons | Why rejected |
|-------------|------|------|--------------|
| JSONB configuration column | Flexible schema, fewer tables | No foreign keys, no constraints, hard to query individual bands, no type safety at DB level | Rejected — violates relational integrity |
| EAV (Entity-Attribute-Value) | Extremely flexible | Poor performance, complex queries, no referential integrity | Rejected — over-engineered for known configuration shape |
| Normalized relational rows (chosen) | Referential integrity, queryable, DB-validated, type-safe | More tables, more joins | Accepted — correctness and maintainability outweigh schema count |

## Consequences

### Positive

- Database enforces valid relationships (e.g. a grade band must belong to a grading scale).
- Reports can aggregate by grade, subject, term, or level with standard SQL.
- Schema evolution is explicit via migrations.

### Negative

- More tables to create and maintain.
- School creation requires inserting multiple related rows in a transaction.

### Neutral / Risks

- Migration complexity grows as configuration tables evolve; must be tested carefully.

## Links

- Spec: `docs/specs/002-education-system-configuration.md`
- Parent ADR: `docs/adr/0007-table-driven-education-configuration.md`
- Migrations & seeding: `docs/adr/0020-education-configuration-migrations-seeding.md`
