package service

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/google/uuid"
	storegen "somotracker/internal/identity/store/gen"
	"somotracker/internal/platform/stytch"
)

const (
	// Token entropy: 32 bytes = 256 bits
	sessionTokenBytes = 32
	// Onboarding ticket TTL (matches IST TTL)
	onboardingTicketTTL = 10 * time.Minute
)

var (
	ErrInvalidToken            = errors.New("invalid or expired token")
	ErrPendingOnboardingNotFound = errors.New("pending onboarding not found")
	ErrOnboardingNotAllowed    = errors.New("onboarding not allowed at this stage")
	ErrIdempotencyConflict     = errors.New("idempotency key already used")
	ErrProvisioningFailed      = errors.New("tenant provisioning failed")
	ErrSessionNotFound         = errors.New("session not found or expired")
	ErrRateLimited             = errors.New("rate limit exceeded")
)

// SessionContext holds validated session data for request context.
type SessionContext struct {
	UserID    uuid.UUID
	TenantID  uuid.UUID
	Role      string
	SessionID uuid.UUID
}

// MembershipContext holds membership info for RBAC decisions
type MembershipContext struct {
	SchoolID uuid.UUID
	Role     string
	IsActive bool
}

// OnboardingState represents the onboarding state returned to frontend.
type OnboardingState struct {
	VerifiedEmail string
	CurrentStep   string
	FormData      map[string]interface{}
	PendingID     uuid.UUID
	SchoolName    *string
	TenantID      *uuid.UUID
	AuthState     string // AC-8: discovered organizations state
}

// Service defines business logic for identity domain.
type Service struct {
	repo           Repository
	stytch         stytch.Client
	db             *sql.DB
	istEncryption  *ISTEncryption
	idleTimeout    time.Duration // sliding window idle timeout; zero means disabled
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

	// Session
	GetUserByID(ctx context.Context, id uuid.UUID) (storegen.User, error)
	GetTenantMembershipRole(ctx context.Context, userID, tenantID uuid.UUID) (string, error)
	UpdateSessionLastSeen(ctx context.Context, sessionID uuid.UUID, idleThreshold time.Duration) error
	GetSessionWithMemberships(ctx context.Context, tokenHash []byte) (storegen.GetSessionWithMembershipsRow, error)
}

// New creates service. idleTimeout is the sliding window idle timeout;
// zero disables idle enforcement (absolute expiry still applies).
func New(repo Repository, stytchClient stytch.Client, db *sql.DB, istEncryptionKey string, idleTimeout time.Duration) (*Service, error) {
	var istEnc *ISTEncryption
	if istEncryptionKey != "" {
		var err error
		istEnc, err = NewISTEncryption(istEncryptionKey)
		if err != nil {
			return nil, fmt.Errorf("init IST encryption: %w", err)
		}
	}
	return &Service{repo: repo, stytch: stytchClient, db: db, istEncryption: istEnc, idleTimeout: idleTimeout}, nil
}

// RequestMagicLink initiates Stytch Discovery magic link flow.
func (s *Service) RequestMagicLink(ctx context.Context, email string) error {
	// Build login URL for Stytch callback from PUBLIC_APP_URL
	appURL := os.Getenv("PUBLIC_APP_URL")
	if appURL == "" {
		return fmt.Errorf("PUBLIC_APP_URL is required for Stytch magic link callback")
	}
	loginURL := strings.TrimRight(appURL, "/") + "/auth/callback"

	// Call Stytch B2B Discovery magic links/email
	if err := s.stytch.SendDiscoveryMagicLink(ctx, email, loginURL); err != nil {
		return fmt.Errorf("stytch send magic link: %w", err)
	}

	return nil
}

// generateRandomString generates a cryptographically secure random string.
func generateRandomString(n int) (string, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

// AuthenticateToken verifies one-time Stytch token, upserts pending onboarding and creates ticket.
func (s *Service) AuthenticateToken(ctx context.Context, token string) (string, *OnboardingState, *string, error) {
	// 1. Call Stytch b2b/discovery/authenticate with token
	resp, err := s.stytch.AuthenticateDiscovery(ctx, token)
	if err != nil {
		return "", nil, nil, fmt.Errorf("stytch authenticate: %w", err)
	}

	// 2. Normalize email
	normalizedEmail := normalizeEmail(resp.Email)

	// 3. Check existing completed onboarding first (returning user after provisioning)
	existingPending, err := s.repo.GetPendingOnboardingByEmail(ctx, normalizedEmail)
	if err == nil && existingPending.Status == "completed" && existingPending.TenantID.Valid {
		authState := "authenticated"
		// Create session for returning user
		var sessionToken *string
		tenantID := existingPending.TenantID.UUID
		return "", &OnboardingState{
			VerifiedEmail: normalizedEmail,
			CurrentStep:   "complete",
			FormData:      map[string]interface{}{},
			PendingID:     existingPending.ID,
			TenantID:      &tenantID,
			AuthState:     authState,
		}, sessionToken, nil
	}

	// 4. Upsert PendingOnboarding with status=pending
	pending, err := s.repo.UpsertPendingOnboarding(ctx, storegen.UpsertPendingOnboardingParams{
		VerifiedEmailNormalized: normalizedEmail,
		Status:                  "pending",
		CurrentStep:             "tenant-name",
		FormData:                json.RawMessage("{}"),
	})
	if err != nil {
		return "", nil, nil, fmt.Errorf("upsert pending onboarding: %w", err)
	}

	// 4. Create OnboardingTicket with ticket_hash + encrypted IST
	// Generate ticket token
	ticketToken, err := generateRandomString(sessionTokenBytes)
	if err != nil {
		return "", nil, nil, fmt.Errorf("generate ticket token: %w", err)
	}
	ticketHash := sha256.Sum256([]byte(ticketToken))

	// Encrypt IST using AES-256-GCM
	var encryptedIST string
	if s.istEncryption != nil {
		encryptedIST, err = s.istEncryption.Encrypt([]byte(resp.IntermediateSessionToken))
		if err != nil {
			return "", nil, nil, fmt.Errorf("encrypt IST: %w", err)
		}
	} else {
		// Fallback: store plaintext (dev only)
		encryptedIST = resp.IntermediateSessionToken
	}

	expiresAt := time.Now().Add(onboardingTicketTTL)
	_, err = s.repo.CreateOnboardingTicket(ctx, storegen.CreateOnboardingTicketParams{
		PendingOnboardingID: pending.ID,
		TicketHash:          ticketHash[:],
		IstCiphertext:       []byte(encryptedIST),
		ExpiresAt:           expiresAt,
	})
	if err != nil {
		return "", nil, nil, fmt.Errorf("create onboarding ticket: %w", err)
	}

	// 5. Determine state based on discovered organizations
	authState := determineAuthState(resp.DiscoveredOrganizations)

	return ticketToken, &OnboardingState{
		VerifiedEmail: normalizedEmail,
		CurrentStep:   "tenant-name",
		FormData:      map[string]interface{}{},
		PendingID:     pending.ID,
		AuthState:     authState,
	}, nil, nil
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

	// 2. SHA-256 hash of hex token string (matches middleware/hash lookup)
	tokenHash := sha256.Sum256([]byte(rawToken))

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

// ValidateSessionWithMemberships validates session and loads active memberships in one DB round-trip.
func (s *Service) ValidateSessionWithMemberships(ctx context.Context, tokenHash []byte) (*SessionContext, []MembershipContext, error) {
	row, err := s.repo.GetSessionWithMemberships(ctx, tokenHash)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil, ErrSessionNotFound
		}
		return nil, nil, fmt.Errorf("query session with memberships: %w", err)
	}

	// Idle timeout enforcement (fail closed).
	// If last_seen_at is older than idleTimeout, the session is expired.
	// last_seen_at is NULL for sessions created before the column existed;
	// treat those as valid (backward compat) and slide them forward.
	if s.idleTimeout > 0 && row.LastSeenAt.Valid {
		idleCutoff := time.Now().Add(-s.idleTimeout)
		if row.LastSeenAt.Time.Before(idleCutoff) {
			return nil, nil, ErrSessionNotFound
		}
	}

	// Slide last_seen_at (throttled: only writes if older than idleThreshold).
	// Use a shorter threshold than idleTimeout to avoid DB writes on every request.
	// Best-effort: a slide failure does not fail the request.
	if s.idleTimeout > 0 {
		idleThreshold := 5 * time.Minute
		_ = s.repo.UpdateSessionLastSeen(ctx, row.ID, idleThreshold)
	}

	memberships := []MembershipContext{}
	if row.MembershipID.Valid && row.SchoolID.Valid && row.Role.Valid {
		memberships = append(memberships, MembershipContext{
			SchoolID: row.SchoolID.UUID,
			Role:     string(row.Role.UserRole),
			IsActive: row.IsActive.Valid && row.IsActive.Bool,
		})
	}

	return &SessionContext{
		UserID:    row.UserID,
		TenantID:  row.TenantID,
		SessionID: row.ID,
		Role:      "",
	}, memberships, nil
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