package cbcclasses

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

func seedTenantSchoolUser(t *testing.T, pool *pgxpool.Pool) (tenantID, schoolID, userID, yearID string) {
	t.Helper()
	ctx := context.Background()
	tenantID = uuid.New().String()
	schoolID = uuid.New().String()
	userID = uuid.New().String()
	yearID = uuid.New().String()
	_, err := pool.Exec(ctx, `INSERT INTO tenants (id, name, slug, stytch_org_id) VALUES ($1, $2, $3, $4)`,
		tenantID, "Test", "slug-"+tenantID[:8], "stytch-"+tenantID[:8])
	require.NoError(t, err)
	_, err = pool.Exec(ctx, `INSERT INTO cbc_schools (id, tenant_id, name, county, sub_county, school_type) VALUES ($1, $2, $3, 'Nairobi', 'Westlands', 'Public')`,
		schoolID, tenantID, "Test School")
	require.NoError(t, err)
	_, err = pool.Exec(ctx, `INSERT INTO users (id, email, tenant_id, full_name) VALUES ($1, $2, $3, $4)`,
		userID, "user@test.com", tenantID, "Test User")
	require.NoError(t, err)
	_, err = pool.Exec(ctx, `INSERT INTO academic_years (id, tenant_id, school_id, name, start_date, end_date, created_by, updated_by) VALUES ($1, $2, $3, '2026', '2026-01-01', '2026-12-31', $4, $4)`,
		yearID, tenantID, schoolID, userID)
	require.NoError(t, err)
	return tenantID, schoolID, userID, yearID
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

	tenantID, schoolID, _, yearID := seedTenantSchoolUser(t, pool)
	repo := newRepo(pool)

	// Create a stream first
	streamID := uuid.New().String()
	_, err := pool.Exec(ctx, `INSERT INTO cbc_streams (id, tenant_id, school_id, name) VALUES ($1, $2, $3, 'East')`,
		streamID, tenantID, schoolID)
	require.NoError(t, err)

	params := CreateClassParams{
		TenantID:       tenantID,
		SchoolID:       schoolID,
		GradeLevel:     "G4",
		StreamID:       streamID,
		AcademicYearID: yearID,
	}

	class, err := repo.Create(ctx, params)
	require.NoError(t, err)
	require.NotEmpty(t, class.ID)

	fetched, err := repo.GetByID(ctx, class.ID, tenantID, schoolID)
	require.NoError(t, err)
	require.Equal(t, "G4", fetched.GradeLevel)
}

func TestPgRepository_List(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	ctx := context.Background()
	pool, cleanup := startPG(t)
	defer cleanup()
	applyAllMigrations(t, pool)

	tenantID, schoolID, _, yearID := seedTenantSchoolUser(t, pool)
	repo := newRepo(pool)

	streamID := uuid.New().String()
	_, err := pool.Exec(ctx, `INSERT INTO cbc_streams (id, tenant_id, school_id, name) VALUES ($1, $2, $3, 'East')`,
		streamID, tenantID, schoolID)
	require.NoError(t, err)

	_, err = repo.Create(ctx, CreateClassParams{
		TenantID: tenantID, SchoolID: schoolID,
		GradeLevel: "G4", StreamID: streamID, AcademicYearID: yearID,
	})
	require.NoError(t, err)

	result, err := repo.List(ctx, ClassListFilter{
		TenantID:       tenantID,
		SchoolID:       schoolID,
		AcademicYearID: yearID,
		Page:           1,
		Limit:          50,
	})
	require.NoError(t, err)
	require.Len(t, result.Items, 1)
	require.Equal(t, 1, result.Total)
}

func TestPgRepository_GetByID_NotFound(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	ctx := context.Background()
	pool, cleanup := startPG(t)
	defer cleanup()
	applyAllMigrations(t, pool)

	repo := newRepo(pool)
	_, err := repo.GetByID(ctx, uuid.New().String(), "00000000-0000-0000-0000-000000000000", "00000000-0000-0000-0000-000000000000")
	require.Error(t, err)
}
