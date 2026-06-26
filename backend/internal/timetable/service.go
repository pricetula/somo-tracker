package timetable

import (
	"context"
	"fmt"

	"somotracker/backend/internal/middleware"
)

// Service contains business logic for the timetable domain.
type Service struct {
	Repo Repository
}

// NewService creates a new Service.
func NewService(repo Repository) *Service {
	return &Service{Repo: repo}
}

// BulkSaveSlots validates and bulk-upserts timetable slots for a term.
func (s *Service) BulkSaveSlots(ctx context.Context, tenantID, schoolID string, input BulkCreateTimetableSlotsInput) error {
	if tenantID == "" || schoolID == "" {
		return fmt.Errorf("timetable.Service.BulkSaveSlots: %w", ErrInvalidInput)
	}
	if input.AcademicYearID == "" || input.AcademicTermID == "" {
		return &middleware.FieldError{
			Err: ErrInvalidInput,
			Fields: map[string][]string{
				"academic_year_id": {"Academic year is required"},
				"academic_term_id": {"Academic term is required"},
			},
		}
	}

	// Validate term belongs to tenant + school
	valid, err := s.Repo.ValidateTerm(ctx, tenantID, schoolID, input.AcademicTermID)
	if err != nil {
		return fmt.Errorf("timetable.Service.BulkSaveSlots: %w", err)
	}
	if !valid {
		return &middleware.FieldError{
			Err:    ErrInvalidInput,
			Fields: map[string][]string{"academic_term_id": {"Academic term not found or does not belong to this school"}},
		}
	}

	if len(input.Slots) == 0 {
		return &middleware.FieldError{
			Err:    ErrInvalidInput,
			Fields: map[string][]string{"slots": {"At least one slot is required"}},
		}
	}

	for i, slot := range input.Slots {
		if slot.ClassID == "" {
			return &middleware.FieldError{
				Err:    ErrInvalidInput,
				Fields: map[string][]string{fmt.Sprintf("slots[%d].class_id", i): {"Class is required"}},
			}
		}
		if slot.TeacherID == "" {
			return &middleware.FieldError{
				Err:    ErrInvalidInput,
				Fields: map[string][]string{fmt.Sprintf("slots[%d].teacher_id", i): {"Teacher is required"}},
			}
		}
		if slot.DayOfWeek < 1 || slot.DayOfWeek > 7 {
			return &middleware.FieldError{
				Err:    ErrInvalidInput,
				Fields: map[string][]string{fmt.Sprintf("slots[%d].day_of_week", i): {"Day of week must be between 1 (Mon) and 7 (Sun)"}},
			}
		}
		if slot.StartTime == "" || slot.EndTime == "" {
			return &middleware.FieldError{
				Err:    ErrInvalidInput,
				Fields: map[string][]string{fmt.Sprintf("slots[%d].start_time", i): {"Start and end time are required"}},
			}
		}
	}

	if err := s.Repo.BulkUpsertSlots(ctx, tenantID, schoolID, input); err != nil {
		return fmt.Errorf("timetable.Service.BulkSaveSlots: %w", err)
	}
	return nil
}

// AssignTeacher validates role-specific rules and assigns a teacher to a class.
func (s *Service) AssignTeacher(ctx context.Context, input ClassTeacherInput) error {
	if input.TenantID == "" || input.SchoolID == "" || input.ClassID == "" || input.UserID == "" {
		return fmt.Errorf("timetable.Service.AssignTeacher: %w", ErrInvalidInput)
	}

	// Validate role-specific rules
	switch input.TeacherRole {
	case TeacherRolePrimary:
		if input.LearningAreaID != nil {
			return &middleware.FieldError{
				Err:    ErrInvalidInput,
				Fields: map[string][]string{"learning_area_id": {"PRIMARY_CLASS_TEACHER must not have a learning area"}},
			}
		}

		// A teacher can only hold PRIMARY_CLASS_TEACHER on one class
		hasPrimary, err := s.Repo.HasPrimaryRole(ctx, input.TenantID, input.UserID)
		if err != nil {
			return fmt.Errorf("timetable.Service.AssignTeacher: %w", err)
		}
		if hasPrimary {
			return fmt.Errorf("timetable.Service.AssignTeacher: teacher already holds PRIMARY_CLASS_TEACHER on another class: %w", ErrConflict)
		}

	case TeacherRoleSubject:
		if input.LearningAreaID == nil {
			return &middleware.FieldError{
				Err:    ErrInvalidInput,
				Fields: map[string][]string{"learning_area_id": {"SUBJECT_TEACHER requires a learning area"}},
			}
		}

	case TeacherRoleSubstitute:
		// No learning area restriction for substitutes
	}

	if err := s.Repo.AssignClassTeacher(ctx, input); err != nil {
		return fmt.Errorf("timetable.Service.AssignTeacher: %w", err)
	}
	return nil
}

// RemoveTeacher removes a teacher assignment from a class.
func (s *Service) RemoveTeacher(ctx context.Context, tenantID, classID, userID string) error {
	if tenantID == "" || classID == "" || userID == "" {
		return fmt.Errorf("timetable.Service.RemoveTeacher: %w", ErrInvalidInput)
	}
	if err := s.Repo.RemoveClassTeacher(ctx, tenantID, classID, userID); err != nil {
		return fmt.Errorf("timetable.Service.RemoveTeacher: %w", err)
	}
	return nil
}

// GetSlots retrieves slots filtered by class or teacher and term.
func (s *Service) GetSlots(ctx context.Context, tenantID, classID, teacherID, termID string) ([]TimetableSlot, error) {
	if tenantID == "" || termID == "" {
		return nil, fmt.Errorf("timetable.Service.GetSlots: %w", ErrInvalidInput)
	}

	var slots []TimetableSlot
	var err error

	switch {
	case classID != "":
		slots, err = s.Repo.GetSlotsByClass(ctx, tenantID, classID, termID)
	case teacherID != "":
		slots, err = s.Repo.GetSlotsByTeacher(ctx, tenantID, teacherID, termID)
	default:
		return nil, &middleware.FieldError{
			Err:    ErrInvalidInput,
			Fields: map[string][]string{"filter": {"Provide class_id or teacher_id"}},
		}
	}

	if err != nil {
		return nil, fmt.Errorf("timetable.Service.GetSlots: %w", err)
	}
	return slots, nil
}
