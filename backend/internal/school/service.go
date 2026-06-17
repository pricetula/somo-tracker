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

// ListByTenant returns all active schools for a tenant.
func (s *Service) ListByTenant(ctx context.Context, tenantID string) ([]School, error) {
	return s.repo.ListByTenant(ctx, tenantID)
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
