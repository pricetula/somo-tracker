package auth

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"
)

// ============================================================================
// Domain Errors — typed sentinels for the error taxonomy (requirement 14).
// ============================================================================

var (
	ErrInvalidInput              = errors.New("invalid_input")
	ErrExpiredToken              = errors.New("expired_token")
	ErrMFARequired               = errors.New("mfa_required")
	ErrOrgAlreadyExists          = errors.New("org_already_exists")
	ErrJITProvisioningNotAllowed = errors.New("jit_provisioning_not_allowed")
	ErrMemberNotFound            = errors.New("member_not_found")
	ErrOrgNotFound               = errors.New("org_not_found")
	ErrNotFound                  = errors.New("not_found")
	ErrAlreadyExists             = errors.New("already exists")
	ErrUnauthorized              = errors.New("unauthorized")
	ErrForbidden                 = errors.New("forbidden")
	ErrConflict                  = errors.New("conflict")
	ErrInternal                  = errors.New("internal_error")
)

// ValidationError carries a user-facing message alongside the sentinel.
type ValidationError struct {
	Err     error
	Message string
}

func (e *ValidationError) Error() string { return e.Message }

func (e *ValidationError) Unwrap() error { return e.Err }

// ============================================================================
// Domain Models
// ============================================================================

// UserSession represents an authenticated browser session.
type UserSession struct {
	ID                 string    `json:"id"`
	Token              string    `json:"-"`
	UserID             string    `json:"user_id"`
	TenantID           string    `json:"tenant_id"`
	Role               string    `json:"role"`
	StytchMemberID     string    `json:"-"`
	StytchOrgID        string    `json:"-"`
	StytchSessionToken string    `json:"-"`
	DeviceFingerprint  string    `json:"-"`
	ExpiresAt          time.Time `json:"expires_at"`
	CreatedAt          time.Time `json:"created_at"`
}

// DiscoveryPayload is sent by the frontend to initiate the magic-link flow.
type DiscoveryPayload struct {
	Email string `json:"email"`
}

// RegistrationPayload is submitted after the user clicks the magic link.
type RegistrationPayload struct {
	SchoolName string `json:"school_name"`
	SessionRef string `json:"session_ref"`
	FirstName  string `json:"first_name"`
	LastName   string `json:"last_name"`
}

var uuidV4Regex = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`)

// Validate checks payload rules per requirement 13.
func (p *RegistrationPayload) Validate() error {
	// Trim whitespace from school_name
	p.SchoolName = strings.TrimSpace(p.SchoolName)

	if p.SchoolName == "" {
		return &ValidationError{Err: ErrInvalidInput, Message: "school_name is required"}
	}
	if len([]rune(p.SchoolName)) < 2 || len([]rune(p.SchoolName)) > 100 {
		return &ValidationError{Err: ErrInvalidInput, Message: "school_name must be between 2 and 100 characters"}
	}
	if !isPrintableUTF8(p.SchoolName) {
		return &ValidationError{Err: ErrInvalidInput, Message: "school_name must contain only printable UTF-8 characters"}
	}

	if p.SessionRef == "" {
		return &ValidationError{Err: ErrInvalidInput, Message: "session_ref is required"}
	}
	if !uuidV4Regex.MatchString(p.SessionRef) {
		return &ValidationError{Err: ErrInvalidInput, Message: "session_ref must be a valid UUID v4"}
	}

	return nil
}

func isPrintableUTF8(s string) bool {
	for _, r := range s {
		if r < 0x20 || r == 0x7F {
			return false
		}
	}
	return true
}

// ExchangeResult is the clean domain result from exchanging an IST.
type ExchangeResult struct {
	MemberAuthenticated bool
	StytchSessionToken  string
	MemberID            string
	OrganizationID      string
}

// ============================================================================
// IdentityProvider interface — abstracts Stytch B2B (requirement 1).
// ============================================================================

// IdentityProvider defines the contract for authentication provider operations.
// All methods accept context.Context as first parameter (requirement 10).
type IdentityProvider interface {
	// SendDiscoveryEmail dispatches a magic link to the given email.
	SendDiscoveryEmail(ctx context.Context, email string) error

	// AuthenticateDiscoveryToken validates a magic-link token and returns
	// the raw Intermediate Session Token (IST) and the verified email address.
	AuthenticateDiscoveryToken(ctx context.Context, token string) (ist, email string, err error)

	// CreateOrganization provisions a new organization in the identity provider.
	CreateOrganization(ctx context.Context, name string) (orgID string, err error)

	// ExchangeIntermediateSession exchanges an IST for a full session within
	// the context of a specific organization. Returns MemberAuthenticated
	// status for MFA enforcement (requirement 3).
	ExchangeIntermediateSession(ctx context.Context, ist, orgID string) (ExchangeResult, error)

	// CreateMember provisions a new member in an existing Stytch organization.
	CreateMember(ctx context.Context, orgID, email, name string) (memberID string, err error)

	// InviteMemberByEmail sends a Stytch invite email to join an organization.
	// Returns the Stytch member ID of the invited member.
	InviteMemberByEmail(ctx context.Context, orgID, email, name, redirectURL string) (memberID string, err error)
}

// ============================================================================
// Repository interface — abstracts Postgres persistence (requirement 9, 12).
// ============================================================================

// CreateTenantParams holds input for creating a new tenant row.
type CreateTenantParams struct {
	Name        string
	Slug        string
	StytchOrgID string
}

// CreateUserParams holds input for creating a new user row.
type CreateUserParams struct {
	Email          string
	TenantID       string
	FirstName      string
	LastName       string
	ExternalAuthID string
}

// CreateSessionParams holds input for creating a new session row.
type CreateSessionParams struct {
	Token              string
	UserID             string
	TenantID           string
	StytchMemberID     string
	StytchOrgID        string
	StytchSessionToken string
	DeviceFingerprint  string
	ExpiresAt          time.Time
}

// Repository defines the contract for database persistence.
// All methods accept context.Context as first parameter (requirement 10).
type Repository interface {
	// TenantExists checks if a tenant already exists with the given Stytch org ID.
	TenantExists(ctx context.Context, orgID string) (bool, error)

	// TenantExistsByName checks if a tenant already exists with the given school name.
	TenantExistsByName(ctx context.Context, name string) (bool, error)

	// GetTenantByName retrieves an existing tenant's ID and Stytch org ID by name.
	GetTenantByName(ctx context.Context, name string) (string, string, error)

	// GetSessionByToken retrieves a session by its opaque token.
	GetSessionByToken(ctx context.Context, token string) (*UserSession, error)

	// DeleteSession removes a session record by token.
	DeleteSession(ctx context.Context, token string) error

	// CreateTenantUserSession creates a tenant, user, and session inside a
	// single database transaction (requirement 9). Returns the user ID and
	// any error. On Stytch-org-created-but-Postgres-failure, logs the
	// stytch_org_id at WARN level.
	CreateTenantUserSession(
		ctx context.Context,
		tenantParams CreateTenantParams,
		userParams CreateUserParams,
		sessionParams CreateSessionParams,
	) (userID string, tenantID string, err error)

	// CreateUserSession creates a user and session inside a single transaction
	// for an existing tenant (no tenant insert). Returns the user ID.
	CreateUserSession(ctx context.Context, userParams CreateUserParams, sessionParams CreateSessionParams) (userID string, err error)

	// CreateSchool creates a new cbc_school for a tenant and returns its ID.
	CreateSchool(ctx context.Context, tenantID string, name string) (schoolID string, err error)

	// CreateMembership creates a membership linking a user to a school with a role.
	CreateMembership(ctx context.Context, userID, schoolID, tenantID, role string) error

	// GetMeInfo returns the full profile info for /me: user details, role,
	// and the active school.
	GetMeInfo(ctx context.Context, token string) (*MeInfo, error)
}

// MeInfo is the result of GetMeInfo.
type MeInfo struct {
	UserID     string
	TenantID   string
	Role       string
	SchoolID   string
	SchoolName string
	FirstName  string
	LastName   string
	Email      string
}

// StytchOrgIDKey is the context key used to pass the stytch_org_id through
// to the repository for reconciliation logging.
type StytchOrgIDKey struct{}

// Stringer for domain errors — used in structured logging.
func ErrorToCode(err error) string {
	switch {
	case errors.Is(err, ErrInvalidInput):
		return "invalid_input"
	case errors.Is(err, ErrExpiredToken):
		return "expired_token"
	case errors.Is(err, ErrMFARequired):
		return "mfa_required"
	case errors.Is(err, ErrOrgAlreadyExists):
		return "org_already_exists"
	case errors.Is(err, ErrJITProvisioningNotAllowed):
		return "jit_provisioning_not_allowed"
	case errors.Is(err, ErrMemberNotFound):
		return "member_not_found"
	case errors.Is(err, ErrOrgNotFound):
		return "org_not_found"
	case errors.Is(err, ErrNotFound):
		return "not_found"
	case errors.Is(err, ErrInternal):
		return "internal_error"
	default:
		return fmt.Sprintf("unknown: %v", err)
	}
}
