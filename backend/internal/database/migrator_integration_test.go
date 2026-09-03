//go:build integration

package database

import (
	"context"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"somotracker/backend/internal/testdb"
)

func TestMigrator_Integration(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	dsn := "postgres://somo_admin:somo_secure_password@127.0.0.1:5433/somotracker_test?sslmode=disable"
	pool, err := pgxpool.New(ctx, dsn)
	require.NoError(t, err, "pgxpool.New should not fail")
	t.Cleanup(pool.Close)

	logger := zap.NewNop()
	migrator, err := NewMigrator(pool, logger)
	require.NoError(t, err, "NewMigrator should not fail")
	t.Cleanup(func() { _ = migrator.Close() })

	err = migrator.Up(ctx)
	require.NoError(t, err, "migrator.Up should not fail")

	ver, dirty, err := migrator.CurrentVersion()
	require.NoError(t, err, "CurrentVersion should not fail")
	require.False(t, dirty, "schema should not be dirty after a clean run")
	require.GreaterOrEqual(t, ver, uint(1), "at least migration 1 should be applied")
}

func TestMigrator_Extensions(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	// Use the shared testdb pool for queries. The migrator owns the connection
	// for DDL; the shared pool connection is used only for reads.
	db := testdb.DB(t)

	dsn := "postgres://somo_admin:somo_secure_password@127.0.0.1:5433/somotracker_test?sslmode=disable"
	pool, err := pgxpool.New(ctx, dsn)
	require.NoError(t, err)
	t.Cleanup(pool.Close)

	logger := zap.NewNop()
	migrator, err := NewMigrator(pool, logger)
	require.NoError(t, err)
	t.Cleanup(func() { _ = migrator.Close() })

	err = migrator.Up(ctx)
	require.NoError(t, err, "migrator.Up should not fail")

	// Query pg_extension to assert the required extensions are present.
	// pg_uuidv7 is optional; uuid-ossp and pgcrypto are required.
	rows, err := db.QueryContext(ctx, `
		SELECT extname FROM pg_extension
		WHERE extname IN ('pgcrypto', 'uuid-ossp', 'pg_uuidv7')
		ORDER BY extname
	`)
	require.NoError(t, err, "querying pg_extension should not fail")
	defer rows.Close()

	found := make(map[string]bool)
	for rows.Next() {
		var name string
		require.NoError(t, rows.Scan(&name))
		found[name] = true
	}
	require.NoError(t, rows.Err())

	require.True(t, found["pgcrypto"], "pgcrypto extension should be active")
	require.True(t, found["uuid-ossp"], "uuid-ossp extension should be active")
	// pg_uuidv7 is tested separately; it may or may not be present.
}

func TestMigrator_TenantsAndUsers(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	// Use the shared testdb pool for queries. The migrator owns the connection
	// for DDL; the shared pool connection is used only for reads.
	db := testdb.DB(t)

	dsn := "postgres://somo_admin:somo_secure_password@127.0.0.1:5433/somotracker_test?sslmode=disable"
	pool, err := pgxpool.New(ctx, dsn)
	require.NoError(t, err)
	t.Cleanup(pool.Close)

	logger := zap.NewNop()
	migrator, err := NewMigrator(pool, logger)
	require.NoError(t, err)
	t.Cleanup(func() { _ = migrator.Close() })

	err = migrator.Up(ctx)
	require.NoError(t, err, "migrator.Up should not fail")

	// --- Tables exist ---
	tables := map[string]bool{
		"tenants": false,
		"users":   false,
	}
	rows, err := db.QueryContext(ctx, `
		SELECT table_name FROM information_schema.tables
		WHERE table_schema = 'public'
		  AND table_name IN ('tenants', 'users')
	`)
	require.NoError(t, err)
	for rows.Next() {
		var name string
		require.NoError(t, rows.Scan(&name))
		tables[name] = true
	}
	require.NoError(t, rows.Close())
	require.True(t, tables["tenants"], "tenants table should exist")
	require.True(t, tables["users"], "users table should exist")

	// --- Foreign key + ON DELETE CASCADE on users.tenant_id ---
	var fkCount int
	err = db.QueryRowContext(ctx, `
		SELECT COUNT(*)
		FROM information_schema.table_constraints tc
		JOIN information_schema.referential_constraints rc
		  ON rc.constraint_name = tc.constraint_name
		WHERE tc.table_name = 'users'
		  AND tc.constraint_type = 'FOREIGN KEY'
		  AND tc.table_schema = 'public'
	`).Scan(&fkCount)
	require.NoError(t, err)
	require.Equal(t, 1, fkCount, "users should have exactly one foreign key")

	var cascadeCount int
	err = db.QueryRowContext(ctx, `
		SELECT COUNT(*)
		FROM information_schema.referential_constraints rc
		JOIN information_schema.table_constraints tc
		  ON tc.constraint_name = rc.constraint_name
		WHERE tc.table_name = 'users'
		  AND tc.constraint_type = 'FOREIGN KEY'
		  AND rc.delete_rule = 'CASCADE'
		  AND tc.table_schema = 'public'
	`).Scan(&cascadeCount)
	require.NoError(t, err)
	require.Equal(t, 1, cascadeCount, "users.tenant_id FK should use ON DELETE CASCADE")

	// --- Required indexes exist ---
	requiredIndexes := []string{
		"users_tenant_id_idx",
		"users_email_idx",
	}
	for _, idx := range requiredIndexes {
		var exists bool
		err = db.QueryRowContext(ctx, `
			SELECT EXISTS (
				SELECT 1 FROM pg_indexes
				WHERE schemaname = 'public' AND indexname = $1
			)
		`, idx).Scan(&exists)
		require.NoError(t, err)
		require.Truef(t, exists, "index %q should exist", idx)
	}

	// --- RLS is enabled on users ---
	var rlsEnabled bool
	err = db.QueryRowContext(ctx, `
		SELECT relrowsecurity
		FROM pg_class
		WHERE relname = 'users' AND relnamespace = 'public'::regnamespace
	`).Scan(&rlsEnabled)
	require.NoError(t, err)
	require.True(t, rlsEnabled, "RLS should be enabled on users")

	// --- RLS policy exists and references app.current_tenant_id ---
	var policyName string
	err = db.QueryRowContext(ctx, `
		SELECT policyname FROM pg_policies
		WHERE schemaname = 'public' AND tablename = 'users'
		  AND policyname = 'users_tenant_isolation'
	`).Scan(&policyName)
	require.NoError(t, err)
	require.Equal(t, "users_tenant_isolation", policyName,
		"users_tenant_isolation policy should be installed")

	var policyQual string
	err = db.QueryRowContext(ctx, `
		SELECT COALESCE(qual, '') FROM pg_policies
		WHERE schemaname = 'public' AND tablename = 'users'
		  AND policyname = 'users_tenant_isolation'
	`).Scan(&policyQual)
	require.NoError(t, err)
	require.Contains(t, policyQual, "current_setting",
		"policy qual should reference current_setting()")
	require.Contains(t, policyQual, "app.current_tenant_id",
		"policy qual should reference app.current_tenant_id")

	// --- Table and column comments are present ---
	requiredComments := []struct {
		table  string
		column string
	}{
		{"tenants", ""},
		{"users", ""},
		{"tenants", "stytch_org_id"},
		{"users", "tenant_id"},
		{"users", "external_auth_id"},
	}
	for _, c := range requiredComments {
		var query string
		var dest string
		if c.column == "" {
			query = `SELECT obj_description(c.oid)
			         FROM pg_class c
			         WHERE c.relname = $1 AND c.relnamespace = 'public'::regnamespace`
		} else {
			query = `SELECT col_description(c.oid, a.attnum)
			         FROM pg_class c
			         JOIN pg_attribute a ON a.attrelid = c.oid
			         WHERE c.relname = $1 AND c.relnamespace = 'public'::regnamespace
			           AND a.attname = $2`
		}
		if c.column == "" {
			err = db.QueryRowContext(ctx, query, c.table).Scan(&dest)
		} else {
			err = db.QueryRowContext(ctx, query, c.table, c.column).Scan(&dest)
		}
		require.NoError(t, err)
		require.NotEmptyf(t, dest,
			"comment should be set on %s.%s", c.table, c.column)
	}

	// --- Functional check: RLS isolation works end-to-end ---
	// 1. Insert a tenant and two users under different tenants.
	// 2. Set app.current_tenant_id to tenant A's UUID.
	// 3. Verify only tenant A's user is visible.
	//
	// We use SET LOCAL inside a transaction so the GUC is scoped and never
	// leaks into the shared pool.
	tx, err := db.BeginTx(ctx, nil)
	require.NoError(t, err)
	defer func() { _ = tx.Rollback() }()

	// Create two tenants.
	var tenantA, tenantB string
	require.NoError(t, tx.QueryRowContext(ctx, `
		INSERT INTO tenants (name, slug, stytch_org_id)
		VALUES ('A', 'a-' || gen_random_uuid()::text, 'org-a-' || gen_random_uuid()::text)
		RETURNING id::text
	`).Scan(&tenantA))
	require.NoError(t, tx.QueryRowContext(ctx, `
		INSERT INTO tenants (name, slug, stytch_org_id)
		VALUES ('B', 'b-' || gen_random_uuid()::text, 'org-b-' || gen_random_uuid()::text)
		RETURNING id::text
	`).Scan(&tenantB))

	// Insert a user into each tenant.
	_, err = tx.ExecContext(ctx, `
		INSERT INTO users (email, tenant_id) VALUES ($1, $2)
	`, "alice@a.test", tenantA)
	require.NoError(t, err)
	_, err = tx.ExecContext(ctx, `
		INSERT INTO users (email, tenant_id) VALUES ($1, $2)
	`, "bob@b.test", tenantB)
	require.NoError(t, err)

	// Scope this transaction to tenant A.
	_, err = tx.ExecContext(ctx,
		`SET LOCAL app.current_tenant_id = $1`, tenantA)
	require.NoError(t, err)

	// RLS should now expose only tenant A's user.
	rows, err = tx.QueryContext(ctx,
		`SELECT email FROM users ORDER BY email`)
	require.NoError(t, err)
	var visibleEmails []string
	for rows.Next() {
		var email string
		require.NoError(t, rows.Scan(&email))
		visibleEmails = append(visibleEmails, email)
	}
	require.NoError(t, rows.Close())
	require.Equal(t, []string{"alice@a.test"}, visibleEmails,
		"RLS should only expose rows whose tenant_id matches the session GUC")

	// Cross-tenant INSERT must be rejected by the WITH CHECK clause of the
	// policy (USING applies to all ops in FOR ALL; PostgreSQL uses USING as
	// both visibility and WITH CHECK for FOR ALL).
	_, err = tx.ExecContext(ctx, `
		INSERT INTO users (email, tenant_id) VALUES ($1, $2)
	`, "evil@x.test", tenantB)
	require.Error(t, err,
		"inserting into a different tenant must be rejected by RLS")

	// ON DELETE CASCADE: deleting tenant A should remove its users.
	_, err = tx.ExecContext(ctx,
		`DELETE FROM tenants WHERE id::text = $1`, tenantA)
	require.NoError(t, err)

	var remainingA int
	require.NoError(t, tx.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM users WHERE tenant_id::text = $1`, tenantA,
	).Scan(&remainingA))
	require.Equal(t, 0, remainingA,
		"ON DELETE CASCADE should remove users when their tenant is deleted")
}
