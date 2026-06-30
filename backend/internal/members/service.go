package members

import (
	"context"
	"fmt"
)

// ServiceRepository defines the repository methods needed by the Service.
// Tests can mock this interface without depending on PgRepository.
type ServiceRepository interface {
	ListByRole(ctx context.Context, tenantID, schoolID, role string, offset, limit int, search string) ([]Member, int, error)
	ListByRoleIncludingInactive(ctx context.Context, tenantID, schoolID, role string, offset, limit int, search string) ([]Member, int, error)
	ToggleActive(ctx context.Context, tenantID, schoolID, userID string, isActive bool) error
}

// Service contains business logic for the members domain.
type Service struct {
	repo ServiceRepository
}

// NewService creates a new Service.
func NewService(repo ServiceRepository) *Service {
	return &Service{repo: repo}
}

// ListMembers returns paginated members filtered by role.
func (s *Service) ListMembers(ctx context.Context, tenantID, schoolID, role string, offset, limit int, search string) ([]Member, int, error) {
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	if offset < 0 {
		offset = 0
	}
	return s.repo.ListByRole(ctx, tenantID, schoolID, role, offset, limit, search)
}

// ListMembersIncludingInactive returns paginated members, including inactive ones.
func (s *Service) ListMembersIncludingInactive(ctx context.Context, tenantID, schoolID, role string, offset, limit int, search string) ([]Member, int, error) {
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	if offset < 0 {
		offset = 0
	}
	return s.repo.ListByRoleIncludingInactive(ctx, tenantID, schoolID, role, offset, limit, search)
}

// ToggleActive toggles the active status of a member's membership.
func (s *Service) ToggleActive(ctx context.Context, tenantID, schoolID, userID string, isActive bool) error {
	if userID == "" {
		return fmt.Errorf("members.Service.ToggleActive: %w", ErrInvalidInput)
	}
	return s.repo.ToggleActive(ctx, tenantID, schoolID, userID, isActive)
}
