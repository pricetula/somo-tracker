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

// Repository defines the contract for member and invitation persistence.
type Repository interface {
	ListByRole(ctx context.Context, tenantID, schoolID, role string, offset, limit int, search string) ([]Member, int, error)
	GetActiveSchoolID(ctx context.Context, tenantID, userID string) (string, error)
	ListInvitations(ctx context.Context, tenantID, schoolID string, filter ListInvitationsFilter) ([]Invitation, int, error)
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

// ─── Invitation ───────────────────────────────────────────────────────────

// Invitation represents a pending/accepted/expired/revoked invitation.
type Invitation struct {
	ID        string    `json:"id"`
	SchoolID  string    `json:"school_id"`
	TenantID  string    `json:"tenant_id"`
	Email     string    `json:"email"`
	Role      string    `json:"role"`
	Status    string    `json:"status"`
	FullName  *string   `json:"full_name,omitempty"`
	ExpiresAt time.Time `json:"expires_at"`
	CreatedAt time.Time `json:"created_at"`
}

// ─── HTTP types ───────────────────────────────────────────────────────────

// ListResponse wraps a paginated member list.
type ListResponse struct {
	Members []Member `json:"members"`
	Total   int      `json:"total"`
}

// ─── Invitation HTTP types ──────────────────────────────────────────────

// ListInvitationsFilter defines filters for listing invitations.
type ListInvitationsFilter struct {
	Search  string
	Email   string
	Status  string
	Role    string
	Expired bool
	Offset  int
	Limit   int
}

// ListInvitationsResponse wraps a paginated invitation list.
type ListInvitationsResponse struct {
	Invitations []Invitation `json:"invitations"`
	Total       int          `json:"total"`
}
