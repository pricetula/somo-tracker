package auth

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
	"go.uber.org/fx"
	"go.uber.org/zap"

	"somotracker/backend/internal/config"
	"somotracker/backend/internal/database"
)

// ============================================================================
// Service — core business logic layer (requirement 4, 8, 9, 10, 11, 13).
// ============================================================================

const (
	istTTL        = 10 * time.Minute    // Redis TTL for IST cache (requirement 2)
	sessionTTL    = 30 * 24 * time.Hour // 30-day Redis TTL for session token (requirement 4)
	istKeyPrefix  = "ist:"              // key pattern: "ist:{env}:{uuid}"
	sessionPrefix = "session:"          // key pattern: "session:{token}"
)

// Service holds business logic dependencies.
type Service struct {
	idp           IdentityProvider
	repo          Repository
	rdb           *redis.Client
	logger        *zap.Logger
	cfg           config.Config
	schoolCreator SchoolCreator
}

// NewService creates a new Service with fx lifecycle hooks for Redis.
func NewService(
	lc fx.Lifecycle,
	idp IdentityProvider,
	repo Repository,
	schoolCreator SchoolCreator,
	pools *database.Pools,
	logger *zap.Logger,
	cfg config.Config,
) *Service {
	// Register Redis lifecycle hooks (requirement 15)
	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			if err := pools.Redis.Ping(ctx).Err(); err != nil {
				return fmt.Errorf("auth.service.OnStart: redis ping failed: %w", err)
			}
			logger.Info("auth service: redis connection verified")
			return nil
		},
		OnStop: func(ctx context.Context) error {
			if err := pools.Redis.Close(); err != nil {
				logger.Error("auth.service.OnStop: redis close error", zap.Error(err))
				return fmt.Errorf("auth.service.OnStop: redis close: %w", err)
			}
			logger.Info("auth service: redis connection closed")
			return nil
		},
	})

	return &Service{
		idp:           idp,
		repo:          repo,
		rdb:           pools.Redis,
		logger:        logger,
		cfg:           cfg,
		schoolCreator: schoolCreator,
	}
}

// Discover initiates the magic-link discovery flow (PHASE 1).
func (s *Service) Discover(ctx context.Context, email string) error {
	s.logger.Info("auth: discovery initiated", zap.String("email", email))

	if err := s.idp.SendDiscoveryEmail(ctx, email); err != nil {
		s.logger.Error("auth: discovery send failed", zap.String("email", email), zap.Error(err))
		return fmt.Errorf("auth.Service.Discover: %w", err)
	}

	s.logger.Info("auth: discovery email sent", zap.String("email", email))
	return nil
}

// istCacheData is the JSON structure stored in Redis for a pending registration.
type istCacheData struct {
	IST   string `json:"ist"`
	Email string `json:"email"`
}

// Verify validates a magic-link token and determines whether the user is new or existing.
//
// For existing users (with discovered organizations): exchanges the IST, creates a session,
// and returns a VerifyResult with SessionToken + Role for direct dashboard redirect.
//
// For new users (no discovered organizations): caches the IST and email in Redis and
// returns a VerifyResult with SessionRef for the registration flow.
func (s *Service) Verify(ctx context.Context, token string, deviceFingerprint string) (*VerifyResult, error) {
	s.logger.Info("auth: verify initiated")

	// Authenticate the discovery token with Stytch — now returns IST, email, and discovered orgs
	ist, email, discoveredOrgs, err := s.idp.AuthenticateDiscoveryToken(ctx, token)
	if err != nil {
		// If it's an expired token error, propagate it directly
		if errors.Is(err, ErrExpiredToken) {
			return nil, err
		}
		return nil, err
	}

	// Check if the user has discovered organizations (existing Stytch memberships)
	if len(discoveredOrgs) > 0 {
		// Existing user path: find matching tenant in our DB, exchange IST, create session
		result, err := s.handleExistingUser(ctx, ist, email, discoveredOrgs, deviceFingerprint)
		if err != nil {
			return nil, err
		}
		return result, nil
	}

	// New user path: cache IST in Redis for the registration flow
	sessionRef, err := generateUUID()
	if err != nil {
		return nil, fmt.Errorf("%w: generate session ref: %v", ErrInternal, err)
	}

	cacheData := istCacheData{IST: ist, Email: email}
	cacheJSON, err := json.Marshal(cacheData)
	if err != nil {
		return nil, fmt.Errorf("%w: marshal cache data: %v", ErrInternal, err)
	}

	istKey := fmt.Sprintf("%s%s:%s", istKeyPrefix, s.cfg.AppEnv, sessionRef)
	if err := s.rdb.Set(ctx, istKey, string(cacheJSON), istTTL).Err(); err != nil {
		return nil, fmt.Errorf("%w: cache ist: %v", ErrInternal, err)
	}

	s.logger.Info("auth: new user — IST and email cached in Redis",
		zap.String("session_ref", sessionRef),
		zap.String("email", email),
		zap.Duration("ttl", istTTL),
	)

	return &VerifyResult{SessionRef: sessionRef, Email: email}, nil
}

// handleExistingUser processes login for a user who already has Stytch memberships.
// It finds a matching tenant in our DB, exchanges the IST, creates a session,
// and returns session + role for direct dashboard redirect.
func (s *Service) handleExistingUser(ctx context.Context, ist, email string, discoveredOrgs []DiscoveredOrg, deviceFingerprint string) (*VerifyResult, error) {
	s.logger.Info("auth: existing user detected — processing direct login",
		zap.String("email", email),
		zap.Int("discovered_orgs", len(discoveredOrgs)),
	)

	// Find the first discovered org that matches a tenant in our database
	for _, org := range discoveredOrgs {
		tenantID, err := s.repo.GetTenantByStytchOrgID(ctx, org.OrganizationID)
		if err == nil && tenantID != "" {
			// Look up the user in this tenant
			userID, _, _, err := s.repo.GetUserByEmailAndTenant(ctx, email, tenantID)
			if err == nil && userID != "" {
				s.logger.Info("auth: found matching tenant and user",
					zap.String("org_id", org.OrganizationID),
					zap.String("tenant_id", tenantID),
					zap.String("user_id", userID),
					zap.String("stytch_member_id", org.MemberID),
				)

				// Exchange the IST against this organization
				exchangeResult, err := s.idp.ExchangeIntermediateSession(ctx, ist, org.OrganizationID)
				if err != nil {
					return nil, fmt.Errorf("auth.Service.handleExistingUser: exchange IST: %w", err)
				}

				if !exchangeResult.MemberAuthenticated {
					s.logger.Warn("auth: MFA required for existing user",
						zap.String("email", email),
						zap.String("org_id", org.OrganizationID),
					)
					return nil, ErrMFARequired
				}

				// Generate a session token
				tokenBytes := make([]byte, 32)
				if _, err := rand.Read(tokenBytes); err != nil {
					return nil, fmt.Errorf("%w: generate session token: %v", ErrInternal, err)
				}
				sessionToken := hex.EncodeToString(tokenBytes)
				expiresAt := time.Now().Add(sessionTTL)

				// Get the user's current highest role
				// We use a lightweight approach: look at the session we'll create,
				// but first just create the session record
				sessionParams := CreateSessionParams{
					Token:              sessionToken,
					UserID:             userID,
					TenantID:           tenantID,
					StytchMemberID:     exchangeResult.MemberID,
					StytchOrgID:        org.OrganizationID,
					StytchSessionToken: exchangeResult.StytchSessionToken,
					DeviceFingerprint:  deviceFingerprint,
					ExpiresAt:          expiresAt,
				}

				if err := s.repo.CreateSessionOnly(ctx, sessionParams); err != nil {
					return nil, fmt.Errorf("auth.Service.handleExistingUser: create session: %w", err)
				}

				// Cache session in Redis
				if err := s.rdb.Set(ctx, s.sessionKey(sessionToken), exchangeResult.StytchSessionToken, sessionTTL).Err(); err != nil {
					return nil, fmt.Errorf("%w: cache session: %v", ErrInternal, err)
				}

				// Retrieve the user's role from the newly created session
				session, err := s.repo.GetSessionByToken(ctx, sessionToken)
				if err != nil {
					return nil, fmt.Errorf("auth.Service.handleExistingUser: get session role: %w", err)
				}

				s.logger.Info("auth: existing user logged in successfully",
					zap.String("user_id", userID),
					zap.String("tenant_id", tenantID),
					zap.String("role", session.Role),
					zap.String("email", email),
					zap.String("session_token_preview", sessionToken[:8]+"..."),
				)

				return &VerifyResult{
					SessionToken: sessionToken,
					Role:         session.Role,
					Email:        email,
				}, nil
			}
		}
	}

	// No matching org found in our database — the Stytch org still exists but our DB
	// was wiped (e.g. database reset). Re-use the existing Stytch org to reconstruct
	// the tenant, user, session, and school in Postgres.
	s.logger.Info("auth: discovered orgs found but no tenant in DB — reconstructing from Stytch",
		zap.String("email", email),
		zap.Int("discovered_org_count", len(discoveredOrgs)),
	)

	return s.reconstructFromStytch(ctx, ist, email, discoveredOrgs, deviceFingerprint)
}

// Register completes the registration flow (PHASE 3).
// Validates input, reads IST from Redis, creates org in Stytch,
// exchanges IST, persists to Postgres, creates school + membership,
// issues session token, and returns the user's assigned role.
func (s *Service) Register(ctx context.Context, sessionRef string, payload RegistrationPayload, deviceFingerprint string) (sessionToken string, role string, schoolID string, err error) {
	s.logger.Info("auth: register initiated",
		zap.String("session_ref", sessionRef),
		zap.String("school_name", payload.SchoolName),
	)

	// 1. Validate payload (requirement 13)
	if err := payload.Validate(); err != nil {
		return "", "", "", err
	}

	// 2. Read AND DELETE IST and email from Redis atomically (requirement 2 — one-time use)
	ist, email, err := s.readAndDeleteIST(ctx, sessionRef)
	if err != nil {
		return "", "", "", err
	}

	// 3. Determine Stytch organization — either existing or new
	//    Check by school name first (cheap lookup). If the tenant already
	//    exists in the database, retrieve its Stytch org ID and skip org
	//    creation. Otherwise, provision a new org in Stytch.
	//
	//    The idempotency scope is the session_ref. If a previous call created
	//    the org and then failed on Postgres, the IST would already be consumed
	//    (not found in Redis). So the key idempotency check happens here at the
	//    IST lookup — if the IST is already consumed, we return ErrExpiredToken.
	tenantExistsByName, err := s.repo.TenantExistsByName(ctx, payload.SchoolName)
	if err != nil {
		return "", "", "", err
	}

	var orgID string
	var existingTenantID string
	var userID, tenantID string

	if tenantExistsByName {
		// Tenant already exists — retrieve its Stytch org ID so we can
		// exchange the IST against the correct org without re-creating it.
		var stytchOrgID string
		existingTenantID, stytchOrgID, err = s.repo.GetTenantByName(ctx, payload.SchoolName)
		if err != nil {
			return "", "", "", err
		}
		orgID = stytchOrgID
		s.logger.Info("auth: tenant already exists, using existing org",
			zap.String("school_name", payload.SchoolName),
			zap.String("existing_tenant_id", existingTenantID),
			zap.String("stytch_org_id", orgID),
		)
	} else {
		// No existing tenant — provision a new organization in Stytch
		orgID, err = s.idp.CreateOrganization(ctx, payload.SchoolName)
		if err != nil {
			s.logger.Error("auth: create org failed",
				zap.String("school_name", payload.SchoolName),
				zap.Error(err),
			)
			return "", "", "", err
		}
	}

	// Set the stytch_org_id in context for reconciliation logging (requirement 9)
	ctx = context.WithValue(ctx, StytchOrgIDKey{}, orgID)

	// 5. Create member in the Stytch org before exchanging the IST.
	//    This ensures the member exists in the org so IST exchange doesn't
	//    fail with email_jit_provisioning_not_allowed.
	memberName := payload.FullName
	if _, err := s.idp.CreateMember(ctx, orgID, email, memberName); err != nil {
		s.logger.Error("auth: create member failed",
			zap.String("org_id", orgID),
			zap.String("email", email),
			zap.Error(err),
		)
		return "", "", "", err
	}

	// 6. Exchange intermediate session (requirement 3 — MFA check)
	result, err := s.idp.ExchangeIntermediateSession(ctx, ist, orgID)
	if err != nil {
		s.logger.Error("auth: IST exchange failed",
			zap.String("org_id", orgID),
			zap.Error(err),
		)
		return "", "", "", err
	}

	// 6. Check MFA status (requirement 3)
	if !result.MemberAuthenticated {
		s.logger.Warn("auth: MFA required — blocking session creation",
			zap.String("member_id", result.MemberID),
			zap.String("org_id", orgID),
		)
		return "", "", "", ErrMFARequired
	}

	// 7. Check for existing tenant (idempotency check after exchange — requirement 8)
	tenantExists, err := s.repo.TenantExists(ctx, orgID)
	if err != nil {
		return "", "", "", err
	}

	// 8. Generate opaque session token (requirement 4)
	tokenBytes := make([]byte, 32)
	if _, err := rand.Read(tokenBytes); err != nil {
		return "", "", "", fmt.Errorf("%w: generate session token: %v", ErrInternal, err)
	}
	sessionToken = hex.EncodeToString(tokenBytes)

	// 9. Prepare persistence parameters (requirement 9 — transactional writes)
	slug := generateSlug(payload.SchoolName)
	expiresAt := time.Now().Add(sessionTTL)

	tenantParams := CreateTenantParams{
		Name:        payload.SchoolName,
		Slug:        slug,
		StytchOrgID: orgID,
	}

	userParams := CreateUserParams{
		Email:          strings.ToLower(email), // normalised for case-insensitive unique index
		TenantID:       "",                     // set after tenant creation
		FullName:       payload.FullName,
		ExternalAuthID: result.MemberID,
	}

	sessionParams := CreateSessionParams{
		Token:              sessionToken,
		UserID:             "", // set after user creation
		TenantID:           "", // set after tenant creation
		StytchMemberID:     result.MemberID,
		StytchOrgID:        orgID,
		StytchSessionToken: result.StytchSessionToken,
		DeviceFingerprint:  deviceFingerprint,
		ExpiresAt:          expiresAt,
	}

	// 10. Persist to database — two distinct paths:
	//     - Existing tenant: create user + session only (no tenant insert)
	//     - Fresh tenant: create tenant + user + session in one transaction
	if tenantExists || tenantExistsByName {
		// Tenant already exists — use its existing ID, create user + session only
		if tenantExistsByName {
			tenantID = existingTenantID
		} else {
			// tenantExists (by orgID) but not by name — shouldn't happen with
			// Stytch dedup, but handle gracefully by looking up the tenant ID
			fetchedID, _, err := s.repo.GetTenantByName(ctx, payload.SchoolName)
			if err != nil {
				return "", "", "", err
			}
			tenantID = fetchedID
		}

		userParams.TenantID = tenantID
		sessionParams.TenantID = tenantID

		s.logger.Info("auth: tenant already exists, creating user and session only",
			zap.String("tenant_id", tenantID),
			zap.String("org_id", orgID),
		)
		if userID, err = s.repo.CreateUserSession(ctx, userParams, sessionParams); err != nil {
			return "", "", "", err
		}
	} else {
		// Fresh registration — create all three in a single transaction
		if userID, tenantID, err = s.repo.CreateTenantUserSession(ctx, tenantParams, userParams, sessionParams); err != nil {
			return "", "", "", err
		}
	}

	// 11. Create school using the full cbcschools.Service.CreateSchool pipeline,
	//     which seeds the CBC curriculum, enrolls the creator with the correct
	//     role, and sets the active school — instead of a bare repository INSERT.
	role = "SCHOOL_ADMIN"

	schoolID, err = s.schoolCreator.CreateSchool(ctx, tenantID, payload.SchoolName, role, userID)
	if err != nil {
		return "", "", "", fmt.Errorf("%w: create school: %v", ErrInternal, err)
	}

	s.logger.Info("auth: school and membership created",
		zap.String("school_id", schoolID),
		zap.String("user_id", userID),
		zap.String("role", role),
	)

	// 12. Persist session mapping in Redis: opaque key → Stytch session token (requirement 4)
	if err := s.rdb.Set(ctx, s.sessionKey(sessionToken), result.StytchSessionToken, sessionTTL).Err(); err != nil {
		return "", "", "", fmt.Errorf("%w: cache session: %v", ErrInternal, err)
	}

	s.logger.Info("auth: registration complete — session issued",
		zap.String("org_id", orgID),
		zap.String("session_token_preview", sessionToken[:8]+"..."),
		zap.String("role", role),
		zap.String("school_id", schoolID),
	)

	return sessionToken, role, schoolID, nil
}

// GetMe returns the full profile info for the authenticated user.
func (s *Service) GetMe(ctx context.Context, token string) (*MeInfo, error) {
	if token == "" {
		return nil, ErrExpiredToken
	}

	// Check Redis first (fast path)
	exists, err := s.rdb.Exists(ctx, s.sessionKey(token)).Result()
	if err != nil {
		return nil, fmt.Errorf("%w: check session in cache: %v", ErrInternal, err)
	}
	if exists == 0 {
		return nil, ErrExpiredToken
	}

	info, err := s.repo.GetMeInfo(ctx, token)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			if delErr := s.rdb.Del(ctx, s.sessionKey(token)).Err(); delErr != nil {
				slog.WarnContext(ctx, "auth.Service.GetMe: failed to remove stale session from cache",
					slog.String("error", delErr.Error()),
				)
			}
			return nil, ErrExpiredToken
		}
		return nil, err
	}

	return info, nil
}

// AcceptInvite completes the invite acceptance flow. It validates the Stytch
// magic-link token, looks up the pending invitation, exchanges the IST for a
// full Stytch session, creates the user/session/membership in Postgres,
// caches the session in Redis, and returns the opaque session token, role,
// and school ID.
func (s *Service) AcceptInvite(ctx context.Context, token string, deviceFingerprint string) (sessionToken string, role string, schoolID string, err error) {
	s.logger.Info("auth: accept invite initiated")

	// 1. Authenticate the invite magic link token using the org-scoped endpoint
	//    (POST /v1/b2b/magic_links/authenticate), NOT the discovery endpoint.
	//    The member was already created by InviteMemberByEmail during the invite
	//    sending phase, so the response includes their MemberID and OrganizationID
	//    directly — no IST exchange or discovery flow needed.
	result, err := s.idp.AuthenticateMagicLink(ctx, token)
	if err != nil {
		return "", "", "", fmt.Errorf("auth.Service.AcceptInvite: authenticate: %w", err)
	}

	// 2. If MFA is required, exchange the intermediate session token for a
	//    full session. Otherwise use the session token from the auth response.
	var stytchSessionToken string
	if result.MemberAuthenticated {
		stytchSessionToken = result.StytchSessionToken
	} else {
		s.logger.Info("auth: MFA required for invite acceptance",
			zap.String("member_id", result.MemberID),
		)
		exchangeResult, err := s.idp.ExchangeIntermediateSession(ctx, result.IntermediateSessionToken, result.OrganizationID)
		if err != nil {
			return "", "", "", fmt.Errorf("auth.Service.AcceptInvite: exchange MFA session: %w", err)
		}
		stytchSessionToken = exchangeResult.StytchSessionToken
	}

	// 3. Look up the pending invitation by email (from the authenticated member)
	inv, err := s.repo.GetInvitationByEmail(ctx, result.Email)
	if err != nil {
		// Map not-found to ErrExpiredToken so the frontend gets a 401
		if errors.Is(err, ErrNotFound) {
			return "", "", "", fmt.Errorf("%w: no pending invitation for email: %s", ErrExpiredToken, result.Email)
		}
		return "", "", "", fmt.Errorf("auth.Service.AcceptInvite: lookup invitation: %w", err)
	}

	// 4. Generate opaque session token (32 random bytes, hex-encoded)
	tokenBytes := make([]byte, 32)
	if _, err := rand.Read(tokenBytes); err != nil {
		return "", "", "", fmt.Errorf("%w: generate session token: %v", ErrInternal, err)
	}
	sessionToken = hex.EncodeToString(tokenBytes)

	// 5. Assemble args and persist via the repository transaction.
	//    memberID comes from the AuthenticateMagicLink response — the member was
	//    already provisioned by InviteMemberByEmail during the invite phase, so
	//    no CreateMember/GetMemberByEmail call is needed here.
	expiresAt := time.Now().Add(sessionTTL)
	args := CreateInvitedUserSessionArgs{
		InvitationID:       inv.ID,
		Email:              inv.Email,
		TenantID:           inv.TenantID,
		SchoolID:           inv.SchoolID,
		Role:               inv.Role,
		FullName:           inv.FullName,
		ExternalAuthID:     result.MemberID,
		SessionToken:       sessionToken,
		StytchMemberID:     result.MemberID,
		StytchOrgID:        result.OrganizationID,
		StytchSessionToken: stytchSessionToken,
		DeviceFingerprint:  deviceFingerprint,
		ExpiresAt:          expiresAt,
		TSCNumber:          inv.RegistrationNumber,
	}

	if err := s.repo.CreateInvitedUserSession(ctx, args); err != nil {
		return "", "", "", fmt.Errorf("auth.Service.AcceptInvite: create session: %w", err)
	}

	// 6. Persist session mapping in Redis: opaque key → Stytch session token
	if err := s.rdb.Set(ctx, s.sessionKey(sessionToken), stytchSessionToken, sessionTTL).Err(); err != nil {
		return "", "", "", fmt.Errorf("%w: cache session: %v", ErrInternal, err)
	}

	s.logger.Info("auth: invite acceptance complete — session issued for invited member",
		zap.String("email", result.Email),
		zap.String("member_id", result.MemberID),
		zap.String("org_id", result.OrganizationID),
		zap.String("tenant_id", inv.TenantID),
		zap.String("school_id", inv.SchoolID),
		zap.String("role", inv.Role),
		zap.String("session_token_preview", sessionToken[:8]+"..."),
	)

	return sessionToken, inv.Role, inv.SchoolID, nil
}

// GetSession validates a session token and returns the user session.
// Checks Redis first (fast path), then cross-references Postgres (requirement 6).
func (s *Service) GetSession(ctx context.Context, token string) (*UserSession, error) {
	if token == "" {
		return nil, ErrExpiredToken
	}

	// Fast path: Redis hit — cross-reference with Postgres
	exists, err := s.rdb.Exists(ctx, s.sessionKey(token)).Result()
	if err != nil {
		return nil, fmt.Errorf("%w: check session in cache: %v", ErrInternal, err)
	}
	if exists == 1 {
		session, err := s.repo.GetSessionByToken(ctx, token)
		if err != nil {
			if errors.Is(err, ErrNotFound) {
				// Token in Redis but not in Postgres — clean up stale Redis entry
				if delErr := s.rdb.Del(ctx, s.sessionKey(token)).Err(); delErr != nil {
					slog.WarnContext(ctx, "auth.Service.GetSession: failed to remove stale session from cache",
						slog.String("error", delErr.Error()),
					)
				}
				return nil, ErrExpiredToken
			}
			return nil, err
		}
		return session, nil
	}

	// Cache miss — fall back to Postgres (cold start or stale cache)
	session, err := s.repo.GetSessionByToken(ctx, token)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return nil, ErrExpiredToken
		}
		return nil, err
	}

	// Repopulate Redis from Postgres for subsequent requests
	if setErr := s.rdb.Set(ctx, s.sessionKey(token), session.StytchSessionToken, sessionTTL).Err(); setErr != nil {
		slog.WarnContext(ctx, "auth.Service.GetSession: failed to repopulate session cache",
			slog.String("error", setErr.Error()),
		)
	}

	return session, nil
}

// Logout destroys a session: removes from Redis, deletes from Postgres,
// and the handler clears the cookie (requirement 7).
func (s *Service) Logout(ctx context.Context, token string) error {
	if token == "" {
		return nil // nothing to do
	}

	// Delete from Postgres
	if err := s.repo.DeleteSession(ctx, token); err != nil {
		if !errors.Is(err, ErrNotFound) {
			return fmt.Errorf("auth.Service.Logout: delete session: %w", err)
		}
	}

	// Delete from Redis
	if err := s.rdb.Del(ctx, s.sessionKey(token)).Err(); err != nil {
		return fmt.Errorf("%w: delete session from cache: %v", ErrInternal, err)
	}

	s.logger.Info("auth: session destroyed")
	return nil
}

// ============================================================================
// Internal helpers
// ============================================================================

// reconstructFromStytch rebuilds local state from an existing Stytch organization when our
// database has been wiped. Uses the first discovered org that has a member, exchanges the IST,
// creates the tenant/user/session/school in Postgres, and returns a VerifyResult with a session
// token for direct dashboard redirect.
func (s *Service) reconstructFromStytch(ctx context.Context, ist, email string, discoveredOrgs []DiscoveredOrg, deviceFingerprint string) (*VerifyResult, error) {
	// Pick the first discovered org with a member — all discovered orgs are valid
	// since the user already has memberships in them.
	org := discoveredOrgs[0]

	if !org.MemberAuthenticated {
		s.logger.Warn("auth: reconstruct: MFA required for existing Stytch member",
			zap.String("email", email),
			zap.String("org_id", org.OrganizationID),
		)
		return nil, ErrMFARequired
	}

	// Exchange the IST against this org to get a full Stytch session
	exchangeResult, err := s.idp.ExchangeIntermediateSession(ctx, ist, org.OrganizationID)
	if err != nil {
		return nil, fmt.Errorf("auth.Service.reconstructFromStytch: exchange IST: %w", err)
	}

	if !exchangeResult.MemberAuthenticated {
		s.logger.Warn("auth: reconstruct: MFA required after IST exchange",
			zap.String("email", email),
			zap.String("org_id", org.OrganizationID),
		)
		return nil, ErrMFARequired
	}

	// Use the Stytch org name as the school name for reconstruction
	schoolName := org.OrganizationName
	if schoolName == "" {
		schoolName = email + "'s School"
	}

	// Generate opaque session token
	tokenBytes := make([]byte, 32)
	if _, err := rand.Read(tokenBytes); err != nil {
		return nil, fmt.Errorf("%w: generate session token: %v", ErrInternal, err)
	}
	sessionToken := hex.EncodeToString(tokenBytes)
	expiresAt := time.Now().Add(sessionTTL)

	// Use the member name from Stytch if available, otherwise fall back to email local part
	fullName := org.MemberName
	if fullName == "" {
		// Extract name from email (before @)
		parts := strings.SplitN(email, "@", 2)
		fullName = parts[0]
	}

	// Prepare persistence parameters
	slug := generateSlug(schoolName)

	tenantParams := CreateTenantParams{
		Name:        schoolName,
		Slug:        slug,
		StytchOrgID: org.OrganizationID,
	}

	userParams := CreateUserParams{
		Email:          strings.ToLower(email), // normalised for case-insensitive unique index
		FullName:       fullName,
		ExternalAuthID: exchangeResult.MemberID,
	}

	sessionParams := CreateSessionParams{
		Token:              sessionToken,
		StytchMemberID:     exchangeResult.MemberID,
		StytchOrgID:        org.OrganizationID,
		StytchSessionToken: exchangeResult.StytchSessionToken,
		DeviceFingerprint:  deviceFingerprint,
		ExpiresAt:          expiresAt,
	}

	// Set the stytch_org_id in context for reconciliation logging
	ctx = context.WithValue(ctx, StytchOrgIDKey{}, org.OrganizationID)

	// Create tenant + user + session in a single transaction
	userID, tenantID, err := s.repo.CreateTenantUserSession(ctx, tenantParams, userParams, sessionParams)
	if err != nil {
		return nil, fmt.Errorf("auth.Service.reconstructFromStytch: create tenant/user/session: %w", err)
	}

	// Create the school using the full cbcschools.Service.CreateSchool pipeline
	role := "SCHOOL_ADMIN"
	schoolID, err := s.schoolCreator.CreateSchool(ctx, tenantID, schoolName, role, userID)
	if err != nil {
		return nil, fmt.Errorf("auth.Service.reconstructFromStytch: create school: %w", err)
	}

	// Cache session in Redis
	if err := s.rdb.Set(ctx, s.sessionKey(sessionToken), exchangeResult.StytchSessionToken, sessionTTL).Err(); err != nil {
		return nil, fmt.Errorf("%w: cache session: %v", ErrInternal, err)
	}

	s.logger.Info("auth: Stytch org reconstructed in DB — session issued",
		zap.String("user_id", userID),
		zap.String("tenant_id", tenantID),
		zap.String("school_id", schoolID),
		zap.String("role", role),
		zap.String("email", email),
		zap.String("stytch_org_id", org.OrganizationID),
		zap.String("session_token_preview", sessionToken[:8]+"..."),
	)

	return &VerifyResult{
		SessionToken: sessionToken,
		Role:         role,
		Email:        email,
		SchoolID:     schoolID,
	}, nil
}

// readAndDeleteIST atomically reads and deletes the IST+email JSON from Redis (requirement 2).
func (s *Service) readAndDeleteIST(ctx context.Context, sessionRef string) (ist, email string, err error) {
	istKey := fmt.Sprintf("%s%s:%s", istKeyPrefix, s.cfg.AppEnv, sessionRef)

	// Use a Lua script for atomic GET + DEL to prevent TOCTOU race conditions
	script := redis.NewScript(`
		local val = redis.call("GET", KEYS[1])
		if val then
			redis.call("DEL", KEYS[1])
		end
		return val
	`)

	val, err := script.Run(ctx, s.rdb, []string{istKey}).Result()
	if err != nil {
		// redis.Nil means the key didn't exist (Lua `return false`).
		// Map this to ErrExpiredToken so the caller gets a 401, not a 500.
		if errors.Is(err, redis.Nil) {
			return "", "", fmt.Errorf("%w: IST not found or already consumed", ErrExpiredToken)
		}
		return "", "", fmt.Errorf("%w: atomic read-delete ist: %v", ErrInternal, err)
	}
	if val == nil {
		return "", "", fmt.Errorf("%w: IST not found or already consumed", ErrExpiredToken)
	}

	valStr, ok := val.(string)
	if !ok || valStr == "" {
		return "", "", fmt.Errorf("%w: invalid IST value in cache", ErrInternal)
	}

	// Decode the JSON payload (backward compatible: plain IST string also accepted)
	var data istCacheData
	if err := json.Unmarshal([]byte(valStr), &data); err != nil || data.IST == "" || data.Email == "" {
		// Not valid JSON cache data — treat the whole value as a plain IST (legacy format)
		data = istCacheData{IST: valStr, Email: ""}
	}

	s.logger.Info("auth: IST and email consumed from Redis",
		zap.String("session_ref", sessionRef),
		zap.String("email", data.Email),
	)

	return data.IST, data.Email, nil
}

// sessionKey returns the Redis key for a session token.
func (s *Service) sessionKey(token string) string {
	return sessionPrefix + token
}

// generateUUID generates a UUID v4 string using crypto/rand.
func generateUUID() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	// Set version 4
	b[6] = (b[6] & 0x0f) | 0x40
	// Set variant bits
	b[8] = (b[8] & 0x3f) | 0x80
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
		b[0:4], b[4:6], b[6:8], b[8:10], b[10:16]), nil
}

// SwitchActiveSchool switches the active school for a user.
// Returns the new school_id on success.
func (s *Service) SwitchActiveSchool(ctx context.Context, userID, tenantID, schoolID string) (string, error) {
	if userID == "" || tenantID == "" || schoolID == "" {
		return "", fmt.Errorf("auth.Service.SwitchActiveSchool: all parameters required: %w", ErrInvalidInput)
	}
	if err := s.repo.SetActiveSchool(ctx, userID, tenantID, schoolID); err != nil {
		return "", fmt.Errorf("auth.Service.SwitchActiveSchool: %w", err)
	}
	return schoolID, nil
}
