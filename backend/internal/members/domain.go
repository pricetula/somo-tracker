package members

import "time"

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

// ErrorBody is the JSON error response body.
type ErrorBody struct {
	Error   string `json:"error"`
	Message string `json:"message,omitempty"`
}
