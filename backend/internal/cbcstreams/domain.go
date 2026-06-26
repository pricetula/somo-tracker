package cbcstreams

import (
	"context"
	"fmt"
	"time"

	"somotracker/backend/internal/middleware"
)

// Sentinel domain errors.
var (
	ErrNotFound      = fmt.Errorf("cbcstreams not found: %w", middleware.ErrNotFound)
	ErrAlreadyExists = fmt.Errorf("cbcstreams already exists: %w", middleware.ErrAlreadyExists)
	ErrInvalidInput  = fmt.Errorf("invalid cbcstreams input: %w", middleware.ErrInvalidInput)
	ErrUnauthorized  = fmt.Errorf("unauthorized: %w", middleware.ErrUnauthorized)
	ErrForbidden     = fmt.Errorf("forbidden: %w", middleware.ErrForbidden)
	ErrConflict      = fmt.Errorf("cbcstreams conflict: %w", middleware.ErrConflict)
	ErrStreamInUse   = fmt.Errorf("cbcstreams in use: %w", middleware.ErrConflict)
)

// Repository defines the contract for stream persistence.
type Repository interface {
	List(ctx context.Context, tenantID, schoolID string) ([]Stream, error)
	GetByID(ctx context.Context, id, tenantID, schoolID string) (*Stream, error)
	Create(ctx context.Context, tenantID, schoolID, name string) (*Stream, error)
	Update(ctx context.Context, id, tenantID, schoolID, name string) (*Stream, error)
	Delete(ctx context.Context, id, tenantID, schoolID string) error
	HasReferencingClasses(ctx context.Context, id, tenantID, schoolID string) (bool, error)
}

// Stream represents a named stream within a school.
type Stream struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// CreateStreamPayload is the request body for POST /api/v1/streams.
type CreateStreamPayload struct {
	Name string `json:"name"`
}

// UpdateStreamPayload is the request body for PUT /api/v1/streams/:id.
type UpdateStreamPayload struct {
	Name string `json:"name"`
}

// ListStreamsResponse wraps a list of streams.
type ListStreamsResponse struct {
	Data []Stream `json:"data"`
}
