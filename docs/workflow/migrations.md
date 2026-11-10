# Migrations

## Overview
Database migrations live in `backend/migrations/` and are applied with Goose. All migrations must be document-led, pass `make compliance`, and be verified by `backend/internal/database/migrator_integration_test.go`.

## Naming & Ordering
* Filename: `<YYYYMMDDHHMMSS>_<slug>.sql`
* Use Goose directives:
  ```sql
  -- +goose Up
  ...
  -- +goose Down
  ...
  ```
* Never edit a migration that has been merged to `develop` or `main`. Create a new migration.

## Standards
* **UUIDs**: Always `DEFAULT uuidv7()` for new rows. Do not use `gen_random_uuid()`.
* **Timestamps**: `created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()`, `updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()`.
* **updated_at trigger**: Use the canonical `trigger_set_timestamp()` function. Create trigger `BEFORE UPDATE` per table. Do not add `updated_at` to tables that will never be updated.
* **Soft delete**: Add `deleted_at TIMESTAMPTZ` + partial index `WHERE deleted_at IS NULL`. Prefer soft delete per BE-041.
* **RLS**: Every multi-tenant table must `ENABLE ROW LEVEL SECURITY` and have an isolation policy per BE-040. Cross-tenant queries must return identical 404.
* **Enums**: Use `CREATE TYPE ... AS ENUM` for status/check fields. Do not use `TEXT CHECK`.
* **FKs**: Reference tables use `ON DELETE RESTRICT`. Domain tables cascade from tenants/schools where appropriate.
* **Indexes**: Create indexes for FK columns and frequent lookup pairs. Use partial indexes for `is_current` / `deleted_at`.
* **Extensions**: `CREATE EXTENSION IF NOT EXISTS "pgcrypto";` in init. Ensure `pg_uuidv7` is available for `uuidv7()`.

## Template
```sql
-- +goose Up
CREATE TABLE my_table (
    id UUID PRIMARY KEY DEFAULT uuidv7(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);

CREATE INDEX my_table_tenant_id_idx ON my_table (tenant_id);
CREATE INDEX my_table_deleted_at_idx ON my_table (deleted_at) WHERE deleted_at IS NULL;

ALTER TABLE my_table ENABLE ROW LEVEL SECURITY;
CREATE POLICY my_table_isolate ON my_table USING (EXISTS (SELECT 1 FROM tenants WHERE tenants.id = my_table.tenant_id));

CREATE TRIGGER trigger_set_timestamp BEFORE UPDATE ON my_table FOR EACH ROW EXECUTE FUNCTION trigger_set_timestamp();

-- +goose Down
DROP TRIGGER IF EXISTS trigger_set_timestamp ON my_table;
DROP POLICY IF EXISTS my_table_isolate ON my_table;
ALTER TABLE my_table DISABLE ROW LEVEL SECURITY;
DROP INDEX IF EXISTS my_table_deleted_at_idx;
DROP INDEX IF EXISTS my_table_tenant_id_idx;
DROP TABLE IF EXISTS my_table;
```

## Testing
* Every new migration must be verified by `migrator_integration_test.go`.
* Test must assert table, enum, index, constraint existence via `information_schema`.
* Run `go test -tags integration ./backend/internal/database -run TestMigration_AllMigrationsPass`.
* Migrations must pass `make compliance` and lefthook pre-commit.

## Checklist Before Commit
- [ ] Spec is approved in `docs/`
- [ ] Migration follows naming, uuidv7, updated_at, soft delete, RLS
- [ ] Down migration is reversible
- [ ] Integration test passes
- [ ] No new domain tables without approved spec
