package members

import (
	"context"
	"time"

	"somotracker/backend/internal/xerrors"
)

// Sentinel domain errors. Each is a *xerrors.DomainError carrying the
// correct HTTP status and machine-readable code.
var (
	ErrNotFound      = xerrors.NotFound("member not found")
	ErrAlreadyExists = xerrors.AlreadyExists("member already exists")
	ErrInvalidInput  = xerrors.InvalidInput("invalid member input")
	ErrUnauthorized  = xerrors.Unauthorized("unauthorized")
	ErrForbidden     = xerrors.Forbidden("forbidden")
	ErrConflict      = xerrors.Conflict("member conflict")
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
