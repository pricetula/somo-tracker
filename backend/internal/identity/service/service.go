package service

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	storegen "somotracker/internal/identity/store/gen"
	"somotracker/internal/platform/stytch"
)

const (
	// Token entropy: 32 bytes = 256 bits
	sessionTokenBytes = 32
	// PKCE code verifier length (43-128 chars per RFC 7636)
	pkceVerifierBytes = 32
	// Onboarding ticket TTL (matches IST TTL)
	onboardingTicketTTL = 10 * time.Minute
)

var (
	ErrInvalidToken           = errors.New("invalid or expired token")
	ErrPendingOnboardingNotFound = errors.New("pending onboarding not found")
	ErrOnboardingNotAllowed   = errors.New("onboarding not allowed at this stage")
	ErrIdempotencyConflict    = errors.New("idempotency key already used")
	ErrProvisioningFailed     = errors.New("tenant provisioning failed")
	ErrSessionNotFound        = errors.New("session not found or expired")
	ErrRateLimited            = errors.New("rate limit exceeded")
)

// SessionContext holds validated session data for request context.
type SessionContext struct {
	UserID    uuid.UUID
	TenantID  uuid.UUID
	Role      string
	SessionID uuid.UUID
}

// OnboardingState represents the onboarding state returned to frontend.
type OnboardingState struct {
	VerifiedEmail string
	CurrentStep   string
	FormData      map[string]interface{}
	PendingID     uuid.UUID
	SchoolName    *string
	TenantID      *uuid.UUID
}

// ProvisionResult holds the result of provisioning.
type ProvisionResult struct {
	TenantID *uuid.UUID
	UserID   *uuid.UUID
	SchoolID *uuid.UUID
}

// ProvisionData holds all data needed for tenant provisioning.
type ProvisionData struct {
	SchoolName     string
	IdempotencyKey string
	SchoolCode     string
	AdminName      string
}

// Service defines business logic for identity domain.
type Service struct {
	repo  Repository
	stytch stytch.Client
	db    *sql.DB
}

// Repository is domain repository interface, using sqlc-generated types.
type Repository interface {
	GetPendingOnboardingByEmail(ctx context.Context, email string) (storegen.PendingOnboarding, error)
	GetPendingOnboardingByID(ctx context.Context, id uuid.UUID) (storegen.PendingOnboarding, error)
	UpsertPendingOnboarding(ctx context.Context, arg storegen.UpsertPendingOnboardingParams) (storegen.PendingOnboarding, error)
	UpdatePendingOnboardingStep(ctx context.Context, arg storegen.UpdatePendingOnboardingStepParams) error
	CreateAppSession(ctx context.Context, arg storegen.CreateAppSessionParams) (storegen.AppSession, error)
	GetAppSessionByTokenHash(ctx context.Context, tokenHash []byte) (storegen.AppSession, error)
	GetAppSessionByID(ctx context.Context, id uuid.UUID) (storegen.AppSession, error)
	RevokeAppSession(ctx context.Context, id uuid.UUID) error
	CreateOnboardingTicket(ctx context.Context, arg storegen.CreateOnboardingTicketParams) (storegen.OnboardingTicket, error)
	GetOnboardingTicketByHash(ctx context.Context, hash []byte) (storegen.OnboardingTicket, error)
	MarkOnboardingTicketUsed(ctx context.Context, id uuid.UUID) error

	// Core domain - for transactional provisioning
	CreateTenant(ctx context.Context, arg storegen.CreateTenantParams) (storegen.Tenant, error)
	GetTenantByStytchOrgID(ctx context.Context, stytchOrgID uuid.NullUUID) (storegen.Tenant, error)
	CreateUser(ctx context.Context, arg storegen.CreateUserParams) (storegen.User, error)
	GetUserByStytchMemberID(ctx context.Context, stytchMemberID uuid.NullUUID) (storegen.User, error)
	CreateSchool(ctx context.Context, arg storegen.CreateSchoolParams) (storegen.School, error)
	CreateMembership(ctx context.Context, arg storegen.CreateMembershipParams) (storegen.Membership, error)
	GetUserByID(ctx context.Context, id uuid.UUID) (storegen.User, error)
	GetTenantMembershipRole(ctx context.Context, userID, tenantID uuid.UUID) (string, error)
}

// New creates service.
func New(repo Repository, stytchClient stytch.Client, db *sql.DB) *Service {
	return &Service{repo: repo, stytch: stytchClient, db: db}
}

// RequestMagicLink initiates Stytch Discovery magic link flow.
func (s *Service) RequestMagicLink(ctx context.Context, email string) (string, error) {
	// Generate PKCE code_verifier
	verifier, err := generateRandomString(pkceVerifierBytes)
	if err != nil {
		return "", fmt.Errorf("generate PKCE verifier: %w", err)
	}

	// Compute code_challenge S256
	challenge := pkceCodeChallenge(verifier)

	// Build login URL for Stytch callback
	loginURL := "https://app.example.com/auth/callback" // TODO: from config

	// Call Stytch B2B Discovery magic_links/email
	if err := s.stytch.SendDiscoveryMagicLink(ctx, email, loginURL, challenge); err != nil {
		return "", fmt.Errorf("stytch send magic link: %w", err)
	}

	// Return PKCE verifier for handler to store in cookie
	return verifier, nil
}

// generateRandomString generates a cryptographically secure random string.
func generateRandomString(n int) (string, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

// pkceCodeChallenge computes S256 code challenge from verifier.
func pkceCodeChallenge(verifier string) string {
	h := sha256.Sum256([]byte(verifier))
	return base64URLEncode(h[:])
}

// base64URLEncode encodes bytes to base64url without padding.
func base64URLEncode(b []byte) string {
	return base64.RawURLEncoding.EncodeToString(b)
}

// AuthenticateToken verifies one-time Stytch token, upserts pending onboarding and creates ticket.
func (s *Service) AuthenticateToken(ctx context.Context, token, pkceCodeVerifier string) (string, *OnboardingState, error) {
	// 1. Call Stytch b2b/discovery/authenticate with token + pkce_code_verifier
	resp, err := s.stytch.AuthenticateDiscovery(ctx, token, pkceCodeVerifier)
	if err != nil {
		return "", nil, fmt.Errorf("stytch authenticate: %w", err)
	}

	// 2. Normalize email
	normalizedEmail := normalizeEmail(resp.Email)

	// 3. Upsert PendingOnboarding with status=pending
	pending, err := s.repo.UpsertPendingOnboarding(ctx, storegen.UpsertPendingOnboardingParams{
		VerifiedEmailNormalized: normalizedEmail,
		Status:                  "pending",
		CurrentStep:             "tenant-name",
		FormData:                json.RawMessage("{}"),
	})
	if err != nil {
		return "", nil, fmt.Errorf("upsert pending onboarding: %w", err)
	}

	// 4. Create OnboardingTicket with ticket_hash + encrypted IST
	// Generate ticket token
	ticketToken, err := generateRandomString(sessionTokenBytes)
	if err != nil {
		return "", nil, fmt.Errorf("generate ticket token: %w", err)
	}
	ticketHash := sha256.Sum256([]byte(ticketToken))

	// Encrypt IST (in production, use proper encryption)
	// For now, store as-is with note that it should be encrypted
	encryptedIST := []byte(resp.IntermediateSessionToken)

	expiresAt := time.Now().Add(onboardingTicketTTL)
	_, err = s.repo.CreateOnboardingTicket(ctx, storegen.CreateOnboardingTicketParams{
		PendingOnboardingID: pending.ID,
		TicketHash:          ticketHash[:],
		EncryptedIst:        encryptedIST,
		ExpiresAt:           expiresAt,
	})
	if err != nil {
		return "", nil, fmt.Errorf("create onboarding ticket: %w", err)
	}

	// 5. Determine state based on discovered organizations
	_ = determineAuthState(resp.DiscoveredOrganizations)

	return ticketToken, &OnboardingState{
		VerifiedEmail: normalizedEmail,
		CurrentStep:   "tenant-name",
		FormData:      map[string]interface{}{},
		PendingID:     pending.ID,
	}, nil
}

// normalizeEmail normalizes email for consistent lookup.
func normalizeEmail(email string) string {
	// Lowercase and trim
	return strings.ToLower(strings.TrimSpace(email))
}

// determineAuthState determines auth state from discovered organizations.
func determineAuthState(orgs []stytch.DiscoveredOrg) string {
	if len(orgs) == 0 {
		return "onboarding_required"
	}
	if len(orgs) == 1 {
		// Single org - could be organization_selection_required or authenticated
		// For now, return organization_selection_required
		return "organization_selection_required"
	}
	// Multiple orgs - user must select
	return "organization_selection_required"
}

// ProvisionTenant creates Stytch org/member and local tenant/user/school/membership in a transaction.
func (s *Service) ProvisionTenant(ctx context.Context, ticketToken string, data ProvisionData) (*ProvisionResult, error) {
	// 1. Validate ticket
	ticketHash := sha256.Sum256([]byte(ticketToken))
	ticket, err := s.repo.GetOnboardingTicketByHash(ctx, ticketHash[:])
	if err != nil {
		return nil, ErrInvalidToken
	}

	// 2. Mark ticket as used
	if err := s.repo.MarkOnboardingTicketUsed(ctx, ticket.ID); err != nil {
		return nil, fmt.Errorf("mark ticket used: %w", err)
	}

	// 3. Get pending onboarding by ID from ticket
	pending, err := s.repo.GetPendingOnboardingByID(ctx, ticket.PendingOnboardingID)
	if err != nil {
		return nil, ErrPendingOnboardingNotFound
	}

	// 4. Check idempotency
	if pending.IdempotencyKey.Valid && pending.IdempotencyKey.String == data.IdempotencyKey {
		// Already processed with this key
		return &ProvisionResult{
			TenantID: nullUUIDToPtr(pending.TenantID),
			UserID:   nil,
		}, nil
	}

	// 5. Execute provisioning in transaction
	return s.provisionInTransaction(ctx, ticket, pending, data)
}

// provisionInTransaction runs the full provisioning in a database transaction.
func (s *Service) provisionInTransaction(ctx context.Context, ticket storegen.OnboardingTicket, pending storegen.PendingOnboarding, data ProvisionData) (*ProvisionResult, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	// Create transactional repository
	txRepo := storegen.New(tx)

	// Exchange IST for Stytch session
	ist := string(ticket.EncryptedIst) // TODO: decrypt
	_, err = s.stytch.ExchangeDiscoverySession(ctx, ist)
	if err != nil {
		return nil, fmt.Errorf("exchange discovery session: %w", err)
	}

	// Create Stytch organization and member
	orgMember, err := s.stytch.CreateOrganizationAndMember(ctx, data.SchoolName, pending.VerifiedEmailNormalized)
	if err != nil {
		return nil, fmt.Errorf("create org/member: %w", err)
	}

	tenantID := uuid.MustParse(orgMember.OrganizationID)
	memberID := uuid.MustParse(orgMember.MemberID)

	// Create local Tenant
	tenant, err := txRepo.CreateTenant(ctx, storegen.CreateTenantParams{
		Name:                   data.SchoolName,
		StytchOrganizationID:   uuid.NullUUID{UUID: tenantID, Valid: true},
	})
	if err != nil {
		return nil, fmt.Errorf("create tenant: %w", err)
	}

	// Create admin User
	adminUser, err := txRepo.CreateUser(ctx, storegen.CreateUserParams{
		TenantID:           tenant.ID,
		EmailNormalized:    pending.VerifiedEmailNormalized,
		StytchMemberID:     uuid.NullUUID{UUID: memberID, Valid: true},
		Name:               data.AdminName,
	})
	if err != nil {
		return nil, fmt.Errorf("create user: %w", err)
	}

	// Generate school code from name
	schoolCode := generateSchoolCode(data.SchoolName)

	// Create School
	school, err := txRepo.CreateSchool(ctx, storegen.CreateSchoolParams{
		TenantID: tenant.ID,
		Name:     data.SchoolName,
		Code:     schoolCode,
	})
	if err != nil {
		return nil, fmt.Errorf("create school: %w", err)
	}

	// Create Membership (tenant_owner role for first admin)
	_, err = txRepo.CreateMembership(ctx, storegen.CreateMembershipParams{
		UserID:   adminUser.ID,
		SchoolID: school.ID,
		Role:     "tenant_owner",
	})
	if err != nil {
		return nil, fmt.Errorf("create membership: %w", err)
	}

	// Update PendingOnboarding to completed
	_, err = txRepo.UpsertPendingOnboarding(ctx, storegen.UpsertPendingOnboardingParams{
		VerifiedEmailNormalized: pending.VerifiedEmailNormalized,
		Status:                  "completed",
		CurrentStep:             "complete",
		FormData:                pending.FormData,
		StytchOrganizationID:    uuid.NullUUID{UUID: tenantID, Valid: true},
		StytchMemberID:          uuid.NullUUID{UUID: memberID, Valid: true},
		TenantID:                uuid.NullUUID{UUID: tenant.ID, Valid: true},
		SchoolName:              sqlNullString(data.SchoolName),
		IdempotencyKey:          sqlNullString(data.IdempotencyKey),
	})
	if err != nil {
		return nil, fmt.Errorf("update pending onboarding: %w", err)
	}

	// Commit transaction
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit transaction: %w", err)
	}

	return &ProvisionResult{
		TenantID: &tenant.ID,
		UserID:   &adminUser.ID,
		SchoolID: &school.ID,
	}, nil
}

// generateSchoolCode generates a unique school code from name.
func generateSchoolCode(name string) string {
	// Simple code generation: first 4 letters + random suffix
	base := strings.ToUpper(strings.ReplaceAll(name, " ", ""))
	if len(base) > 4 {
		base = base[:4]
	}
	suffix, _ := generateRandomString(4)
	return base + "-" + strings.ToUpper(suffix)
}

// CreateSession creates opaque app session after tenant provisioning.
func (s *Service) CreateSession(ctx context.Context, userID, tenantID uuid.UUID, stytchMemberSessionID string, expiresAt time.Time) (string, error) {
	// 1. Generate 256-bit random token
	tokenBytes := make([]byte, sessionTokenBytes)
	if _, err := rand.Read(tokenBytes); err != nil {
		return "", fmt.Errorf("generate token: %w", err)
	}
	rawToken := hex.EncodeToString(tokenBytes)

	// 2. SHA-256 hash
	tokenHash := sha256.Sum256(tokenBytes)

	// 3. Store in AppSession
	var stytchSessionID sql.NullString
	if stytchMemberSessionID != "" {
		stytchSessionID = sql.NullString{String: stytchMemberSessionID, Valid: true}
	}

	_, err := s.repo.CreateAppSession(ctx, storegen.CreateAppSessionParams{
		UserID:                userID,
		TenantID:              tenantID,
		TokenHash:             tokenHash[:],
		StytchMemberSessionID: stytchSessionID,
		ExpiresAt:             expiresAt,
	})
	if err != nil {
		return "", fmt.Errorf("create app session: %w", err)
	}

	return rawToken, nil
}

// ValidateSession validates opaque token hash and returns session context with role.
func (s *Service) ValidateSession(ctx context.Context, tokenHash []byte) (*SessionContext, error) {
	// 1. Lookup AppSession by hash, check revoked_at IS NULL and expires_at > now
	session, err := s.repo.GetAppSessionByTokenHash(ctx, tokenHash)
	if err != nil {
		return nil, ErrSessionNotFound
	}

	// 2. Get user's role from membership
	role, err := s.repo.GetTenantMembershipRole(ctx, session.UserID, session.TenantID)
	if err != nil {
		// Default to empty role if not found
		role = ""
	}

	// 3. Return SessionContext
	return &SessionContext{
		UserID:    session.UserID,
		TenantID:  session.TenantID,
		SessionID: session.ID,
		Role:      role,
	}, nil
}

// Logout revokes session and Stytch member session.
func (s *Service) Logout(ctx context.Context, sessionID uuid.UUID) error {
	// 1. Get session to find member/session info for Stytch revocation
	session, err := s.repo.GetAppSessionByID(ctx, sessionID)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return fmt.Errorf("get session for revocation: %w", err)
	}

	// 2. Revoke Stytch member session if available
	if session.StytchMemberSessionID.Valid && session.StytchMemberSessionID.String != "" {
		user, userErr := s.repo.GetUserByID(ctx, session.UserID)
		if userErr == nil && user.StytchMemberID.Valid {
			_ = s.stytch.RevokeMemberSession(ctx, user.StytchMemberID.UUID.String(), session.StytchMemberSessionID.String)
		}
	}

	// 3. Revoke local session
	if err := s.repo.RevokeAppSession(ctx, sessionID); err != nil {
		return fmt.Errorf("revoke app session: %w", err)
	}

	return nil
}

// UpdateOnboardingStep updates the current step and form data for pending onboarding.
func (s *Service) UpdateOnboardingStep(ctx context.Context, pendingID uuid.UUID, step string, formData map[string]interface{}) error {
	data, _ := json.Marshal(formData)
	return s.repo.UpdatePendingOnboardingStep(ctx, storegen.UpdatePendingOnboardingStepParams{
		ID:          pendingID,
		CurrentStep: step,
		FormData:    data,
	})
}

// GetOnboardingCurrent retrieves the current onboarding state.
func (s *Service) GetOnboardingCurrent(ctx context.Context, ticketToken string) (*OnboardingState, error) {
	ticketHash := sha256.Sum256([]byte(ticketToken))
	ticket, err := s.repo.GetOnboardingTicketByHash(ctx, ticketHash[:])
	if err != nil {
		return nil, ErrInvalidToken
	}

	pending, err := s.repo.GetPendingOnboardingByID(ctx, ticket.PendingOnboardingID)
	if err != nil {
		return nil, ErrPendingOnboardingNotFound
	}

	formData := map[string]interface{}{}
	if len(pending.FormData) > 0 && string(pending.FormData) != "{}" {
		_ = json.Unmarshal(pending.FormData, &formData)
	}

	return &OnboardingState{
		VerifiedEmail: pending.VerifiedEmailNormalized,
		CurrentStep:   pending.CurrentStep,
		FormData:      formData,
		PendingID:     pending.ID,
		SchoolName:    sqlNullStringToStringPtr(pending.SchoolName),
		TenantID:      nullUUIDToPtr(pending.TenantID),
	}, nil
}

// nullUUIDToPtr converts uuid.NullUUID to *uuid.UUID
func nullUUIDToPtr(n uuid.NullUUID) *uuid.UUID {
	if n.Valid {
		return &n.UUID
	}
	return nil
}

// sqlNullStringToStringPtr converts sql.NullString to *string
func sqlNullStringToStringPtr(n sql.NullString) *string {
	if n.Valid {
		return &n.String
	}
	return nil
}

// sqlNullString converts string to sql.NullString
func sqlNullString(s string) sql.NullString {
	if s == "" {
		return sql.NullString{Valid: false}
	}
	return sql.NullString{String: s, Valid: true}
}