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
)

// Stream represents a named stream within a school.
type Stream struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Color     string    `json:"color"`
	CreatedAt time.Time `json:"created_at,omitempty"`
	UpdatedAt time.Time `json:"updated_at,omitempty"`
}

// StreamListResult holds the response for stream listing.
type StreamListResult struct {
	Items []Stream `json:"items"`
	Total int      `json:"total"`
}

// CreateStreamPayload is the request body for creating a stream.
type CreateStreamPayload struct {
	Name  string `json:"name"`
	Color string `json:"color"`
}

// UpdateStreamPayload is the request body for updating a stream.
type UpdateStreamPayload struct {
	Name  string `json:"name"`
	Color string `json:"color"`
}

// Repository defines the contract for stream persistence.
type Repository interface {
	List(ctx context.Context, tenantID, schoolID string) ([]Stream, error)
	Create(ctx context.Context, tenantID, schoolID, name, color string) (*Stream, error)
	GetByID(ctx context.Context, id, tenantID, schoolID string) (*Stream, error)
	Update(ctx context.Context, id, tenantID, schoolID, name, color string) (*Stream, error)
	Delete(ctx context.Context, id, tenantID, schoolID string) error
}
