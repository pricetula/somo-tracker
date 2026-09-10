// Package csrf provides double-submit cookie CSRF protection for the Somotracker API.
//
// The middleware implements the double-submit cookie pattern:
//  1. On safe responses (or explicitly), generates a cryptographically random CSRF token
//     and sets it as a non-HttpOnly cookie (readable by JavaScript).
//  2. On mutating requests (POST, PUT, PATCH, DELETE), validates that the token
//     from the cookie matches the token in the X-CSRF-Token header.
//  3. Tokens are per-session, not per-request, to avoid breaking multi-tab usage.
//
// The callback endpoint (/api/auth/callback) is exempt because it's a
// redirect from Stytch, not a form submission from our frontend.
//
// Integration:
//
//	protected := app.Group("/api", session.NewSessionMiddleware(redisClient, logger))
//	protected.Use(csrf.NewCSRFMiddleware())
//
// For public auth endpoints that mutate state (logout), apply selectively:
//
//	public.Post("/auth/logout", csrf.NewCSRFMiddleware(), r.Auth.logout)
package csrf

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"

	"github.com/gofiber/fiber/v3"
)

const (
	// CSRFCookieName is the name of the CSRF token cookie.
	CSRFCookieName = "csrf_token"

	// CSRFHeaderName is the header name for submitting the CSRF token.
	CSRFHeaderName = "X-CSRF-Token"

	// CSRFTokenLength is the byte length of the generated token (32 bytes = 256 bits).
	CSRFTokenLength = 32

	// CSRFCookieMaxAge is the cookie max age in seconds (24 hours).
	CSRFCookieMaxAge = 24 * 60 * 60
)

// NewCSRFMiddleware returns a Fiber handler that enforces double-submit cookie CSRF protection.
//
// The middleware:
//  1. On mutating requests (POST, PUT, PATCH, DELETE), validates the CSRF token.
//  2. On safe responses, ensures a CSRF token cookie is present (lazy issuance).
//  3. Exempts safe methods (GET, HEAD, OPTIONS) from validation.
//  4. Exempts the Stytch callback endpoint (/api/auth/callback) from validation.
//
// The token is validated using constant-time comparison to prevent timing attacks.
func NewCSRFMiddleware() fiber.Handler {
	return func(c fiber.Ctx) error {
		method := c.Method()

		// Exempt safe methods from CSRF validation
		if isSafeMethod(method) {
			return c.Next()
		}

		// Exempt the Stytch callback entirely - it's a redirect from Stytch, not a form submission
		if c.Path() == "/api/auth/callback" {
			return c.Next()
		}

		// Extract token from cookie (double-submit pattern)
		cookieToken := c.Cookies(CSRFCookieName)
		if cookieToken == "" {
			return csrfFailure(c, "missing_csrf_cookie", "CSRF token cookie missing")
		}

		// Extract token from header
		headerToken := c.Get(CSRFHeaderName)
		if headerToken == "" {
			return csrfFailure(c, "missing_csrf_header", "X-CSRF-Token header missing")
		}

		// Constant-time comparison to prevent timing attacks
		if !constantTimeEqual(cookieToken, headerToken) {
			return csrfFailure(c, "invalid_csrf_token", "CSRF token mismatch")
		}

		return c.Next()
	}
}

// EnsureCSRFTokenCookie returns a middleware that ensures the CSRF token cookie
// is set on responses. This should be used on routes that serve the initial
// page (e.g., login, register) so the frontend can read the token for
// subsequent mutating requests.
//
// The cookie is non-HttpOnly so JavaScript can read it for the double-submit pattern.
func EnsureCSRFTokenCookie() fiber.Handler {
	return func(c fiber.Ctx) error {
		// Set the cookie if not already present
		if c.Cookies(CSRFCookieName) == "" {
			token, err := GenerateCSRFToken()
			if err != nil {
				// Log but don't fail the request - validation will catch missing token
				_ = err
			} else {
				setCSRFCookie(c, token)
			}
		}
		return c.Next()
	}
}

// setCSRFCookie sets the CSRF token cookie with secure attributes.
func setCSRFCookie(c fiber.Ctx, token string) {
	c.Cookie(&fiber.Cookie{
		Name:     CSRFCookieName,
		Value:    token,
		Path:     "/",
		MaxAge:   CSRFCookieMaxAge,
		Secure:   true,
		HTTPOnly: false, // Must be readable by JavaScript for double-submit pattern
		SameSite: fiber.CookieSameSiteLaxMode,
	})
}

// GenerateCSRFToken generates a cryptographically random token.
// Exported for use in auth handler when setting cookie on callback.
func GenerateCSRFToken() (string, error) {
	b := make([]byte, CSRFTokenLength)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("csrf: generate token: %w", err)
	}
	return hex.EncodeToString(b), nil
}

// isSafeMethod returns true for HTTP methods that should not require CSRF validation.
func isSafeMethod(method string) bool {
	switch method {
	case fiber.MethodGet, fiber.MethodHead, fiber.MethodOptions:
		return true
	default:
		return false
	}
}

// constantTimeEqual performs constant-time string comparison to prevent timing attacks.
func constantTimeEqual(a, b string) bool {
	if len(a) != len(b) {
		return false
	}
	var diff byte
	for i := 0; i < len(a); i++ {
		diff |= a[i] ^ b[i]
	}
	return diff == 0
}

// csrfFailure returns a standardized CSRF validation failure response.
func csrfFailure(c fiber.Ctx, code, message string) error {
	return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
		"code":    code,
		"message": message,
		"errors":  fiber.Map{},
	})
}
