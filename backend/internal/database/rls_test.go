package database_test

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"
)

// TestRLSTenantIsolationEndToEnd proves the RLS + tenant-context wiring works
// for a non-superuser role:
//
//  1. Without app.current_tenant_id set, RLS-protected queries return ZERO rows
//     (safe-by-default).
//  2. With the GUC set inside a transaction (exactly what the WithTenantContext
//     middleware does via set_config(..., true)), the role sees only its own
//     tenant's rows.
//  3. Cross-tenant access is impossible: tenant A's context never sees tenant B.
//  4. The SECURITY DEFINER session-resolver (fn_resolve_session) still works
//     for the app role before the tenant is known.
//
// This is the runtime proof that the middleware → set_config → fn_current_tenant_id
// → RLS policy chain is functional, and that composite FKs reject cross-tenant
// references even when RLS is bypassed (superuser) or active (app role).
func TestRLSTenantIsolationEndToEnd(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	ctx := context.Background()
	pgC, hostPort, err := startPG(ctx)
	require.NoError(t, err)
	defer func() { _ = pgC.Terminate(ctx) }()

	adminURL := fmt.Sprintf("postgres://somo_admin:somo_secure_password@%s/somotracker_test?sslmode=disable", hostPort)
	pool, err := pgxpool.New(ctx, adminURL)
	require.NoError(t, err)
	defer pool.Close()

	// Apply migrations + seed (fresh DB).
	for _, f := range []string{"000001_initial_schema.up.sql", "000002_seed.up.sql"} {
		sql, err := os.ReadFile(filepath.Join(migrationsDir(), f))
		require.NoError(t, err, "read %s", f)
		_, err = pool.Exec(ctx, string(sql))
		require.NoError(t, err, "apply %s", f)
	}

	// ── Fixture: two tenants, each with a school ─────────────────────────
	tenantA := "11111111-1111-1111-1111-111111111111"
	tenantB := "22222222-2222-2222-2222-222222222222"
	schoolA := "aaaaaaaa-0000-0000-0000-000000000001"
	schoolB := "bbbbbbbb-0000-0000-0000-000000000001"

	_, err = pool.Exec(ctx, `INSERT INTO tenants (id, name, slug, stytch_org_id) VALUES
		($1, 'Tenant A', 'tenant-a-rls', 'stytch-a-rls'), ($2, 'Tenant B', 'tenant-b-rls', 'stytch-b-rls')`,
		tenantA, tenantB)
	require.NoError(t, err)

	for _, s := range []struct{ id, tenant, code string }{
		{schoolA, tenantA, "SCA"}, {schoolB, tenantB, "SCB"},
	} {
		_, err = pool.Exec(ctx, `INSERT INTO cbc_schools (id, tenant_id, name, county, sub_county, school_type)
			VALUES ($1, $2, $3, 'County', 'Sub', 'Private')`, s.id, s.tenant, s.code)
		require.NoError(t, err)
	}

	// ── Create a non-superuser app role (RLS actually applies) ───────────
	_, err = pool.Exec(ctx, `CREATE ROLE somo_app_rls LOGIN PASSWORD 'rls_pw' NOSUPERUSER NOBYPASSRLS`)
	require.NoError(t, err)
	_, err = pool.Exec(ctx, `GRANT USAGE ON SCHEMA public TO somo_app_rls`)
	require.NoError(t, err)
	_, err = pool.Exec(ctx, `GRANT ALL ON ALL TABLES IN SCHEMA public TO somo_app_rls`)
	require.NoError(t, err)
	_, err = pool.Exec(ctx, `GRANT ALL ON ALL SEQUENCES IN SCHEMA public TO somo_app_rls`)
	require.NoError(t, err)

	appURL := fmt.Sprintf("postgres://somo_app_rls:rls_pw@%s/somotracker_test?sslmode=disable", hostPort)
	appPool, err := pgxpool.New(ctx, appURL)
	require.NoError(t, err)
	defer appPool.Close()

	// ── 1. No tenant context → zero rows (safe by default) ───────────────
	var count int
	err = appPool.QueryRow(ctx, `SELECT COUNT(*) FROM cbc_schools`).Scan(&count)
	require.NoError(t, err)
	require.Zero(t, count, "RLS must hide all rows without tenant context")

	// ── 2. Tenant A context → only tenant A's school ─────────────────────
	txA, err := appPool.Begin(ctx)
	require.NoError(t, err)
	_, err = txA.Exec(ctx, "SELECT set_config('app.current_tenant_id', $1, true)", tenantA)
	require.NoError(t, err)
	err = txA.QueryRow(ctx, `SELECT COUNT(*) FROM cbc_schools`).Scan(&count)
	require.NoError(t, err)
	require.Equal(t, 1, count, "tenant A must see exactly one school")
	var name string
	err = txA.QueryRow(ctx, `SELECT name FROM cbc_schools`).Scan(&name)
	require.NoError(t, err)
	require.Equal(t, "SCA", name)
	// RLS also enforces the WITH CHECK on writes: tenant A cannot insert a
	// school claiming to belong to tenant B.
	_, err = txA.Exec(ctx, `INSERT INTO cbc_schools (id, tenant_id, name, county, sub_county, school_type)
		VALUES ($1, $2, 'evil', 'C', 'S', 'Private')`,
		"cccccccc-0000-0000-0000-000000000001", tenantB)
	require.Error(t, err, "tenant A must not be able to insert tenant B's school (RLS WITH CHECK)")
	_ = txA.Rollback(ctx)

	// ── 3. Tenant B context is fully isolated from tenant A ──────────────
	txB, err := appPool.Begin(ctx)
	require.NoError(t, err)
	_, err = txB.Exec(ctx, "SELECT set_config('app.current_tenant_id', $1, true)", tenantB)
	require.NoError(t, err)
	err = txB.QueryRow(ctx, `SELECT COUNT(*) FROM cbc_schools`).Scan(&count)
	require.NoError(t, err)
	require.Equal(t, 1, count)
	err = txB.QueryRow(ctx, `SELECT name FROM cbc_schools`).Scan(&name)
	require.NoError(t, err)
	require.Equal(t, "SCB", name)
	_ = txB.Rollback(ctx)

	// ── 4. SECURITY DEFINER session resolver works pre-tenant for the app role ──
	// Insert a session (as admin) and resolve it as the app role WITHOUT any
	// tenant context — fn_resolve_session bypasses RLS by design.
	userA := "dddddddd-0000-0000-0000-000000000001"
	_, err = pool.Exec(ctx, `INSERT INTO users (id, email, tenant_id, full_name) VALUES ($1, $2, $3, 'User A')`,
		userA, "rls-a@test.com", tenantA)
	require.NoError(t, err)
	tokenHash := "abcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789"
	_, err = pool.Exec(ctx, `INSERT INTO sessions (user_id, tenant_id, stytch_member_id, stytch_org_id, token_hash, expires_at)
		VALUES ($1, $2, 'mem-a', 'stytch-a-rls', $3, NOW() + INTERVAL '1 day')`,
		userA, tenantA, tokenHash)
	require.NoError(t, err)

	var resolvedTenant string
	err = appPool.QueryRow(ctx, `SELECT tenant_id FROM fn_resolve_session($1)`, tokenHash).Scan(&resolvedTenant)
	require.NoError(t, err, "fn_resolve_session must work for the app role without tenant context")
	require.Equal(t, tenantA, resolvedTenant)

	// ── 5. Composite FK rejects cross-tenant references even as superuser ──
	// (RLS is bypassed for superusers, so the composite FK is the backstop.)
	userB := "eeeeeeee-0000-0000-0000-000000000001"
	_, err = pool.Exec(ctx, `INSERT INTO users (id, email, tenant_id, full_name) VALUES ($1, $2, $3, 'User B')`,
		userB, "rls-b@test.com", tenantB)
	require.NoError(t, err)
	_, err = pool.Exec(ctx, `INSERT INTO cbc_parents (id, tenant_id, user_id, phone_number)
		VALUES ($1, $2, $3, '0700000000')`, "ffffffff-0000-0000-0000-000000000001", tenantB, userB)
	require.NoError(t, err)

	// Attempt: tenant A's student linked to tenant B's parent (cross-tenant).
	studentA := "aaaaaaaa-0000-0000-0000-000000000002"
	_, err = pool.Exec(ctx, `INSERT INTO cbc_students (id, tenant_id, school_id, full_name, gender)
		VALUES ($1, $2, $3, 'Student A', 'F')`, studentA, tenantA, schoolA)
	require.NoError(t, err)

	_, err = pool.Exec(ctx, `INSERT INTO cbc_student_parents (tenant_id, student_id, parent_id)
		VALUES ($1, $2, $3)`, tenantA, studentA, "ffffffff-0000-0000-0000-000000000001")
	require.Error(t, err, "composite FK must reject a tenant A student linked to a tenant B parent")
	require.Contains(t, err.Error(), "fk_cbc_student_parents_tenant_parent")
}
