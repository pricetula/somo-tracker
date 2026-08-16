package parents

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

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
	return NewRepository(&database.Pools{PG: pool}, zap.NewNop().Sugar())
}

func TestPgRepository_CreateAndGetByUserID(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	ctx := context.Background()
	pool, cleanup := startPG(t)
	defer cleanup()
	applyAllMigrations(t, pool)

	tenantID, schoolID, _ := seedTenantSchoolUser(t, pool)
	repo := newRepo(pool)

	// First create a user for the parent
	parentEmail := "parent@test.com"
	parentUserID := uuid.New().String()
	_, err := pool.Exec(ctx, `INSERT INTO users (id, email, tenant_id, full_name) VALUES ($1, $2, $3, $4)`,
		parentUserID, parentEmail, tenantID, "Parent User")
	require.NoError(t, err)

	// Register the parent in the memberships table (role = 'PARENT').
	// GetByID/GetDetail query parents through memberships, so this row is
	// required for the test to find the parent after Create.
	_, err = pool.Exec(ctx, `INSERT INTO memberships (tenant_id, user_id, school_id, role) VALUES ($1, $2, $3, 'PARENT')`,
		tenantID, parentUserID, schoolID)
	require.NoError(t, err)

	parentID, err := repo.Create(ctx, tenantID, CreateParentPayload{
		Email:    parentEmail,
		FullName: "Parent User",
	})
	require.NoError(t, err)
	require.NotEmpty(t, parentID)

	// Look up parent by user email
	parent, err := repo.GetByID(ctx, parentID, tenantID)
	require.NoError(t, err)
	require.NotNil(t, parent)
	require.Equal(t, parentEmail, parent.Email)
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
	_, err := repo.GetByID(ctx, "missing_id", "tenant_001")
	require.Error(t, err)
}

func TestPgRepository_LinkStudent(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	ctx := context.Background()
	pool, cleanup := startPG(t)
	defer cleanup()
	applyAllMigrations(t, pool)

	tenantID, schoolID, _ := seedTenantSchoolUser(t, pool)
	_ = schoolID
	repo := newRepo(pool)

	// Create a student
	studentID := uuid.New().String()
	_, err := pool.Exec(ctx, `INSERT INTO cbc_students (id, tenant_id, school_id, admission_number, full_name, gender, date_of_birth) VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		studentID, tenantID, schoolID, "ADM001", "Test Student", "M", time.Date(2012, 1, 1, 0, 0, 0, 0, time.UTC))
	require.NoError(t, err)

	// Create a parent via existing user
	parentEmail := "parent2@test.com"
	parentUserID := uuid.New().String()
	_, err = pool.Exec(ctx, `INSERT INTO users (id, email, tenant_id, full_name) VALUES ($1, $2, $3, $4)`,
		parentUserID, parentEmail, tenantID, "Parent Two")
	require.NoError(t, err)

	// Register the parent in the memberships table (role = 'PARENT').
	// GetByID/GetDetail query parents through memberships, so this row is
	// required for the test to find the parent after Create.
	_, err = pool.Exec(ctx, `INSERT INTO memberships (tenant_id, user_id, school_id, role) VALUES ($1, $2, $3, 'PARENT')`,
		tenantID, parentUserID, schoolID)
	require.NoError(t, err)

	parentID, err := repo.Create(ctx, tenantID, CreateParentPayload{
		Email:    parentEmail,
		FullName: "Parent Two",
	})
	require.NoError(t, err)

	// Link student to parent
	isPrimary := true
	err = repo.LinkStudent(ctx, parentID, tenantID, LinkStudentPayload{
		StudentID: studentID,
		IsPrimary: &isPrimary,
	})
	require.NoError(t, err)

	// Verify the parent detail shows the linked student
	detail, err := repo.GetDetail(ctx, parentID, tenantID)
	require.NoError(t, err)
	require.NotNil(t, detail)
	require.Len(t, detail.LinkedStudents, 1)
	require.Equal(t, studentID, detail.LinkedStudents[0].StudentID)
}
