package cbcschools

import (
	"context"
	"fmt"
)

// Service contains business logic for the cbcschools domain.
type Service struct {
	repo Repository
}

// NewService creates a new Service.
func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

// CreateSchool creates a new school and returns its ID.
func (s *Service) CreateSchool(ctx context.Context, tenantID string, name string) (string, error) {
	if name == "" {
		return "", fmt.Errorf("cbcschools.Service.CreateSchool: %w", ErrInvalidInput)
	}
	return s.repo.Create(ctx, tenantID, name)
}
