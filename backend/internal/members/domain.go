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
	GetPendingInviteByEmail(ctx context.Context, schoolID, email string) (*Invitation, error)
	GetMemberByEmail(ctx context.Context, schoolID, email string) (*Member, error)
	GetTenantStytchOrgID(ctx context.Context, tenantID string) (string, error)
	CreateInvitation(ctx context.Context, inv *Invitation, invitedBy string) error
	SetInvitationStytchMemberID(ctx context.Context, id, stytchMemberID string) error
}

// ─── Member (user + membership join) ──────────────────────────────────────

// Member represents a user with an active membership in a school.
type Member struct {
	ID        string    `json:"id"`
	Email     string    `json:"email"`
	FirstName string    `json:"first_name"`
	LastName  string    `json:"last_name"`
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
	FirstName *string   `json:"first_name,omitempty"`
	LastName  *string   `json:"last_name,omitempty"`
	ExpiresAt time.Time `json:"expires_at"`
	CreatedAt time.Time `json:"created_at"`
}

// ─── HTTP types ───────────────────────────────────────────────────────────

// ListResponse wraps a paginated member list.
type ListResponse struct {
	Members []Member `json:"members"`
	Total   int      `json:"total"`
}

// BulkInviteRequest is the request body for POST /api/v1/members/invite.
type BulkInviteRequest struct {
	Role    string       `json:"role"`
	Invites []InviteItem `json:"invites"`
}

// InviteItem is a single invite entry.
type InviteItem struct {
	Email     string `json:"email"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
}

// BulkInviteResponse is the response for a bulk invite.
type BulkInviteResponse struct {
	Sent   int               `json:"sent"`
	Failed int               `json:"failed"`
	Errors []InviteErrorItem `json:"errors,omitempty"`
}

// InviteErrorItem captures a per-invite failure.
type InviteErrorItem struct {
	Email string `json:"email"`
	Error string `json:"error"`
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

// CreateInvitationsRequest is the request body for POST /api/v1/invitations.
type CreateInvitationsRequest struct {
	Invites []CreateInviteItem `json:"invites"`
}

// CreateInviteItem is a single invite entry for the new endpoint.
type CreateInviteItem struct {
	Email     string `json:"email"`
	FirstName string `json:"first_name,omitempty"`
	LastName  string `json:"last_name,omitempty"`
	Role      string `json:"role"`
}

// ErrorBody is the JSON error response body.
type ErrorBody struct {
	Error   string `json:"code"`
	Message string `json:"message,omitempty"`
}
