package handler

import (
	"crypto/sha256"
	"net/http"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"

	"somotracker/internal/auth/service"
	"somotracker/internal/platform/apperr"
	"somotracker/internal/platform/config"
	"somotracker/internal/platform/logger"
)

// Handler is the HTTP handler for the auth domain.
type Handler struct {
	svc               *service.Service
	cfg               *config.Config
	log               *logger.Logger
	sessionCookieName string
}

// New creates an auth handler.
func New(svc *service.Service, cfg *config.Config, log *logger.Logger) *Handler {
	return &Handler{
		svc:               svc,
		cfg:               cfg,
		log:               log,
		sessionCookieName: "__Host-session",
	}
}

// RequestOTPRequest is the request body for POST /api/v1/auth/otp/request.
type RequestOTPRequest struct {
	Email          string `json:"email" validate:"required,email"`
	TurnstileToken string `json:"turnstile_token,omitempty"`
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
type VerifyOTPResponse struct {
	OK             bool   `json:"ok"`
	Message        string `json:"message"`
	SessionToken   string `json:"session_token,omitempty"`
	NeedsOnboarding bool  `json:"needs_onboarding"`
}

// RequestOTP handles POST /api/v1/auth/otp/request.
func (h *Handler) RequestOTP(c fiber.Ctx) error {
	var req RequestOTPRequest
	if err := c.Bind().Body(&req); err != nil {
		return apperr.New(apperr.INVALID_REQUEST, "invalid request body")
	}

	resp, err := h.svc.RequestOTP(c.Context(), service.RequestOTPRequest{
		Email:          req.Email,
		TurnstileToken: req.TurnstileToken,
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
			c.Cookie(&fiber.Cookie{
				Name:     "needs_onboarding",
				Value:    "true",
				Path:     "/",
				HTTPOnly: true,
				Secure:   c.Scheme() == "https",
				SameSite: fiber.CookieSameSiteLaxMode,
				MaxAge:   int(sessionTTL.Seconds()),
			})
		}
		h.setSessionCookie(c, resp.SessionToken, time.Now().Add(sessionTTL))
	}

	return c.Status(http.StatusOK).JSON(resp)
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
	clearOnboardingCookie(c)

	return c.Status(http.StatusOK).JSON(fiber.Map{
		"ok":               true,
		"message":          "Onboarding complete",
		"needs_onboarding": false,
	})
}

// SessionResponse is the response body for GET /api/v1/auth/session.
type SessionResponse struct {
	OK             bool   `json:"ok"`
	Authenticated  bool   `json:"authenticated"`
	SessionType    string `json:"session_type,omitempty"`
	NeedsOnboarding bool  `json:"needs_onboarding,omitempty"`
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
		Secure:   c.Scheme() == "https",
		SameSite: fiber.CookieSameSiteLaxMode,
		MaxAge:   maxAge,
	})
}

// clearSessionCookie clears the session cookie.
func (h *Handler) clearSessionCookie(c fiber.Ctx) {
	c.Cookie(&fiber.Cookie{
		Name:     h.sessionCookieName,
		Value:    "",
		Path:     "/",
		HTTPOnly: true,
		Secure:   c.Scheme() == "https",
		SameSite: fiber.CookieSameSiteLaxMode,
		MaxAge:   -1,
	})
}

// clearOnboardingCookie removes the needs_onboarding flag cookie.
func clearOnboardingCookie(c fiber.Ctx) {
	c.Cookie(&fiber.Cookie{
		Name:     "needs_onboarding",
		Value:    "",
		Path:     "/",
		HTTPOnly: true,
		Secure:   c.Scheme() == "https",
		SameSite: fiber.CookieSameSiteLaxMode,
		MaxAge:   -1,
	})
}
