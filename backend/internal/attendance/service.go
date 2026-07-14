package attendance

import (
	"context"
	"fmt"
	"time"
)

// Service handles business logic for attendance operations.
type Service struct {
	repo Repository
}

// NewService creates a new attendance Service.
func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

// GetRosterForSlot returns the class roster with existing attendance marks for a slot.
func (s *Service) GetRosterForSlot(ctx context.Context, tenantID, schoolID, timetableSlotID, date string) (*SlotRosterResponse, error) {
	if timetableSlotID == "" {
		return nil, fmt.Errorf("attendance.Service.GetRosterForSlot: timetable_slot_id is required: %w", ErrInvalidInput)
	}
	if date == "" {
		return nil, fmt.Errorf("attendance.Service.GetRosterForSlot: date is required: %w", ErrInvalidInput)
	}
	return s.repo.GetRosterForSlot(ctx, tenantID, schoolID, timetableSlotID, date)
}

// BulkMarkAttendance records attendance for all students in a given slot/date.
func (s *Service) BulkMarkAttendance(ctx context.Context, tenantID, schoolID string, payload BulkAttendancePayload, markedBy string) error {
	if payload.TimetableSlotID == "" {
		return fmt.Errorf("attendance.Service.BulkMarkAttendance: timetable_slot_id is required: %w", ErrInvalidInput)
	}
	if payload.Date == "" {
		return fmt.Errorf("attendance.Service.BulkMarkAttendance: date is required: %w", ErrInvalidInput)
	}
	if len(payload.Entries) == 0 {
		return fmt.Errorf("attendance.Service.BulkMarkAttendance: at least one entry is required: %w", ErrInvalidInput)
	}
	for _, entry := range payload.Entries {
		if entry.StudentID == "" {
			return fmt.Errorf("attendance.Service.BulkMarkAttendance: student_id is required in all entries: %w", ErrInvalidInput)
		}
		if entry.Status == "" {
			return fmt.Errorf("attendance.Service.BulkMarkAttendance: status is required for student %s: %w", entry.StudentID, ErrInvalidInput)
		}
	}
	return s.repo.BulkUpsert(ctx, tenantID, schoolID, payload, markedBy)
}

// GetStudentHistory returns attendance history for a specific student.
func (s *Service) GetStudentHistory(ctx context.Context, tenantID, schoolID, studentID string, filter StudentHistoryFilter) ([]AttendanceRecord, error) {
	if studentID == "" {
		return nil, fmt.Errorf("attendance.Service.GetStudentHistory: student_id is required: %w", ErrInvalidInput)
	}
	return s.repo.GetStudentHistory(ctx, tenantID, schoolID, studentID, filter)
}

// UpdateAttendanceRecord updates a single attendance record (admin correction).
func (s *Service) UpdateAttendanceRecord(ctx context.Context, id, tenantID string, payload UpdateAttendanceEntryPayload) error {
	if id == "" {
		return fmt.Errorf("attendance.Service.UpdateAttendanceRecord: id is required: %w", ErrInvalidInput)
	}
	if payload.Status == "" {
		return fmt.Errorf("attendance.Service.UpdateAttendanceRecord: status is required: %w", ErrInvalidInput)
	}
	return s.repo.UpdateRecord(ctx, id, tenantID, payload)
}

// GetAdminDashboard returns the school-wide attendance completion view.
func (s *Service) GetAdminDashboard(ctx context.Context, tenantID, schoolID, date string) (*AdminDashboardResponse, error) {
	if date == "" {
		date = time.Now().Format("2006-01-02")
	}
	if schoolID == "" {
		return nil, fmt.Errorf("attendance.Service.GetAdminDashboard: school_id is required: %w", ErrInvalidInput)
	}
	return s.repo.GetAdminDashboard(ctx, tenantID, schoolID, date)
}

// GetChildAttendanceSummary returns a parent-facing summary for their child.
func (s *Service) GetChildAttendanceSummary(ctx context.Context, tenantID, schoolID, studentID, termID string) (*ChildAttendanceSummary, error) {
	if studentID == "" {
		return nil, fmt.Errorf("attendance.Service.GetChildAttendanceSummary: student_id is required: %w", ErrInvalidInput)
	}
	if termID == "" {
		return nil, fmt.Errorf("attendance.Service.GetChildAttendanceSummary: term_id is required: %w", ErrInvalidInput)
	}
	return s.repo.GetChildAttendanceSummary(ctx, tenantID, schoolID, studentID, termID)
}

// ComputeTermSummaries triggers a recalculation of attendance_term_summaries.
func (s *Service) ComputeTermSummaries(ctx context.Context, tenantID, schoolID, termID string) (int, error) {
	if termID == "" {
		return 0, fmt.Errorf("attendance.Service.ComputeTermSummaries: term_id is required: %w", ErrInvalidInput)
	}
	return s.repo.ComputeTermSummaries(ctx, tenantID, schoolID, termID)
}
