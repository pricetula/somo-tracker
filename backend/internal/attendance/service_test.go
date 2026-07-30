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
	createSessionFn            func(ctx context.Context, tenantID, schoolID string, payload CreateSessionPayload) (*AttendanceSession, error)
	getSessionByIDFn           func(ctx context.Context, id, tenantID string) (*AttendanceSession, error)
	getEnrichedSessionByIDFn   func(ctx context.Context, id, tenantID string) (*SessionWithEnrichedData, error)
	listSessionsFn             func(ctx context.Context, filter SessionFilter) ([]SessionWithEnrichedData, error)
	updateSessionFn            func(ctx context.Context, id, tenantID string, payload UpdateSessionPayload) (*AttendanceSession, error)
	getSessionsForClassDateFn  func(ctx context.Context, tenantID, schoolID, classID, date string) ([]SessionWithEnrichedData, error)
	batchMarkFn                func(ctx context.Context, tenantID, schoolID string, payload BatchMarkPayload, markedBy, termID string) (*BatchMarkResult, error)
	updateRecordFn             func(ctx context.Context, id, tenantID string, payload UpdateRecordPayload) (*AttendanceRecord, error)
	getRecordByIDFn            func(ctx context.Context, id, tenantID string) (*AttendanceRecord, error)
	listRecordsBySlotDateFn    func(ctx context.Context, tenantID, schoolID, timetableSlotID, date string) ([]RecordWithEnrichedData, error)
	listRecordsByStudentTermFn func(ctx context.Context, tenantID, schoolID, studentID, termID string) ([]RecordWithEnrichedData, error)
	listRecordsByClassDateFn   func(ctx context.Context, tenantID, schoolID, classID, date string) ([]RecordWithEnrichedData, error)
	listRecordsFn              func(ctx context.Context, filter RecordFilter) ([]RecordWithEnrichedData, error)
	getTermIDByDateFn          func(ctx context.Context, tenantID, schoolID, date string) (string, error)
	getStudentTermSummaryFn    func(ctx context.Context, tenantID, schoolID, studentID, termID string) ([]AttendanceTermSummary, error)
	getClassTermSummaryFn      func(ctx context.Context, tenantID, schoolID, classID, termID string) ([]AttendanceTermSummary, error)
	refreshSummariesFn         func(ctx context.Context, tenantID, schoolID, termID string) error
	getClassDailySummaryFn     func(ctx context.Context, tenantID, schoolID, classID, date string) (*ClassDailyAttendanceSummary, error)
	refreshClassDailySummaryFn func(ctx context.Context, tenantID, schoolID, classID, date string) error
	listClassDailySummariesFn  func(ctx context.Context, tenantID, schoolID, classID, startDate, endDate string) ([]ClassDailyAttendanceSummary, error)
	listCalendarStatusFn       func(ctx context.Context, tenantID, schoolID, startDate, endDate string) ([]CalendarDayStatusRaw, error)
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

func TestCreateSession_HappyPath(t *testing.T) {
	h := newTestHarness()

	h.repo.createSessionFn = func(ctx context.Context, tenantID, schoolID string, payload CreateSessionPayload) (*AttendanceSession, error) {
		if payload.TimetableSlotID != "slot_001" {
			t.Errorf("expected slot_001, got %q", payload.TimetableSlotID)
		}
		if payload.Date != "2026-07-15" {
			t.Errorf("expected date 2026-07-15, got %q", payload.Date)
		}
		if payload.Status != "SUBMITTED" {
			t.Errorf("expected SUBMITTED, got %q", payload.Status)
		}
		return &AttendanceSession{ID: "session_001", TimetableSlotID: "slot_001", Date: "2026-07-15", Status: SessionSubmitted}, nil
	}

	session, err := h.svc.CreateSession(context.Background(), "tenant_001", "school_001", CreateSessionPayload{
		TimetableSlotID: "slot_001",
		Date:            "2026-07-15",
		Status:          "SUBMITTED",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if session.ID != "session_001" {
		t.Fatalf("expected id session_001, got %q", session.ID)
	}
	if session.Status != SessionSubmitted {
		t.Fatalf("expected SUBMITTED, got %q", session.Status)
	}
}

func TestCreateSession_SkippedWithReason(t *testing.T) {
	h := newTestHarness()

	reason := "Teacher Absence"
	h.repo.createSessionFn = func(ctx context.Context, tenantID, schoolID string, payload CreateSessionPayload) (*AttendanceSession, error) {
		if payload.Status != "SKIPPED" {
			t.Errorf("expected SKIPPED, got %q", payload.Status)
		}
		if payload.SkipReason == nil || *payload.SkipReason != "Teacher Absence" {
			t.Errorf("expected skip_reason 'Teacher Absence', got %v", payload.SkipReason)
		}
		return &AttendanceSession{ID: "session_002", Status: SessionSkipped, SkipReason: &reason}, nil
	}

	session, err := h.svc.CreateSession(context.Background(), "tenant_001", "school_001", CreateSessionPayload{
		TimetableSlotID: "slot_001",
		Date:            "2026-07-15",
		Status:          "SKIPPED",
		SkipReason:      &reason,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if session.Status != SessionSkipped {
		t.Fatalf("expected SKIPPED, got %q", session.Status)
	}
	if session.SkipReason == nil || *session.SkipReason != "Teacher Absence" {
		t.Fatalf("expected skip_reason 'Teacher Absence', got %v", session.SkipReason)
	}
}

func TestCreateSession_EmptyTimetableSlotID(t *testing.T) {
	h := newTestHarness()

	_, err := h.svc.CreateSession(context.Background(), "tenant_001", "school_001", CreateSessionPayload{
		Date:   "2026-07-15",
		Status: "SUBMITTED",
	})
	if err == nil {
		t.Fatal("expected error for empty timetable_slot_id, got nil")
	}
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got %v", err)
	}
}

func TestCreateSession_EmptyDate(t *testing.T) {
	h := newTestHarness()

	_, err := h.svc.CreateSession(context.Background(), "tenant_001", "school_001", CreateSessionPayload{
		TimetableSlotID: "slot_001",
		Status:          "SUBMITTED",
	})
	if err == nil {
		t.Fatal("expected error for empty date, got nil")
	}
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got %v", err)
	}
}

func TestCreateSession_InvalidStatus(t *testing.T) {
	h := newTestHarness()

	_, err := h.svc.CreateSession(context.Background(), "tenant_001", "school_001", CreateSessionPayload{
		TimetableSlotID: "slot_001",
		Date:            "2026-07-15",
		Status:          "INVALID",
	})
	if err == nil {
		t.Fatal("expected error for invalid status, got nil")
	}
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got %v", err)
	}
}

func TestCreateSession_SkippedMissingReason(t *testing.T) {
	h := newTestHarness()

	_, err := h.svc.CreateSession(context.Background(), "tenant_001", "school_001", CreateSessionPayload{
		TimetableSlotID: "slot_001",
		Date:            "2026-07-15",
		Status:          "SKIPPED",
	})
	if err == nil {
		t.Fatal("expected error for skipped without reason, got nil")
	}
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got %v", err)
	}
}

// ============================================================================
// Tests: Sessions — ListSessions
// ============================================================================

func TestListSessions_HappyPath(t *testing.T) {
	h := newTestHarness()

	expected := []SessionWithEnrichedData{
		{AttendanceSession: AttendanceSession{ID: "session_001", Status: SessionSubmitted}},
		{AttendanceSession: AttendanceSession{ID: "session_002", Status: SessionSkipped}},
	}

	h.repo.listSessionsFn = func(ctx context.Context, filter SessionFilter) ([]SessionWithEnrichedData, error) {
		if filter.TenantID != "tenant_001" {
			t.Errorf("expected tenant_001, got %q", filter.TenantID)
		}
		return expected, nil
	}

	result, err := h.svc.ListSessions(context.Background(), SessionFilter{TenantID: "tenant_001"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Total != 2 {
		t.Fatalf("expected 2 sessions, got %d", result.Total)
	}
	if result.Items[0].ID != "session_001" {
		t.Fatalf("expected session_001, got %q", result.Items[0].ID)
	}
}

func TestListSessions_Empty(t *testing.T) {
	h := newTestHarness()

	h.repo.listSessionsFn = func(ctx context.Context, filter SessionFilter) ([]SessionWithEnrichedData, error) {
		return []SessionWithEnrichedData{}, nil
	}

	result, err := h.svc.ListSessions(context.Background(), SessionFilter{TenantID: "tenant_001"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Total != 0 {
		t.Fatalf("expected 0 sessions, got %d", result.Total)
	}
}

// ============================================================================
// Tests: Sessions — UpdateSession
// ============================================================================

func TestUpdateSession_HappyPath(t *testing.T) {
	h := newTestHarness()

	status := "SKIPPED"
	reason := "School Assembly"
	h.repo.updateSessionFn = func(ctx context.Context, id, tenantID string, payload UpdateSessionPayload) (*AttendanceSession, error) {
		if id != "session_001" {
			t.Errorf("expected session_001, got %q", id)
		}
		if payload.Status == nil || *payload.Status != "SKIPPED" {
			t.Errorf("expected SKIPPED, got %v", payload.Status)
		}
		if payload.SkipReason == nil || *payload.SkipReason != "School Assembly" {
			t.Errorf("expected 'School Assembly', got %v", payload.SkipReason)
		}
		return &AttendanceSession{ID: id, Status: SessionSkipped, SkipReason: &reason}, nil
	}

	session, err := h.svc.UpdateSession(context.Background(), "session_001", "tenant_001", UpdateSessionPayload{
		Status: &status, SkipReason: &reason,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if session.Status != SessionSkipped {
		t.Fatalf("expected SKIPPED, got %q", session.Status)
	}
}

func TestUpdateSession_EmptyID(t *testing.T) {
	h := newTestHarness()

	status := "SKIPPED"
	_, err := h.svc.UpdateSession(context.Background(), "", "tenant_001", UpdateSessionPayload{Status: &status})
	if err == nil {
		t.Fatal("expected error for empty id, got nil")
	}
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got %v", err)
	}
}

func TestUpdateSession_InvalidStatus(t *testing.T) {
	h := newTestHarness()

	status := "INVALID"
	_, err := h.svc.UpdateSession(context.Background(), "session_001", "tenant_001", UpdateSessionPayload{Status: &status})
	if err == nil {
		t.Fatal("expected error for invalid status, got nil")
	}
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got %v", err)
	}
}

// ============================================================================
// Tests: Sessions — GetSessionsForClassDate
// ============================================================================

func TestGetSessionsForClassDate_HappyPath(t *testing.T) {
	h := newTestHarness()

	expected := []SessionWithEnrichedData{
		{AttendanceSession: AttendanceSession{ID: "session_001", TimetableSlotID: "slot_001"}},
		{AttendanceSession: AttendanceSession{ID: "session_002", TimetableSlotID: "slot_002"}},
	}

	h.repo.getSessionsForClassDateFn = func(ctx context.Context, tenantID, schoolID, classID, date string) ([]SessionWithEnrichedData, error) {
		if classID != "class_001" {
			t.Errorf("expected class_001, got %q", classID)
		}
		if date != "2026-07-15" {
			t.Errorf("expected 2026-07-15, got %q", date)
		}
		return expected, nil
	}

	result, err := h.svc.GetSessionsForClassDate(context.Background(), "tenant_001", "school_001", "class_001", "2026-07-15")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Total != 2 {
		t.Fatalf("expected 2 sessions, got %d", result.Total)
	}
}

func TestGetSessionsForClassDate_EmptyClassID(t *testing.T) {
	h := newTestHarness()

	_, err := h.svc.GetSessionsForClassDate(context.Background(), "tenant_001", "school_001", "", "2026-07-15")
	if err == nil {
		t.Fatal("expected error for empty classID, got nil")
	}
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got %v", err)
	}
}

func TestGetSessionsForClassDate_EmptyDate(t *testing.T) {
	h := newTestHarness()

	_, err := h.svc.GetSessionsForClassDate(context.Background(), "tenant_001", "school_001", "class_001", "")
	if err == nil {
		t.Fatal("expected error for empty date, got nil")
	}
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got %v", err)
	}
}

// ============================================================================
// Tests: Records — BatchMark
// ============================================================================

func TestBatchMark_HappyPath(t *testing.T) {
	h := newTestHarness()

	h.repo.batchMarkFn = func(ctx context.Context, tenantID, schoolID string, payload BatchMarkPayload, markedBy, termID string) (*BatchMarkResult, error) {
		if payload.TimetableSlotID != "slot_001" {
			t.Errorf("expected slot_001, got %q", payload.TimetableSlotID)
		}
		if payload.Date != "2026-07-15" {
			t.Errorf("expected 2026-07-15, got %q", payload.Date)
		}
		if len(payload.Records) != 2 {
			t.Errorf("expected 2 records, got %d", len(payload.Records))
		}
		if markedBy != "user_001" {
			t.Errorf("expected user_001, got %q", markedBy)
		}
		if termID != "term_001" {
			t.Errorf("expected term_001, got %q", termID)
		}
		return &BatchMarkResult{Created: 2, Updated: 0, Failed: 0}, nil
	}

	result, err := h.svc.BatchMark(context.Background(), "tenant_001", "school_001", BatchMarkPayload{
		TimetableSlotID: "slot_001",
		Date:            "2026-07-15",
		Records: []StudentAttendanceMark{
			{StudentID: "stu_001", Status: StatusPresent},
			{StudentID: "stu_002", Status: StatusAbsent},
		},
	}, "user_001", "term_001")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Created != 2 {
		t.Fatalf("expected 2 created, got %d", result.Created)
	}
}

func TestBatchMark_EmptyTimetableSlotID(t *testing.T) {
	h := newTestHarness()

	_, err := h.svc.BatchMark(context.Background(), "tenant_001", "school_001", BatchMarkPayload{
		Date: "2026-07-15",
		Records: []StudentAttendanceMark{
			{StudentID: "stu_001", Status: StatusPresent},
		},
	}, "user_001", "term_001")
	if err == nil {
		t.Fatal("expected error for empty timetable_slot_id, got nil")
	}
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got %v", err)
	}
}

func TestBatchMark_EmptyDate(t *testing.T) {
	h := newTestHarness()

	_, err := h.svc.BatchMark(context.Background(), "tenant_001", "school_001", BatchMarkPayload{
		TimetableSlotID: "slot_001",
		Records: []StudentAttendanceMark{
			{StudentID: "stu_001", Status: StatusPresent},
		},
	}, "user_001", "term_001")
	if err == nil {
		t.Fatal("expected error for empty date, got nil")
	}
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got %v", err)
	}
}

func TestBatchMark_EmptyRecords(t *testing.T) {
	h := newTestHarness()

	_, err := h.svc.BatchMark(context.Background(), "tenant_001", "school_001", BatchMarkPayload{
		TimetableSlotID: "slot_001",
		Date:            "2026-07-15",
		Records:         []StudentAttendanceMark{},
	}, "user_001", "term_001")
	if err == nil {
		t.Fatal("expected error for empty records, got nil")
	}
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got %v", err)
	}
}

func TestBatchMark_InvalidStatus(t *testing.T) {
	h := newTestHarness()

	_, err := h.svc.BatchMark(context.Background(), "tenant_001", "school_001", BatchMarkPayload{
		TimetableSlotID: "slot_001",
		Date:            "2026-07-15",
		Records: []StudentAttendanceMark{
			{StudentID: "stu_001", Status: "INVALID"},
		},
	}, "user_001", "term_001")
	if err == nil {
		t.Fatal("expected error for invalid status, got nil")
	}
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got %v", err)
	}
}

func TestBatchMark_EmptyMarkedBy(t *testing.T) {
	h := newTestHarness()

	_, err := h.svc.BatchMark(context.Background(), "tenant_001", "school_001", BatchMarkPayload{
		TimetableSlotID: "slot_001",
		Date:            "2026-07-15",
		Records: []StudentAttendanceMark{
			{StudentID: "stu_001", Status: StatusPresent},
		},
	}, "", "term_001")
	if err == nil {
		t.Fatal("expected error for empty markedBy, got nil")
	}
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got %v", err)
	}
}

func TestBatchMark_EmptyTermID_AutoResolves(t *testing.T) {
	h := newTestHarness()

	var capturedTermID string
	h.repo.batchMarkFn = func(ctx context.Context, tenantID, schoolID string, payload BatchMarkPayload, markedBy, termID string) (*BatchMarkResult, error) {
		capturedTermID = termID
		return &BatchMarkResult{Created: 1}, nil
	}

	result, err := h.svc.BatchMark(context.Background(), "tenant_001", "school_001", BatchMarkPayload{
		TimetableSlotID: "slot_001",
		Date:            "2026-07-15",
		Records: []StudentAttendanceMark{
			{StudentID: "stu_001", Status: StatusPresent},
		},
	}, "user_001", "")
	if err != nil {
		t.Fatalf("expected no error when termID auto-resolves, got %v", err)
	}
	if result.Created != 1 {
		t.Errorf("expected 1 created, got %d", result.Created)
	}
	if capturedTermID != "term_001" {
		t.Errorf("expected auto-resolved term_001, got %q", capturedTermID)
	}
}

// ============================================================================
// Tests: Records — UpdateRecord
// ============================================================================

func TestUpdateRecord_HappyPath(t *testing.T) {
	h := newTestHarness()

	status := StatusLate
	note := "Arrived 15 minutes late"
	h.repo.updateRecordFn = func(ctx context.Context, id, tenantID string, payload UpdateRecordPayload) (*AttendanceRecord, error) {
		if id != "rec_001" {
			t.Errorf("expected rec_001, got %q", id)
		}
		if payload.Status == nil || *payload.Status != StatusLate {
			t.Errorf("expected LATE, got %v", payload.Status)
		}
		if payload.Note == nil || *payload.Note != "Arrived 15 minutes late" {
			t.Errorf("expected note, got %v", payload.Note)
		}
		return &AttendanceRecord{ID: id, Status: StatusLate, Note: &note}, nil
	}

	rec, err := h.svc.UpdateRecord(context.Background(), "rec_001", "tenant_001", UpdateRecordPayload{
		Status: &status, Note: &note,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rec.Status != StatusLate {
		t.Fatalf("expected LATE, got %q", rec.Status)
	}
	if rec.Note == nil || *rec.Note != "Arrived 15 minutes late" {
		t.Fatalf("expected note, got %v", rec.Note)
	}
}

func TestUpdateRecord_EmptyID(t *testing.T) {
	h := newTestHarness()

	status := StatusPresent
	_, err := h.svc.UpdateRecord(context.Background(), "", "tenant_001", UpdateRecordPayload{Status: &status})
	if err == nil {
		t.Fatal("expected error for empty id, got nil")
	}
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got %v", err)
	}
}

func TestUpdateRecord_InvalidStatus(t *testing.T) {
	h := newTestHarness()

	status := AttendanceStatus("INVALID")
	_, err := h.svc.UpdateRecord(context.Background(), "rec_001", "tenant_001", UpdateRecordPayload{Status: &status})
	if err == nil {
		t.Fatal("expected error for invalid status, got nil")
	}
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got %v", err)
	}
}

// ============================================================================
// Tests: Records — ListRecordsBySlotDate
// ============================================================================

func TestListRecordsBySlotDate_HappyPath(t *testing.T) {
	h := newTestHarness()

	expected := []RecordWithEnrichedData{
		{AttendanceRecord: AttendanceRecord{ID: "rec_001", StudentID: "stu_001", Status: StatusPresent}},
		{AttendanceRecord: AttendanceRecord{ID: "rec_002", StudentID: "stu_002", Status: StatusAbsent}},
	}

	h.repo.listRecordsBySlotDateFn = func(ctx context.Context, tenantID, schoolID, timetableSlotID, date string) ([]RecordWithEnrichedData, error) {
		if timetableSlotID != "slot_001" {
			t.Errorf("expected slot_001, got %q", timetableSlotID)
		}
		if date != "2026-07-15" {
			t.Errorf("expected 2026-07-15, got %q", date)
		}
		return expected, nil
	}

	result, err := h.svc.ListRecordsBySlotDate(context.Background(), "tenant_001", "school_001", "slot_001", "2026-07-15")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Total != 2 {
		t.Fatalf("expected 2 records, got %d", result.Total)
	}
}

func TestListRecordsBySlotDate_EmptySlotID(t *testing.T) {
	h := newTestHarness()

	_, err := h.svc.ListRecordsBySlotDate(context.Background(), "tenant_001", "school_001", "", "2026-07-15")
	if err == nil {
		t.Fatal("expected error for empty slot id, got nil")
	}
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got %v", err)
	}
}

func TestListRecordsBySlotDate_EmptyDate(t *testing.T) {
	h := newTestHarness()

	_, err := h.svc.ListRecordsBySlotDate(context.Background(), "tenant_001", "school_001", "slot_001", "")
	if err == nil {
		t.Fatal("expected error for empty date, got nil")
	}
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got %v", err)
	}
}

// ============================================================================
// Tests: Records — ListRecordsByStudentTerm
// ============================================================================

func TestListRecordsByStudentTerm_HappyPath(t *testing.T) {
	h := newTestHarness()

	expected := []RecordWithEnrichedData{
		{AttendanceRecord: AttendanceRecord{ID: "rec_001", Date: "2026-07-15", Status: StatusPresent}},
	}

	h.repo.listRecordsByStudentTermFn = func(ctx context.Context, tenantID, schoolID, studentID, termID string) ([]RecordWithEnrichedData, error) {
		if studentID != "stu_001" {
			t.Errorf("expected stu_001, got %q", studentID)
		}
		if termID != "term_001" {
			t.Errorf("expected term_001, got %q", termID)
		}
		return expected, nil
	}

	result, err := h.svc.ListRecordsByStudentTerm(context.Background(), "tenant_001", "school_001", "stu_001", "term_001")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Total != 1 {
		t.Fatalf("expected 1 record, got %d", result.Total)
	}
}

func TestListRecordsByStudentTerm_EmptyStudentID(t *testing.T) {
	h := newTestHarness()

	_, err := h.svc.ListRecordsByStudentTerm(context.Background(), "tenant_001", "school_001", "", "term_001")
	if err == nil {
		t.Fatal("expected error for empty student id, got nil")
	}
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got %v", err)
	}
}

func TestListRecordsByStudentTerm_EmptyTermID(t *testing.T) {
	h := newTestHarness()

	_, err := h.svc.ListRecordsByStudentTerm(context.Background(), "tenant_001", "school_001", "stu_001", "")
	if err == nil {
		t.Fatal("expected error for empty term id, got nil")
	}
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got %v", err)
	}
}

// ============================================================================
// Tests: Records — ListRecords (generic filter)
// ============================================================================

func TestListRecords_HappyPath(t *testing.T) {
	h := newTestHarness()

	expected := []RecordWithEnrichedData{
		{AttendanceRecord: AttendanceRecord{ID: "rec_001", Status: StatusPresent}},
	}

	h.repo.listRecordsFn = func(ctx context.Context, filter RecordFilter) ([]RecordWithEnrichedData, error) {
		if filter.StudentID != "stu_001" {
			t.Errorf("expected stu_001, got %q", filter.StudentID)
		}
		if filter.AcademicTermID != "term_001" {
			t.Errorf("expected term_001, got %q", filter.AcademicTermID)
		}
		return expected, nil
	}

	result, err := h.svc.ListRecords(context.Background(), RecordFilter{
		StudentID:      "stu_001",
		AcademicTermID: "term_001",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Total != 1 {
		t.Fatalf("expected 1 record, got %d", result.Total)
	}
}

func TestListRecords_Empty(t *testing.T) {
	h := newTestHarness()

	h.repo.listRecordsFn = func(ctx context.Context, filter RecordFilter) ([]RecordWithEnrichedData, error) {
		return []RecordWithEnrichedData{}, nil
	}

	result, err := h.svc.ListRecords(context.Background(), RecordFilter{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Total != 0 {
		t.Fatalf("expected 0 records, got %d", result.Total)
	}
}

// ============================================================================
// Tests: Summaries — GetStudentTermSummary
// ============================================================================

func TestGetStudentTermSummary_HappyPath(t *testing.T) {
	h := newTestHarness()

	expected := []AttendanceTermSummary{
		{
			StudentID:            "stu_001",
			LearningAreaName:     "Mathematics",
			PeriodsTotal:         40,
			PeriodsPresent:       35,
			PeriodsAbsent:        3,
			PeriodsLate:          2,
			PeriodsExcused:       0,
			AttendancePercentage: 87.5,
		},
	}

	h.repo.getStudentTermSummaryFn = func(ctx context.Context, tenantID, schoolID, studentID, termID string) ([]AttendanceTermSummary, error) {
		if studentID != "stu_001" {
			t.Errorf("expected stu_001, got %q", studentID)
		}
		if termID != "term_001" {
			t.Errorf("expected term_001, got %q", termID)
		}
		return expected, nil
	}

	result, err := h.svc.GetStudentTermSummary(context.Background(), "tenant_001", "school_001", "stu_001", "term_001")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Total != 1 {
		t.Fatalf("expected 1 summary, got %d", result.Total)
	}
	if result.Items[0].AttendancePercentage != 87.5 {
		t.Fatalf("expected 87.5%%, got %f", result.Items[0].AttendancePercentage)
	}
}

func TestGetStudentTermSummary_EmptyStudentID(t *testing.T) {
	h := newTestHarness()

	_, err := h.svc.GetStudentTermSummary(context.Background(), "tenant_001", "school_001", "", "term_001")
	if err == nil {
		t.Fatal("expected error for empty student id, got nil")
	}
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got %v", err)
	}
}

func TestGetStudentTermSummary_EmptyTermID(t *testing.T) {
	h := newTestHarness()

	_, err := h.svc.GetStudentTermSummary(context.Background(), "tenant_001", "school_001", "stu_001", "")
	if err == nil {
		t.Fatal("expected error for empty term id, got nil")
	}
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got %v", err)
	}
}

// ============================================================================
// Tests: Summaries — GetClassTermSummary
// ============================================================================

func TestGetClassTermSummary_HappyPath(t *testing.T) {
	h := newTestHarness()

	expected := []AttendanceTermSummary{
		{StudentID: "stu_001", LearningAreaName: "English", PeriodsTotal: 40, PeriodsPresent: 38, AttendancePercentage: 95.0},
		{StudentID: "stu_002", LearningAreaName: "English", PeriodsTotal: 40, PeriodsPresent: 30, AttendancePercentage: 75.0},
	}

	h.repo.getClassTermSummaryFn = func(ctx context.Context, tenantID, schoolID, classID, termID string) ([]AttendanceTermSummary, error) {
		if classID != "class_001" {
			t.Errorf("expected class_001, got %q", classID)
		}
		if termID != "term_001" {
			t.Errorf("expected term_001, got %q", termID)
		}
		return expected, nil
	}

	result, err := h.svc.GetClassTermSummary(context.Background(), "tenant_001", "school_001", "class_001", "term_001")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Total != 2 {
		t.Fatalf("expected 2 summaries, got %d", result.Total)
	}
}

func TestGetClassTermSummary_EmptyClassID(t *testing.T) {
	h := newTestHarness()

	_, err := h.svc.GetClassTermSummary(context.Background(), "tenant_001", "school_001", "", "term_001")
	if err == nil {
		t.Fatal("expected error for empty class id, got nil")
	}
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got %v", err)
	}
}

func TestGetClassTermSummary_EmptyTermID(t *testing.T) {
	h := newTestHarness()

	_, err := h.svc.GetClassTermSummary(context.Background(), "tenant_001", "school_001", "class_001", "")
	if err == nil {
		t.Fatal("expected error for empty term id, got nil")
	}
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got %v", err)
	}
}

// ============================================================================
// Tests: Summaries — RefreshSummaries
// ============================================================================

func TestRefreshSummaries_HappyPath(t *testing.T) {
	h := newTestHarness()

	h.repo.refreshSummariesFn = func(ctx context.Context, tenantID, schoolID, termID string) error {
		if termID != "term_001" {
			t.Errorf("expected term_001, got %q", termID)
		}
		return nil
	}

	result, err := h.svc.RefreshSummaries(context.Background(), "tenant_001", "school_001", "term_001")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.TermID != "term_001" {
		t.Fatalf("expected term_001, got %q", result.TermID)
	}
	if result.Message == "" {
		t.Fatal("expected a message, got empty")
	}
}

func TestRefreshSummaries_EmptyTermID(t *testing.T) {
	h := newTestHarness()

	_, err := h.svc.RefreshSummaries(context.Background(), "tenant_001", "school_001", "")
	if err == nil {
		t.Fatal("expected error for empty term id, got nil")
	}
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got %v", err)
	}
}

// ============================================================================
// Tests: Sessions — GetSession / GetEnrichedSession
// ============================================================================

func TestGetSession_HappyPath(t *testing.T) {
	h := newTestHarness()

	now := time.Now()
	h.repo.getSessionByIDFn = func(ctx context.Context, id, tenantID string) (*AttendanceSession, error) {
		if id != "session_001" {
			t.Errorf("expected session_001, got %q", id)
		}
		return &AttendanceSession{ID: id, Status: SessionSubmitted, CreatedAt: now}, nil
	}

	session, err := h.svc.GetSession(context.Background(), "session_001", "tenant_001")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if session.Status != SessionSubmitted {
		t.Fatalf("expected SUBMITTED, got %q", session.Status)
	}
}

func TestGetSession_NotFound(t *testing.T) {
	h := newTestHarness()

	h.repo.getSessionByIDFn = func(ctx context.Context, id, tenantID string) (*AttendanceSession, error) {
		return nil, ErrNotFound
	}

	_, err := h.svc.GetSession(context.Background(), "session_999", "tenant_001")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

// ============================================================================
// Tests: Records — GetRecord
// ============================================================================

func TestGetRecord_HappyPath(t *testing.T) {
	h := newTestHarness()

	now := time.Now()
	h.repo.getRecordByIDFn = func(ctx context.Context, id, tenantID string) (*AttendanceRecord, error) {
		if id != "rec_001" {
			t.Errorf("expected rec_001, got %q", id)
		}
		return &AttendanceRecord{ID: id, Status: StatusPresent, CreatedAt: now}, nil
	}

	rec, err := h.svc.GetRecord(context.Background(), "rec_001", "tenant_001")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rec.Status != StatusPresent {
		t.Fatalf("expected PRESENT, got %q", rec.Status)
	}
}

func TestGetRecord_NotFound(t *testing.T) {
	h := newTestHarness()

	h.repo.getRecordByIDFn = func(ctx context.Context, id, tenantID string) (*AttendanceRecord, error) {
		return nil, ErrNotFound
	}

	_, err := h.svc.GetRecord(context.Background(), "rec_999", "tenant_001")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
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
