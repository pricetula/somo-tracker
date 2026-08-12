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

func seedTenantSchool(t *testing.T, pool *pgxpool.Pool) (tenantID, schoolID string) {
	t.Helper()
	ctx := context.Background()
	tenantID = uuid.New().String()
	schoolID = uuid.New().String()
	_, err := pool.Exec(ctx, `INSERT INTO tenants (id, name, slug, stytch_org_id) VALUES ($1, $2, $3, $4)`,
		tenantID, "Test Tenant", "slug-"+tenantID[:8], "stytch-"+tenantID[:8])
	require.NoError(t, err)
	_, err = pool.Exec(ctx, `INSERT INTO cbc_schools (id, tenant_id, name, county, sub_county, school_type) VALUES ($1, $2, $3, 'Nairobi', 'Westlands', 'Public')`,
		schoolID, tenantID, "Test School")
	require.NoError(t, err)
	return tenantID, schoolID
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

func seedClass(t *testing.T, pool *pgxpool.Pool, tenantID, schoolID, yearID string) (classID string) {
	t.Helper()
	ctx := context.Background()
	classID = uuid.New().String()
	streamID := uuid.New().String()
	_, err := pool.Exec(ctx, `INSERT INTO cbc_streams (id, tenant_id, school_id, name) VALUES ($1, $2, $3, 'Main')`,
		streamID, tenantID, schoolID)
	require.NoError(t, err)
	_, err = pool.Exec(ctx, `INSERT INTO cbc_classes (id, tenant_id, school_id, academic_year_id, grade_level, stream_id) VALUES ($1, $2, $3, $4, $5, $6)`,
		classID, tenantID, schoolID, yearID, "G7", streamID)
	require.NoError(t, err)
	return classID
}

func seedLearningArea(t *testing.T, pool *pgxpool.Pool, tenantID, schoolID string) (areaID string) {
	t.Helper()
	ctx := context.Background()
	areaID = uuid.New().String()
	_, err := pool.Exec(ctx, `INSERT INTO cbc_learning_areas (id, tenant_id, school_id, name, code, education_level, grade_level) VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		areaID, tenantID, schoolID, "Mathematics", "MATH", "Upper_Primary", "G7")
	require.NoError(t, err)
	return areaID
}

func seedUser(t *testing.T, pool *pgxpool.Pool, tenantID string) (userID string) {
	t.Helper()
	ctx := context.Background()
	userID = uuid.New().String()
	_, err := pool.Exec(ctx, `INSERT INTO users (id, email, tenant_id, full_name) VALUES ($1, $2, $3, $4)`,
		userID, "teacher@test.com", tenantID, "Test Teacher")
	require.NoError(t, err)
	return userID
}

func newRepo(pool *pgxpool.Pool) *PgRepository {
	return NewRepository(&database.Pools{PG: pool})
}

func f64(v float64) *float64 { return &v }

// seedScaleProfile creates a default grading scale profile for tests.
func seedScaleProfile(t *testing.T, repo *PgRepository, ctx context.Context, tenantID, schoolID string) (profileID string) {
	t.Helper()
	pid, _, err := repo.CreateScaleProfileWithRanges(ctx, CreateScaleProfileParams{
		TenantID: tenantID, SchoolID: schoolID, Name: "Test Profile",
		Ranges: []CreateScaleRangeParams{
			{TenantID: tenantID, PerformanceLevel: "EE", MinPercentage: 80, MaxPercentage: 100},
			{TenantID: tenantID, PerformanceLevel: "ME", MinPercentage: 50, MaxPercentage: 79.99},
			{TenantID: tenantID, PerformanceLevel: "AE", MinPercentage: 30, MaxPercentage: 49.99},
			{TenantID: tenantID, PerformanceLevel: "BE", MinPercentage: 0, MaxPercentage: 29.99},
		},
	})
	require.NoError(t, err)
	return pid
}

// seedStudent creates a minimal student for FK constraints.
func seedStudent(t *testing.T, pool *pgxpool.Pool, tenantID, schoolID string) (studentID string) {
	t.Helper()
	ctx := context.Background()
	studentID = uuid.New().String()
	_, err := pool.Exec(ctx, `INSERT INTO cbc_students (id, tenant_id, school_id, full_name, gender) VALUES ($1, $2, $3, $4, $5)`,
		studentID, tenantID, schoolID, "Test Student", "M")
	require.NoError(t, err)
	return studentID
}

// ── Tests ────────────────────────────────────────────────────────────────

func TestPgRepository_CreateAndGetScaleProfile(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	ctx := context.Background()
	pool, cleanup := startPG(t)
	defer cleanup()
	applyMigrations(t, pool, "000001_initial_schema.up.sql")

	tenantID, schoolID := seedTenantSchool(t, pool)
	repo := newRepo(pool)

	profileID, rangeIDs, err := repo.CreateScaleProfileWithRanges(ctx, CreateScaleProfileParams{
		TenantID: tenantID,
		SchoolID: schoolID,
		Name:     "Standard CBC",
		Ranges: []CreateScaleRangeParams{
			{TenantID: tenantID, PerformanceLevel: "EE", MinPercentage: 80, MaxPercentage: 100},
			{TenantID: tenantID, PerformanceLevel: "ME", MinPercentage: 50, MaxPercentage: 79.99},
			{TenantID: tenantID, PerformanceLevel: "AE", MinPercentage: 30, MaxPercentage: 49.99},
			{TenantID: tenantID, PerformanceLevel: "BE", MinPercentage: 0, MaxPercentage: 29.99},
		},
	})
	require.NoError(t, err)
	require.NotEmpty(t, profileID)
	require.Len(t, rangeIDs, 4)

	// Fetch the profile with ranges
	fetched, err := repo.GetScaleProfileByID(ctx, profileID, tenantID, schoolID)
	require.NoError(t, err)
	require.NotNil(t, fetched)
	require.Equal(t, "Standard CBC", fetched.Name)
	require.Len(t, fetched.Ranges, 4)
	require.Equal(t, "EE", fetched.Ranges[3].PerformanceLevel) // ordered by min_percentage ASC
}

func TestPgRepository_ListScaleProfiles(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	ctx := context.Background()
	pool, cleanup := startPG(t)
	defer cleanup()
	applyMigrations(t, pool, "000001_initial_schema.up.sql")

	tenantID, schoolID := seedTenantSchool(t, pool)
	repo := newRepo(pool)

	// Create two profiles
	baseRanges := []CreateScaleRangeParams{
		{TenantID: tenantID, PerformanceLevel: "EE", MinPercentage: 80, MaxPercentage: 100},
		{TenantID: tenantID, PerformanceLevel: "ME", MinPercentage: 50, MaxPercentage: 79.99},
		{TenantID: tenantID, PerformanceLevel: "AE", MinPercentage: 30, MaxPercentage: 49.99},
		{TenantID: tenantID, PerformanceLevel: "BE", MinPercentage: 0, MaxPercentage: 29.99},
	}

	_, _, err := repo.CreateScaleProfileWithRanges(ctx, CreateScaleProfileParams{
		TenantID: tenantID, SchoolID: schoolID, Name: "Profile A", Ranges: baseRanges,
	})
	require.NoError(t, err)

	_, _, err = repo.CreateScaleProfileWithRanges(ctx, CreateScaleProfileParams{
		TenantID: tenantID, SchoolID: schoolID, Name: "Profile B", Ranges: baseRanges,
	})
	require.NoError(t, err)

	profiles, err := repo.ListScaleProfiles(ctx, tenantID, schoolID, false)
	require.NoError(t, err)
	require.Len(t, profiles, 2)
}

func TestPgRepository_ToggleScaleProfileActive(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	ctx := context.Background()
	pool, cleanup := startPG(t)
	defer cleanup()
	applyMigrations(t, pool, "000001_initial_schema.up.sql")

	tenantID, schoolID := seedTenantSchool(t, pool)
	repo := newRepo(pool)

	profileID, _, err := repo.CreateScaleProfileWithRanges(ctx, CreateScaleProfileParams{
		TenantID: tenantID, SchoolID: schoolID, Name: "Toggle Profile",
		Ranges: []CreateScaleRangeParams{
			{TenantID: tenantID, PerformanceLevel: "EE", MinPercentage: 80, MaxPercentage: 100},
			{TenantID: tenantID, PerformanceLevel: "ME", MinPercentage: 50, MaxPercentage: 79.99},
			{TenantID: tenantID, PerformanceLevel: "AE", MinPercentage: 30, MaxPercentage: 49.99},
			{TenantID: tenantID, PerformanceLevel: "BE", MinPercentage: 0, MaxPercentage: 29.99},
		},
	})
	require.NoError(t, err)

	// Toggle to inactive
	err = repo.ToggleScaleProfileActive(ctx, profileID, tenantID, schoolID, false)
	require.NoError(t, err)

	// Verify inactive
	profiles, err := repo.ListScaleProfiles(ctx, tenantID, schoolID, true)
	require.NoError(t, err)
	require.Len(t, profiles, 0)
}

func TestPgRepository_DeleteScaleProfile(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	ctx := context.Background()
	pool, cleanup := startPG(t)
	defer cleanup()
	applyMigrations(t, pool, "000001_initial_schema.up.sql")

	tenantID, schoolID := seedTenantSchool(t, pool)
	repo := newRepo(pool)

	profileID, _, err := repo.CreateScaleProfileWithRanges(ctx, CreateScaleProfileParams{
		TenantID: tenantID, SchoolID: schoolID, Name: "To Delete",
		Ranges: []CreateScaleRangeParams{
			{TenantID: tenantID, PerformanceLevel: "EE", MinPercentage: 80, MaxPercentage: 100},
			{TenantID: tenantID, PerformanceLevel: "ME", MinPercentage: 50, MaxPercentage: 79.99},
			{TenantID: tenantID, PerformanceLevel: "AE", MinPercentage: 30, MaxPercentage: 49.99},
			{TenantID: tenantID, PerformanceLevel: "BE", MinPercentage: 0, MaxPercentage: 29.99},
		},
	})
	require.NoError(t, err)

	err = repo.DeleteScaleProfile(ctx, profileID, tenantID, schoolID)
	require.NoError(t, err)

	_, err = repo.GetScaleProfileByID(ctx, profileID, tenantID, schoolID)
	require.Error(t, err)
}

func TestPgRepository_ReplaceScaleRanges(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	ctx := context.Background()
	pool, cleanup := startPG(t)
	defer cleanup()
	applyMigrations(t, pool, "000001_initial_schema.up.sql")

	tenantID, schoolID := seedTenantSchool(t, pool)
	repo := newRepo(pool)

	profileID, _, err := repo.CreateScaleProfileWithRanges(ctx, CreateScaleProfileParams{
		TenantID: tenantID, SchoolID: schoolID, Name: "Replace Me",
		Ranges: []CreateScaleRangeParams{
			{TenantID: tenantID, PerformanceLevel: "EE", MinPercentage: 80, MaxPercentage: 100},
			{TenantID: tenantID, PerformanceLevel: "ME", MinPercentage: 50, MaxPercentage: 79.99},
			{TenantID: tenantID, PerformanceLevel: "AE", MinPercentage: 30, MaxPercentage: 49.99},
			{TenantID: tenantID, PerformanceLevel: "BE", MinPercentage: 0, MaxPercentage: 29.99},
		},
	})
	require.NoError(t, err)

	// Replace ranges with different boundaries
	newRangeIDs, err := repo.ReplaceScaleRanges(ctx, profileID, []CreateScaleRangeParams{
		{TenantID: tenantID, PerformanceLevel: "EE", MinPercentage: 85, MaxPercentage: 100},
		{TenantID: tenantID, PerformanceLevel: "ME", MinPercentage: 55, MaxPercentage: 84.99},
		{TenantID: tenantID, PerformanceLevel: "AE", MinPercentage: 35, MaxPercentage: 54.99},
		{TenantID: tenantID, PerformanceLevel: "BE", MinPercentage: 0, MaxPercentage: 34.99},
	})
	require.NoError(t, err)
	require.Len(t, newRangeIDs, 4)

	// Verify new ranges
	ranges, err := repo.GetScaleRanges(ctx, profileID)
	require.NoError(t, err)
	require.Len(t, ranges, 4)
	require.InDelta(t, 85.0, ranges[3].MinPercentage, 0.01)
}

func TestPgRepository_SessionLifecycle(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	ctx := context.Background()
	pool, cleanup := startPG(t)
	defer cleanup()
	applyMigrations(t, pool, "000001_initial_schema.up.sql")

	tenantID, schoolID := seedTenantSchool(t, pool)
	yearID := seedAcademicYear(t, pool, tenantID, schoolID)
	termID := seedAcademicTerm(t, pool, tenantID, schoolID, yearID)
	classID := seedClass(t, pool, tenantID, schoolID, yearID)
	areaID := seedLearningArea(t, pool, tenantID, schoolID)
	userID := seedUser(t, pool, tenantID)

	repo := newRepo(pool)

	// Create a scale profile first
	profileID, _, err := repo.CreateScaleProfileWithRanges(ctx, CreateScaleProfileParams{
		TenantID: tenantID, SchoolID: schoolID, Name: "Session Profile",
		Ranges: []CreateScaleRangeParams{
			{TenantID: tenantID, PerformanceLevel: "EE", MinPercentage: 80, MaxPercentage: 100},
			{TenantID: tenantID, PerformanceLevel: "ME", MinPercentage: 50, MaxPercentage: 79.99},
			{TenantID: tenantID, PerformanceLevel: "AE", MinPercentage: 30, MaxPercentage: 49.99},
			{TenantID: tenantID, PerformanceLevel: "BE", MinPercentage: 0, MaxPercentage: 29.99},
		},
	})
	require.NoError(t, err)

	// Create session
	sessionID, err := repo.CreateSession(ctx, CreateSessionParams{
		TenantID:              tenantID,
		SchoolID:              schoolID,
		ClassID:               classID,
		LearningAreaID:        areaID,
		AcademicTermID:        termID,
		AcademicYearID:        yearID,
		Name:                  "Math Quiz 1",
		EvaluationMethod:      "QUANTITATIVE",
		MaxPoints:             f64(100),
		GradingScaleProfileID: &profileID,
		CreatedBy:             userID,
	})
	require.NoError(t, err)
	require.NotEmpty(t, sessionID)

	// Get session by ID
	session, err := repo.GetSessionByID(ctx, sessionID, tenantID, schoolID)
	require.NoError(t, err)
	require.Equal(t, "Math Quiz 1", session.Name)
	require.Equal(t, "DRAFT", session.Status)

	// Submit -> PENDING_APPROVAL
	err = repo.UpdateSessionStatus(ctx, sessionID, tenantID, schoolID, "PENDING_APPROVAL", nil, &userID)
	require.NoError(t, err)

	session, err = repo.GetSessionByID(ctx, sessionID, tenantID, schoolID)
	require.NoError(t, err)
	require.Equal(t, "PENDING_APPROVAL", session.Status)

	// Approve -> PUBLISHED
	err = repo.UpdateSessionStatus(ctx, sessionID, tenantID, schoolID, "PUBLISHED", nil, &userID)
	require.NoError(t, err)

	session, err = repo.GetSessionByID(ctx, sessionID, tenantID, schoolID)
	require.NoError(t, err)
	require.Equal(t, "PUBLISHED", session.Status)
}

func TestPgRepository_ListSessions(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	ctx := context.Background()
	pool, cleanup := startPG(t)
	defer cleanup()
	applyMigrations(t, pool, "000001_initial_schema.up.sql")

	tenantID, schoolID := seedTenantSchool(t, pool)
	yearID := seedAcademicYear(t, pool, tenantID, schoolID)
	termID := seedAcademicTerm(t, pool, tenantID, schoolID, yearID)
	classID := seedClass(t, pool, tenantID, schoolID, yearID)
	areaID := seedLearningArea(t, pool, tenantID, schoolID)
	userID := seedUser(t, pool, tenantID)

	repo := newRepo(pool)
	profileID := seedScaleProfile(t, repo, ctx, tenantID, schoolID)

	// Create two sessions
	for i := 0; i < 2; i++ {
		_, err := repo.CreateSession(ctx, CreateSessionParams{
			TenantID:              tenantID,
			SchoolID:              schoolID,
			ClassID:               classID,
			LearningAreaID:        areaID,
			AcademicTermID:        termID,
			AcademicYearID:        yearID,
			Name:                  fmt.Sprintf("Session %d", i+1),
			EvaluationMethod:      "QUANTITATIVE",
			MaxPoints:             f64(100),
			GradingScaleProfileID: &profileID,
			CreatedBy:             userID,
		})
		require.NoError(t, err)
	}

	sessions, total, err := repo.ListSessions(ctx, tenantID, schoolID, SessionFilters{Page: 1, Limit: 50})
	require.NoError(t, err)
	require.Equal(t, 2, total)
	require.Len(t, sessions, 2)
}

func TestPgRepository_DeleteSession(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	ctx := context.Background()
	pool, cleanup := startPG(t)
	defer cleanup()
	applyMigrations(t, pool, "000001_initial_schema.up.sql")

	tenantID, schoolID := seedTenantSchool(t, pool)
	yearID := seedAcademicYear(t, pool, tenantID, schoolID)
	termID := seedAcademicTerm(t, pool, tenantID, schoolID, yearID)
	classID := seedClass(t, pool, tenantID, schoolID, yearID)
	areaID := seedLearningArea(t, pool, tenantID, schoolID)
	userID := seedUser(t, pool, tenantID)

	repo := newRepo(pool)
	profileID := seedScaleProfile(t, repo, ctx, tenantID, schoolID)

	sessionID, err := repo.CreateSession(ctx, CreateSessionParams{
		TenantID:              tenantID,
		SchoolID:              schoolID,
		ClassID:               classID,
		LearningAreaID:        areaID,
		AcademicTermID:        termID,
		AcademicYearID:        yearID,
		Name:                  "To Delete",
		EvaluationMethod:      "QUANTITATIVE",
		MaxPoints:             f64(100),
		GradingScaleProfileID: &profileID,
		CreatedBy:             userID,
	})
	require.NoError(t, err)

	err = repo.DeleteSession(ctx, sessionID, tenantID, schoolID)
	require.NoError(t, err)

	_, err = repo.GetSessionByID(ctx, sessionID, tenantID, schoolID)
	require.Error(t, err)
}

func TestPgRepository_UpsertAndGetStudentScores(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	ctx := context.Background()
	pool, cleanup := startPG(t)
	defer cleanup()
	applyMigrations(t, pool, "000001_initial_schema.up.sql")

	tenantID, schoolID := seedTenantSchool(t, pool)
	yearID := seedAcademicYear(t, pool, tenantID, schoolID)
	termID := seedAcademicTerm(t, pool, tenantID, schoolID, yearID)
	classID := seedClass(t, pool, tenantID, schoolID, yearID)
	areaID := seedLearningArea(t, pool, tenantID, schoolID)
	userID := seedUser(t, pool, tenantID)

	repo := newRepo(pool)
	profileID := seedScaleProfile(t, repo, ctx, tenantID, schoolID)

	sessionID, err := repo.CreateSession(ctx, CreateSessionParams{
		TenantID:              tenantID,
		SchoolID:              schoolID,
		ClassID:               classID,
		LearningAreaID:        areaID,
		AcademicTermID:        termID,
		AcademicYearID:        yearID,
		Name:                  "Score Test",
		EvaluationMethod:      "QUANTITATIVE",
		MaxPoints:             f64(100),
		GradingScaleProfileID: &profileID,
		CreatedBy:             userID,
	})
	require.NoError(t, err)

	studentID := seedStudent(t, pool, tenantID, schoolID)

	// Upsert a score
	err = repo.UpsertStudentScore(ctx, UpsertScoreParams{
		TenantID:         tenantID,
		SessionID:        sessionID,
		StudentID:        studentID,
		RawScore:         f64(85),
		EnrollmentStatus: "ACTIVE",
	})
	require.NoError(t, err)

	// Get scores
	scores, err := repo.GetStudentScoresBySession(ctx, sessionID, tenantID, schoolID)
	require.NoError(t, err)
	require.Len(t, scores, 1)
	require.Equal(t, studentID, scores[0].StudentID)
	require.NotNil(t, scores[0].RawScore)
	require.InDelta(t, 85.0, *scores[0].RawScore, 0.01)
}

func TestPgRepository_BulkUpsertStudentScores(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	ctx := context.Background()
	pool, cleanup := startPG(t)
	defer cleanup()
	applyMigrations(t, pool, "000001_initial_schema.up.sql")

	tenantID, schoolID := seedTenantSchool(t, pool)
	yearID := seedAcademicYear(t, pool, tenantID, schoolID)
	termID := seedAcademicTerm(t, pool, tenantID, schoolID, yearID)
	classID := seedClass(t, pool, tenantID, schoolID, yearID)
	areaID := seedLearningArea(t, pool, tenantID, schoolID)
	userID := seedUser(t, pool, tenantID)

	repo := newRepo(pool)
	profileID := seedScaleProfile(t, repo, ctx, tenantID, schoolID)

	sessionID, err := repo.CreateSession(ctx, CreateSessionParams{
		TenantID:              tenantID,
		SchoolID:              schoolID,
		ClassID:               classID,
		LearningAreaID:        areaID,
		AcademicTermID:        termID,
		AcademicYearID:        yearID,
		Name:                  "Bulk Score Test",
		EvaluationMethod:      "QUANTITATIVE",
		MaxPoints:             f64(100),
		GradingScaleProfileID: &profileID,
		CreatedBy:             userID,
	})
	require.NoError(t, err)

	studentA := seedStudent(t, pool, tenantID, schoolID)
	studentB := seedStudent(t, pool, tenantID, schoolID)

	err = repo.BulkUpsertStudentScores(ctx, []UpsertScoreParams{
		{TenantID: tenantID, SessionID: sessionID, StudentID: studentA, RawScore: f64(90), EnrollmentStatus: "ACTIVE"},
		{TenantID: tenantID, SessionID: sessionID, StudentID: studentB, RawScore: f64(75), EnrollmentStatus: "ACTIVE"},
	})
	require.NoError(t, err)

	scores, err := repo.GetStudentScoresBySession(ctx, sessionID, tenantID, schoolID)
	require.NoError(t, err)
	require.Len(t, scores, 2)
}

func TestPgRepository_UpsertAndGetOutcomeGrades(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	ctx := context.Background()
	pool, cleanup := startPG(t)
	defer cleanup()
	applyMigrations(t, pool, "000001_initial_schema.up.sql")

	tenantID, schoolID := seedTenantSchool(t, pool)
	yearID := seedAcademicYear(t, pool, tenantID, schoolID)
	termID := seedAcademicTerm(t, pool, tenantID, schoolID, yearID)
	classID := seedClass(t, pool, tenantID, schoolID, yearID)
	areaID := seedLearningArea(t, pool, tenantID, schoolID)
	userID := seedUser(t, pool, tenantID)

	repo := newRepo(pool)

	sessionID, err := repo.CreateSession(ctx, CreateSessionParams{
		TenantID:         tenantID,
		SchoolID:         schoolID,
		ClassID:          classID,
		LearningAreaID:   areaID,
		AcademicTermID:   termID,
		AcademicYearID:   yearID,
		Name:             "Rubric Test",
		EvaluationMethod: "RUBRIC",
		CreatedBy:        userID,
	})
	require.NoError(t, err)

	// Create FK chain: strand → sub_strand → performance_indicator
	strandID := uuid.New().String()
	_, err = pool.Exec(ctx, `INSERT INTO cbc_strands (id, tenant_id, learning_area_id, name) VALUES ($1, $2, $3, $4)`,
		strandID, tenantID, areaID, "Number Sense")
	require.NoError(t, err)

	subStrandID := uuid.New().String()
	_, err = pool.Exec(ctx, `INSERT INTO cbc_sub_strands (id, tenant_id, strand_id, name) VALUES ($1, $2, $3, $4)`,
		subStrandID, tenantID, strandID, "Fractions")
	require.NoError(t, err)

	indicatorID := uuid.New().String()
	_, err = pool.Exec(ctx, `INSERT INTO performance_indicators (id, tenant_id, sub_strand_id, description, sequence_order) VALUES ($1, $2, $3, $4, $5)`,
		indicatorID, tenantID, subStrandID, "Can identify fractions", 1)
	require.NoError(t, err)

	// Create a real student for FK
	studentID := seedStudent(t, pool, tenantID, schoolID)

	// Upsert outcome grade
	err = repo.UpsertOutcomeGrade(ctx, UpsertOutcomeGradeParams{
		TenantID:               tenantID,
		SessionID:              sessionID,
		StudentID:              studentID,
		PerformanceIndicatorID: indicatorID,
		AwardedLevel:           "ME",
	})
	require.NoError(t, err)

	// Get outcome grades
	grades, err := repo.GetOutcomeGradesBySession(ctx, sessionID, tenantID, schoolID)
	require.NoError(t, err)
	require.Len(t, grades, 1)
	require.Equal(t, "ME", grades[0].AwardedLevel)
}

func TestPgRepository_WeightConfigCRUD(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	ctx := context.Background()
	pool, cleanup := startPG(t)
	defer cleanup()
	applyMigrations(t, pool, "000001_initial_schema.up.sql")

	repo := newRepo(pool)

	// Create
	id, err := repo.CreateWeightConfig(ctx, CreateWeightConfigParams{
		GradeLevel:         "G6",
		AssessmentTypeCode: "OPENER",
		TargetExam:         "KPSEA",
		WeightPercent:      10.5,
		EffectiveFrom:      2025,
		Notes:              strPtr("Opener exam weight"),
	})
	require.NoError(t, err)
	require.NotEmpty(t, id)

	// Get by ID
	cfg, err := repo.GetWeightConfigByID(ctx, id)
	require.NoError(t, err)
	require.Equal(t, "G6", cfg.GradeLevel)
	require.InDelta(t, 10.5, cfg.WeightPercent, 0.01)

	// List
	filter := AssessmentWeightConfigFilter{GradeLevel: strPtr("G6")}
	items, err := repo.ListWeightConfigs(ctx, filter)
	require.NoError(t, err)
	require.Len(t, items, 1)

	// Delete
	err = repo.DeleteWeightConfig(ctx, id)
	require.NoError(t, err)

	_, err = repo.GetWeightConfigByID(ctx, id)
	require.Error(t, err)
}

func TestPgRepository_HasScoresForSession(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	ctx := context.Background()
	pool, cleanup := startPG(t)
	defer cleanup()
	applyMigrations(t, pool, "000001_initial_schema.up.sql")

	tenantID, schoolID := seedTenantSchool(t, pool)
	yearID := seedAcademicYear(t, pool, tenantID, schoolID)
	termID := seedAcademicTerm(t, pool, tenantID, schoolID, yearID)
	classID := seedClass(t, pool, tenantID, schoolID, yearID)
	areaID := seedLearningArea(t, pool, tenantID, schoolID)
	userID := seedUser(t, pool, tenantID)

	repo := newRepo(pool)
	profileID := seedScaleProfile(t, repo, ctx, tenantID, schoolID)

	sessionID, err := repo.CreateSession(ctx, CreateSessionParams{
		TenantID:              tenantID,
		SchoolID:              schoolID,
		ClassID:               classID,
		LearningAreaID:        areaID,
		AcademicTermID:        termID,
		AcademicYearID:        yearID,
		Name:                  "Has Scores Test",
		EvaluationMethod:      "QUANTITATIVE",
		MaxPoints:             f64(100),
		GradingScaleProfileID: &profileID,
		CreatedBy:             userID,
	})
	require.NoError(t, err)

	has, err := repo.HasScoresForSession(ctx, sessionID)
	require.NoError(t, err)
	require.False(t, has)

	// Add a score
	studentID := seedStudent(t, pool, tenantID, schoolID)
	err = repo.UpsertStudentScore(ctx, UpsertScoreParams{
		TenantID:         tenantID,
		SessionID:        sessionID,
		StudentID:        studentID,
		RawScore:         f64(80),
		EnrollmentStatus: "ACTIVE",
	})
	require.NoError(t, err)

	has, err = repo.HasScoresForSession(ctx, sessionID)
	require.NoError(t, err)
	require.True(t, has)
}

func TestPgRepository_CountSessionsReferencingScale(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	ctx := context.Background()
	pool, cleanup := startPG(t)
	defer cleanup()
	applyMigrations(t, pool, "000001_initial_schema.up.sql")

	tenantID, schoolID := seedTenantSchool(t, pool)
	yearID := seedAcademicYear(t, pool, tenantID, schoolID)
	termID := seedAcademicTerm(t, pool, tenantID, schoolID, yearID)
	classID := seedClass(t, pool, tenantID, schoolID, yearID)
	areaID := seedLearningArea(t, pool, tenantID, schoolID)
	userID := seedUser(t, pool, tenantID)

	repo := newRepo(pool)

	profileID, _, err := repo.CreateScaleProfileWithRanges(ctx, CreateScaleProfileParams{
		TenantID: tenantID, SchoolID: schoolID, Name: "Ref Count",
		Ranges: []CreateScaleRangeParams{
			{TenantID: tenantID, PerformanceLevel: "EE", MinPercentage: 80, MaxPercentage: 100},
			{TenantID: tenantID, PerformanceLevel: "ME", MinPercentage: 50, MaxPercentage: 79.99},
			{TenantID: tenantID, PerformanceLevel: "AE", MinPercentage: 30, MaxPercentage: 49.99},
			{TenantID: tenantID, PerformanceLevel: "BE", MinPercentage: 0, MaxPercentage: 29.99},
		},
	})
	require.NoError(t, err)

	// Create a session referencing the profile
	_, err = repo.CreateSession(ctx, CreateSessionParams{
		TenantID:              tenantID,
		SchoolID:              schoolID,
		ClassID:               classID,
		LearningAreaID:        areaID,
		AcademicTermID:        termID,
		AcademicYearID:        yearID,
		Name:                  "Ref Session",
		EvaluationMethod:      "QUANTITATIVE",
		MaxPoints:             f64(100),
		GradingScaleProfileID: &profileID,
		CreatedBy:             userID,
	})
	require.NoError(t, err)

	count, err := repo.CountSessionsReferencingScale(ctx, profileID)
	require.NoError(t, err)
	require.Equal(t, 1, count)
}

func TestPgRepository_IsTermFinalised(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	ctx := context.Background()
	pool, cleanup := startPG(t)
	defer cleanup()
	applyMigrations(t, pool, "000001_initial_schema.up.sql")

	tenantID, schoolID := seedTenantSchool(t, pool)
	userID := seedUser(t, pool, tenantID)
	yearID := seedAcademicYear(t, pool, tenantID, schoolID)
	repo := newRepo(pool)

	// Create a non-finalised term (bypass helper to test directly)
	termID := uuid.New().String()
	_, err := pool.Exec(ctx, `INSERT INTO academic_terms (id, tenant_id, school_id, academic_year_id, name, term_number, start_date, end_date, is_final, created_by, updated_by) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)`,
		termID, tenantID, schoolID, yearID, "Non-Final", 2, "2025-05-01", "2025-08-30", false, userID, userID)
	require.NoError(t, err)

	finalised, err := repo.IsTermFinalised(ctx, termID)
	require.NoError(t, err)
	require.False(t, finalised)

	// Create a finalised term
	finalTermID := uuid.New().String()
	_, err = pool.Exec(ctx, `INSERT INTO academic_terms (id, tenant_id, school_id, academic_year_id, name, term_number, start_date, end_date, is_final, created_by, updated_by) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)`,
		finalTermID, tenantID, schoolID, yearID, "Final", 3, "2025-09-01", "2025-12-31", true, userID, userID)
	require.NoError(t, err)

	finalised, err = repo.IsTermFinalised(ctx, finalTermID)
	require.NoError(t, err)
	require.True(t, finalised)
}

func TestPgRepository_GetSessionStatusAndTerm(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	ctx := context.Background()
	pool, cleanup := startPG(t)
	defer cleanup()
	applyMigrations(t, pool, "000001_initial_schema.up.sql")

	tenantID, schoolID := seedTenantSchool(t, pool)
	userID := seedUser(t, pool, tenantID)
	yearID := seedAcademicYear(t, pool, tenantID, schoolID)
	termID := seedAcademicTerm(t, pool, tenantID, schoolID, yearID)
	classID := seedClass(t, pool, tenantID, schoolID, yearID)
	areaID := seedLearningArea(t, pool, tenantID, schoolID)

	repo := newRepo(pool)
	profileID := seedScaleProfile(t, repo, ctx, tenantID, schoolID)

	sessionID, err := repo.CreateSession(ctx, CreateSessionParams{
		TenantID:              tenantID,
		SchoolID:              schoolID,
		ClassID:               classID,
		LearningAreaID:        areaID,
		AcademicTermID:        termID,
		AcademicYearID:        yearID,
		Name:                  "Status Term Test",
		EvaluationMethod:      "QUANTITATIVE",
		MaxPoints:             f64(100),
		GradingScaleProfileID: &profileID,
		CreatedBy:             userID,
	})
	require.NoError(t, err)

	status, gotTermID, err := repo.GetSessionStatusAndTerm(ctx, sessionID, tenantID)
	require.NoError(t, err)
	require.Equal(t, "DRAFT", status)
	require.Equal(t, termID, gotTermID)
}
