package cbcstreams

import (
	"context"
	"errors"
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

// GetStreamByID returns a single stream by ID.
func (s *Service) GetStreamByID(ctx context.Context, id, tenantID, schoolID string) (*Stream, error) {
	if id == "" || tenantID == "" || schoolID == "" {
		return nil, fmt.Errorf("cbcstreams.Service.GetStreamByID: %w", ErrInvalidInput)
	}
	return s.Repo.GetByID(ctx, id, tenantID, schoolID)
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

// CreateStream creates a new stream.
func (s *Service) CreateStream(ctx context.Context, tenantID, schoolID, name, color string) (*Stream, error) {
	if tenantID == "" || schoolID == "" || name == "" {
		return nil, fmt.Errorf("cbcstreams.Service.CreateStream: %w", ErrInvalidInput)
	}

	stream, err := s.Repo.Create(ctx, tenantID, schoolID, name, color)
	if err != nil {
		return nil, fmt.Errorf("cbcstreams.Service.CreateStream: %w", err)
	}
	return stream, nil
}

// UpdateStream updates a stream's name and color.
func (s *Service) UpdateStream(ctx context.Context, id, tenantID, schoolID, name, color string) (*Stream, error) {
	if id == "" || tenantID == "" || schoolID == "" || name == "" {
		return nil, fmt.Errorf("cbcstreams.Service.UpdateStream: %w", ErrInvalidInput)
	}

	stream, err := s.Repo.Update(ctx, id, tenantID, schoolID, name, color)
	if err != nil {
		return nil, fmt.Errorf("cbcstreams.Service.UpdateStream: %w", err)
	}
	return stream, nil
}

// DeleteStream deletes a stream by ID.
// Returns ErrStreamHasActiveEnrollments (wrapping the stream name) if the
// stream has classes with active student enrollments.
func (s *Service) DeleteStream(ctx context.Context, id, tenantID, schoolID string) error {
	if id == "" || tenantID == "" || schoolID == "" {
		return fmt.Errorf("cbcstreams.Service.DeleteStream: %w", ErrInvalidInput)
	}

	// Fetch the stream first so we can include its name in error messages.
	stream, err := s.Repo.GetByID(ctx, id, tenantID, schoolID)
	if err != nil {
		return fmt.Errorf("cbcstreams.Service.DeleteStream: %w", err)
	}

	if err := s.Repo.Delete(ctx, id, tenantID, schoolID); err != nil {
		if errors.Is(err, ErrStreamHasActiveEnrollments) {
			return fmt.Errorf(
				"cannot delete stream %q: this stream is currently active and contains enrolled students in one or more classes. Please reassign or unenroll the students before deleting this stream: %w",
				stream.Name, ErrStreamHasActiveEnrollments,
			)
		}
		return fmt.Errorf("cbcstreams.Service.DeleteStream: %w", err)
	}
	return nil
}
