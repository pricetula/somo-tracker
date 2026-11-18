package handler

import (
	"crypto/sha256"
	"errors"
	"strings"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"somotracker/internal/identity/service"
	"somotracker/internal/platform/apperr"
	"somotracker/internal/platform/audit"
	"somotracker/internal/platform/config"
	"somotracker/internal/platform/logger"
)

// Handler is HTTP handler for identity domain (thin, BE-006).
type Handler struct {
	svc           *service.Service
	cfg           *config.Config
	auditRecorder *audit.Recorder
}

// New creates handler.
func New(svc *service.Service, cfg *config.Config, auditRecorder *audit.Recorder) *Handler {
	return &Handler{svc: svc, cfg: cfg, auditRecorder: auditRecorder}
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
	Status      string           `json:"status"`
	AuthState   string           `json:"auth_state"`
	TicketToken string           `json:"ticket_token,omitempty"`
	Onboarding  *OnboardingState `json:"onboarding,omitempty"`
}

type OnboardingState struct {
	VerifiedEmail string                 `json:"verified_email"`
	CurrentStep   string                 `json:"current_step"`
	FormData      map[string]interface{} `json:"form_data"`
	PendingID     string                 `json:"pending_id"`
	SchoolName    *string                `json:"school_name,omitempty"`
	TenantID      *string                `json:"tenant_id,omitempty"`
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
// @Summary Request a Stytch Discovery magic link
// @Tags Authentication
// @Accept json
// @Produce json
// @Param request body RequestMagicLinkRequest true "request"
// @Success 200 {object} RequestMagicLinkResponse
// @Failure 400 {object} map[string]interface{}
// @Failure 429 {object} map[string]interface{}
// @Router /api/v1/auth/request-magic-link [post]
func (h *Handler) RequestMagicLink(c fiber.Ctx) error {
	var req RequestMagicLinkRequest
	if err := c.Bind().Body(&req); err != nil {
		return apperr.New(apperr.INVALID_REQUEST, "invalid request body")
	}

	// Basic email validation
	if req.Email == "" || !contains(req.Email, "@") {
		return apperr.New(apperr.INVALID_EMAIL, "invalid email address")
	}

	// Call service
	if err := h.svc.RequestMagicLink(c.Context(), req.Email); err != nil {
		// Don't reveal if email exists (generic success per spec)
		// Log error internally
		logger.Error(c.Context(), err, "RequestMagicLink failed")
		return c.Status(fiber.StatusOK).JSON(RequestMagicLinkResponse{Success: true})
	}

	// Generic success - don't reveal account existence
	return c.Status(fiber.StatusOK).JSON(RequestMagicLinkResponse{Success: true})
}

// AuthenticateToken handles POST /api/v1/auth/authenticate-token
// @Summary Authenticate Stytch Discovery magic link token
// @Tags Authentication
// @Accept json
// @Produce json
// @Param request body AuthenticateTokenRequest true "request"
// @Success 200 {object} AuthenticateTokenResponse
// @Failure 400 {object} map[string]interface{}
// @Failure 401 {object} map[string]interface{}
// @Router /api/v1/auth/authenticate-token [post]
func (h *Handler) AuthenticateToken(c fiber.Ctx) error {
	var req AuthenticateTokenRequest
	if err := c.Bind().Body(&req); err != nil {
		return apperr.New(apperr.INVALID_REQUEST, "invalid request body")
	}

	// Call service
	ticketToken, onboardingState, sessionToken, err := h.svc.AuthenticateToken(c.Context(), req.Token)
	if err != nil {
		if errors.Is(err, service.ErrInvalidToken) || errors.Is(err, service.ErrRateLimited) {
			return apperr.New(apperr.INVALID_TOKEN, "invalid or expired magic link")
		}
		logger.Error(c.Context(), err, "AuthenticateToken failed")
		return apperr.New(apperr.INTERNAL_ERROR, "authentication failed")
	}

	// If session was created for returning user, set session cookie
	if sessionToken != nil {
		// Session cookie expires in 30 days (matches service creation)
		expiresAt := time.Now().Add(30 * 24 * time.Hour)
		h.setSessionCookie(c, *sessionToken, expiresAt)
		// Clear onboarding ticket cookie if set
		onboardingCookieName := "__Host-onboarding"
		onboardingSecure := h.cfg.Env == "production" && c.Scheme() == "https"
		if h.cfg.Env != "production" {
			onboardingCookieName = "onboarding"
			onboardingSecure = false
		}
		c.Cookie(&fiber.Cookie{
			Name:     onboardingCookieName,
			Value:    "",
			Path:     "/",
			HTTPOnly: true,
			Secure:   onboardingSecure,
			SameSite: fiber.CookieSameSiteLaxMode,
			MaxAge:   -1,
		})
	} else {
		// Set onboarding ticket cookie (host-only, short-lived) per AC-3/AC-9
		onboardingCookieName := "__Host-onboarding"
		onboardingSecure := h.cfg.Env == "production" && c.Scheme() == "https"
		if h.cfg.Env != "production" {
			onboardingCookieName = "onboarding"
			onboardingSecure = false
		}
		c.Cookie(&fiber.Cookie{
			Name:     onboardingCookieName,
			Value:    ticketToken,
			Path:     "/",
			HTTPOnly: true,
			Secure:   onboardingSecure,
			SameSite: fiber.CookieSameSiteLaxMode,
			MaxAge:   int(10 * time.Minute / time.Second),
		})
	}

	// Build response
	resp := AuthenticateTokenResponse{
		TicketToken: ticketToken,
	}

	if onboardingState != nil {
		resp.Status = onboardingState.CurrentStep
		resp.AuthState = onboardingState.AuthState
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
	} else {
		resp.Status = ""
		resp.AuthState = ""
	}

	return c.Status(fiber.StatusOK).JSON(resp)
}

// Logout handles POST /api/v1/auth/logout
func (h *Handler) Logout(c fiber.Ctx) error {
	// Read session cookie directly (not protected by middleware)
	sessionToken := c.Cookies(h.sessionCookieName())
	if sessionToken == "" {
		// Clear cookie regardless and return success
		h.clearSessionCookie(c)
		if h.auditRecorder != nil {
			h.auditRecorder.Record(c.Context(), audit.Event{
				Type:      audit.EventLogout,
				RequestID: c.Locals("request_id").(string),
				Outcome:   "success",
				IP:        c.IP(),
				Details:   map[string]string{"reason": "no_session_cookie"},
			})
		}
		return c.Status(fiber.StatusOK).JSON(LogoutResponse{Success: true})
	}

	tokenHash := sha256.Sum256([]byte(sessionToken))
	sessionCtx, err := h.svc.ValidateSession(c.Context(), tokenHash[:])
	if err != nil {
		if errors.Is(err, service.ErrSessionNotFound) {
			// Session not found; clear cookie anyway
			h.clearSessionCookie(c)
			if h.auditRecorder != nil {
				h.auditRecorder.Record(c.Context(), audit.Event{
					Type:      audit.EventLogout,
					RequestID: c.Locals("request_id").(string),
					Outcome:   "success",
					IP:        c.IP(),
					Details:   map[string]string{"reason": "session_not_found"},
				})
			}
			return c.Status(fiber.StatusOK).JSON(LogoutResponse{Success: true})
		}
		logger.Error(c.Context(), err, "session validation failed on logout")
	}

	var sessionID uuid.UUID
	if sessionCtx != nil {
		sessionID = sessionCtx.SessionID
	}

	// Call service to revoke session
	if sessionID != (uuid.UUID{}) {
		if err := h.svc.Logout(c.Context(), sessionID); err != nil {
			logger.Error(c.Context(), err, "Logout failed")
		}
	}

	// Clear session cookie
	h.clearSessionCookie(c)

	if h.auditRecorder != nil && sessionCtx != nil {
		h.auditRecorder.Record(c.Context(), audit.Event{
			Type:      audit.EventLogout,
			RequestID: c.Locals("request_id").(string),
			UserID:    sessionCtx.UserID.String(),
			TenantID:  sessionCtx.TenantID.String(),
			Outcome:   "success",
			IP:        c.IP(),
		})
	}

	return c.Status(fiber.StatusOK).JSON(LogoutResponse{Success: true})
}

// GetOnboardingCurrent handles GET /api/v1/onboarding/current
// Returns ticket-based onboarding state while pending,
// falling back to session-based final state after provisioning
// completes (ticket cookie is cleared, see ProvisionTenant).
func (h *Handler) GetOnboardingCurrent(c fiber.Ctx) error {
	onboardingCookieName := "__Host-onboarding"
	if h.cfg.Env != "production" {
		onboardingCookieName = "onboarding"
	}
	ticketToken := c.Cookies(onboardingCookieName)

	if ticketToken != "" {
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

	// Fall back to session-based state (after provisioning, ticket cookie is cleared)
	sessionToken := c.Cookies(h.sessionCookieName())
	if sessionToken == "" {
		return apperr.New(apperr.UNAUTHENTICATED, "no session or ticket cookie")
	}

	tokenHash := sha256.Sum256([]byte(sessionToken))
	sessionCtx, err := h.svc.ValidateSession(c.Context(), tokenHash[:])
	if err != nil {
		if errors.Is(err, service.ErrSessionNotFound) {
			return apperr.New(apperr.UNAUTHENTICATED, "session not found or expired")
		}
		logger.Error(c.Context(), err, "session validation failed")
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"code":    "INTERNAL_ERROR",
			"message": "session validation failed",
		})
	}

	// Build final onboarding state from session user/tenant
	resp := GetOnboardingCurrentResponse{
		VerifiedEmail: "",
		CurrentStep:   "complete",
		FormData:      map[string]interface{}{},
	}
	id := sessionCtx.TenantID.String()
	resp.TenantID = &id
	return c.Status(fiber.StatusOK).JSON(resp)
}

// UpdateOnboardingCurrent handles PUT /api/v1/onboarding/current
func (h *Handler) UpdateOnboardingCurrent(c fiber.Ctx) error {
	onboardingCookieName := "__Host-onboarding"
	if h.cfg.Env != "production" {
		onboardingCookieName = "onboarding"
	}
	ticketToken := c.Cookies(onboardingCookieName)
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
}

// sessionCookieName returns the cookie name with environment suffix per ADR-0002
// Uses config.SessionCookieName() which returns __Host-session_token in production
// and session_token_<env> in development/test.
func (h *Handler) sessionCookieName() string {
	return h.cfg.SessionCookieName()
}

// sessionCookieDomain returns the cookie domain from config
// For __Host- prefix cookies, this must be empty (browser ignores Domain)
func (h *Handler) sessionCookieDomain() string {
	return h.cfg.SessionCookieDomain()
}

// setSessionCookie sets the session cookie with proper attributes
func (h *Handler) setSessionCookie(c fiber.Ctx, token string, expiresAt time.Time) {
	cookieName := h.sessionCookieName()
	cookieDomain := h.sessionCookieDomain()

	maxAge := int(time.Until(expiresAt).Seconds())
	if maxAge < 0 {
		maxAge = 0
	}

	// Session cookie always uses __Secure- prefix, so Secure must be true for https
	// In development (http), the cookie name won't have __Secure- prefix
	secure := c.Scheme() == "https"
	c.Cookie(&fiber.Cookie{
		Name:     cookieName,
		Value:    token,
		Path:     "/",
		Domain:   cookieDomain,
		HTTPOnly: true,
		Secure:   secure,
		SameSite: fiber.CookieSameSiteLaxMode,
		MaxAge:   maxAge,
	})
}

// clearSessionCookie clears the session cookie
func (h *Handler) clearSessionCookie(c fiber.Ctx) {
	cookieName := h.sessionCookieName()
	cookieDomain := h.sessionCookieDomain()
	secure := c.Scheme() == "https"
	c.Cookie(&fiber.Cookie{
		Name:     cookieName,
		Value:    "",
		Path:     "/",
		Domain:   cookieDomain,
		HTTPOnly: true,
		Secure:   secure,
		SameSite: fiber.CookieSameSiteLaxMode,
		MaxAge:   -1,
	})
}

// contains checks if a string contains a substring
func contains(s, substr string) bool {
	return strings.Contains(s, substr)
}
