package attendance

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/google/uuid"
	"github.com/hibiken/asynq"
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

func newRepo(pool *pgxpool.Pool) Repository {
	return NewRepository(&database.Pools{PG: pool})
}

// testIDs holds common IDs used across integration tests.
type testIDs struct {
	TenantID        string
	SchoolID        string
	UserID          string
	AcademicYearID  string
	AcademicTermID  string
	ClassID1        string
	ClassID2        string
	LearningAreaID1 string
	LearningAreaID2 string
	StudentID1      string
	StudentID2      string
}

// setupTestTables applies migrations and seeds common data for integration tests.
func setupTestTables(t *testing.T, pool *pgxpool.Pool) testIDs {
	t.Helper()
	ctx := context.Background()

	// Apply base schema and initial extensions
	applyMigration(t, pool, "000001_initial_schema.up.sql")
	applyMigration(t, pool, "000005_extend_summaries_and_daily.up.sql")
	applyMigration(t, pool, "000016_create_class_attendance_rollups.up.sql")

	tenantID, schoolID, userID := seedTenantSchoolUser(t, pool)

	academicYearID := uuid.New().String()
	_, err := pool.Exec(ctx, `INSERT INTO academic_years (id, tenant_id, school_id, name, start_date, end_date, is_current, created_by, updated_by) VALUES ($1, $2, $3, '2026', '2026-01-01', '2026-12-31', true, $4, $4)`,
		academicYearID, tenantID, schoolID, userID)
	require.NoError(t, err)

	academicTermID := uuid.New().String()
	_, err = pool.Exec(ctx, `INSERT INTO academic_terms (id, tenant_id, school_id, academic_year_id, name, term_number, start_date, end_date, is_current, is_final, created_by, updated_by) VALUES ($1, $2, $3, $4, 'Term 1', 1, '2026-01-01', '2026-03-31', true, false, $5, $5)`,
		academicTermID, tenantID, schoolID, academicYearID, userID)
	require.NoError(t, err)

	streamID1 := uuid.New().String()
	_, err = pool.Exec(ctx, `INSERT INTO cbc_streams (id, tenant_id, school_id, name) VALUES ($1, $2, $3, 'Blue')`,
		streamID1, tenantID, schoolID)
	require.NoError(t, err)

	streamID2 := uuid.New().String()
	_, err = pool.Exec(ctx, `INSERT INTO cbc_streams (id, tenant_id, school_id, name) VALUES ($1, $2, $3, 'Green')`,
		streamID2, tenantID, schoolID)
	require.NoError(t, err)

	classID1 := uuid.New().String()
	_, err = pool.Exec(ctx, `INSERT INTO cbc_classes (id, tenant_id, school_id, academic_year_id, grade_level, stream_id, is_active) VALUES ($1, $2, $3, $4, 'G1', $5, true)`,
		classID1, tenantID, schoolID, academicYearID, streamID1)
	require.NoError(t, err)

	classID2 := uuid.New().String()
	_, err = pool.Exec(ctx, `INSERT INTO cbc_classes (id, tenant_id, school_id, academic_year_id, grade_level, stream_id, is_active) VALUES ($1, $2, $3, $4, 'G1', $5, true)`,
		classID2, tenantID, schoolID, academicYearID, streamID2)
	require.NoError(t, err)

	learningAreaID1 := uuid.New().String()
	_, err = pool.Exec(ctx, `INSERT INTO cbc_learning_areas (id, tenant_id, school_id, name, code, education_level, grade_level) VALUES ($1, $2, $3, 'Mathematics', 'MATH', 'Early_Years', 'G1')`,
		learningAreaID1, tenantID, schoolID)
	require.NoError(t, err)

	learningAreaID2 := uuid.New().String()
	_, err = pool.Exec(ctx, `INSERT INTO cbc_learning_areas (id, tenant_id, school_id, name, code, education_level, grade_level) VALUES ($1, $2, $3, 'English', 'ENG', 'Early_Years', 'G1')`,
		learningAreaID2, tenantID, schoolID)
	require.NoError(t, err)

	studentID1 := uuid.New().String()
	_, err = pool.Exec(ctx, `INSERT INTO cbc_students (id, tenant_id, school_id, full_name, gender, learning_pathway) VALUES ($1, $2, $3, 'Alice Smith', 'F', 'Age_Based')`,
		studentID1, tenantID, schoolID)
	require.NoError(t, err)
	_, err = pool.Exec(ctx, `INSERT INTO cbc_student_enrollments (id, tenant_id, school_id, student_id, academic_term_id, class_id) VALUES ($1, $2, $3, $4, $5, $6)`,
		uuid.New().String(), tenantID, schoolID, studentID1, academicTermID, classID1)
	require.NoError(t, err)

	studentID2 := uuid.New().String()
	_, err = pool.Exec(ctx, `INSERT INTO cbc_students (id, tenant_id, school_id, full_name, gender, learning_pathway) VALUES ($1, $2, $3, 'Bob Johnson', 'M', 'Age_Based')`,
		studentID2, tenantID, schoolID)
	require.NoError(t, err)
	_, err = pool.Exec(ctx, `INSERT INTO cbc_student_enrollments (id, tenant_id, school_id, student_id, academic_term_id, class_id) VALUES ($1, $2, $3, $4, $5, $6)`,
		uuid.New().String(), tenantID, schoolID, studentID2, academicTermID, classID2)
	require.NoError(t, err)

	return testIDs{
		TenantID:        tenantID,
		SchoolID:        schoolID,
		UserID:          userID,
		AcademicYearID:  academicYearID,
		AcademicTermID:  academicTermID,
		ClassID1:        classID1,
		ClassID2:        classID2,
		LearningAreaID1: learningAreaID1,
		LearningAreaID2: learningAreaID2,
		StudentID1:      studentID1,
		StudentID2:      studentID2,
	}
}

// insertAttendanceTermSummary helper to populate attendance_term_summaries.
func insertAttendanceTermSummary(t *testing.T, pool *pgxpool.Pool, ids testIDs, studentID, learningAreaID string, periodsTotal, periodsPresent, periodsAbsent, periodsLate, periodsExcused int) {
	t.Helper()
	ctx := context.Background()
	_, err := pool.Exec(ctx, `
		INSERT INTO attendance_term_summaries (
			id, tenant_id, school_id, student_id, academic_term_id, academic_year_id,
			learning_area_id, periods_total, periods_present, periods_absent, periods_late, periods_excused,
			attendance_percentage, last_refreshed_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, NOW())
		ON CONFLICT (student_id, academic_term_id, learning_area_id) DO UPDATE SET
			periods_total = EXCLUDED.periods_total, periods_present = EXCLUDED.periods_present,
			periods_absent = EXCLUDED.periods_absent, periods_late = EXCLUDED.periods_late,
			periods_excused = EXCLUDED.periods_excused, attendance_percentage = EXCLUDED.attendance_percentage,
			last_refreshed_at = NOW()
	`, uuid.New().String(), ids.TenantID, ids.SchoolID, studentID, ids.AcademicTermID, ids.AcademicYearID,
		learningAreaID, periodsTotal, periodsPresent, periodsAbsent, periodsLate, periodsExcused,
		float64(periodsPresent)*100.0/float64(periodsTotal),
	)
	require.NoError(t, err)
}

// insertClassDailyAttendanceSummary helper to populate class_daily_attendance_summaries.
func insertClassDailyAttendanceSummary(t *testing.T, pool *pgxpool.Pool, ids testIDs, classID, date string, totalEnrolled, presentCount, absentCount, lateCount, excusedCount int) {
	t.Helper()
	ctx := context.Background()
	_, err := pool.Exec(ctx, `
		INSERT INTO class_daily_attendance_summaries (
			id, tenant_id, school_id, class_id, academic_term_id, date,
			total_enrolled, present_count, absent_count, late_count, excused_count,
			daily_attendance_rate, last_refreshed_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, NOW())
		ON CONFLICT (class_id, date) DO UPDATE SET
			total_enrolled = EXCLUDED.total_enrolled, present_count = EXCLUDED.present_count,
			absent_count = EXCLUDED.absent_count, late_count = EXCLUDED.late_count,
			excused_count = EXCLUDED.excused_count, daily_attendance_rate = EXCLUDED.daily_attendance_rate,
			last_refreshed_at = NOW()
	`, uuid.New().String(), ids.TenantID, ids.SchoolID, classID, ids.AcademicTermID, date,
		totalEnrolled, presentCount, absentCount, lateCount, excusedCount,
		float64(presentCount)*100.0/float64(totalEnrolled),
	)
	require.NoError(t, err)
}

func TestPgRepository_CreateAndGetSession(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	ctx := context.Background()
	pool, cleanup := startPG(t)
	defer cleanup()
	applyMigration(t, pool, "000001_initial_schema.up.sql")

	tenantID, schoolID, _ := seedTenantSchoolUser(t, pool)
	repo := newRepo(pool)

	slotID := uuid.New().String()
	_, err := pool.Exec(ctx, `INSERT INTO cbc_timetable_slots (id, tenant_id, school_id, day_of_week, start_time, end_time, period_name, is_break) VALUES ($1, $2, $3, 1, '08:00', '08:40', 'Period 1', false)`,
		slotID, tenantID, schoolID)
	require.NoError(t, err)

	session, err := repo.CreateSession(ctx, tenantID, schoolID, CreateSessionPayload{
		TimetableSlotID: slotID,
		Date:            "2026-01-15",
		Status:          string(SessionSubmitted),
	})
	require.NoError(t, err)
	require.NotNil(t, session)
	require.Equal(t, string(SessionSubmitted), string(session.Status))

	fetched, err := repo.GetSessionByID(ctx, session.ID, tenantID)
	require.NoError(t, err)
	require.Equal(t, session.ID, fetched.ID)
}

func TestPgRepository_GetSession_NotFound(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	ctx := context.Background()
	pool, cleanup := startPG(t)
	defer cleanup()
	applyMigration(t, pool, "000001_initial_schema.up.sql")

	repo := newRepo(pool)
	_, err := repo.GetSessionByID(ctx, "missing_id", "tenant_001")
	require.Error(t, err)
	require.ErrorIs(t, err, ErrNotFound)
}

// runWorkerJob is a helper to synchronously execute an Asynq task handler.
func runWorkerJob(t *testing.T, w *Worker, taskType string, payload interface{}) {
	t.Helper()
	payloadBytes, err := json.Marshal(payload)
	require.NoError(t, err)
	task := asynq.NewTask(taskType, payloadBytes)
	ctx := context.Background()
	var handleErr error
	switch taskType {
	case TaskRefreshClassLearningAreaTermSummary:
		handleErr = w.handleClassLearningAreaTermRefresh(ctx, task)
	case TaskRefreshClassTermSummary:
		handleErr = w.handleClassTermRefresh(ctx, task)
	default:
		t.Fatalf("unknown task type: %s", taskType)
	}
	require.NoError(t, handleErr)
}

func TestPgRepository_GetClassLearningAreaTermSummary(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	ctx := context.Background()
	pool, cleanup := startPG(t)
	defer cleanup()

	ids := setupTestTables(t, pool)
	repo := newRepo(pool)

	// Seed attendance_term_summaries. setupTestTables creates 2 students in 2 classes.
	// We will insert data for student 1 (class 1) in math, and student 2 (class 2) in eng.
	insertAttendanceTermSummary(t, pool, ids, ids.StudentID1, ids.LearningAreaID1, 10, 8, 1, 1, 0)  // Class 1, Math
	insertAttendanceTermSummary(t, pool, ids, ids.StudentID2, ids.LearningAreaID2, 20, 15, 3, 2, 0) // Class 2, Eng

	// Instantiate a worker to manually run the job to populate the rollup table.
	// We pass the pool directly.
	w := &Worker{pools: &database.Pools{PG: pool}}

	// Run the class learning area term refresh job for the whole school/term.
	runWorkerJob(t, w, TaskRefreshClassLearningAreaTermSummary, ClassLearningAreaTermRefreshPayload{
		TenantID: ids.TenantID,
		SchoolID: ids.SchoolID,
		TermID:   ids.AcademicTermID,
	})

	t.Run("found", func(t *testing.T) {
		summary, err := repo.GetClassLearningAreaTermSummary(ctx, ids.TenantID, ids.SchoolID, ids.ClassID1, ids.LearningAreaID1, ids.AcademicTermID)
		require.NoError(t, err)
		require.NotNil(t, summary)
		require.Equal(t, ids.TenantID, summary.TenantID)
		require.Equal(t, ids.SchoolID, summary.SchoolID)
		require.Equal(t, ids.ClassID1, summary.ClassID)
		require.Equal(t, ids.LearningAreaID1, summary.LearningAreaID)
		require.Equal(t, ids.AcademicTermID, summary.AcademicTermID)
		require.Equal(t, 1, summary.StudentsIncluded)
		require.Equal(t, 10, summary.PeriodsTotal)
		require.Equal(t, 8, summary.PeriodsPresent)
		require.Equal(t, 1, summary.PeriodsAbsent)
		require.Equal(t, 1, summary.PeriodsLate)
		require.Equal(t, 0, summary.PeriodsExcused)
		require.InDelta(t, 80.0, summary.AttendancePercentage, 0.01)
	})

	t.Run("not_found", func(t *testing.T) {
		_, err := repo.GetClassLearningAreaTermSummary(ctx, ids.TenantID, ids.SchoolID, uuid.New().String(), ids.LearningAreaID1, ids.AcademicTermID)
		require.Error(t, err)
		require.ErrorIs(t, err, ErrNotFound)
	})

	t.Run("rls_enforced", func(t *testing.T) {
		// Different tenant trying to access the summary.
		wrongTenantID := uuid.New().String()
		_, err := repo.GetClassLearningAreaTermSummary(ctx, wrongTenantID, ids.SchoolID, ids.ClassID1, ids.LearningAreaID1, ids.AcademicTermID)
		require.Error(t, err)
		require.ErrorIs(t, err, ErrNotFound)
	})
}

func TestPgRepository_ListClassLearningAreaTermSummaries(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	ctx := context.Background()
	pool, cleanup := startPG(t)
	defer cleanup()

	ids := setupTestTables(t, pool)
	repo := newRepo(pool)

	insertAttendanceTermSummary(t, pool, ids, ids.StudentID1, ids.LearningAreaID1, 10, 8, 1, 1, 0)  // Class 1, Math
	insertAttendanceTermSummary(t, pool, ids, ids.StudentID1, ids.LearningAreaID2, 10, 5, 5, 0, 0)  // Class 1, Eng
	insertAttendanceTermSummary(t, pool, ids, ids.StudentID2, ids.LearningAreaID1, 20, 15, 3, 2, 0) // Class 2, Math

	w := &Worker{pools: &database.Pools{PG: pool}}
	runWorkerJob(t, w, TaskRefreshClassLearningAreaTermSummary, ClassLearningAreaTermRefreshPayload{
		TenantID: ids.TenantID,
		SchoolID: ids.SchoolID,
		TermID:   ids.AcademicTermID,
	})

	t.Run("list_all_in_term", func(t *testing.T) {
		summaries, err := repo.ListClassLearningAreaTermSummaries(ctx, ids.TenantID, ids.SchoolID, "", "", ids.AcademicTermID)
		require.NoError(t, err)
		require.Len(t, summaries, 3)
	})

	t.Run("filter_by_class", func(t *testing.T) {
		summaries, err := repo.ListClassLearningAreaTermSummaries(ctx, ids.TenantID, ids.SchoolID, ids.ClassID1, "", ids.AcademicTermID)
		require.NoError(t, err)
		require.Len(t, summaries, 2)
		for _, s := range summaries {
			require.Equal(t, ids.ClassID1, s.ClassID)
		}
	})

	t.Run("filter_by_learning_area", func(t *testing.T) {
		summaries, err := repo.ListClassLearningAreaTermSummaries(ctx, ids.TenantID, ids.SchoolID, "", ids.LearningAreaID1, ids.AcademicTermID)
		require.NoError(t, err)
		require.Len(t, summaries, 2) // Class 1 math, Class 2 math
		for _, s := range summaries {
			require.Equal(t, ids.LearningAreaID1, s.LearningAreaID)
		}
	})

	t.Run("rls_enforced", func(t *testing.T) {
		wrongTenantID := uuid.New().String()
		summaries, err := repo.ListClassLearningAreaTermSummaries(ctx, wrongTenantID, ids.SchoolID, "", "", ids.AcademicTermID)
		require.NoError(t, err)
		require.Empty(t, summaries)
	})
}

func TestPgRepository_GetClassTermAttendanceSummary(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	ctx := context.Background()
	pool, cleanup := startPG(t)
	defer cleanup()

	ids := setupTestTables(t, pool)
	repo := newRepo(pool)

	// Seed class_daily_attendance_summaries.
	insertClassDailyAttendanceSummary(t, pool, ids, ids.ClassID1, "2026-01-15", 30, 25, 3, 2, 0)
	insertClassDailyAttendanceSummary(t, pool, ids, ids.ClassID1, "2026-01-16", 32, 28, 2, 1, 1)
	insertClassDailyAttendanceSummary(t, pool, ids, ids.ClassID2, "2026-01-15", 30, 20, 5, 3, 2)

	w := &Worker{pools: &database.Pools{PG: pool}}
	runWorkerJob(t, w, TaskRefreshClassTermSummary, ClassTermRefreshPayload{
		TenantID: ids.TenantID,
		SchoolID: ids.SchoolID,
		TermID:   ids.AcademicTermID,
	})

	t.Run("found", func(t *testing.T) {
		summary, err := repo.GetClassTermAttendanceSummary(ctx, ids.TenantID, ids.SchoolID, ids.ClassID1, ids.AcademicTermID)
		require.NoError(t, err)
		require.NotNil(t, summary)
		require.Equal(t, ids.TenantID, summary.TenantID)
		require.Equal(t, ids.SchoolID, summary.SchoolID)
		require.Equal(t, ids.ClassID1, summary.ClassID)
		require.Equal(t, ids.AcademicTermID, summary.AcademicTermID)
		require.Equal(t, 2, summary.DaysInTerm)
		require.Equal(t, 53, summary.PresentCount)                  // 25 + 28
		require.Equal(t, 5, summary.AbsentCount)                    // 3 + 2
		require.Equal(t, 3, summary.LateCount)                      // 2 + 1
		require.Equal(t, 1, summary.ExcusedCount)                   // 0 + 1
		require.InDelta(t, 31.0, summary.TotalEnrolledAvg, 0.01)    // (30+32)/2
		require.InDelta(t, 85.48, summary.TermAttendanceRate, 0.01) // 53/(53+5+3+1) * 100
	})

	t.Run("not_found", func(t *testing.T) {
		_, err := repo.GetClassTermAttendanceSummary(ctx, ids.TenantID, ids.SchoolID, uuid.New().String(), ids.AcademicTermID)
		require.Error(t, err)
		require.ErrorIs(t, err, ErrNotFound)
	})

	t.Run("rls_enforced", func(t *testing.T) {
		wrongTenantID := uuid.New().String()
		_, err := repo.GetClassTermAttendanceSummary(ctx, wrongTenantID, ids.SchoolID, ids.ClassID1, ids.AcademicTermID)
		require.Error(t, err)
		require.ErrorIs(t, err, ErrNotFound)
	})
}

func TestPgRepository_ListClassTermAttendanceSummaries(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	ctx := context.Background()
	pool, cleanup := startPG(t)
	defer cleanup()

	ids := setupTestTables(t, pool)
	repo := newRepo(pool)

	insertClassDailyAttendanceSummary(t, pool, ids, ids.ClassID1, "2026-01-15", 30, 25, 3, 2, 0)
	insertClassDailyAttendanceSummary(t, pool, ids, ids.ClassID2, "2026-01-15", 30, 20, 5, 3, 2)

	w := &Worker{pools: &database.Pools{PG: pool}}
	runWorkerJob(t, w, TaskRefreshClassTermSummary, ClassTermRefreshPayload{
		TenantID: ids.TenantID,
		SchoolID: ids.SchoolID,
		TermID:   ids.AcademicTermID,
	})

	t.Run("list_all_in_term", func(t *testing.T) {
		summaries, err := repo.ListClassTermAttendanceSummaries(ctx, ids.TenantID, ids.SchoolID, "", ids.AcademicTermID)
		require.NoError(t, err)
		require.Len(t, summaries, 2)
	})

	t.Run("filter_by_class", func(t *testing.T) {
		summaries, err := repo.ListClassTermAttendanceSummaries(ctx, ids.TenantID, ids.SchoolID, ids.ClassID1, ids.AcademicTermID)
		require.NoError(t, err)
		require.Len(t, summaries, 1)
		require.Equal(t, ids.ClassID1, summaries[0].ClassID)
	})

	t.Run("rls_enforced", func(t *testing.T) {
		wrongTenantID := uuid.New().String()
		summaries, err := repo.ListClassTermAttendanceSummaries(ctx, wrongTenantID, ids.SchoolID, "", ids.AcademicTermID)
		require.NoError(t, err)
		require.Empty(t, summaries)
	})
}
