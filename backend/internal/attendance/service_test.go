package attendance

import (
	"context"
	"errors"
	"testing"
	"time"
)

// ============================================================================
// MockRepository
// ============================================================================

type MockRepository struct {
	getRosterForSlotFn          func(ctx context.Context, tenantID, schoolID, timetableSlotID, date string) (*SlotRosterResponse, error)
	bulkUpsertFn                func(ctx context.Context, tenantID, schoolID string, payload BulkAttendancePayload, markedBy string) (string, string, error)
	getStudentHistoryFn         func(ctx context.Context, tenantID, schoolID, studentID string, filter StudentHistoryFilter) ([]AttendanceRecord, error)
	getRecordByIDFn             func(ctx context.Context, id, tenantID string) (*AttendanceRecord, error)
	updateRecordFn              func(ctx context.Context, id, tenantID string, payload UpdateAttendanceEntryPayload) error
	getAdminDashboardFn         func(ctx context.Context, tenantID, schoolID, date string, filter DashboardFilter) (*AdminDashboardResponse, error)
	getChildAttendanceSummaryFn func(ctx context.Context, tenantID, schoolID, studentID, termID string) (*ChildAttendanceSummary, error)
	computeTermSummariesFn      func(ctx context.Context, tenantID, schoolID, termID string) (int, error)
	computeClassSummariesFn     func(ctx context.Context, tenantID, schoolID, termID, classID string) (int, error)
	getRecordsBySlotDateFn      func(ctx context.Context, timetableSlotID, date string) ([]AttendanceRecord, error)
}

func (m *MockRepository) GetRosterForSlot(ctx context.Context, tenantID, schoolID, timetableSlotID, date string) (*SlotRosterResponse, error) {
	if m.getRosterForSlotFn != nil {
		return m.getRosterForSlotFn(ctx, tenantID, schoolID, timetableSlotID, date)
	}
	return &SlotRosterResponse{TimetableSlotID: timetableSlotID, Date: date}, nil
}

func (m *MockRepository) BulkUpsert(ctx context.Context, tenantID, schoolID string, payload BulkAttendancePayload, markedBy string) (string, string, error) {
	if m.bulkUpsertFn != nil {
		return m.bulkUpsertFn(ctx, tenantID, schoolID, payload, markedBy)
	}
	return "class_001", "term_001", nil
}

func (m *MockRepository) GetStudentHistory(ctx context.Context, tenantID, schoolID, studentID string, filter StudentHistoryFilter) ([]AttendanceRecord, error) {
	if m.getStudentHistoryFn != nil {
		return m.getStudentHistoryFn(ctx, tenantID, schoolID, studentID, filter)
	}
	return []AttendanceRecord{}, nil
}

func (m *MockRepository) GetRecordsBySlotDate(ctx context.Context, timetableSlotID, date string) ([]AttendanceRecord, error) {
	if m.getRecordsBySlotDateFn != nil {
		return m.getRecordsBySlotDateFn(ctx, timetableSlotID, date)
	}
	return []AttendanceRecord{}, nil
}

func (m *MockRepository) GetRecordByID(ctx context.Context, id, tenantID string) (*AttendanceRecord, error) {
	if m.getRecordByIDFn != nil {
		return m.getRecordByIDFn(ctx, id, tenantID)
	}
	return &AttendanceRecord{ID: id, TenantID: tenantID, Date: time.Now().Format("2006-01-02")}, nil
}

func (m *MockRepository) UpdateRecord(ctx context.Context, id, tenantID string, payload UpdateAttendanceEntryPayload) error {
	if m.updateRecordFn != nil {
		return m.updateRecordFn(ctx, id, tenantID, payload)
	}
	return nil
}

func (m *MockRepository) GetAdminDashboard(ctx context.Context, tenantID, schoolID, date string, filter DashboardFilter) (*AdminDashboardResponse, error) {
	if m.getAdminDashboardFn != nil {
		return m.getAdminDashboardFn(ctx, tenantID, schoolID, date, filter)
	}
	return &AdminDashboardResponse{Date: date, Items: []CompletionStatus{}, Total: 0}, nil
}

func (m *MockRepository) GetChildAttendanceSummary(ctx context.Context, tenantID, schoolID, studentID, termID string) (*ChildAttendanceSummary, error) {
	if m.getChildAttendanceSummaryFn != nil {
		return m.getChildAttendanceSummaryFn(ctx, tenantID, schoolID, studentID, termID)
	}
	return &ChildAttendanceSummary{StudentID: studentID, TermID: termID}, nil
}

func (m *MockRepository) ComputeTermSummaries(ctx context.Context, tenantID, schoolID, termID string) (int, error) {
	if m.computeTermSummariesFn != nil {
		return m.computeTermSummariesFn(ctx, tenantID, schoolID, termID)
	}
	return 0, nil
}

func (m *MockRepository) ComputeClassSummaries(ctx context.Context, tenantID, schoolID, termID, classID string) (int, error) {
	if m.computeClassSummariesFn != nil {
		return m.computeClassSummariesFn(ctx, tenantID, schoolID, termID, classID)
	}
	return 0, nil
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
	// Create service with nil embedded to avoid needing real Redis connection.
	// Only methods that don't call enqueueClassRecompute can be tested this way.
	svc := &Service{
		repo:  repo,
		redis: nil,
	}
	return &testHarness{
		svc:  svc,
		repo: repo,
	}
}

// ============================================================================
// Tests: GetRosterForSlot
// ============================================================================

func TestGetRosterForSlot_HappyPath(t *testing.T) {
	h := newTestHarness()

	expectedRoster := &SlotRosterResponse{
		TimetableSlotID: "slot_001",
		Date:            "2026-07-15",
		ClassName:       "G4 Blue",
		LearningArea:    "Mathematics",
		Students: []RosterStudent{
			{StudentID: "stu_001", FullName: "John Doe", AdmissionNumber: "ADM001"},
			{StudentID: "stu_002", FullName: "Jane Smith", AdmissionNumber: "ADM002"},
		},
	}

	h.repo.getRosterForSlotFn = func(ctx context.Context, tenantID, schoolID, timetableSlotID, date string) (*SlotRosterResponse, error) {
		if timetableSlotID != "slot_001" {
			t.Errorf("expected timetableSlotID 'slot_001', got %q", timetableSlotID)
		}
		if date != "2026-07-15" {
			t.Errorf("expected date '2026-07-15', got %q", date)
		}
		return expectedRoster, nil
	}

	roster, err := h.svc.GetRosterForSlot(context.Background(), "tenant_001", "school_001", "slot_001", "2026-07-15")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if roster.TimetableSlotID != "slot_001" {
		t.Fatalf("expected slot ID 'slot_001', got %q", roster.TimetableSlotID)
	}
	if len(roster.Students) != 2 {
		t.Fatalf("expected 2 students, got %d", len(roster.Students))
	}
	if roster.Students[0].FullName != "John Doe" {
		t.Fatalf("expected first student 'John Doe', got %q", roster.Students[0].FullName)
	}
}

func TestGetRosterForSlot_EmptySlotID(t *testing.T) {
	h := newTestHarness()

	_, err := h.svc.GetRosterForSlot(context.Background(), "tenant_001", "school_001", "", "2026-07-15")
	if err == nil {
		t.Fatal("expected error for empty slotID, got nil")
	}
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got %v", err)
	}
}

func TestGetRosterForSlot_EmptyDate(t *testing.T) {
	h := newTestHarness()

	_, err := h.svc.GetRosterForSlot(context.Background(), "tenant_001", "school_001", "slot_001", "")
	if err == nil {
		t.Fatal("expected error for empty date, got nil")
	}
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got %v", err)
	}
}

func TestGetRosterForSlot_NotFound(t *testing.T) {
	h := newTestHarness()

	h.repo.getRosterForSlotFn = func(ctx context.Context, tenantID, schoolID, timetableSlotID, date string) (*SlotRosterResponse, error) {
		return nil, ErrNotFound
	}

	_, err := h.svc.GetRosterForSlot(context.Background(), "tenant_001", "school_001", "slot_999", "2026-07-15")
	if err == nil {
		t.Fatal("expected error for non-existent slot, got nil")
	}
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

// ============================================================================
// Tests: GetStudentHistory
// ============================================================================

func TestGetStudentHistory_HappyPath(t *testing.T) {
	h := newTestHarness()

	expectedHistory := []AttendanceRecord{
		{ID: "rec_001", StudentID: "stu_001", Date: "2026-07-15", Status: StatusPresent},
		{ID: "rec_002", StudentID: "stu_001", Date: "2026-07-14", Status: StatusAbsent},
	}

	h.repo.getStudentHistoryFn = func(ctx context.Context, tenantID, schoolID, studentID string, filter StudentHistoryFilter) ([]AttendanceRecord, error) {
		if studentID != "stu_001" {
			t.Errorf("expected studentID 'stu_001', got %q", studentID)
		}
		return expectedHistory, nil
	}

	records, err := h.svc.GetStudentHistory(context.Background(), "tenant_001", "school_001", "stu_001", StudentHistoryFilter{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(records) != 2 {
		t.Fatalf("expected 2 records, got %d", len(records))
	}
	if records[0].Status != StatusPresent {
		t.Fatalf("expected first status PRESENT, got %q", records[0].Status)
	}
}

func TestGetStudentHistory_EmptyStudentID(t *testing.T) {
	h := newTestHarness()

	_, err := h.svc.GetStudentHistory(context.Background(), "tenant_001", "school_001", "", StudentHistoryFilter{})
	if err == nil {
		t.Fatal("expected error for empty studentID, got nil")
	}
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got %v", err)
	}
}

func TestGetStudentHistory_WithDateFilter(t *testing.T) {
	h := newTestHarness()

	h.repo.getStudentHistoryFn = func(ctx context.Context, tenantID, schoolID, studentID string, filter StudentHistoryFilter) ([]AttendanceRecord, error) {
		if filter.StartDate != "2026-07-01" {
			t.Errorf("expected StartDate '2026-07-01', got %q", filter.StartDate)
		}
		if filter.EndDate != "2026-07-15" {
			t.Errorf("expected EndDate '2026-07-15', got %q", filter.EndDate)
		}
		return []AttendanceRecord{
			{ID: "rec_001", Date: "2026-07-10", Status: StatusPresent},
		}, nil
	}

	records, err := h.svc.GetStudentHistory(context.Background(), "tenant_001", "school_001", "stu_001", StudentHistoryFilter{
		StartDate: "2026-07-01",
		EndDate:   "2026-07-15",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(records) != 1 {
		t.Fatalf("expected 1 record, got %d", len(records))
	}
}

// ============================================================================
// Tests: UpdateAttendanceRecord
// ============================================================================

func TestUpdateAttendanceRecord_HappyPath(t *testing.T) {
	h := newTestHarness()

	today := time.Now().Format("2006-01-02")

	// Mock GetRecordByID to return today's date so the edit is allowed
	h.repo.getRecordByIDFn = func(ctx context.Context, id, tenantID string) (*AttendanceRecord, error) {
		return &AttendanceRecord{ID: id, TenantID: tenantID, Date: today}, nil
	}

	called := false
	h.repo.updateRecordFn = func(ctx context.Context, id, tenantID string, payload UpdateAttendanceEntryPayload) error {
		called = true
		if payload.Status != StatusLate {
			t.Errorf("expected Status 'LATE', got %q", payload.Status)
		}
		return nil
	}

	note := "Traffic delay"
	err := h.svc.UpdateAttendanceRecord(context.Background(), "rec_001", "tenant_001", UpdateAttendanceEntryPayload{
		Status: StatusLate,
		Note:   &note,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !called {
		t.Fatal("expected updateRecordFn to be called")
	}
}

func TestUpdateAttendanceRecord_EmptyID(t *testing.T) {
	h := newTestHarness()

	err := h.svc.UpdateAttendanceRecord(context.Background(), "", "tenant_001", UpdateAttendanceEntryPayload{
		Status: StatusPresent,
	})
	if err == nil {
		t.Fatal("expected error for empty id, got nil")
	}
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got %v", err)
	}
}

func TestUpdateAttendanceRecord_EmptyStatus(t *testing.T) {
	h := newTestHarness()

	err := h.svc.UpdateAttendanceRecord(context.Background(), "rec_001", "tenant_001", UpdateAttendanceEntryPayload{})
	if err == nil {
		t.Fatal("expected error for empty status, got nil")
	}
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got %v", err)
	}
}

func TestUpdateAttendanceRecord_PastDate(t *testing.T) {
	h := newTestHarness()

	// Return a record with yesterday's date
	yesterday := time.Now().AddDate(0, 0, -1).Format("2006-01-02")
	h.repo.getRecordByIDFn = func(ctx context.Context, id, tenantID string) (*AttendanceRecord, error) {
		return &AttendanceRecord{ID: id, TenantID: tenantID, Date: yesterday}, nil
	}

	err := h.svc.UpdateAttendanceRecord(context.Background(), "rec_001", "tenant_001", UpdateAttendanceEntryPayload{
		Status: StatusPresent,
	})
	if err == nil {
		t.Fatal("expected error for past date record, got nil")
	}
	if !errors.Is(err, ErrForbidden) {
		t.Fatalf("expected ErrForbidden, got %v", err)
	}
}

func TestUpdateAttendanceRecord_RecordNotFound(t *testing.T) {
	h := newTestHarness()

	h.repo.getRecordByIDFn = func(ctx context.Context, id, tenantID string) (*AttendanceRecord, error) {
		return nil, ErrNotFound
	}

	err := h.svc.UpdateAttendanceRecord(context.Background(), "rec_999", "tenant_001", UpdateAttendanceEntryPayload{
		Status: StatusPresent,
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

// ============================================================================
// Tests: GetAdminDashboard
// ============================================================================

func TestGetAdminDashboard_HappyPath(t *testing.T) {
	h := newTestHarness()

	expected := &AdminDashboardResponse{
		Date: "2026-07-15",
		Items: []CompletionStatus{
			{ClassID: "class_001", ClassName: "G4 Blue", TotalSlots: 8, MarkedSlots: 6, IsComplete: false},
			{ClassID: "class_002", ClassName: "G5 Red", TotalSlots: 8, MarkedSlots: 8, IsComplete: true},
		},
		Total: 2,
		Page:  1,
		Limit: 50,
	}

	h.repo.getAdminDashboardFn = func(ctx context.Context, tenantID, schoolID, date string, filter DashboardFilter) (*AdminDashboardResponse, error) {
		if date != "2026-07-15" {
			t.Errorf("expected date '2026-07-15', got %q", date)
		}
		return expected, nil
	}

	result, err := h.svc.GetAdminDashboard(context.Background(), "tenant_001", "school_001", "2026-07-15", DashboardFilter{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Total != 2 {
		t.Fatalf("expected total 2, got %d", result.Total)
	}
	if len(result.Items) != 2 {
		t.Fatalf("expected 2 items, got %d", len(result.Items))
	}
	if !result.Items[1].IsComplete {
		t.Fatal("expected second class to be complete")
	}
}

func TestGetAdminDashboard_DefaultDate(t *testing.T) {
	h := newTestHarness()

	today := time.Now().Format("2006-01-02")

	h.repo.getAdminDashboardFn = func(ctx context.Context, tenantID, schoolID, date string, filter DashboardFilter) (*AdminDashboardResponse, error) {
		if date != today {
			t.Errorf("expected today's date %q, got %q", today, date)
		}
		return &AdminDashboardResponse{Date: date}, nil
	}

	_, err := h.svc.GetAdminDashboard(context.Background(), "tenant_001", "school_001", "", DashboardFilter{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestGetAdminDashboard_EmptySchoolID(t *testing.T) {
	h := newTestHarness()

	_, err := h.svc.GetAdminDashboard(context.Background(), "tenant_001", "", "2026-07-15", DashboardFilter{})
	if err == nil {
		t.Fatal("expected error for empty schoolID, got nil")
	}
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got %v", err)
	}
}

func TestGetAdminDashboard_DefaultPagination(t *testing.T) {
	h := newTestHarness()

	h.repo.getAdminDashboardFn = func(ctx context.Context, tenantID, schoolID, date string, filter DashboardFilter) (*AdminDashboardResponse, error) {
		if filter.Page != 1 {
			t.Errorf("expected default Page 1, got %d", filter.Page)
		}
		if filter.Limit != 50 {
			t.Errorf("expected default Limit 50, got %d", filter.Limit)
		}
		return &AdminDashboardResponse{}, nil
	}

	_, err := h.svc.GetAdminDashboard(context.Background(), "tenant_001", "school_001", "2026-07-15", DashboardFilter{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestGetAdminDashboard_LimitCappedAt100(t *testing.T) {
	h := newTestHarness()

	h.repo.getAdminDashboardFn = func(ctx context.Context, tenantID, schoolID, date string, filter DashboardFilter) (*AdminDashboardResponse, error) {
		if filter.Limit > 100 {
			t.Errorf("expected limit capped at 100, got %d", filter.Limit)
		}
		return &AdminDashboardResponse{}, nil
	}

	_, err := h.svc.GetAdminDashboard(context.Background(), "tenant_001", "school_001", "2026-07-15", DashboardFilter{
		Limit: 200,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// ============================================================================
// Tests: GetChildAttendanceSummary
// ============================================================================

func TestGetChildAttendanceSummary_HappyPath(t *testing.T) {
	h := newTestHarness()

	expected := &ChildAttendanceSummary{
		StudentID:            "stu_001",
		TermID:               "term_001",
		AttendancePercentage: 85.5,
		RecentPeriods: []StudentAttendanceRecord{
			{Date: "2026-07-15", Subject: "Mathematics", Status: StatusPresent},
			{Date: "2026-07-14", Subject: "English", Status: StatusLate},
		},
	}

	h.repo.getChildAttendanceSummaryFn = func(ctx context.Context, tenantID, schoolID, studentID, termID string) (*ChildAttendanceSummary, error) {
		if studentID != "stu_001" {
			t.Errorf("expected studentID 'stu_001', got %q", studentID)
		}
		if termID != "term_001" {
			t.Errorf("expected termID 'term_001', got %q", termID)
		}
		return expected, nil
	}

	summary, err := h.svc.GetChildAttendanceSummary(context.Background(), "tenant_001", "school_001", "stu_001", "term_001")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if summary.AttendancePercentage != 85.5 {
		t.Fatalf("expected 85.5%%, got %f", summary.AttendancePercentage)
	}
	if len(summary.RecentPeriods) != 2 {
		t.Fatalf("expected 2 recent periods, got %d", len(summary.RecentPeriods))
	}
}

func TestGetChildAttendanceSummary_EmptyStudentID(t *testing.T) {
	h := newTestHarness()

	_, err := h.svc.GetChildAttendanceSummary(context.Background(), "tenant_001", "school_001", "", "term_001")
	if err == nil {
		t.Fatal("expected error for empty studentID, got nil")
	}
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got %v", err)
	}
}

func TestGetChildAttendanceSummary_EmptyTermID(t *testing.T) {
	h := newTestHarness()

	_, err := h.svc.GetChildAttendanceSummary(context.Background(), "tenant_001", "school_001", "stu_001", "")
	if err == nil {
		t.Fatal("expected error for empty termID, got nil")
	}
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got %v", err)
	}
}

// ============================================================================
// Tests: ComputeTermSummaries
// ============================================================================

func TestComputeTermSummaries_HappyPath(t *testing.T) {
	h := newTestHarness()

	h.repo.computeTermSummariesFn = func(ctx context.Context, tenantID, schoolID, termID string) (int, error) {
		if termID != "term_001" {
			t.Errorf("expected termID 'term_001', got %q", termID)
		}
		return 50, nil
	}

	count, err := h.svc.ComputeTermSummaries(context.Background(), "tenant_001", "school_001", "term_001")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if count != 50 {
		t.Fatalf("expected count 50, got %d", count)
	}
}

func TestComputeTermSummaries_EmptyTermID(t *testing.T) {
	h := newTestHarness()

	_, err := h.svc.ComputeTermSummaries(context.Background(), "tenant_001", "school_001", "")
	if err == nil {
		t.Fatal("expected error for empty termID, got nil")
	}
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got %v", err)
	}
}

// ============================================================================
// Tests: ComputeClassSummaries
// ============================================================================

func TestComputeClassSummaries_HappyPath(t *testing.T) {
	h := newTestHarness()

	h.repo.computeClassSummariesFn = func(ctx context.Context, tenantID, schoolID, termID, classID string) (int, error) {
		if termID != "term_001" {
			t.Errorf("expected termID 'term_001', got %q", termID)
		}
		if classID != "class_001" {
			t.Errorf("expected classID 'class_001', got %q", classID)
		}
		return 30, nil
	}

	count, err := h.svc.ComputeClassSummaries(context.Background(), "tenant_001", "school_001", "term_001", "class_001")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if count != 30 {
		t.Fatalf("expected count 30, got %d", count)
	}
}

func TestComputeClassSummaries_EmptyTermID(t *testing.T) {
	h := newTestHarness()

	_, err := h.svc.ComputeClassSummaries(context.Background(), "tenant_001", "school_001", "", "class_001")
	if err == nil {
		t.Fatal("expected error for empty termID, got nil")
	}
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got %v", err)
	}
}

func TestComputeClassSummaries_EmptyClassID(t *testing.T) {
	h := newTestHarness()

	_, err := h.svc.ComputeClassSummaries(context.Background(), "tenant_001", "school_001", "term_001", "")
	if err == nil {
		t.Fatal("expected error for empty classID, got nil")
	}
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got %v", err)
	}
}
