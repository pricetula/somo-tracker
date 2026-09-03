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
2. Assert the expected schema artifacts exist (tables, columns, indexes,
   constraints, RLS policies, comments, etc.).
3. Optionally include a functional end-to-end check (e.g. RLS isolation via
   `SET LOCAL app.current_tenant_id`).

All migration tests are tagged `//go:build integration` and run against a
real PostgreSQL instance. Do **not** use mock pools — use the same `testdb.DB(t)`
pattern as the existing tests.

---

## General Rules

For all other backend conventions (error handling, API patterns, dependencies,
etc.) refer to the **root** `AGENTS.md` at the project root.

**Version:** 1.0.0 (September 2025)
