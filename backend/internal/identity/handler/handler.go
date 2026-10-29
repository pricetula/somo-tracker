package handler

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"somotracker/internal/identity/service"
)

// Handler is HTTP handler for identity domain (thin, BE-006).
type Handler struct {
	svc *service.Service
}

// New creates handler.
func New(svc *service.Service) *Handler {
	return &Handler{svc: svc}
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
	Status        string            `json:"status"`
	TicketToken   string            `json:"ticket_token,omitempty"`
	Onboarding    *OnboardingState  `json:"onboarding,omitempty"`
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

type ErrorResponse struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// RequestMagicLink handles POST /api/v1/auth/request-magic-link
func (h *Handler) RequestMagicLink(c fiber.Ctx) error {
	var req RequestMagicLinkRequest
	if err := c.Bind().Body(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(ErrorResponse{
			Code:    "INVALID_REQUEST",
			Message: "invalid request body",
		})
	}

	// Basic email validation
	if req.Email == "" || !strings.Contains(req.Email, "@") {
		return c.Status(fiber.StatusBadRequest).JSON(ErrorResponse{
			Code:    "INVALID_EMAIL",
			Message: "invalid email address",
		})
	}

	// Call service - returns PKCE verifier to store in cookie
	pkceVerifier, err := h.svc.RequestMagicLink(c.Context(), req.Email)
	if err != nil {
		// Don't reveal if email exists (generic success per spec)
		// Log error internally
		fmt.Printf("RequestMagicLink error: %v\n", err)
		return c.Status(fiber.StatusOK).JSON(RequestMagicLinkResponse{Success: true})
	}

	// Store PKCE verifier in HttpOnly cookie for callback
	c.Cookie(&fiber.Cookie{
		Name:     "__Host-pkce_verifier",
		Value:    pkceVerifier,
		Path:     "/",
		HTTPOnly: true,
		Secure:   true,
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
		return c.Status(fiber.StatusBadRequest).JSON(ErrorResponse{
			Code:    "INVALID_REQUEST",
			Message: "invalid request body",
		})
	}

	// Get PKCE verifier from cookie
	pkceVerifier := c.Cookies("__Host-pkce_verifier")
	if pkceVerifier == "" {
		return c.Status(fiber.StatusBadRequest).JSON(ErrorResponse{
			Code:    "MISSING_PKCE",
			Message: "PKCE verifier missing, please restart login flow",
		})
	}

	// Call service
	ticketToken, onboardingState, err := h.svc.AuthenticateToken(c.Context(), req.Token, pkceVerifier)
	if err != nil {
		if errors.Is(err, service.ErrInvalidToken) || errors.Is(err, service.ErrRateLimited) {
			return c.Status(fiber.StatusUnauthorized).JSON(ErrorResponse{
				Code:    "INVALID_TOKEN",
				Message: "invalid or expired magic link",
			})
		}
		fmt.Printf("AuthenticateToken error: %v\n", err)
		return c.Status(fiber.StatusInternalServerError).JSON(ErrorResponse{
			Code:    "INTERNAL_ERROR",
			Message: "authentication failed",
		})
	}

	// Clear PKCE cookie
	c.Cookie(&fiber.Cookie{
		Name:     "__Host-pkce_verifier",
		Value:    "",
		Path:     "/",
		HTTPOnly: true,
		Secure:   true,
		SameSite: "Lax",
		MaxAge:   -1,
	})

	// Set onboarding ticket cookie (host-only, short-lived) per AC-3/AC-9
	c.Cookie(&fiber.Cookie{
		Name:     "__Host-onboarding",
		Value:    ticketToken,
		Path:     "/",
		HTTPOnly: true,
		Secure:   true,
		SameSite: "Lax",
		MaxAge:   int(10 * time.Minute / time.Second),
	})

	// Build response
	resp := AuthenticateTokenResponse{
		Status:      onboardingState.CurrentStep,
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
		return c.Status(fiber.StatusBadRequest).JSON(ErrorResponse{
			Code:    "MISSING_TICKET",
			Message: "onboarding ticket missing, please restart login flow",
		})
	}

	var req ProvisionTenantRequest
	if err := c.Bind().Body(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(ErrorResponse{
			Code:    "INVALID_REQUEST",
			Message: "invalid request body",
		})
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
			return c.Status(fiber.StatusBadRequest).JSON(ErrorResponse{
				Code:    "INVALID_TICKET",
				Message: "invalid or expired onboarding ticket",
			})
		}
		if errors.Is(err, service.ErrIdempotencyConflict) {
			return c.Status(fiber.StatusConflict).JSON(ErrorResponse{
				Code:    "IDEMPOTENCY_CONFLICT",
				Message: "onboarding already completed with this key",
			})
		}
		fmt.Printf("ProvisionTenant error: %v\n", err)
		return c.Status(fiber.StatusInternalServerError).JSON(ErrorResponse{
			Code:    "PROVISIONING_FAILED",
			Message: "tenant provisioning failed",
		})
	}

	// Clear onboarding ticket cookie
	c.Cookie(&fiber.Cookie{
		Name:     "__Host-onboarding",
		Value:    "",
		Path:     "/",
		HTTPOnly: true,
		Secure:   true,
		SameSite: "Lax",
		MaxAge:   -1,
	})

	// Create session after provisioning
	// Parse expires from Stytch (for now use 30 days)
	expiresAt := time.Now().Add(30 * 24 * time.Hour)
	sessionToken, err := h.svc.CreateSession(c.Context(), *result.UserID, *result.TenantID, "", expiresAt)
	if err != nil {
		fmt.Printf("CreateSession error: %v\n", err)
		return c.Status(fiber.StatusInternalServerError).JSON(ErrorResponse{
			Code:    "SESSION_CREATE_FAILED",
			Message: "failed to create session",
		})
	}

	// Set session cookie
	setSessionCookie(c, sessionToken, expiresAt)

	return c.Status(fiber.StatusOK).JSON(ProvisionTenantResponse{Success: true})
}

// Logout handles POST /api/v1/auth/logout
func (h *Handler) Logout(c fiber.Ctx) error {
	// Get session from context (set by auth middleware)
	sessionID, ok := c.Locals("session_id").(uuid.UUID)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(ErrorResponse{
			Code:    "UNAUTHENTICATED",
			Message: "no active session",
		})
	}

	// Call service to revoke session
	if err := h.svc.Logout(c.Context(), sessionID); err != nil {
		fmt.Printf("Logout error: %v\n", err)
	}

	// Clear session cookie
	clearSessionCookie(c)

	return c.Status(fiber.StatusOK).JSON(LogoutResponse{Success: true})
}

// GetOnboardingCurrent handles GET /api/v1/onboarding/current
func (h *Handler) GetOnboardingCurrent(c fiber.Ctx) error {
	ticketToken := c.Cookies("__Host-onboarding")
	if ticketToken == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(ErrorResponse{
			Code:    "UNAUTHENTICATED",
			Message: "no onboarding session",
		})
	}

	state, err := h.svc.GetOnboardingCurrent(c.Context(), ticketToken)
	if err != nil {
		if errors.Is(err, service.ErrInvalidToken) || errors.Is(err, service.ErrPendingOnboardingNotFound) {
			return c.Status(fiber.StatusUnauthorized).JSON(ErrorResponse{
				Code:    "INVALID_TICKET",
				Message: "invalid or expired onboarding ticket",
			})
		}
		fmt.Printf("GetOnboardingCurrent error: %v\n", err)
		return c.Status(fiber.StatusInternalServerError).JSON(ErrorResponse{
			Code:    "INTERNAL_ERROR",
			Message: "failed to load onboarding state",
		})
	}

	resp := fiber.Map{
		"verified_email": state.VerifiedEmail,
		"current_step":   state.CurrentStep,
		"form_data":      state.FormData,
		"pending_id":     state.PendingID.String(),
	}
	if state.SchoolName != nil {
		resp["school_name"] = *state.SchoolName
	}
	if state.TenantID != nil {
		resp["tenant_id"] = state.TenantID.String()
	}
	return c.Status(fiber.StatusOK).JSON(resp)
}

// UpdateOnboardingCurrent handles PUT /api/v1/onboarding/current
func (h *Handler) UpdateOnboardingCurrent(c fiber.Ctx) error {
	ticketToken := c.Cookies("__Host-onboarding")
	if ticketToken == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(ErrorResponse{
			Code:    "UNAUTHENTICATED",
			Message: "no onboarding session",
		})
	}

	var req struct {
		CurrentStep string                 `json:"current_step" validate:"required"`
		FormData    map[string]interface{} `json:"form_data"`
		SchoolName  string                 `json:"school_name,omitempty"`
	}
	if err := c.Bind().Body(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(ErrorResponse{
			Code:    "INVALID_REQUEST",
			Message: "invalid request body",
		})
	}

	state, err := h.svc.GetOnboardingCurrent(c.Context(), ticketToken)
	if err != nil {
		if errors.Is(err, service.ErrInvalidToken) || errors.Is(err, service.ErrPendingOnboardingNotFound) {
			return c.Status(fiber.StatusUnauthorized).JSON(ErrorResponse{
				Code:    "INVALID_TICKET",
				Message: "invalid or expired onboarding ticket",
			})
		}
		fmt.Printf("UpdateOnboardingCurrent error: %v\n", err)
		return c.Status(fiber.StatusInternalServerError).JSON(ErrorResponse{
			Code:    "INTERNAL_ERROR",
			Message: "failed to load onboarding state",
		})
	}

	if req.SchoolName != "" {
		state.SchoolName = &req.SchoolName
	}

	if err := h.svc.UpdateOnboardingStep(c.Context(), state.PendingID, req.CurrentStep, req.FormData); err != nil {
		fmt.Printf("UpdateOnboardingStep error: %v\n", err)
		return c.Status(fiber.StatusInternalServerError).JSON(ErrorResponse{
			Code:    "INTERNAL_ERROR",
			Message: "failed to update onboarding step",
		})
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{"success": true})
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

// setSessionCookie sets the session cookie with proper attributes
func setSessionCookie(c fiber.Ctx, token string, expiresAt time.Time) {
	// Cookie name per ADR-0002: __Secure-session_token with env suffix
	cookieName := "__Secure-session_token"
	// TODO: Add env suffix (e.g., __Secure-session_token_stg)

	maxAge := int(time.Until(expiresAt).Seconds())
	if maxAge < 0 {
		maxAge = 0
	}

	c.Cookie(&fiber.Cookie{
		Name:     cookieName,
		Value:    token,
		Path:     "/",
		Domain:   "", // Will be set to shared domain in production
		HTTPOnly: true,
		Secure:   true,
		SameSite: "Lax",
		MaxAge:   maxAge,
	})
}

// clearSessionCookie clears the session cookie
func clearSessionCookie(c fiber.Ctx) {
	cookieName := "__Secure-session_token"
	c.Cookie(&fiber.Cookie{
		Name:     cookieName,
		Value:    "",
		Path:     "/",
		Domain:   "",
		HTTPOnly: true,
		Secure:   true,
		SameSite: "Lax",
		MaxAge:   -1,
	})
}