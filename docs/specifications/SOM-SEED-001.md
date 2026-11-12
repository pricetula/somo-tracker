---
id: SOM-SEED-001
title: Reference Data Seeding
status: implemented
author: agent
date: 2026-11-11
updated: 2026-11-11
fr_ids: [BE-001, BE-005, BE-012, BE-033, BE-038, BE-039]
nf_ids: [BE-031, BE-032]
api_contract: no
migrations: yes (RLS policies for reference tables required)
generated_files: []
---

## 1. Summary

Reference data (countries, education systems, class levels) is created empty by migrations 03 but never populated. A new school cannot be created meaningfully until a country and education system exist, and subjects cannot be created until an education system + class level exist. This spec defines a post-migration seed step that populates these reference tables idempotently on server startup, behind a config flag.

## 2. Requirements

| FR/NFR ID | Rationale |
|-----------|-----------|
| BE-001 | Layout: seed code lives in `internal/platform/seed/` or `internal/platform/config/`; no `utils`, `helpers`, `common` |
| BE-005 | Manual DI via `New<Type>` constructors wired in `internal/app`; no DI frameworks |
| BE-012 | One transaction per seed call via `TxManager.Run` |
| BE-033 | Seeding controlled by typed config (env flag `SEED_REFERENCE_DATA`) |
| BE-038 | UUID defaults always `uuidv7()` for seeded rows |
| BE-039 | `updated_at` via DB trigger `trigger_set_timestamp` |
| BE-031 | Log seed events (request, counts) — no PII |
| BE-032 | Sentry: panics and 5xx only during seed |

## 3. Domain Model Affected

### Existing tables (migration 03) — populated by seed

| Table | Description |
|-------|-------------|
| `countries` | ISO 3166-1 alpha-2 countries (KE, TZ, UG, ...) |
| `education_systems` | Curriculum frameworks per country (Kenya CBC, ...) |
| `class_levels` | Grade levels per education system (Grade 1-6, Junior Secondary, ...) |

### RLS gap (migration 06)

`countries`, `education_systems`, and `class_levels` have RLS **enabled and forced** but **no RLS policies** — they are completely inaccessible (deny all). New policies are required so reference data is readable by all authenticated sessions.

### No new domain

Seed logic is platform infrastructure, not a business domain. It does **not** create `internal/seed/{handler,service,store/}`. It lives in `internal/platform/seed/` and is wired in `internal/app/app.go`.

## 4. Seed data

### Countries (idempotent, `ON CONFLICT (country_code) DO NOTHING`)

| country_name | country_code |
|--------------|--------------|
| Kenya | KE |
| Tanzania | TZ |
| Uganda | UG |

### Education systems (idempotent, `ON CONFLICT (country_id, system_name) DO NOTHING`)

| country_code | system_name |
|--------------|-------------|
| KE | Kenya CBC |
| TZ | Tanzania NECTA |
| UG | Uganda UNEB |

### Class levels (idempotent, `ON CONFLICT (education_system_id, sequence_index) DO NOTHING`)

| system_name | tier_stage | local_label | sequence_index |
|-------------|------------|-------------|----------------|
| Kenya CBC | Pre-Primary | PP1 | 1 |
| Kenya CBC | Pre-Primary | PP2 | 2 |
| Kenya CBC | Lower Primary | Grade 1 | 3 |
| Kenya CBC | Lower Primary | Grade 2 | 4 |
| Kenya CBC | Lower Primary | Grade 3 | 5 |
| Kenya CBC | Lower Primary | Grade 4 | 6 |
| Kenya CBC | Lower Primary | Grade 5 | 7 |
| Kenya CBC | Lower Primary | Grade 6 | 8 |
| Kenya CBC | Junior Secondary | Grade 7 | 9 |
| Kenya CBC | Junior Secondary | Grade 8 | 10 |
| Kenya CBC | Junior Secondary | Grade 9 | 11 |

## 5. Implementation Plan

### Backend

1. **Migration:** Add RLS policies for reference tables (countries, education_systems, class_levels) — `USING (true)` for SELECT so all authenticated sessions can read reference data.
   - File: `backend/migrations/00000000000007_reference_data_rls.sql`
2. **`internal/platform/seed/seed.go`** — `New(db)`, `Seed(ctx)` constructor + method:
   - Idempotent inserts via `INSERT ... ON CONFLICT DO NOTHING`
   - Wrapped in `TxManager.Run` for atomicity
   - Returns counts of inserted rows per table
   - Logs seed events (no PII)
3. **`internal/platform/seed/seed_test.go`** — table-driven tests for each seed SQL
4. **`internal/app/app.go`** — After `m.Up(ctx)`, conditionally call seed:
   - Read `SEED_REFERENCE_DATA` env flag (typed config, BE-033)
   - Default: `false` in production (opt-in), `true` in dev/test
   - Call `seed.Seed(ctx)` inside a transaction

### Config

- `SEED_REFERENCE_DATA` boolean in `platform/config/config.go`
- No new external dependencies

## 6. Business Rules

1. **Idempotent:** Re-running the seed must not duplicate rows. Uses `ON CONFLICT DO NOTHING` on the unique constraints:
   - `countries_country_code_uniq` on `countries(country_code)`
   - `education_systems_country_name_uniq` on `education_systems(country_id, system_name)`
   - `class_levels_sys_country_seq_uniq` on `class_levels(education_system_id, sequence_index)`
2. **Atomic:** All seed inserts run in one transaction via `TxManager.Run`. Partial failure rolls back everything.
3. **Opt-in:** Seeding does not run by default in production. Controlled by `SEED_REFERENCE_DATA` env flag.
4. **Readable reference data:** RLS policies on reference tables allow SELECT for all authenticated sessions (`USING (true)`). Writes remain restricted by existing tenant-scoped policies where applicable.
5. **No tenant scoping:** Reference tables (`countries`, `education_systems`, `class_levels`) have no `tenant_id` column. They are global reference data, not tenant-scoped.

## 7. Tests Required

### Backend

- `seed/seed_test.go`:
  - Idempotency: running seed twice produces no duplicates
  - Atomicity: partial failure rolls back all inserts
  - Each table's rows match expected seed data
  - Config flag: seed skipped when `SEED_REFERENCE_DATA=false`
- `app.go` integration: seed runs after migration on startup when flag enabled
- RLS: reference tables are readable by authenticated sessions after policy migration

## 8. Resolved decisions

| Question | Decision |
|----------|----------|
| Where does seed code live | `internal/platform/seed/` (platform capability, not a domain) |
| When does it run | After `m.Up(ctx)` in `app.go`, before HTTP server starts |
| Config control | `SEED_REFERENCE_DATA` env flag, default `false` in production |
| Atomicity | One `TxManager.Run` transaction for all seed inserts |
| Idempotency | `INSERT ... ON CONFLICT DO NOTHING` on existing unique constraints |
| RLS for reference tables | `USING (true)` SELECT policy — global readable reference data |
| Tenant scoping | Reference tables are not tenant-scoped (no `tenant_id` column) |
| Migration for RLS policies | New migration `00000000000007_reference_data_rls.sql` |

## 9. Open Questions

1. **Seed in production deployments** — Should `SEED_REFERENCE_DATA=true` be set in production, or should reference data be seeded via a separate one-off migration/job instead of app startup?
2. **Additional countries** — Should more countries be seeded beyond KE/TZ/UG, or should onboarding flow let schools select from a full ISO list?

## Links

- BE-001 layout rules
- BE-005 manual DI
- BE-012 TxManager
- BE-033 typed config
- backend/migrations/00000000000003_core_domain.sql — countries, education_systems, class_levels tables
- backend/migrations/00000000000006_rls_and_security.sql — RLS enabled (no policies) on reference tables
- backend/internal/app/app.go — wiring point after migration
- backend/internal/platform/config/config.go — config flag location