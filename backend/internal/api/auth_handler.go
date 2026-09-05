package api

import (
	"strings"
	"time"

	"github.com/gofiber/fiber/v3"

	"somotracker/backend/internal/services"
)

// authHandler responds to /api/auth/*.
type authHandler struct {
	svc services.AuthService
}

func newAuthHandler(svc services.AuthService) *authHandler {
	return &authHandler{svc: svc}
}

// sendMagicLink initiates a magic-link email for the provided address.
//
// Request body (JSON):
//
//	{ "email": "user@example.com" }
//
// Rate limiting is applied by the route group (see RegisterRoutes).
func (h *authHandler) sendMagicLink(c fiber.Ctx) error {
	email := strings.TrimSpace(c.FormValue("email"))
	if email == "" {
		// Try JSON body for clients that prefer application/json.
		var body struct {
			Email string `json:"email"`
		}
		if err := c.Bind().Body(&body); err == nil {
			email = strings.TrimSpace(body.Email)
		}
	}
	if email == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"code":    "missing_email",
			"message": "email is required",
			"errors":  fiber.Map{"email": []string{"required"}},
		})
	}

	// Accept optional discovery context (org slug / existing org id) from JSON or query.
	orgIDOrSlug := strings.TrimSpace(c.Query("org_id"))
	if orgIDOrSlug == "" {
		var body struct {
			OrgIDOrSlug string `json:"org_id"`
		}
		if err := c.Bind().Body(&body); err == nil {
			orgIDOrSlug = strings.TrimSpace(body.OrgIDOrSlug)
		}
	}

	if err := h.svc.SendMagicLink(c.Context(), email, orgIDOrSlug); err != nil {
		return mapAuthError(c, err)
	}

	// Always return a sanitized neutral 200 to prevent user enumeration.
	return c.JSON(fiber.Map{
		"code":    "magic_link_sent",
		"message": "If that email is registered, a magic link has been sent.",
		"errors":  fiber.Map{},
	})
}

// mapAuthError maps service-layer errors into HTTP responses.
func mapAuthError(c fiber.Ctx, err error) error {
	msg := err.Error()
	switch {
	case strings.HasPrefix(msg, "bad_request:"):
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"code":    "invalid_request",
			"message": strings.TrimPrefix(msg, "bad_request: "),
			"errors":  fiber.Map{},
		})
	case strings.HasPrefix(msg, "unauthorized:"):
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"code":    "unauthorized",
			"message": strings.TrimPrefix(msg, "unauthorized: "),
			"errors":  fiber.Map{},
		})
	case strings.HasPrefix(msg, "too_many_requests:"):
		return c.Status(fiber.StatusTooManyRequests).JSON(fiber.Map{
			"code":    "rate_limit_exceeded",
			"message": strings.TrimPrefix(msg, "too_many_requests: "),
			"errors":  fiber.Map{},
		})
	default:
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"code":    "internal_error",
			"message": "An unexpected error occurred",
			"errors":  fiber.Map{},
		})
	}
}

// callback handles the Stytch magic-link redirect.
// Stytch redirects the user's browser to this URL with the magic-link token
// in the query string. Rate limiting is applied by the route group
// (see RegisterRoutes).
func (h *authHandler) callback(c fiber.Ctx) error {
	token := strings.TrimSpace(c.Query("token"))
	if token == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"code":    "missing_token",
			"message": "token is required",
			"errors":  fiber.Map{"token": []string{"required"}},
		})
	}

	sessionResult, err := h.svc.AuthenticateCallback(c.Context(), token, c)
	if err != nil {
		return mapAuthError(c, err)
	}

	// Issue the opaque session token as an HttpOnly, Secure, SameSite=Lax cookie.
	// The raw token is NEVER exposed to client-side JavaScript.
	c.Cookie(&fiber.Cookie{
		Name:     "session_token",
		Value:    sessionResult.OpaqueToken,
		Path:     "/",
		Expires:  sessionResult.ExpiresAt,
		Secure:   true,
		HTTPOnly: true,
		SameSite: fiber.CookieSameSiteLaxMode,
	})

	return c.JSON(fiber.Map{
		"code":    "authenticated",
		"message": "Authentication successful",
		"errors":  fiber.Map{},
	})
}

// logout handles user logout by revoking the session.
// It requires a valid session cookie and returns a sanitized response.
func (h *authHandler) logout(c fiber.Ctx) error {
	// Extract the session token from the cookie.
	token := c.Cookies("session_token")
	if token == "" {
		// No session cookie - still return success to prevent enumeration
		// but clear any stale cookie.
		c.Cookie(&fiber.Cookie{
			Name:     "session_token",
			Value:    "",
			Path:     "/",
			Expires:  time.Now().Add(-24 * time.Hour),
			MaxAge:   -1,
			HTTPOnly: true,
			Secure:   true,
		})
		return c.JSON(fiber.Map{
			"code":    "logged_out",
			"message": "Logged out successfully",
			"errors":  fiber.Map{},
		})
	}

	// Get request ID for audit logging
	requestID := c.Get("X-Request-ID")

	// Revoke the session (deletes from Redis and database)
	if err := h.svc.RevokeSession(c.Context(), token, requestID); err != nil {
		return mapAuthError(c, err)
	}

	// Clear the session cookie
	c.Cookie(&fiber.Cookie{
		Name:     "session_token",
		Value:    "",
		Path:     "/",
		Expires:  time.Now().Add(-24 * time.Hour),
		MaxAge:   -1,
		HTTPOnly: true,
		Secure:   true,
		SameSite: fiber.CookieSameSiteLaxMode,
	})

	return c.JSON(fiber.Map{
		"code":    "logged_out",
		"message": "Logged out successfully",
		"errors":  fiber.Map{},
	})
}
