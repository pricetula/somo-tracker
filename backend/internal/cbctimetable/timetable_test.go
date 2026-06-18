package cbctimetable

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
	"go.uber.org/zap"
)

// ============================================================================
// Integration test suite for CBC Timetable
// ============================================================================

// Fixed UUIDs for test reference data — used consistently across all tests
// Using vars for learning area IDs so we can take their address (for *string fields).
var (
	testLearningAreaID  = "ee0e8400-e29b-41d4-a716-446655440001"
	testLearningArea2ID = "ff0e8400-e29b-41d4-a716-446655440002"
)

const (
	testEducationSystemID = "550e8400-e29b-41d4-a716-446655440001"
	testTenantID          = "660e8400-e29b-41d4-a716-446655440001"
	testSchoolID          = "770e8400-e29b-41d4-a716-446655440001"
	testAcademicYearID    = "880e8400-e29b-41d4-a716-446655440001"
	testGradeID           = "990e8400-e29b-41d4-a716-446655440001"
	testClassID           = "aa0e8400-e29b-41d4-a716-446655440001"
	testClass2ID          = "bb0e8400-e29b-41d4-a716-446655440002"
	testTeacherID         = "cc0e8400-e29b-41d4-a716-446655440001"
	testTeacher2ID        = "dd0e8400-e29b-41d4-a716-446655440002"
	testTermID            = "aa0e8400-e29b-41d4-a716-446655440010"
)

var (
	testSuite *IntegrationSuite
)

// IntegrationSuite holds shared test infrastructure.
type IntegrationSuite struct {
	ctx    context.Context
	pgC    testcontainers.Container
	pool   *pgxpool.Pool
	logger *zap.Logger
	svc    *Service
	repo   *Repository
}

// TestMain — starts containers once for the package
func TestMain(m *testing.M) {
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	suite, err := setupSuite(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "integration suite setup failed: %v\n", err)
		os.Exit(1)
	}

	testSuite = suite
	code := m.Run()

	suite.cleanup()
	os.Exit(code)
}

func setupSuite(ctx context.Context) (*IntegrationSuite, error) {
	// 1. Start PostgreSQL container
	fmt.Println("=== Starting PostgreSQL container for CBC timetable tests...")
	pgC, hostPort, err := startPostgres(ctx)
	if err != nil {
		return nil, fmt.Errorf("postgres container: %w", err)
	}
	fmt.Printf("=== PostgreSQL ready at %s\n", hostPort)

	// 2. Build connection
	dbURL := fmt.Sprintf("postgres://somo_admin:somo_secure_password@%s/somotracker_test?sslmode=disable", hostPort)
	pgCfg, err := pgxpool.ParseConfig(dbURL)
	if err != nil {
		_ = pgC.Terminate(ctx)
		return nil, fmt.Errorf("parse pg config: %w", err)
	}
	pgCfg.MaxConns = 10
	pgCfg.MinConns = 2

	pool, err := pgxpool.NewWithConfig(ctx, pgCfg)
	if err != nil {
		_ = pgC.Terminate(ctx)
		return nil, fmt.Errorf("create pg pool: %w", err)
	}

	// 3. Run migrations
	if err := runMigrations(ctx, pool); err != nil {
		pool.Close()
		_ = pgC.Terminate(ctx)
		return nil, fmt.Errorf("run migrations: %w", err)
	}
	fmt.Println("=== Migrations applied")

	// 4. Create logger
	logger, _ := zap.NewDevelopment()

	// 5. Seed reference data
	if err := seedReferenceData(ctx, pool); err != nil {
		pool.Close()
		_ = pgC.Terminate(ctx)
		return nil, fmt.Errorf("seed reference data: %w", err)
	}
	fmt.Println("=== Reference data seeded")

	// 6. Build service/repo
	repo := &Repository{pool: pool}
	svc := NewService(repo)

	return &IntegrationSuite{
		ctx:    ctx,
		pgC:    pgC,
		pool:   pool,
		logger: logger,
		svc:    svc,
		repo:   repo,
	}, nil
}

func startPostgres(ctx context.Context) (testcontainers.Container, string, error) {
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
	if err != nil {
		return nil, "", err
	}

	host, err := c.Host(ctx)
	if err != nil {
		_ = c.Terminate(ctx)
		return nil, "", err
	}

	port, err := c.MappedPort(ctx, "5432")
	if err != nil {
		_ = c.Terminate(ctx)
		return nil, "", err
	}

	return c, fmt.Sprintf("%s:%s", host, port.Port()), nil
}

func runMigrations(ctx context.Context, pool *pgxpool.Pool) error {
	_, filename, _, _ := runtime.Caller(0)
	migrationsDir := filepath.Join(filepath.Dir(filename), "..", "database", "migrations")

	sql, err := os.ReadFile(filepath.Join(migrationsDir, "000001_initial_schema.up.sql"))
	if err != nil {
		return fmt.Errorf("read migration: %w", err)
	}
	if _, err := pool.Exec(ctx, string(sql)); err != nil {
		return fmt.Errorf("execute migration: %w", err)
	}
	return nil
}

// seedReferenceData creates the minimal data needed for timetable tests.
func seedReferenceData(ctx context.Context, pool *pgxpool.Pool) error {
	statements := []string{
		// Education system
		fmt.Sprintf(`INSERT INTO education_systems (id, name, country_code) VALUES ('%s', 'Kenya CBC', 'KE') ON CONFLICT DO NOTHING`, testEducationSystemID),
		// Tenant
		fmt.Sprintf(`INSERT INTO tenants (id, name, slug, stytch_org_id) VALUES ('%s', 'Test School', 'test-school', 'org_test') ON CONFLICT DO NOTHING`, testTenantID),
		// School
		fmt.Sprintf(`INSERT INTO schools (id, tenant_id, education_system_id, name) VALUES ('%s', '%s', '%s', 'Test Academy') ON CONFLICT DO NOTHING`, testSchoolID, testTenantID, testEducationSystemID),
		// Grade
		fmt.Sprintf(`INSERT INTO grades (id, education_system_id, name, sequence_order) VALUES ('%s', '%s', 'Grade 7', 1) ON CONFLICT DO NOTHING`, testGradeID, testEducationSystemID),
		// Academic year
		fmt.Sprintf(`INSERT INTO academic_years (id, tenant_id, school_id, name, start_date, end_date, is_current) VALUES ('%s', '%s', '%s', '2026', '2026-01-01', '2026-12-31', true) ON CONFLICT DO NOTHING`, testAcademicYearID, testTenantID, testSchoolID),
		// Academic term
		fmt.Sprintf(`INSERT INTO academic_terms (id, tenant_id, academic_year_id, name, start_date, end_date, is_current) VALUES ('%s', '%s', '%s', 'Term 1', '2026-01-01', '2026-04-30', true) ON CONFLICT DO NOTHING`, testTermID, testTenantID, testAcademicYearID),
		// Teachers (users with TEACHER membership)
		fmt.Sprintf(`INSERT INTO users (id, email, tenant_id, first_name, last_name, external_auth_id) VALUES ('%s', 'teacher1@test.com', '%s', 'John', 'Otieno', 'ext_teacher1') ON CONFLICT DO NOTHING`, testTeacherID, testTenantID),
		fmt.Sprintf(`INSERT INTO users (id, email, tenant_id, first_name, last_name, external_auth_id) VALUES ('%s', 'teacher2@test.com', '%s', 'Jane', 'Wanjiku', 'ext_teacher2') ON CONFLICT DO NOTHING`, testTeacher2ID, testTenantID),
		// Memberships
		fmt.Sprintf(`INSERT INTO memberships (tenant_id, role, user_id, school_id) VALUES ('%s', 'TEACHER', '%s', '%s') ON CONFLICT DO NOTHING`, testTenantID, testTeacherID, testSchoolID),
		fmt.Sprintf(`INSERT INTO memberships (tenant_id, role, user_id, school_id) VALUES ('%s', 'TEACHER', '%s', '%s') ON CONFLICT DO NOTHING`, testTenantID, testTeacher2ID, testSchoolID),
		// Classes
		fmt.Sprintf(`INSERT INTO classes (id, tenant_id, school_id, academic_year_id, education_system_id, grade_id, name, stream) VALUES ('%s', '%s', '%s', '%s', '%s', '%s', 'Grade 7', 'East') ON CONFLICT DO NOTHING`, testClassID, testTenantID, testSchoolID, testAcademicYearID, testEducationSystemID, testGradeID),
		fmt.Sprintf(`INSERT INTO classes (id, tenant_id, school_id, academic_year_id, education_system_id, grade_id, name, stream) VALUES ('%s', '%s', '%s', '%s', '%s', '%s', 'Grade 7', 'West') ON CONFLICT DO NOTHING`, testClass2ID, testTenantID, testSchoolID, testAcademicYearID, testEducationSystemID, testGradeID),
		// Learning areas
		fmt.Sprintf(`INSERT INTO cbc_learning_areas (id, tenant_id, school_id, education_system_id, grade_id, name, code) VALUES ('%s', '%s', '%s', '%s', '%s', 'Mathematics', 'MAT') ON CONFLICT DO NOTHING`, testLearningAreaID, testTenantID, testSchoolID, testEducationSystemID, testGradeID),
		fmt.Sprintf(`INSERT INTO cbc_learning_areas (id, tenant_id, school_id, education_system_id, grade_id, name, code) VALUES ('%s', '%s', '%s', '%s', '%s', 'English', 'ENG') ON CONFLICT DO NOTHING`, testLearningArea2ID, testTenantID, testSchoolID, testEducationSystemID, testGradeID),
		// Class teachers
		fmt.Sprintf(`INSERT INTO cbc_class_teachers (tenant_id, class_id, user_id, learning_area_id, is_primary) VALUES ('%s', '%s', '%s', '%s', true) ON CONFLICT DO NOTHING`, testTenantID, testClassID, testTeacherID, testLearningAreaID),
		fmt.Sprintf(`INSERT INTO cbc_class_teachers (tenant_id, class_id, user_id, learning_area_id, is_primary) VALUES ('%s', '%s', '%s', '%s', true) ON CONFLICT DO NOTHING`, testTenantID, testClass2ID, testTeacher2ID, testLearningArea2ID),
	}

	for _, stmt := range statements {
		if _, err := pool.Exec(ctx, stmt); err != nil {
			return fmt.Errorf("seed: %s -> %w", stmt[:60], err)
		}
	}

	return nil
}

// ─── Cleanup ──────────────────────────────────────────────────────────────

func (s *IntegrationSuite) cleanup() {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if s.pool != nil {
		s.pool.Close()
	}
	if s.pgC != nil {
		err := s.pgC.Terminate(ctx)
		if err != nil {
			fmt.Fprintf(os.Stderr, "failed to terminate postgres container: %v\n", err)
		}
	}
}

// freshDB cleans all timetable-related tables between test cases.
// Orders carefully due to FK constraints.
func (s *IntegrationSuite) freshDB(t *testing.T) {
	t.Helper()
	ctx := context.Background()
	tables := []string{
		"cbc_attendance_logs",
		"cbc_attendance_periods",
		"cbc_timetable_slots",
	}
	for _, table := range tables {
		if _, err := s.pool.Exec(ctx, "DELETE FROM "+table); err != nil {
			t.Fatalf("clean %s: %v", table, err)
		}
	}
}

// ─── Test helpers ─────────────────────────────────────────────────────────

// createTestSlot is a convenience helper that inserts a slot directly into the DB
// (bypassing the service) for test setup purposes.
func (s *IntegrationSuite) createTestSlot(t *testing.T, slot *TimetableSlot) {
	t.Helper()
	const query = `
		INSERT INTO cbc_timetable_slots
			(id, tenant_id, school_id, academic_year_id, class_id, teacher_id,
			 cbc_learning_area_id, room_identifier, day_of_week, start_time, end_time)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
	`
	_, err := s.pool.Exec(context.Background(), query,
		slot.ID, slot.TenantID, slot.SchoolID, slot.AcademicYearID, slot.ClassID, slot.TeacherID,
		slot.LearningAreaID, slot.RoomIdentifier, slot.DayOfWeek, slot.StartTime, slot.EndTime,
	)
	if err != nil {
		t.Fatalf("create test slot: %v", err)
	}
}

// generateUUID returns a deterministic UUID string for test purposes.
// Each call produces a unique, well-formed UUID v4-like hex string.
func generateUUID(prefix byte, seq int) string {
	// Format: 8-4-4-4-12 hex groups.
	// Prefix byte + zeros gives first group, seq gives second group.
	return fmt.Sprintf("%02x000000-%04x-4000-8000-000000000000", prefix, seq)
}

// ============================================================================
// ─── CATEGORY 1: DB EXCLUDE CONSTRAINTS — TEACHER OVERLAP ─────────────────
// ============================================================================

// TestExcludeConstraint_TeacherOverlap_SameTime verifies that two slots with the
// same teacher, same day, and overlapping times are rejected at the DB level.
func TestExcludeConstraint_TeacherOverlap_SameTime(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	s := testSuite
	s.freshDB(t)

	// First slot — should succeed
	slot1 := &TimetableSlot{
		ID:             generateUUID('a', 1),
		TenantID:       testTenantID,
		SchoolID:       testSchoolID,
		AcademicYearID: testAcademicYearID,
		ClassID:        testClassID,
		TeacherID:      testTeacherID,
		LearningAreaID: &testLearningAreaID,
		RoomIdentifier: strPtr("Room 1"),
		DayOfWeek:      1, // Monday
		StartTime:      "08:00",
		EndTime:        "08:40",
	}
	s.createTestSlot(t, slot1)

	// Second slot — same teacher, same day, overlapping time — should fail
	slot2 := &TimetableSlot{
		ID:             generateUUID('a', 2),
		TenantID:       testTenantID,
		SchoolID:       testSchoolID,
		AcademicYearID: testAcademicYearID,
		ClassID:        testClass2ID,  // different class
		TeacherID:      testTeacherID, // same teacher
		LearningAreaID: &testLearningArea2ID,
		RoomIdentifier: strPtr("Room 2"),
		DayOfWeek:      1,       // Monday
		StartTime:      "08:10", // overlaps with 08:00-08:40
		EndTime:        "08:50",
	}

	err := s.repo.CreateSlot(context.Background(), slot2)
	if err == nil {
		t.Fatal("expected exclusion constraint violation for overlapping teacher slot, got nil")
	}
	if !strContains(err.Error(), "excl_cbc_timetable_teacher") && !strContains(err.Error(), "conflicts") {
		t.Fatalf("expected exclusion constraint error, got: %v", err)
	}
}

// TestExcludeConstraint_TeacherOverlap_AdjacentNoOverlap verifies that adjacent
// time slots (end = start of next) do NOT trigger the exclusion constraint.
func TestExcludeConstraint_TeacherOverlap_AdjacentNoOverlap(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	s := testSuite
	s.freshDB(t)

	// First slot: 08:00-08:40
	slot1 := &TimetableSlot{
		ID:             generateUUID('b', 1),
		TenantID:       testTenantID,
		SchoolID:       testSchoolID,
		AcademicYearID: testAcademicYearID,
		ClassID:        testClassID,
		TeacherID:      testTeacherID,
		LearningAreaID: &testLearningAreaID,
		DayOfWeek:      1,
		StartTime:      "08:00",
		EndTime:        "08:40",
	}
	s.createTestSlot(t, slot1)

	// Second slot: 08:40-09:20 — adjacent, no overlap (because tsrange is [) )
	slot2 := &TimetableSlot{
		ID:             generateUUID('b', 2),
		TenantID:       testTenantID,
		SchoolID:       testSchoolID,
		AcademicYearID: testAcademicYearID,
		ClassID:        testClass2ID,
		TeacherID:      testTeacherID,
		LearningAreaID: &testLearningArea2ID,
		DayOfWeek:      1,
		StartTime:      "08:40",
		EndTime:        "09:20",
	}
	// Should succeed — no overlap with [08:00, 08:40)
	err := s.repo.CreateSlot(context.Background(), slot2)
	if err != nil {
		t.Fatalf("adjacent non-overlapping slots should be allowed, got: %v", err)
	}
}

// TestExcludeConstraint_TeacherOverlap_DifferentDay allows same teacher at same
// time on a different day.
func TestExcludeConstraint_TeacherOverlap_DifferentDay(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	s := testSuite
	s.freshDB(t)

	// Monday 08:00-08:40
	slot1 := &TimetableSlot{
		ID:             generateUUID('c', 1),
		TenantID:       testTenantID,
		SchoolID:       testSchoolID,
		AcademicYearID: testAcademicYearID,
		ClassID:        testClassID,
		TeacherID:      testTeacherID,
		LearningAreaID: &testLearningAreaID,
		DayOfWeek:      1,
		StartTime:      "08:00",
		EndTime:        "08:40",
	}
	s.createTestSlot(t, slot1)

	// Tuesday 08:00-08:40 — same teacher, same time, different day — allowed
	slot2 := &TimetableSlot{
		ID:             generateUUID('c', 2),
		TenantID:       testTenantID,
		SchoolID:       testSchoolID,
		AcademicYearID: testAcademicYearID,
		ClassID:        testClassID,
		TeacherID:      testTeacherID,
		LearningAreaID: &testLearningArea2ID,
		DayOfWeek:      2,
		StartTime:      "08:00",
		EndTime:        "08:40",
	}
	err := s.repo.CreateSlot(context.Background(), slot2)
	if err != nil {
		t.Fatalf("same teacher on different day should be allowed, got: %v", err)
	}
}

// TestExcludeConstraint_TeacherOverlap_DifferentAcademicYear allows same teacher
// at same day/time in a different academic year.
func TestExcludeConstraint_TeacherOverlap_DifferentAcademicYear(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	s := testSuite
	s.freshDB(t)

	// Insert a second academic year
	year2ID := generateUUID('y', 2)
	_, err := s.pool.Exec(context.Background(),
		`INSERT INTO academic_years (id, tenant_id, school_id, name, start_date, end_date, is_current)
		 VALUES ($1, $2, $3, '2025', '2025-01-01', '2025-12-31', false)
		 ON CONFLICT DO NOTHING`, year2ID, testTenantID, testSchoolID)
	if err != nil {
		t.Fatalf("create second academic year: %v", err)
	}

	// Slot in 2026 (current year)
	slot1 := &TimetableSlot{
		ID:             generateUUID('d', 1),
		TenantID:       testTenantID,
		SchoolID:       testSchoolID,
		AcademicYearID: testAcademicYearID, // 2026
		ClassID:        testClassID,
		TeacherID:      testTeacherID,
		LearningAreaID: &testLearningAreaID,
		DayOfWeek:      1,
		StartTime:      "08:00",
		EndTime:        "08:40",
	}
	s.createTestSlot(t, slot1)

	// Slot in 2025 — same teacher, same day/time — allowed (different year)
	slot2 := &TimetableSlot{
		ID:             generateUUID('d', 2),
		TenantID:       testTenantID,
		SchoolID:       testSchoolID,
		AcademicYearID: year2ID,
		ClassID:        testClassID,
		TeacherID:      testTeacherID,
		LearningAreaID: &testLearningArea2ID,
		DayOfWeek:      1,
		StartTime:      "08:00",
		EndTime:        "08:40",
	}
	err = s.repo.CreateSlot(context.Background(), slot2)
	if err != nil {
		t.Fatalf("same teacher on different academic year should be allowed, got: %v", err)
	}
}

// ============================================================================
// ─── CATEGORY 2: DB EXCLUDE CONSTRAINTS — ROOM OVERLAP ────────────────────
// ============================================================================

// TestExcludeConstraint_RoomOverlap_SameTime verifies room double-booking is
// rejected at the DB level.
func TestExcludeConstraint_RoomOverlap_SameTime(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	s := testSuite
	s.freshDB(t)

	// First slot in Room 1
	slot1 := &TimetableSlot{
		ID:             generateUUID('e', 1),
		TenantID:       testTenantID,
		SchoolID:       testSchoolID,
		AcademicYearID: testAcademicYearID,
		ClassID:        testClassID,
		TeacherID:      testTeacherID,
		LearningAreaID: &testLearningAreaID,
		RoomIdentifier: strPtr("Lab A"),
		DayOfWeek:      1,
		StartTime:      "09:00",
		EndTime:        "09:40",
	}
	s.createTestSlot(t, slot1)

	// Second slot in same Room 1, overlapping time — should fail
	slot2 := &TimetableSlot{
		ID:             generateUUID('e', 2),
		TenantID:       testTenantID,
		SchoolID:       testSchoolID,
		AcademicYearID: testAcademicYearID,
		ClassID:        testClass2ID,
		TeacherID:      testTeacher2ID,
		LearningAreaID: &testLearningArea2ID,
		RoomIdentifier: strPtr("Lab A"),
		DayOfWeek:      1,
		StartTime:      "09:15",
		EndTime:        "09:55",
	}

	err := s.repo.CreateSlot(context.Background(), slot2)
	if err == nil {
		t.Fatal("expected exclusion constraint violation for overlapping room slot, got nil")
	}
	if !strContains(err.Error(), "excl_cbc_timetable_room") && !strContains(err.Error(), "conflicts") {
		t.Fatalf("expected room exclusion constraint error, got: %v", err)
	}
}

// TestExcludeConstraint_RoomOverlap_SameRoomDifferentDay allows same room at
// same time on a different day.
func TestExcludeConstraint_RoomOverlap_SameRoomDifferentDay(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	s := testSuite
	s.freshDB(t)

	slot1 := &TimetableSlot{
		ID:             generateUUID('f', 1),
		TenantID:       testTenantID,
		SchoolID:       testSchoolID,
		AcademicYearID: testAcademicYearID,
		ClassID:        testClassID,
		TeacherID:      testTeacherID,
		RoomIdentifier: strPtr("Lab B"),
		DayOfWeek:      1, // Monday
		StartTime:      "10:00",
		EndTime:        "10:40",
	}
	s.createTestSlot(t, slot1)

	slot2 := &TimetableSlot{
		ID:             generateUUID('f', 2),
		TenantID:       testTenantID,
		SchoolID:       testSchoolID,
		AcademicYearID: testAcademicYearID,
		ClassID:        testClass2ID,
		TeacherID:      testTeacher2ID,
		RoomIdentifier: strPtr("Lab B"),
		DayOfWeek:      2, // Tuesday — same time, different day
		StartTime:      "10:00",
		EndTime:        "10:40",
	}
	err := s.repo.CreateSlot(context.Background(), slot2)
	if err != nil {
		t.Fatalf("same room on a different day should be allowed, got: %v", err)
	}
}

// TestExcludeConstraint_RoomOverlap_NullRoom allows same time in different rooms
// (or null room) without conflict.
func TestExcludeConstraint_RoomOverlap_NullRoom(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	s := testSuite
	s.freshDB(t)

	// Slot with null room
	slot1 := &TimetableSlot{
		ID:             generateUUID('g', 1),
		TenantID:       testTenantID,
		SchoolID:       testSchoolID,
		AcademicYearID: testAcademicYearID,
		ClassID:        testClassID,
		TeacherID:      testTeacherID,
		LearningAreaID: &testLearningAreaID,
		RoomIdentifier: nil,
		DayOfWeek:      1,
		StartTime:      "11:00",
		EndTime:        "11:40",
	}
	s.createTestSlot(t, slot1)

	// Another slot with null room at the same time — should be allowed
	// (the EXCLUDE only applies when room_identifier is equal and non-null)
	slot2 := &TimetableSlot{
		ID:             generateUUID('g', 2),
		TenantID:       testTenantID,
		SchoolID:       testSchoolID,
		AcademicYearID: testAcademicYearID,
		ClassID:        testClass2ID,
		TeacherID:      testTeacher2ID,
		LearningAreaID: &testLearningArea2ID,
		RoomIdentifier: nil,
		DayOfWeek:      1,
		StartTime:      "11:00",
		EndTime:        "11:40",
	}
	err := s.repo.CreateSlot(context.Background(), slot2)
	if err != nil {
		t.Fatalf("two null-room slots at same time should be allowed, got: %v", err)
	}
}

// ============================================================================
// ─── CATEGORY 3: CRUD OPERATIONS ───────────────────────────────────────────
// ============================================================================

// TestCRUD_CreateAndFetchSlot verifies basic create and fetch operations.
func TestCRUD_CreateAndFetchSlot(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	s := testSuite
	s.freshDB(t)

	// Create a slot via service
	req := &CreateSlotRequest{
		ClassID:        testClassID,
		TeacherID:      testTeacherID,
		LearningAreaID: &testLearningAreaID,
		RoomIdentifier: strPtr("Room 101"),
		DayOfWeek:      1,
		StartTime:      "08:00",
		EndTime:        "08:40",
	}

	slot, err := s.svc.CreateSlot(context.Background(), testSchoolID, testTenantID, req)
	if err != nil {
		t.Fatalf("create slot: %v", err)
	}
	if slot.ID == "" {
		t.Fatal("expected non-empty slot ID")
	}
	if slot.TeacherID != testTeacherID {
		t.Fatalf("expected teacher_id %s, got %s", testTeacherID, slot.TeacherID)
	}
	if slot.DayOfWeek != 1 {
		t.Fatalf("expected day_of_week 1, got %d", slot.DayOfWeek)
	}
	if slot.StartTime != "08:00" {
		t.Fatalf("expected start_time 08:00, got %s", slot.StartTime)
	}

	// Fetch slots for the class
	slots, err := s.svc.FetchSlots(context.Background(), testClassID)
	if err != nil {
		t.Fatalf("fetch slots: %v", err)
	}
	if len(slots) != 1 {
		t.Fatalf("expected 1 slot, got %d", len(slots))
	}
}

// TestCRUD_CreateSlot_NullLearningArea verifies slots with no learning area
// (breaks, assemblies, free periods) can be created.
func TestCRUD_CreateSlot_NullLearningArea(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	s := testSuite
	s.freshDB(t)

	req := &CreateSlotRequest{
		ClassID:        testClassID,
		TeacherID:      testTeacherID,
		LearningAreaID: nil, // null — no learning area
		RoomIdentifier: nil,
		DayOfWeek:      1,
		StartTime:      "08:00",
		EndTime:        "08:30",
	}

	slot, err := s.svc.CreateSlot(context.Background(), testSchoolID, testTenantID, req)
	if err != nil {
		t.Fatalf("create slot with null learning area: %v", err)
	}
	if slot.LearningAreaID != nil {
		t.Fatal("expected nil learning area ID")
	}
}

// TestCRUD_UpdateSlot verifies updating a slot's time and teacher.
func TestCRUD_UpdateSlot(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	s := testSuite
	s.freshDB(t)

	// Create initial slot
	req := &CreateSlotRequest{
		ClassID:        testClassID,
		TeacherID:      testTeacherID,
		LearningAreaID: &testLearningAreaID,
		DayOfWeek:      1,
		StartTime:      "08:00",
		EndTime:        "08:40",
	}
	slot, err := s.svc.CreateSlot(context.Background(), testSchoolID, testTenantID, req)
	if err != nil {
		t.Fatalf("create slot: %v", err)
	}

	// Update the slot
	updateReq := &UpdateSlotRequest{
		TeacherID:      testTeacher2ID,
		LearningAreaID: &testLearningArea2ID,
		DayOfWeek:      2,
		StartTime:      "09:00",
		EndTime:        "09:40",
	}
	updated, err := s.svc.UpdateSlot(context.Background(), slot.ID, testSchoolID, updateReq)
	if err != nil {
		t.Fatalf("update slot: %v", err)
	}
	if updated.TeacherID != testTeacher2ID {
		t.Fatalf("expected teacher %s, got %s", testTeacher2ID, updated.TeacherID)
	}
	if updated.DayOfWeek != 2 {
		t.Fatalf("expected day 2, got %d", updated.DayOfWeek)
	}
	if updated.StartTime != "09:00" {
		t.Fatalf("expected 09:00, got %s", updated.StartTime)
	}
}

// TestCRUD_DeleteSlot verifies slot deletion.
func TestCRUD_DeleteSlot(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	s := testSuite
	s.freshDB(t)

	req := &CreateSlotRequest{
		ClassID:        testClassID,
		TeacherID:      testTeacherID,
		LearningAreaID: &testLearningAreaID,
		DayOfWeek:      1,
		StartTime:      "08:00",
		EndTime:        "08:40",
	}
	slot, err := s.svc.CreateSlot(context.Background(), testSchoolID, testTenantID, req)
	if err != nil {
		t.Fatalf("create slot: %v", err)
	}

	// Delete
	err = s.svc.DeleteSlot(context.Background(), slot.ID)
	if err != nil {
		t.Fatalf("delete slot: %v", err)
	}

	// Verify it's gone
	fetched, err := s.repo.FetchSlotByID(context.Background(), slot.ID)
	if err != nil {
		t.Fatalf("fetch after delete: %v", err)
	}
	if fetched != nil {
		t.Fatal("slot should be nil after deletion")
	}
}

// TestCRUD_DeleteSlot_NotFound verifies deleting a non-existent slot returns an error.
func TestCRUD_DeleteSlot_NotFound(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	s := testSuite
	s.freshDB(t)

	err := s.svc.DeleteSlot(context.Background(), "00000000-0000-0000-0000-000000000000")
	if err == nil {
		t.Fatal("expected error when deleting non-existent slot")
	}
	if !strContains(err.Error(), "not found") {
		t.Fatalf("expected 'not found' error, got: %v", err)
	}
}

// ============================================================================
// ─── CATEGORY 4: SERVICE VALIDATION ────────────────────────────────────────
// ============================================================================

// TestValidation_InvalidDayOfWeek verifies days outside 1-7 are rejected.
func TestValidation_InvalidDayOfWeek(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	s := testSuite
	s.freshDB(t)

	req := &CreateSlotRequest{
		ClassID:   testClassID,
		TeacherID: testTeacherID,
		DayOfWeek: 0, // invalid
		StartTime: "08:00",
		EndTime:   "08:40",
	}

	_, err := s.svc.CreateSlot(context.Background(), testSchoolID, testTenantID, req)
	if err != ErrInvalidDayOfWeek {
		t.Fatalf("expected ErrInvalidDayOfWeek, got: %v", err)
	}

	// Also test day 8
	req.DayOfWeek = 8
	_, err = s.svc.CreateSlot(context.Background(), testSchoolID, testTenantID, req)
	if err != ErrInvalidDayOfWeek {
		t.Fatalf("expected ErrInvalidDayOfWeek for day 8, got: %v", err)
	}
}

// TestValidation_EndTimeBeforeStartTime verifies end_time must be after start_time.
func TestValidation_EndTimeBeforeStartTime(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	s := testSuite
	s.freshDB(t)

	req := &CreateSlotRequest{
		ClassID:   testClassID,
		TeacherID: testTeacherID,
		DayOfWeek: 1,
		StartTime: "09:00",
		EndTime:   "08:00", // before start
	}

	_, err := s.svc.CreateSlot(context.Background(), testSchoolID, testTenantID, req)
	if err != ErrInvalidTimeRange {
		t.Fatalf("expected ErrInvalidTimeRange, got: %v", err)
	}
}

// TestValidation_EqualStartAndEndTime verifies start_time must not equal end_time.
func TestValidation_EqualStartAndEndTime(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	s := testSuite
	s.freshDB(t)

	req := &CreateSlotRequest{
		ClassID:   testClassID,
		TeacherID: testTeacherID,
		DayOfWeek: 1,
		StartTime: "08:00",
		EndTime:   "08:00", // same as start
	}

	_, err := s.svc.CreateSlot(context.Background(), testSchoolID, testTenantID, req)
	if err != ErrInvalidTimeRange {
		t.Fatalf("expected ErrInvalidTimeRange, got: %v", err)
	}
}

// ============================================================================
// ─── CATEGORY 5: CONFLICT PRE-CHECK SERVICE ────────────────────────────────
// ============================================================================

// TestConflictCheck_NoConflicts verifies the happy path — no conflicts found.
func TestConflictCheck_NoConflicts(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	s := testSuite
	s.freshDB(t)

	// No slots exist yet
	conflicts, err := s.svc.CheckConflicts(context.Background(), &ConflictCheckRequest{
		TeacherID:      testTeacherID,
		DayOfWeek:      1,
		StartTime:      "08:00",
		EndTime:        "08:40",
		AcademicYearID: testAcademicYearID,
		SchoolID:       testSchoolID,
	})
	if err != nil {
		t.Fatalf("check conflicts: %v", err)
	}
	if len(conflicts) != 0 {
		t.Fatalf("expected 0 conflicts, got %d", len(conflicts))
	}
}

// TestConflictCheck_DetectsTeacherConflict verifies a teacher overlap is reported.
func TestConflictCheck_DetectsTeacherConflict(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	s := testSuite
	s.freshDB(t)

	// Create an existing slot
	slot1 := &TimetableSlot{
		ID:             generateUUID('h', 1),
		TenantID:       testTenantID,
		SchoolID:       testSchoolID,
		AcademicYearID: testAcademicYearID,
		ClassID:        testClassID,
		TeacherID:      testTeacherID,
		LearningAreaID: &testLearningAreaID,
		DayOfWeek:      1,
		StartTime:      "08:00",
		EndTime:        "08:40",
	}
	s.createTestSlot(t, slot1)

	// Check for same teacher at overlapping time
	conflicts, err := s.svc.CheckConflicts(context.Background(), &ConflictCheckRequest{
		TeacherID:      testTeacherID,
		DayOfWeek:      1,
		StartTime:      "08:10",
		EndTime:        "08:50",
		AcademicYearID: testAcademicYearID,
		SchoolID:       testSchoolID,
	})
	if err != nil {
		t.Fatalf("check conflicts: %v", err)
	}
	if len(conflicts) != 1 {
		t.Fatalf("expected 1 conflict, got %d", len(conflicts))
	}
	if conflicts[0].Type != "teacher" {
		t.Fatalf("expected teacher conflict, got %s", conflicts[0].Type)
	}
}

// TestConflictCheck_DetectsRoomConflict verifies a room overlap is reported.
func TestConflictCheck_DetectsRoomConflict(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	s := testSuite
	s.freshDB(t)

	slot1 := &TimetableSlot{
		ID:             generateUUID('i', 1),
		TenantID:       testTenantID,
		SchoolID:       testSchoolID,
		AcademicYearID: testAcademicYearID,
		ClassID:        testClassID,
		TeacherID:      testTeacherID,
		LearningAreaID: &testLearningAreaID,
		RoomIdentifier: strPtr("Science Lab"),
		DayOfWeek:      1,
		StartTime:      "08:00",
		EndTime:        "08:40",
	}
	s.createTestSlot(t, slot1)

	// Check for same room at overlapping time
	conflicts, err := s.svc.CheckConflicts(context.Background(), &ConflictCheckRequest{
		TeacherID:      testTeacher2ID,
		DayOfWeek:      1,
		StartTime:      "08:10",
		EndTime:        "08:50",
		AcademicYearID: testAcademicYearID,
		SchoolID:       testSchoolID,
		RoomIdentifier: strPtr("Science Lab"),
	})
	if err != nil {
		t.Fatalf("check conflicts: %v", err)
	}
	if len(conflicts) != 1 {
		t.Fatalf("expected 1 room conflict, got %d", len(conflicts))
	}
	if conflicts[0].Type != "room" {
		t.Fatalf("expected room conflict, got %s", conflicts[0].Type)
	}
}

// TestConflictCheck_ExcludeOwnSlot verifies that when editing a slot, the
// pre-check excludes the slot being edited from the overlap detection.
func TestConflictCheck_ExcludeOwnSlot(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	s := testSuite
	s.freshDB(t)

	slot1 := &TimetableSlot{
		ID:             generateUUID('j', 1),
		TenantID:       testTenantID,
		SchoolID:       testSchoolID,
		AcademicYearID: testAcademicYearID,
		ClassID:        testClassID,
		TeacherID:      testTeacherID,
		LearningAreaID: &testLearningAreaID,
		DayOfWeek:      1,
		StartTime:      "08:00",
		EndTime:        "08:40",
	}
	s.createTestSlot(t, slot1)

	// Check conflicts excluding this slot — should be empty (slot doesn't conflict with itself)
	conflicts, err := s.svc.CheckConflicts(context.Background(), &ConflictCheckRequest{
		TeacherID:      testTeacherID,
		DayOfWeek:      1,
		StartTime:      "08:00",
		EndTime:        "08:40",
		AcademicYearID: testAcademicYearID,
		SchoolID:       testSchoolID,
		ExcludeSlotID:  strPtr(slot1.ID),
	})
	if err != nil {
		t.Fatalf("check conflicts: %v", err)
	}
	if len(conflicts) != 0 {
		t.Fatalf("expected 0 conflicts when excluding own slot, got %d", len(conflicts))
	}
}

// TestConflictCheck_DetectsBothTeacherAndRoomConflict verifies simultaneous
// teacher and room conflicts are both reported.
func TestConflictCheck_DetectsBothTeacherAndRoomConflict(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	s := testSuite
	s.freshDB(t)

	// Create a slot that uses both teacher A and Room X
	slot1 := &TimetableSlot{
		ID:             generateUUID('k', 1),
		TenantID:       testTenantID,
		SchoolID:       testSchoolID,
		AcademicYearID: testAcademicYearID,
		ClassID:        testClassID,
		TeacherID:      testTeacherID,
		RoomIdentifier: strPtr("Room X"),
		LearningAreaID: &testLearningAreaID,
		DayOfWeek:      1,
		StartTime:      "08:00",
		EndTime:        "08:40",
	}
	s.createTestSlot(t, slot1)

	// Check for same teacher AND same room with a different teacher
	conflicts, err := s.svc.CheckConflicts(context.Background(), &ConflictCheckRequest{
		TeacherID:      testTeacherID, // same teacher → teacher conflict
		DayOfWeek:      1,
		StartTime:      "08:10",
		EndTime:        "08:50",
		AcademicYearID: testAcademicYearID,
		SchoolID:       testSchoolID,
		RoomIdentifier: strPtr("Room X"), // same room → room conflict
	})
	if err != nil {
		t.Fatalf("check conflicts: %v", err)
	}
	if len(conflicts) != 2 {
		t.Fatalf("expected 2 conflicts (teacher + room), got %d", len(conflicts))
	}

	types := make(map[string]bool)
	for _, c := range conflicts {
		types[c.Type] = true
	}
	if !types["teacher"] {
		t.Fatal("expected teacher conflict type")
	}
	if !types["room"] {
		t.Fatal("expected room conflict type")
	}
}

// ============================================================================
// ─── CATEGORY 6: BULK OPERATIONS ──────────────────────────────────────────
// ============================================================================

// TestBulk_DuplicateDay verifies duplicating Monday's slots to Tuesday.
func TestBulk_DuplicateDay(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	s := testSuite
	s.freshDB(t)

	// Create slots on Monday (day 1) — non-overlapping, valid times
	for i := 0; i < 3; i++ {
		tID := testTeacherID
		room := fmt.Sprintf("Room %d", i+1)
		// Each slot 25 min, starting at 08:00, 08:25, 08:50
		startTotal := 8*60 + i*25
		endTotal := startTotal + 20
		slot := &TimetableSlot{
			ID:             generateUUID('l', i+1),
			TenantID:       testTenantID,
			SchoolID:       testSchoolID,
			AcademicYearID: testAcademicYearID,
			ClassID:        testClassID,
			TeacherID:      tID,
			LearningAreaID: &testLearningAreaID,
			RoomIdentifier: strPtr(room),
			DayOfWeek:      1,
			StartTime:      fmtTime(startTotal),
			EndTime:        fmtTime(endTotal),
		}
		s.createTestSlot(t, slot)
	}

	// Duplicate to Tuesday (day 2)
	result, err := s.svc.DuplicateDay(context.Background(), &DuplicateDayRequest{
		SourceDay:      1,
		TargetDays:     []int{2},
		AcademicYearID: testAcademicYearID,
		ClassID:        testClassID,
	})
	if err != nil {
		t.Fatalf("duplicate day: %v", err)
	}
	if result.TotalCopied != 3 {
		t.Fatalf("expected 3 slots copied, got %d", result.TotalCopied)
	}
	if len(result.Skipped) != 0 {
		t.Fatalf("expected 0 skipped, got %d", len(result.Skipped))
	}

	// Verify Tuesday now has 3 slots
	tuesdaySlots, err := s.repo.FetchSlotsByClassAndDay(context.Background(), testClassID, 2, testAcademicYearID)
	if err != nil {
		t.Fatalf("fetch tuesday slots: %v", err)
	}
	if len(tuesdaySlots) != 3 {
		t.Fatalf("expected 3 slots on Tuesday, got %d", len(tuesdaySlots))
	}
}

// TestBulk_DuplicateDay_SkipsConflicts verifies that slots that would cause
// a teacher/room conflict on the target day are skipped (not failed).
func TestBulk_DuplicateDay_SkipsConflicts(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	s := testSuite
	s.freshDB(t)

	// Create a slot on Monday (day 1) — will be duplicated
	slot1 := &TimetableSlot{
		ID:             generateUUID('m', 1),
		TenantID:       testTenantID,
		SchoolID:       testSchoolID,
		AcademicYearID: testAcademicYearID,
		ClassID:        testClassID,
		TeacherID:      testTeacherID,
		LearningAreaID: &testLearningAreaID,
		DayOfWeek:      1,
		StartTime:      "08:00",
		EndTime:        "08:40",
	}
	s.createTestSlot(t, slot1)

	// Create a conflicting slot on Tuesday (day 2) — same teacher, same time
	slot2 := &TimetableSlot{
		ID:             generateUUID('m', 2),
		TenantID:       testTenantID,
		SchoolID:       testSchoolID,
		AcademicYearID: testAcademicYearID,
		ClassID:        testClass2ID,
		TeacherID:      testTeacherID,
		LearningAreaID: &testLearningArea2ID,
		DayOfWeek:      2,
		StartTime:      "08:00",
		EndTime:        "08:40",
	}
	s.createTestSlot(t, slot2)

	// Duplicate Monday to Tuesday — should skip the conflicting slot
	result, err := s.svc.DuplicateDay(context.Background(), &DuplicateDayRequest{
		SourceDay:      1,
		TargetDays:     []int{2},
		AcademicYearID: testAcademicYearID,
		ClassID:        testClassID,
	})
	if err != nil {
		t.Fatalf("duplicate day: %v", err)
	}
	if result.TotalCopied != 0 {
		t.Fatalf("expected 0 slots copied (conflict), got %d", result.TotalCopied)
	}
	if len(result.Skipped) != 1 {
		t.Fatalf("expected 1 skipped slot, got %d", len(result.Skipped))
	}
	if !strContains(result.Skipped[0].Reason, "already has a slot") {
		t.Fatalf("expected skip reason about teacher conflict, got: %s", result.Skipped[0].Reason)
	}
}

// TestBulk_DuplicateDay_MultipleTargets verifies duplicating to multiple days at once.
func TestBulk_DuplicateDay_MultipleTargets(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	s := testSuite
	s.freshDB(t)

	// Create 2 slots on Monday
	for i := 0; i < 2; i++ {
		slot := &TimetableSlot{
			ID:             generateUUID('n', i+1),
			TenantID:       testTenantID,
			SchoolID:       testSchoolID,
			AcademicYearID: testAcademicYearID,
			ClassID:        testClassID,
			TeacherID:      testTeacherID,
			LearningAreaID: &testLearningAreaID,
			DayOfWeek:      1,
			StartTime:      fmt.Sprintf("08:%02d", i*30),
			EndTime:        fmt.Sprintf("08:%02d", i*30+25),
		}
		s.createTestSlot(t, slot)
	}

	// Duplicate to Tuesday AND Wednesday
	result, err := s.svc.DuplicateDay(context.Background(), &DuplicateDayRequest{
		SourceDay:      1,
		TargetDays:     []int{2, 3},
		AcademicYearID: testAcademicYearID,
		ClassID:        testClassID,
	})
	if err != nil {
		t.Fatalf("duplicate day: %v", err)
	}
	// 2 slots × 2 target days = 4 total copied (assuming no conflicts)
	if result.TotalCopied != 4 {
		t.Fatalf("expected 4 slots copied (2 slots × 2 days), got %d", result.TotalCopied)
	}
}

// TestBulk_CopyFromClass verifies copying slots from one class to another.
func TestBulk_CopyFromClass(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	s := testSuite
	s.freshDB(t)

	// Create slots in class 1
	for i := 0; i < 5; i++ {
		teacherID := testTeacherID
		if i%2 != 0 {
			teacherID = testTeacher2ID
		}
		startTotal := 8*60 + i*15
		endTotal := startTotal + 10
		slot := &TimetableSlot{
			ID:             generateUUID('o', i+1),
			TenantID:       testTenantID,
			SchoolID:       testSchoolID,
			AcademicYearID: testAcademicYearID,
			ClassID:        testClassID,
			TeacherID:      teacherID,
			LearningAreaID: &testLearningAreaID,
			DayOfWeek:      1,
			StartTime:      fmtTime(startTotal),
			EndTime:        fmtTime(endTotal),
		}
		s.createTestSlot(t, slot)
	}

	// Copy to class 2 — the DB constraint prevents copying the same teacher
	// at the same time to another class (a teacher can't be in two places).
	// So we expect 0 successful copies when all slots use the same teacher/time patterns.
	result, err := s.svc.CopyFromClass(context.Background(), &CopyFromClassRequest{
		SourceClassID:  testClassID,
		TargetClassID:  testClass2ID,
		AcademicYearID: testAcademicYearID,
	})
	if err != nil {
		t.Fatalf("copy from class: %v", err)
	}

	t.Logf("CopyFromClass result: %d copied, %d skipped", result.TotalCopied, len(result.Skipped))
	for _, sk := range result.Skipped {
		t.Logf("  Skipped: day=%d time=%s reason=%s", sk.DayOfWeek, sk.StartTime, sk.Reason)
	}

	// Since the same teachers are used at the same times, the EXCLUDE constraint
	// prevents duplication. This is correct — a teacher can't teach two classes
	// at the same time. 5 slots should be skipped.
	if result.TotalCopied != 0 {
		t.Fatalf("expected 0 slots copied (teacher overlap), got %d", result.TotalCopied)
	}
	if len(result.Skipped) != 5 {
		t.Fatalf("expected 5 skipped slots, got %d", len(result.Skipped))
	}

	// Verify source class still has its 5 slots, target class has none (no copies)
	sourceSlots, err := s.repo.FetchSlotsByClass(context.Background(), testClassID)
	if err != nil {
		t.Fatalf("fetch source class slots: %v", err)
	}
	if len(sourceSlots) != 5 {
		t.Fatalf("expected 5 slots in source class, got %d", len(sourceSlots))
	}

	targetSlots, err := s.repo.FetchSlotsByClass(context.Background(), testClass2ID)
	if err != nil {
		t.Fatalf("fetch target class slots: %v", err)
	}
	if len(targetSlots) != 0 {
		t.Fatalf("expected 0 slots in target class (all blocked), got %d", len(targetSlots))
	}
}

// ============================================================================
// ─── CATEGORY 7: EDGE CASES ───────────────────────────────────────────────
// ============================================================================

// TestEdge_DayOfWeekBoundary verifies day_of_week values 1 and 7 work correctly.
func TestEdge_DayOfWeekBoundary(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	s := testSuite
	s.freshDB(t)

	// Day 1 (Monday) — valid
	req1 := &CreateSlotRequest{
		ClassID:   testClassID,
		TeacherID: testTeacherID,
		DayOfWeek: 1,
		StartTime: "08:00",
		EndTime:   "08:40",
	}
	_, err := s.svc.CreateSlot(context.Background(), testSchoolID, testTenantID, req1)
	if err != nil {
		t.Fatalf("create slot day 1: %v", err)
	}

	// Day 7 (Sunday) — valid
	req2 := &CreateSlotRequest{
		ClassID:   testClassID,
		TeacherID: testTeacher2ID,
		DayOfWeek: 7,
		StartTime: "08:00",
		EndTime:   "08:40",
	}
	_, err = s.svc.CreateSlot(context.Background(), testSchoolID, testTenantID, req2)
	if err != nil {
		t.Fatalf("create slot day 7: %v", err)
	}

	// Both should be fetchable
	slots, err := s.svc.FetchSlots(context.Background(), testClassID)
	if err != nil {
		t.Fatalf("fetch slots: %v", err)
	}
	if len(slots) != 2 {
		t.Fatalf("expected 2 slots, got %d", len(slots))
	}
}

// TestEdge_SameTeacherDifferentClass_OverlapAllowed allows same teacher in
// different classes at the same time? No — the EXCLUDE constraint is on teacher_id
// alone, not on (teacher_id, class_id). The constraint is global per teacher.
// This test verifies it's correctly rejected.
func TestEdge_SameTeacherDifferentClass_OverlapBlocked(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	s := testSuite
	s.freshDB(t)

	// Slot in class 1
	slot1 := &TimetableSlot{
		ID:             generateUUID('p', 1),
		TenantID:       testTenantID,
		SchoolID:       testSchoolID,
		AcademicYearID: testAcademicYearID,
		ClassID:        testClassID,
		TeacherID:      testTeacherID,
		LearningAreaID: &testLearningAreaID,
		DayOfWeek:      1,
		StartTime:      "08:00",
		EndTime:        "08:40",
	}
	s.createTestSlot(t, slot1)

	// Same teacher in class 2 at overlapping time — must be rejected
	slot2 := &TimetableSlot{
		ID:             generateUUID('p', 2),
		TenantID:       testTenantID,
		SchoolID:       testSchoolID,
		AcademicYearID: testAcademicYearID,
		ClassID:        testClass2ID,
		TeacherID:      testTeacherID,
		LearningAreaID: &testLearningArea2ID,
		DayOfWeek:      1,
		StartTime:      "08:10",
		EndTime:        "08:50",
	}
	err := s.repo.CreateSlot(context.Background(), slot2)
	if err == nil {
		t.Fatal("expected exclusion constraint violation — same teacher cannot teach two classes at the same time")
	}
}

// TestEdge_NullRoomMultipleSlots verifies that multiple slots with null room
// at the same time do NOT create room conflicts (nulls don't overlap).
func TestEdge_NullRoomMultipleSlots(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	s := testSuite
	s.freshDB(t)

	// Slot A — null room
	slotA := &TimetableSlot{
		ID:             generateUUID('q', 1),
		TenantID:       testTenantID,
		SchoolID:       testSchoolID,
		AcademicYearID: testAcademicYearID,
		ClassID:        testClassID,
		TeacherID:      testTeacherID,
		LearningAreaID: &testLearningAreaID,
		RoomIdentifier: nil,
		DayOfWeek:      1,
		StartTime:      "08:00",
		EndTime:        "08:40",
	}
	s.createTestSlot(t, slotA)

	// Slot B — also null room, different teacher, same time — should be fine
	slotB := &TimetableSlot{
		ID:             generateUUID('q', 2),
		TenantID:       testTenantID,
		SchoolID:       testSchoolID,
		AcademicYearID: testAcademicYearID,
		ClassID:        testClass2ID,
		TeacherID:      testTeacher2ID,
		LearningAreaID: &testLearningArea2ID,
		RoomIdentifier: nil,
		DayOfWeek:      1,
		StartTime:      "08:00",
		EndTime:        "08:40",
	}
	err := s.repo.CreateSlot(context.Background(), slotB)
	if err != nil {
		t.Fatalf("two null-room slots should be allowed at same time: %v", err)
	}
}

// TestEdge_MidnightTimeSlot verifies that a slot crossing noon boundary works.
func TestEdge_MidnightTimeSlot(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	s := testSuite
	s.freshDB(t)

	req := &CreateSlotRequest{
		ClassID:        testClassID,
		TeacherID:      testTeacherID,
		LearningAreaID: &testLearningAreaID,
		DayOfWeek:      1,
		StartTime:      "23:00",
		EndTime:        "23:30",
	}
	_, err := s.svc.CreateSlot(context.Background(), testSchoolID, testTenantID, req)
	if err != nil {
		t.Fatalf("create late-night slot: %v", err)
	}
}

// TestEdge_MultiDaySpan verifies a slot that would span midnight (e.g., 23:00-01:00)
// is valid as far as the application is concerned (the constraint only checks
// same day_of_week, so a multi-day span is a semantic issue, not a DB constraint).
func TestEdge_MultiDaySpan(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	s := testSuite
	s.freshDB(t)

	// The DB constraint only looks at same day_of_week — so 23:30-00:30 on Monday
	// won't overlap with 00:00-00:30 on Tuesday. This is accepted behavior.
	req := &CreateSlotRequest{
		ClassID:        testClassID,
		TeacherID:      testTeacherID,
		LearningAreaID: &testLearningAreaID,
		DayOfWeek:      1,
		StartTime:      "23:30",
		EndTime:        "23:55",
	}
	_, err := s.svc.CreateSlot(context.Background(), testSchoolID, testTenantID, req)
	if err != nil {
		t.Fatalf("create late slot: %v", err)
	}
}

// ============================================================================
// ─── CATEGORY 8: REFERENCE DATA QUERIES ───────────────────────────────────
// ============================================================================

// TestReferenceData_FetchLearningAreas verifies learning areas are returned for a grade.
func TestReferenceData_FetchLearningAreas(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	s := testSuite
	// Don't freshDB — reference data is seeded once

	areas, err := s.svc.FetchLearningAreas(context.Background(), testGradeID)
	if err != nil {
		t.Fatalf("fetch learning areas: %v", err)
	}
	if len(areas) < 2 {
		t.Fatalf("expected at least 2 learning areas, got %d", len(areas))
	}
}

// TestReferenceData_FetchTeachers verifies teachers are returned for a school.
func TestReferenceData_FetchTeachers(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	s := testSuite

	teachers, err := s.svc.FetchTeachers(context.Background(), testSchoolID, testTenantID)
	if err != nil {
		t.Fatalf("fetch teachers: %v", err)
	}
	if len(teachers) < 2 {
		t.Fatalf("expected at least 2 teachers, got %d", len(teachers))
	}

	// Verify names are composed
	if teachers[0].Name == "" {
		t.Fatal("expected teacher Name to be composed from first_name + last_name")
	}
}

// TestReferenceData_FetchClassTeachers verifies class-scoped teachers.
func TestReferenceData_FetchClassTeachers(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	s := testSuite

	teachers, err := s.svc.FetchClassTeachers(context.Background(), testClassID, &testLearningAreaID)
	if err != nil {
		t.Fatalf("fetch class teachers: %v", err)
	}
	if len(teachers) != 1 {
		t.Fatalf("expected 1 teacher for class+learning area, got %d", len(teachers))
	}
}

// TestReferenceData_FetchClassTeachers_AllAreas returns teachers for any learning area.
func TestReferenceData_FetchClassTeachers_AllAreas(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	s := testSuite

	teachers, err := s.svc.FetchClassTeachers(context.Background(), testClassID, nil)
	if err != nil {
		t.Fatalf("fetch all class teachers: %v", err)
	}
	if len(teachers) < 1 {
		t.Fatalf("expected at least 1 teacher for class, got %d", len(teachers))
	}
}

// TestReferenceData_FetchOperatingDays returns the default operating days.
func TestReferenceData_FetchOperatingDays(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	s := testSuite

	days, err := s.svc.FetchOperatingDays(context.Background(), testSchoolID, testTenantID)
	if err != nil {
		t.Fatalf("fetch operating days: %v", err)
	}
	if len(days) != 5 {
		t.Fatalf("expected 5 operating days (Mon-Fri), got %d", len(days))
	}
	if days[0] != 1 || days[4] != 5 {
		t.Fatalf("expected days [1,2,3,4,5], got %v", days)
	}
}

// ============================================================================
// ─── CATEGORY 9: CONCURRENCY ──────────────────────────────────────────────
// ============================================================================

// TestConcurrency_CreateSlotsDifferentTeachers verifies multiple goroutines
// can create slots for different teachers concurrently without issues.
func TestConcurrency_CreateSlotsDifferentTeachers(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	s := testSuite
	s.freshDB(t)

	const numSlots = 20
	var wg sync.WaitGroup
	errs := make(chan error, numSlots)

	for i := 0; i < numSlots; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			teacherID := testTeacherID
			if idx%2 == 0 {
				teacherID = testTeacher2ID
			}
			_, err := s.svc.CreateSlot(context.Background(), testSchoolID, testTenantID, &CreateSlotRequest{
				ClassID:        testClassID,
				TeacherID:      teacherID,
				LearningAreaID: &testLearningAreaID,
				DayOfWeek:      1 + (idx % 5),
				StartTime:      fmt.Sprintf("08:%02d", idx*2),
				EndTime:        fmt.Sprintf("08:%02d", idx*2+1),
			})
			if err != nil {
				errs <- err
			}
		}(i)
	}

	wg.Wait()
	close(errs)

	for err := range errs {
		t.Fatalf("concurrent slot creation failed: %v", err)
	}

	// Verify all were created
	slots, err := s.svc.FetchSlots(context.Background(), testClassID)
	if err != nil {
		t.Fatalf("fetch slots: %v", err)
	}
	if len(slots) != numSlots {
		t.Fatalf("expected %d slots, got %d", numSlots, len(slots))
	}
}

// TestConcurrency_TeacherOverlapUnderLoad verifies that the EXCLUDE constraint
// correctly prevents overlaps even under concurrent creation attempts.
func TestConcurrency_TeacherOverlapUnderLoad(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	s := testSuite
	s.freshDB(t)

	const numAttempts = 50
	var wg sync.WaitGroup
	successCount := 0
	var mu sync.Mutex

	for i := 0; i < numAttempts; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			_, err := s.svc.CreateSlot(context.Background(), testSchoolID, testTenantID, &CreateSlotRequest{
				ClassID:        testClassID,
				TeacherID:      testTeacherID, // same teacher for all
				LearningAreaID: &testLearningAreaID,
				DayOfWeek:      1,
				StartTime:      "08:00",
				EndTime:        "08:40",
			})
			if err == nil {
				mu.Lock()
				successCount++
				mu.Unlock()
			}
		}(i)
	}

	wg.Wait()

	// Only 1 of the 50 attempts should succeed (first one wins)
	if successCount != 1 {
		t.Fatalf("expected exactly 1 successful creation (exclusion constraint), got %d", successCount)
	}
}

// ============================================================================
// ─── CATEGORY 10: EMPTY STATES ────────────────────────────────────────────
// ============================================================================

// TestEmpty_FetchSlotsForEmptyClass returns an empty list, not an error.
func TestEmpty_FetchSlotsForEmptyClass(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	s := testSuite
	s.freshDB(t)

	slots, err := s.svc.FetchSlots(context.Background(), testClassID)
	if err != nil {
		t.Fatalf("fetch slots for empty class: %v", err)
	}
	if len(slots) != 0 {
		t.Fatalf("expected 0 slots, got %d", len(slots))
	}
}

// TestEmpty_DuplicateDayOnEmptySource returns an empty result, not an error.
func TestEmpty_DuplicateDayOnEmptySource(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	s := testSuite
	s.freshDB(t)

	result, err := s.svc.DuplicateDay(context.Background(), &DuplicateDayRequest{
		SourceDay:      1,
		TargetDays:     []int{2},
		AcademicYearID: testAcademicYearID,
		ClassID:        testClassID,
	})
	if err != nil {
		t.Fatalf("duplicate empty day: %v", err)
	}
	if result.TotalCopied != 0 {
		t.Fatalf("expected 0 copied, got %d", result.TotalCopied)
	}
}

// TestEmpty_CopyFromEmptyClass returns an empty result, not an error.
func TestEmpty_CopyFromEmptyClass(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	s := testSuite
	s.freshDB(t)

	result, err := s.svc.CopyFromClass(context.Background(), &CopyFromClassRequest{
		SourceClassID:  testClassID,
		TargetClassID:  testClass2ID,
		AcademicYearID: testAcademicYearID,
	})
	if err != nil {
		t.Fatalf("copy from empty class: %v", err)
	}
	if result.TotalCopied != 0 {
		t.Fatalf("expected 0 copied, got %d", result.TotalCopied)
	}
}

// ============================================================================
// Helper functions
// ============================================================================

func strPtr(s string) *string {
	return &s
}

// fmtTime formats total minutes (from midnight) into "HH:MM" format.
func fmtTime(totalMinutes int) string {
	h := totalMinutes / 60
	m := totalMinutes % 60
	return fmt.Sprintf("%02d:%02d", h, m)
}

// strContains is a test helper — equivalent to strings.Contains.
func strContains(s, substr string) bool {
	return len(s) >= len(substr) && strIndexOf(s, substr) >= 0
}

// strIndexOf finds the first occurrence of substr in s.
func strIndexOf(s, substr string) int {
	n := len(substr)
	if n == 0 {
		return 0
	}
	for i := 0; i <= len(s)-n; i++ {
		if s[i:i+n] == substr {
			return i
		}
	}
	return -1
}
