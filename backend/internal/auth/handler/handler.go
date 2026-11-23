package handler

import (
	"context"
	"crypto/sha256"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"

	"somotracker/internal/auth/service"
	"somotracker/internal/platform/apperr"
	"somotracker/internal/platform/config"
	"somotracker/internal/platform/logger"
	"somotracker/internal/platform/turnstile"
)

// Handler is the HTTP handler for the auth domain.
type Handler struct {
	svc                  *service.Service
	cfg                  *config.Config
	log                  *logger.Logger
	turnstile            turnstile.Verifier
	sessionCookieName    string
	onboardingCookieName string
}

// New creates an auth handler.
func New(svc *service.Service, cfg *config.Config, log *logger.Logger) *Handler {
	sessionCookieName := cfg.SessionCookieName
	// Derive onboarding cookie name from session cookie name
	onboardingCookieName := deriveOnboardingCookieName(sessionCookieName)

	// Turnstile verifier for OTP request protection
	var ts turnstile.Verifier
	if cfg.TurnstileSecretKey != "" {
		ts = turnstile.New(turnstile.Config{
			SecretKey:    cfg.TurnstileSecretKey,
			ExpectedHost: cfg.TurnstileExpectedHost,
		})
	}

	log.Info(context.Background(), "auth handler initialized",
		"session_cookie", sessionCookieName,
		"onboarding_cookie", onboardingCookieName,
		"app_env", cfg.AppEnv,
		"is_production", cfg.IsProduction(),
		"turnstile_enabled", ts != nil,
	)

	return &Handler{
		svc:                  svc,
		cfg:                  cfg,
		log:                  log,
		turnstile:            ts,
		sessionCookieName:    sessionCookieName,
		onboardingCookieName: onboardingCookieName,
	}
}

// deriveOnboardingCookieName derives the onboarding cookie name from session cookie name.
// If session cookie starts with "__Host-", onboarding cookie must also start with "__Host-".
func deriveOnboardingCookieName(sessionCookieName string) string {
	if strings.HasPrefix(sessionCookieName, "__Host-") {
		return "__Host-needs_onboarding"
	}
	return "needs_onboarding"
}

// RequestOTPRequest is the request body for POST /api/v1/auth/otp/request.
type RequestOTPRequest struct {
	Email          string `json:"email" validate:"required,email"`
	TurnstileToken string `json:"turnstile_token" validate:"required"`
}

// RequestOTPResponse is the response body.
type RequestOTPResponse struct {
	OK           bool   `json:"ok"`
	Message      string `json:"message"`
	ChallengeID  string `json:"challenge_id"`
}

// VerifyOTPRequest is the request body for POST /api/v1/auth/otp/verify.
type VerifyOTPRequest struct {
	ChallengeID string `json:"challenge_id" validate:"required,uuid"`
	Code        string `json:"code" validate:"required,len=6"`
}

// VerifyOTPResponse is the response body.
// SessionToken is NOT returned to the browser - it's only set via HttpOnly cookie.
type VerifyOTPResponse struct {
	OK              bool   `json:"ok"`
	Message         string `json:"message"`
	NeedsOnboarding bool   `json:"needs_onboarding"`
}

// RequestOTP handles POST /api/v1/auth/otp/request.
func (h *Handler) RequestOTP(c fiber.Ctx) error {
	var req RequestOTPRequest
	if err := c.Bind().Body(&req); err != nil {
		return apperr.New(apperr.INVALID_REQUEST, "invalid request body")
	}

	// Require a Turnstile token before doing anything else.
	if req.TurnstileToken == "" {
		return apperr.New(apperr.INVALID_REQUEST, "turnstile token required")
	}

	// Verify the Turnstile token with Cloudflare before generating an OTP,
	// touching the DB, or sending any email.
	if h.turnstile != nil {
		// Use CF-Connecting-IP when behind Cloudflare, otherwise fall back to
		// the connection's remote IP.
		remoteIP := c.Get("CF-Connecting-IP")
		if remoteIP == "" {
			remoteIP = c.IP()
		}

		result, err := h.turnstile.Verify(c.Context(), req.TurnstileToken, remoteIP)
		if err != nil {
			if errors.Is(err, turnstile.ErrUpstream) {
				return apperr.New(apperr.BAD_GATEWAY, "turnstile verification unavailable")
			}
			// Verification failed (success: false or hostname mismatch).
			// Log error codes for debugging but do not return them to the client.
			h.log.Error(c.Context(), err, "turnstile verification failed",
				"error_codes", result.ErrorCodes,
				"hostname", result.Hostname,
			)
			return apperr.New(apperr.FORBIDDEN, "turnstile verification failed")
		}
	}

	resp, err := h.svc.RequestOTP(c.Context(), service.RequestOTPRequest{
		Email: req.Email,
	})
	if err != nil {
		return err
	}

	return c.Status(http.StatusOK).JSON(resp)
}

// VerifyOTP handles POST /api/v1/auth/otp/verify.
func (h *Handler) VerifyOTP(c fiber.Ctx) error {
	var req VerifyOTPRequest
	if err := c.Bind().Body(&req); err != nil {
		return apperr.New(apperr.INVALID_REQUEST, "invalid request body")
	}

	challengeID, err := uuid.Parse(req.ChallengeID)
	if err != nil {
		return apperr.New(apperr.INVALID_REQUEST, "invalid challenge ID")
	}

	resp, err := h.svc.VerifyOTP(c.Context(), service.VerifyOTPRequest{
		ChallengeID: challengeID,
		Code:        req.Code,
	})
	if err != nil {
		return err
	}

	// If verification succeeded and a session token was returned, set the cookie
	if resp.OK && resp.SessionToken != "" {
		sessionTTL := h.cfg.SessionAbsoluteTimeout()
		if resp.NeedsOnboarding {
			sessionTTL = 30 * time.Minute
			// Set onboarding flag cookie so frontend proxy can route without
			// calling the backend. HttpOnly so JS can't read it, but proxy.ts
			// (server-side) can.
			h.setOnboardingCookie(c, "true", int(sessionTTL.Seconds()))
		}
		h.setSessionCookie(c, resp.SessionToken, time.Now().Add(sessionTTL))
	}

	// Return response WITHOUT session_token - it's in the HttpOnly cookie
	return c.Status(http.StatusOK).JSON(VerifyOTPResponse{
		OK:              resp.OK,
		Message:         resp.Message,
		NeedsOnboarding: resp.NeedsOnboarding,
	})
}

// OnboardRequest is the request body for POST /api/v1/auth/onboard.
type OnboardRequest struct {
	OrganizationName string `json:"organization_name" validate:"required"`
	SchoolName       string `json:"school_name" validate:"required"`
	UserName         string `json:"user_name" validate:"required"`
}

// Onboard handles POST /api/v1/auth/onboard. Completes organization onboarding
// and upgrades the session from temporary to full. Requires a valid temporary
// session cookie.
func (h *Handler) Onboard(c fiber.Ctx) error {
	var req OnboardRequest
	if err := c.Bind().Body(&req); err != nil {
		return apperr.New(apperr.INVALID_REQUEST, "invalid request body")
	}

	// Read session token from cookie
	sessionToken := c.Cookies(h.sessionCookieName)
	if sessionToken == "" {
		return apperr.New(apperr.UNAUTHENTICATED, "session required")
	}

	// Hash the token to look up the session
	tokenHash := service.HashToken(sessionToken)

	session, err := h.svc.Store().GetSessionByTokenHash(c.Context(), tokenHash)
	if err != nil {
		return apperr.New(apperr.UNAUTHENTICATED, "invalid session")
	}

	if session.SessionType != "temporary" {
		return apperr.New(apperr.INVALID_REQUEST, "session is not temporary")
	}

	// Complete onboarding
	if err := h.svc.Onboard(c.Context(), session.UserID, session.OrganizationID, req.OrganizationName, req.SchoolName, req.UserName); err != nil {
		return err
	}

	// Clear onboarding flag cookie
	h.clearOnboardingCookie(c)

	return c.Status(http.StatusOK).JSON(fiber.Map{
		"ok":               true,
		"message":          "Onboarding complete",
		"needs_onboarding": false,
	})
}

// SessionResponse is the response body for GET /api/v1/auth/session.
type SessionResponse struct {
	OK              bool   `json:"ok"`
	Authenticated   bool   `json:"authenticated"`
	SessionType     string `json:"session_type,omitempty"`
	NeedsOnboarding bool   `json:"needs_onboarding,omitempty"`
}

// Session handles GET /api/v1/auth/session. Returns session info for middleware
// to make redirect decisions.
func (h *Handler) Session(c fiber.Ctx) error {
	sessionToken := c.Cookies(h.sessionCookieName)
	if sessionToken == "" {
		return c.Status(http.StatusOK).JSON(SessionResponse{
			OK:           true,
			Authenticated: false,
		})
	}

	tokenHash := service.HashToken(sessionToken)

	session, err := h.svc.Store().GetSessionByTokenHash(c.Context(), tokenHash)
	if err != nil {
		return c.Status(http.StatusOK).JSON(SessionResponse{
			OK:           true,
			Authenticated: false,
		})
	}

	// Determine if onboarding is needed based on session type
	needsOnboarding := session.SessionType == "temporary"

	return c.Status(http.StatusOK).JSON(SessionResponse{
		OK:               true,
		Authenticated:    true,
		SessionType:      session.SessionType,
		NeedsOnboarding:  needsOnboarding,
	})
}

// Logout handles POST /api/v1/auth/logout.
func (h *Handler) Logout(c fiber.Ctx) error {
	sessionToken := c.Cookies(h.sessionCookieName)
	if sessionToken != "" {
		tokenHash := sha256.Sum256([]byte(sessionToken))
		_ = tokenHash // TODO: revoke session in service
	}
	h.clearSessionCookie(c)
	h.clearOnboardingCookie(c)
	return c.Status(http.StatusOK).JSON(fiber.Map{
		"ok":      true,
		"message": "logged out",
	})
}

// RegisterRoutes registers auth routes under /api/v1/auth.
func (h *Handler) RegisterRoutes(auth fiber.Router) {
	auth.Post("/otp/request", h.RequestOTP)
	auth.Post("/otp/verify", h.VerifyOTP)
	auth.Post("/onboard", h.Onboard)
	auth.Get("/session", h.Session)
	auth.Post("/logout", h.Logout)
}

// secureCookie returns true if the cookie should be Secure.
// In production (APP_ENV=production), always true because TLS is terminated at load balancer.
// In dev, false for HTTP localhost.
func (h *Handler) secureCookie() bool {
	return h.cfg.IsProduction()
}

// setSessionCookie sets the session cookie on the response.
func (h *Handler) setSessionCookie(c fiber.Ctx, token string, expiresAt time.Time) {
	maxAge := int(time.Until(expiresAt).Seconds())
	if maxAge < 0 {
		maxAge = 0
	}
	c.Cookie(&fiber.Cookie{
		Name:     h.sessionCookieName,
		Value:    token,
		Path:     "/",
		HTTPOnly: true,
		Secure:   h.secureCookie(),
		SameSite: fiber.CookieSameSiteLaxMode,
		MaxAge:   maxAge,
		// NO Domain attribute - __Host- cookies forbid it
	})
}

// clearSessionCookie clears the session cookie.
func (h *Handler) clearSessionCookie(c fiber.Ctx) {
	c.Cookie(&fiber.Cookie{
		Name:     h.sessionCookieName,
		Value:    "",
		Path:     "/",
		HTTPOnly: true,
		Secure:   h.secureCookie(),
		SameSite: fiber.CookieSameSiteLaxMode,
		MaxAge:   -1,
		// NO Domain attribute
	})
}

// setOnboardingCookie sets the needs_onboarding flag cookie.
func (h *Handler) setOnboardingCookie(c fiber.Ctx, value string, maxAgeSeconds int) {
	c.Cookie(&fiber.Cookie{
		Name:     h.onboardingCookieName,
		Value:    value,
		Path:     "/",
		HTTPOnly: true,
		Secure:   h.secureCookie(),
		SameSite: fiber.CookieSameSiteLaxMode,
		MaxAge:   maxAgeSeconds,
		// NO Domain attribute
	})
}

// clearOnboardingCookie removes the needs_onboarding flag cookie.
func (h *Handler) clearOnboardingCookie(c fiber.Ctx) {
	c.Cookie(&fiber.Cookie{
		Name:     h.onboardingCookieName,
		Value:    "",
		Path:     "/",
		HTTPOnly: true,
		Secure:   h.secureCookie(),
		SameSite: fiber.CookieSameSiteLaxMode,
		MaxAge:   -1,
		// NO Domain attribute
	})
}