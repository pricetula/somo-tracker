package activeschool

import (
	"context"
	"fmt"
)

// Service contains business logic for the activeschool domain.
type Service struct {
	repo Repository
}

// NewService creates a new Service.
func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

// SwitchActiveSchool upserts the active school for a user.
// This is used when a user switches their current school context, or when
// a new school is created and the user wants to immediately switch to it.
func (s *Service) SwitchActiveSchool(ctx context.Context, tenantID, userID, schoolID string) error {
	if tenantID == "" {
		return fmt.Errorf("activeschool.Service.SwitchActiveSchool: %w", ErrInvalidInput)
	}
	if userID == "" {
		return fmt.Errorf("activeschool.Service.SwitchActiveSchool: %w", ErrInvalidInput)
	}
	if schoolID == "" {
		return fmt.Errorf("activeschool.Service.SwitchActiveSchool: %w", ErrInvalidInput)
	}
	return s.repo.Upsert(ctx, tenantID, userID, schoolID)
}

// GetActiveSchoolID returns the active school ID for a user in a tenant.
func (s *Service) GetActiveSchoolID(ctx context.Context, tenantID, userID string) (string, error) {
	if tenantID == "" {
		return "", fmt.Errorf("activeschool.Service.GetActiveSchoolID: %w", ErrInvalidInput)
	}
	if userID == "" {
		return "", fmt.Errorf("activeschool.Service.GetActiveSchoolID: %w", ErrInvalidInput)
	}
	return s.repo.GetActiveSchoolID(ctx, tenantID, userID)
}
