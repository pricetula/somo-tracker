package attendance

import (
	"context"
	"errors"
	"testing"
)

// ============================================================================
// MockRepository
// ============================================================================

type MockRepository struct {
	createSessionFn                       func(ctx context.Context, tenantID, schoolID string, payload CreateSessionPayload) (*AttendanceSession, error)
	getSessionByIDFn                      func(ctx context.Context, id, tenantID string) (*AttendanceSession, error)
	getEnrichedSessionByIDFn              func(ctx context.Context, id, tenantID string) (*SessionWithEnrichedData, error)
	listSessionsFn                        func(ctx context.Context, filter SessionFilter) ([]SessionWithEnrichedData, error)
	updateSessionFn                       func(ctx context.Context, id, tenantID string, payload UpdateSessionPayload) (*AttendanceSession, error)
	getSessionsForClassDateFn             func(ctx context.Context, tenantID, schoolID, classID, date string) ([]SessionWithEnrichedData, error)
	batchMarkFn                           func(ctx context.Context, tenantID, schoolID string, payload BatchMarkPayload, markedBy, termID string) (*BatchMarkResult, error)
	updateRecordFn                        func(ctx context.Context, id, tenantID string, payload UpdateRecordPayload) (*AttendanceRecord, error)
	getRecordByIDFn                       func(ctx context.Context, id, tenantID string) (*AttendanceRecord, error)
	listRecordsBySlotDateFn               func(ctx context.Context, tenantID, schoolID, timetableSlotID, date string) ([]RecordWithEnrichedData, error)
	listRecordsByStudentTermFn            func(ctx context.Context, tenantID, schoolID, studentID, termID string) ([]RecordWithEnrichedData, error)
	listRecordsByClassDateFn              func(ctx context.Context, tenantID, schoolID, classID, date string) ([]RecordWithEnrichedData, error)
	listRecordsFn                         func(ctx context.Context, filter RecordFilter) ([]RecordWithEnrichedData, error)
	getTermIDByDateFn                     func(ctx context.Context, tenantID, schoolID, date string) (string, error)
	getStudentTermSummaryFn               func(ctx context.Context, tenantID, schoolID, studentID, termID string) ([]AttendanceTermSummary, error)
	getClassTermSummaryFn                 func(ctx context.Context, tenantID, schoolID, classID, termID string) ([]AttendanceTermSummary, error)
	refreshSummariesFn                    func(ctx context.Context, tenantID, schoolID, termID string) error
	getClassDailySummaryFn                func(ctx context.Context, tenantID, schoolID, classID, date string) (*ClassDailyAttendanceSummary, error)
	refreshClassDailySummaryFn            func(ctx context.Context, tenantID, schoolID, classID, date string) error
	listClassDailySummariesFn             func(ctx context.Context, tenantID, schoolID, classID, startDate, endDate string) ([]ClassDailyAttendanceSummary, error)
	listCalendarStatusFn                  func(ctx context.Context, tenantID, schoolID, startDate, endDate string) ([]CalendarDayStatusRaw, error)
	getClassLearningAreaTermSummaryFn     func(ctx context.Context, tenantID, schoolID, classID, learningAreaID, termID string) (*ClassLearningAreaTermSummary, error)
	listClassLearningAreaTermSummariesFn  func(ctx context.Context, tenantID, schoolID, classID, learningAreaID, termID string) ([]ClassLearningAreaTermSummary, error)
	refreshClassLearningAreaTermSummaryFn func(ctx context.Context, tenantID, schoolID, termID, classID string) error
	getClassTermAttendanceSummaryFn       func(ctx context.Context, tenantID, schoolID, classID, termID string) (*ClassTermAttendanceSummary, error)
	listClassTermAttendanceSummariesFn    func(ctx context.Context, tenantID, schoolID, classID, termID string) ([]ClassTermAttendanceSummary, error)
	refreshClassTermAttendanceSummaryFn   func(ctx context.Context, tenantID, schoolID, termID, classID string) error
	listClassAttendanceBreakdownsFn       func(ctx context.Context, tenantID, schoolID, termID string) ([]ClassAttendanceBreakdownItem, error)
	listLearningAreaBreakdownsFn          func(ctx context.Context, tenantID, schoolID, termID string) ([]LearningAreaAttendanceBreakdownItem, error)
	getDayOfWeekSummariesFn               func(ctx context.Context, tenantID string, classID *string) (DayOfWeekSummariesResponse, error)
	getSchoolAttendanceKPIsFn             func(ctx context.Context, tenantID, schoolID, date, termID string) (*SchoolAttendanceKPI, error)
	listClassTermPercentagesFn            func(ctx context.Context, tenantID, schoolID string) ([]ClassTermPercentageItem, error)
	getLowestAttendanceStudentsFn         func(ctx context.Context, tenantID, schoolID string, limit int) ([]LowestAttendanceStudent, error)
	listAttendanceSummaryFn               func(ctx context.Context, tenantID, schoolID, academicYear string) ([]AttendanceSummaryRow, error)
}

func (m *MockRepository) CreateSession(ctx context.Context, tenantID, schoolID string, payload CreateSessionPayload) (*AttendanceSession, error) {
	if m.createSessionFn != nil {
		return m.createSessionFn(ctx, tenantID, schoolID, payload)
	}
	return &AttendanceSession{ID: "session_001", Status: SessionSubmitted}, nil
}

func (m *MockRepository) GetSessionByID(ctx context.Context, id, tenantID string) (*AttendanceSession, error) {
	if m.getSessionByIDFn != nil {
		return m.getSessionByIDFn(ctx, id, tenantID)
	}
	return &AttendanceSession{ID: id, Status: SessionSubmitted}, nil
}

func (m *MockRepository) GetEnrichedSessionByID(ctx context.Context, id, tenantID string) (*SessionWithEnrichedData, error) {
	if m.getEnrichedSessionByIDFn != nil {
		return m.getEnrichedSessionByIDFn(ctx, id, tenantID)
	}
	return &SessionWithEnrichedData{AttendanceSession: AttendanceSession{ID: id}}, nil
}

func (m *MockRepository) ListSessions(ctx context.Context, filter SessionFilter) ([]SessionWithEnrichedData, error) {
	if m.listSessionsFn != nil {
		return m.listSessionsFn(ctx, filter)
	}
	return []SessionWithEnrichedData{}, nil
}

func (m *MockRepository) UpdateSession(ctx context.Context, id, tenantID string, payload UpdateSessionPayload) (*AttendanceSession, error) {
	if m.updateSessionFn != nil {
		return m.updateSessionFn(ctx, id, tenantID, payload)
	}
	return &AttendanceSession{ID: id, Status: SessionSubmitted}, nil
}

func (m *MockRepository) GetSessionsForClassDate(ctx context.Context, tenantID, schoolID, classID, date string) ([]SessionWithEnrichedData, error) {
	if m.getSessionsForClassDateFn != nil {
		return m.getSessionsForClassDateFn(ctx, tenantID, schoolID, classID, date)
	}
	return []SessionWithEnrichedData{}, nil
}

func (m *MockRepository) BatchMark(ctx context.Context, tenantID, schoolID string, payload BatchMarkPayload, markedBy, termID string) (*BatchMarkResult, error) {
	if m.batchMarkFn != nil {
		return m.batchMarkFn(ctx, tenantID, schoolID, payload, markedBy, termID)
	}
	return &BatchMarkResult{Created: len(payload.Records)}, nil
}

func (m *MockRepository) UpdateRecord(ctx context.Context, id, tenantID string, payload UpdateRecordPayload) (*AttendanceRecord, error) {
	if m.updateRecordFn != nil {
		return m.updateRecordFn(ctx, id, tenantID, payload)
	}
	status := StatusPresent
	return &AttendanceRecord{ID: id, Status: status}, nil
}

func (m *MockRepository) GetRecordByID(ctx context.Context, id, tenantID string) (*AttendanceRecord, error) {
	if m.getRecordByIDFn != nil {
		return m.getRecordByIDFn(ctx, id, tenantID)
	}
	return &AttendanceRecord{ID: id, Status: StatusPresent}, nil
}

func (m *MockRepository) ListRecordsBySlotDate(ctx context.Context, tenantID, schoolID, timetableSlotID, date string) ([]RecordWithEnrichedData, error) {
	if m.listRecordsBySlotDateFn != nil {
		return m.listRecordsBySlotDateFn(ctx, tenantID, schoolID, timetableSlotID, date)
	}
	return []RecordWithEnrichedData{}, nil
}

func (m *MockRepository) ListRecordsByStudentTerm(ctx context.Context, tenantID, schoolID, studentID, termID string) ([]RecordWithEnrichedData, error) {
	if m.listRecordsByStudentTermFn != nil {
		return m.listRecordsByStudentTermFn(ctx, tenantID, schoolID, studentID, termID)
	}
	return []RecordWithEnrichedData{}, nil
}

func (m *MockRepository) ListRecordsByClassDate(ctx context.Context, tenantID, schoolID, classID, date string) ([]RecordWithEnrichedData, error) {
	if m.listRecordsByClassDateFn != nil {
		return m.listRecordsByClassDateFn(ctx, tenantID, schoolID, classID, date)
	}
	return []RecordWithEnrichedData{}, nil
}

func (m *MockRepository) ListRecords(ctx context.Context, filter RecordFilter) ([]RecordWithEnrichedData, error) {
	if m.listRecordsFn != nil {
		return m.listRecordsFn(ctx, filter)
	}
	return []RecordWithEnrichedData{}, nil
}

func (m *MockRepository) GetStudentTermSummary(ctx context.Context, tenantID, schoolID, studentID, termID string) ([]AttendanceTermSummary, error) {
	if m.getStudentTermSummaryFn != nil {
		return m.getStudentTermSummaryFn(ctx, tenantID, schoolID, studentID, termID)
	}
	return []AttendanceTermSummary{}, nil
}

func (m *MockRepository) GetClassTermSummary(ctx context.Context, tenantID, schoolID, classID, termID string) ([]AttendanceTermSummary, error) {
	if m.getClassTermSummaryFn != nil {
		return m.getClassTermSummaryFn(ctx, tenantID, schoolID, classID, termID)
	}
	return []AttendanceTermSummary{}, nil
}

func (m *MockRepository) GetTermIDByDate(ctx context.Context, tenantID, schoolID, date string) (string, error) {
	if m.getTermIDByDateFn != nil {
		return m.getTermIDByDateFn(ctx, tenantID, schoolID, date)
	}
	return "term_001", nil
}

func (m *MockRepository) RefreshSummaries(ctx context.Context, tenantID, schoolID, termID string) error {
	if m.refreshSummariesFn != nil {
		return m.refreshSummariesFn(ctx, tenantID, schoolID, termID)
	}
	return nil
}

func (m *MockRepository) GetClassDailySummary(ctx context.Context, tenantID, schoolID, classID, date string) (*ClassDailyAttendanceSummary, error) {
	if m.getClassDailySummaryFn != nil {
		return m.getClassDailySummaryFn(ctx, tenantID, schoolID, classID, date)
	}
	return &ClassDailyAttendanceSummary{ClassID: classID, Date: date}, nil
}

func (m *MockRepository) RefreshClassDailySummary(ctx context.Context, tenantID, schoolID, classID, date string) error {
	if m.refreshClassDailySummaryFn != nil {
		return m.refreshClassDailySummaryFn(ctx, tenantID, schoolID, classID, date)
	}
	return nil
}

func (m *MockRepository) ListClassDailySummaries(ctx context.Context, tenantID, schoolID, classID, startDate, endDate string) ([]ClassDailyAttendanceSummary, error) {
	if m.listClassDailySummariesFn != nil {
		return m.listClassDailySummariesFn(ctx, tenantID, schoolID, classID, startDate, endDate)
	}
	return []ClassDailyAttendanceSummary{}, nil
}

func (m *MockRepository) ListCalendarStatus(ctx context.Context, tenantID, schoolID, startDate, endDate string) ([]CalendarDayStatusRaw, error) {
	if m.listCalendarStatusFn != nil {
		return m.listCalendarStatusFn(ctx, tenantID, schoolID, startDate, endDate)
	}
	return []CalendarDayStatusRaw{}, nil
}

func (m *MockRepository) GetClassLearningAreaTermSummary(ctx context.Context, tenantID, schoolID, classID, learningAreaID, termID string) (*ClassLearningAreaTermSummary, error) {
	if m.getClassLearningAreaTermSummaryFn != nil {
		return m.getClassLearningAreaTermSummaryFn(ctx, tenantID, schoolID, classID, learningAreaID, termID)
	}
	return &ClassLearningAreaTermSummary{}, nil
}

func (m *MockRepository) ListClassLearningAreaTermSummaries(ctx context.Context, tenantID, schoolID, classID, learningAreaID, termID string) ([]ClassLearningAreaTermSummary, error) {
	if m.listClassLearningAreaTermSummariesFn != nil {
		return m.listClassLearningAreaTermSummariesFn(ctx, tenantID, schoolID, classID, learningAreaID, termID)
	}
	return []ClassLearningAreaTermSummary{}, nil
}

func (m *MockRepository) RefreshClassLearningAreaTermSummary(ctx context.Context, tenantID, schoolID, termID, classID string) error {
	if m.refreshClassLearningAreaTermSummaryFn != nil {
		return m.refreshClassLearningAreaTermSummaryFn(ctx, tenantID, schoolID, termID, classID)
	}
	return nil
}

func (m *MockRepository) GetClassTermAttendanceSummary(ctx context.Context, tenantID, schoolID, classID, termID string) (*ClassTermAttendanceSummary, error) {
	if m.getClassTermAttendanceSummaryFn != nil {
		return m.getClassTermAttendanceSummaryFn(ctx, tenantID, schoolID, classID, termID)
	}
	return &ClassTermAttendanceSummary{}, nil
}

func (m *MockRepository) ListClassTermAttendanceSummaries(ctx context.Context, tenantID, schoolID, classID, termID string) ([]ClassTermAttendanceSummary, error) {
	if m.listClassTermAttendanceSummariesFn != nil {
		return m.listClassTermAttendanceSummariesFn(ctx, tenantID, schoolID, classID, termID)
	}
	return []ClassTermAttendanceSummary{}, nil
}

func (m *MockRepository) RefreshClassTermAttendanceSummary(ctx context.Context, tenantID, schoolID, termID, classID string) error {
	if m.refreshClassTermAttendanceSummaryFn != nil {
		return m.refreshClassTermAttendanceSummaryFn(ctx, tenantID, schoolID, termID, classID)
	}
	return nil
}

func (m *MockRepository) ListClassAttendanceBreakdowns(ctx context.Context, tenantID, schoolID, termID string) ([]ClassAttendanceBreakdownItem, error) {
	if m.listClassAttendanceBreakdownsFn != nil {
		return m.listClassAttendanceBreakdownsFn(ctx, tenantID, schoolID, termID)
	}
	return []ClassAttendanceBreakdownItem{}, nil
}

func (m *MockRepository) ListLearningAreaBreakdowns(ctx context.Context, tenantID, schoolID, termID string) ([]LearningAreaAttendanceBreakdownItem, error) {
	if m.listLearningAreaBreakdownsFn != nil {
		return m.listLearningAreaBreakdownsFn(ctx, tenantID, schoolID, termID)
	}
	return []LearningAreaAttendanceBreakdownItem{}, nil
}

func (m *MockRepository) GetDayOfWeekSummaries(ctx context.Context, tenantID string, classID *string) (DayOfWeekSummariesResponse, error) {
	if m.getDayOfWeekSummariesFn != nil {
		return m.getDayOfWeekSummariesFn(ctx, tenantID, classID)
	}
	return DayOfWeekSummariesResponse{}, nil
}

func (m *MockRepository) GetSchoolAttendanceKPIs(ctx context.Context, tenantID, schoolID, date, termID string) (*SchoolAttendanceKPI, error) {
	if m.getSchoolAttendanceKPIsFn != nil {
		return m.getSchoolAttendanceKPIsFn(ctx, tenantID, schoolID, date, termID)
	}
	return &SchoolAttendanceKPI{}, nil
}

// ListClassTermPercentages returns the percentage of attendance statuses (present, absent,
// excused, late) for each class and term in the current academic year for a school, with a rollup row
// for "All" classes.
func (m *MockRepository) ListClassTermPercentages(ctx context.Context, tenantID, schoolID string) ([]ClassTermPercentageItem, error) {
	if m.listClassTermPercentagesFn != nil {
		return m.listClassTermPercentagesFn(ctx, tenantID, schoolID)
	}
	return []ClassTermPercentageItem{}, nil
}

func (m *MockRepository) GetLowestAttendanceStudents(ctx context.Context, tenantID, schoolID string, limit int) ([]LowestAttendanceStudent, error) {
	if m.getLowestAttendanceStudentsFn != nil {
		return m.getLowestAttendanceStudentsFn(ctx, tenantID, schoolID, limit)
	}
	return []LowestAttendanceStudent{}, nil
}

func (m *MockRepository) ListAttendanceSummary(ctx context.Context, tenantID, schoolID, academicYear string) ([]AttendanceSummaryRow, error) {
	if m.listAttendanceSummaryFn != nil {
		return m.listAttendanceSummaryFn(ctx, tenantID, schoolID, academicYear)
	}
	return []AttendanceSummaryRow{}, nil
}

// ============================================================================
// Test Harness
// ============================================================================

type testHarness struct {
	svc  *Service
	repo *MockRepository
}

func newTestHarness() *testHarness {
	repo := &MockRepository{}
	svc := NewService(repo)
	return &testHarness{
		svc:  svc,
		repo: repo,
	}
}

// ============================================================================
// Tests: Sessions — CreateSession
// ============================================================================

// ============================================================================
// Tests: Records — GetRecord
// ============================================================================

// ── Class Attendance Breakdown Unit Tests ────────────────────────────────

func TestListClassAttendanceBreakdowns_HappyPath(t *testing.T) {
	h := newTestHarness()

	expected := []ClassAttendanceBreakdownItem{
		{
			ClassID:      "class_002",
			ClassName:    "G1 Green",
			PresentCount: 20,
			LateCount:    3,
			AbsentCount:  5,
		},
		{
			ClassID:      "class_001",
			ClassName:    "G1 Blue",
			PresentCount: 25,
			LateCount:    2,
			AbsentCount:  3,
		},
	}
	h.repo.listClassAttendanceBreakdownsFn = func(ctx context.Context, tenantID, schoolID, termID string) ([]ClassAttendanceBreakdownItem, error) {
		return expected, nil
	}

	resp, err := h.svc.ListClassAttendanceBreakdowns(context.Background(), "tenant_001", "school_001", "term_001")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp == nil {
		t.Fatal("expected response, got nil")
	}
	if resp.Total != len(expected) {
		t.Fatalf("expected total %d, got %d", len(expected), resp.Total)
	}
	if len(resp.Items) != 2 || resp.Items[0].ClassName != "G1 Green" {
		t.Fatalf("expected repo result passthrough, got %+v", resp.Items)
	}
}

func TestListClassAttendanceBreakdowns_EmptyTermID(t *testing.T) {
	h := newTestHarness()

	_, err := h.svc.ListClassAttendanceBreakdowns(context.Background(), "tenant_001", "school_001", "")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got %v", err)
	}
}

func TestListClassAttendanceBreakdowns_RepoErrorWrapped(t *testing.T) {
	h := newTestHarness()

	repoErr := errors.New("db down")
	h.repo.listClassAttendanceBreakdownsFn = func(ctx context.Context, tenantID, schoolID, termID string) ([]ClassAttendanceBreakdownItem, error) {
		return nil, repoErr
	}

	_, err := h.svc.ListClassAttendanceBreakdowns(context.Background(), "tenant_001", "school_001", "term_001")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, repoErr) {
		t.Fatalf("expected repo error to be wrapped and preserved, got %v", err)
	}
}

func TestListClassAttendanceBreakdowns_NilToEmptySlice(t *testing.T) {
	h := newTestHarness()

	h.repo.listClassAttendanceBreakdownsFn = func(ctx context.Context, tenantID, schoolID, termID string) ([]ClassAttendanceBreakdownItem, error) {
		return nil, nil
	}

	resp, err := h.svc.ListClassAttendanceBreakdowns(context.Background(), "tenant_001", "school_001", "term_001")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Total != 0 || resp.Items == nil || len(resp.Items) != 0 {
		t.Fatalf("expected empty non-nil items, got %+v", resp)
	}
}

// ============================================================================
// Tests: ListLearningAreaBreakdowns
// ============================================================================

func TestListLearningAreaBreakdowns_HappyPath(t *testing.T) {
	h := newTestHarness()

	expected := []LearningAreaAttendanceBreakdownItem{
		{
			LearningAreaID:       "la_002",
			LearningAreaName:     "English",
			PeriodsTotal:         120,
			PeriodsPresent:       90,
			PeriodsAbsent:        25,
			PeriodsExcused:       5,
			AttendancePercentage: 75.00,
		},
		{
			LearningAreaID:       "la_001",
			LearningAreaName:     "Mathematics",
			PeriodsTotal:         180,
			PeriodsPresent:       160,
			PeriodsAbsent:        12,
			PeriodsExcused:       8,
			AttendancePercentage: 88.89,
		},
	}
	h.repo.listLearningAreaBreakdownsFn = func(ctx context.Context, tenantID, schoolID, termID string) ([]LearningAreaAttendanceBreakdownItem, error) {
		if tenantID != "tenant_001" {
			t.Errorf("expected tenant_001, got %q", tenantID)
		}
		if schoolID != "school_001" {
			t.Errorf("expected school_001, got %q", schoolID)
		}
		if termID != "term_001" {
			t.Errorf("expected term_001, got %q", termID)
		}
		return expected, nil
	}

	resp, err := h.svc.ListLearningAreaBreakdowns(context.Background(), "tenant_001", "school_001", "term_001")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp == nil {
		t.Fatal("expected response, got nil")
	}
	if resp.Total != len(expected) {
		t.Fatalf("expected total %d, got %d", len(expected), resp.Total)
	}
	if len(resp.Items) != 2 || resp.Items[0].LearningAreaName != "English" {
		t.Fatalf("expected repo result passthrough, got %+v", resp.Items)
	}
	if resp.Items[0].PeriodsAbsent != 25 {
		t.Fatalf("expected 25 absent periods, got %d", resp.Items[0].PeriodsAbsent)
	}
}

func TestListLearningAreaBreakdowns_EmptyTermID(t *testing.T) {
	h := newTestHarness()

	_, err := h.svc.ListLearningAreaBreakdowns(context.Background(), "tenant_001", "school_001", "")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got %v", err)
	}
}

func TestListLearningAreaBreakdowns_RepoErrorWrapped(t *testing.T) {
	h := newTestHarness()

	repoErr := errors.New("db down")
	h.repo.listLearningAreaBreakdownsFn = func(ctx context.Context, tenantID, schoolID, termID string) ([]LearningAreaAttendanceBreakdownItem, error) {
		return nil, repoErr
	}

	_, err := h.svc.ListLearningAreaBreakdowns(context.Background(), "tenant_001", "school_001", "term_001")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, repoErr) {
		t.Fatalf("expected repo error to be wrapped and preserved, got %v", err)
	}
}

func TestListLearningAreaBreakdowns_NilToEmptySlice(t *testing.T) {
	h := newTestHarness()

	h.repo.listLearningAreaBreakdownsFn = func(ctx context.Context, tenantID, schoolID, termID string) ([]LearningAreaAttendanceBreakdownItem, error) {
		return nil, nil
	}

	resp, err := h.svc.ListLearningAreaBreakdowns(context.Background(), "tenant_001", "school_001", "term_001")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Total != 0 || resp.Items == nil || len(resp.Items) != 0 {
		t.Fatalf("expected empty non-nil items, got %+v", resp)
	}
}

// Tests: GetDayOfWeekSummaries

func TestGetDayOfWeekSummaries_HappyPath(t *testing.T) {
	h := newTestHarness()

	classID := "class_001"
	expected := DayOfWeekSummariesResponse{
		AcademicYear: "2025",
		ClassName:    "Grade 9 East",
		Data: []DayOfWeekSummaryItem{
			{DayOfWeekNumber: 1, DayName: "Monday", AbsentCount: 12, LateCount: 5, ExcusedCount: 2},
			{DayOfWeekNumber: 5, DayName: "Friday", AbsentCount: 18, LateCount: 3, ExcusedCount: 1},
		},
	}
	h.repo.getDayOfWeekSummariesFn = func(ctx context.Context, tenantID string, cid *string) (DayOfWeekSummariesResponse, error) {
		if tenantID != "tenant_001" {
			t.Fatalf("expected tenant_001, got %s", tenantID)
		}
		if cid == nil || *cid != classID {
			t.Fatalf("expected class filter %q, got %v", classID, cid)
		}
		return expected, nil
	}

	resp, err := h.svc.GetDayOfWeekSummaries(context.Background(), "tenant_001", &classID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp == nil {
		t.Fatal("expected response, got nil")
	}
	if resp.AcademicYear != expected.AcademicYear || resp.ClassName != expected.ClassName {
		t.Fatalf("expected passthrough of %+v, got %+v", expected, resp)
	}
	if len(resp.Data) != 2 || resp.Data[0].DayName != "Monday" {
		t.Fatalf("expected repo data passthrough, got %+v", resp.Data)
	}
}

func TestGetDayOfWeekSummaries_RepoErrorWrapped(t *testing.T) {
	h := newTestHarness()

	repoErr := errors.New("db down")
	h.repo.getDayOfWeekSummariesFn = func(ctx context.Context, tenantID string, cid *string) (DayOfWeekSummariesResponse, error) {
		return DayOfWeekSummariesResponse{}, repoErr
	}

	_, err := h.svc.GetDayOfWeekSummaries(context.Background(), "tenant_001", nil)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, repoErr) {
		t.Fatalf("expected repo error to be wrapped and preserved, got %v", err)
	}
}

func TestGetDayOfWeekSummaries_NilToEmptySlice(t *testing.T) {
	h := newTestHarness()

	h.repo.getDayOfWeekSummariesFn = func(ctx context.Context, tenantID string, cid *string) (DayOfWeekSummariesResponse, error) {
		return DayOfWeekSummariesResponse{}, nil
	}

	resp, err := h.svc.GetDayOfWeekSummaries(context.Background(), "tenant_001", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp == nil || resp.Data == nil || len(resp.Data) != 0 {
		t.Fatalf("expected empty non-nil data, got %+v", resp)
	}
}

// ── Calendar Status Unit Tests ───────────────────────────────────────────

func TestComputeDayStatus_ExpectedZero(t *testing.T) {
	// expected_count == 0 → "none" (weekend, non-school day)
	status := ComputeDayStatus(0, 0)
	if status != DayStatusNone {
		t.Fatalf("expected 'none', got %q", status)
	}
}

func TestComputeDayStatus_FullyMarked(t *testing.T) {
	// handled_count == expected_count (and expected > 0) → "green"
	status := ComputeDayStatus(6, 6)
	if status != DayStatusGreen {
		t.Fatalf("expected 'green', got %q", status)
	}
}

func TestComputeDayStatus_NoneMarked(t *testing.T) {
	// handled_count == 0 (and expected > 0) → "red"
	status := ComputeDayStatus(6, 0)
	if status != DayStatusRed {
		t.Fatalf("expected 'red', got %q", status)
	}
}

func TestComputeDayStatus_PartiallyMarked(t *testing.T) {
	// 0 < handled_count < expected_count → "yellow"
	status := ComputeDayStatus(6, 3)
	if status != DayStatusYellow {
		t.Fatalf("expected 'yellow', got %q", status)
	}
}

func TestGetCalendarStatus_Success(t *testing.T) {
	h := newTestHarness()

	h.repo.listCalendarStatusFn = func(ctx context.Context, tenantID, schoolID, startDate, endDate string) ([]CalendarDayStatusRaw, error) {
		return []CalendarDayStatusRaw{
			{Date: "2026-06-01", ExpectedCount: 6, HandledCount: 6}, // green
			{Date: "2026-06-02", ExpectedCount: 6, HandledCount: 3}, // yellow
			{Date: "2026-06-03", ExpectedCount: 6, HandledCount: 0}, // red
			{Date: "2026-06-04", ExpectedCount: 0, HandledCount: 0}, // none (weekend)
		}, nil
	}

	result, err := h.svc.GetCalendarStatus(context.Background(), "tenant_001", "school_001", "2026-06-01", "2026-06-04")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Total != 4 {
		t.Fatalf("expected 4 items, got %d", result.Total)
	}

	expected := []DayStatus{DayStatusGreen, DayStatusYellow, DayStatusRed, DayStatusNone}
	for i, item := range result.Items {
		if item.Status != expected[i] {
			t.Fatalf("item %d: expected status %q, got %q", i, expected[i], item.Status)
		}
	}
}

func TestGetCalendarStatus_Validation(t *testing.T) {
	h := newTestHarness()

	// Empty start_date
	_, err := h.svc.GetCalendarStatus(context.Background(), "t", "s", "", "2026-06-30")
	if err == nil {
		t.Fatal("expected error for empty start_date")
	}

	// Empty end_date
	_, err = h.svc.GetCalendarStatus(context.Background(), "t", "s", "2026-06-01", "")
	if err == nil {
		t.Fatal("expected error for empty end_date")
	}
}

func TestGetCalendarStatus_NilToEmptySlice(t *testing.T) {
	h := newTestHarness()

	h.repo.listCalendarStatusFn = func(ctx context.Context, tenantID, schoolID, startDate, endDate string) ([]CalendarDayStatusRaw, error) {
		return nil, nil
	}

	result, err := h.svc.GetCalendarStatus(context.Background(), "t", "s", "2026-06-01", "2026-06-30")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Items == nil {
		t.Fatal("expected non-nil empty Items slice")
	}
	if result.Total != 0 {
		t.Fatalf("expected 0 total, got %d", result.Total)
	}
}

// ============================================================================
// ListAttendanceSummary — service unit (mock repo)
// ============================================================================
func TestListAttendanceSummary_ServiceUnit(t *testing.T) {
	h := newTestHarness()

	h.repo.listAttendanceSummaryFn = func(ctx context.Context, tenantID, schoolID, academicYear string) ([]AttendanceSummaryRow, error) {
		return []AttendanceSummaryRow{
			{ClassName: "All", TermName: "Term 1", TermNumber: 1, AcademicYear: "2026", PresentPercentage: 92.0},
		}, nil
	}

	result, err := h.svc.ListAttendanceSummary(context.Background(), "t", "s", "2026")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if result.AcademicYear != "2026" {
		t.Errorf("expected academic year 2026, got %s", result.AcademicYear)
	}
	if len(result.Data) != 1 {
		t.Errorf("expected 1 data row, got %d", len(result.Data))
	}
}
