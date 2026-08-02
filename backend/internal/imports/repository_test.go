package imports

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

func seedTenantSchoolUser(t *testing.T, pool *pgxpool.Pool) (tenantID, schoolID, userID uuid.UUID) {
	t.Helper()
	ctx := context.Background()
	tenantID = uuid.New()
	schoolID = uuid.New()
	userID = uuid.New()
	_, err := pool.Exec(ctx, `INSERT INTO tenants (id, name, slug, stytch_org_id) VALUES ($1, $2, $3, $4)`,
		tenantID, "Test", "slug-"+tenantID.String()[:8], "stytch-"+tenantID.String()[:8])
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

func TestPgRepository_CreateJob(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	ctx := context.Background()
	pool, cleanup := startPG(t)
	defer cleanup()
	applyMigration(t, pool, "000001_initial_schema.up.sql")

	tenantID, schoolID, _ := seedTenantSchoolUser(t, pool)
	repo := newRepo(pool)

	job := &Job{
		TenantID:     tenantID,
		SchoolID:     schoolID,
		JobType:      ImportJobTypeStudentImport,
		TotalRecords: 10,
	}
	jobID, err := repo.CreateJob(ctx, job)
	require.NoError(t, err)
	require.NotEqual(t, uuid.Nil, jobID)

	fetched, err := repo.GetJobByID(ctx, jobID)
	require.NoError(t, err)
	require.Equal(t, jobID, fetched.ID)
	require.Equal(t, ImportJobTypeStudentImport, fetched.JobType)
	require.Equal(t, int64(10), fetched.TotalRecords)
}

func TestPgRepository_GetJobByIDempotencyKey(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	ctx := context.Background()
	pool, cleanup := startPG(t)
	defer cleanup()
	applyMigration(t, pool, "000001_initial_schema.up.sql")

	tenantID, schoolID, _ := seedTenantSchoolUser(t, pool)
	repo := newRepo(pool)

	idempotencyKey := "idempotency-key-123"
	job := &Job{
		TenantID:       tenantID,
		SchoolID:       schoolID,
		JobType:        ImportJobTypeStudentImport,
		TotalRecords:   5,
		IDempotencyKey: &idempotencyKey,
	}
	jobID, isNew, err := repo.CreateJobIdempotent(ctx, job, "hash")
	require.NoError(t, err)
	require.True(t, isNew)
	require.NotEqual(t, uuid.Nil, jobID)

	// call again with same key and same payload hash
	job2, isNew2, err := repo.CreateJobIdempotent(ctx, job, "hash")
	require.NoError(t, err)
	require.False(t, isNew2)
	require.Equal(t, jobID, job2.ID)
}

func TestPgRepository_InsertStagingRowsAndGetStagingRows(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	ctx := context.Background()
	pool, cleanup := startPG(t)
	defer cleanup()
	applyMigration(t, pool, "000001_initial_schema.up.sql")

	tenantID, schoolID, _ := seedTenantSchoolUser(t, pool)
	repo := newRepo(pool)

	// create a job first
	job := &Job{
		TenantID:     tenantID,
		SchoolID:     schoolID,
		JobType:      ImportJobTypeStudentImport,
		TotalRecords: 2,
	}
	jobID, err := repo.CreateJob(ctx, job)
	require.NoError(t, err)

	// insert staging rows
	rows := []StagingRow{
		{JobID: jobID, TenantID: tenantID, SchoolID: schoolID, RowNumber: 0, RawData: []byte(`{"admission_number":"001"}`), Status: ImportStagingStatusPending},
		{JobID: jobID, TenantID: tenantID, SchoolID: schoolID, RowNumber: 1, RawData: []byte(`{"admission_number":"002"}`), Status: ImportStagingStatusPending},
	}
	err = repo.InsertStagingRows(ctx, rows)
	require.NoError(t, err)

	// retrieve rows
	fetched, err := repo.GetStagingRows(ctx, jobID, 0, 2)
	require.NoError(t, err)
	require.Len(t, fetched, 2)
	require.Equal(t, int64(0), fetched[0].RowNumber)
	require.Equal(t, int64(1), fetched[1].RowNumber)
	require.Equal(t, []byte(`{"admission_number":"001"}`), fetched[0].RawData)
	require.Equal(t, []byte(`{"admission_number":"002"}`), fetched[1].RawData)
}

func TestPgRepository_UpdateJobStatus(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	ctx := context.Background()
	pool, cleanup := startPG(t)
	defer cleanup()
	applyMigration(t, pool, "000001_initial_schema.up.sql")

	tenantID, schoolID, _ := seedTenantSchoolUser(t, pool)
	repo := newRepo(pool)

	job := &Job{
		TenantID:     tenantID,
		SchoolID:     schoolID,
		JobType:      ImportJobTypeStudentImport,
		TotalRecords: 5,
	}
	jobID, err := repo.CreateJob(ctx, job)
	require.NoError(t, err)

	err = repo.UpdateJobStatus(ctx, jobID, ImportJobStatusProcessing)
	require.NoError(t, err)

	fetched, err := repo.GetJobByID(ctx, jobID)
	require.NoError(t, err)
	require.Equal(t, ImportJobStatusProcessing, fetched.Status)
}
