package billing

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"

	"somotracker/backend/internal/database"
)

func migrationsDir() string {
	_, filename, _, _ := runtime.Caller(0)
	dir := filepath.Dir(filename)
	for dir != "/" {
		if filepath.Base(dir) == "backend" {
			break
		}
		dir = filepath.Dir(dir)
	}
	return filepath.Join(dir, "internal", "database", "migrations")
}

func startPG(t *testing.T) (*pgxpool.Pool, func()) {
	t.Helper()
	ctx := context.Background()
	req := testcontainers.ContainerRequest{
		Image: "postgres:16-alpine",
		Env: map[string]string{
			"POSTGRES_DB":       "somotracker_test",
			"POSTGRES_USER":     "somo_admin",
			"POSTGRES_PASSWORD": "somo_secure_password",
		},
		ExposedPorts: []string{"5432/tcp"},
		WaitingFor: wait.ForAll(
			wait.ForLog("database system is ready to accept connections").WithOccurrence(2),
			wait.ForListeningPort("5432/tcp"),
		),
	}
	c, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	require.NoError(t, err)
	host, err := c.Host(ctx)
	require.NoError(t, err)
	port, err := c.MappedPort(ctx, "5432")
	require.NoError(t, err)
	dbURL := fmt.Sprintf("postgres://somo_admin:somo_secure_password@%s:%s/somotracker_test?sslmode=disable", host, port.Port())
	pool, err := pgxpool.New(ctx, dbURL)
	require.NoError(t, err)
	cleanup := func() { pool.Close(); _ = c.Terminate(ctx) }
	return pool, cleanup
}

func applyMigration(t *testing.T, pool *pgxpool.Pool, filename string) {
	t.Helper()
	path := filepath.Join(migrationsDir(), filename)
	sql, err := os.ReadFile(path)
	require.NoError(t, err, "read migration %s", filename)
	_, err = pool.Exec(context.Background(), string(sql))
	require.NoError(t, err, "apply migration %s", filename)
}

func seedTenantSchool(t *testing.T, pool *pgxpool.Pool) (tenantID, schoolID string) {
	t.Helper()
	ctx := context.Background()
	tenantID = uuid.New().String()
	schoolID = uuid.New().String()
	_, err := pool.Exec(ctx, `INSERT INTO tenants (id, name, slug, stytch_org_id) VALUES ($1, $2, $3, $4)`,
		tenantID, "Test", "slug-"+tenantID[:8], "stytch-"+tenantID[:8])
	require.NoError(t, err)
	_, err = pool.Exec(ctx, `INSERT INTO cbc_schools (id, tenant_id, name, county, sub_county, school_type) VALUES ($1, $2, $3, 'Nairobi', 'Westlands', 'Public')`,
		schoolID, tenantID, "Test School")
	require.NoError(t, err)
	return tenantID, schoolID
}

func newRepo(pool *pgxpool.Pool) *PgRepository {
	return NewRepository(&database.Pools{PG: pool})
}

func TestPgRepository_CreateAndListFeeCategories(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	ctx := context.Background()
	pool, cleanup := startPG(t)
	defer cleanup()
	applyMigration(t, pool, "000001_initial_schema.up.sql")

	tenantID, schoolID := seedTenantSchool(t, pool)
	repo := newRepo(pool)

	id, err := repo.CreateFeeCategory(ctx, tenantID, schoolID, "Tuition", true)
	require.NoError(t, err)
	require.NotEmpty(t, id)

	categories, err := repo.ListFeeCategories(ctx, tenantID, schoolID)
	require.NoError(t, err)
	require.Len(t, categories, 1)
	require.Equal(t, "Tuition", categories[0].Name)
	require.True(t, categories[0].IsMandatory)
}

func TestPgRepository_CreateFeeCategory_Duplicate(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	ctx := context.Background()
	pool, cleanup := startPG(t)
	defer cleanup()
	applyMigration(t, pool, "000001_initial_schema.up.sql")
	applyMigration(t, pool, "000017_add_fee_category_name_uniqueness.up.sql")

	tenantID, schoolID := seedTenantSchool(t, pool)
	repo := newRepo(pool)

	_, err := repo.CreateFeeCategory(ctx, tenantID, schoolID, "Tuition", true)
	require.NoError(t, err)

	_, err = repo.CreateFeeCategory(ctx, tenantID, schoolID, "Tuition", true)
	require.Error(t, err)
}

func TestPgRepository_UpdateFeeCategory(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	ctx := context.Background()
	pool, cleanup := startPG(t)
	defer cleanup()
	applyMigration(t, pool, "000001_initial_schema.up.sql")

	tenantID, schoolID := seedTenantSchool(t, pool)
	repo := newRepo(pool)

	id, err := repo.CreateFeeCategory(ctx, tenantID, schoolID, "Tuition", true)
	require.NoError(t, err)

	newName := "School Fees"
	err = repo.UpdateFeeCategory(ctx, id, tenantID, schoolID, &newName, nil)
	require.NoError(t, err)

	categories, err := repo.ListFeeCategories(ctx, tenantID, schoolID)
	require.NoError(t, err)
	require.Equal(t, "School Fees", categories[0].Name)
}

func TestPgRepository_DeleteFeeCategory(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	ctx := context.Background()
	pool, cleanup := startPG(t)
	defer cleanup()
	applyMigration(t, pool, "000001_initial_schema.up.sql")

	tenantID, schoolID := seedTenantSchool(t, pool)
	repo := newRepo(pool)

	id, err := repo.CreateFeeCategory(ctx, tenantID, schoolID, "Tuition", true)
	require.NoError(t, err)

	err = repo.DeleteFeeCategory(ctx, id, tenantID, schoolID)
	require.NoError(t, err)

	categories, err := repo.ListFeeCategories(ctx, tenantID, schoolID)
	require.NoError(t, err)
	require.Empty(t, categories)
}
