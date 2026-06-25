package invitations

import (
	"context"
)

// Service contains business logic for the invitations domain.
type Service struct {
	repo Repository
}

// NewService creates a new Service.
func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

// ListInvitations returns paginated invitations with optional filters.
func (s *Service) ListInvitations(ctx context.Context, tenantID, schoolID string, filter ListInvitationsFilter) ([]Invitation, int, error) {
	if filter.Limit <= 0 || filter.Limit > 100 {
		filter.Limit = 50
	}
	if filter.Offset < 0 {
		filter.Offset = 0
	}
	return s.repo.ListInvitations(ctx, tenantID, schoolID, filter)
}
