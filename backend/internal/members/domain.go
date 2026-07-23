package members

import (
	"context"
	"fmt"
	"time"

	"somotracker/backend/internal/middleware"
)

// Sentinel domain errors, each wrapping the corresponding middleware sentinel
// so that middleware.HTTPError can match them via errors.Is.
var (
	ErrNotFound      = fmt.Errorf("members not found: %w", middleware.ErrNotFound)
	ErrAlreadyExists = fmt.Errorf("members already exists: %w", middleware.ErrAlreadyExists)
	ErrInvalidInput  = fmt.Errorf("invalid members input: %w", middleware.ErrInvalidInput)
	ErrUnauthorized  = fmt.Errorf("unauthorized: %w", middleware.ErrUnauthorized)
	ErrForbidden     = fmt.Errorf("forbidden: %w", middleware.ErrForbidden)
	ErrConflict      = fmt.Errorf("members conflict: %w", middleware.ErrConflict)
)

// Repository defines the contract for member persistence.
// Used by invitations.SchoolResolver and imports.SchoolResolver.
type Repository interface {
	ListByRole(ctx context.Context, tenantID, schoolID, role string, offset, limit int, search string) ([]Member, int, error)
	GetActiveSchoolID(ctx context.Context, tenantID, userID string) (string, error)
}

// UpdateMemberPayload is the payload for updating a member's profile.
type UpdateMemberPayload struct {
	FullName *string `json:"full_name,omitempty"`
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
	Items []Member `json:"items"`
	Total int      `json:"total"`
	Page  int      `json:"page"`
	Limit int      `json:"limit"`
}

// ToggleActiveRequest is the payload for activating/deactivating a member.
type ToggleActiveRequest struct {
	IsActive bool `json:"is_active"`
}
