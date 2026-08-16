package teacherworkloadsummaries

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

// ── Test helpers ─────────────────────────────────────────────────────────

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

func seedTenantSchoolUser(t *testing.T, pool *pgxpool.Pool) (tenantID, schoolID, userID string) {
	t.Helper()
	ctx := context.Background()
	tenantID = uuid.New().String()
	schoolID = uuid.New().String()
	userID = uuid.New().String()
	_, err := pool.Exec(ctx, `INSERT INTO tenants (id, name, slug, stytch_org_id) VALUES ($1, $2, $3, $4)`,
		tenantID, "Test Tenant", "slug-"+tenantID[:8], "stytch-"+tenantID[:8])
	require.NoError(t, err)
	_, err = pool.Exec(ctx, `INSERT INTO cbc_schools (id, tenant_id, name, county, sub_county, school_type) VALUES ($1, $2, $3, 'Nairobi', 'Westlands', 'Public')`,
		schoolID, tenantID, "Test School")
	require.NoError(t, err)
	_, err = pool.Exec(ctx, `INSERT INTO users (id, email, tenant_id, full_name) VALUES ($1, $2, $3, $4)`,
		userID, "teacher@test.com", tenantID, "Test Teacher")
	require.NoError(t, err)
	return tenantID, schoolID, userID
}

func seedAcademicYear(t *testing.T, pool *pgxpool.Pool, tenantID, schoolID string) (yearID string) {
	t.Helper()
	ctx := context.Background()
	yearID = uuid.New().String()
	userID := uuid.New().String()
	_, err := pool.Exec(ctx, `INSERT INTO users (id, email, tenant_id, full_name) VALUES ($1, $2, $3, $4) ON CONFLICT (id) DO NOTHING`,
		userID, "sys-academicyear@test.com", tenantID, "Acad Year System")
	require.NoError(t, err)
	_, err = pool.Exec(ctx, `INSERT INTO academic_years (id, tenant_id, school_id, name, start_date, end_date, created_by, updated_by) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
		yearID, tenantID, schoolID, "2025", "2025-01-01", "2025-12-31", userID, userID)
	require.NoError(t, err)
	return yearID
}

func newRepo(pool *pgxpool.Pool) Repository {
	return NewRepository(&database.Pools{PG: pool})
}

// ── Tests ────────────────────────────────────────────────────────────────

func TestPgRepository_ListByTeacher(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	ctx := context.Background()
	pool, cleanup := startPG(t)
	defer cleanup()
	applyAllMigrations(t, pool)

	tenantID, schoolID, userID := seedTenantSchoolUser(t, pool)
	yearID := seedAcademicYear(t, pool, tenantID, schoolID)

	repo := newRepo(pool)

	// Directly insert a workload summary row
	summaryID := uuid.New().String()
	_, err := pool.Exec(ctx, `
		INSERT INTO teacher_workload_summaries (id, tenant_id, school_id, user_id, academic_year_id,
			total_assigned_periods, unique_subjects, classes_taught,
			utilization_percentage, is_overcapacity, last_refreshed_at)
		VALUES ($1, $2, $3, $4, $5, 30, 4, 3, 0.85, false, NOW())
	`, summaryID, tenantID, schoolID, userID, yearID)
	require.NoError(t, err)

	// List by teacher
	resp, err := repo.ListByTeacher(ctx, tenantID, schoolID, userID, yearID)
	require.NoError(t, err)
	require.Equal(t, 1, resp.Total)
	require.Len(t, resp.Items, 1)
	require.Equal(t, 30, resp.Items[0].TotalAssignedPeriods)
	require.Equal(t, 4, resp.Items[0].UniqueSubjects)
	require.Equal(t, 3, resp.Items[0].ClassesTaught)
	require.NotNil(t, resp.Items[0].UtilizationPercentage)
	require.InDelta(t, 0.85, *resp.Items[0].UtilizationPercentage, 0.01)
	require.False(t, resp.Items[0].IsOvercapacity)
}

func TestPgRepository_ListByTeacher_NotFound(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	ctx := context.Background()
	pool, cleanup := startPG(t)
	defer cleanup()
	applyAllMigrations(t, pool)

	tenantID, schoolID, userID := seedTenantSchoolUser(t, pool)

	repo := newRepo(pool)

	_, err := repo.ListByTeacher(ctx, tenantID, schoolID, userID, "00000000-0000-0000-0000-000000000000")
	require.Error(t, err)
}

func TestPgRepository_ListByYear(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	ctx := context.Background()
	pool, cleanup := startPG(t)
	defer cleanup()
	applyAllMigrations(t, pool)

	tenantID, schoolID, userID1 := seedTenantSchoolUser(t, pool)

	// Add a second user
	userID2 := uuid.New().String()
	_, err := pool.Exec(ctx, `INSERT INTO users (id, email, tenant_id, full_name) VALUES ($1, $2, $3, $4)`,
		userID2, "teacher2@test.com", tenantID, "Teacher Two")
	require.NoError(t, err)

	yearID := seedAcademicYear(t, pool, tenantID, schoolID)

	// Insert two summary rows
	for _, uid := range []string{userID1, userID2} {
		summaryID := uuid.New().String()
		_, err = pool.Exec(ctx, `
			INSERT INTO teacher_workload_summaries (id, tenant_id, school_id, user_id, academic_year_id,
				total_assigned_periods, unique_subjects, classes_taught,
				utilization_percentage, is_overcapacity, last_refreshed_at)
			VALUES ($1, $2, $3, $4, $5, 25, 3, 2, 0.70, false, NOW())
		`, summaryID, tenantID, schoolID, uid, yearID)
		require.NoError(t, err)
	}

	repo := newRepo(pool)

	resp, err := repo.ListByYear(ctx, tenantID, schoolID, yearID)
	require.NoError(t, err)
	require.Equal(t, 2, resp.Total)
	require.Len(t, resp.Items, 2)
}

func TestPgRepository_ListByYear_Empty(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	ctx := context.Background()
	pool, cleanup := startPG(t)
	defer cleanup()
	applyAllMigrations(t, pool)

	tenantID, schoolID, _ := seedTenantSchoolUser(t, pool)
	repo := newRepo(pool)

	resp, err := repo.ListByYear(ctx, tenantID, schoolID, "00000000-0000-0000-0000-000000000000")
	require.NoError(t, err)
	require.Equal(t, 0, resp.Total)
	require.Empty(t, resp.Items)
}

func TestPgRepository_GetByTeacherYear(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	ctx := context.Background()
	pool, cleanup := startPG(t)
	defer cleanup()
	applyAllMigrations(t, pool)

	tenantID, schoolID, userID := seedTenantSchoolUser(t, pool)
	yearID := seedAcademicYear(t, pool, tenantID, schoolID)

	repo := newRepo(pool)

	summaryID := uuid.New().String()
	_, err := pool.Exec(ctx, `
		INSERT INTO teacher_workload_summaries (id, tenant_id, school_id, user_id, academic_year_id,
			total_assigned_periods, unique_subjects, classes_taught,
			utilization_percentage, is_overcapacity, last_refreshed_at)
		VALUES ($1, $2, $3, $4, $5, 40, 5, 4, 0.90, true, NOW())
	`, summaryID, tenantID, schoolID, userID, yearID)
	require.NoError(t, err)

	summary, err := repo.GetByTeacherYear(ctx, userID, yearID)
	require.NoError(t, err)
	require.NotNil(t, summary)
	require.Equal(t, 40, summary.TotalAssignedPeriods)
	require.Equal(t, 5, summary.UniqueSubjects)
	require.True(t, summary.IsOvercapacity)
}

func TestPgRepository_GetByTeacherYear_NotFound(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	ctx := context.Background()
	pool, cleanup := startPG(t)
	defer cleanup()
	applyAllMigrations(t, pool)

	repo := newRepo(pool)

	_, err := repo.GetByTeacherYear(ctx, "00000000-0000-0000-0000-000000000001", "00000000-0000-0000-0000-000000000000")
	require.Error(t, err)
}

func TestPgRepository_RefreshComputation(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	ctx := context.Background()
	pool, cleanup := startPG(t)
	defer cleanup()
	applyAllMigrations(t, pool)

	repo := newRepo(pool)

	err := repo.RefreshComputation(ctx, "00000000-0000-0000-0000-000000000000")
	// The function will run and not error (it simply may not find any data)
	require.NoError(t, err)
}
