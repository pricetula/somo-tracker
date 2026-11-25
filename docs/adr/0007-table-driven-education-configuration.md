# ADR-0007: Table-Driven Multi-Country Education Configuration

## Metadata

| Field | Value |
|-------|-------|
| id | ADR-0007 |
| title | Table-Driven Multi-Country Education Configuration |
| status | proposed |
| date | 2026-11-25 |
| author | engineering |
| deciders | backend, product |
| relates_to | SPEC-002, ADR-0008, ADR-0009, ADR-0010, ADR-0011 |

## Status

proposed

## Context

Somotracker must support the current primary-to-high-school education structures of Kenya, Rwanda, Tanzania, and Uganda within a single codebase. Each country has a different number of years per level, different local grade names (e.g. Grade 1 vs P1 vs Standard 1), different subjects, and different term structures.

Hardcoding country-specific grades, terms, or subjects in code would require a deploy every time a country’s structure changes or a new country is added. It would also scatter country logic across the codebase, making it impossible to validate at the database level.

## Decision

Adopt a **table-driven configuration model** where everything that varies by country or school lives in relational tables:

1. **Global seed data** (countries, education systems) is seeded on server start and is not owned by any school.
2. **Per-school seed data** (curriculum, grade years, subjects, academic periods) is generated when a school is created, based on the education system the school selects.
3. A school admin can later override per-school settings without a code deploy.

## Alternatives Considered

| Alternative | Pros | Cons | Why rejected |
|-------------|------|------|--------------|
| Hardcoded country structs in code | Simple for a single country | Requires deploy for every structural change; no DB-level validation; does not scale to multiple countries | Rejected — multi-country support is a core requirement |
| JSONB configuration blobs | Flexible schema | Loses referential integrity, foreign keys, and queryability; harder to validate and report on | Rejected — see ADR-0009 |
| Separate database/schema per country | Strong isolation | Operational complexity; cannot share students or staff across countries (not needed anyway) | Overkill for current scope |

## Consequences

### Positive

- Adding a new country or changing a country’s structure is a data change, not a code deploy.
- Database constraints enforce valid grade/level/subject combinations.
- Generic UI can manage configuration rows without knowing country-specific details.

### Negative

- More tables and foreign keys to manage.
- Seeding logic required at server start and at school creation.

### Neutral / Risks

- Seeding must be idempotent and run safely on every server start.
- School creation must be wrapped in a transaction to ensure all per-school seed rows insert together.

## Links

- Spec: `docs/specs/002-education-system-configuration.md`
- Per-school overrides: `docs/adr/0008-per-school-configuration-overrides.md`
- No JSONB: `docs/adr/0009-normalized-relational-configuration.md`
- Teacher qualifications: `docs/adr/0014-teacher-qualification-subject-assignment.md`
- Migrations & seeding: `docs/adr/0020-education-configuration-migrations-seeding.md`
