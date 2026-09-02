package assessments

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

	tenantID, schoolID, userID, yearID := seedTenantSchoolUser(t, pool)
	repo := newRepo(pool)

	// Insert required class + learning_area references manually
	streamID := uuid.New().String()
	_, err := pool.Exec(ctx, `INSERT INTO cbc_streams (id, tenant_id, school_id, name) VALUES ($1, $2, $3, 'East')`,
		streamID, tenantID, schoolID)
	require.NoError(t, err)

	classID := uuid.New().String()
	_, err = pool.Exec(ctx, `INSERT INTO cbc_classes (id, tenant_id, school_id, grade_level, stream_id, academic_year_id) VALUES ($1, $2, $3, 'G4', $4, $5)`,
		classID, tenantID, schoolID, streamID, yearID)
	require.NoError(t, err)

	learningAreaID := uuid.New().String()
	_, err = pool.Exec(ctx, `INSERT INTO cbc_learning_areas (id, tenant_id, school_id, name, code, education_level, grade_level) VALUES ($1, $2, $3, 'Mathematics', 'MATH', 'Upper_Primary', 'G4')`,
		learningAreaID, tenantID, schoolID)
	require.NoError(t, err)

	termID := uuid.New().String()
	_, err = pool.Exec(ctx, `INSERT INTO academic_terms (id, tenant_id, school_id, academic_year_id, name, term_number, start_date, end_date, is_current, created_by, updated_by) VALUES ($1, $2, $3, $4, 'Term 1', 1, '2026-01-01', '2026-04-30', true, $5, $5)`,
		termID, tenantID, schoolID, yearID, userID)
	require.NoError(t, err)

	gradeScaleProfileID := uuid.New().String()
	_, err = pool.Exec(ctx, `INSERT INTO grading_scale_profiles (id, tenant_id, school_id, name) VALUES ($1, $2, $3, 'Standard Scale')`,
		gradeScaleProfileID, tenantID, schoolID)
	require.NoError(t, err)

	params := CreateSessionParams{
		TenantID:              tenantID,
		SchoolID:              schoolID,
		AcademicYearID:        yearID,
		ClassID:               classID,
		LearningAreaID:        learningAreaID,
		AcademicTermID:        termID,
		Name:                  "Mid-Term Math",
		EvaluationMethod:      "QUANTITATIVE",
		MaxPoints:             ptrFloat(100),
		GradingScaleProfileID: &gradeScaleProfileID,
		CreatedBy:             userID,
	}
	session, err := repo.Create(ctx, params)
	require.NoError(t, err)
	require.NotEmpty(t, session.ID)
	require.Equal(t, "Mid-Term Math", session.Name)
	require.Equal(t, "QUANTITATIVE", session.EvaluationMethod)

	fetched, err := repo.GetByID(ctx, session.ID, tenantID, schoolID)
	require.NoError(t, err)
	require.Equal(t, session.ID, fetched.ID)
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

func ptrFloat(v float64) *float64 { return &v }
func ptrStr(v string) *string     { return &v }

func TestPgRepository_UpsertAndListScores(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	ctx := context.Background()
	pool, cleanup := startPG(t)
	defer cleanup()
	applyAllMigrations(t, pool)

	tenantID, schoolID, userID, yearID := seedTenantSchoolUser(t, pool)
	repo := newRepo(pool)

	// stream + class + learning_area + term + grading scale
	streamID := uuid.New().String()
	_, err := pool.Exec(ctx, `INSERT INTO cbc_streams (id, tenant_id, school_id, name) VALUES ($1, $2, $3, 'East')`,
		streamID, tenantID, schoolID)
	require.NoError(t, err)
	classID := uuid.New().String()
	_, err = pool.Exec(ctx, `INSERT INTO cbc_classes (id, tenant_id, school_id, grade_level, stream_id, academic_year_id) VALUES ($1, $2, $3, 'G4', $4, $5)`,
		classID, tenantID, schoolID, streamID, yearID)
	require.NoError(t, err)
	learningAreaID := uuid.New().String()
	_, err = pool.Exec(ctx, `INSERT INTO cbc_learning_areas (id, tenant_id, school_id, name, code, education_level, grade_level) VALUES ($1, $2, $3, 'Mathematics', 'MATH', 'Upper_Primary', 'G4')`,
		learningAreaID, tenantID, schoolID)
	require.NoError(t, err)
	termID := uuid.New().String()
	_, err = pool.Exec(ctx, `INSERT INTO academic_terms (id, tenant_id, school_id, academic_year_id, name, term_number, start_date, end_date, is_current, created_by, updated_by) VALUES ($1, $2, $3, $4, 'Term 1', 1, '2026-01-01', '2026-04-30', true, $5, $5)`,
		termID, tenantID, schoolID, yearID, userID)
	require.NoError(t, err)
	scaleID := uuid.New().String()
	_, err = pool.Exec(ctx, `INSERT INTO grading_scale_profiles (id, tenant_id, school_id, name) VALUES ($1, $2, $3, 'Standard')`,
		scaleID, tenantID, schoolID)
	require.NoError(t, err)

	// Create session
	session, err := repo.Create(ctx, CreateSessionParams{
		TenantID: tenantID, SchoolID: schoolID, AcademicYearID: yearID,
		ClassID: classID, LearningAreaID: learningAreaID, AcademicTermID: termID,
		Name: "Mid-Term", EvaluationMethod: "QUANTITATIVE",
		MaxPoints: ptrFloat(100), GradingScaleProfileID: &scaleID, CreatedBy: userID,
	})
	require.NoError(t, err)

	// Create two students + enrollments
	s1 := uuid.New().String()
	s2 := uuid.New().String()
	_, err = pool.Exec(ctx, `INSERT INTO cbc_students (id, tenant_id, school_id, full_name, gender, date_of_birth) VALUES ($1, $2, $3, 'Alice', 'F', '2016-01-01'), ($4, $2, $3, 'Bob', 'M', '2016-02-01')`,
		s1, tenantID, schoolID, s2)
	require.NoError(t, err)
	_, err = pool.Exec(ctx, `INSERT INTO cbc_student_enrollments (student_id, class_id, academic_term_id, academic_year_id, tenant_id, school_id, status) VALUES ($1, $2, $3, $4, $5, $6, 'ACTIVE'), ($7, $2, $3, $4, $5, $6, 'ACTIVE')`,
		s1, classID, termID, yearID, tenantID, schoolID, s2)
	require.NoError(t, err)

	// Upsert valid scores
	count, err := repo.UpsertScores(ctx, session.ID, tenantID, []ScoreEntryPayload{
		{StudentID: s1, RawScore: ptrFloat(85.5)},
		{StudentID: s2, RawScore: ptrFloat(72)},
	})
	require.NoError(t, err)
	require.Equal(t, 2, count)

	// List
	result, err := repo.ListScores(ctx, session.ID, tenantID, 1, 50)
	require.NoError(t, err)
	require.Equal(t, 2, result.Total)
}
