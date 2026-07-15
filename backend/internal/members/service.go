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
	GetByID(ctx context.Context, userID, tenantID, schoolID string) (*Member, error)
	Update(ctx context.Context, userID, tenantID, schoolID string, payload UpdateMemberPayload) error
	ToggleActive(ctx context.Context, tenantID, schoolID, userID string, isActive bool) error
	Delete(ctx context.Context, tenantID, schoolID, userID, role string) error
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

// GetMemberByID returns a single member by user ID.
func (s *Service) GetMemberByID(ctx context.Context, userID, tenantID, schoolID string) (*Member, error) {
	if userID == "" {
		return nil, fmt.Errorf("members.Service.GetMemberByID: %w", ErrInvalidInput)
	}
	return s.repo.GetByID(ctx, userID, tenantID, schoolID)
}

// UpdateMember updates a member's profile.
func (s *Service) UpdateMember(ctx context.Context, userID, tenantID, schoolID string, payload UpdateMemberPayload) error {
	if userID == "" {
		return fmt.Errorf("members.Service.UpdateMember: %w", ErrInvalidInput)
	}
	if payload.FullName == nil {
		return fmt.Errorf("members.Service.UpdateMember: at least one field to update is required: %w", ErrInvalidInput)
	}
	return s.repo.Update(ctx, userID, tenantID, schoolID, payload)
}

// Delete hard-deletes a member's membership.
func (s *Service) Delete(ctx context.Context, tenantID, schoolID, userID, role string) error {
	if userID == "" {
		return fmt.Errorf("members.Service.Delete: %w", ErrInvalidInput)
	}
	return s.repo.Delete(ctx, tenantID, schoolID, userID, role)
}
