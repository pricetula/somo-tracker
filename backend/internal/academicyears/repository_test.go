package academicyears

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

func d(year int, month time.Month, day int) DateOnly {
	return DateOnly{Time: time.Date(year, month, day, 0, 0, 0, 0, time.UTC)}
}

func TestPgRepository_CreateAndListYears(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	ctx := context.Background()
	pool, cleanup := startPG(t)
	defer cleanup()

	applyMigration(t, pool, "000001_initial_schema.up.sql")

	tenantID, schoolID, userID := seedTenantSchoolUser(t, pool)
	repo := newRepo(pool)

	year := &AcademicYear{
		TenantID:  tenantID,
		SchoolID:  schoolID,
		Name:      "2026",
		StartDate: d(2026, 1, 1),
		EndDate:   d(2026, 12, 31),
		CreatedBy: userID,
		UpdatedBy: userID,
	}

	id, err := repo.CreateYear(ctx, year)
	require.NoError(t, err)
	require.NotEmpty(t, id)

	years, err := repo.ListYears(ctx, tenantID, schoolID)
	require.NoError(t, err)
	require.Len(t, years, 1)
	require.Equal(t, "2026", years[0].Name)
	require.False(t, years[0].IsCurrent)

	empty, err := repo.ListYears(ctx, tenantID, uuid.New().String())
	require.NoError(t, err)
	require.Empty(t, empty)
}

func TestPgRepository_GetYearByID(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	ctx := context.Background()
	pool, cleanup := startPG(t)
	defer cleanup()

	applyMigration(t, pool, "000001_initial_schema.up.sql")

	tenantID, schoolID, userID := seedTenantSchoolUser(t, pool)
	repo := newRepo(pool)

	year := &AcademicYear{
		TenantID:  tenantID,
		SchoolID:  schoolID,
		Name:      "2026",
		StartDate: d(2026, 1, 1),
		EndDate:   d(2026, 12, 31),
		CreatedBy: userID,
		UpdatedBy: userID,
	}

	id, err := repo.CreateYear(ctx, year)
	require.NoError(t, err)

	found, err := repo.GetYearByID(ctx, id, tenantID, schoolID)
	require.NoError(t, err)
	require.Equal(t, id, found.ID)
	require.Equal(t, "2026", found.Name)

	_, err = repo.GetYearByID(ctx, id, uuid.New().String(), schoolID)
	require.Error(t, err)
	require.ErrorIs(t, err, ErrNotFound)
}

func TestPgRepository_SetCurrentYear(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	ctx := context.Background()
	pool, cleanup := startPG(t)
	defer cleanup()

	applyMigration(t, pool, "000001_initial_schema.up.sql")

	tenantID, schoolID, userID := seedTenantSchoolUser(t, pool)
	repo := newRepo(pool)

	year1 := &AcademicYear{TenantID: tenantID, SchoolID: schoolID, Name: "2025",
		StartDate: d(2025, 1, 1), EndDate: d(2025, 12, 31),
		CreatedBy: userID, UpdatedBy: userID}
	id1, err := repo.CreateYear(ctx, year1)
	require.NoError(t, err)

	year2 := &AcademicYear{TenantID: tenantID, SchoolID: schoolID, Name: "2026",
		StartDate: d(2026, 1, 1), EndDate: d(2026, 12, 31),
		CreatedBy: userID, UpdatedBy: userID}
	id2, err := repo.CreateYear(ctx, year2)
	require.NoError(t, err)

	changed, err := repo.SetCurrentYear(ctx, id2, tenantID, schoolID, userID)
	require.NoError(t, err)
	require.True(t, changed)

	var isCurrent bool
	err = pool.QueryRow(ctx, `SELECT is_current FROM academic_years WHERE id = $1`, id1).Scan(&isCurrent)
	require.NoError(t, err)
	require.False(t, isCurrent)

	err = pool.QueryRow(ctx, `SELECT is_current FROM academic_years WHERE id = $1`, id2).Scan(&isCurrent)
	require.NoError(t, err)
	require.True(t, isCurrent)
}

func TestPgRepository_ActivateTerm(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	ctx := context.Background()
	pool, cleanup := startPG(t)
	defer cleanup()

	applyMigration(t, pool, "000001_initial_schema.up.sql")

	tenantID, schoolID, userID := seedTenantSchoolUser(t, pool)
	repo := newRepo(pool)

	// Two years, each with one term, so cross-year activation is exercised.
	year1 := &AcademicYear{TenantID: tenantID, SchoolID: schoolID, Name: "2025",
		StartDate: d(2025, 1, 1), EndDate: d(2025, 12, 31),
		CreatedBy: userID, UpdatedBy: userID}
	year1ID, err := repo.CreateYear(ctx, year1)
	require.NoError(t, err)
	year2 := &AcademicYear{TenantID: tenantID, SchoolID: schoolID, Name: "2026",
		StartDate: d(2026, 1, 1), EndDate: d(2026, 12, 31),
		CreatedBy: userID, UpdatedBy: userID}
	year2ID, err := repo.CreateYear(ctx, year2)
	require.NoError(t, err)

	term1 := &AcademicTerm{TenantID: tenantID, SchoolID: schoolID, AcademicYearID: year1ID,
		Name: "Term 1", TermNumber: 1, StartDate: d(2025, 1, 1), EndDate: d(2025, 4, 30),
		CreatedBy: userID, UpdatedBy: userID}
	term1ID, err := repo.CreateTerm(ctx, term1)
	require.NoError(t, err)
	term2 := &AcademicTerm{TenantID: tenantID, SchoolID: schoolID, AcademicYearID: year2ID,
		Name: "Term 1", TermNumber: 1, StartDate: d(2026, 1, 1), EndDate: d(2026, 4, 30),
		CreatedBy: userID, UpdatedBy: userID}
	term2ID, err := repo.CreateTerm(ctx, term2)
	require.NoError(t, err)

	// Activate term 1 of year 1.
	activated, err := repo.ActivateTerm(ctx, term1ID, tenantID, schoolID, userID)
	require.NoError(t, err)
	require.Equal(t, term1ID, activated.ID)
	require.True(t, activated.IsCurrent)
	require.Equal(t, 2, activated.Version)

	assertTermCurrent := func(termID string, want bool) {
		t.Helper()
		var got bool
		err := pool.QueryRow(ctx, `SELECT is_current FROM academic_terms WHERE id = $1`, termID).Scan(&got)
		require.NoError(t, err)
		require.Equal(t, want, got)
	}
	assertYearCurrent := func(yearID string, want bool) {
		t.Helper()
		var got bool
		err := pool.QueryRow(ctx, `SELECT is_current FROM academic_years WHERE id = $1`, yearID).Scan(&got)
		require.NoError(t, err)
		require.Equal(t, want, got)
	}
	assertTermCurrent(term1ID, true)
	assertTermCurrent(term2ID, false)
	assertYearCurrent(year1ID, true)
	assertYearCurrent(year2ID, false)

	// Switch to term 1 of year 2 — old term AND old year must be cleared.
	activated, err = repo.ActivateTerm(ctx, term2ID, tenantID, schoolID, userID)
	require.NoError(t, err)
	require.Equal(t, term2ID, activated.ID)
	require.True(t, activated.IsCurrent)

	assertTermCurrent(term1ID, false)
	assertTermCurrent(term2ID, true)
	assertYearCurrent(year1ID, false)
	assertYearCurrent(year2ID, true)

	// Audit fields: updated_by must be the actor on the newly-activated term.
	var updatedBy string
	err = pool.QueryRow(ctx, `SELECT updated_by FROM academic_terms WHERE id = $1`, term2ID).Scan(&updatedBy)
	require.NoError(t, err)
	require.Equal(t, userID, updatedBy)

	// Idempotent re-activation of the already-current term is a no-op success.
	_, err = repo.ActivateTerm(ctx, term2ID, tenantID, schoolID, userID)
	require.NoError(t, err)
	assertTermCurrent(term2ID, true)
	assertYearCurrent(year2ID, true)

	// Tenant isolation: activating against another tenant returns not found.
	_, err = repo.ActivateTerm(ctx, term2ID, uuid.New().String(), schoolID, userID)
	require.ErrorIs(t, err, ErrNotFound)
}

func TestPgRepository_TermDependencyCounts(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	ctx := context.Background()
	pool, cleanup := startPG(t)
	defer cleanup()

	applyMigration(t, pool, "000001_initial_schema.up.sql")

	tenantID, schoolID, userID := seedTenantSchoolUser(t, pool)
	repo := newRepo(pool)

	year := &AcademicYear{TenantID: tenantID, SchoolID: schoolID, Name: "2026",
		StartDate: d(2026, 1, 1), EndDate: d(2026, 12, 31),
		CreatedBy: userID, UpdatedBy: userID}
	yearID, err := repo.CreateYear(ctx, year)
	require.NoError(t, err)
	term := &AcademicTerm{TenantID: tenantID, SchoolID: schoolID, AcademicYearID: yearID,
		Name: "Term 1", TermNumber: 1, StartDate: d(2026, 1, 1), EndDate: d(2026, 4, 30),
		CreatedBy: userID, UpdatedBy: userID}
	termID, err := repo.CreateTerm(ctx, term)
	require.NoError(t, err)

	// No dependents yet.
	counts, err := repo.TermDependencyCounts(ctx, termID)
	require.NoError(t, err)
	require.Zero(t, counts["cbc_student_enrollments"])
	require.Zero(t, counts["fee_templates"])
	require.Zero(t, counts["invoices"])
	require.Zero(t, counts["attendance_records"])
	require.Zero(t, counts["assessment_sessions"])

	// Invoices require a real student (tenant-scoped FK).
	studentID := uuid.New().String()
	_, err = pool.Exec(ctx, `
		INSERT INTO cbc_students (id, tenant_id, school_id, full_name, gender)
		VALUES ($1, $2, $3, 'Test Student', 'M')
	`, studentID, tenantID, schoolID)
	require.NoError(t, err)

	// Insert one invoice referencing the term.
	_, err = pool.Exec(ctx, `
		INSERT INTO invoices (tenant_id, student_id, school_id, academic_term_id,
		                      invoice_label, payment_status, amount_due)
		VALUES ($1, $2, $3, $4, 'Term fee', 'UNPAID', 100)
	`, tenantID, studentID, schoolID, termID)
	require.NoError(t, err)

	counts, err = repo.TermDependencyCounts(ctx, termID)
	require.NoError(t, err)
	require.Equal(t, int64(1), counts["invoices"])
	require.Equal(t, int64(0), counts["cbc_student_enrollments"])
	require.Equal(t, int64(0), counts["attendance_records"])
}

func TestPgRepository_CountOrphansOutsideRange(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	ctx := context.Background()
	pool, cleanup := startPG(t)
	defer cleanup()

	applyMigration(t, pool, "000001_initial_schema.up.sql")

	tenantID, schoolID, userID := seedTenantSchoolUser(t, pool)
	repo := newRepo(pool)

	year := &AcademicYear{TenantID: tenantID, SchoolID: schoolID, Name: "2026",
		StartDate: d(2026, 1, 1), EndDate: d(2026, 12, 31),
		CreatedBy: userID, UpdatedBy: userID}
	yearID, err := repo.CreateYear(ctx, year)
	require.NoError(t, err)
	term := &AcademicTerm{TenantID: tenantID, SchoolID: schoolID, AcademicYearID: yearID,
		Name: "Term 1", TermNumber: 1, StartDate: d(2026, 1, 1), EndDate: d(2026, 4, 30),
		CreatedBy: userID, UpdatedBy: userID}
	termID, err := repo.CreateTerm(ctx, term)
	require.NoError(t, err)

	// An assessment session scheduled before the proposed new start date.
	// Needs a stream + class + learning area (all tenant-scoped).
	streamID := uuid.New().String()
	_, err = pool.Exec(ctx, `
		INSERT INTO cbc_streams (id, tenant_id, school_id, name)
		VALUES ($1, $2, $3, 'East')
	`, streamID, tenantID, schoolID)
	require.NoError(t, err)

	classID := uuid.New().String()
	_, err = pool.Exec(ctx, `
		INSERT INTO cbc_classes (id, tenant_id, school_id, academic_year_id, grade_level, stream_id)
		VALUES ($1, $2, $3, $4, 'G7', $5)
	`, classID, tenantID, schoolID, yearID, streamID)
	require.NoError(t, err)

	areaID := uuid.New().String()
	_, err = pool.Exec(ctx, `
		INSERT INTO cbc_learning_areas (id, tenant_id, school_id, name, code, education_level, grade_level)
		VALUES ($1, $2, $3, 'Mathematics', 'MAT', 'Junior_Secondary', 'G7')
	`, areaID, tenantID, schoolID)
	require.NoError(t, err)

	// QUANTITATIVE assessment sessions require a grading scale profile.
	scaleID := uuid.New().String()
	_, err = pool.Exec(ctx, `
		INSERT INTO grading_scale_profiles (id, tenant_id, school_id, name)
		VALUES ($1, $2, $3, 'CBC Scale')
	`, scaleID, tenantID, schoolID)
	require.NoError(t, err)

	_, err = pool.Exec(ctx, `
		INSERT INTO assessment_sessions (tenant_id, school_id, class_id, learning_area_id,
		                                academic_term_id, academic_year_id, name, evaluation_method,
		                                max_points, grading_scale_profile_id, status, scheduled_date, created_by)
		VALUES ($1, $2, $3, $4, $5, $6, 'Early Quiz', 'QUANTITATIVE', 20, $7, 'DRAFT', $8::date, $9)
	`, tenantID, schoolID, classID, areaID, termID, yearID, scaleID, d(2026, 1, 15), userID)
	require.NoError(t, err)

	// No orphans if the range still covers the recorded date.
	counts, err := repo.CountOrphansOutsideRange(ctx, termID, d(2026, 1, 10).Time, d(2026, 4, 30).Time)
	require.NoError(t, err)
	require.Equal(t, int64(0), counts["assessment_sessions"])

	// Moving the start to Feb 1 orphans the Jan 15 session.
	counts, err = repo.CountOrphansOutsideRange(ctx, termID, d(2026, 2, 1).Time, d(2026, 4, 30).Time)
	require.NoError(t, err)
	require.Equal(t, int64(1), counts["assessment_sessions"])
}

func TestPgRepository_UpdateTerm_IsFinal(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	ctx := context.Background()
	pool, cleanup := startPG(t)
	defer cleanup()

	applyMigration(t, pool, "000001_initial_schema.up.sql")

	tenantID, schoolID, userID := seedTenantSchoolUser(t, pool)
	repo := newRepo(pool)

	year := &AcademicYear{TenantID: tenantID, SchoolID: schoolID, Name: "2026",
		StartDate: d(2026, 1, 1), EndDate: d(2026, 12, 31),
		CreatedBy: userID, UpdatedBy: userID}
	yearID, err := repo.CreateYear(ctx, year)
	require.NoError(t, err)
	term := &AcademicTerm{TenantID: tenantID, SchoolID: schoolID, AcademicYearID: yearID,
		Name: "Term 1", TermNumber: 1, StartDate: d(2026, 1, 1), EndDate: d(2026, 4, 30),
		CreatedBy: userID, UpdatedBy: userID}
	termID, err := repo.CreateTerm(ctx, term)
	require.NoError(t, err)

	// Flip is_final via UpdateTerm.
	upd := &AcademicTerm{ID: termID, Name: "Term 1", Version: 1,
		StartDate: d(2026, 1, 1), EndDate: d(2026, 4, 30), IsFinal: true,
		UpdatedBy: userID}
	err = repo.UpdateTerm(ctx, upd)
	require.NoError(t, err)

	var isFinal bool
	err = pool.QueryRow(ctx, `SELECT is_final FROM academic_terms WHERE id = $1`, termID).Scan(&isFinal)
	require.NoError(t, err)
	require.True(t, isFinal)

	// Bounds trigger (P0001) is translated to TermOutOfYearBoundsError.
	bad := &AcademicTerm{ID: termID, Name: "Term 1", Version: 2,
		StartDate: d(2026, 12, 1), EndDate: d(2027, 1, 31), IsFinal: false,
		UpdatedBy: userID}
	err = repo.UpdateTerm(ctx, bad)
	require.Error(t, err)
	var outOfBounds *TermOutOfYearBoundsError
	require.ErrorAs(t, err, &outOfBounds)

	// Exclusion constraint (23P01) is translated to TermDateOverlapError.
	other := &AcademicTerm{TenantID: tenantID, SchoolID: schoolID, AcademicYearID: yearID,
		Name: "Term 2", TermNumber: 2, StartDate: d(2026, 5, 1), EndDate: d(2026, 8, 31),
		CreatedBy: userID, UpdatedBy: userID}
	otherID, err := repo.CreateTerm(ctx, other)
	require.NoError(t, err)

	overlap := &AcademicTerm{ID: otherID, Name: "Term 2", Version: 1,
		StartDate: d(2026, 4, 1), EndDate: d(2026, 6, 30), IsFinal: false,
		UpdatedBy: userID}
	err = repo.UpdateTerm(ctx, overlap)
	require.Error(t, err)
	var overlapErr *TermDateOverlapError
	require.ErrorAs(t, err, &overlapErr)
}

func TestPgRepository_CreateAndListTerms(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	ctx := context.Background()
	pool, cleanup := startPG(t)
	defer cleanup()

	applyMigration(t, pool, "000001_initial_schema.up.sql")

	tenantID, schoolID, userID := seedTenantSchoolUser(t, pool)
	repo := newRepo(pool)

	year := &AcademicYear{TenantID: tenantID, SchoolID: schoolID, Name: "2026",
		StartDate: d(2026, 1, 1), EndDate: d(2026, 12, 31),
		CreatedBy: userID, UpdatedBy: userID}
	yearID, err := repo.CreateYear(ctx, year)
	require.NoError(t, err)

	term1 := &AcademicTerm{
		TenantID:       tenantID,
		SchoolID:       schoolID,
		AcademicYearID: yearID,
		Name:           "Term 1",
		TermNumber:     1,
		StartDate:      d(2026, 1, 1),
		EndDate:        d(2026, 4, 30),
		CreatedBy:      userID,
		UpdatedBy:      userID,
	}
	termID, err := repo.CreateTerm(ctx, term1)
	require.NoError(t, err)
	require.NotEmpty(t, termID)

	terms, err := repo.ListTerms(ctx, tenantID, schoolID, nil)
	require.NoError(t, err)
	require.Len(t, terms, 1)
	require.Equal(t, "Term 1", terms[0].Name)
	require.Equal(t, 1, terms[0].TermNumber)
}
