package service

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/google/uuid"
	"somotracker/internal/auth/store"
	"somotracker/internal/platform/apperr"
	"somotracker/internal/platform/brevo"
	"somotracker/internal/platform/config"
	"somotracker/internal/platform/logger"
)

// errNoRows is returned when a database query finds no matching row.
var errNoRows = sql.ErrNoRows

// Service handles auth business logic: OTP, sessions, school context.
type Service struct {
	store      store.StoreInterface
	codeSender brevo.CodeSender
	cfg        *config.Config
	log        *logger.Logger
}

// New creates an auth service.
func New(s store.StoreInterface, cs brevo.CodeSender, cfg *config.Config, log *logger.Logger) *Service {
	return &Service{store: s, codeSender: cs, cfg: cfg, log: log}
}

// Store returns the underlying store interface.
func (s *Service) Store() store.StoreInterface {
	return s.store
}

// RequestOTPRequest is the input for requesting an OTP.
type RequestOTPRequest struct {
	Email          string
	TurnstileToken string
}

// RequestOTPResponse is the response for requesting an OTP.
type RequestOTPResponse struct {
	OK           bool   `json:"ok"`
	Message      string `json:"message"`
	ChallengeID  string `json:"challenge_id"`
}

// RequestOTP handles the OTP request flow.
func (s *Service) RequestOTP(ctx context.Context, req RequestOTPRequest) (RequestOTPResponse, error) {
	// Normalize email
	normalizedEmail := normalizeEmail(req.Email)

	// Validate email syntax
	if !isValidEmail(normalizedEmail) {
		return RequestOTPResponse{}, apperr.New(apperr.INVALID_EMAIL, "invalid email address")
	}

	// Turnstile verification (if enabled)
	if s.cfg.TurnstileEnabled {
		if req.TurnstileToken == "" {
			return RequestOTPResponse{}, apperr.New(apperr.TURNSTILE_REQUIRED, "turnstile token required")
		}
		if err := s.verifyTurnstile(ctx, req.TurnstileToken); err != nil {
			return RequestOTPResponse{}, err
		}
	}

	// Generate 6-digit OTP code
	code, err := generateOTP()
	if err != nil {
		return RequestOTPResponse{}, apperr.Wrap(apperr.INTERNAL_ERROR, "generate OTP", err)
	}

	// Compute HMAC of challenge_id || code
	challengeID := uuid.New()
	codeMac := computeHMAC(s.cfg.EmailHMACKey, challengeID, code)

	expiresAt := time.Now().Add(s.cfg.OTPValidity())

	challenge, err := s.store.CreateLoginChallenge(ctx, challengeID, normalizedEmail, codeMac, expiresAt)
	if err != nil {
		return RequestOTPResponse{}, apperr.Wrap(apperr.INTERNAL_ERROR, "create login challenge", err)
	}

	// Create email outbox record (encrypted payload placeholder)
	payload := []byte(code) // TODO: encrypt with key outside DB

	_, err = s.store.CreateEmailOutbox(ctx, challenge.ID, payload)
	if err != nil {
		return RequestOTPResponse{}, apperr.Wrap(apperr.INTERNAL_ERROR, "create email outbox", err)
	}

	// Send email via Brevo (fire and forget - don't block response)
	go func() {
		if err := s.codeSender.SendCode(context.Background(), normalizedEmail, code); err != nil {
			s.log.Error(context.Background(), err, "brevo send code failed", "email", normalizedEmail)
		}
	}()

	// Always return generic success - never reveal account existence
	return RequestOTPResponse{
		OK:           true,
		Message:      "If an account exists for this email, a code has been sent.",
		ChallengeID:  challenge.ID.String(),
	}, nil
}

// normalizeEmail lowercases and trims an email address.
func normalizeEmail(email string) string {
	result := ""
	for _, c := range email {
		if c >= 'A' && c <= 'Z' {
			result += string(c + 32)
		} else {
			result += string(c)
		}
	}
	return result
}

// isValidEmail performs basic email syntax validation.
func isValidEmail(email string) bool {
	atIndex := -1
	for i, c := range email {
		if c == '@' {
			atIndex = i
			break
		}
	}
	if atIndex <= 0 || atIndex == len(email)-1 {
		return false
	}
	for i := atIndex + 1; i < len(email); i++ {
		if email[i] == '.' {
			return true
		}
	}
	return false
}

// generateOTP creates a 6-digit numeric OTP.
func generateOTP() (string, error) {
	bytes := make([]byte, 6)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	result := ""
	for _, b := range bytes {
		digit := int(b) % 10
		result += string(rune('0' + digit))
	}
	return result, nil
}

// verifyTurnstile validates a Turnstile token against Cloudflare's Siteverify API.
func (s *Service) verifyTurnstile(ctx context.Context, token string) error {
	if s.cfg.TurnstileSecretKey == "" {
		return apperr.New(apperr.INTERNAL_ERROR, "turnstile not configured")
	}

	data := url.Values{}
	data.Set("secret", s.cfg.TurnstileSecretKey)
	data.Set("response", token)

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		"https://challenges.cloudflare.com/turnstile/v0/siteverify",
		strings.NewReader(data.Encode()),
	)
	if err != nil {
		return apperr.Wrap(apperr.INTERNAL_ERROR, "build turnstile request", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return apperr.Wrap(apperr.INTERNAL_ERROR, "turnstile request failed", err)
	}
	defer func() { _ = res.Body.Close() }()

	var result struct {
		Success     bool     `json:"success"`
		ChallengeTS string   `json:"challenge_ts"`
		Hostname    string   `json:"hostname"`
		ErrorCodes  []string `json:"error-codes"`
	}
	if err := json.NewDecoder(res.Body).Decode(&result); err != nil {
		return apperr.Wrap(apperr.INTERNAL_ERROR, "decode turnstile response", err)
	}

	if !result.Success {
		return apperr.New(apperr.TURNSTILE_INVALID, "turnstile verification failed")
	}

	return nil
}

// computeHMAC computes HMAC-SHA256(server_secret, challenge_id || code).
func computeHMAC(secret string, challengeID uuid.UUID, code string) []byte {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(challengeID[:])
	mac.Write([]byte(code))
	return mac.Sum(nil)
}

// generateSessionToken creates a 256-bit opaque random session token.
func generateSessionToken() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

// HashToken computes SHA-256 of the raw token.
func HashToken(token string) []byte {
	sum := sha256.Sum256([]byte(token))
	return sum[:]
}

// VerifyOTPRequest is the input for verifying an OTP.
type VerifyOTPRequest struct {
	ChallengeID uuid.UUID
	Code        string
}

// VerifyOTPResponse is the response for verifying an OTP.
// SessionToken is internal only (used by handler to set HttpOnly cookie).
// It is NOT serialized to JSON for the browser.
type VerifyOTPResponse struct {
	OK              bool   `json:"ok"`
	Message         string `json:"message"`
	SessionToken    string `json:"-"`
	NeedsOnboarding bool   `json:"needs_onboarding"`
}

// VerifyOTP handles OTP verification and session creation.
func (s *Service) VerifyOTP(ctx context.Context, req VerifyOTPRequest) (VerifyOTPResponse, error) {
	challenge, err := s.store.GetLoginChallengeByID(ctx, req.ChallengeID)
	if err != nil {
		if errors.Is(err, errNoRows) {
			return VerifyOTPResponse{}, apperr.New(apperr.INVALID_TOKEN, "invalid or expired challenge")
		}
		return VerifyOTPResponse{}, apperr.Wrap(apperr.INTERNAL_ERROR, "get challenge", err)
	}

	// Check expiry
	if time.Now().After(challenge.ExpiresAt) {
		return VerifyOTPResponse{}, apperr.New(apperr.INVALID_TOKEN, "challenge expired")
	}

	// Check attempts
	if challenge.AttemptCount >= challenge.MaxAttempts {
		return VerifyOTPResponse{}, apperr.New(apperr.INVALID_TOKEN, "max attempts exceeded")
	}

	// Check already used
	if challenge.UsedAt.Valid {
		return VerifyOTPResponse{}, apperr.New(apperr.INVALID_TOKEN, "challenge already used")
	}

	// Compute expected HMAC
	expectedMac := computeHMAC(s.cfg.EmailHMACKey, challenge.ID, req.Code)

	// Constant-time comparison
	if !hmac.Equal(challenge.CodeMac, expectedMac) {
		return VerifyOTPResponse{}, apperr.New(apperr.INVALID_TOKEN, "invalid code")
	}

	// Mark challenge as used
	if err := s.store.MarkChallengeUsed(ctx, challenge.ID); err != nil {
		return VerifyOTPResponse{}, apperr.Wrap(apperr.INTERNAL_ERROR, "mark challenge used", err)
	}

	// Resolve user by email. If none exists, just-in-time provision a minimal
	// account (org + school + user with placeholder names) and issue a
	// temporary 30-minute session so the user can complete onboarding.
	user, err := s.store.GetUserByNormalizedEmail(ctx, challenge.EmailNormalized)
	justCreated := false
	if err != nil {
		if !errors.Is(err, errNoRows) {
			return VerifyOTPResponse{}, apperr.Wrap(apperr.INTERNAL_ERROR, "get user", err)
		}

		orgName := fmt.Sprintf("org-%s", challenge.ID.String()[:8])
		org, createOrgErr := s.store.CreateOrganization(ctx, orgName)
		if createOrgErr != nil {
			return VerifyOTPResponse{}, apperr.Wrap(apperr.INTERNAL_ERROR, "create organization", createOrgErr)
		}

		_, createSchoolErr := s.store.CreateSchool(ctx, org.ID, "School")
		if createSchoolErr != nil {
			return VerifyOTPResponse{}, apperr.Wrap(apperr.INTERNAL_ERROR, "create school", createSchoolErr)
		}

		user, err = s.store.CreateUser(ctx, org.ID, challenge.EmailNormalized, challenge.EmailNormalized, "")
		if err != nil {
			return VerifyOTPResponse{}, apperr.Wrap(apperr.INTERNAL_ERROR, "create user", err)
		}
		justCreated = true
	}

	// Generate opaque session token
	rawToken, err := generateSessionToken()
	if err != nil {
		return VerifyOTPResponse{}, apperr.Wrap(apperr.INTERNAL_ERROR, "generate session token", err)
	}
	tokenHash := HashToken(rawToken)

	// Get user's schools
	schools, err := s.store.GetSchoolsByUser(ctx, user.ID)
	if err != nil {
		return VerifyOTPResponse{}, apperr.Wrap(apperr.INTERNAL_ERROR, "get user schools", err)
	}

	// Determine active school: auto-select if exactly one, NULL if multiple
	var activeSchoolID uuid.NullUUID
	if len(schools) == 1 {
		activeSchoolID = uuid.NullUUID{UUID: schools[0].ID, Valid: true}
	}

	// Temp session TTL: 30 minutes for onboarding, full timeout otherwise
	sessionTTL := s.cfg.SessionAbsoluteTimeout()
	sessionType := "full"
	if justCreated {
		sessionTTL = 30 * time.Minute
		sessionType = "temporary"
	}

	expiresAt := time.Now().Add(sessionTTL)

	// Create session
	_, err = s.store.CreateSession(ctx, tokenHash, user.ID, user.OrganizationID, activeSchoolID, sessionType, expiresAt)
	if err != nil {
		return VerifyOTPResponse{}, apperr.Wrap(apperr.INTERNAL_ERROR, "create session", err)
	}

	return VerifyOTPResponse{
		OK:             true,
		Message:        "Code verified successfully.",
		SessionToken:   rawToken,
		NeedsOnboarding: justCreated,
	}, nil
}

// Onboard completes organization onboarding: sets org name, school name,
// user name, marks onboarding complete, and upgrades the session from
// temporary to full.
func (s *Service) Onboard(ctx context.Context, userID, orgID uuid.UUID, orgName, schoolName, userName string) error {
	// Update organization name and mark onboarding complete
	if err := s.store.UpdateOrganization(ctx, orgID, orgName); err != nil {
		return apperr.Wrap(apperr.INTERNAL_ERROR, "update organization", err)
	}

	// Update user name
	if err := s.store.UpdateUser(ctx, userID, userName); err != nil {
		return apperr.Wrap(apperr.INTERNAL_ERROR, "update user", err)
	}

	// Create school
	if _, err := s.store.CreateSchool(ctx, orgID, schoolName); err != nil {
		return apperr.Wrap(apperr.INTERNAL_ERROR, "create school", err)
	}

	// Upgrade all temporary sessions for this user to full
	if err := s.store.UpdateSessionType(ctx, userID); err != nil {
		return apperr.Wrap(apperr.INTERNAL_ERROR, "upgrade session", err)
	}

	return nil
}
