package members

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

func seedTenantSchoolUser(t *testing.T, pool *pgxpool.Pool) (tenantID, schoolID, userID string) {
	t.Helper()
	ctx := context.Background()
	tenantID = uuid.New().String()
	schoolID = uuid.New().String()
	userID = uuid.New().String()
	_, err := pool.Exec(ctx, `INSERT INTO tenants (id, name, slug, stytch_org_id) VALUES ($1, $2, $3, $4)`,
		tenantID, "Test", "slug-"+tenantID[:8], "stytch-"+tenantID[:8])
	require.NoError(t, err)
	_, err = pool.Exec(ctx, `INSERT INTO cbc_schools (id, tenant_id, name, county, sub_county, school_type) VALUES ($1, $2, $3, 'Nairobi', 'Westlands', 'Public')`,
		schoolID, tenantID, "Test School")
	require.NoError(t, err)
	_, err = pool.Exec(ctx, `INSERT INTO users (id, email, tenant_id, full_name) VALUES ($1, $2, $3, $4)`,
		userID, "user@test.com", tenantID, "Test User")
	require.NoError(t, err)
	return tenantID, schoolID, userID
}

func newRepo(pool *pgxpool.Pool) *PgRepository {
	return NewRepository(&database.Pools{PG: pool})
}

func TestPgRepository_ListByRole(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	ctx := context.Background()
	pool, cleanup := startPG(t)
	defer cleanup()
	applyMigration(t, pool, "000001_initial_schema.up.sql")

	tenantID, schoolID, userID := seedTenantSchoolUser(t, pool)
	repo := newRepo(pool)

	// Create membership
	_, err := pool.Exec(ctx, `INSERT INTO memberships (user_id, tenant_id, school_id, role, is_active) VALUES ($1, $2, $3, 'TEACHER', true)`,
		userID, tenantID, schoolID)
	require.NoError(t, err)

	members, total, err := repo.ListByRole(ctx, tenantID, schoolID, "TEACHER", 0, 50, "")
	require.NoError(t, err)
	require.Equal(t, 1, total)
	require.Len(t, members, 1)
	require.Equal(t, "Test User", members[0].FullName)
}

func TestPgRepository_ListByRole_Empty(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	ctx := context.Background()
	pool, cleanup := startPG(t)
	defer cleanup()
	applyMigration(t, pool, "000001_initial_schema.up.sql")

	tenantID, schoolID, _ := seedTenantSchoolUser(t, pool)
	repo := newRepo(pool)

	members, total, err := repo.ListByRole(ctx, tenantID, schoolID, "TEACHER", 0, 50, "")
	require.NoError(t, err)
	require.Equal(t, 0, total)
	require.Empty(t, members)
}

func TestPgRepository_ListByRole_Search(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	ctx := context.Background()
	pool, cleanup := startPG(t)
	defer cleanup()
	applyMigration(t, pool, "000001_initial_schema.up.sql")

	tenantID, schoolID, userID := seedTenantSchoolUser(t, pool)
	repo := newRepo(pool)

	_, err := pool.Exec(ctx, `INSERT INTO memberships (user_id, tenant_id, school_id, role, is_active) VALUES ($1, $2, $3, 'TEACHER', true)`,
		userID, tenantID, schoolID)
	require.NoError(t, err)

	members, total, err := repo.ListByRole(ctx, tenantID, schoolID, "TEACHER", 0, 50, "Test")
	require.NoError(t, err)
	require.Equal(t, 1, total)
	require.Len(t, members, 1)
}

func TestPgRepository_GetByID(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	ctx := context.Background()
	pool, cleanup := startPG(t)
	defer cleanup()
	applyMigration(t, pool, "000001_initial_schema.up.sql")

	tenantID, schoolID, userID := seedTenantSchoolUser(t, pool)
	repo := newRepo(pool)

	_, err := pool.Exec(ctx, `INSERT INTO memberships (user_id, tenant_id, school_id, role, is_active) VALUES ($1, $2, $3, 'TEACHER', true)`,
		userID, tenantID, schoolID)
	require.NoError(t, err)

	member, err := repo.GetByID(ctx, userID, tenantID, schoolID)
	require.NoError(t, err)
	require.NotNil(t, member)
	require.Equal(t, "Test User", member.FullName)
}

func TestPgRepository_GetByID_NotFound(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	ctx := context.Background()
	pool, cleanup := startPG(t)
	defer cleanup()
	applyMigration(t, pool, "000001_initial_schema.up.sql")

	tenantID, schoolID, _ := seedTenantSchoolUser(t, pool)
	repo := newRepo(pool)
	_, err := repo.GetByID(ctx, "missing_user", tenantID, schoolID)
	require.Error(t, err)
}
