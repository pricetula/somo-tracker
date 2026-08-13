package middleware

import (
	"context"
	"fmt"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"

	"somotracker/backend/internal/database"
)

// TestWithTenantContext_RequestScopedRLS proves the full middleware wiring:
//   - WithTenantContext reads the resolved session's tenant,
//   - opens a request-scoped transaction with set_config('app.current_tenant_id'),
//   - stores it in Fiber locals so handlers passing c.Context() resolve it via
//     ctx.Value(database.TenantTxKey),
//   - repository code using database.FromContext(ctx, pool) therefore runs under
//     RLS for exactly the request's tenant.
//
// The app pool is a NON-superuser role so RLS actually applies (the schema's
// policies would otherwise be bypassed by the admin role).
func TestWithTenantContext_RequestScopedRLS(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	ctx := context.Background()
	pools, _, cleanup := setupTestPools(t)
	defer cleanup()

	// Fresh DB: apply base schema + seed.
	for _, f := range []string{"000001_initial_schema.up.sql", "000002_seed.up.sql"} {
		applyMigration(t, pools.PG, f)
	}

	tenantA := "aaaaaaaa-1111-1111-1111-111111111111"
	tenantB := "bbbbbbbb-2222-2222-2222-222222222222"
	_, err := pools.PG.Exec(ctx, `INSERT INTO tenants (id, name, slug, stytch_org_id) VALUES
		($1, 'Tenant A', 'mw-a', 'stytch-mw-a'), ($2, 'Tenant B', 'mw-b', 'stytch-mw-b')`,
		tenantA, tenantB)
	require.NoError(t, err)
	for _, s := range []struct{ id, tenant, code string }{
		{"aaaaaaaa-0000-0000-0000-000000000001", tenantA, "SCA"},
		{"bbbbbbbb-0000-0000-0000-000000000001", tenantB, "SCB"},
	} {
		_, err := pools.PG.Exec(ctx, `INSERT INTO cbc_schools (id, tenant_id, name, county, sub_county, school_type)
			VALUES ($1, $2, $3, 'C', 'S', 'Private')`, s.id, s.tenant, s.code)
		require.NoError(t, err)
	}

	// Non-superuser role for the app pool — RLS is active for it.
	_, err = pools.PG.Exec(ctx, `CREATE ROLE somo_mw_app LOGIN PASSWORD 'mw_pw' NOSUPERUSER NOBYPASSRLS`)
	require.NoError(t, err)
	_, err = pools.PG.Exec(ctx, `GRANT USAGE ON SCHEMA public TO somo_mw_app`)
	require.NoError(t, err)
	_, err = pools.PG.Exec(ctx, `GRANT ALL ON ALL TABLES IN SCHEMA public TO somo_mw_app`)
	require.NoError(t, err)
	_, err = pools.PG.Exec(ctx, `GRANT ALL ON ALL SEQUENCES IN SCHEMA public TO somo_mw_app`)
	require.NoError(t, err)

	host := pools.PG.Config().ConnConfig.Host
	port := pools.PG.Config().ConnConfig.Port
	appURL := fmt.Sprintf("postgres://somo_mw_app:mw_pw@%s:%d/somotracker_test?sslmode=disable", host, port)
	appPool, err := pgxpool.New(ctx, appURL)
	require.NoError(t, err)
	defer appPool.Close()

	appPools := &database.Pools{PG: appPool}

	app := fiber.New()

	// Simulate the session resolver having resolved a session (sets locals).
	sessionSetter := func(tenantID string) fiber.Handler {
		return func(c *fiber.Ctx) error {
			c.Locals("session", &SessionInfo{TenantID: tenantID})
			return c.Next()
		}
	}

	// Handler reads rows through the request-scoped executor exactly like a
	// repository would: ctx.Value(database.TenantTxKey) resolves the tx.
	handler := func(c *fiber.Ctx) error {
		exec := database.FromContext(c.Context(), appPool)
		var count int
		var name string
		if err := exec.QueryRow(c.Context(), `SELECT COUNT(*) FROM cbc_schools`).Scan(&count); err != nil {
			return err
		}
		if err := exec.QueryRow(c.Context(), `SELECT name FROM cbc_schools LIMIT 1`).Scan(&name); err != nil {
			return err
		}
		return c.JSON(fiber.Map{"count": count, "name": name})
	}

	app.Get("/schools-a", sessionSetter(tenantA), WithTenantContext(appPools), handler)
	app.Get("/schools-b", sessionSetter(tenantB), WithTenantContext(appPools), handler)

	t.Run("tenant_a_sees_only_its_own_school", func(t *testing.T) {
		resp, err := app.Test(httptest.NewRequest("GET", "/schools-a", nil))
		require.NoError(t, err)
		require.Equal(t, 200, resp.StatusCode)
		body := make([]byte, 1024)
		n, _ := resp.Body.Read(body)
		require.Contains(t, string(body[:n]), `"count":1`)
		require.Contains(t, string(body[:n]), `"name":"SCA"`)
	})

	t.Run("tenant_b_sees_only_its_own_school", func(t *testing.T) {
		resp, err := app.Test(httptest.NewRequest("GET", "/schools-b", nil))
		require.NoError(t, err)
		require.Equal(t, 200, resp.StatusCode)
		body := make([]byte, 1024)
		n, _ := resp.Body.Read(body)
		require.Contains(t, string(body[:n]), `"count":1`)
		require.Contains(t, string(body[:n]), `"name":"SCB"`)
	})

	t.Run("no_session_returns_zero_rows_safely", func(t *testing.T) {
		// Without a session there is no tenant tx and no GUC: RLS hides all
		// rows (safe by default). A COUNT sees zero; a single-row lookup would
		// surface pgx.ErrNoRows → repo maps it to ErrNotFound (404).
		anonHandler := func(c *fiber.Ctx) error {
			var count int
			if err := appPool.QueryRow(c.Context(), `SELECT COUNT(*) FROM cbc_schools`).Scan(&count); err != nil {
				return err
			}
			return c.JSON(fiber.Map{"count": count})
		}
		app.Get("/schools-anon", anonHandler) // no sessionSetter, no middleware
		resp, err := app.Test(httptest.NewRequest("GET", "/schools-anon", nil))
		require.NoError(t, err)
		require.Equal(t, 200, resp.StatusCode)
		body := make([]byte, 1024)
		n, _ := resp.Body.Read(body)
		require.Contains(t, string(body[:n]), `"count":0`)
	})
}
