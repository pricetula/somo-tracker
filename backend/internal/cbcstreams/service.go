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

// ListStreams returns all streams for the given tenant and school.
func (s *Service) ListStreams(ctx context.Context, tenantID, schoolID string) ([]Stream, error) {
	if tenantID == "" || schoolID == "" {
		return nil, fmt.Errorf("cbcstreams.Service.ListStreams: %w", ErrInvalidInput)
	}
	return s.Repo.List(ctx, tenantID, schoolID)
}

// CreateStream creates a new stream and returns it.
func (s *Service) CreateStream(ctx context.Context, tenantID, schoolID, name string) (*Stream, error) {
	if tenantID == "" || schoolID == "" {
		return nil, fmt.Errorf("cbcstreams.Service.CreateStream: %w", ErrInvalidInput)
	}
	if name == "" {
		return nil, fmt.Errorf("cbcstreams.Service.CreateStream: name is required: %w", ErrInvalidInput)
	}
	if len(name) > 100 {
		return nil, fmt.Errorf("cbcstreams.Service.CreateStream: name exceeds 100 characters: %w", ErrInvalidInput)
	}

	stream, err := s.Repo.Create(ctx, tenantID, schoolID, name)
	if err != nil {
		// Wrap unique constraint violations from the DB as conflict
		return nil, fmt.Errorf("cbcstreams.Service.CreateStream: %w", err)
	}
	return stream, nil
}

// UpdateStream updates a stream's name and returns the updated stream.
func (s *Service) UpdateStream(ctx context.Context, id, tenantID, schoolID, name string) (*Stream, error) {
	if id == "" || tenantID == "" || schoolID == "" {
		return nil, fmt.Errorf("cbcstreams.Service.UpdateStream: %w", ErrInvalidInput)
	}
	if name == "" {
		return nil, fmt.Errorf("cbcstreams.Service.UpdateStream: name is required: %w", ErrInvalidInput)
	}
	if len(name) > 100 {
		return nil, fmt.Errorf("cbcstreams.Service.UpdateStream: name exceeds 100 characters: %w", ErrInvalidInput)
	}

	stream, err := s.Repo.Update(ctx, id, tenantID, schoolID, name)
	if err != nil {
		return nil, fmt.Errorf("cbcstreams.Service.UpdateStream: %w", err)
	}
	return stream, nil
}

// DeleteStream removes a stream if it has no referencing classes.
func (s *Service) DeleteStream(ctx context.Context, id, tenantID, schoolID string) error {
	if id == "" || tenantID == "" || schoolID == "" {
		return fmt.Errorf("cbcstreams.Service.DeleteStream: %w", ErrInvalidInput)
	}

	// Pre-flight check: assert no cbc_classes row references this stream
	hasClasses, err := s.Repo.HasReferencingClasses(ctx, id, tenantID, schoolID)
	if err != nil {
		return fmt.Errorf("cbcstreams.Service.DeleteStream: %w", err)
	}
	if hasClasses {
		return fmt.Errorf("cbcstreams.Service.DeleteStream: stream is in use: %w", ErrStreamInUse)
	}

	if err := s.Repo.Delete(ctx, id, tenantID, schoolID); err != nil {
		return fmt.Errorf("cbcstreams.Service.DeleteStream: %w", err)
	}
	return nil
}
