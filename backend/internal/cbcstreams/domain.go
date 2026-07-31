package cbcstreams

import (
	"context"
	"time"

	"somotracker/backend/internal/xerrors"
)

// Sentinel domain errors.
var (
	ErrNotFound      = xerrors.NotFound("cbcstream not found")
	ErrAlreadyExists = xerrors.AlreadyExists("cbcstream already exists")
	ErrInvalidInput  = xerrors.InvalidInput("invalid cbcstream input")
	ErrUnauthorized  = xerrors.Unauthorized("unauthorized")
	ErrForbidden     = xerrors.Forbidden("forbidden")
	ErrConflict      = xerrors.Conflict("cbcstream conflict")
	// ErrStreamHasActiveEnrollments is returned when attempting to delete a stream
	// that has classes with active student enrollments.
	ErrStreamHasActiveEnrollments = xerrors.Conflict("cbcstream has active enrollments")
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

	// HasActiveEnrollments checks whether any class referencing this stream has
	// active student enrollments in the current term.
	HasActiveEnrollments(ctx context.Context, id, tenantID, schoolID string) (bool, error)
}
