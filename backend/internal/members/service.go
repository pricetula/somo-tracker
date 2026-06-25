package members

import (
	"context"
)

// Service contains business logic for the members domain.
type Service struct {
	repo Repository
}

// NewService creates a new Service.
func NewService(repo Repository) *Service {
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
