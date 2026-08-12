package auth

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"somotracker/backend/internal/xerrors"
)

// ============================================================================
// Domain Errors — typed sentinels for the error taxonomy.
// ============================================================================

var (
	// Required sentinels (AGENTS.md contract).
	ErrNotFound      = xerrors.NotFound("auth not found")
	ErrAlreadyExists = xerrors.AlreadyExists("auth already exists")
	ErrInvalidInput  = xerrors.InvalidInput("invalid auth input")
	ErrUnauthorized  = xerrors.Unauthorized("unauthorized")
	ErrForbidden     = xerrors.Forbidden("forbidden")
	ErrConflict      = xerrors.Conflict("auth conflict")

	// Auth-specific sentinels.
	ErrExpiredToken              = xerrors.Unauthorized("expired_token")             // maps to 401, triggers frontend eviction
	ErrSessionRefExpired         = xerrors.Unauthorized("session_ref_expired")       // maps to 401 — IST already consumed/never cached
	ErrMFARequired               = xerrors.Unauthorized("mfa_required")              // maps to 401
	ErrOrgAlreadyExists          = xerrors.Conflict("org_already_exists")            // maps to 409
	ErrJITProvisioningNotAllowed = xerrors.Forbidden("jit_provisioning_not_allowed") // maps to 403
	ErrMemberNotFound            = xerrors.NotFound("member_not_found")              // maps to 404
	ErrOrgNotFound               = xerrors.NotFound("org_not_found")                 // maps to 404

	// ErrInternal is a recognized *xerrors.DomainError so that the HTTP
	// middleware can map it to a proper 500 response instead of falling
	// through to the "unclassified error" path.
	ErrInternal = &xerrors.DomainError{Code: "internal_error", Status: 500, Message: "an unexpected error occurred"}
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
	FullName   string `json:"full_name"`
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

	// FullName is forwarded to Stytch (CreateMember) and stored on users.full_name.
	// Empty or unbounded names would create junk identity records and could trip
	// Stytch's own length limits with a confusing 500 — validate here instead.
	p.FullName = strings.TrimSpace(p.FullName)
	if p.FullName == "" {
		return &ValidationError{Err: ErrInvalidInput, Message: "full_name is required"}
	}
	if len([]rune(p.FullName)) > 255 {
		return &ValidationError{Err: ErrInvalidInput, Message: "full_name must be at most 255 characters"}
	}
	if !isPrintableUTF8(p.FullName) {
		return &ValidationError{Err: ErrInvalidInput, Message: "full_name must contain only printable UTF-8 characters"}
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

// DiscoveredOrg represents a Stytch organization the user already belongs to.
type DiscoveredOrg struct {
	OrganizationID      string
	OrganizationName    string
	MemberID            string
	MemberName          string
	MemberAuthenticated bool
}

// VerifyResult tells the handler what to do next after verifying a magic-link token.
type VerifyResult struct {
	// For new user flow (registration): non-empty session_ref
	SessionRef string
	// For existing user flow (direct login): non-empty SessionToken + Role
	SessionToken string
	Role         string
	Email        string
	SchoolID     string
}

// ExchangeResult is the clean domain result from exchanging an IST.
type ExchangeResult struct {
	MemberAuthenticated bool
	StytchSessionToken  string
	MemberID            string
	OrganizationID      string
}

// MagicLinkAuthResult is the result of authenticating an org-scoped magic link
// token (from invite or login emails). It contains the member identity, org,
// and either a session token (if fully authenticated) or an intermediate
// session token (if MFA is required).
type MagicLinkAuthResult struct {
	MemberID                 string
	OrganizationID           string
	Email                    string
	StytchSessionToken       string
	IntermediateSessionToken string
	MemberAuthenticated      bool
}

// ============================================================================
// IdentityProvider interface — abstracts Stytch B2B (requirement 1).
// ============================================================================

// IdentityProvider defines the contract for authentication provider operations.
// All methods accept context.Context as first parameter (requirement 10).
type IdentityProvider interface {
	// SendDiscoveryEmail dispatches a magic link to the given email.
	// The redirect URL is determined by the identity provider configuration.
	SendDiscoveryEmail(ctx context.Context, email string) error

	// SendDiscoveryEmailWithRedirect dispatches a magic link to the given email
	// with a custom redirect URL. Used by invite flows where the callback must
	// go to the invite acceptance endpoint (/api/auth/invite/callback) instead
	// of the default login callback.
	SendDiscoveryEmailWithRedirect(ctx context.Context, email, redirectURL string) error

	// AuthenticateDiscoveryToken validates a magic-link token and returns
	// the raw Intermediate Session Token (IST), the verified email address,
	// and any discovered organizations the user already belongs to.
	AuthenticateDiscoveryToken(ctx context.Context, token string) (ist, email string, discoveredOrgs []DiscoveredOrg, err error)

	// CreateOrganization provisions a new organization in the identity provider.
	CreateOrganization(ctx context.Context, name string) (orgID string, err error)

	// ExchangeIntermediateSession exchanges an IST for a full session within
	// the context of a specific organization. Returns MemberAuthenticated
	// status for MFA enforcement (requirement 3).
	ExchangeIntermediateSession(ctx context.Context, ist, orgID string) (ExchangeResult, error)

	// CreateMember provisions a new member in an existing Stytch organization.
	CreateMember(ctx context.Context, orgID, email, name string) (memberID string, err error)

	// GetMemberByEmail retrieves an existing member's ID by email address.
	// Used when an invite attempt returns duplicate_member_email and we need the
	// member ID for our local invitation record.
	GetMemberByEmail(ctx context.Context, orgID, email string) (memberID string, err error)

	// InviteMemberByEmail sends a Stytch invite email to join an organization.
	// Returns the Stytch member ID of the invited member.
	InviteMemberByEmail(ctx context.Context, orgID, email, name, redirectURL string) (memberID string, err error)

	// AuthenticateMagicLink authenticates an org-scoped magic link token
	// (from invite or login emails) against the proper Stytch B2B endpoint
	// ("/v1/b2b/magic_links/authenticate", NOT the discovery endpoint).
	// Returns the member identity, organization, and either a session token
	// (if fully authenticated) or an intermediate session token (if MFA is
	// required). Use ExchangeIntermediateSession to convert an IST after MFA.
	AuthenticateMagicLink(ctx context.Context, token string) (*MagicLinkAuthResult, error)

	// ExchangeInviteSession exchanges an IST for a full Stytch session within a
	// specific organization. Enforces MemberAuthenticated == true and returns the
	// Stytch session token directly. Returns ErrMFARequired if the member has not
	// completed MFA.
	ExchangeInviteSession(ctx context.Context, ist, orgID string) (stytchSessionToken string, err error)
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
	FullName       string
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

	// GetTenantByStytchOrgID retrieves a tenant's ID by their Stytch org ID.
	GetTenantByStytchOrgID(ctx context.Context, stytchOrgID string) (string, error)

	// GetUserByEmailAndTenant retrieves a user's ID, full name, and external auth ID
	// by email and tenant ID. Returns ErrNotFound if no matching user exists.
	GetUserByEmailAndTenant(ctx context.Context, email, tenantID string) (userID, fullName, externalAuthID string, err error)

	// CreateSessionOnly creates a new session record for an existing user
	// without creating a user or tenant. Used during re-login.
	CreateSessionOnly(ctx context.Context, params CreateSessionParams) error

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

	// CreateMembership creates a membership linking a user to a school with a role.
	CreateMembership(ctx context.Context, userID, schoolID, tenantID, role string) error

	// SetActiveSchool upserts the member_active_school row for a user.
	SetActiveSchool(ctx context.Context, userID, tenantID, schoolID string) error

	// GetMeInfo returns the full profile info for /me: user details, role,
	// and the active school.
	GetMeInfo(ctx context.Context, token string) (*MeInfo, error)

	// GetActiveSchoolID returns the active school ID for a user in a tenant.
	// Checks member_active_school first; if none exists, falls back to the
	// user's first active membership's school ID.
	GetActiveSchoolID(ctx context.Context, userID, tenantID string) (string, error)

	// GetUserRoleInTenant returns the highest-privilege role for a user in a tenant.
	GetUserRoleInTenant(ctx context.Context, userID, tenantID string) (string, error)

	// GetInvitationByEmail looks up a pending, non-expired invitation by email.
	// Returns the invitation record or ErrNotFound if none exists.
	GetInvitationByEmail(ctx context.Context, email string) (*Invitation, error)

	// GetTenantStytchOrgID returns the Stytch org ID for a tenant.
	GetTenantStytchOrgID(ctx context.Context, tenantID string) (string, error)

	// CreateInvitedUserSession runs a single transaction to create a user,
	// session, membership, and mark the invitation as accepted.
	CreateInvitedUserSession(ctx context.Context, args CreateInvitedUserSessionArgs) error
}

// Invitation represents a pending invitation record used during the invite
// acceptance flow. It mirrors the invitations table but in the auth domain.
type Invitation struct {
	ID                 string    `json:"id"`
	TenantID           string    `json:"tenant_id"`
	SchoolID           string    `json:"school_id"`
	Role               string    `json:"role"`
	Email              string    `json:"email"`
	FullName           string    `json:"full_name"`
	Status             string    `json:"status"`
	StytchMemberID     string    `json:"stytch_member_id"`
	RegistrationNumber string    `json:"registration_number"`
	ExpiresAt          time.Time `json:"expires_at"`
}

// CreateInvitedUserSessionArgs holds all parameters for creating a user session
// during invite acceptance. All fields are required.
type CreateInvitedUserSessionArgs struct {
	InvitationID       string
	Email              string
	TenantID           string
	SchoolID           string
	Role               string
	FullName           string
	ExternalAuthID     string
	SessionToken       string
	StytchMemberID     string
	StytchOrgID        string
	StytchSessionToken string
	DeviceFingerprint  string
	ExpiresAt          time.Time
	TSCNumber          string // Maps from invitations.registration_number for TEACHER role
}

// MeInfo is the result of GetMeInfo.
type MeInfo struct {
	UserID     string
	TenantID   string
	Role       string
	SchoolID   string
	SchoolName string
	FullName   string
	Email      string
}

// SchoolCreator abstracts school creation so auth does not import cbcschools.
// The CreateSchool method matches cbcschools.Service.CreateSchool so that the
// full service pipeline (curriculum seeding, creator enrollment, active school)
// is used instead of a bare repository insert.
//
// GetSchoolByName returns the school ID for a school within a tenant, or
// ErrNotFound when no such school exists. Errors are translated to auth's own
// sentinels by the adapter in module.go.
type SchoolCreator interface {
	CreateSchool(ctx context.Context, tenantID string, name string, role string, creatorUserID ...string) (string, error)
	GetSchoolByName(ctx context.Context, tenantID, name string) (string, error)
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
	case errors.Is(err, ErrSessionRefExpired):
		return "session_ref_expired"
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
