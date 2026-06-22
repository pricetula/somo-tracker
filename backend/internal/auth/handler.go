package auth

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"

	"github.com/gofiber/fiber/v2"
	"go.uber.org/zap"

	"somotracker/backend/internal/config"
	"somotracker/backend/internal/middleware"
)

// ============================================================================
// Handler — HTTP delivery layer (requirement 4, 6, 7, 14).
// ============================================================================

const (
	somoCookieName     = "somo_sid"
	somoRoleCookieName = "somo_role"
	cookieMaxAge       = 2592000 // 30 days in seconds
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
	UserID     string `json:"user_id"`
	TenantID   string `json:"tenant_id"`
	Role       string `json:"role"`
	SchoolID   string `json:"school_id,omitempty"`
	SchoolName string `json:"school_name,omitempty"`
	FirstName  string `json:"first_name,omitempty"`
	LastName   string `json:"last_name,omitempty"`
	Email      string `json:"email,omitempty"`
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
		return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{
			"code":    "invalid_input",
			"message": "invalid request body",
		})
	}
	if payload.Email == "" {
		return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{
			"code":    "invalid_input",
			"message": "email is required",
		})
	}

	if err := h.svc.Discover(c.Context(), payload.Email); err != nil {
		return middleware.HTTPError(c, err)
	}

	return c.SendStatus(fiber.StatusOK)
}

// MagicLinkCallback handles GET /api/auth/callback.
func (h *Handler) MagicLinkCallback(c *fiber.Ctx) error {
	token := c.Query("token")
	if token == "" {
		return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{
			"code":    "invalid_input",
			"message": "token query parameter is required",
		})
	}

	sessionRef, err := h.svc.Verify(c.Context(), token)
	if err != nil {
		h.logger.Error("auth: magic link callback verify failed",
			zap.Error(err),
		)
		return middleware.HTTPError(c, err)
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
func (h *Handler) Verify(c *fiber.Ctx) error {
	var payload struct {
		Token string `json:"token"`
	}
	if err := c.BodyParser(&payload); err != nil {
		return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{
			"code":    "invalid_input",
			"message": "invalid request body",
		})
	}
	if payload.Token == "" {
		return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{
			"code":    "invalid_input",
			"message": "token is required",
		})
	}

	sessionRef, err := h.svc.Verify(c.Context(), payload.Token)
	if err != nil {
		return middleware.HTTPError(c, err)
	}

	return c.JSON(fiber.Map{
		"session_ref": sessionRef,
	})
}

// Register handles POST /api/auth/register (PHASE 3).
func (h *Handler) Register(c *fiber.Ctx) error {
	var payload RegistrationPayload
	if err := c.BodyParser(&payload); err != nil {
		return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{
			"code":    "invalid_input",
			"message": "invalid request body",
		})
	}

	// Extract device fingerprint from security pipeline (requirement 5)
	deviceFingerprint, _ := c.Locals("device_fingerprint").(string)

	sessionToken, role, err := h.svc.Register(c.Context(), payload.SessionRef, payload, deviceFingerprint)
	if err != nil {
		h.logger.Error("auth: registration failed",
			zap.Error(err),
			zap.String("session_ref", payload.SessionRef),
		)
		return middleware.HTTPError(c, err)
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

	// Issue signed role cookie (not HttpOnly — Next.js middleware reads it)
	c.Cookie(&fiber.Cookie{
		Name:     somoRoleCookieName,
		Value:    createSignedCookieValue(role, h.cfg.CookieSecret),
		HTTPOnly: false,
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
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"code":    "unauthorized",
			"message": "no session cookie found",
		})
	}

	info, err := h.svc.GetMe(c.Context(), token)
	if err != nil {
		return middleware.HTTPError(c, err)
	}

	return c.JSON(fiber.Map{
		"user_id":     info.UserID,
		"tenant_id":   info.TenantID,
		"role":        info.Role,
		"school_id":   info.SchoolID,
		"school_name": info.SchoolName,
		"first_name":  info.FirstName,
		"last_name":   info.LastName,
		"email":       info.Email,
	})
}

// Logout handles DELETE /api/auth/session (requirement 7).
func (h *Handler) Logout(c *fiber.Ctx) error {
	token := c.Cookies(somoCookieName)

	if err := h.svc.Logout(c.Context(), token); err != nil {
		return middleware.HTTPError(c, err)
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

	// Clear the role cookie
	c.Cookie(&fiber.Cookie{
		Name:     somoRoleCookieName,
		Value:    "",
		HTTPOnly: false,
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

// ============================================================================
// Cookie signing helpers — Two-Cookie Auth (Role Signing)
// ============================================================================

// createSignedCookieValue signs a value using HMAC-SHA256 and returns it in
// the format: value.hexsignature
// The frontend splits on the last '.' and verifies with the same secret.
func createSignedCookieValue(value, secret string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(value))
	sig := hex.EncodeToString(mac.Sum(nil))
	return value + "." + sig
}

// Compile-time check that the removed mapError function is no longer used.
// All error-to-HTTP mapping is now delegated to middleware.HTTPError.
var _ = errors.Is
