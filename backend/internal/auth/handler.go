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
	frontendURL := "http://localhost:3000/register"
	if h.cfg.AppEnv != "development" {
		frontendURL = "https://app.somotracker.com/register"
	}
	redirectURL := fmt.Sprintf("%s?session_ref=%s", frontendURL, sessionRef)

	h.logger.Info("auth: magic link callback — redirecting to frontend",
		zap.String("session_ref", sessionRef),
		zap.String("redirect_url", frontendURL),
	)

	return c.Redirect(redirectURL, fiber.StatusFound)
}

// Verify handles POST /api/auth/verify (PHASE 2).
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
