package teachers

import (
	"context"
	"fmt"
	"time"
)

// Service contains business logic for the teachers domain.
type Service struct {
	repo Repository
}

// NewService creates a new Service.
func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

// ListTeachers returns paginated teachers for a school.
func (s *Service) ListTeachers(ctx context.Context, tenantID, schoolID string, includeInactive bool, offset, limit int, search string) ([]Teacher, int, error) {
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	if offset < 0 {
		offset = 0
	}
	return s.repo.ListBySchool(ctx, tenantID, schoolID, includeInactive, offset, limit, search)
}

// GetTeacherByID returns a single teacher by user ID, scoped to tenant + school.
func (s *Service) GetTeacherByID(ctx context.Context, userID, tenantID, schoolID string) (*Teacher, error) {
	if userID == "" {
		return nil, fmt.Errorf("teachers.Service.GetTeacherByID: %w", ErrInvalidInput)
	}
	return s.repo.GetByID(ctx, userID, tenantID, schoolID)
}

// UpdateTeacher applies partial updates to a teacher's profile.
// Supported fields: full_name, tsc_number, knec_panel_assessor_id.
func (s *Service) UpdateTeacher(ctx context.Context, userID, tenantID, schoolID string, payload UpdateTeacherPayload) error {
	if userID == "" {
		return fmt.Errorf("teachers.Service.UpdateTeacher: %w", ErrInvalidInput)
	}
	if payload.FullName == nil && payload.TSCNumber == nil && payload.KNECPanelAssessor == nil {
		return fmt.Errorf("teachers.Service.UpdateTeacher: at least one field to update is required: %w", ErrInvalidInput)
	}
	return s.repo.Update(ctx, userID, tenantID, schoolID, payload)
}

// ToggleActive toggles the active status of a teacher's membership.
// If the teacher is not found, returns ErrNotFound.
func (s *Service) ToggleActive(ctx context.Context, tenantID, schoolID, userID string, isActive bool) error {
	if userID == "" {
		return fmt.Errorf("teachers.Service.ToggleActive: %w", ErrInvalidInput)
	}
	return s.repo.ToggleActive(ctx, tenantID, schoolID, userID, isActive)
}

// Delete hard-deletes a teacher's membership and user record.
func (s *Service) Delete(ctx context.Context, tenantID, schoolID, userID string) error {
	if userID == "" {
		return fmt.Errorf("teachers.Service.Delete: %w", ErrInvalidInput)
	}
	return s.repo.Delete(ctx, tenantID, schoolID, userID)
}

// ListTeacherClasses returns all classes assigned to a teacher in a term.
func (s *Service) ListTeacherClasses(ctx context.Context, tenantID, schoolID, userID, termID string) (*TeacherClassListResponse, error) {
	if userID == "" {
		return nil, fmt.Errorf("teachers.Service.ListTeacherClasses: user_id is required: %w", ErrInvalidInput)
	}
	if termID == "" {
		return nil, fmt.Errorf("teachers.Service.ListTeacherClasses: term_id is required: %w", ErrInvalidInput)
	}
	items, err := s.repo.ListTeacherClasses(ctx, tenantID, schoolID, userID, termID)
	if err != nil {
		return nil, err
	}
	if items == nil {
		items = []TeacherClassItem{}
	}
	return &TeacherClassListResponse{Items: items, Total: len(items)}, nil
}

// GetTeacherTimetable returns the teacher's timetable for a given day.
func (s *Service) ListTeacherLessonTimeline(ctx context.Context, tenantID, schoolID, userID, weekStart string, limit int) (*TeacherLessonTimelinePage, error) {
	if userID == "" {
		return nil, fmt.Errorf("teachers.Service.ListTeacherLessonTimeline: user_id is required: %w", ErrInvalidInput)
	}
	if weekStart == "" {
		weekStart = time.Now().UTC().Format("2006-01-02")
	}
	items, nextCursor, err := s.repo.ListTeacherLessonTimeline(ctx, tenantID, schoolID, userID, weekStart, limit)
	if err != nil {
		return nil, err
	}
	return &TeacherLessonTimelinePage{
		Entries:    items,
		NextCursor: strPtrIfNotEmpty(nextCursor),
	}, nil
}

func strPtrIfNotEmpty(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

func (s *Service) GetTeacherTimetable(ctx context.Context, tenantID, schoolID, userID string, dayOfWeek int) (*TeacherTimetableResponse, error) {
	if userID == "" {
		return nil, fmt.Errorf("teachers.Service.GetTeacherTimetable: user_id is required: %w", ErrInvalidInput)
	}
	if dayOfWeek < 1 || dayOfWeek > 7 {
		return nil, fmt.Errorf("teachers.Service.GetTeacherTimetable: day_of_week must be 1-7: %w", ErrInvalidInput)
	}
	items, err := s.repo.GetTeacherTimetable(ctx, tenantID, schoolID, userID, dayOfWeek)
	if err != nil {
		return nil, err
	}
	if items == nil {
		items = []TeacherTimetableAllocation{}
	}
	return &TeacherTimetableResponse{Items: items, Total: len(items)}, nil
}
