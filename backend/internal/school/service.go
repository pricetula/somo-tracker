package school

import (
	"context"
	"errors"
	"fmt"
)

// ErrNameAlreadyExists is returned when a school with the same name already exists in the tenant.
var ErrNameAlreadyExists = errors.New("a school with this name already exists in your tenant")

// Service contains business logic for school operations.
type Service struct {
	repo *SqlcRepository
}

// NewService creates a new Service.
func NewService(repo *SqlcRepository) *Service {
	return &Service{repo: repo}
}

// GetByID returns a school by ID.
func (s *Service) GetByID(ctx context.Context, id string) (*School, error) {
	return s.repo.GetByID(ctx, id)
}

// ListByTenant returns all active schools for a tenant.
func (s *Service) ListByTenant(ctx context.Context, tenantID string) ([]School, error) {
	return s.repo.ListByTenant(ctx, tenantID)
}

// ActivateSchool switches the user's active school.
// Deactivates all memberships and activates the target one.
func (s *Service) ActivateSchool(ctx context.Context, userID, schoolID, tenantID string) error {
	// Verify the school belongs to this tenant
	school, err := s.repo.GetByID(ctx, schoolID)
	if err != nil {
		return err
	}
	if school == nil {
		return fmt.Errorf("school not found")
	}
	if school.TenantID != tenantID {
		return fmt.Errorf("school does not belong to this tenant")
	}

	return s.repo.ActivateSchoolMembership(ctx, userID, schoolID, tenantID)
}

// UpdateSchoolName updates a school's name. Only the school's tenant can update it.
func (s *Service) UpdateSchoolName(ctx context.Context, id, tenantID, name string) (*School, error) {
	// Verify the school belongs to this tenant
	school, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if school == nil {
		return nil, fmt.Errorf("school not found")
	}
	if school.TenantID != tenantID {
		return nil, fmt.Errorf("school does not belong to this tenant")
	}

	// Check for duplicate name within the tenant
	existing, err := s.repo.ListByTenant(ctx, tenantID)
	if err != nil {
		return nil, fmt.Errorf("check existing schools: %w", err)
	}
	for _, sch := range existing {
		if sch.Name == name && sch.ID != id {
			return nil, ErrNameAlreadyExists
		}
	}

	return s.repo.UpdateName(ctx, id, name)
}

// DeleteSchool soft-deletes a school. Only the school's tenant can delete it.
func (s *Service) DeleteSchool(ctx context.Context, id, tenantID string) error {
	// Verify the school belongs to this tenant
	school, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if school == nil {
		return fmt.Errorf("school not found")
	}
	if school.TenantID != tenantID {
		return fmt.Errorf("school does not belong to this tenant")
	}

	return s.repo.Delete(ctx, id)
}

// CreateSchool creates a new school and assigns the creator as SCHOOL_ADMIN.
func (s *Service) CreateSchool(ctx context.Context, tenantID, name, educationSystemID, userID string) (*School, error) {
	// Check for duplicate name within the tenant
	existing, err := s.repo.ListByTenant(ctx, tenantID)
	if err != nil {
		return nil, fmt.Errorf("check existing schools: %w", err)
	}
	for _, sch := range existing {
		if sch.Name == name {
			return nil, ErrNameAlreadyExists
		}
	}

	// Create school and membership in a single transaction
	school, err := s.repo.CreateSchoolAndMembership(ctx, tenantID, name, educationSystemID, userID, "SCHOOL_ADMIN")
	if err != nil {
		return nil, err
	}

	return school, nil
}
