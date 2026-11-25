# ADR-0020: Education Configuration Migrations & Seeding

## Metadata

| Field | Value |
|-------|-------|
| id | ADR-0020 |
| title | Education Configuration Migrations & Seeding |
| status | proposed |
| date | 2026-11-25 |
| author | backend |
| deciders | backend, product |
| relates_to | SPEC-002, ADR-0007, ADR-0008, ADR-0009, ADR-0012, ADR-0013, ADR-0014 |

## Status

proposed

## Context

SPEC-002 defines ~19 education configuration tables (`countries`, `education_systems`, `education_levels`, `education_system_levels`, `education_system_grading_scales`, `education_system_grading_bands`, `education_system_subjects`, `education_system_term_structures`, `school_grading_scales`, `school_grading_bands`, `school_subjects`, `school_academic_years`, `school_academic_terms`, `school_grade_years`, `streams`, `teachers`, `students`, `enrollments`, `assessment_types`). None of these have migration files yet.

Additionally, the core `schools` table (migration `00000000000002_core_domain.sql`) is missing columns that SPEC-002 requires: `country_id`, `education_system_id`, `school_type`, `language_of_instruction`, `registration_number`. Without these, a school cannot reference `education_system_id` at creation time.

Per ADR-0007, seed data must be idempotent and run safely on every server start. Per ADR-0008, school creation must be wrapped in a transaction so all per-school seed rows insert together.

## Decision

Adopt a **three-part migration and seeding strategy**:

### 1. Core schools table migration (new migration `00000000000005_school_education_system`)

Add the missing columns to `schools`:

| Column | Type | Notes |
|--------|------|-------|
| `country_id` | UUID → `countries.id` | NULL until education system is selected |
| `education_system_id` | UUID → `education_systems.id` | NULL until education system is selected |
| `school_type` | TEXT | `public`, `private`, `boarding`, `day` |
| `language_of_instruction` | TEXT | Defaults to system default |
| `registration_number` | TEXT | Local registration identifier |

### 2. Education configuration migrations (`00000000000006_*` through `00000000000010_*`)

Create all education configuration tables from SPEC-002 in numbered migrations:

- `00000000000006_countries_systems_levels.sql` — `countries`, `education_systems`, `education_levels`
- `00000000000007_system_levels_grading.sql` — `education_system_levels`, `education_system_grading_scales`, `education_system_grading_bands`
- `00000000000008_subjects_terms.sql` — `education_system_subjects`, `education_system_term_structures`
- `00000000000009_school_config.sql` — `school_grading_scales`, `school_grading_bands`, `school_subjects`, `school_academic_years`, `school_academic_terms`, `school_grade_years`
- `00000000000010_domain.sql` — `streams`, `teachers`, `students`, `enrollments`, `assessment_types`, `assessments`, `assessment_results`

Plus new tables introduced by ADR-0012–0014:
- `attendance_records` (ADR-0012)
- `guardians`, `student_guardians` (ADR-0013)
- `teacher_qualifications` (ADR-0014)
- `consent_records` (ADR-0017)

### 3. Seeding strategy

**Global seed data** (countries, education systems, levels, grading scales, subjects, term structures) is seeded on server start via idempotent `INSERT ... ON CONFLICT DO NOTHING` statements. Not owned by any school.

**Per-school seed data** (school-level grading scales, subjects, academic years/terms, grade years) is generated inside a database transaction when a school is created with an `education_system_id`. Based on the education system's defaults. A school admin may later override these settings.

### 4. Seeding data

#### Countries

| id | name | code |
|----|------|------|
| seeded | Kenya | KE |
| seeded | Rwanda | RW |
| seeded | Tanzania | TZ |
| seeded | Uganda | UG |

#### Education Systems (current systems only, per ADR-0011)

| Country | System Name | years_primary | years_lower_secondary | years_upper_secondary | default_language |
|---------|-------------|---------------|----------------------|----------------------|------------------|
| Kenya | CBC | 6 | 3 | 3 | English |
| Rwanda | 9-Year Basic Education | 6 | 3 | 3 | English |
| Tanzania | 7-4-2-3 | 7 | 4 | 2 | English |
| Uganda | 7-4-2 | 7 | 4 | 2 | English |

#### Levels (shared across all systems)

| name | sort_order |
|------|------------|
| Primary | 1 |
| Lower Secondary | 2 |
| Upper Secondary | 3 |

#### Grading scales per system

**Kenya CBC** — competency-based scale (rubric):

| grade | scale_type | rubric_level | min_score | max_score | sort_order |
|-------|------------|--------------|-----------|-----------|------------|
| Exceeds Expectations | rubric | EE1 | 80 | 100 | 1 |
| Meets Expectations | rubric | ME1 | 65 | 79 | 2 |
| Approaching Expectations | rubric | AE1 | 50 | 64 | 3 |
| Below Expectations | rubric | BE1 | 0 | 49 | 4 |

**Rwanda, Tanzania, Uganda** — percentage-based scale:

| grade | scale_type | rubric_level | min_score | max_score | sort_order |
|-------|------------|--------------|-----------|-----------|------------|
| A | percentage | NULL | 80 | 100 | 1 |
| B | percentage | NULL | 70 | 79 | 2 |
| C | percentage | NULL | 60 | 69 | 3 |
| D | percentage | NULL | 50 | 59 | 4 |
| E | percentage | NULL | 40 | 49 | 5 |
| F | percentage | NULL | 0 | 39 | 6 |

#### Subjects per system (default, seeded to `education_system_subjects`)

**Kenya CBC — Primary (Level: Primary):**

| subject_code | subject_name | education_level | is_compulsory_default |
|--------------|--------------|-----------------|----------------------|
| MATH | Mathematics | Primary | true |
| ENG | English | Primary | true |
| KIS | Kiswahili | Primary | true |
| SCI | Science and Technology | Primary | true |
| SOC | Social Studies | Primary | true |
| CRE | Christian Religious Education | Primary | false |
| IRE | Islamic Religious Education | Primary | false |
| HRE | Hindu Religious Education | Primary | false |
| ART | Creative Arts | Primary | true |
| PHE | Physical and Health Education | Primary | true |
| AGR | Agriculture | Primary | true |

**Kenya CBC — Lower Secondary (Level: Lower Secondary):**

| subject_code | subject_name | education_level | is_compulsory_default |
|--------------|--------------|-----------------|----------------------|
| MATH | Mathematics | Lower Secondary | true |
| ENG | English | Lower Secondary | true |
| KIS | Kiswahili | Lower Secondary | true |
| SCI | Integrated Science | Lower Secondary | true |
| SOC | Social Studies | Lower Secondary | true |
| PRE | Pre-Technical and Pre-Career Education | Lower Secondary | true |
| BSE | Business Studies | Lower Secondary | true |
| AGR | Agriculture | Lower Secondary | true |
| HSC | Health Science | Lower Secondary | false |
| MUS | Music | Lower Secondary | false |

**Kenya CBC — Upper Secondary (Level: Upper Secondary):**

| subject_code | subject_name | education_level | is_compulsory_default |
|--------------|--------------|-----------------|----------------------|
| ENG | English | Upper Secondary | true |
| KIS | Kiswahili | Upper Secondary | true |
| MATH | Mathematics | Upper Secondary | false |
| BIO | Biology | Upper Secondary | false |
| CHE | Chemistry | Upper Secondary | false |
| PHY | Physics | Upper Secondary | false |
| HIS | History | Upper Secondary | false |
| GEO | Geography | Upper Secondary | false |
| CIV | Civics | Upper Secondary | false |
| BUS | Business Studies | Upper Secondary | false |

#### Term structures per system

| System | Terms | term_sort_order |
|--------|-------|-----------------|
| Kenya CBC | Term 1, Term 2, Term 3 | 1, 2, 3 |
| Rwanda | Term 1, Term 2, Term 3 | 1, 2, 3 |
| Tanzania | Term 1, Term 2 | 1, 2 |
| Uganda | Term 1, Term 2, Term 3 | 1, 2, 3 |

## Alternatives Considered

| Alternative | Pros | Cons | Why rejected |
|-------------|------|------|--------------|
| Seed via migration `Up` scripts | Runs once at migration time | Data changes require new migrations; hard to update without schema change | Rejected — seed data evolves independently of schema |
| Seed via separate CLI script | Separate concern from migrations | Easy to forget to run; not tied to server start | Rejected — ADR-0007 requires seeding on server start |
| Seed on server start (chosen) | Idempotent, always up-to-date, no manual step | Slightly longer startup time | Accepted — correctness outweighs startup cost |
| Seed per-school data outside transaction | Simpler code | Partial seed on failure leaves school in inconsistent state | Rejected — ADR-0008 requires transactional school creation |

## Consequences

### Positive

- All education configuration tables have migration files; schema matches SPEC-002.
- Seed data is idempotent and runs on every server start — safe for deploys and local development.
- School creation is transactional — no partial configuration.
- Adding a new country or changing a country's structure is a data change, not a code deploy.

### Negative

- More migration files to maintain (5 new migrations).
- Server startup includes seeding overhead (acceptable for current scale).
- Seed data for subjects/terms is a best-effort default; schools must verify and adjust for their local context.

### Neutral / Risks

- Seed data accuracy depends on current ministry of education curricula; must be reviewed before launch in each country.
- Migration numbering must not collide with existing migrations (00000000000001–00000000000004).
- If seeding fails mid-startup, the app should fail fast rather than start with incomplete configuration.

## Implementation Notes

1. **Migration order**: Run `00000000000005` before `00000000000006`–`00000000000010` because `schools.education_system_id` references `education_systems.id`.
2. **Goose down migrations**: Each migration must have a corresponding `Down` that drops tables/columns in reverse dependency order.
3. **Seed function**: Implement `SeedEducationConfig(db *sqlx.DB) error` called during `app.New(cfg)` startup, after migrations run.
4. **School creation transaction**: Wrap school insert + all per-school seed inserts (`school_grading_scales`, `school_grading_bands`, `school_subjects`, `school_academic_years`, `school_academic_terms`, `school_grade_years`) in a single `BEGIN`/`COMMIT`.
5. **UUIDv7**: All new tables use `uuidv7()` default, consistent with existing schema.

## Links

- Spec: `docs/specs/002-education-system-configuration.md`
- Parent ADR: `docs/adr/0007-table-driven-education-configuration.md`
- Per-school overrides: `docs/adr/0008-per-school-configuration-overrides.md`
- No JSONB: `docs/adr/0009-normalized-relational-configuration.md`
- Attendance: `docs/adr/0012-attendance-tracking.md`
- Guardians: `docs/adr/0013-guardian-management.md`
- Teacher qualifications: `docs/adr/0014-teacher-qualification-subject-assignment.md`
- Data protection: `docs/adr/0017-data-protection-child-privacy.md`
