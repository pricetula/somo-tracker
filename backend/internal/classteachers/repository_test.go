package classteachers

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

// seedAcademicYearAndStream creates the supporting rows required by the
// current cbc_classes schema (academic_year_id, stream_id are NOT NULL FKs).
func seedAcademicYearAndStream(t *testing.T, pool *pgxpool.Pool, tenantID, schoolID, userID string) (academicYearID, streamID string) {
	t.Helper()
	ctx := context.Background()
	academicYearID = uuid.New().String()
	streamID = uuid.New().String()
	_, err := pool.Exec(ctx, `INSERT INTO academic_years (id, tenant_id, school_id, name, start_date, end_date, is_current, created_by, updated_by)
		VALUES ($1, $2, $3, '2026', '2026-01-01', '2026-12-31', true, $4, $4)`,
		academicYearID, tenantID, schoolID, userID)
	require.NoError(t, err)
	_, err = pool.Exec(ctx, `INSERT INTO cbc_streams (id, tenant_id, school_id, name, color)
		VALUES ($1, $2, $3, 'Blue', '#0000FF')`,
		streamID, tenantID, schoolID)
	require.NoError(t, err)
	return academicYearID, streamID
}

// seedLearningArea inserts a cbc_learning_areas row (required for SUBJECT_TEACHER
// class-teacher rows whose chk_cct_subject_area_required constraint mandates a
// non-null learning_area_id). Returns the generated id via a pointer suitable
// for AssignableToClassTeacherParams.LearningAreaID.
func seedLearningArea(t *testing.T, pool *pgxpool.Pool, tenantID, schoolID string) string {
	t.Helper()
	ctx := context.Background()
	areaID := uuid.New().String()
	_, err := pool.Exec(ctx, `INSERT INTO cbc_learning_areas (id, tenant_id, school_id, name, code, education_level, grade_level)
		VALUES ($1, $2, $3, 'Mathematics', 'MATH', 'Upper_Primary', 'G4')`,
		areaID, tenantID, schoolID)
	require.NoError(t, err)
	return areaID
}

// ptrString returns a pointer to s.
func ptrString(s string) *string { return &s }

func TestPgRepository_CreateAndGetByID(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	ctx := context.Background()
	pool, cleanup := startPG(t)
	defer cleanup()
	applyMigration(t, pool, "000001_initial_schema.up.sql")

	tenantID, schoolID, userID := seedTenantSchoolUser(t, pool)
	academicYearID, streamID := seedAcademicYearAndStream(t, pool, tenantID, schoolID, userID)
	repo := newRepo(pool)

	// Create a class first
	classID := uuid.New().String()
	_, err := pool.Exec(ctx, `INSERT INTO cbc_classes (id, tenant_id, school_id, academic_year_id, grade_level, stream_id, is_active)
		VALUES ($1, $2, $3, $4, 'G4', $5, true)`,
		classID, tenantID, schoolID, academicYearID, streamID)
	require.NoError(t, err)

	params := CreateClassTeacherParams{
		TenantID:    tenantID,
		ClassID:     classID,
		UserID:      userID,
		TeacherRole: "PRIMARY_CLASS_TEACHER",
	}

	id, err := repo.Create(ctx, params)
	require.NoError(t, err)
	require.NotEmpty(t, id)

	ct, err := repo.GetByID(ctx, id, tenantID)
	require.NoError(t, err)
	require.Equal(t, id, ct.ID)
	require.Equal(t, userID, ct.UserID)
	require.Equal(t, "PRIMARY_CLASS_TEACHER", ct.TeacherRole)
}

func TestPgRepository_DuplicateCreate(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	ctx := context.Background()
	pool, cleanup := startPG(t)
	defer cleanup()
	applyMigration(t, pool, "000001_initial_schema.up.sql")

	tenantID, schoolID, userID := seedTenantSchoolUser(t, pool)
	academicYearID, streamID := seedAcademicYearAndStream(t, pool, tenantID, schoolID, userID)
	repo := newRepo(pool)

	classID := uuid.New().String()
	_, err := pool.Exec(ctx, `INSERT INTO cbc_classes (id, tenant_id, school_id, academic_year_id, grade_level, stream_id, is_active) VALUES ($1, $2, $3, $4, 'G4', $5, true)`,
		classID, tenantID, schoolID, academicYearID, streamID)
	require.NoError(t, err)

	params := CreateClassTeacherParams{
		TenantID:    tenantID,
		ClassID:     classID,
		UserID:      userID,
		TeacherRole: "PRIMARY_CLASS_TEACHER",
	}
	_, err = repo.Create(ctx, params)
	require.NoError(t, err)

	// Duplicate should fail
	_, err = repo.Create(ctx, params)
	require.Error(t, err)
	require.ErrorIs(t, err, ErrAlreadyExists)
}

func TestPgRepository_ListByClass(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	ctx := context.Background()
	pool, cleanup := startPG(t)
	defer cleanup()
	applyMigration(t, pool, "000001_initial_schema.up.sql")

	tenantID, schoolID, userID := seedTenantSchoolUser(t, pool)
	academicYearID, streamID := seedAcademicYearAndStream(t, pool, tenantID, schoolID, userID)
	repo := newRepo(pool)

	classID := uuid.New().String()
	_, err := pool.Exec(ctx, `INSERT INTO cbc_classes (id, tenant_id, school_id, academic_year_id, grade_level, stream_id, is_active) VALUES ($1, $2, $3, $4, 'G4', $5, true)`,
		classID, tenantID, schoolID, academicYearID, streamID)
	require.NoError(t, err)

	params := CreateClassTeacherParams{
		TenantID:    tenantID,
		ClassID:     classID,
		UserID:      userID,
		TeacherRole: "PRIMARY_CLASS_TEACHER",
	}
	_, err = repo.Create(ctx, params)
	require.NoError(t, err)

	items, err := repo.ListByClass(ctx, classID, tenantID)
	require.NoError(t, err)
	require.Len(t, items, 1)
	require.Equal(t, userID, items[0].UserID)
}

func TestPgRepository_ListByTeacher(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	ctx := context.Background()
	pool, cleanup := startPG(t)
	defer cleanup()
	applyMigration(t, pool, "000001_initial_schema.up.sql")

	tenantID, schoolID, userID := seedTenantSchoolUser(t, pool)
	academicYearID, streamID := seedAcademicYearAndStream(t, pool, tenantID, schoolID, userID)
	repo := newRepo(pool)

	classID1 := uuid.New().String()
	classID2 := uuid.New().String()
	_, err := pool.Exec(ctx, `INSERT INTO cbc_classes (id, tenant_id, school_id, academic_year_id, grade_level, stream_id, is_active) VALUES ($1, $2, $3, $4, 'G4', $5, true), ($6, $2, $3, $4, 'G5', $5, true)`,
		classID1, tenantID, schoolID, academicYearID, streamID, classID2)
	require.NoError(t, err)

	// SUBJECT_TEACHER rows require a non-null learning_area_id (enforced by
	// chk_cct_subject_area_required), so seed one and reference it.
	learningAreaID := seedLearningArea(t, pool, tenantID, schoolID)

	_, err = repo.Create(ctx, CreateClassTeacherParams{TenantID: tenantID, ClassID: classID1, UserID: userID, TeacherRole: "SUBJECT_TEACHER", LearningAreaID: ptrString(learningAreaID)})
	require.NoError(t, err)
	_, err = repo.Create(ctx, CreateClassTeacherParams{TenantID: tenantID, ClassID: classID2, UserID: userID, TeacherRole: "SUBJECT_TEACHER", LearningAreaID: ptrString(learningAreaID)})
	require.NoError(t, err)

	items, err := repo.ListByTeacher(ctx, userID, tenantID)
	require.NoError(t, err)
	require.Len(t, items, 2)
}

func TestPgRepository_Delete(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	ctx := context.Background()
	pool, cleanup := startPG(t)
	defer cleanup()
	applyMigration(t, pool, "000001_initial_schema.up.sql")

	tenantID, schoolID, userID := seedTenantSchoolUser(t, pool)
	academicYearID, streamID := seedAcademicYearAndStream(t, pool, tenantID, schoolID, userID)
	repo := newRepo(pool)

	classID := uuid.New().String()
	_, err := pool.Exec(ctx, `INSERT INTO cbc_classes (id, tenant_id, school_id, academic_year_id, grade_level, stream_id, is_active) VALUES ($1, $2, $3, $4, 'G4', $5, true)`,
		classID, tenantID, schoolID, academicYearID, streamID)
	require.NoError(t, err)

	params := CreateClassTeacherParams{TenantID: tenantID, ClassID: classID, UserID: userID, TeacherRole: "SUBSTITUTE_TEACHER"}
	id, err := repo.Create(ctx, params)
	require.NoError(t, err)

	err = repo.Delete(ctx, id, tenantID)
	require.NoError(t, err)

	// Verify gone
	_, err = repo.GetByID(ctx, id, tenantID)
	require.ErrorIs(t, err, ErrNotFound)
}

func TestPgRepository_CountPrimaryForClass(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	ctx := context.Background()
	pool, cleanup := startPG(t)
	defer cleanup()
	applyMigration(t, pool, "000001_initial_schema.up.sql")

	tenantID, schoolID, userID := seedTenantSchoolUser(t, pool)
	academicYearID, streamID := seedAcademicYearAndStream(t, pool, tenantID, schoolID, userID)
	repo := newRepo(pool)

	classID := uuid.New().String()
	_, err := pool.Exec(ctx, `INSERT INTO cbc_classes (id, tenant_id, school_id, academic_year_id, grade_level, stream_id, is_active) VALUES ($1, $2, $3, $4, 'G4', $5, true)`,
		classID, tenantID, schoolID, academicYearID, streamID)
	require.NoError(t, err)

	count, err := repo.CountPrimaryForClass(ctx, classID, tenantID)
	require.NoError(t, err)
	require.Equal(t, 0, count)

	_, err = repo.Create(ctx, CreateClassTeacherParams{TenantID: tenantID, ClassID: classID, UserID: userID, TeacherRole: "PRIMARY_CLASS_TEACHER"})
	require.NoError(t, err)

	count, err = repo.CountPrimaryForClass(ctx, classID, tenantID)
	require.NoError(t, err)
	require.Equal(t, 1, count)
}

func TestPgRepository_ExistsForSubject(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	ctx := context.Background()
	pool, cleanup := startPG(t)
	defer cleanup()
	applyMigration(t, pool, "000001_initial_schema.up.sql")

	tenantID, schoolID, userID := seedTenantSchoolUser(t, pool)
	academicYearID, streamID := seedAcademicYearAndStream(t, pool, tenantID, schoolID, userID)
	repo := newRepo(pool)

	classID := uuid.New().String()
	_, err := pool.Exec(ctx, `INSERT INTO cbc_classes (id, tenant_id, school_id, academic_year_id, grade_level, stream_id, is_active) VALUES ($1, $2, $3, $4, 'G4', $5, true)`,
		classID, tenantID, schoolID, academicYearID, streamID)
	require.NoError(t, err)

	// Use a real (but unassigned) learning-area id so the query can compare
	// against a UUID column instead of choking on an invalid literal.
	learningAreaID := seedLearningArea(t, pool, tenantID, schoolID)

	// No teacher record exists yet → should report false.
	exists, err := repo.ExistsForSubject(ctx, classID, userID, learningAreaID, tenantID)
	require.NoError(t, err)
	require.False(t, exists)

	// Insert a matching record → should now report true.
	_, err = repo.Create(ctx, CreateClassTeacherParams{TenantID: tenantID, ClassID: classID, UserID: userID, TeacherRole: "SUBJECT_TEACHER", LearningAreaID: ptrString(learningAreaID)})
	require.NoError(t, err)

	exists, err = repo.ExistsForSubject(ctx, classID, userID, learningAreaID, tenantID)
	require.NoError(t, err)
	require.True(t, exists)
}
