package classteachers

import (
	"context"
	"fmt"
	"strings"
)

// Service contains business logic for class teacher assignments.
type Service struct {
	repo Repository
}

// NewService creates a new Service.
func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

// ── CREATE ───────────────────────────────────────────────────────────────

// Create assigns a teacher to a class.
// Validates constraints: PRIMARY_CLASS_TEACHER (one per class),
// SUBJECT_TEACHER requires learning_area_id.
func (s *Service) Create(ctx context.Context, tenantID string, payload CreateClassTeacherPayload) (*ClassTeacher, error) {
	if tenantID == "" {
		return nil, fmt.Errorf("classteachers.Service.Create: %w", ErrInvalidInput)
	}
	if strings.TrimSpace(payload.UserID) == "" {
		return nil, fmt.Errorf("classteachers.Service.Create: user_id is required: %w", ErrInvalidInput)
	}
	if strings.TrimSpace(payload.ClassID) == "" {
		return nil, fmt.Errorf("classteachers.Service.Create: class_id is required: %w", ErrInvalidInput)
	}

	role := strings.ToUpper(strings.TrimSpace(payload.TeacherRole))
	switch role {
	case "PRIMARY_CLASS_TEACHER":
		// learning_area_id must be NULL for primary teacher
		if payload.LearningAreaID != nil && *payload.LearningAreaID != "" {
			return nil, fmt.Errorf("classteachers.Service.Create: PRIMARY_CLASS_TEACHER must not have learning_area_id: %w", ErrInvalidInput)
		}
		// Only one primary teacher per class
		count, err := s.repo.CountPrimaryForClass(ctx, payload.ClassID, tenantID)
		if err != nil {
			return nil, fmt.Errorf("classteachers.Service.Create: %w", err)
		}
		if count > 0 {
			return nil, fmt.Errorf("classteachers.Service.Create: class already has a primary teacher: %w", ErrPrimaryAlreadyAssigned)
		}
	case "SUBJECT_TEACHER":
		if payload.LearningAreaID == nil || *payload.LearningAreaID == "" {
			return nil, fmt.Errorf("classteachers.Service.Create: SUBJECT_TEACHER requires learning_area_id: %w", ErrInvalidInput)
		}
	case "SUBSTITUTE_TEACHER":
		// No additional constraints
	default:
		return nil, fmt.Errorf("classteachers.Service.Create: invalid teacher_role %q: %w", payload.TeacherRole, ErrInvalidInput)
	}

	params := CreateClassTeacherParams{
		TenantID:       tenantID,
		ClassID:        payload.ClassID,
		UserID:         payload.UserID,
		LearningAreaID: payload.LearningAreaID,
		TeacherRole:    role,
	}

	id, err := s.repo.Create(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("classteachers.Service.Create: %w", err)
	}

	assignment, err := s.repo.GetByID(ctx, id, tenantID)
	if err != nil {
		return nil, fmt.Errorf("classteachers.Service.Create: %w", err)
	}
	return assignment, nil
}

// ── GET BY ID ────────────────────────────────────────────────────────────

// GetByID retrieves a single class teacher assignment.
func (s *Service) GetByID(ctx context.Context, id, tenantID string) (*ClassTeacher, error) {
	if id == "" || tenantID == "" {
		return nil, fmt.Errorf("classteachers.Service.GetByID: %w", ErrInvalidInput)
	}
	ct, err := s.repo.GetByID(ctx, id, tenantID)
	if err != nil {
		return nil, fmt.Errorf("classteachers.Service.GetByID: %w", err)
	}
	return ct, nil
}

// ── LIST BY CLASS ────────────────────────────────────────────────────────

// ListByClass returns all teacher assignments for a class.
func (s *Service) ListByClass(ctx context.Context, classID, tenantID string) (*ClassTeacherListResponse, error) {
	if classID == "" || tenantID == "" {
		return nil, fmt.Errorf("classteachers.Service.ListByClass: %w", ErrInvalidInput)
	}
	items, err := s.repo.ListByClass(ctx, classID, tenantID)
	if err != nil {
		return nil, fmt.Errorf("classteachers.Service.ListByClass: %w", err)
	}
	return &ClassTeacherListResponse{
		Items: items,
		Total: len(items),
		Page:  1,
		Limit: len(items),
	}, nil
}

// ── LIST BY TEACHER ──────────────────────────────────────────────────────

// ListByTeacher returns all class assignments for a teacher.
func (s *Service) ListByTeacher(ctx context.Context, userID, tenantID string) (*ClassTeacherListResponse, error) {
	if userID == "" || tenantID == "" {
		return nil, fmt.Errorf("classteachers.Service.ListByTeacher: %w", ErrInvalidInput)
	}
	items, err := s.repo.ListByTeacher(ctx, userID, tenantID)
	if err != nil {
		return nil, fmt.Errorf("classteachers.Service.ListByTeacher: %w", err)
	}
	return &ClassTeacherListResponse{
		Items: items,
		Total: len(items),
		Page:  1,
		Limit: len(items),
	}, nil
}

// ── DELETE ───────────────────────────────────────────────────────────────

// Delete removes a class teacher assignment.
func (s *Service) Delete(ctx context.Context, id, tenantID string) error {
	if id == "" || tenantID == "" {
		return fmt.Errorf("classteachers.Service.Delete: %w", ErrInvalidInput)
	}
	return s.repo.Delete(ctx, id, tenantID)
}
