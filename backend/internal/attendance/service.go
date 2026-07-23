package attendance

import (
	"context"
	"fmt"
)

// Service handles business logic for attendance operations.
type Service struct {
	repo Repository
}

// NewService creates a new attendance Service.
func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

// ── Sessions ──────────────────────────────────────────────────────────────

// CreateSession creates a new attendance session.
func (s *Service) CreateSession(ctx context.Context, tenantID, schoolID string, payload CreateSessionPayload) (*AttendanceSession, error) {
	if payload.TimetableSlotID == "" {
		return nil, fmt.Errorf("attendance.Service.CreateSession: timetable_slot_id is required: %w", ErrInvalidInput)
	}
	if payload.Date == "" {
		return nil, fmt.Errorf("attendance.Service.CreateSession: date is required: %w", ErrInvalidInput)
	}
	if payload.Status == "" {
		return nil, fmt.Errorf("attendance.Service.CreateSession: status is required: %w", ErrInvalidInput)
	}
	if payload.Status != string(SessionSubmitted) && payload.Status != string(SessionSkipped) {
		return nil, fmt.Errorf("attendance.Service.CreateSession: status must be SUBMITTED or SKIPPED: %w", ErrInvalidInput)
	}
	if payload.Status == string(SessionSkipped) && (payload.SkipReason == nil || *payload.SkipReason == "") {
		return nil, fmt.Errorf("attendance.Service.CreateSession: skip_reason is required when status is SKIPPED: %w", ErrInvalidInput)
	}

	return s.repo.CreateSession(ctx, tenantID, schoolID, payload)
}

// GetSession returns a single session by ID.
func (s *Service) GetSession(ctx context.Context, id, tenantID string) (*AttendanceSession, error) {
	return s.repo.GetSessionByID(ctx, id, tenantID)
}

// GetEnrichedSession returns a single session with enriched data.
func (s *Service) GetEnrichedSession(ctx context.Context, id, tenantID string) (*SessionWithEnrichedData, error) {
	return s.repo.GetEnrichedSessionByID(ctx, id, tenantID)
}

// ListSessions returns sessions matching the filter.
func (s *Service) ListSessions(ctx context.Context, filter SessionFilter) (*SessionListResponse, error) {
	items, err := s.repo.ListSessions(ctx, filter)
	if err != nil {
		return nil, err
	}
	if items == nil {
		items = []SessionWithEnrichedData{}
	}
	return &SessionListResponse{Items: items, Total: len(items)}, nil
}

// UpdateSession updates a session's status and/or skip reason.
func (s *Service) UpdateSession(ctx context.Context, id, tenantID string, payload UpdateSessionPayload) (*AttendanceSession, error) {
	if id == "" {
		return nil, fmt.Errorf("attendance.Service.UpdateSession: id is required: %w", ErrInvalidInput)
	}
	if payload.Status != nil {
		if *payload.Status != string(SessionSubmitted) && *payload.Status != string(SessionSkipped) {
			return nil, fmt.Errorf("attendance.Service.UpdateSession: status must be SUBMITTED or SKIPPED: %w", ErrInvalidInput)
		}
	}
	return s.repo.UpdateSession(ctx, id, tenantID, payload)
}

// GetSessionsForClassDate returns all sessions for a class on a given date.
func (s *Service) GetSessionsForClassDate(ctx context.Context, tenantID, schoolID, classID, date string) (*SessionListResponse, error) {
	if classID == "" {
		return nil, fmt.Errorf("attendance.Service.GetSessionsForClassDate: class_id is required: %w", ErrInvalidInput)
	}
	if date == "" {
		return nil, fmt.Errorf("attendance.Service.GetSessionsForClassDate: date is required: %w", ErrInvalidInput)
	}
	items, err := s.repo.GetSessionsForClassDate(ctx, tenantID, schoolID, classID, date)
	if err != nil {
		return nil, err
	}
	if items == nil {
		items = []SessionWithEnrichedData{}
	}
	return &SessionListResponse{Items: items, Total: len(items)}, nil
}

// ── Records ───────────────────────────────────────────────────────────────

// BatchMark marks attendance for multiple students in a single slot+date.
func (s *Service) BatchMark(ctx context.Context, tenantID, schoolID string, payload BatchMarkPayload, markedBy, termID string) (*BatchMarkResult, error) {
	if payload.TimetableSlotID == "" {
		return nil, fmt.Errorf("attendance.Service.BatchMark: timetable_slot_id is required: %w", ErrInvalidInput)
	}
	if payload.Date == "" {
		return nil, fmt.Errorf("attendance.Service.BatchMark: date is required: %w", ErrInvalidInput)
	}
	if len(payload.Records) == 0 {
		return nil, fmt.Errorf("attendance.Service.BatchMark: at least one record is required: %w", ErrInvalidInput)
	}
	if markedBy == "" {
		return nil, fmt.Errorf("attendance.Service.BatchMark: marked_by is required: %w", ErrInvalidInput)
	}
	if termID == "" {
		resolved, err := s.repo.GetTermIDByDate(ctx, tenantID, schoolID, payload.Date)
		if err != nil {
			return nil, fmt.Errorf("attendance.Service.BatchMark: resolve academic_term_id from date %s: %w", payload.Date, err)
		}
		termID = resolved
	}

	validStatuses := map[AttendanceStatus]bool{
		StatusPresent: true,
		StatusAbsent:  true,
		StatusLate:    true,
		StatusExcused: true,
	}
	for _, rec := range payload.Records {
		if rec.StudentID == "" {
			return nil, fmt.Errorf("attendance.Service.BatchMark: student_id is required for all records: %w", ErrInvalidInput)
		}
		if !validStatuses[rec.Status] {
			return nil, fmt.Errorf("attendance.Service.BatchMark: invalid status %q for student %s: %w", rec.Status, rec.StudentID, ErrInvalidInput)
		}
	}

	return s.repo.BatchMark(ctx, tenantID, schoolID, payload, markedBy, termID)
}

// UpdateRecord updates a single attendance record.
func (s *Service) UpdateRecord(ctx context.Context, id, tenantID string, payload UpdateRecordPayload) (*AttendanceRecord, error) {
	if id == "" {
		return nil, fmt.Errorf("attendance.Service.UpdateRecord: id is required: %w", ErrInvalidInput)
	}
	if payload.Status != nil {
		validStatuses := map[AttendanceStatus]bool{
			StatusPresent: true,
			StatusAbsent:  true,
			StatusLate:    true,
			StatusExcused: true,
		}
		if !validStatuses[*payload.Status] {
			return nil, fmt.Errorf("attendance.Service.UpdateRecord: invalid status %q: %w", *payload.Status, ErrInvalidInput)
		}
	}
	return s.repo.UpdateRecord(ctx, id, tenantID, payload)
}

// GetRecord returns a single attendance record.
func (s *Service) GetRecord(ctx context.Context, id, tenantID string) (*AttendanceRecord, error) {
	return s.repo.GetRecordByID(ctx, id, tenantID)
}

// ListRecordsBySlotDate returns all records for a timetable slot on a date.
func (s *Service) ListRecordsBySlotDate(ctx context.Context, tenantID, schoolID, timetableSlotID, date string) (*RecordListResponse, error) {
	if timetableSlotID == "" {
		return nil, fmt.Errorf("attendance.Service.ListRecordsBySlotDate: timetable_slot_id is required: %w", ErrInvalidInput)
	}
	if date == "" {
		return nil, fmt.Errorf("attendance.Service.ListRecordsBySlotDate: date is required: %w", ErrInvalidInput)
	}
	items, err := s.repo.ListRecordsBySlotDate(ctx, tenantID, schoolID, timetableSlotID, date)
	if err != nil {
		return nil, err
	}
	if items == nil {
		items = []RecordWithEnrichedData{}
	}
	return &RecordListResponse{Items: items, Total: len(items)}, nil
}

// ListRecordsByStudentTerm returns all records for a student in a term.
func (s *Service) ListRecordsByStudentTerm(ctx context.Context, tenantID, schoolID, studentID, termID string) (*RecordListResponse, error) {
	if studentID == "" {
		return nil, fmt.Errorf("attendance.Service.ListRecordsByStudentTerm: student_id is required: %w", ErrInvalidInput)
	}
	if termID == "" {
		return nil, fmt.Errorf("attendance.Service.ListRecordsByStudentTerm: term_id is required: %w", ErrInvalidInput)
	}
	items, err := s.repo.ListRecordsByStudentTerm(ctx, tenantID, schoolID, studentID, termID)
	if err != nil {
		return nil, err
	}
	if items == nil {
		items = []RecordWithEnrichedData{}
	}
	return &RecordListResponse{Items: items, Total: len(items)}, nil
}

// ListRecordsByClassDate returns all records for a class on a date.
func (s *Service) ListRecordsByClassDate(ctx context.Context, tenantID, schoolID, classID, date, termID string) (*RecordListResponse, error) {
	if classID == "" {
		return nil, fmt.Errorf("attendance.Service.ListRecordsByClassDate: class_id is required: %w", ErrInvalidInput)
	}
	if date == "" {
		return nil, fmt.Errorf("attendance.Service.ListRecordsByClassDate: date is required: %w", ErrInvalidInput)
	}

	// Get all non-break timetable slots for this class on this day of week
	sessions, err := s.repo.GetSessionsForClassDate(ctx, tenantID, schoolID, classID, date)
	if err != nil {
		return nil, err
	}

	// Collect records for all slots
	var allRecords []RecordWithEnrichedData
	for _, session := range sessions {
		records, err := s.repo.ListRecordsBySlotDate(ctx, tenantID, schoolID, session.TimetableSlotID, date)
		if err != nil {
			return nil, err
		}
		allRecords = append(allRecords, records...)
	}

	if allRecords == nil {
		allRecords = []RecordWithEnrichedData{}
	}
	return &RecordListResponse{Items: allRecords, Total: len(allRecords)}, nil
}

// ListRecords returns attendance records matching the filter.
func (s *Service) ListRecords(ctx context.Context, filter RecordFilter) (*RecordListResponse, error) {
	items, err := s.repo.ListRecords(ctx, filter)
	if err != nil {
		return nil, err
	}
	if items == nil {
		items = []RecordWithEnrichedData{}
	}
	return &RecordListResponse{Items: items, Total: len(items)}, nil
}

// ── Summaries ─────────────────────────────────────────────────────────────

// GetStudentTermSummary returns attendance summaries for a student in a term.
func (s *Service) GetStudentTermSummary(ctx context.Context, tenantID, schoolID, studentID, termID string) (*SummaryListResponse, error) {
	if studentID == "" {
		return nil, fmt.Errorf("attendance.Service.GetStudentTermSummary: student_id is required: %w", ErrInvalidInput)
	}
	if termID == "" {
		return nil, fmt.Errorf("attendance.Service.GetStudentTermSummary: term_id is required: %w", ErrInvalidInput)
	}
	items, err := s.repo.GetStudentTermSummary(ctx, tenantID, schoolID, studentID, termID)
	if err != nil {
		return nil, err
	}
	if items == nil {
		items = []AttendanceTermSummary{}
	}
	return &SummaryListResponse{Items: items, Total: len(items)}, nil
}

// GetClassTermSummary returns attendance summaries for all students in a class.
func (s *Service) GetClassTermSummary(ctx context.Context, tenantID, schoolID, classID, termID string) (*SummaryListResponse, error) {
	if classID == "" {
		return nil, fmt.Errorf("attendance.Service.GetClassTermSummary: class_id is required: %w", ErrInvalidInput)
	}
	if termID == "" {
		return nil, fmt.Errorf("attendance.Service.GetClassTermSummary: term_id is required: %w", ErrInvalidInput)
	}
	items, err := s.repo.GetClassTermSummary(ctx, tenantID, schoolID, classID, termID)
	if err != nil {
		return nil, err
	}
	if items == nil {
		items = []AttendanceTermSummary{}
	}
	return &SummaryListResponse{Items: items, Total: len(items)}, nil
}

// RefreshSummaries triggers a recomputation of attendance summaries for a term.
func (s *Service) RefreshSummaries(ctx context.Context, tenantID, schoolID, termID string) (*RefreshSummaryResponse, error) {
	if termID == "" {
		return nil, fmt.Errorf("attendance.Service.RefreshSummaries: term_id is required: %w", ErrInvalidInput)
	}
	if err := s.repo.RefreshSummaries(ctx, tenantID, schoolID, termID); err != nil {
		return nil, err
	}
	return &RefreshSummaryResponse{
		Message: "Attendance summaries refreshed successfully",
		TermID:  termID,
	}, nil
}

// ── Class Daily Summaries ─────────────────────────────────────────────────

// GetClassDailySummary returns the daily attendance summary for a class on a date.
func (s *Service) GetClassDailySummary(ctx context.Context, tenantID, schoolID, classID, date string) (*ClassDailyAttendanceSummary, error) {
	if classID == "" {
		return nil, fmt.Errorf("attendance.Service.GetClassDailySummary: class_id is required: %w", ErrInvalidInput)
	}
	if date == "" {
		return nil, fmt.Errorf("attendance.Service.GetClassDailySummary: date is required: %w", ErrInvalidInput)
	}
	return s.repo.GetClassDailySummary(ctx, tenantID, schoolID, classID, date)
}

// RefreshClassDailySummary triggers a recomputation of the daily summary for a class on a date.
func (s *Service) RefreshClassDailySummary(ctx context.Context, tenantID, schoolID, classID, date string) (*RefreshSummaryResponse, error) {
	if classID == "" {
		return nil, fmt.Errorf("attendance.Service.RefreshClassDailySummary: class_id is required: %w", ErrInvalidInput)
	}
	if date == "" {
		return nil, fmt.Errorf("attendance.Service.RefreshClassDailySummary: date is required: %w", ErrInvalidInput)
	}
	if err := s.repo.RefreshClassDailySummary(ctx, tenantID, schoolID, classID, date); err != nil {
		return nil, err
	}
	return &RefreshSummaryResponse{
		Message: "Class daily attendance summary refreshed successfully",
		TermID:  date,
	}, nil
}

// ListClassDailySummaries returns daily summaries for a class within a date range.
func (s *Service) ListClassDailySummaries(ctx context.Context, tenantID, schoolID, classID, startDate, endDate string) (*ClassDailySummaryListResponse, error) {
	if classID == "" {
		return nil, fmt.Errorf("attendance.Service.ListClassDailySummaries: class_id is required: %w", ErrInvalidInput)
	}
	if startDate == "" {
		return nil, fmt.Errorf("attendance.Service.ListClassDailySummaries: start_date is required: %w", ErrInvalidInput)
	}
	if endDate == "" {
		return nil, fmt.Errorf("attendance.Service.ListClassDailySummaries: end_date is required: %w", ErrInvalidInput)
	}
	items, err := s.repo.ListClassDailySummaries(ctx, tenantID, schoolID, classID, startDate, endDate)
	if err != nil {
		return nil, err
	}
	if items == nil {
		items = []ClassDailyAttendanceSummary{}
	}
	return &ClassDailySummaryListResponse{Items: items, Total: len(items)}, nil
}
