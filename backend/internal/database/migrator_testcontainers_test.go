//go:build integration

package database

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"somotracker/backend/internal/testcontainers"
)

// TestMigrator_BasicIntegration verifies the migrator can apply all migrations successfully
// against a fresh testcontainers PostgreSQL instance.
func TestMigrator_BasicIntegration(t *testing.T) {
	testcontainers.SkipIfNoDocker(t)

	ctx := context.Background()
	tc := testcontainers.SetupPostgres(t)
	defer tc.Terminate(t)

	pool := tc.Pool()
	require.NotNil(t, pool)

	logger := zap.NewNop()
	migrator, err := NewMigrator(pool, logger)
	require.NoError(t, err)
	defer migrator.Close()

	err = migrator.Up(ctx)
	require.NoError(t, err, "migrator.Up should not fail")

	ver, dirty, err := migrator.CurrentVersion()
	require.NoError(t, err, "CurrentVersion should not fail")
	require.False(t, dirty, "schema should not be dirty after a clean run")
	require.GreaterOrEqual(t, ver, uint(1), "at least migration 1 should be applied")
}

// TestMigrator_Extensions verifies required PostgreSQL extensions are installed.
func TestMigrator_Extensions(t *testing.T) {
	testcontainers.SkipIfNoDocker(t)

	ctx := context.Background()
	tc := testcontainers.SetupPostgres(t)
	defer tc.Terminate(t)

	pool := tc.Pool()
	require.NotNil(t, pool)

	logger := zap.NewNop()
	migrator, err := NewMigrator(pool, logger)
	require.NoError(t, err)
	defer migrator.Close()

	err = migrator.Up(ctx)
	require.NoError(t, err, "migrator.Up should not fail")

	// Query pg_extension to assert the required extensions are present.
	rows, err := pool.Query(ctx, `
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
	// pg_uuidv7 is optional; it may or may not be present.
}

// TestMigrator_TenantsAndUsers verifies the base schema (tenants, users, RLS).
func TestMigrator_TenantsAndUsers(t *testing.T) {
	testcontainers.SkipIfNoDocker(t)

	ctx := context.Background()
	tc := testcontainers.SetupPostgres(t)
	defer tc.Terminate(t)

	pool := tc.Pool()

	logger := zap.NewNop()
	migrator, err := NewMigrator(pool, logger)
	require.NoError(t, err)
	defer migrator.Close()

	err = migrator.Up(ctx)
	require.NoError(t, err, "migrator.Up should not fail")

	// --- Tables exist ---
	tables := map[string]bool{
		"tenants": false,
		"users":   false,
	}
	rows, err := pool.Query(ctx, `
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
	rows.Close()
	require.True(t, tables["tenants"], "tenants table should exist")
	require.True(t, tables["users"], "users table should exist")

	// --- Foreign key + ON DELETE CASCADE on users.tenant_id ---
	var fkCount int
	err = pool.QueryRow(ctx, `
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
	err = pool.QueryRow(ctx, `
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
		"users_tenant_email_uniq",
	}
	for _, idx := range requiredIndexes {
		var exists bool
		err = pool.QueryRow(ctx, `
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
	err = pool.QueryRow(ctx, `
		SELECT relrowsecurity
		FROM pg_class
		WHERE relname = 'users' AND relnamespace = 'public'::regnamespace
	`).Scan(&rlsEnabled)
	require.NoError(t, err)
	require.True(t, rlsEnabled, "RLS should be enabled on users")

	// --- RLS policy exists and references app.current_tenant_id ---
	var policyName string
	err = pool.QueryRow(ctx, `
		SELECT policyname FROM pg_policies
		WHERE schemaname = 'public' AND tablename = 'users'
		  AND policyname = 'users_tenant_isolation'
	`).Scan(&policyName)
	require.NoError(t, err)
	require.Equal(t, "users_tenant_isolation", policyName,
		"users_tenant_isolation policy should be installed")

	var policyQual string
	err = pool.QueryRow(ctx, `
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
			err = pool.QueryRow(ctx, query, c.table).Scan(&dest)
		} else {
			err = pool.QueryRow(ctx, query, c.table, c.column).Scan(&dest)
		}
		require.NoError(t, err)
		require.NotEmptyf(t, dest,
			"comment should be set on %s.%s", c.table, c.column)
	}
}
