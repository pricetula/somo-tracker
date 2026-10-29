package handler

import (
	"errors"
	"strings"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"somotracker/internal/identity/service"
	"somotracker/internal/platform/apperr"
	"somotracker/internal/platform/config"
	"somotracker/internal/platform/logger"
	"somotracker/internal/platform/ratelimit"
)

// Handler is HTTP handler for identity domain (thin, BE-006).
type Handler struct {
	svc     *service.Service
	cfg     *config.Config
	rl      *ratelimit.Limiter
}

// New creates handler.
func New(svc *service.Service, cfg *config.Config, rl *ratelimit.Limiter) *Handler {
	return &Handler{svc: svc, cfg: cfg, rl: rl}
}

// DTOs are defined per endpoint, not domain structs (BE-007).

type RequestMagicLinkRequest struct {
	Email string `json:"email" validate:"required,email"`
}

type RequestMagicLinkResponse struct {
	Success bool `json:"success"`
}

type AuthenticateTokenRequest struct {
	Token string `json:"token" validate:"required"`
}

type AuthenticateTokenResponse struct {
	Status      string            `json:"status"`
	AuthState   string            `json:"auth_state"`
	TicketToken string            `json:"ticket_token,omitempty"`
	Onboarding  *OnboardingState  `json:"onboarding,omitempty"`
}

type OnboardingState struct {
	VerifiedEmail string                 `json:"verified_email"`
	CurrentStep   string                 `json:"current_step"`
	FormData      map[string]interface{} `json:"form_data"`
	PendingID     string                 `json:"pending_id"`
	SchoolName    *string                `json:"school_name,omitempty"`
	TenantID      *string                `json:"tenant_id,omitempty"`
}

type ProvisionTenantRequest struct {
	SchoolName     string `json:"school_name" validate:"required,min=1,max=255"`
	IdempotencyKey string `json:"idempotency_key" validate:"required"`
}

type ProvisionTenantResponse struct {
	Success bool `json:"success"`
}

type LogoutRequest struct {
	// Empty body for CSRF protection
}

type LogoutResponse struct {
	Success bool `json:"success"`
}

type GetOnboardingCurrentResponse struct {
	VerifiedEmail string                 `json:"verified_email"`
	CurrentStep   string                 `json:"current_step"`
	FormData      map[string]interface{} `json:"form_data"`
	PendingID     string                 `json:"pending_id"`
	SchoolName    *string                `json:"school_name,omitempty"`
	TenantID      *string                `json:"tenant_id,omitempty"`
}

type UpdateOnboardingResponse struct {
	Success bool `json:"success"`
}

// RequestMagicLink handles POST /api/v1/auth/request-magic-link
func (h *Handler) RequestMagicLink(c fiber.Ctx) error {
	var req RequestMagicLinkRequest
	if err := c.Bind().Body(&req); err != nil {
		return apperr.New(apperr.INVALID_REQUEST, "invalid request body")
	}

	// Basic email validation
	if req.Email == "" || !contains(req.Email, "@") {
		return apperr.New(apperr.INVALID_EMAIL, "invalid email address")
	}

	// Rate limiting and provider quotas
	if h.rl != nil {
		ip := c.IP()
		allowed, err := h.rl.AllowIP(c.Context(), "magic-link", ip, h.cfg.RateLimitMagicLinkIPPerMin, time.Minute)
		if err != nil {
			logger.Error(c.Context(), err, "rate limit check failed")
			return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{"message": "If the email address is eligible, a sign-in link will be sent."})
		}
		if !allowed {
			return c.Status(fiber.StatusTooManyRequests).JSON(fiber.Map{"message": "Too many requests"})
		}

		allowed, err = h.rl.AllowEmail(c.Context(), "magic-link", req.Email, h.cfg.RateLimitMagicLinkEmailPer15Min, 15*time.Minute)
		if err != nil {
			logger.Error(c.Context(), err, "email rate limit check failed")
			return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{"message": "If the email address is eligible, a sign-in link will be sent."})
		}
		if !allowed {
			return c.Status(fiber.StatusTooManyRequests).JSON(fiber.Map{"message": "Too many requests"})
		}

		allowed, err = h.rl.AllowGlobal(c.Context(), "email-provider-hourly", h.cfg.EmailProviderHourlyCap, time.Hour)
		if err != nil {
			logger.Error(c.Context(), err, "email provider hourly quota check failed")
			return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{"message": "If the email address is eligible, a sign-in link will be sent."})
		}
		if !allowed {
			return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{"message": "If the email address is eligible, a sign-in link will be sent."})
		}
		allowed, err = h.rl.AllowGlobal(c.Context(), "email-provider-daily", h.cfg.EmailProviderDailyCap, 24*time.Hour)
		if err != nil {
			logger.Error(c.Context(), err, "email provider daily quota check failed")
			return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{"message": "If the email address is eligible, a sign-in link will be sent."})
		}
		if !allowed {
			return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{"message": "If the email address is eligible, a sign-in link will be sent."})
		}
	}

	// Call service - returns PKCE verifier to store in cookie
	pkceVerifier, err := h.svc.RequestMagicLink(c.Context(), req.Email)
	if err != nil {
		// Don't reveal if email exists (generic success per spec)
		// Log error internally
		logger.Error(c.Context(), err, "RequestMagicLink failed")
		return c.Status(fiber.StatusOK).JSON(RequestMagicLinkResponse{Success: true})
	}

	// Store PKCE verifier in HttpOnly cookie for callback
	c.Cookie(&fiber.Cookie{
		Name:     "__Host-pkce_verifier",
		Value:    pkceVerifier,
		Path:     "/",
		HTTPOnly: true,
		Secure:   h.cfg.Env == "production",
		SameSite: "Lax",
		MaxAge:   int(10 * time.Minute / time.Second),
	})

	// Generic success - don't reveal account existence
	return c.Status(fiber.StatusOK).JSON(RequestMagicLinkResponse{Success: true})
}

// AuthenticateToken handles POST /api/v1/auth/authenticate-token
func (h *Handler) AuthenticateToken(c fiber.Ctx) error {
	var req AuthenticateTokenRequest
	if err := c.Bind().Body(&req); err != nil {
		return apperr.New(apperr.INVALID_REQUEST, "invalid request body")
	}

	// IP rate limit for token exchange
	if h.rl != nil {
		ip := c.IP()
		allowed, err := h.rl.AllowIP(c.Context(), "token-exchange", ip, h.cfg.RateLimitTokenExchangeIPPerMin, time.Minute)
		if err != nil {
			logger.Error(c.Context(), err, "token exchange rate limit check failed")
			return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{"message": "authentication failed"})
		}
		if !allowed {
			return c.Status(fiber.StatusTooManyRequests).JSON(fiber.Map{"message": "Too many requests"})
		}
	}

	// Get PKCE verifier from cookie
	pkceVerifier := c.Cookies("__Host-pkce_verifier")
	if pkceVerifier == "" {
		return apperr.New(apperr.MISSING_PKCE, "PKCE verifier missing, please restart login flow")
	}

	// Call service
	ticketToken, onboardingState, err := h.svc.AuthenticateToken(c.Context(), req.Token, pkceVerifier)
	if err != nil {
		if errors.Is(err, service.ErrInvalidToken) || errors.Is(err, service.ErrRateLimited) {
			return apperr.New(apperr.INVALID_TOKEN, "invalid or expired magic link")
		}
		logger.Error(c.Context(), err, "AuthenticateToken failed")
		return apperr.New(apperr.INTERNAL_ERROR, "authentication failed")
	}

	// Clear PKCE cookie
	c.Cookie(&fiber.Cookie{
		Name:     "__Host-pkce_verifier",
		Value:    "",
		Path:     "/",
		HTTPOnly: true,
		Secure:   h.cfg.Env == "production",
		SameSite: "Lax",
		MaxAge:   -1,
	})

	// Set onboarding ticket cookie (host-only, short-lived) per AC-3/AC-9
	c.Cookie(&fiber.Cookie{
		Name:     "__Host-onboarding",
		Value:    ticketToken,
		Path:     "/",
		HTTPOnly: true,
		Secure:   h.cfg.Env == "production",
		SameSite: "Lax",
		MaxAge:   int(10 * time.Minute / time.Second),
	})

	// Build response
	resp := AuthenticateTokenResponse{
		Status:      onboardingState.CurrentStep,
		AuthState:   onboardingState.AuthState,
		TicketToken: ticketToken,
	}

	if onboardingState != nil {
		resp.Onboarding = &OnboardingState{
			VerifiedEmail: onboardingState.VerifiedEmail,
			CurrentStep:   onboardingState.CurrentStep,
			FormData:      onboardingState.FormData,
			PendingID:     onboardingState.PendingID.String(),
		}
		if onboardingState.SchoolName != nil {
			resp.Onboarding.SchoolName = onboardingState.SchoolName
		}
		if onboardingState.TenantID != nil {
			id := onboardingState.TenantID.String()
			resp.Onboarding.TenantID = &id
		}
	}

	return c.Status(fiber.StatusOK).JSON(resp)
}

// ProvisionTenant handles POST /api/v1/onboarding/complete
func (h *Handler) ProvisionTenant(c fiber.Ctx) error {
	// Get ticket token from cookie (set by callback page)
	ticketToken := c.Cookies("__Host-onboarding")
	if ticketToken == "" {
		return apperr.New(apperr.MISSING_TICKET, "onboarding ticket missing, please restart login flow")
	}

	var req ProvisionTenantRequest
	if err := c.Bind().Body(&req); err != nil {
		return apperr.New(apperr.INVALID_REQUEST, "invalid request body")
	}

	// Call service
	data := service.ProvisionData{
		SchoolName:     req.SchoolName,
		IdempotencyKey: req.IdempotencyKey,
		AdminName:      req.SchoolName, // Use school name as admin name for now
	}
	result, err := h.svc.ProvisionTenant(c.Context(), ticketToken, data)
	if err != nil {
		if errors.Is(err, service.ErrInvalidToken) {
			return apperr.New(apperr.INVALID_TICKET, "invalid or expired onboarding ticket")
		}
		if errors.Is(err, service.ErrIdempotencyConflict) {
			return apperr.New(apperr.IDEMPOTENCY_CONFLICT, "onboarding already completed with this key")
		}
		logger.Error(c.Context(), err, "ProvisionTenant failed")
		return apperr.New(apperr.PROVISIONING_FAILED, "tenant provisioning failed")
	}

	// Clear onboarding ticket cookie
	c.Cookie(&fiber.Cookie{
		Name:     "__Host-onboarding",
		Value:    "",
		Path:     "/",
		HTTPOnly: true,
		Secure:   h.cfg.Env == "production",
		SameSite: "Lax",
		MaxAge:   -1,
	})

	// Create session after provisioning
	// Parse expires from Stytch (for now use 30 days)
	expiresAt := time.Now().Add(30 * 24 * time.Hour)
	sessionToken, err := h.svc.CreateSession(c.Context(), *result.UserID, *result.TenantID, "", expiresAt)
	if err != nil {
		logger.Error(c.Context(), err, "CreateSession failed")
		return apperr.New(apperr.SESSION_CREATE_FAILED, "failed to create session")
	}

	// Set session cookie
	h.setSessionCookie(c, sessionToken, expiresAt)

	return c.Status(fiber.StatusOK).JSON(ProvisionTenantResponse{Success: true})
}

// Logout handles POST /api/v1/auth/logout
func (h *Handler) Logout(c fiber.Ctx) error {
	// Get session from context (set by auth middleware)
	sessionID, ok := c.Locals("session_id").(uuid.UUID)
	if !ok {
		return apperr.New(apperr.UNAUTHENTICATED, "no active session")
	}

	// Call service to revoke session
	if err := h.svc.Logout(c.Context(), sessionID); err != nil {
		logger.Error(c.Context(), err, "Logout failed")
	}

	// Clear session cookie
	h.clearSessionCookie(c)

	return c.Status(fiber.StatusOK).JSON(LogoutResponse{Success: true})
}

// GetOnboardingCurrent handles GET /api/v1/onboarding/current
func (h *Handler) GetOnboardingCurrent(c fiber.Ctx) error {
	ticketToken := c.Cookies("__Host-onboarding")
	if ticketToken == "" {
		return apperr.New(apperr.UNAUTHENTICATED, "no onboarding session")
	}

	state, err := h.svc.GetOnboardingCurrent(c.Context(), ticketToken)
	if err != nil {
		if errors.Is(err, service.ErrInvalidToken) || errors.Is(err, service.ErrPendingOnboardingNotFound) {
			return apperr.New(apperr.INVALID_TICKET, "invalid or expired onboarding ticket")
		}
		logger.Error(c.Context(), err, "GetOnboardingCurrent failed")
		return apperr.New(apperr.INTERNAL_ERROR, "failed to load onboarding state")
	}

	resp := GetOnboardingCurrentResponse{
		VerifiedEmail: state.VerifiedEmail,
		CurrentStep:   state.CurrentStep,
		FormData:      state.FormData,
		PendingID:     state.PendingID.String(),
		SchoolName:    state.SchoolName,
	}
	if state.TenantID != nil {
		id := state.TenantID.String()
		resp.TenantID = &id
	}
	return c.Status(fiber.StatusOK).JSON(resp)
}

// UpdateOnboardingCurrent handles PUT /api/v1/onboarding/current
func (h *Handler) UpdateOnboardingCurrent(c fiber.Ctx) error {
	ticketToken := c.Cookies("__Host-onboarding")
	if ticketToken == "" {
		return apperr.New(apperr.UNAUTHENTICATED, "no onboarding session")
	}

	var req struct {
		CurrentStep string                 `json:"current_step" validate:"required"`
		FormData    map[string]interface{} `json:"form_data"`
		SchoolName  string                 `json:"school_name,omitempty"`
	}
	if err := c.Bind().Body(&req); err != nil {
		return apperr.New(apperr.INVALID_REQUEST, "invalid request body")
	}

	state, err := h.svc.GetOnboardingCurrent(c.Context(), ticketToken)
	if err != nil {
		if errors.Is(err, service.ErrInvalidToken) || errors.Is(err, service.ErrPendingOnboardingNotFound) {
			return apperr.New(apperr.INVALID_TICKET, "invalid or expired onboarding ticket")
		}
		logger.Error(c.Context(), err, "UpdateOnboardingCurrent failed")
		return apperr.New(apperr.INTERNAL_ERROR, "failed to load onboarding state")
	}

	if req.SchoolName != "" {
		state.SchoolName = &req.SchoolName
	}

	if err := h.svc.UpdateOnboardingStep(c.Context(), state.PendingID, req.CurrentStep, req.FormData); err != nil {
		logger.Error(c.Context(), err, "UpdateOnboardingStep failed")
		return apperr.New(apperr.INTERNAL_ERROR, "failed to update onboarding step")
	}

	return c.Status(fiber.StatusOK).JSON(UpdateOnboardingResponse{Success: true})
}

// RegisterRoutes registers identity routes under /api/v1
func (h *Handler) RegisterRoutes(api fiber.Router) {
	auth := api.Group("/auth")
	auth.Post("/request-magic-link", h.RequestMagicLink)
	auth.Post("/authenticate-token", h.AuthenticateToken)
	auth.Post("/logout", h.Logout)

	onboarding := api.Group("/onboarding")
	onboarding.Get("/current", h.GetOnboardingCurrent)
	onboarding.Put("/current", h.UpdateOnboardingCurrent)
	onboarding.Post("/complete", h.ProvisionTenant)
}

// sessionCookieName returns the cookie name with environment suffix per ADR-0002
func (h *Handler) sessionCookieName() string {
	base := "__Secure-session_token"
	if h.cfg.Env != "" && h.cfg.Env != "production" {
		return base + "_" + h.cfg.Env
	}
	return base
}

// sessionCookieDomain returns the cookie domain from config
func (h *Handler) sessionCookieDomain() string {
	// In production, set to shared domain (e.g., "example.com")
	// For now, return empty string (no Domain attribute = host-only)
	// TODO: Add cookie domain to config when production domain is known
	return ""
}

// setSessionCookie sets the session cookie with proper attributes
func (h *Handler) setSessionCookie(c fiber.Ctx, token string, expiresAt time.Time) {
	cookieName := h.sessionCookieName()
	cookieDomain := h.sessionCookieDomain()

	maxAge := int(time.Until(expiresAt).Seconds())
	if maxAge < 0 {
		maxAge = 0
	}

	c.Cookie(&fiber.Cookie{
		Name:     cookieName,
		Value:    token,
		Path:     "/",
		Domain:   cookieDomain,
		HTTPOnly: true,
		Secure:   h.cfg.Env == "production",
		SameSite: "Lax",
		MaxAge:   maxAge,
	})
}

// clearSessionCookie clears the session cookie
func (h *Handler) clearSessionCookie(c fiber.Ctx) {
	cookieName := h.sessionCookieName()
	cookieDomain := h.sessionCookieDomain()
	c.Cookie(&fiber.Cookie{
		Name:     cookieName,
		Value:    "",
		Path:     "/",
		Domain:   cookieDomain,
		HTTPOnly: true,
		Secure:   h.cfg.Env == "production",
		SameSite: "Lax",
		MaxAge:   -1,
	})
}

// contains checks if a string contains a substring
func contains(s, substr string) bool {
	return strings.Contains(s, substr)
}