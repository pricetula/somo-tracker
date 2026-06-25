package cbcschools

import (
	"context"
	"fmt"
)

// Service contains business logic for the cbcschools domain.
type Service struct {
	Repo Repository
}

// NewService creates a new Service.
func NewService(repo Repository) *Service {
	return &Service{Repo: repo}
}

// CreateSchool creates a new school and returns its ID.
func (s *Service) CreateSchool(ctx context.Context, tenantID string, name string) (string, error) {
	if name == "" {
		return "", fmt.Errorf("cbcschools.Service.CreateSchool: %w", ErrInvalidInput)
	}
	return s.Repo.Create(ctx, tenantID, name)
}

// ListSchoolsByTenantID returns all schools for a tenant with member counts
// and whether each school is the user's currently active school.
func (s *Service) ListSchoolsByTenantID(ctx context.Context, tenantID, userID string) ([]SchoolWithMemberCount, error) {
	if tenantID == "" {
		return nil, fmt.Errorf("cbcschools.Service.ListSchoolsByTenantID: %w", ErrInvalidInput)
	}
	return s.Repo.ListByTenantID(ctx, tenantID, userID)
}

// UpdateSchool applies partial updates to a school.
func (s *Service) UpdateSchool(ctx context.Context, school SchoolUpdateFields) error {
	if school.ID == "" {
		return fmt.Errorf("cbcschools.Service.UpdateSchool: %w", ErrInvalidInput)
	}
	// Ensure at least one field is being updated
	if school.Name == nil && school.County == nil && school.SubCounty == nil &&
		school.Ward == nil && school.KnecSchoolCode == nil && school.NemisCode == nil &&
		school.SchoolType == nil && school.IsActive == nil {
		return fmt.Errorf("cbcschools.Service.UpdateSchool: %w", ErrInvalidInput)
	}
	return s.Repo.Update(ctx, school)
}

// DeleteSchool removes a school by ID.
func (s *Service) DeleteSchool(ctx context.Context, id string) error {
	if id == "" {
		return fmt.Errorf("cbcschools.Service.DeleteSchool: %w", ErrInvalidInput)
	}
	return s.Repo.Delete(ctx, id)
}
