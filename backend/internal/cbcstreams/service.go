package cbcstreams

import (
	"context"
	"fmt"
)

// Service contains business logic for the cbcstreams domain.
type Service struct {
	Repo Repository
}

// NewService creates a new Service.
func NewService(repo Repository) *Service {
	return &Service{Repo: repo}
}

// ListStreams returns all streams for a tenant and school.
func (s *Service) ListStreams(ctx context.Context, tenantID, schoolID string) (*StreamListResult, error) {
	if tenantID == "" || schoolID == "" {
		return nil, fmt.Errorf("cbcstreams.Service.ListStreams: %w", ErrInvalidInput)
	}

	streams, err := s.Repo.List(ctx, tenantID, schoolID)
	if err != nil {
		return nil, fmt.Errorf("cbcstreams.Service.ListStreams: %w", err)
	}

	return &StreamListResult{
		Items: streams,
		Total: len(streams),
	}, nil
}
