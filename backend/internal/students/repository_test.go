package students

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
	"go.uber.org/zap"

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

func applyAllMigrations(t *testing.T, pool *pgxpool.Pool) {
	t.Helper()
	files, err := filepath.Glob(filepath.Join(migrationsDir(), "*.up.sql"))
	require.NoError(t, err, "glob migration files")
	for _, path := range files {
		sql, err := os.ReadFile(path)
		require.NoError(t, err, "read migration %s", path)
		_, err = pool.Exec(context.Background(), string(sql))
		require.NoError(t, err, "apply migration %s", path)
	}
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
	return NewRepository(&database.Pools{PG: pool}, zap.NewNop().Sugar())
}

func TestPgRepository_CreateAndGetByID(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	ctx := context.Background()
	pool, cleanup := startPG(t)
	defer cleanup()
	applyAllMigrations(t, pool)

	tenantID, schoolID := seedTenantSchool(t, pool)
	repo := newRepo(pool)

	student := &Student{
		TenantID:    tenantID,
		SchoolID:    schoolID,
		FullName:    "Test Student",
		Gender:      "M",
		DateOfBirth: strPtr("2012-01-01"),
	}

	id, err := repo.Create(ctx, student)
	require.NoError(t, err)
	require.NotEmpty(t, id)

	fetched, err := repo.GetByID(ctx, id, tenantID, schoolID)
	require.NoError(t, err)
	require.NotNil(t, fetched)
	require.Equal(t, "Test Student", fetched.FullName)
}

func TestPgRepository_GetByID_NotFound(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	ctx := context.Background()
	pool, cleanup := startPG(t)
	defer cleanup()
	applyAllMigrations(t, pool)

	tenantID, schoolID := seedTenantSchool(t, pool)
	repo := newRepo(pool)

	_, err := repo.GetByID(ctx, "missing_student", tenantID, schoolID)
	require.Error(t, err)
}

func TestPgRepository_Update(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	ctx := context.Background()
	pool, cleanup := startPG(t)
	defer cleanup()
	applyAllMigrations(t, pool)

	tenantID, schoolID := seedTenantSchool(t, pool)
	repo := newRepo(pool)

	id, err := repo.Create(ctx, &Student{
		TenantID: tenantID, SchoolID: schoolID, FullName: "Original", Gender: "M", DateOfBirth: strPtr("2012-01-01"),
	})
	require.NoError(t, err)

	// repo.Update takes full Student object
	student := &Student{ID: id, FullName: "Updated Name", Gender: "M", DateOfBirth: strPtr("2012-01-01")}
	err = repo.Update(ctx, student)
	require.NoError(t, err)

	fetched, err := repo.GetByID(ctx, id, tenantID, schoolID)
	require.NoError(t, err)
	require.Equal(t, "Updated Name", fetched.FullName)
}

func TestPgRepository_Delete(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	ctx := context.Background()
	pool, cleanup := startPG(t)
	defer cleanup()
	applyAllMigrations(t, pool)

	tenantID, schoolID := seedTenantSchool(t, pool)
	repo := newRepo(pool)

	id, err := repo.Create(ctx, &Student{
		TenantID: tenantID, SchoolID: schoolID, FullName: "To Delete", Gender: "M", DateOfBirth: strPtr("2012-01-01"),
	})
	require.NoError(t, err)

	err = repo.Delete(ctx, id, tenantID, schoolID)
	require.NoError(t, err)

	_, err = repo.GetByID(ctx, id, tenantID, schoolID)
	require.Error(t, err)
}

func strPtr(s string) *string { return &s }
