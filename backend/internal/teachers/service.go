package teachers

import (
	"context"
	"fmt"
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

// ToggleActive toggles the active status of a teacher's membership.
// If the teacher is not found, returns ErrNotFound.
func (s *Service) ToggleActive(ctx context.Context, tenantID, schoolID, userID string, isActive bool) error {
	if userID == "" {
		return fmt.Errorf("teachers.Service.ToggleActive: %w", ErrInvalidInput)
	}
	return s.repo.ToggleActive(ctx, tenantID, schoolID, userID, isActive)
}
