package attendance

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/hibiken/asynq"
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

	// Apply the squashed base schema (000001 contains everything incl. rollups)
	applyAllMigrations(t, pool)

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
	_, err = pool.Exec(ctx, `INSERT INTO cbc_student_enrollments (id, tenant_id, school_id, student_id, academic_term_id, academic_year_id, class_id) VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		uuid.New().String(), tenantID, schoolID, studentID1, academicTermID, academicYearID, classID1)
	require.NoError(t, err)

	studentID2 := uuid.New().String()
	_, err = pool.Exec(ctx, `INSERT INTO cbc_students (id, tenant_id, school_id, full_name, gender, learning_pathway) VALUES ($1, $2, $3, 'Bob Johnson', 'M', 'Age_Based')`,
		studentID2, tenantID, schoolID)
	require.NoError(t, err)
	_, err = pool.Exec(ctx, `INSERT INTO cbc_student_enrollments (id, tenant_id, school_id, student_id, academic_term_id, academic_year_id, class_id) VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		uuid.New().String(), tenantID, schoolID, studentID2, academicTermID, academicYearID, classID2)
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

	ids := setupTestTables(t, pool)
	repo := newRepo(pool)

	// Insert a timetable structure row
	structID := uuid.New().String()
	_, err := pool.Exec(ctx, `INSERT INTO timetable_blocks (id, tenant_id, school_id, academic_year_id, day_of_week, start_time, end_time, period_name) VALUES ($1, $2, $3, $4, 1, '08:00', '08:40', 'Period 1')`,
		structID, ids.TenantID, ids.SchoolID, ids.AcademicYearID)
	require.NoError(t, err)

	// Insert a timetable_allocation referencing the structure
	slotID := uuid.New().String()
	_, err = pool.Exec(ctx, `INSERT INTO timetable_allocations (id, tenant_id, school_id, academic_year_id, block_id, class_id, learning_area_id, teacher_id) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
		slotID, ids.TenantID, ids.SchoolID, ids.AcademicYearID, structID, ids.ClassID1, ids.LearningAreaID1, ids.UserID)
	require.NoError(t, err)

	session, err := repo.CreateSession(ctx, ids.TenantID, ids.SchoolID, CreateSessionPayload{
		TimetableAllocationID: slotID,
		Date:                  "2026-01-15",
		Status:                string(SessionSubmitted),
	})
	require.NoError(t, err)
	require.NotNil(t, session)
	require.Equal(t, string(SessionSubmitted), string(session.Status))

	fetched, err := repo.GetSessionByID(ctx, session.ID, ids.TenantID)
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
	applyAllMigrations(t, pool)

	repo := newRepo(pool)
	nonExistentID := uuid.New().String()
	_, err := repo.GetSessionByID(ctx, nonExistentID, "00000000-0000-0000-0000-000000000000")
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
	w := &Worker{pools: &database.Pools{PG: pool}, logger: zap.NewNop().Sugar()}

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

	w := &Worker{pools: &database.Pools{PG: pool}, logger: zap.NewNop().Sugar()}
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

	w := &Worker{pools: &database.Pools{PG: pool}, logger: zap.NewNop().Sugar()}
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

	w := &Worker{pools: &database.Pools{PG: pool}, logger: zap.NewNop().Sugar()}
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

func TestPgRepository_ListClassAttendanceBreakdowns(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	ctx := context.Background()
	pool, cleanup := startPG(t)
	defer cleanup()

	ids := setupTestTables(t, pool)
	repo := newRepo(pool)

	// ClassID1: 25 present / 3 late / 2 absent / 0 excused
	// ClassID2: 20 present / 3 late / 5 absent / 2 excused → higher absenteeism
	insertClassDailyAttendanceSummary(t, pool, ids, ids.ClassID1, "2026-01-15", 30, 25, 3, 2, 0)
	insertClassDailyAttendanceSummary(t, pool, ids, ids.ClassID2, "2026-01-15", 30, 20, 5, 3, 2)

	// A third class with NO daily/term summary — the LEFT JOIN must still
	// surface it with zeroed counts (ordered last via NULLS LAST).
	streamID3 := uuid.New().String()
	_, err := pool.Exec(ctx, `INSERT INTO cbc_streams (id, tenant_id, school_id, name) VALUES ($1, $2, $3, 'Red')`,
		streamID3, ids.TenantID, ids.SchoolID)
	require.NoError(t, err)
	classID3 := uuid.New().String()
	_, err = pool.Exec(ctx, `INSERT INTO cbc_classes (id, tenant_id, school_id, academic_year_id, grade_level, stream_id, is_active) VALUES ($1, $2, $3, $4, 'G2', $5, true)`,
		classID3, ids.TenantID, ids.SchoolID, ids.AcademicYearID, streamID3)
	require.NoError(t, err)

	w := &Worker{pools: &database.Pools{PG: pool}, logger: zap.NewNop().Sugar()}
	runWorkerJob(t, w, TaskRefreshClassTermSummary, ClassTermRefreshPayload{
		TenantID: ids.TenantID,
		SchoolID: ids.SchoolID,
		TermID:   ids.AcademicTermID,
	})

	t.Run("lists_all_classes_sorted_by_absent_desc", func(t *testing.T) {
		items, err := repo.ListClassAttendanceBreakdowns(ctx, ids.TenantID, ids.SchoolID, ids.AcademicTermID)
		require.NoError(t, err)
		require.Len(t, items, 3)

		// ClassID2 has 5 absents — must surface first (truancy watch).
		require.Equal(t, ids.ClassID2, items[0].ClassID)
		require.Equal(t, "G1 Green", items[0].ClassName)
		require.Equal(t, 5, items[0].AbsentCount)
		require.Equal(t, 20, items[0].PresentCount)
		require.Equal(t, 3, items[0].LateCount)
		require.Equal(t, 2, items[0].ExcusedCount)

		require.Equal(t, ids.ClassID1, items[1].ClassID)
		require.Equal(t, "G1 Blue", items[1].ClassName)
		require.Equal(t, 3, items[1].AbsentCount)

		// Class without a summary still appears, zeroed, ordered last.
		require.Equal(t, classID3, items[2].ClassID)
		require.Equal(t, "G2 Red", items[2].ClassName)
		require.Zero(t, items[2].AbsentCount)
		require.Zero(t, items[2].PresentCount)
		require.Zero(t, items[2].TermAttendanceRate)
	})

	t.Run("rls_enforced", func(t *testing.T) {
		wrongTenantID := uuid.New().String()
		items, err := repo.ListClassAttendanceBreakdowns(ctx, wrongTenantID, ids.SchoolID, ids.AcademicTermID)
		require.NoError(t, err)
		require.Empty(t, items)
	})
}

func TestPgRepository_ListLearningAreaBreakdowns(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	ctx := context.Background()
	pool, cleanup := startPG(t)
	defer cleanup()

	ids := setupTestTables(t, pool)
	repo := newRepo(pool)

	// Student1 (Class 1): Math 10 total / 8 present / 1 absent; English 10 total / 5 present / 5 absent.
	// Student2 (Class 2): Math 20 total / 15 present / 3 absent.
	// After the class-learning-area rollup: Math totals (30/23/4), English (10/5/5).
	insertAttendanceTermSummary(t, pool, ids, ids.StudentID1, ids.LearningAreaID1, 10, 8, 1, 1, 0)
	insertAttendanceTermSummary(t, pool, ids, ids.StudentID1, ids.LearningAreaID2, 10, 5, 5, 0, 0)
	insertAttendanceTermSummary(t, pool, ids, ids.StudentID2, ids.LearningAreaID1, 20, 15, 3, 2, 0)

	// A third learning area with NO summaries — the LEFT JOIN must still
	// surface it with zeroed counts (ordered last via NULLS LAST).
	learningAreaID3 := uuid.New().String()
	_, err := pool.Exec(ctx, `INSERT INTO cbc_learning_areas (id, tenant_id, school_id, name, code, education_level, grade_level) VALUES ($1, $2, $3, 'Kiswahili', 'KISW', 'Early_Years', 'G1')`,
		learningAreaID3, ids.TenantID, ids.SchoolID)
	require.NoError(t, err)

	w := &Worker{pools: &database.Pools{PG: pool}, logger: zap.NewNop().Sugar()}
	runWorkerJob(t, w, TaskRefreshClassLearningAreaTermSummary, ClassLearningAreaTermRefreshPayload{
		TenantID: ids.TenantID,
		SchoolID: ids.SchoolID,
		TermID:   ids.AcademicTermID,
	})

	t.Run("aggregates_across_classes_sorted_by_absent_desc", func(t *testing.T) {
		items, err := repo.ListLearningAreaBreakdowns(ctx, ids.TenantID, ids.SchoolID, ids.AcademicTermID)
		require.NoError(t, err)
		require.Len(t, items, 3)

		// English has 5 absent periods — must surface first (truancy hotspot).
		require.Equal(t, ids.LearningAreaID2, items[0].LearningAreaID)
		require.Equal(t, "English", items[0].LearningAreaName)
		require.Equal(t, 10, items[0].PeriodsTotal)
		require.Equal(t, 5, items[0].PeriodsPresent)
		require.Equal(t, 5, items[0].PeriodsAbsent)
		require.Zero(t, items[0].PeriodsExcused)
		require.InDelta(t, 50.00, items[0].AttendancePercentage, 0.01)

		// Math aggregates Class 1 + Class 2 rows: 30 total / 23 present / 4 absent.
		require.Equal(t, ids.LearningAreaID1, items[1].LearningAreaID)
		require.Equal(t, "Mathematics", items[1].LearningAreaName)
		require.Equal(t, 30, items[1].PeriodsTotal)
		require.Equal(t, 23, items[1].PeriodsPresent)
		require.Equal(t, 4, items[1].PeriodsAbsent)
		require.Zero(t, items[1].PeriodsExcused)
		require.InDelta(t, 76.67, items[1].AttendancePercentage, 0.01)

		// Learning area without summaries still appears, zeroed, ordered last.
		require.Equal(t, learningAreaID3, items[2].LearningAreaID)
		require.Equal(t, "Kiswahili", items[2].LearningAreaName)
		require.Zero(t, items[2].PeriodsTotal)
		require.Zero(t, items[2].PeriodsPresent)
		require.Zero(t, items[2].PeriodsAbsent)
		require.Zero(t, items[2].PeriodsExcused)
		require.Zero(t, items[2].AttendancePercentage)
	})

	t.Run("rls_enforced", func(t *testing.T) {
		wrongTenantID := uuid.New().String()
		items, err := repo.ListLearningAreaBreakdowns(ctx, wrongTenantID, ids.SchoolID, ids.AcademicTermID)
		require.NoError(t, err)
		require.Empty(t, items)
	})
}

// insertAttendanceRecords helper to populate attendance_records for the current week.
func insertAttendanceRecords(
	t *testing.T,
	pool *pgxpool.Pool,
	ids testIDs,
	studentID string,
	slotID string,
	status string,
	count int,
) {
	t.Helper()
	ctx := context.Background()

	// Get the Monday of current week for the test date
	var testDate time.Time
	err := pool.QueryRow(ctx, `SELECT DATE_TRUNC('week', CURRENT_DATE)::date`).Scan(&testDate)
	require.NoError(t, err)

	// Use different days of the week to avoid ON CONFLICT DO UPDATE replacing records
	// Each record gets a different date within the current week (Mon-Sun)
	days := []int{0, 1, 2, 3, 4, 5, 6} // Mon-Sun
	for i := 0; i < count; i++ {
		dayOffset := days[i%len(days)]
		recordDate := testDate.AddDate(0, 0, dayOffset)
		recordDateStr := recordDate.Format("2006-01-02")

		_, err := pool.Exec(ctx, `
			INSERT INTO attendance_records (
				id, tenant_id, school_id, student_id, timetable_allocation_id,
				date, status, academic_term_id, marked_by
			) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
			ON CONFLICT (student_id, timetable_allocation_id, date) DO UPDATE SET
				status = EXCLUDED.status
			`, uuid.New().String(), ids.TenantID, ids.SchoolID, studentID, slotID,
			recordDateStr, status, ids.AcademicTermID, ids.UserID)
		require.NoError(t, err)
	}
}

// insertAttendanceRecordsWithStatuses inserts attendance records with specific statuses
// for a student, using different dates within the current week.
func insertAttendanceRecordsWithStatuses(
	t *testing.T,
	pool *pgxpool.Pool,
	ids testIDs,
	studentID string,
	slotID string,
	statuses []string,
) {
	t.Helper()
	ctx := context.Background()

	var testDate time.Time
	err := pool.QueryRow(ctx, `SELECT DATE_TRUNC('week', CURRENT_DATE)::date`).Scan(&testDate)
	require.NoError(t, err)

	days := []int{0, 1, 2, 3, 4, 5, 6}
	for i, status := range statuses {
		dayOffset := days[i%len(days)]
		recordDate := testDate.AddDate(0, 0, dayOffset)
		recordDateStr := recordDate.Format("2006-01-02")

		_, err := pool.Exec(ctx, `
			INSERT INTO attendance_records (
				id, tenant_id, school_id, student_id, timetable_allocation_id,
				date, status, academic_term_id, marked_by
			) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
			ON CONFLICT (student_id, timetable_allocation_id, date) DO UPDATE SET
				status = EXCLUDED.status
			`, uuid.New().String(), ids.TenantID, ids.SchoolID, studentID, slotID,
			recordDateStr, status, ids.AcademicTermID, ids.UserID)
		require.NoError(t, err)
	}
}

func TestPgRepository_GetLowestAttendanceStudents(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	ctx := context.Background()
	pool, cleanup := startPG(t)
	defer cleanup()

	ids := setupTestTables(t, pool)
	repo := newRepo(pool)

	// Create a timetable structure and slot for the class
	structID := uuid.New().String()
	_, err := pool.Exec(ctx, `
		INSERT INTO timetable_blocks (id, tenant_id, school_id, academic_year_id, day_of_week, period_name, start_time, end_time, is_break) VALUES ($1, $2, $3, $4, 1, 'Lesson 1', '08:00', '09:00', false)
		`, structID, ids.TenantID, ids.SchoolID, ids.AcademicYearID)
	require.NoError(t, err)

	slotID := uuid.New().String()
	_, err = pool.Exec(ctx, `
		INSERT INTO timetable_allocations (id, tenant_id, school_id, academic_year_id, block_id, class_id, learning_area_id, teacher_id) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		`, slotID, ids.TenantID, ids.SchoolID, ids.AcademicYearID, structID, ids.ClassID1, ids.LearningAreaID1, ids.UserID)
	require.NoError(t, err)

	t.Run("returns_empty_when_no_students", func(t *testing.T) {
		// Create a new tenant/school with no students
		tenantID := uuid.New().String()
		schoolID := uuid.New().String()
		_, err := pool.Exec(ctx, `INSERT INTO tenants (id, name, slug, stytch_org_id) VALUES ($1, $2, $3, $4)`,
			tenantID, "Test2", "slug-"+tenantID[:8], "stytch-"+tenantID[:8])
		require.NoError(t, err)
		_, err = pool.Exec(ctx, `INSERT INTO cbc_schools (id, tenant_id, name, county, sub_county, school_type) VALUES ($1, $2, $3, 'Nairobi', 'Westlands', 'Public')`,
			schoolID, tenantID, "Test School 2")
		require.NoError(t, err)

		items, err := repo.GetLowestAttendanceStudents(ctx, tenantID, schoolID, 5)
		require.NoError(t, err)
		require.Empty(t, items)
	})

	t.Run("returns_empty_when_no_attendance_records", func(t *testing.T) {
		// Students exist but no attendance records for current week
		items, err := repo.GetLowestAttendanceStudents(ctx, ids.TenantID, ids.SchoolID, 5)
		require.NoError(t, err)
		require.Empty(t, items)
	})

	t.Run("returns_students_with_lowest_attendance_first", func(t *testing.T) {
		// Student1 (Alice): 3 PRESENT out of 7 = 42.86%
		insertAttendanceRecordsWithStatuses(t, pool, ids, ids.StudentID1, slotID, []string{
			"PRESENT", "PRESENT", "PRESENT",
			"ABSENT", "ABSENT", "ABSENT", "ABSENT",
		})

		// Student2 (Bob): 5 PRESENT out of 7 = 71.43%
		insertAttendanceRecordsWithStatuses(t, pool, ids, ids.StudentID2, slotID, []string{
			"PRESENT", "PRESENT", "PRESENT", "PRESENT", "PRESENT",
			"ABSENT", "ABSENT",
		})

		items, err := repo.GetLowestAttendanceStudents(ctx, ids.TenantID, ids.SchoolID, 5)
		require.NoError(t, err)
		require.Len(t, items, 2)

		// Alice (42.86%) should be first (lowest attendance)
		require.Equal(t, ids.StudentID1, items[0].StudentID)
		require.Equal(t, "Alice", items[0].FirstName)
		require.Equal(t, "Smith", items[0].LastName)
		require.Equal(t, 7, items[0].TotalPeriods)
		require.Equal(t, 3, items[0].PresentCount)
		require.InDelta(t, 42.86, items[0].AttendancePercentage, 0.01)

		// Bob (71.43%) should be second
		require.Equal(t, ids.StudentID2, items[1].StudentID)
		require.Equal(t, "Bob", items[1].FirstName)
		require.Equal(t, "Johnson", items[1].LastName)
		require.Equal(t, 7, items[1].TotalPeriods)
		require.Equal(t, 5, items[1].PresentCount)
		require.InDelta(t, 71.43, items[1].AttendancePercentage, 0.01)
	})

	t.Run("limits_results_to_specified_count", func(t *testing.T) {
		// Create 3 more students with varying attendance
		studentID3 := uuid.New().String()
		_, err = pool.Exec(ctx, `INSERT INTO cbc_students (id, tenant_id, school_id, full_name, gender, learning_pathway) VALUES ($1, $2, $3, 'Charlie Brown', 'M', 'Age_Based')`,
			studentID3, ids.TenantID, ids.SchoolID)
		require.NoError(t, err)
		_, err = pool.Exec(ctx, `INSERT INTO cbc_student_enrollments (id, tenant_id, school_id, student_id, academic_term_id, academic_year_id, class_id) VALUES ($1, $2, $3, $4, $5, $6, $7)`,
			uuid.New().String(), ids.TenantID, ids.SchoolID, studentID3, ids.AcademicTermID, ids.AcademicYearID, ids.ClassID1)
		require.NoError(t, err)

		studentID4 := uuid.New().String()
		_, err = pool.Exec(ctx, `INSERT INTO cbc_students (id, tenant_id, school_id, full_name, gender, learning_pathway) VALUES ($1, $2, $3, 'Diana Prince', 'F', 'Age_Based')`,
			studentID4, ids.TenantID, ids.SchoolID)
		require.NoError(t, err)
		_, err = pool.Exec(ctx, `INSERT INTO cbc_student_enrollments (id, tenant_id, school_id, student_id, academic_term_id, academic_year_id, class_id) VALUES ($1, $2, $3, $4, $5, $6, $7)`,
			uuid.New().String(), ids.TenantID, ids.SchoolID, studentID4, ids.AcademicTermID, ids.AcademicYearID, ids.ClassID1)
		require.NoError(t, err)

		studentID5 := uuid.New().String()
		_, err = pool.Exec(ctx, `INSERT INTO cbc_students (id, tenant_id, school_id, full_name, gender, learning_pathway) VALUES ($1, $2, $3, 'Eve Adams', 'F', 'Age_Based')`,
			studentID5, ids.TenantID, ids.SchoolID)
		require.NoError(t, err)
		_, err = pool.Exec(ctx, `INSERT INTO cbc_student_enrollments (id, tenant_id, school_id, student_id, academic_term_id, academic_year_id, class_id) VALUES ($1, $2, $3, $4, $5, $6, $7)`,
			uuid.New().String(), ids.TenantID, ids.SchoolID, studentID5, ids.AcademicTermID, ids.AcademicYearID, ids.ClassID1)
		require.NoError(t, err)

		// Charlie: 1 PRESENT out of 7 = 14.29% (lowest)
		insertAttendanceRecordsWithStatuses(t, pool, ids, studentID3, slotID, []string{
			"PRESENT",
			"ABSENT", "ABSENT", "ABSENT", "ABSENT", "ABSENT", "ABSENT",
		})

		// Diana: 2 PRESENT out of 7 = 28.57%
		insertAttendanceRecordsWithStatuses(t, pool, ids, studentID4, slotID, []string{
			"PRESENT", "PRESENT",
			"ABSENT", "ABSENT", "ABSENT", "ABSENT", "ABSENT",
		})

		// Eve: 3 PRESENT out of 7 = 42.86%
		insertAttendanceRecordsWithStatuses(t, pool, ids, studentID5, slotID, []string{
			"PRESENT", "PRESENT", "PRESENT",
			"ABSENT", "ABSENT", "ABSENT", "ABSENT",
		})

		// Alice and Bob already have records from previous subtest:
		// Alice: 3 PRESENT out of 7 = 42.86%
		// Bob: 5 PRESENT out of 7 = 71.43%

		// Limit to 3
		items, err := repo.GetLowestAttendanceStudents(ctx, ids.TenantID, ids.SchoolID, 3)
		require.NoError(t, err)
		require.Len(t, items, 3)

		// Should return the 3 lowest: Charlie (14.29%), Diana (28.57%), Alice/Eve (42.86%)
		require.Equal(t, studentID3, items[0].StudentID)
		require.InDelta(t, 14.29, items[0].AttendancePercentage, 0.01)
		require.Equal(t, studentID4, items[1].StudentID)
		require.InDelta(t, 28.57, items[1].AttendancePercentage, 0.01)
		// Third could be Alice or Eve (both 42.86%), accept either
		thirdPercentage := items[2].AttendancePercentage
		require.True(t, thirdPercentage >= 42.85 && thirdPercentage <= 42.87, "Expected ~42.86, got %f", thirdPercentage)
	})

	t.Run("rls_enforced", func(t *testing.T) {
		wrongTenantID := uuid.New().String()
		items, err := repo.GetLowestAttendanceStudents(ctx, wrongTenantID, ids.SchoolID, 5)
		require.NoError(t, err)
		require.Empty(t, items)
	})
}

// ============================================================================
// ListAttendanceSummary — repo integration (real DB via testcontainers)
// ============================================================================
func TestListAttendanceSummary_RealDB(t *testing.T) {
	pool, cleanup := startPG(t)
	defer cleanup()

	ids := setupTestTables(t, pool)
	repo := newRepo(pool)
	ctx := context.Background()

	// Insert class term attendance summaries for two classes
	_, err := pool.Exec(ctx, `
		INSERT INTO class_term_attendance_summaries
		(tenant_id, school_id, class_id, academic_term_id, academic_year_id, days_in_term, total_enrolled_avg, present_count, absent_count, late_count, excused_count, term_attendance_rate)
		VALUES ($1, $2, $3, $4, $5, 60, 150, 92, 4, 2, 2, 92.0)`,
		ids.TenantID, ids.SchoolID, ids.ClassID1, ids.AcademicTermID, ids.AcademicYearID)
	require.NoError(t, err)

	_, err = pool.Exec(ctx, `
		INSERT INTO class_term_attendance_summaries
		(tenant_id, school_id, class_id, academic_term_id, academic_year_id, days_in_term, total_enrolled_avg, present_count, absent_count, late_count, excused_count, term_attendance_rate)
		VALUES ($1, $2, $3, $4, $5, 60, 50, 45, 3, 1, 1, 90.0)`,
		ids.TenantID, ids.SchoolID, ids.ClassID2, ids.AcademicTermID, ids.AcademicYearID)
	require.NoError(t, err)

	rows, err := repo.ListAttendanceSummary(ctx, ids.TenantID, ids.SchoolID, "2026")
	require.NoError(t, err)
	require.NotEmpty(t, rows)

	// Should include per-class rows and an "All" aggregate
	classNames := make([]string, len(rows))
	for i, r := range rows {
		classNames[i] = r.ClassName
	}
	require.Contains(t, classNames, "All")
	require.True(t, len(rows) >= 2, "expected at least class + All aggregate")
}

func TestPgRepository_GetMarkedTimetableAllocation(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	ctx := context.Background()
	pool, cleanup := startPG(t)
	defer cleanup()

	applyAllMigrations(t, pool)
	tenantID, schoolID, userID := seedTenantSchoolUser(t, pool)

	academicYearID := uuid.New().String()
	_, err := pool.Exec(ctx, `INSERT INTO academic_years (id, tenant_id, school_id, name, start_date, end_date, is_current, created_by, updated_by) VALUES ($1, $2, $3, '2026', '2026-01-01', '2026-12-31', true, $4, $4)`,
		academicYearID, tenantID, schoolID, userID)
	require.NoError(t, err)

	academicTermID := uuid.New().String()
	_, err = pool.Exec(ctx, `INSERT INTO academic_terms (id, tenant_id, school_id, academic_year_id, name, term_number, start_date, end_date, is_current, is_final, created_by, updated_by) VALUES ($1, $2, $3, $4, 'Term 1', 1, '2026-01-01', '2026-03-31', true, false, $5, $5)`,
		academicTermID, tenantID, schoolID, academicYearID, userID)
	require.NoError(t, err)

	classID := uuid.New().String()
	laID := uuid.New().String()
	streamID := uuid.New().String()
	_, err = pool.Exec(ctx, `INSERT INTO cbc_streams (id, tenant_id, school_id, name) VALUES ($1,$2,$3,'Blue')`, streamID, tenantID, schoolID)
	require.NoError(t, err)
	_, err = pool.Exec(ctx, `INSERT INTO cbc_classes (id, tenant_id, school_id, academic_year_id, grade_level, stream_id, is_active) VALUES ($1,$2,$3,$4,'G1',$5,true)`, classID, tenantID, schoolID, academicYearID, streamID)
	require.NoError(t, err)
	_, err = pool.Exec(ctx, `INSERT INTO cbc_learning_areas (id, tenant_id, school_id, name, code, education_level, grade_level) VALUES ($1,$2,$3,'Math','MATH','Early_Years','G1')`, laID, tenantID, schoolID)
	require.NoError(t, err)

	trackID := uuid.New().String()
	_, err = pool.Exec(ctx, `INSERT INTO timetable_tracks (id, tenant_id, school_id, academic_year_id, academic_term_id, name, is_default) VALUES ($1,$2,$3,$4,$5,'T',true)`, trackID, tenantID, schoolID, academicYearID, academicTermID)
	require.NoError(t, err)

	blockID := uuid.New().String()
	_, err = pool.Exec(ctx, `INSERT INTO timetable_blocks (id, tenant_id, school_id, track_id, day_of_week, period_name, start_time, end_time, is_break, order_index) VALUES ($1,$2,$3,$4,1,'P1','08:00','09:00',false,1)`, blockID, tenantID, schoolID, trackID)
	require.NoError(t, err)

	allocationID := uuid.New().String()
	_, err = pool.Exec(ctx, `INSERT INTO timetable_allocations (id, tenant_id, school_id, block_id, class_id, learning_area_id, teacher_id, room_identifier) VALUES ($1,$2,$3,$4,$5,$6,$7,'R1')`, allocationID, tenantID, schoolID, blockID, classID, laID, userID)
	require.NoError(t, err)

	repo := NewRepository(&database.Pools{PG: pool})
	resp, err := repo.GetMarkedTimetableAllocation(ctx, tenantID, schoolID, allocationID, academicTermID, "2026-02-10")
	require.NoError(t, err)
	require.NotNil(t, resp)
	require.Equal(t, classID, resp.ClassID)
	require.Empty(t, resp.Students)
	require.Nil(t, resp.SessionID)
}
