package attendance

import (
	"context"
	"fmt"

	"somotracker/backend/internal/middleware"
)

// Service contains business logic for the attendance domain.
type Service struct {
	Repo Repository
}

// NewService creates a new Service.
func NewService(repo Repository) *Service {
	return &Service{Repo: repo}
}

// OpenAndSubmitAttendance opens (or reuses) an attendance period and submits
// the full batch of student attendance logs.
func (s *Service) OpenAndSubmitAttendance(ctx context.Context, tenantID, userID string, input MarkAttendanceInput) error {
	// --- Validation ---
	if tenantID == "" || userID == "" {
		return fmt.Errorf("attendance.Service.OpenAndSubmitAttendance: %w", ErrUnauthorized)
	}
	if input.AcademicTermID == "" || input.ClassID == "" || input.LearningAreaID == "" || input.Date == "" {
		return &middleware.FieldError{
			Err:    ErrInvalidInput,
			Fields: map[string][]string{"field": {"academic_term_id, class_id, learning_area_id, and date are required"}},
		}
	}
	for i, s := range input.Students {
		if s.StudentID == "" {
			return &middleware.FieldError{
				Err:    ErrInvalidInput,
				Fields: map[string][]string{fmt.Sprintf("students[%d].student_id", i): {"Student ID is required"}},
			}
		}
		if s.Status == "" {
			return &middleware.FieldError{
				Err:    ErrInvalidInput,
				Fields: map[string][]string{fmt.Sprintf("students[%d].status", i): {"Attendance status is required"}},
			}
		}
	}

	// --- Authorization check ---
	schoolID := input.SchoolID
	auth, err := s.Repo.IsAuthorizedRecorder(ctx, tenantID, userID, input.ClassID, input.LearningAreaID, input.AcademicTermID)
	if err != nil {
		return fmt.Errorf("attendance.Service.OpenAndSubmitAttendance: %w", err)
	}
	if !auth.Authorized {
		return fmt.Errorf("attendance.Service.OpenAndSubmitAttendance: %w", ErrForbidden)
	}

	// --- Get or create the attendance period ---
	openInput := OpenPeriodInput{
		TenantID:       tenantID,
		SchoolID:       schoolID,
		AcademicTermID: input.AcademicTermID,
		ClassID:        input.ClassID,
		LearningAreaID: input.LearningAreaID,
		Date:           input.Date,
		RecordedBy:     userID,
	}

	period, isNew, err := s.Repo.GetOrCreatePeriod(ctx, openInput)
	if err != nil {
		return fmt.Errorf("attendance.Service.OpenAndSubmitAttendance: %w", err)
	}

	// If newly created, stamp the authorized_by_role
	if isNew && auth.Role != nil {
		if err := s.Repo.UpdatePeriodAuthorizedBy(ctx, period.ID, tenantID, *auth.Role); err != nil {
			return fmt.Errorf("attendance.Service.OpenAndSubmitAttendance: %w", err)
		}
	}

	// --- Upsert attendance logs ---
	if len(input.Students) > 0 {
		if err := s.Repo.UpsertAttendanceLogs(ctx, tenantID, period.ID, userID, input.Students); err != nil {
			return fmt.Errorf("attendance.Service.OpenAndSubmitAttendance: %w", err)
		}
	}

	return nil
}

// GetPeriod returns a period with all its attendance logs.
func (s *Service) GetPeriod(ctx context.Context, tenantID, periodID string) (*AttendancePeriod, []AttendanceLog, error) {
	if tenantID == "" || periodID == "" {
		return nil, nil, fmt.Errorf("attendance.Service.GetPeriod: %w", ErrInvalidInput)
	}
	period, logs, err := s.Repo.GetPeriodLogs(ctx, tenantID, periodID)
	if err != nil {
		return nil, nil, fmt.Errorf("attendance.Service.GetPeriod: %w", err)
	}
	return period, logs, nil
}
