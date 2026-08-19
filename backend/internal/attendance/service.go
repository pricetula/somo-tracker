package attendance

import (
	"context"
	"errors"
	"fmt"
)

// Service handles business logic for attendance operations.
type Service struct {
	repo     Repository
	enqueuer *Enqueuer
}

// NewService creates a new attendance Service.
func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

// SetEnqueuer sets the background task enqueuer for summary refreshes.
func (s *Service) SetEnqueuer(e *Enqueuer) {
	s.enqueuer = e
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

	result, err := s.repo.BatchMark(ctx, tenantID, schoolID, payload, markedBy, termID)
	if err != nil {
		return nil, fmt.Errorf("attendance.Service.BatchMark: %w", err)
	}

	// Asynchronously refresh all attendance-related summaries for the term.
	// These are best-effort — the HTTP response is not blocked.
	if s.enqueuer != nil {
		s.enqueuer.EnqueueTeacherDeliveryRefresh(ctx, tenantID, termID)
		s.enqueuer.EnqueueAttendanceTermRefresh(ctx, tenantID, schoolID, termID)
		s.enqueuer.EnqueueClassDailyRefresh(ctx, tenantID, schoolID, payload.TimetableSlotID, payload.Date)
	}

	return result, nil
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

// ── Class Learning Area Term Summaries ─────────────────────────────────

// GetClassLearningAreaTermSummary returns the class learning area term summary
// for a class, learning area, and term.
func (s *Service) GetClassLearningAreaTermSummary(ctx context.Context, tenantID, schoolID, classID, learningAreaID, termID string) (*ClassLearningAreaTermSummary, error) {
	if classID == "" {
		return nil, fmt.Errorf("attendance.Service.GetClassLearningAreaTermSummary: class_id is required: %w", ErrInvalidInput)
	}
	if learningAreaID == "" {
		return nil, fmt.Errorf("attendance.Service.GetClassLearningAreaTermSummary: learning_area_id is required: %w", ErrInvalidInput)
	}
	if termID == "" {
		return nil, fmt.Errorf("attendance.Service.GetClassLearningAreaTermSummary: term_id is required: %w", ErrInvalidInput)
	}
	return s.repo.GetClassLearningAreaTermSummary(ctx, tenantID, schoolID, classID, learningAreaID, termID)
}

// ListClassLearningAreaTermSummaries returns all class learning area term summaries
// for a school/term, optionally filtered by class and/or learning_area.
func (s *Service) ListClassLearningAreaTermSummaries(ctx context.Context, tenantID, schoolID, classID, learningAreaID, termID string) (*ClassLearningAreaTermSummaryListResponse, error) {
	if termID == "" {
		return nil, fmt.Errorf("attendance.Service.ListClassLearningAreaTermSummaries: term_id is required: %w", ErrInvalidInput)
	}
	items, err := s.repo.ListClassLearningAreaTermSummaries(ctx, tenantID, schoolID, classID, learningAreaID, termID)
	if err != nil {
		return nil, fmt.Errorf("attendance.Service.ListClassLearningAreaTermSummaries: %w", err)
	}
	if items == nil {
		items = []ClassLearningAreaTermSummary{}
	}
	return &ClassLearningAreaTermSummaryListResponse{Items: items, Total: len(items)}, nil
}

// RefreshClassLearningAreaTermSummary triggers an async recomputation of the
// class learning area term summary for a class, learning area, and term.
// The recomputation is owned exclusively by the Asynq worker, so this only
// enqueues the task via the Service enqueuer (best-effort, non-blocking).
func (s *Service) RefreshClassLearningAreaTermSummary(ctx context.Context, tenantID, schoolID, termID, classID string) (*RefreshSummaryResponse, error) {
	if termID == "" {
		return nil, fmt.Errorf("attendance.Service.RefreshClassLearningAreaTermSummary: term_id is required: %w", ErrInvalidInput)
	}
	if s.enqueuer != nil {
		s.enqueuer.EnqueueClassLearningAreaTermRefresh(ctx, tenantID, schoolID, termID, classID)
	}
	return &RefreshSummaryResponse{
		Message: "Class learning area term summary refresh enqueued",
		TermID:  termID,
	}, nil
}

// ── Class Term Attendance Summaries ───────────────────────────────────

// GetClassTermAttendanceSummary returns the class term attendance summary for a class and term.
func (s *Service) GetClassTermAttendanceSummary(ctx context.Context, tenantID, schoolID, classID, termID string) (*ClassTermAttendanceSummary, error) {
	if classID == "" {
		return nil, fmt.Errorf("attendance.Service.GetClassTermAttendanceSummary: class_id is required: %w", ErrInvalidInput)
	}
	if termID == "" {
		return nil, fmt.Errorf("attendance.Service.GetClassTermAttendanceSummary: term_id is required: %w", ErrInvalidInput)
	}
	return s.repo.GetClassTermAttendanceSummary(ctx, tenantID, schoolID, classID, termID)
}

// ListClassTermAttendanceSummaries returns all class term attendance summaries
// for a school/term, optionally filtered by class.
func (s *Service) ListClassTermAttendanceSummaries(ctx context.Context, tenantID, schoolID, classID, termID string) (*ClassTermAttendanceSummaryListResponse, error) {
	if termID == "" {
		return nil, fmt.Errorf("attendance.Service.ListClassTermAttendanceSummaries: term_id is required: %w", ErrInvalidInput)
	}
	items, err := s.repo.ListClassTermAttendanceSummaries(ctx, tenantID, schoolID, classID, termID)
	if err != nil {
		return nil, fmt.Errorf("attendance.Service.ListClassTermAttendanceSummaries: %w", err)
	}
	if items == nil {
		items = []ClassTermAttendanceSummary{}
	}
	return &ClassTermAttendanceSummaryListResponse{Items: items, Total: len(items)}, nil
}

// RefreshClassTermAttendanceSummary triggers an async recomputation of the
// class term attendance summary for a class and term.
// The recomputation is owned exclusively by the Asynq worker, so this only
// enqueues the task via the Service enqueuer (best-effort, non-blocking).
func (s *Service) RefreshClassTermAttendanceSummary(ctx context.Context, tenantID, schoolID, termID, classID string) (*RefreshSummaryResponse, error) {
	if termID == "" {
		return nil, fmt.Errorf("attendance.Service.RefreshClassTermAttendanceSummary: term_id is required: %w", ErrInvalidInput)
	}
	if s.enqueuer != nil {
		s.enqueuer.EnqueueClassTermRefresh(ctx, tenantID, schoolID, termID, classID)
	}
	return &RefreshSummaryResponse{
		Message: "Class term attendance summary refresh enqueued",
		TermID:  termID,
	}, nil
}

// ListClassAttendanceBreakdowns returns per-class Present/Late/Absent counts
// for the School Administrator dashboard grouped bar chart, sorted by absent
// count descending so high-absenteeism classes surface first (truancy and
// chronic absenteeism watch).
func (s *Service) ListClassAttendanceBreakdowns(ctx context.Context, tenantID, schoolID, termID string) (*ClassAttendanceBreakdownListResponse, error) {
	if termID == "" {
		return nil, fmt.Errorf("attendance.Service.ListClassAttendanceBreakdowns: academic_term_id is required: %w", ErrInvalidInput)
	}
	items, err := s.repo.ListClassAttendanceBreakdowns(ctx, tenantID, schoolID, termID)
	if err != nil {
		return nil, fmt.Errorf("attendance.Service.ListClassAttendanceBreakdowns: %w", err)
	}
	if items == nil {
		items = []ClassAttendanceBreakdownItem{}
	}
	return &ClassAttendanceBreakdownListResponse{Items: items, Total: len(items)}, nil
}

// ListLearningAreaBreakdowns returns per-learning-area Present/Absent/Excused
// period counts for the School Administrator dashboard grouped bar chart,
// aggregated across all classes and sorted by absent period count descending
// so the highest-absenteeism subjects surface first (truancy hotspot watch).
func (s *Service) ListLearningAreaBreakdowns(ctx context.Context, tenantID, schoolID, termID string) (*LearningAreaAttendanceBreakdownListResponse, error) {
	if termID == "" {
		return nil, fmt.Errorf("attendance.Service.ListLearningAreaBreakdowns: academic_term_id is required: %w", ErrInvalidInput)
	}
	items, err := s.repo.ListLearningAreaBreakdowns(ctx, tenantID, schoolID, termID)
	if err != nil {
		return nil, fmt.Errorf("attendance.Service.ListLearningAreaBreakdowns: %w", err)
	}
	if items == nil {
		items = []LearningAreaAttendanceBreakdownItem{}
	}
	return &LearningAreaAttendanceBreakdownListResponse{Items: items, Total: len(items)}, nil
}

// GetDayOfWeekSummaries returns attendance exceptions (absent/late/excused)
// aggregated by day of week for the current academic year. When classID is
// empty, results are aggregated across all classes in the tenant.
func (s *Service) GetDayOfWeekSummaries(ctx context.Context, tenantID string, classID *string) (*DayOfWeekSummariesResponse, error) {
	result, err := s.repo.GetDayOfWeekSummaries(ctx, tenantID, classID)
	if err != nil {
		return nil, fmt.Errorf("attendance.Service.GetDayOfWeekSummaries: %w", err)
	}
	if result.Data == nil {
		result.Data = []DayOfWeekSummaryItem{}
	}
	return &result, nil
}

// ── School Attendance KPIs ──────────────────────────────────────────────

// GetSchoolAttendanceKPIs returns macro-level attendance KPIs for the School
// Administrator dashboard. When termID is empty, the active term covering
// `date` is resolved automatically via GetTermIDByDate; if no term covers the
// date (holiday, weekend, future date) the active-term rate degrades to 0.00
// rather than failing the whole dashboard, because today's rate and the
// slot/session counts remain meaningful on their own.
func (s *Service) GetSchoolAttendanceKPIs(ctx context.Context, tenantID, schoolID, date, termID string) (*SchoolAttendanceKPI, error) {
	if date == "" {
		return nil, fmt.Errorf("attendance.Service.GetSchoolAttendanceKPIs: date is required: %w", ErrInvalidInput)
	}

	if termID == "" {
		resolved, err := s.repo.GetTermIDByDate(ctx, tenantID, schoolID, date)
		switch {
		case err == nil:
			termID = resolved
		case errors.Is(err, ErrInvalidInput):
			// No academic term covers this date — degrade the active-term rate.
			termID = ""
		default:
			return nil, fmt.Errorf("attendance.Service.GetSchoolAttendanceKPIs: resolve active term for date %s: %w", date, err)
		}
	}

	kpi, err := s.repo.GetSchoolAttendanceKPIs(ctx, tenantID, schoolID, date, termID)
	if err != nil {
		return nil, fmt.Errorf("attendance.Service.GetSchoolAttendanceKPIs: %w", err)
	}
	return kpi, nil
}

// ListClassTermPercentages returns the percentage of attendance statuses (present, absent, excused, late) for each class and term in the current academic year for a school, with a rollup row for "All" classes.
func (s *Service) ListClassTermPercentages(ctx context.Context, tenantID, schoolID string) ([]ClassTermPercentageItem, error) {
	return s.repo.ListClassTermPercentages(ctx, tenantID, schoolID)
}

// ── Calendar Status ───────────────────────────────────────────────────────

// ComputeDayStatus maps expected/handled counts to a DayStatus.
// This is a pure function — no DB, no side effects — making it unit-testable.
func ComputeDayStatus(expectedCount, handledCount int) DayStatus {
	switch {
	case expectedCount == 0:
		return DayStatusNone
	case handledCount == expectedCount:
		return DayStatusGreen
	case handledCount == 0:
		return DayStatusRed
	default:
		return DayStatusYellow
	}
}

// GetCalendarStatus returns per-date attendance status for a school over a date range.
func (s *Service) GetCalendarStatus(ctx context.Context, tenantID, schoolID, startDate, endDate string) (*CalendarStatusListResponse, error) {
	if startDate == "" {
		return nil, fmt.Errorf("attendance.Service.GetCalendarStatus: start_date is required: %w", ErrInvalidInput)
	}
	if endDate == "" {
		return nil, fmt.Errorf("attendance.Service.GetCalendarStatus: end_date is required: %w", ErrInvalidInput)
	}

	raw, err := s.repo.ListCalendarStatus(ctx, tenantID, schoolID, startDate, endDate)
	if err != nil {
		return nil, fmt.Errorf("attendance.Service.GetCalendarStatus: %w", err)
	}

	items := make([]CalendarDayStatus, 0, len(raw))
	for _, r := range raw {
		items = append(items, CalendarDayStatus{
			Date:          r.Date,
			ExpectedCount: r.ExpectedCount,
			HandledCount:  r.HandledCount,
			Status:        ComputeDayStatus(r.ExpectedCount, r.HandledCount),
		})
	}

	return &CalendarStatusListResponse{Items: items, Total: len(items)}, nil
}

// GetLowestAttendanceStudents returns the N students with the lowest attendance percentage
// for the current week (or a specified limit). If limit is 0, defaults to 5.
func (s *Service) GetLowestAttendanceStudents(ctx context.Context, tenantID, schoolID string, limit int) ([]LowestAttendanceStudent, error) {
	if limit <= 0 {
		limit = 5
	}
	return s.repo.GetLowestAttendanceStudents(ctx, tenantID, schoolID, limit)
}
