package invitations

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
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

	// BulkInvite repository methods
	CheckExistingEmails(ctx context.Context, tenantID, schoolID string, emails []string) (existingInUsers, existingInInvitations []string, err error)
	InsertInvitation(ctx context.Context, tx pgx.Tx, params InsertInvitationParams) error
	GetStytchOrgID(ctx context.Context, tenantID string) (string, error)
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

// ============================================================================
// Bulk Invitation Domain Types
// ============================================================================

// InviteRow is a single row in a bulk invitation request.
type InviteRow struct {
	Email    string  `json:"email"`
	FullName *string `json:"full_name,omitempty"`
}

// BulkInviteRequest is the request body for POST /api/v1/staff/invite.
type BulkInviteRequest struct {
	Role string      `json:"role"`
	Rows []InviteRow `json:"rows"`
}

// BulkInviteResponse is returned immediately after creating the bulk invite job.
type BulkInviteResponse struct {
	JobID        string `json:"job_id"`
	TotalRecords int    `json:"total_records"`
	TotalChunks  int    `json:"total_chunks"`
	Status       string `json:"status"`
	IsReplay     bool   `json:"is_replay,omitempty"`
}

// InsertInvitationParams holds parameters for inserting a new invitation record.
type InsertInvitationParams struct {
	Email          string
	FullName       string
	TenantID       string
	SchoolID       string
	Role           string
	InvitedBy      string
	Status         string
	StytchMemberID string
	ExpiresAt      time.Time
	ImportJobID    string
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
	Items []Invitation `json:"items"`
	Total int          `json:"total"`
	Page  int          `json:"page"`
	Limit int          `json:"limit"`
}
