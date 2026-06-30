package members

import (
	"context"
	"errors"
	"time"
)

// Sentinel domain errors.
var (
	ErrNotFound      = errors.New("members not found")
	ErrAlreadyExists = errors.New("members already exists")
	ErrInvalidInput  = errors.New("invalid members input")
	ErrUnauthorized  = errors.New("unauthorized")
	ErrForbidden     = errors.New("forbidden")
	ErrConflict      = errors.New("members conflict")
)

// Repository defines the contract for member persistence.
// Used by invitations.SchoolResolver and imports.SchoolResolver.
type Repository interface {
	ListByRole(ctx context.Context, tenantID, schoolID, role string, offset, limit int, search string) ([]Member, int, error)
	GetActiveSchoolID(ctx context.Context, tenantID, userID string) (string, error)
}

// ─── Member (user + membership join) ──────────────────────────────────────

// Member represents a user with an active membership in a school.
type Member struct {
	ID        string    `json:"id"`
	Email     string    `json:"email"`
	FullName  string    `json:"full_name"`
	Role      string    `json:"role"`
	IsActive  bool      `json:"is_active"`
	CreatedAt time.Time `json:"created_at"`
}

// ─── HTTP types ───────────────────────────────────────────────────────────

// ListResponse wraps a paginated member list.
type ListResponse struct {
	Members []Member `json:"members"`
	Total   int      `json:"total"`
}

// ToggleActiveRequest is the payload for activating/deactivating a member.
type ToggleActiveRequest struct {
	IsActive bool `json:"is_active"`
}
