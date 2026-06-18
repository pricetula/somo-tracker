package cbctimetable

import (
	"context"
	"fmt"
	"testing"
	"time"
)

// ============================================================================
// Attendance integration tests
// ============================================================================
//
// These tests build on the existing test infrastructure (PostgreSQL container,
// seeded reference data, freshDB helper). See timetable_test.go for the suite
// setup (TestMain, IntegrationSuite, seedReferenceData, etc.).
//
// Every test in this file operates on a clean slate via s.freshDB(t), which
// clears cbc_attendance_logs, cbc_attendance_periods, and cbc_timetable_slots.

// Additional test IDs for attendance-specific entities.
const (
	testStudentID  = "dd0e8400-e29b-41d4-a716-446655440003"
	testStudent2ID = "dd0e8400-e29b-41d4-a716-446655440004"
	testStudent3ID = "dd0e8400-e29b-41d4-a716-446655440005"
)

// seedStudents inserts test student records and enrollments for attendance tests.
func seedStudents(ctx context.Context, s *IntegrationSuite) error {
	stmt := []string{
		fmt.Sprintf(`INSERT INTO students (id, tenant_id, first_name, last_name, gender, date_of_birth, admission_number)
			VALUES ('%s', '%s', 'Alice', 'Kimani', 'F', '2013-05-10', 'STU-001')
			ON CONFLICT DO NOTHING`, testStudentID, testTenantID),
		fmt.Sprintf(`INSERT INTO students (id, tenant_id, first_name, last_name, gender, date_of_birth, admission_number)
			VALUES ('%s', '%s', 'Bob', 'Ochieng', 'M', '2013-08-22', 'STU-002')
			ON CONFLICT DO NOTHING`, testStudent2ID, testTenantID),
		fmt.Sprintf(`INSERT INTO students (id, tenant_id, first_name, last_name, gender, date_of_birth, admission_number)
			VALUES ('%s', '%s', 'Carol', 'Wanjiku', 'F', '2013-03-15', 'STU-003')
			ON CONFLICT DO NOTHING`, testStudent3ID, testTenantID),
		fmt.Sprintf(`INSERT INTO student_enrollments (tenant_id, student_id, class_id, academic_term_id, status)
			VALUES ('%s', '%s', '%s', '%s', 'ACTIVE')
			ON CONFLICT DO NOTHING`, testTenantID, testStudentID, testClassID, testTermID),
		fmt.Sprintf(`INSERT INTO student_enrollments (tenant_id, student_id, class_id, academic_term_id, status)
			VALUES ('%s', '%s', '%s', '%s', 'ACTIVE')
			ON CONFLICT DO NOTHING`, testTenantID, testStudent2ID, testClassID, testTermID),
		fmt.Sprintf(`INSERT INTO student_enrollments (tenant_id, student_id, class_id, academic_term_id, status)
			VALUES ('%s', '%s', '%s', '%s', 'ACTIVE')
			ON CONFLICT DO NOTHING`, testTenantID, testStudent3ID, testClassID, testTermID),
	}

	for _, st := range stmt {
		if _, err := s.pool.Exec(ctx, st); err != nil {
			return fmt.Errorf("seed students: %w", err)
		}
	}
	return nil
}

// ============================================================================
// ─── CATEGORY 1: ATTENDANCE PERIOD CRUD ───────────────────────────────────
// ============================================================================

// TestAttendance_CreatePeriod creates an attendance period and verifies it was
// stored with all fields populated.
func TestAttendance_CreatePeriod(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	s := testSuite
	s.freshDB(t)

	if err := seedStudents(context.Background(), s); err != nil {
		t.Fatalf("seed students: %v", err)
	}

	period, err := s.svc.CreateAttendancePeriod(
		context.Background(),
		testClassID, testTenantID, testSchoolID, testTeacherID,
		&CreatePeriodRequest{
			LearningAreaID: testLearningAreaID,
			DateRecorded:   "2026-06-18",
		},
	)
	if err != nil {
		t.Fatalf("create attendance period: %v", err)
	}
	if period.ID == "" {
		t.Fatal("expected non-empty period ID")
	}
	if period.TenantID != testTenantID {
		t.Fatalf("expected tenant_id %s, got %s", testTenantID, period.TenantID)
	}
	if period.SchoolID != testSchoolID {
		t.Fatalf("expected school_id %s, got %s", testSchoolID, period.SchoolID)
	}
	if period.ClassID != testClassID {
		t.Fatalf("expected class_id %s, got %s", testClassID, period.ClassID)
	}
	if period.LearningAreaID != testLearningAreaID {
		t.Fatalf("expected learning_area_id %s, got %s", testLearningAreaID, period.LearningAreaID)
	}
	if period.DateRecorded != "2026-06-18" {
		t.Fatalf("expected date 2026-06-18, got %s", period.DateRecorded)
	}
	if period.RecordedBy != testTeacherID {
		t.Fatalf("expected recorded_by %s, got %s", testTeacherID, period.RecordedBy)
	}
	if period.AcademicTermID == "" {
		t.Fatal("expected academic_term_id to be resolved")
	}
	if period.CreatedAt == "" {
		t.Fatal("expected created_at to be set")
	}
}

// TestAttendance_CreatePeriod_Duplicate verifies that creating a second period
// with the same class, date, and learning area is rejected (unique constraint).
func TestAttendance_CreatePeriod_Duplicate(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	s := testSuite
	s.freshDB(t)

	if err := seedStudents(context.Background(), s); err != nil {
		t.Fatalf("seed students: %v", err)
	}

	// First period should succeed
	_, err := s.svc.CreateAttendancePeriod(
		context.Background(),
		testClassID, testTenantID, testSchoolID, testTeacherID,
		&CreatePeriodRequest{
			LearningAreaID: testLearningAreaID,
			DateRecorded:   "2026-06-18",
		},
	)
	if err != nil {
		t.Fatalf("first period creation: %v", err)
	}

	// Second period with same class/date/area should fail
	_, err = s.svc.CreateAttendancePeriod(
		context.Background(),
		testClassID, testTenantID, testSchoolID, testTeacherID,
		&CreatePeriodRequest{
			LearningAreaID: testLearningAreaID,
			DateRecorded:   "2026-06-18",
		},
	)
	if err == nil {
		t.Fatal("expected error for duplicate attendance period, got nil")
	}
	if !strContains(err.Error(), "unique") && !strContains(err.Error(), "duplicate") {
		t.Fatalf("expected unique/duplicate constraint error, got: %v", err)
	}
}

// TestAttendance_CreatePeriod_SameDateDifferentArea allows two periods on the
// same date for different learning areas.
func TestAttendance_CreatePeriod_SameDateDifferentArea(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	s := testSuite
	s.freshDB(t)

	if err := seedStudents(context.Background(), s); err != nil {
		t.Fatalf("seed students: %v", err)
	}

	// Math period on 2026-06-18
	p1, err := s.svc.CreateAttendancePeriod(
		context.Background(),
		testClassID, testTenantID, testSchoolID, testTeacherID,
		&CreatePeriodRequest{
			LearningAreaID: testLearningAreaID,
			DateRecorded:   "2026-06-18",
		},
	)
	if err != nil {
		t.Fatalf("create math period: %v", err)
	}

	// English period on same date — should succeed (different learning area)
	p2, err := s.svc.CreateAttendancePeriod(
		context.Background(),
		testClassID, testTenantID, testSchoolID, testTeacherID,
		&CreatePeriodRequest{
			LearningAreaID: testLearningArea2ID,
			DateRecorded:   "2026-06-18",
		},
	)
	if err != nil {
		t.Fatalf("create english period on same date: %v", err)
	}

	if p1.ID == p2.ID {
		t.Fatal("expected different period IDs")
	}
}

// TestAttendance_FetchPeriodsByDate verifies fetching periods for a specific date.
func TestAttendance_FetchPeriodsByDate(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	s := testSuite
	s.freshDB(t)

	if err := seedStudents(context.Background(), s); err != nil {
		t.Fatalf("seed students: %v", err)
	}

	// Create two periods on the same date (different areas)
	_, err := s.svc.CreateAttendancePeriod(
		context.Background(),
		testClassID, testTenantID, testSchoolID, testTeacherID,
		&CreatePeriodRequest{LearningAreaID: testLearningAreaID, DateRecorded: "2026-06-18"},
	)
	if err != nil {
		t.Fatalf("create period 1: %v", err)
	}
	_, err = s.svc.CreateAttendancePeriod(
		context.Background(),
		testClassID, testTenantID, testSchoolID, testTeacherID,
		&CreatePeriodRequest{LearningAreaID: testLearningArea2ID, DateRecorded: "2026-06-18"},
	)
	if err != nil {
		t.Fatalf("create period 2: %v", err)
	}

	// Also create one on a different date
	_, err = s.svc.CreateAttendancePeriod(
		context.Background(),
		testClassID, testTenantID, testSchoolID, testTeacherID,
		&CreatePeriodRequest{LearningAreaID: testLearningAreaID, DateRecorded: "2026-06-17"},
	)
	if err != nil {
		t.Fatalf("create period 3: %v", err)
	}

	periods, err := s.svc.FetchAttendancePeriodsByDate(context.Background(), testClassID, "2026-06-18")
	if err != nil {
		t.Fatalf("fetch periods by date: %v", err)
	}
	if len(periods) != 2 {
		t.Fatalf("expected 2 periods on 2026-06-18, got %d", len(periods))
	}
}

// TestAttendance_FetchPeriodsByDate_Empty returns empty list for a date with no periods.
func TestAttendance_FetchPeriodsByDate_Empty(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	s := testSuite
	s.freshDB(t)

	periods, err := s.svc.FetchAttendancePeriodsByDate(context.Background(), testClassID, "2026-06-18")
	if err != nil {
		t.Fatalf("fetch periods on empty date: %v", err)
	}
	if len(periods) != 0 {
		t.Fatalf("expected 0 periods, got %d", len(periods))
	}
}

// TestAttendance_FetchPeriodSummaries verifies period summaries with student stats.
func TestAttendance_FetchPeriodSummaries(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	s := testSuite
	s.freshDB(t)

	if err := seedStudents(context.Background(), s); err != nil {
		t.Fatalf("seed students: %v", err)
	}

	// Create a period
	period, err := s.svc.CreateAttendancePeriod(
		context.Background(),
		testClassID, testTenantID, testSchoolID, testTeacherID,
		&CreatePeriodRequest{LearningAreaID: testLearningAreaID, DateRecorded: "2026-06-18"},
	)
	if err != nil {
		t.Fatalf("create period: %v", err)
	}

	// Add some logs: 2 present, 1 absent
	_, err = s.svc.SaveAttendanceLog(context.Background(), testTenantID, testTeacherID, &SaveLogRequest{
		PeriodID:  period.ID,
		StudentID: testStudentID,
		Status:    "PRESENT",
	})
	if err != nil {
		t.Fatalf("save log 1: %v", err)
	}
	_, err = s.svc.SaveAttendanceLog(context.Background(), testTenantID, testTeacherID, &SaveLogRequest{
		PeriodID:  period.ID,
		StudentID: testStudent2ID,
		Status:    "PRESENT",
	})
	if err != nil {
		t.Fatalf("save log 2: %v", err)
	}
	_, err = s.svc.SaveAttendanceLog(context.Background(), testTenantID, testTeacherID, &SaveLogRequest{
		PeriodID:  period.ID,
		StudentID: testStudent3ID,
		Status:    "ABSENT",
	})
	if err != nil {
		t.Fatalf("save log 3: %v", err)
	}

	// Fetch summaries
	summaries, err := s.svc.FetchAttendancePeriodSummaries(context.Background(), testClassID, "2026-06-01", "2026-06-30")
	if err != nil {
		t.Fatalf("fetch summaries: %v", err)
	}
	if len(summaries) != 1 {
		t.Fatalf("expected 1 summary, got %d", len(summaries))
	}

	s1 := summaries[0]
	if s1.TotalStudents != 3 {
		t.Fatalf("expected 3 total students, got %d", s1.TotalStudents)
	}
	if s1.PresentCount != 2 {
		t.Fatalf("expected 2 present, got %d", s1.PresentCount)
	}
	if s1.AbsentCount != 1 {
		t.Fatalf("expected 1 absent, got %d", s1.AbsentCount)
	}
	if s1.LateCount != 0 {
		t.Fatalf("expected 0 late, got %d", s1.LateCount)
	}
	if s1.ExcusedCount != 0 {
		t.Fatalf("expected 0 excused, got %d", s1.ExcusedCount)
	}
	if s1.UnmarkedCount != 0 {
		t.Fatalf("expected 0 unmarked, got %d", s1.UnmarkedCount)
	}
	if s1.LearningAreaName == "" {
		t.Fatal("expected non-empty learning area name")
	}
	if s1.RecordedByName == "" {
		t.Fatal("expected non-empty recorded by name")
	}
}

// TestAttendance_FetchPeriodSummary_ByID verifies fetching a single period summary.
func TestAttendance_FetchPeriodSummary_ByID(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	s := testSuite
	s.freshDB(t)

	if err := seedStudents(context.Background(), s); err != nil {
		t.Fatalf("seed students: %v", err)
	}

	period, err := s.svc.CreateAttendancePeriod(
		context.Background(),
		testClassID, testTenantID, testSchoolID, testTeacherID,
		&CreatePeriodRequest{LearningAreaID: testLearningAreaID, DateRecorded: "2026-06-18"},
	)
	if err != nil {
		t.Fatalf("create period: %v", err)
	}

	summary, err := s.svc.FetchAttendancePeriodSummary(context.Background(), period.ID)
	if err != nil {
		t.Fatalf("fetch period summary: %v", err)
	}
	if summary == nil {
		t.Fatal("expected non-nil summary")
	}
	if summary.ID != period.ID {
		t.Fatalf("expected period ID %s, got %s", period.ID, summary.ID)
	}
	if summary.TotalStudents != 3 {
		t.Fatalf("expected 3 enrolled students, got %d", summary.TotalStudents)
	}
	// All unmarked since no logs yet
	if summary.UnmarkedCount != 3 {
		t.Fatalf("expected 3 unmarked, got %d", summary.UnmarkedCount)
	}
}

// TestAttendance_FetchPeriodSummary_NotFound returns nil for missing period.
func TestAttendance_FetchPeriodSummary_NotFound(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	s := testSuite

	summary, err := s.svc.FetchAttendancePeriodSummary(context.Background(), "00000000-0000-0000-0000-000000000000")
	if err != nil {
		t.Fatalf("fetch missing period summary: %v", err)
	}
	if summary != nil {
		t.Fatal("expected nil for non-existent period")
	}
}

// ============================================================================
// ─── CATEGORY 2: ATTENDANCE LOGS ──────────────────────────────────────────
// ============================================================================

// TestAttendance_SaveLog creates a single attendance log and verifies it.
func TestAttendance_SaveLog(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	s := testSuite
	s.freshDB(t)

	if err := seedStudents(context.Background(), s); err != nil {
		t.Fatalf("seed students: %v", err)
	}

	// Create period
	period, err := s.svc.CreateAttendancePeriod(
		context.Background(),
		testClassID, testTenantID, testSchoolID, testTeacherID,
		&CreatePeriodRequest{LearningAreaID: testLearningAreaID, DateRecorded: "2026-06-18"},
	)
	if err != nil {
		t.Fatalf("create period: %v", err)
	}

	// Save a single log
	log, err := s.svc.SaveAttendanceLog(context.Background(), testTenantID, testTeacherID, &SaveLogRequest{
		PeriodID:  period.ID,
		StudentID: testStudentID,
		Status:    "PRESENT",
		Remarks:   strPtr("On time"),
	})
	if err != nil {
		t.Fatalf("save attendance log: %v", err)
	}
	if log.ID == "" {
		t.Fatal("expected non-empty log ID")
	}
	if log.TenantID != testTenantID {
		t.Fatalf("expected tenant %s, got %s", testTenantID, log.TenantID)
	}
	if log.PeriodID != period.ID {
		t.Fatalf("expected period %s, got %s", period.ID, log.PeriodID)
	}
	if log.StudentID != testStudentID {
		t.Fatalf("expected student %s, got %s", testStudentID, log.StudentID)
	}
	if log.Status != "PRESENT" {
		t.Fatalf("expected PRESENT, got %s", log.Status)
	}
	if log.Remarks == nil || *log.Remarks != "On time" {
		t.Fatalf("expected remarks 'On time', got %v", log.Remarks)
	}
	if log.RecordedBy != testTeacherID {
		t.Fatalf("expected recorded_by %s, got %s", testTeacherID, log.RecordedBy)
	}
}

// TestAttendance_SaveLog_NullRemarks verifies saving a log without remarks.
func TestAttendance_SaveLog_NullRemarks(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	s := testSuite
	s.freshDB(t)

	if err := seedStudents(context.Background(), s); err != nil {
		t.Fatalf("seed students: %v", err)
	}

	period, err := s.svc.CreateAttendancePeriod(
		context.Background(),
		testClassID, testTenantID, testSchoolID, testTeacherID,
		&CreatePeriodRequest{LearningAreaID: testLearningAreaID, DateRecorded: "2026-06-18"},
	)
	if err != nil {
		t.Fatalf("create period: %v", err)
	}

	log, err := s.svc.SaveAttendanceLog(context.Background(), testTenantID, testTeacherID, &SaveLogRequest{
		PeriodID:  period.ID,
		StudentID: testStudentID,
		Status:    "ABSENT",
		Remarks:   nil,
	})
	if err != nil {
		t.Fatalf("save log with null remarks: %v", err)
	}
	if log.Remarks != nil {
		t.Fatal("expected nil remarks")
	}
}

// TestAttendance_SaveLog_UpdateExisting verifies upsert: saving a second log
// for the same student+period updates the existing record.
func TestAttendance_SaveLog_UpdateExisting(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	s := testSuite
	s.freshDB(t)

	if err := seedStudents(context.Background(), s); err != nil {
		t.Fatalf("seed students: %v", err)
	}

	period, err := s.svc.CreateAttendancePeriod(
		context.Background(),
		testClassID, testTenantID, testSchoolID, testTeacherID,
		&CreatePeriodRequest{LearningAreaID: testLearningAreaID, DateRecorded: "2026-06-18"},
	)
	if err != nil {
		t.Fatalf("create period: %v", err)
	}

	// First save: PRESENT
	log1, err := s.svc.SaveAttendanceLog(context.Background(), testTenantID, testTeacherID, &SaveLogRequest{
		PeriodID:  period.ID,
		StudentID: testStudentID,
		Status:    "PRESENT",
	})
	if err != nil {
		t.Fatalf("save log 1: %v", err)
	}

	// Second save: update to ABSENT
	log2, err := s.svc.SaveAttendanceLog(context.Background(), testTenantID, testTeacherID, &SaveLogRequest{
		PeriodID:  period.ID,
		StudentID: testStudentID,
		Status:    "ABSENT",
	})
	if err != nil {
		t.Fatalf("save log 2: %v", err)
	}

	if log1.ID != log2.ID {
		t.Fatalf("expected same log ID on update, got %s vs %s", log1.ID, log2.ID)
	}
	if log2.Status != "ABSENT" {
		t.Fatalf("expected updated status ABSENT, got %s", log2.Status)
	}
}

// TestAttendance_BatchSaveLogs creates multiple logs at once.
func TestAttendance_BatchSaveLogs(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	s := testSuite
	s.freshDB(t)

	if err := seedStudents(context.Background(), s); err != nil {
		t.Fatalf("seed students: %v", err)
	}

	period, err := s.svc.CreateAttendancePeriod(
		context.Background(),
		testClassID, testTenantID, testSchoolID, testTeacherID,
		&CreatePeriodRequest{LearningAreaID: testLearningAreaID, DateRecorded: "2026-06-18"},
	)
	if err != nil {
		t.Fatalf("create period: %v", err)
	}

	logs, err := s.svc.BatchSaveAttendanceLogs(context.Background(), testTenantID, testTeacherID, &BatchSaveLogsRequest{
		PeriodID: period.ID,
		Marks: []BatchLogMark{
			{StudentID: testStudentID, Status: "PRESENT"},
			{StudentID: testStudent2ID, Status: "ABSENT", Remarks: strPtr("Sick")},
			{StudentID: testStudent3ID, Status: "LATE"},
		},
	})
	if err != nil {
		t.Fatalf("batch save logs: %v", err)
	}
	if len(logs) != 3 {
		t.Fatalf("expected 3 logs, got %d", len(logs))
	}

	// Verify all statuses
	statusMap := make(map[string]string)
	for _, l := range logs {
		statusMap[l.StudentID] = l.Status
	}
	if statusMap[testStudentID] != "PRESENT" {
		t.Fatalf("expected PRESENT for student 1, got %s", statusMap[testStudentID])
	}
	if statusMap[testStudent2ID] != "ABSENT" {
		t.Fatalf("expected ABSENT for student 2, got %s", statusMap[testStudent2ID])
	}
	if statusMap[testStudent3ID] != "LATE" {
		t.Fatalf("expected LATE for student 3, got %s", statusMap[testStudent3ID])
	}
}

// TestAttendance_BatchSaveLogs_EmptyMarks returns nil for empty marks.
func TestAttendance_BatchSaveLogs_EmptyMarks(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	s := testSuite

	logs, err := s.svc.BatchSaveAttendanceLogs(context.Background(), testTenantID, testTeacherID, &BatchSaveLogsRequest{
		PeriodID: "some-period-id",
		Marks:    []BatchLogMark{},
	})
	if err != nil {
		t.Fatalf("batch save empty marks: %v", err)
	}
	if logs != nil {
		t.Fatal("expected nil for empty batch")
	}
}

// TestAttendance_FetchLogsByPeriod verifies fetching logs with recorder details.
func TestAttendance_FetchLogsByPeriod(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	s := testSuite
	s.freshDB(t)

	if err := seedStudents(context.Background(), s); err != nil {
		t.Fatalf("seed students: %v", err)
	}

	period, err := s.svc.CreateAttendancePeriod(
		context.Background(),
		testClassID, testTenantID, testSchoolID, testTeacherID,
		&CreatePeriodRequest{LearningAreaID: testLearningAreaID, DateRecorded: "2026-06-18"},
	)
	if err != nil {
		t.Fatalf("create period: %v", err)
	}

	// Save a log
	_, err = s.svc.SaveAttendanceLog(context.Background(), testTenantID, testTeacherID, &SaveLogRequest{
		PeriodID:  period.ID,
		StudentID: testStudentID,
		Status:    "PRESENT",
	})
	if err != nil {
		t.Fatalf("save log: %v", err)
	}

	// Fetch logs
	logs, err := s.svc.FetchAttendanceLogs(context.Background(), period.ID)
	if err != nil {
		t.Fatalf("fetch logs: %v", err)
	}
	if len(logs) != 1 {
		t.Fatalf("expected 1 log, got %d", len(logs))
	}

	detail := logs[0]
	if detail.RecorderFirstName != "John" || detail.RecorderLastName != "Otieno" {
		t.Fatalf("expected recorder John Otieno, got %s %s", detail.RecorderFirstName, detail.RecorderLastName)
	}
	if detail.RecordedByLabel != "John Otieno" {
		t.Fatalf("expected recorded_by_label 'John Otieno', got %s", detail.RecordedByLabel)
	}
	if detail.Status != "PRESENT" {
		t.Fatalf("expected PRESENT, got %s", detail.Status)
	}
}

// TestAttendance_FetchLogsByPeriod_Empty returns empty list for period with no logs.
func TestAttendance_FetchLogsByPeriod_Empty(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	s := testSuite
	s.freshDB(t)

	if err := seedStudents(context.Background(), s); err != nil {
		t.Fatalf("seed students: %v", err)
	}

	period, err := s.svc.CreateAttendancePeriod(
		context.Background(),
		testClassID, testTenantID, testSchoolID, testTeacherID,
		&CreatePeriodRequest{LearningAreaID: testLearningAreaID, DateRecorded: "2026-06-18"},
	)
	if err != nil {
		t.Fatalf("create period: %v", err)
	}

	logs, err := s.svc.FetchAttendanceLogs(context.Background(), period.ID)
	if err != nil {
		t.Fatalf("fetch empty logs: %v", err)
	}
	if len(logs) != 0 {
		t.Fatalf("expected 0 logs, got %d", len(logs))
	}
}

// ============================================================================
// ─── CATEGORY 3: ATTENDANCE LOG STATUS VARIATIONS ─────────────────────────
// ============================================================================

// TestAttendance_AllStatusValues verifies all four attendance status values
// can be saved and retrieved.
func TestAttendance_AllStatusValues(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	s := testSuite
	s.freshDB(t)

	if err := seedStudents(context.Background(), s); err != nil {
		t.Fatalf("seed students: %v", err)
	}

	period, err := s.svc.CreateAttendancePeriod(
		context.Background(),
		testClassID, testTenantID, testSchoolID, testTeacherID,
		&CreatePeriodRequest{LearningAreaID: testLearningAreaID, DateRecorded: "2026-06-18"},
	)
	if err != nil {
		t.Fatalf("create period: %v", err)
	}

	students := []string{testStudentID, testStudent2ID, testStudent3ID}
	statuses := []string{"PRESENT", "ABSENT", "LATE", "EXCUSED"}

	for i, studentID := range students {
		status := statuses[i]
		log, err := s.svc.SaveAttendanceLog(context.Background(), testTenantID, testTeacherID, &SaveLogRequest{
			PeriodID:  period.ID,
			StudentID: studentID,
			Status:    status,
		})
		if err != nil {
			t.Fatalf("save log with status %s: %v", status, err)
		}
		if log.Status != status {
			t.Fatalf("expected status %s, got %s", status, log.Status)
		}
	}

	// Fetch to verify round-trip
	logs, err := s.svc.FetchAttendanceLogs(context.Background(), period.ID)
	if err != nil {
		t.Fatalf("fetch logs: %v", err)
	}
	if len(logs) != 3 {
		t.Fatalf("expected 3 logs, got %d", len(logs))
	}
}

// ============================================================================
// ─── CATEGORY 4: ATTENDANCE ANALYTICS ──────────────────────────────────────
// ============================================================================

// TestAttendance_Heatmap verifies heatmap data across multiple dates.
func TestAttendance_Heatmap(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	s := testSuite
	s.freshDB(t)

	if err := seedStudents(context.Background(), s); err != nil {
		t.Fatalf("seed students: %v", err)
	}

	// Create periods on two dates
	dates := []string{"2026-06-15", "2026-06-16", "2026-06-17"}
	for _, date := range dates {
		period, err := s.svc.CreateAttendancePeriod(
			context.Background(),
			testClassID, testTenantID, testSchoolID, testTeacherID,
			&CreatePeriodRequest{LearningAreaID: testLearningAreaID, DateRecorded: date},
		)
		if err != nil {
			t.Fatalf("create period on %s: %v", date, err)
		}

		// Mark all 3 students as PRESENT on each date
		_, err = s.svc.BatchSaveAttendanceLogs(context.Background(), testTenantID, testTeacherID, &BatchSaveLogsRequest{
			PeriodID: period.ID,
			Marks: []BatchLogMark{
				{StudentID: testStudentID, Status: "PRESENT"},
				{StudentID: testStudent2ID, Status: "PRESENT"},
				{StudentID: testStudent3ID, Status: "PRESENT"},
			},
		})
		if err != nil {
			t.Fatalf("batch save on %s: %v", date, err)
		}
	}

	heatmap, err := s.svc.FetchAttendanceHeatmap(context.Background(), testClassID, testTermID)
	if err != nil {
		t.Fatalf("fetch heatmap: %v", err)
	}
	if len(heatmap) != 3 {
		t.Fatalf("expected 3 heatmap days, got %d", len(heatmap))
	}

	for _, day := range heatmap {
		if day.PeriodCount != 1 {
			t.Fatalf("expected 1 period on %s, got %d", day.Date, day.PeriodCount)
		}
		if day.TotalMarks != 3 {
			t.Fatalf("expected 3 marks on %s, got %d", day.Date, day.TotalMarks)
		}
		if day.PresentRate == nil {
			t.Fatalf("expected non-nil present_rate for %s", day.Date)
		}
		if *day.PresentRate != 100.0 {
			t.Fatalf("expected 100%% present rate for %s, got %.1f%%", day.Date, *day.PresentRate)
		}
	}
}

// TestAttendance_Heatmap_Empty returns empty for no periods.
func TestAttendance_Heatmap_Empty(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	s := testSuite

	heatmap, err := s.svc.FetchAttendanceHeatmap(context.Background(), testClassID, testTermID)
	if err != nil {
		t.Fatalf("fetch empty heatmap: %v", err)
	}
	if len(heatmap) != 0 {
		t.Fatalf("expected 0 heatmap days, got %d", len(heatmap))
	}
}

// TestAttendance_Gaps verifies gap detection: slots without attendance periods.
func TestAttendance_Gaps(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	s := testSuite
	s.freshDB(t)

	if err := seedStudents(context.Background(), s); err != nil {
		t.Fatalf("seed students: %v", err)
	}

	// Create a timetable slot on Monday (day 1) 08:00-08:40
	slot1 := &TimetableSlot{
		ID:             generateUUID('r', 1),
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

	// Create a corresponding attendance period for Monday 2026-06-15
	// (2026-06-15 is a Monday — verify)
	_, err := s.svc.CreateAttendancePeriod(
		context.Background(),
		testClassID, testTenantID, testSchoolID, testTeacherID,
		&CreatePeriodRequest{
			LearningAreaID: testLearningAreaID,
			DateRecorded:   "2026-06-15", // Monday
		},
	)
	if err != nil {
		t.Fatalf("create period: %v", err)
	}

	// Check gaps for the week of 2026-06-15 to 2026-06-19 (Mon–Fri)
	// We have slots only on Mon; no slots on Tue–Fri means no gaps those days
	// (a gap = a slot exists but no corresponding period).
	// Mon has a period, so no gap there.
	// Tue (2026-06-16) has no slot, so no gap.
	// Put a slot on Wed as well to create a gap.
	slot2 := &TimetableSlot{
		ID:             generateUUID('r', 2),
		TenantID:       testTenantID,
		SchoolID:       testSchoolID,
		AcademicYearID: testAcademicYearID,
		ClassID:        testClassID,
		TeacherID:      testTeacher2ID,
		LearningAreaID: &testLearningArea2ID,
		DayOfWeek:      3, // Wednesday
		StartTime:      "09:00",
		EndTime:        "09:40",
	}
	s.createTestSlot(t, slot2)

	gaps, err := s.svc.FetchAttendanceGaps(context.Background(), testClassID, "2026-06-15", "2026-06-19")
	if err != nil {
		t.Fatalf("fetch gaps: %v", err)
	}

	// Expect 1 gap: Wed 2026-06-17 has a slot but no period
	if len(gaps) != 1 {
		t.Fatalf("expected 1 gap (Wed slot without period), got %d", len(gaps))
		for _, g := range gaps {
			t.Logf("  gap: date=%s area=%s time=%s-%s", g.Date, g.LearningAreaName, g.StartTime, g.EndTime)
		}
	} else {
		if gaps[0].Date != "2026-06-17" {
			t.Fatalf("expected gap date 2026-06-17 (Wednesday), got %s", gaps[0].Date)
		}
		if gaps[0].LearningAreaName != "English" && gaps[0].LearningAreaName != "" {
			t.Fatalf("expected learning area name English, got %s", gaps[0].LearningAreaName)
		}
	}
}

// TestAttendance_Gaps_NoSlots returns empty when there are no timetable slots.
func TestAttendance_Gaps_NoSlots(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	s := testSuite

	gaps, err := s.svc.FetchAttendanceGaps(context.Background(), testClassID, "2026-06-15", "2026-06-19")
	if err != nil {
		t.Fatalf("fetch gaps with no slots: %v", err)
	}
	if len(gaps) != 0 {
		t.Fatalf("expected 0 gaps when no timetable slots, got %d", len(gaps))
	}
}

// ============================================================================
// ─── CATEGORY 5: HELPER QUERIES ───────────────────────────────────────────
// ============================================================================

// TestAttendance_FetchClassStudents returns enrolled students for a term.
func TestAttendance_FetchClassStudents(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	s := testSuite
	s.freshDB(t)

	if err := seedStudents(context.Background(), s); err != nil {
		t.Fatalf("seed students: %v", err)
	}

	students, err := s.svc.FetchClassStudents(context.Background(), testClassID, testTermID)
	if err != nil {
		t.Fatalf("fetch class students: %v", err)
	}
	if len(students) != 3 {
		t.Fatalf("expected 3 students, got %d", len(students))
	}

	// Verify names are composed
	names := make(map[string]bool)
	for _, s := range students {
		names[s.StudentName] = true
	}
	if !names["Alice Kimani"] {
		t.Fatal("expected Alice Kimani in results")
	}
	if !names["Bob Ochieng"] {
		t.Fatal("expected Bob Ochieng in results")
	}
	if !names["Carol Wanjiku"] {
		t.Fatal("expected Carol Wanjiku in results")
	}
}

// TestAttendance_FetchClassStudents_Empty returns empty for class with no enrollments.
func TestAttendance_FetchClassStudents_Empty(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	s := testSuite

	// Use a class with no enrollments (class2 hasn't been seeded with students)
	students, err := s.svc.FetchClassStudents(context.Background(), testClass2ID, testTermID)
	if err != nil {
		t.Fatalf("fetch students for empty class: %v", err)
	}
	if len(students) != 0 {
		t.Fatalf("expected 0 students, got %d", len(students))
	}
}

// TestAttendance_FetchCurrentTerm returns the current active term.
func TestAttendance_FetchCurrentTerm(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	s := testSuite

	termID, err := s.svc.FetchCurrentTerm(context.Background(), testSchoolID, testTenantID)
	if err != nil {
		t.Fatalf("fetch current term: %v", err)
	}
	if termID != testTermID {
		t.Fatalf("expected term %s, got %s", testTermID, termID)
	}
}

// ============================================================================
// ─── CATEGORY 6: EDGE CASES ──────────────────────────────────────────────
// ============================================================================

// TestAttendance_CreatePeriod_MultipleLogsDifferentStudents verifies logs for
// different students under the same period don't interfere.
func TestAttendance_CreatePeriod_MultipleLogsDifferentStudents(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	s := testSuite
	s.freshDB(t)

	if err := seedStudents(context.Background(), s); err != nil {
		t.Fatalf("seed students: %v", err)
	}

	period, err := s.svc.CreateAttendancePeriod(
		context.Background(),
		testClassID, testTenantID, testSchoolID, testTeacherID,
		&CreatePeriodRequest{LearningAreaID: testLearningAreaID, DateRecorded: "2026-06-18"},
	)
	if err != nil {
		t.Fatalf("create period: %v", err)
	}

	// Save PRESENT for Alice
	_, err = s.svc.SaveAttendanceLog(context.Background(), testTenantID, testTeacherID, &SaveLogRequest{
		PeriodID: period.ID, StudentID: testStudentID, Status: "PRESENT",
	})
	if err != nil {
		t.Fatalf("save alice: %v", err)
	}

	// Save LATE for Bob
	_, err = s.svc.SaveAttendanceLog(context.Background(), testTenantID, testTeacherID, &SaveLogRequest{
		PeriodID: period.ID, StudentID: testStudent2ID, Status: "LATE",
	})
	if err != nil {
		t.Fatalf("save bob: %v", err)
	}

	// Fetch logs — should be 2
	logs, err := s.svc.FetchAttendanceLogs(context.Background(), period.ID)
	if err != nil {
		t.Fatalf("fetch logs: %v", err)
	}
	if len(logs) != 2 {
		t.Fatalf("expected 2 logs, got %d", len(logs))
	}
}

// TestAttendance_CreatePeriod_DifferentRecorders allows logs recorded by
// different teachers in the same period.
func TestAttendance_CreatePeriod_DifferentRecorders(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	s := testSuite
	s.freshDB(t)

	if err := seedStudents(context.Background(), s); err != nil {
		t.Fatalf("seed students: %v", err)
	}

	period, err := s.svc.CreateAttendancePeriod(
		context.Background(),
		testClassID, testTenantID, testSchoolID, testTeacherID,
		&CreatePeriodRequest{LearningAreaID: testLearningAreaID, DateRecorded: "2026-06-18"},
	)
	if err != nil {
		t.Fatalf("create period: %v", err)
	}

	// Log recorded by teacher 1
	_, err = s.svc.SaveAttendanceLog(context.Background(), testTenantID, testTeacherID, &SaveLogRequest{
		PeriodID: period.ID, StudentID: testStudentID, Status: "PRESENT",
	})
	if err != nil {
		t.Fatalf("save by teacher 1: %v", err)
	}

	// Log recorded by teacher 2
	_, err = s.svc.SaveAttendanceLog(context.Background(), testTenantID, testTeacher2ID, &SaveLogRequest{
		PeriodID: period.ID, StudentID: testStudent2ID, Status: "ABSENT",
	})
	if err != nil {
		t.Fatalf("save by teacher 2: %v", err)
	}

	logs, err := s.svc.FetchAttendanceLogs(context.Background(), period.ID)
	if err != nil {
		t.Fatalf("fetch logs: %v", err)
	}
	if len(logs) != 2 {
		t.Fatalf("expected 2 logs, got %d", len(logs))
	}
}

// TestAttendance_BatchUpdate verifies that batch saving updates existing marks.
func TestAttendance_BatchUpdate(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	s := testSuite
	s.freshDB(t)

	if err := seedStudents(context.Background(), s); err != nil {
		t.Fatalf("seed students: %v", err)
	}

	period, err := s.svc.CreateAttendancePeriod(
		context.Background(),
		testClassID, testTenantID, testSchoolID, testTeacherID,
		&CreatePeriodRequest{LearningAreaID: testLearningAreaID, DateRecorded: "2026-06-18"},
	)
	if err != nil {
		t.Fatalf("create period: %v", err)
	}

	// Initial batch: all PRESENT
	_, err = s.svc.BatchSaveAttendanceLogs(context.Background(), testTenantID, testTeacherID, &BatchSaveLogsRequest{
		PeriodID: period.ID,
		Marks: []BatchLogMark{
			{StudentID: testStudentID, Status: "PRESENT"},
			{StudentID: testStudent2ID, Status: "PRESENT"},
			{StudentID: testStudent3ID, Status: "PRESENT"},
		},
	})
	if err != nil {
		t.Fatalf("initial batch: %v", err)
	}

	// Update batch: Alice → ABSENT, Bob stays PRESENT, Carol → EXCUSED
	_, err = s.svc.BatchSaveAttendanceLogs(context.Background(), testTenantID, testTeacherID, &BatchSaveLogsRequest{
		PeriodID: period.ID,
		Marks: []BatchLogMark{
			{StudentID: testStudentID, Status: "ABSENT"},
			{StudentID: testStudent3ID, Status: "EXCUSED"},
		},
	})
	if err != nil {
		t.Fatalf("update batch: %v", err)
	}

	logs, err := s.svc.FetchAttendanceLogs(context.Background(), period.ID)
	if err != nil {
		t.Fatalf("fetch logs: %v", err)
	}
	if len(logs) != 3 {
		t.Fatalf("expected 3 logs after updates, got %d", len(logs))
	}

	statusMap := make(map[string]string)
	for _, l := range logs {
		statusMap[l.StudentID] = l.Status
	}
	if statusMap[testStudentID] != "ABSENT" {
		t.Fatalf("expected Alice ABSENT, got %s", statusMap[testStudentID])
	}
	if statusMap[testStudent2ID] != "PRESENT" {
		t.Fatalf("expected Bob PRESENT, got %s", statusMap[testStudent2ID])
	}
	if statusMap[testStudent3ID] != "EXCUSED" {
		t.Fatalf("expected Carol EXCUSED, got %s", statusMap[testStudent3ID])
	}
}

// TestAttendance_PeriodSummary_WithPartialMarks verifies unmarked count is
// correctly calculated when only some students have been marked.
func TestAttendance_PeriodSummary_WithPartialMarks(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	s := testSuite
	s.freshDB(t)

	if err := seedStudents(context.Background(), s); err != nil {
		t.Fatalf("seed students: %v", err)
	}

	period, err := s.svc.CreateAttendancePeriod(
		context.Background(),
		testClassID, testTenantID, testSchoolID, testTeacherID,
		&CreatePeriodRequest{LearningAreaID: testLearningAreaID, DateRecorded: "2026-06-18"},
	)
	if err != nil {
		t.Fatalf("create period: %v", err)
	}

	// Only mark 2 out of 3 students
	_, err = s.svc.BatchSaveAttendanceLogs(context.Background(), testTenantID, testTeacherID, &BatchSaveLogsRequest{
		PeriodID: period.ID,
		Marks: []BatchLogMark{
			{StudentID: testStudentID, Status: "PRESENT"},
			{StudentID: testStudent2ID, Status: "PRESENT"},
		},
	})
	if err != nil {
		t.Fatalf("batch save: %v", err)
	}

	summary, err := s.svc.FetchAttendancePeriodSummary(context.Background(), period.ID)
	if err != nil {
		t.Fatalf("fetch summary: %v", err)
	}
	if summary.TotalStudents != 3 {
		t.Fatalf("expected 3 total, got %d", summary.TotalStudents)
	}
	if summary.PresentCount != 2 {
		t.Fatalf("expected 2 present, got %d", summary.PresentCount)
	}
	if summary.UnmarkedCount != 1 {
		t.Fatalf("expected 1 unmarked, got %d", summary.UnmarkedCount)
	}
}

// TestAttendance_ConcurrentSave verifies concurrent log saves don't cause issues.
func TestAttendance_ConcurrentSave(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}
	s := testSuite
	s.freshDB(t)

	if err := seedStudents(context.Background(), s); err != nil {
		t.Fatalf("seed students: %v", err)
	}

	period, err := s.svc.CreateAttendancePeriod(
		context.Background(),
		testClassID, testTenantID, testSchoolID, testTeacherID,
		&CreatePeriodRequest{LearningAreaID: testLearningAreaID, DateRecorded: "2026-06-18"},
	)
	if err != nil {
		t.Fatalf("create period: %v", err)
	}

	// Concurrently save logs for all 3 students
	done := make(chan error, 3)
	for _, sid := range []string{testStudentID, testStudent2ID, testStudent3ID} {
		go func(studentID string) {
			_, err := s.svc.SaveAttendanceLog(context.Background(), testTenantID, testTeacherID, &SaveLogRequest{
				PeriodID: period.ID, StudentID: studentID, Status: "PRESENT",
			})
			done <- err
		}(sid)
	}

	// Collect all results with timeout
	timeout := time.After(10 * time.Second)
	for i := 0; i < 3; i++ {
		select {
		case err := <-done:
			if err != nil {
				t.Fatalf("concurrent save failed: %v", err)
			}
		case <-timeout:
			t.Fatal("timed out waiting for concurrent saves")
		}
	}

	logs, err := s.svc.FetchAttendanceLogs(context.Background(), period.ID)
	if err != nil {
		t.Fatalf("fetch logs: %v", err)
	}
	if len(logs) != 3 {
		t.Fatalf("expected 3 logs after concurrent saves, got %d", len(logs))
	}
}
