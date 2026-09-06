# Somotracker — Backend Agent Contract

## Golden Rule: Never Commit

The AI agent must **never** run `git add`, `git commit`, or `git push`.
All changes must be left as unstaged modifications in the working tree.
Only the user decides when and what to commit.

---

## Migration Testing Rule

Migration files live at `backend/db/migrations/`. They are embedded at build time
via `backend/db/embed_migrations.go`.

**Every new migration SQL file must have a corresponding integration test.**

The canonical test file is:

```
backend/internal/database/migrator_integration_test.go
```

Add a new `func TestMigrator_<FeatureName>(t *testing.T)` alongside the existing
functions. The test must:

1. Run `migrator.Up(ctx)` to apply the migration.
2. Assert the expected schema artifacts exist using simple catalog queries:
   - `SELECT table_name FROM information_schema.tables` — tables exist
   - `SELECT enumlabel FROM pg_enum` — enum values exist
   - `SELECT indexname FROM pg_indexes WHERE tablename = 'x'` — indexes exist
   - `SELECT 1 FROM information_schema.table_constraints WHERE ...` — unique constraints
   - `SELECT 1 FROM information_schema.triggers WHERE ...` — triggers exist
3. Optional: a functional end-to-end check (e.g. RLS isolation via
   `SET LOCAL app.current_tenant_id`, or a single INSERT + SELECT to verify FK works).

All migration tests are tagged `//go:build integration` and run against a
real PostgreSQL instance. Do **not** use mock pools — use the same `testdb.DB(t)`
pattern as the existing tests.

### Migration Test Pitfalls (Never Do These)

- **Complex multi-table joins** across `information_schema.referential_constraints` + `key_column_usage` — these can hang on large catalogs. Use `information_schema.table_constraints` alone for basic constraint existence checks.
- **Multi-step cascade transactions** — inserting into 5+ dependent tables then deleting and counting is a fragile pattern that can deadlock. Instead, assert the FK constraint exists (via `table_constraints`) and trust PostgreSQL to enforce cascade at runtime.
- **Variable reuse across transactions** — do not assign INSERT-returned IDs to variables that will be used in a subsequent transaction; the first transaction may be rolled back, leaving dangling references.
- **Parallel tests against shared DB** — avoid `t.Parallel()` in migration tests that share the same test database; tests should run sequentially to prevent connection pool exhaustion.

---

## General Rules

For all other backend conventions (error handling, API patterns, dependencies,
etc.) refer to the **root** `AGENTS.md` at the project root.

**Version:** 1.0.0 (September 2025)
