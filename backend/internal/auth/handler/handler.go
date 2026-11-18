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
	OK      bool   `json:"ok"`
	Message string `json:"message"`
}

// VerifyOTPRequest is the request body for POST /api/v1/auth/otp/verify.
type VerifyOTPRequest struct {
	ChallengeID string `json:"challenge_id" validate:"required,uuid"`
	Code        string `json:"code" validate:"required,len=6"`
}

// VerifyOTPResponse is the response body.
type VerifyOTPResponse struct {
	OK           bool   `json:"ok"`
	Message      string `json:"message"`
	SessionToken string `json:"session_token,omitempty"`
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
		h.setSessionCookie(c, resp.SessionToken, time.Now().Add(h.cfg.SessionAbsoluteTimeout()))
	}

	return c.Status(http.StatusOK).JSON(resp)
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
