package teacherdeliverysummaries

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

func applyMigration(t *testing.T, pool *pgxpool.Pool, filename string) {
	t.Helper()
	path := filepath.Join(migrationsDir(), filename)
	sql, err := os.ReadFile(path)
	require.NoError(t, err, "read migration %s", filename)
	_, err = pool.Exec(context.Background(), string(sql))
	require.NoError(t, err, "apply migration %s", filename)
}

func applyMigrations(t *testing.T, pool *pgxpool.Pool, filenames ...string) {
	t.Helper()
	for _, f := range filenames {
		applyMigration(t, pool, f)
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

func seedAcademicTerm(t *testing.T, pool *pgxpool.Pool, tenantID, schoolID, yearID string) (termID string) {
	t.Helper()
	ctx := context.Background()
	termID = uuid.New().String()
	userID := uuid.New().String()
	_, err := pool.Exec(ctx, `INSERT INTO users (id, email, tenant_id, full_name) VALUES ($1, $2, $3, $4) ON CONFLICT (id) DO NOTHING`,
		userID, "sys-academicterm@test.com", tenantID, "Acad Term System")
	require.NoError(t, err)
	_, err = pool.Exec(ctx, `INSERT INTO academic_terms (id, tenant_id, school_id, academic_year_id, name, term_number, start_date, end_date, is_final, created_by, updated_by) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)`,
		termID, tenantID, schoolID, yearID, "Term 1", 1, "2025-01-01", "2025-04-30", false, userID, userID)
	require.NoError(t, err)
	return termID
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
	applyMigrations(t, pool, "000001_initial_schema.up.sql", "000013_create_teacher_delivery_summaries.up.sql")

	tenantID, schoolID, userID := seedTenantSchoolUser(t, pool)
	yearID := seedAcademicYear(t, pool, tenantID, schoolID)
	termID := seedAcademicTerm(t, pool, tenantID, schoolID, yearID)

	repo := newRepo(pool)

	// Directly insert a delivery summary row
	summaryID := uuid.New().String()
	_, err := pool.Exec(ctx, `
		INSERT INTO teacher_delivery_summaries (id, tenant_id, school_id, user_id, academic_term_id,
			total_assigned_slots, marked_slots, missed_slots, sessions_created, sessions_approved,
			on_time_submission_rate, last_refreshed_at)
		VALUES ($1, $2, $3, $4, $5, 20, 18, 2, 10, 8, 0.85, NOW())
	`, summaryID, tenantID, schoolID, userID, termID)
	require.NoError(t, err)

	// List by teacher
	resp, err := repo.ListByTeacher(ctx, tenantID, schoolID, userID, termID)
	require.NoError(t, err)
	require.Equal(t, 1, resp.Total)
	require.Len(t, resp.Items, 1)
	require.Equal(t, 20, resp.Items[0].TotalAssignedSlots)
	require.Equal(t, 18, resp.Items[0].MarkedSlots)
	require.Equal(t, 2, resp.Items[0].MissedSlots)
	require.Equal(t, 10, resp.Items[0].SessionsCreated)
	require.Equal(t, 8, resp.Items[0].SessionsApproved)
	require.NotNil(t, resp.Items[0].OnTimeSubmissionRate)
	require.InDelta(t, 0.85, *resp.Items[0].OnTimeSubmissionRate, 0.01)
}

func TestPgRepository_ListByTeacher_NotFound(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	ctx := context.Background()
	pool, cleanup := startPG(t)
	defer cleanup()
	applyMigrations(t, pool, "000001_initial_schema.up.sql", "000013_create_teacher_delivery_summaries.up.sql")

	tenantID, schoolID, userID := seedTenantSchoolUser(t, pool)

	repo := newRepo(pool)

	_, err := repo.ListByTeacher(ctx, tenantID, schoolID, userID, "00000000-0000-0000-0000-000000000000")
	require.Error(t, err)
}

func TestPgRepository_ListByTerm(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	ctx := context.Background()
	pool, cleanup := startPG(t)
	defer cleanup()
	applyMigrations(t, pool, "000001_initial_schema.up.sql", "000013_create_teacher_delivery_summaries.up.sql")

	tenantID, schoolID, userID1 := seedTenantSchoolUser(t, pool)

	// Add a second user
	userID2 := uuid.New().String()
	_, err := pool.Exec(ctx, `INSERT INTO users (id, email, tenant_id, full_name) VALUES ($1, $2, $3, $4)`,
		userID2, "teacher2@test.com", tenantID, "Teacher Two")
	require.NoError(t, err)

	yearID := seedAcademicYear(t, pool, tenantID, schoolID)
	termID := seedAcademicTerm(t, pool, tenantID, schoolID, yearID)

	// Insert two summary rows
	for _, uid := range []string{userID1, userID2} {
		summaryID := uuid.New().String()
		_, err = pool.Exec(ctx, `
			INSERT INTO teacher_delivery_summaries (id, tenant_id, school_id, user_id, academic_term_id,
				total_assigned_slots, marked_slots, missed_slots, sessions_created, sessions_approved,
				on_time_submission_rate, last_refreshed_at)
			VALUES ($1, $2, $3, $4, $5, 10, 9, 1, 5, 4, 0.90, NOW())
		`, summaryID, tenantID, schoolID, uid, termID)
		require.NoError(t, err)
	}

	repo := newRepo(pool)

	resp, err := repo.ListByTerm(ctx, tenantID, schoolID, termID)
	require.NoError(t, err)
	require.Equal(t, 2, resp.Total)
	require.Len(t, resp.Items, 2)
}

func TestPgRepository_GetByTeacherTerm(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	ctx := context.Background()
	pool, cleanup := startPG(t)
	defer cleanup()
	applyMigrations(t, pool, "000001_initial_schema.up.sql", "000013_create_teacher_delivery_summaries.up.sql")

	tenantID, schoolID, userID := seedTenantSchoolUser(t, pool)
	yearID := seedAcademicYear(t, pool, tenantID, schoolID)
	termID := seedAcademicTerm(t, pool, tenantID, schoolID, yearID)

	repo := newRepo(pool)

	summaryID := uuid.New().String()
	_, err := pool.Exec(ctx, `
		INSERT INTO teacher_delivery_summaries (id, tenant_id, school_id, user_id, academic_term_id,
			total_assigned_slots, marked_slots, missed_slots, sessions_created, sessions_approved,
			on_time_submission_rate, last_refreshed_at)
		VALUES ($1, $2, $3, $4, $5, 15, 12, 3, 7, 5, 0.75, NOW())
	`, summaryID, tenantID, schoolID, userID, termID)
	require.NoError(t, err)

	summary, err := repo.GetByTeacherTerm(ctx, userID, termID)
	require.NoError(t, err)
	require.NotNil(t, summary)
	require.Equal(t, 15, summary.TotalAssignedSlots)
	require.Equal(t, 12, summary.MarkedSlots)
	require.Equal(t, 3, summary.MissedSlots)
}

func TestPgRepository_GetByTeacherTerm_NotFound(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	ctx := context.Background()
	pool, cleanup := startPG(t)
	defer cleanup()
	applyMigrations(t, pool, "000001_initial_schema.up.sql", "000013_create_teacher_delivery_summaries.up.sql")

	repo := newRepo(pool)

	_, err := repo.GetByTeacherTerm(ctx, "00000000-0000-0000-0000-000000000001", "00000000-0000-0000-0000-000000000000")
	require.Error(t, err)
}

func TestPgRepository_ListByTerm_Empty(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	ctx := context.Background()
	pool, cleanup := startPG(t)
	defer cleanup()
	applyMigrations(t, pool, "000001_initial_schema.up.sql", "000013_create_teacher_delivery_summaries.up.sql")

	tenantID, schoolID, _ := seedTenantSchoolUser(t, pool)
	repo := newRepo(pool)

	resp, err := repo.ListByTerm(ctx, tenantID, schoolID, "00000000-0000-0000-0000-000000000000")
	require.NoError(t, err)
	require.Equal(t, 0, resp.Total)
	require.Empty(t, resp.Items)
}

func TestPgRepository_RefreshComputation(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	ctx := context.Background()
	pool, cleanup := startPG(t)
	defer cleanup()
	applyMigrations(t, pool, "000001_initial_schema.up.sql", "000013_create_teacher_delivery_summaries.up.sql")

	repo := newRepo(pool)

	err := repo.RefreshComputation(ctx, "00000000-0000-0000-0000-000000000000")
	// The function will run and not error (it simply may not find any data)
	require.NoError(t, err)
}
