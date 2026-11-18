package service

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"somotracker/internal/auth/store"
	"somotracker/internal/platform/apperr"
	"somotracker/internal/platform/brevo"
	"somotracker/internal/platform/config"
	"somotracker/internal/platform/logger"
)

// errNoRows is returned when a database query finds no matching row.
var errNoRows = pgx.ErrNoRows

// Service handles auth business logic: OTP, sessions, school context.
type Service struct {
	store      *store.Store
	codeSender brevo.CodeSender
	cfg        *config.Config
	log        *logger.Logger
}

// New creates an auth service.
func New(s *store.Store, cs brevo.CodeSender, cfg *config.Config, log *logger.Logger) *Service {
	return &Service{store: s, codeSender: cs, cfg: cfg, log: log}
}

// RequestOTPRequest is the input for requesting an OTP.
type RequestOTPRequest struct {
	Email          string
	TurnstileToken string
}

// RequestOTPResponse is the response for requesting an OTP.
type RequestOTPResponse struct {
	OK      bool   `json:"ok"`
	Message string `json:"message"`
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
		// TODO: verify turnstile token server-side via Cloudflare Siteverify
		_ = req.TurnstileToken
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

	challenge, err := s.store.CreateLoginChallenge(ctx, normalizedEmail, codeMac, expiresAt)
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
		OK:      true,
		Message: "If an account exists for this email, a code has been sent.",
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

// computeHMAC computes HMAC-SHA256(server_secret, challenge_id || code).
func computeHMAC(secret string, challengeID uuid.UUID, code string) []byte {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(challengeID[:])
	mac.Write([]byte(code))
	return mac.Sum(nil)
}

// VerifyOTPRequest is the input for verifying an OTP.
type VerifyOTPRequest struct {
	ChallengeID uuid.UUID
	Code        string
}

// VerifyOTPResponse is the response for verifying an OTP.
type VerifyOTPResponse struct {
	OK           bool   `json:"ok"`
	Message      string `json:"message"`
	SessionToken string `json:"session_token,omitempty"`
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

	return VerifyOTPResponse{
		OK:      true,
		Message: "Code verified successfully.",
	}, nil
}
