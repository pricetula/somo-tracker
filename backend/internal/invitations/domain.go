package invitations

import (
	"context"
	"errors"
	"time"
)

// SchoolResolver resolves the active school for an authenticated user.
// Declared at the consumer side per the architecture contract.
type SchoolResolver interface {
	GetActiveSchoolID(ctx context.Context, tenantID, userID string) (string, error)
}

// Sentinel domain errors.
var (
	ErrNotFound      = errors.New("invitations not found")
	ErrAlreadyExists = errors.New("invitations already exists")
	ErrInvalidInput  = errors.New("invalid invitations input")
	ErrUnauthorized  = errors.New("unauthorized")
	ErrForbidden     = errors.New("forbidden")
	ErrConflict      = errors.New("invitations conflict")
)

// Repository defines the contract for invitation persistence.
type Repository interface {
	ListInvitations(ctx context.Context, tenantID, schoolID string, filter ListInvitationsFilter) ([]Invitation, int, error)
}

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
