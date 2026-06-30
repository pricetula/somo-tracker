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
	somoCookieName         = "somo_sid"
	somoRoleCookieName     = "somo_role"
	somoSchoolIDCookieName = "somo_school_id"
	cookieMaxAge           = 2592000 // 30 days in seconds
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
	FullName   string `json:"full_name,omitempty"`
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
	auth.Get("/invite/callback", h.AcceptInvite)
	auth.Get("/me", h.Me)
	auth.Delete("/session", h.Logout)
}

// setSessionCookies sets all three session cookies (session ID, role, school ID).
func (h *Handler) setSessionCookies(c *fiber.Ctx, sessionToken, role, schoolID string) {
	// HttpOnly session cookie
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

	// Signed role cookie (not HttpOnly — frontend reads it for routing)
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

	// School ID cookie (not HttpOnly — frontend can read it for context)
	if schoolID != "" {
		c.Cookie(&fiber.Cookie{
			Name:     somoSchoolIDCookieName,
			Value:    schoolID,
			HTTPOnly: false,
			Secure:   h.cfg.AppEnv != "development",
			SameSite: "Lax",
			Path:     "/",
			Domain:   h.cfg.CookieDomain,
			MaxAge:   cookieMaxAge,
		})
	}
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

	// Extract device fingerprint from security pipeline
	deviceFingerprint, _ := c.Locals("device_fingerprint").(string)

	result, err := h.svc.Verify(c.Context(), token, deviceFingerprint)
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

	// Branch: existing user (has session_token) vs new user (has session_ref)
	if result.SessionToken != "" {
		// EXISTING USER PATH: set session cookies and redirect to dashboard
		h.setSessionCookies(c, result.SessionToken, result.Role, result.SchoolID)

		dashboardURL := h.cfg.FrontendURL + "/"

		h.logger.Info("auth: existing user — redirecting to dashboard",
			zap.String("role", result.Role),
			zap.String("school_id", result.SchoolID),
			zap.String("redirect_url", dashboardURL),
		)

		return c.Redirect(dashboardURL, fiber.StatusFound)
	}

	// NEW USER PATH: redirect to frontend registration page with session_ref
	frontendRegisterURL := h.cfg.FrontendURL + "/register"
	redirectURL := fmt.Sprintf("%s?session_ref=%s", frontendRegisterURL, result.SessionRef)

	h.logger.Info("auth: new user — redirecting to registration",
		zap.String("session_ref", result.SessionRef),
		zap.String("redirect_url", redirectURL),
	)

	return c.Redirect(redirectURL, fiber.StatusFound)
}

// AcceptInvite handles GET /api/auth/invite/callback.
// This endpoint is called directly by Stytch after the user clicks the magic
// link in the invite email. It authenticates the token, looks up the pending
// invitation, creates the user/session/membership, sets session cookies,
// and redirects the browser to the frontend dashboard.
func (h *Handler) AcceptInvite(c *fiber.Ctx) error {
	token := c.Query("token")
	if token == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"code":    "invalid_input",
			"message": "token query parameter is required",
		})
	}

	// Extract device fingerprint from security pipeline
	deviceFingerprint, _ := c.Locals("device_fingerprint").(string)

	sessionToken, role, schoolID, err := h.svc.AcceptInvite(c.Context(), token, deviceFingerprint)
	if err != nil {
		h.logger.Error("auth: accept invite failed",
			zap.Error(err),
		)
		return middleware.HTTPError(c, err)
	}

	// Set session, role, and school ID cookies
	h.setSessionCookies(c, sessionToken, role, schoolID)

	// Set CSRF token cookie (non-HttpOnly so frontend JS can read it)
	csrfToken, err := generateCSRFToken()
	if err != nil {
		h.logger.Error("auth: failed to generate CSRF token", zap.Error(err))
	} else {
		h.setCSRFTokenCookie(c, csrfToken)
	}

	// Redirect to frontend dashboard
	dashboardURL := h.cfg.FrontendURL + "/"

	h.logger.Info("auth: invite accepted — redirecting to dashboard",
		zap.String("role", role),
		zap.String("school_id", schoolID),
		zap.String("redirect_url", dashboardURL),
	)

	return c.Redirect(dashboardURL, fiber.StatusFound)
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

	deviceFingerprint, _ := c.Locals("device_fingerprint").(string)

	result, err := h.svc.Verify(c.Context(), payload.Token, deviceFingerprint)
	if err != nil {
		return middleware.HTTPError(c, err)
	}

	// For the POST /api/auth/verify endpoint (used by non-browser clients),
	// return session_ref for new users or session_token + role for existing users
	if result.SessionToken != "" {
		resp := fiber.Map{
			"session_token": result.SessionToken,
			"role":          result.Role,
			"email":         result.Email,
		}
		if result.SchoolID != "" {
			resp["school_id"] = result.SchoolID
		}
		return c.JSON(resp)
	}

	return c.JSON(fiber.Map{
		"session_ref": result.SessionRef,
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

	sessionToken, role, schoolID, err := h.svc.Register(c.Context(), payload.SessionRef, payload, deviceFingerprint)
	if err != nil {
		h.logger.Error("auth: registration failed",
			zap.Error(err),
			zap.String("session_ref", payload.SessionRef),
		)
		return middleware.HTTPError(c, err)
	}

	// Issue all session cookies (session ID, role, school ID)
	h.setSessionCookies(c, sessionToken, role, schoolID)

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

	// Also set the school ID cookie for existing sessions that don't have it yet
	if info.SchoolID != "" {
		c.Cookie(&fiber.Cookie{
			Name:     somoSchoolIDCookieName,
			Value:    info.SchoolID,
			HTTPOnly: false,
			Secure:   h.cfg.AppEnv != "development",
			SameSite: "Lax",
			Path:     "/",
			Domain:   h.cfg.CookieDomain,
			MaxAge:   cookieMaxAge,
		})
	}

	return c.JSON(fiber.Map{
		"user_id":     info.UserID,
		"tenant_id":   info.TenantID,
		"role":        info.Role,
		"school_id":   info.SchoolID,
		"school_name": info.SchoolName,
		"full_name":   info.FullName,
		"email":       info.Email,
	})
}

// clearAuthCookies removes all auth-related cookies from the response.
// Extracted so both Logout and the error path can clear cookies consistently.
func (h *Handler) clearAuthCookies(c *fiber.Ctx) {
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

	// Clear the school ID cookie
	c.Cookie(&fiber.Cookie{
		Name:     somoSchoolIDCookieName,
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
}

// Logout handles DELETE /api/auth/session (requirement 7).
func (h *Handler) Logout(c *fiber.Ctx) error {
	token := c.Cookies(somoCookieName)

	err := h.svc.Logout(c.Context(), token)

	// Always clear cookies — even if the service call fails (e.g. DB/Redis
	// hiccup), we must avoid leaving stale cookies that cause a redirect
	// loop (login → dashboard → logout → login → …).
	h.clearAuthCookies(c)

	if err != nil {
		return middleware.HTTPError(c, err)
	}

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
