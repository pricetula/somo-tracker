package auth

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"

	"github.com/gofiber/fiber/v2"
	"go.uber.org/zap"

	"somotracker/backend/internal/config"
)

// ============================================================================
// Handler — HTTP delivery layer (requirement 4, 6, 7, 14).
// ============================================================================

const (
	somoCookieName = "somo_sid"
	cookieMaxAge   = 2592000 // 30 days in seconds
)

// ErrorBody is the JSON response body for error responses (requirement 14).
type ErrorBody struct {
	Error   string `json:"error"`
	Message string `json:"message,omitempty"`
}

// VerifyPayload is the request body for POST /api/auth/verify.
type VerifyPayload struct {
	Token string `json:"token"`
}

// VerifyResponse is the response body for POST /api/auth/verify.
type VerifyResponse struct {
	SessionRef string `json:"session_ref"`
}

// MeResponse is the response body for GET /api/auth/me.
type MeResponse struct {
	UserID   string `json:"user_id"`
	TenantID string `json:"tenant_id"`
}

// Handler exposes auth HTTP endpoints.
type Handler struct {
	svc    *Service
	logger *zap.Logger
	cfg    config.Config
}

// NewHandler creates a new Handler.
func NewHandler(svc *Service, logger *zap.Logger, cfg config.Config) *Handler {
	return &Handler{
		svc:    svc,
		logger: logger,
		cfg:    cfg,
	}
}

// RegisterRoutes mounts auth routes on the given router.
func (h *Handler) RegisterRoutes(router fiber.Router) {
	auth := router.Group("/api/auth")

	auth.Post("/discover", h.Discover)
	auth.Post("/verify", h.Verify)
	auth.Post("/register", h.Register)
	auth.Get("/callback", h.MagicLinkCallback)
	auth.Get("/me", h.Me)
	auth.Delete("/session", h.Logout)
}

// Discover handles POST /api/auth/discover (PHASE 1).
//
// @Summary      Initiate magic-link discovery
// @Description  Sends a magic-link email to the given address to discover or create an organization.
// @Tags         Auth
// @Accept       json
// @Produce      json
// @Param        body  body  DiscoveryPayload  true  "Email address to send magic link to"
// @Success      200   "Magic link sent"
// @Failure      422   {object}  ErrorBody  "Invalid input"
// @Failure      500   {object}  ErrorBody  "Internal error"
// @Router       /api/auth/discover [post]
func (h *Handler) Discover(c *fiber.Ctx) error {
	var payload DiscoveryPayload
	if err := c.BodyParser(&payload); err != nil {
		return c.Status(fiber.StatusUnprocessableEntity).JSON(ErrorBody{
			Error:   "invalid_input",
			Message: "invalid request body",
		})
	}
	if payload.Email == "" {
		return c.Status(fiber.StatusUnprocessableEntity).JSON(ErrorBody{
			Error:   "invalid_input",
			Message: "email is required",
		})
	}

	if err := h.svc.Discover(c.Context(), payload.Email); err != nil {
		status, body := h.mapError(err)
		return c.Status(status).JSON(body)
	}

	return c.SendStatus(fiber.StatusOK)
}

// MagicLinkCallback handles GET /api/auth/callback.
// Stytch redirects the user's browser here after clicking a magic link.
// The URL includes ?token=...&stytch_token_type=discovery.
// We verify the token, cache the IST in Redis, set CSRF cookie,
// and redirect the browser to the frontend's /register page with the session_ref.
//
// @Summary      Magic-link callback
// @Description  Stytch redirects users here after clicking a magic link. Verifies the token, caches the IST, and redirects to the frontend.
// @Tags         Auth
// @Accept       json
// @Produce      json
// @Param        token  query  string  true  "Stytch discovery magic link token"
// @Success      302    "Redirects to frontend /register with session_ref"
// @Failure      422    {object}  ErrorBody  "Invalid input"
// @Failure      500    {object}  ErrorBody  "Internal error"
// @Router       /api/auth/callback [get]
func (h *Handler) MagicLinkCallback(c *fiber.Ctx) error {
	token := c.Query("token")
	if token == "" {
		return c.Status(fiber.StatusUnprocessableEntity).JSON(ErrorBody{
			Error:   "invalid_input",
			Message: "token query parameter is required",
		})
	}

	sessionRef, err := h.svc.Verify(c.Context(), token)
	if err != nil {
		h.logger.Error("auth: magic link callback verify failed",
			zap.Error(err),
		)
		status, body := h.mapError(err)
		return c.Status(status).JSON(body)
	}

	// Set a CSRF token cookie so the frontend can include it on mutating requests
	csrfToken, err := generateCSRFToken()
	if err != nil {
		h.logger.Error("auth: failed to generate CSRF token", zap.Error(err))
	} else {
		h.setCSRFTokenCookie(c, csrfToken)
	}

	// Redirect browser to frontend registration page with session_ref
	frontendRegisterURL := h.cfg.FrontendURL + "/register"
	redirectURL := fmt.Sprintf("%s?session_ref=%s", frontendRegisterURL, sessionRef)

	h.logger.Info("auth: magic link callback — redirecting to frontend",
		zap.String("session_ref", sessionRef),
		zap.String("redirect_url", frontendRegisterURL),
	)

	return c.Redirect(redirectURL, fiber.StatusFound)
}

// Verify handles POST /api/auth/verify (PHASE 2).
//
// @Summary      Verify magic-link token
// @Description  Validates a magic-link discovery token and returns a session reference for the registration flow.
// @Tags         Auth
// @Accept       json
// @Produce      json
// @Param        body  body      VerifyPayload  true  "Magic link token"
// @Success      200   {object}  VerifyResponse
// @Failure      422   {object}  ErrorBody  "Invalid input"
// @Failure      401   {object}  ErrorBody  "Token expired"
// @Failure      500   {object}  ErrorBody  "Internal error"
// @Router       /api/auth/verify [post]
func (h *Handler) Verify(c *fiber.Ctx) error {
	var payload struct {
		Token string `json:"token"`
	}
	if err := c.BodyParser(&payload); err != nil {
		return c.Status(fiber.StatusUnprocessableEntity).JSON(ErrorBody{
			Error:   "invalid_input",
			Message: "invalid request body",
		})
	}
	if payload.Token == "" {
		return c.Status(fiber.StatusUnprocessableEntity).JSON(ErrorBody{
			Error:   "invalid_input",
			Message: "token is required",
		})
	}

	sessionRef, err := h.svc.Verify(c.Context(), payload.Token)
	if err != nil {
		status, body := h.mapError(err)
		return c.Status(status).JSON(body)
	}

	return c.JSON(fiber.Map{
		"session_ref": sessionRef,
	})
}

// Register handles POST /api/auth/register (PHASE 3).
//
// @Summary      Complete registration
// @Description  Creates a tenant (school), user, and session. Sets the session cookie on success.
// @Tags         Auth
// @Accept       json
// @Produce      json
// @Param        body  body  RegistrationPayload  true  "Registration details"
// @Success      204   "Session cookie set; no content"
// @Failure      422   {object}  ErrorBody  "Invalid input"
// @Failure      401   {object}  ErrorBody  "Token expired or MFA required"
// @Failure      409   {object}  ErrorBody  "Organization already exists"
// @Failure      500   {object}  ErrorBody  "Internal error"
// @Router       /api/auth/register [post]
func (h *Handler) Register(c *fiber.Ctx) error {
	var payload RegistrationPayload
	if err := c.BodyParser(&payload); err != nil {
		return c.Status(fiber.StatusUnprocessableEntity).JSON(ErrorBody{
			Error:   "invalid_input",
			Message: "invalid request body",
		})
	}

	// Extract device fingerprint from security pipeline (requirement 5)
	deviceFingerprint, _ := c.Locals("device_fingerprint").(string)

	sessionToken, err := h.svc.Register(c.Context(), payload.SessionRef, payload, deviceFingerprint)
	if err != nil {
		status, body := h.mapError(err)
		return c.Status(status).JSON(body)
	}

	// Issue HTTPOnly session cookie (requirement 4)
	c.Cookie(&fiber.Cookie{
		Name:     somoCookieName,
		Value:    sessionToken,
		HTTPOnly: true,
		Secure:   h.cfg.AppEnv != "development",
		SameSite: "Lax",
		Path:     "/",
		Domain:   h.cfg.CookieDomain,
		MaxAge:   cookieMaxAge,
	})

	// Issue non-HttpOnly CSRF token cookie so the frontend JS can read it
	csrfToken, err := generateCSRFToken()
	if err != nil {
		h.logger.Error("auth: failed to generate CSRF token", zap.Error(err))
	} else {
		h.setCSRFTokenCookie(c, csrfToken)
	}

	return c.SendStatus(fiber.StatusNoContent)
}

// Me handles GET /api/auth/me (requirement 6).
//
// @Summary      Get current session
// @Description  Returns the authenticated user's ID and tenant ID from the session cookie.
// @Tags         Auth
// @Accept       json
// @Produce      json
// @Success      200  {object}  MeResponse
// @Failure      401  {object}  ErrorBody  "Session expired or missing"
// @Failure      500  {object}  ErrorBody  "Internal error"
// @Router       /api/auth/me [get]
func (h *Handler) Me(c *fiber.Ctx) error {
	token := c.Cookies(somoCookieName)
	if token == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(ErrorBody{
			Error:   "expired_token",
			Message: "no session cookie found",
		})
	}

	session, err := h.svc.GetSession(c.Context(), token)
	if err != nil {
		status, body := h.mapError(err)
		return c.Status(status).JSON(body)
	}

	return c.JSON(fiber.Map{
		"user_id":   session.UserID,
		"tenant_id": session.TenantID,
	})
}

// Logout handles DELETE /api/auth/session (requirement 7).
//
// @Summary      Logout
// @Description  Destroys the current session and clears cookies.
// @Tags         Auth
// @Accept       json
// @Produce      json
// @Success      204  "Session destroyed; cookies cleared"
// @Failure      500  {object}  ErrorBody  "Internal error"
// @Router       /api/auth/session [delete]
func (h *Handler) Logout(c *fiber.Ctx) error {
	token := c.Cookies(somoCookieName)

	if err := h.svc.Logout(c.Context(), token); err != nil {
		status, body := h.mapError(err)
		return c.Status(status).JSON(body)
	}

	// Clear the session cookie
	c.Cookie(&fiber.Cookie{
		Name:     somoCookieName,
		Value:    "",
		HTTPOnly: true,
		Secure:   h.cfg.AppEnv != "development",
		SameSite: "Lax",
		Path:     "/",
		Domain:   h.cfg.CookieDomain,
		MaxAge:   -1,
	})

	// Clear the CSRF token cookie
	c.Cookie(&fiber.Cookie{
		Name:     "csrf_token",
		Value:    "",
		HTTPOnly: false,
		Secure:   h.cfg.AppEnv != "development",
		SameSite: "Lax",
		Path:     "/",
		Domain:   h.cfg.CookieDomain,
		MaxAge:   -1,
	})

	return c.SendStatus(fiber.StatusNoContent)
}

// mapError maps domain errors to HTTP status codes and JSON bodies (requirement 14).
func (h *Handler) mapError(err error) (int, ErrorBody) {
	var validationErr *ValidationError
	if errors.As(err, &validationErr) {
		return fiber.StatusUnprocessableEntity, ErrorBody{
			Error:   "invalid_input",
			Message: validationErr.Message,
		}
	}

	switch {
	case errors.Is(err, ErrInvalidInput):
		return fiber.StatusUnprocessableEntity, ErrorBody{
			Error:   "invalid_input",
			Message: err.Error(),
		}
	case errors.Is(err, ErrExpiredToken):
		return fiber.StatusUnauthorized, ErrorBody{
			Error:   "expired_token",
			Message: "session expired or invalid",
		}
	case errors.Is(err, ErrMFARequired):
		return fiber.StatusUnauthorized, ErrorBody{
			Error:   "mfa_required",
			Message: "multi-factor authentication challenge required",
		}
	case errors.Is(err, ErrOrgAlreadyExists):
		return fiber.StatusConflict, ErrorBody{
			Error:   "org_already_exists",
			Message: "organization already exists",
		}
	case errors.Is(err, ErrNotFound):
		return fiber.StatusNotFound, ErrorBody{
			Error:   "not_found",
			Message: "resource not found",
		}
	default:
		return fiber.StatusInternalServerError, ErrorBody{
			Error:   "internal_error",
			Message: "an unexpected error occurred",
		}
	}
}

// ============================================================================
// CSRF helpers — Double-Submit Cookie pattern
// ============================================================================

// setCSRFTokenCookie sets a non-HttpOnly csrf_token cookie so the frontend JS
// can read it and include it as an X-CSRF-Token header on mutating requests.
func (h *Handler) setCSRFTokenCookie(c *fiber.Ctx, token string) {
	c.Cookie(&fiber.Cookie{
		Name:     "csrf_token",
		Value:    token,
		HTTPOnly: false,
		Secure:   h.cfg.AppEnv != "development",
		SameSite: "Lax",
		Path:     "/",
		Domain:   h.cfg.CookieDomain,
		MaxAge:   cookieMaxAge,
	})
}

// generateCSRFToken returns a cryptographically random token suitable for use
// as a CSRF token. It returns a URL-safe base64-encoded string.
func generateCSRFToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}
